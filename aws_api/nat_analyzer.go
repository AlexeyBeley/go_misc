package aws_api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type NatAnalyzerConfig struct {
	Region  string
	Subnets []string
}

func AnalyzeFlow(region string, subnetIds []string, networkInterfacesFilePath string) error {
	networkInterfaces := make(map[string]ec2.NetworkInterface)

	RecordNetworkInterfaces(&region, &networkInterfaces)
	time.Sleep(30 * time.Second)
	//logGroups := getSubnetsFlowLogGroups(region, subnetIds)
	startLogScraper(region, []string{}, &networkInterfaces)

	return nil
}

func loadTestRealData(filePath string) map[string]any {
	myMap := make(map[string]any)
	jsonString, err := os.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal([]byte(jsonString), &myMap)
	if err != nil {
		panic(err)
	}

	return myMap
}

func getSubnetsFlowLogGroups(region string, subnetIds []string) map[string]string {
	ret := make(map[string]string)

	realConfig := loadTestRealData("/tmp/ec2.json")

	var subnetValues []*string
	//subnetIds = make([]*string, 0)

	flowLogSubnets, ok := realConfig["DescribeFlowLogsPagesSubnets"].([]any)
	if !ok {
		panic(subnetIds)
	}
	for _, subnetInterface := range flowLogSubnets {
		subnetString, ok := subnetInterface.(string)
		if !ok {
			panic(subnetInterface)
		}
		subnetValues = append(subnetValues, &subnetString)
	}

	region, ok = realConfig["Region"].(string)
	if !ok {
		panic(region)
	}

	client := getEC2Client(&region)
	Filters := []*ec2.Filter{{
		Name:   aws.String("resource-id"), // Filter by resource ID
		Values: subnetValues,
	}}
	objects := make([]any, 0)
	err := DescribeFlowLogsPages(client, Filters, AggregatorInitializer(&objects))
	if err != nil {
		panic(err)
	}
	if len(objects) != len(subnetValues) {
		panic(err)
	}

	for _, obj := range objects {
		flowLog, ok := obj.(*ec2.FlowLog)
		if !ok {
			panic(obj)
		}
		ret[*flowLog.ResourceId] = *flowLog.LogGroupName
	}
	return ret
}

func startLogScraper(region string, logGroups []string, networkInterfaces *map[string]ec2.NetworkInterface) error {
	for _, logGroupName := range logGroups {
		err := YieldCloudwatchLogStreams(region, logGroupName, SubnetFlowStreamByteSumCallback(region, logGroupName, networkInterfaces))
		if err != nil {
			return err
		}
	}
	return nil
}

func SubnetFlowStreamByteSumCallback(region, logGroupName string, networkInterfaces *map[string]ec2.NetworkInterface) func(LogStream any) error {
	return func(anyLogStream any) error {
		LogStream, ok := anyLogStream.(*cloudwatchlogs.LogStream)
		if !ok {
			panic(anyLogStream)
		}
		nameTokens := strings.Split(*LogStream.LogStreamName, "-")
		nicName := nameTokens[0] + "-" + nameTokens[1]
		_, exists := (*networkInterfaces)[nicName]
		if !exists {
			return nil
		}

		var sum uint64
		sum = 0

		nowUTC := time.Now().UTC()
		epochEndSeconds := nowUTC.Unix()
		epochStartSeconds := epochEndSeconds - 24*60*60
		//LogStream.FirstEventTimestamp
		epochEndMiliSeconds := epochEndSeconds * 1000
		epochStartMiliSeconds := epochStartSeconds * 1000

		if *LogStream.LastEventTimestamp < epochStartSeconds {
			return nil
		}

		YieldCloudwatchLogStream(&region, &logGroupName, LogStream.LogStreamName, nil, &epochStartMiliSeconds, &epochEndMiliSeconds, BytesSummarizer(&sum))

		filename := "/tmp/eni_bytes.out"
		file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(fmt.Sprintf("Error opening file: %s", err))
		}
		defer file.Close()
		textToAppend := fmt.Sprintf("streamName: %s Sum: %d \n", *LogStream.LogStreamName, sum)
		_, err = file.WriteString(textToAppend)
		if err != nil {
			panic(fmt.Sprintf("Error writing to file: %s", err))
		}

		return nil
	}

}

func BytesSummarizer(aggregator *uint64) func(*cloudwatchlogs.OutputLogEvent) error {
	return func(event *cloudwatchlogs.OutputLogEvent) error {
		if strings.Contains(*event.Message, "NODATA") {
			return nil
		}
		//fmt.Println("  ", *event.Message)
		stringSplit := strings.Split(*event.Message, " ")
		srcaddr := stringSplit[3]
		dstaddr := stringSplit[4]
		ipSrc := net.ParseIP(srcaddr)
		ipDst := net.ParseIP(dstaddr)
		if ipSrc == nil || ipDst == nil {
			return fmt.Errorf("srcaddr: %v, dstaddr: %v ", srcaddr, dstaddr)
		}

		if ipSrc.IsPrivate() && ipDst.IsPrivate() {
			return nil
		}

		bytes, err := strconv.Atoi(stringSplit[9])
		if err != nil {
			return err
		}
		*aggregator += uint64(bytes)
		return nil
	}
}

func StartRecording(configFilePath string) error {
	config := NatAnalyzerConfig{}
	jsonString, err := os.ReadFile(configFilePath)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal([]byte(jsonString), &config)
	if err != nil {
		return err
	}

	subnetLogGroupNames := getSubnetsFlowLogGroups(config.Region, config.Subnets)

	for _, subnetId := range config.Subnets {
		go StartSubnetRecording(config.Region, subnetId, subnetLogGroupNames[subnetId])
	}

	for {
		time.Sleep(60 * time.Second)
	}
	return nil
}

func StartSubnetRecording(region, subnetId string, subnetLogGroupName string) error {
	networkInterfaces := make(map[string]context.CancelFunc)
	for {
		interfaceIds := GetSubnetInterfaceIds(region, subnetId)
		for _, interId := range interfaceIds {
			if _, ok := networkInterfaces[interId]; !ok {
				ctx, cancel := context.WithCancel(context.Background())
				networkInterfaces[interId] = cancel
				go StartInterfaceRecording(interId, subnetId, subnetLogGroupName, region, &ctx)
			}
		}

		for interId := range networkInterfaces {

			if !slices.Contains(interfaceIds, interId) {
				StopInterfaceRecording(interId, networkInterfaces)
			}
		}
		log.Printf("subnet %s recording %d interfaces", subnetId, len(networkInterfaces))
		time.Sleep(30 * time.Second)
	}

}

func GetSubnetInterfaceIds(region, subnetId string) []string {
	Filters := []*ec2.Filter{{
		Name:   aws.String("subnet-id"), // Filter by resource ID
		Values: []*string{&subnetId},
	}}
	describeNetworkInterfacesInput := ec2.DescribeNetworkInterfacesInput{Filters: Filters}
	client := getEC2Client(&region)
	objects := make([]any, 0)
	err := DescribeNetworkInterfaces(client, AggregatorInitializer(&objects), &describeNetworkInterfacesInput)
	if err != nil {
		log.Printf("call GetSubnetInterfaceIds(%s, %s)->DescribeNetworkInterfaces %v", region, subnetId, err)
	}
	ret := []string{}
	for _, anyObject := range objects {
		nInt, ok := anyObject.(*ec2.NetworkInterface)
		if !ok {
			panic(anyObject)
		}
		ret = append(ret, *nInt.NetworkInterfaceId)

	}
	return ret
}

func StartInterfaceRecording(interId, subnetId, subnetLogGroupName, region string, ctx *context.Context) error {
	svc := getCloudwatchLogClient(&region)
	limit := int64(50)
	objects := make([]any, 0)
	err := GetLogStreamsRaw(svc, &limit, &subnetLogGroupName, &interId, AggregatorInitializer(&objects))
	if err != nil {
		return err
	}

	if len(objects) != 1 {
		return fmt.Errorf("expected to find single stream by interface prefix '%s' but found %d", interId, len(objects))
	}

	stream, ok := objects[0].(*cloudwatchlogs.LogStream)
	if !ok {
		return fmt.Errorf("expected  '%v' ", objects[0])
	}

	var nextToken *string
	var sum uint64
	for {
		sum = 0
		lastResp, err := YieldCloudwatchLogStream(&region, &subnetLogGroupName, stream.LogStreamName, nextToken, nil, nil, BytesSummarizer(&sum))
		if err != nil {
			log.Printf("call StartInterfaceRecording(%s, %s)->YieldCloudwatchLogStream %v", region, subnetLogGroupName, err)
			time.Sleep(5 * time.Second)
		}
		if lastResp != nil {
			nextToken = lastResp.NextForwardToken
		}

		select {
		case <-(*ctx).Done():
			log.Printf("stopping interface recording: %s", interId)
			return nil
		default:
			time.Sleep(5 * time.Second)
		}
	}
}

func StopInterfaceRecording(interId string, networkInterfaces map[string]context.CancelFunc) {

	networkInterfaces[interId]()
	time.Sleep(15 * time.Second)
	//networkInterfaces[interId]
}
