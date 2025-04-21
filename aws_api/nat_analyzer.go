package aws_api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/AlexeyBeley/go_common/logger"
	clients "github.com/AlexeyBeley/go_misc/aws_api/clients"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var lg = &(logger.Logger{})

type NatAnalyzerConfig struct {
	Region  string
	Subnets []string
}

func StartRecording(configFilePath string) error {
	lg.FileDst = "/tmp/nat_analyzer.log"
	workPool := make(chan bool, 5)

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
		go StartSubnetRecording(&workPool, config.Region, subnetId, subnetLogGroupNames[subnetId])
	}

	for {
		time.Sleep(60 * time.Second)
	}

}

func getSubnetsFlowLogGroups(region string, subnetIds []string) map[string]string {
	ret := make(map[string]string)

	var subnetValues []*string
	//subnetIds = make([]*string, 0)

	for _, subnetString := range subnetIds {
		subnetValues = append(subnetValues, &subnetString)
	}

	client := clients.GetEC2Client(&region)
	Filters := []*ec2.Filter{{
		Name:   aws.String("resource-id"), // Filter by resource ID
		Values: subnetValues,
	}}
	objects := make([]any, 0)
	err := clients.DescribeFlowLogsPages(client, Filters, clients.AggregatorInitializer(&objects))
	if err != nil {
		panic(err)
	}
	if len(objects) != len(subnetValues) {
		panic(fmt.Sprintf("Expected %d flow logs but found %d", len(subnetValues), len(objects)))
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

type interfaceDataCollector struct {
	MaxMinuteBytesIn       uint64
	MinMinuteBytesIn       uint64
	TotalBytesIn           uint64
	TotaldurationSecondsIn uint64

	MaxMinuteBytesOut       uint64
	MinMinuteBytesOut       uint64
	TotalBytesOut           uint64
	TotaldurationSecondsOut uint64
}

func BytesSummarizer(dataCollector *interfaceDataCollector) func(*cloudwatchlogs.OutputLogEvent) error {
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
		if !ipSrc.IsPrivate() && !ipDst.IsPrivate() {
			return fmt.Errorf("public to public not supported yet: %s", *event.Message)
		}

		bytes, err := strconv.Atoi(stringSplit[9])
		if err != nil {
			return err
		}

		secondsStart, err := strconv.Atoi(stringSplit[10])
		if err != nil {
			return err
		}

		secondsEnd, err := strconv.Atoi(stringSplit[11])
		if err != nil {
			return err
		}
		secondsDuration := uint64(secondsEnd) - uint64(secondsStart)
		if secondsDuration == 0 {
			secondsDuration = 1
		}
		minuteBytes := uint64(bytes*60) / secondsDuration

		if ipSrc.IsPrivate() {
			dataCollector.MaxMinuteBytesOut = max(uint64(minuteBytes), dataCollector.MaxMinuteBytesOut)
			dataCollector.MinMinuteBytesOut = min(uint64(minuteBytes), dataCollector.MinMinuteBytesOut)
			dataCollector.TotalBytesOut += uint64(bytes)
			dataCollector.TotaldurationSecondsOut += secondsDuration

		} else {
			dataCollector.MaxMinuteBytesIn = max(uint64(minuteBytes), dataCollector.MaxMinuteBytesIn)
			dataCollector.MinMinuteBytesIn = min(uint64(minuteBytes), dataCollector.MinMinuteBytesIn)
			dataCollector.TotalBytesIn += uint64(bytes)
			dataCollector.TotaldurationSecondsIn += secondsDuration
		}
		return nil
	}
}

func StartSubnetRecording(workPool *chan bool, region, subnetId string, subnetLogGroupName string) error {
	networkInterfaces := make(map[string]context.CancelFunc)
	for {
		interfaceIds := GetSubnetInterfaceIds(region, subnetId)
		for _, interId := range interfaceIds {
			if _, ok := networkInterfaces[interId]; !ok {
				ctx, cancel := context.WithCancel(context.Background())
				networkInterfaces[interId] = cancel
				go StartInterfaceRecording(workPool, interId, subnetId, subnetLogGroupName, region, &ctx)
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
	client := clients.GetEC2Client(&region)
	objects := make([]any, 0)
	err := clients.DescribeNetworkInterfaces(client, clients.AggregatorInitializer(&objects), &describeNetworkInterfacesInput)
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

// ${version} ${account-id} ${interface-id} ${srcaddr} ${dstaddr} ${srcport} ${dstport} ${protocol} ${packets} ${bytes} ${start} ${end} ${action} ${log-status}
func StartInterfaceRecording(workPool *chan bool, interId, subnetId, subnetLogGroupName, region string, ctx *context.Context) error {
	svc := clients.GetCloudwatchLogClient(&region)
	limit := int64(50)
	objects := make([]any, 0)
	err := clients.GetLogStreamsRaw(svc, &limit, &subnetLogGroupName, &interId, clients.AggregatorInitializer(&objects))
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
	dataCollector := interfaceDataCollector{MinMinuteBytesIn: math.MaxInt64, MinMinuteBytesOut: math.MaxInt64}
	dataCollectorPrev := interfaceDataCollector{MinMinuteBytesIn: math.MaxInt64, MinMinuteBytesOut: math.MaxInt64}

	nowUTC := time.Now().UTC()
	epochNowSeconds := nowUTC.Unix()
	epochStartMiliSeconds := epochNowSeconds * 1000
	pEpochStartMiliSeconds := &epochStartMiliSeconds

	for {
		if nextToken != nil {
			pEpochStartMiliSeconds = nil
		}
		*workPool <- true
		lastResp, err := clients.YieldCloudwatchLogStream(&region, &subnetLogGroupName, stream.LogStreamName, nextToken, pEpochStartMiliSeconds, nil, BytesSummarizer(&dataCollector))
		<-*workPool

		if err != nil {
			log.Printf("call StartInterfaceRecording(%s, %s)->YieldCloudwatchLogStream %v", region, subnetLogGroupName, err)
			time.Sleep(5 * time.Second)
		} else {
			if dataCollectorPrev != dataCollector {
				log.Printf("NetworkInterface %s  MinThroughputIn: %d, MaxThroughputIn: %d, TotalBytesIn: %d, TotalDurationIn: %d", interId, dataCollector.MinMinuteBytesIn, dataCollector.MaxMinuteBytesIn, dataCollector.TotalBytesIn, dataCollector.TotaldurationSecondsIn)
				log.Printf("NetworkInterface %s  MinThroughputOut: %d, MaxThroughputOut: %d, TotalBytesOut: %d, TotalDurationOut: %d", interId, dataCollector.MinMinuteBytesOut, dataCollector.MaxMinuteBytesOut, dataCollector.TotalBytesOut, dataCollector.TotaldurationSecondsOut)
				output := map[string]any{"NetworkInterface": interId,
					"MinMinuteBytesIn":        dataCollector.MinMinuteBytesIn,
					"MaxMinuteBytesIn":        dataCollector.MaxMinuteBytesIn,
					"TotalBytesIn":            dataCollector.TotalBytesIn,
					"TotaldurationSecondsIn":  dataCollector.TotaldurationSecondsIn,
					"MinMinuteBytesOut":       dataCollector.MinMinuteBytesOut,
					"MaxMinuteBytesOut":       dataCollector.MaxMinuteBytesOut,
					"TotalBytesOut":           dataCollector.TotalBytesOut,
					"TotaldurationSecondsOut": dataCollector.TotaldurationSecondsOut,
				}
				lg.InfoM(output)
				dataCollectorPrev = dataCollector
			}

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
