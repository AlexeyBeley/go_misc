package aws_api

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/AlexeyBeley/go_common/logger"
	replacementEngine "github.com/AlexeyBeley/go_common/replacement_engine"
	clients "github.com/AlexeyBeley/go_misc/aws_api/clients"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type AWSTCPDumpConfig struct {
	Region                   string   `json:"Region"`
	Subnets                  []string `json:"Subnets"`
	AWSProfile               string   `json:"AWSProfile"`
	InterfacesOutputFilePath string   `json:"InterfacesOutputFilePath"`
	LogOutputFilePath        string   `json:"LogOutputFilePath"`
	ProcessedOutputFilePath  string   `json:"ProcessedOutputFilePath"`
	IamDataDirPath           string   `json:"IamDataDirPath"`
	LiveRecording            bool     `json:"LiveRecording"`
	AddrFilters              []string `json:"AddrFilters"`
}

type AWSTCPDump struct {
	Config         *AWSTCPDumpConfig
	KnownIntefaces []*ec2.NetworkInterface
	Done           bool
	JsonLogger     *logger.Logger
	EventsFilter   func(*FlowLogEvent) (*FlowLogEvent, error)
	EventProcessor func(*FlowLogEvent) error
}

func AWSTCPDumpNew() (*AWSTCPDump, error) {
	new := &AWSTCPDump{}
	new.initConfig()

	new.Done = false

	new.JsonLogger = &(logger.Logger{FileDst: new.Config.InterfacesOutputFilePath, AddDateTime: true})

	return new, nil
}
func (AwsTCPDump *AWSTCPDump) LoadConfig(filePath string) (*AWSTCPDumpConfig, error) {
	fileString, err := os.ReadFile(filePath)
	_ = fileString
	// todo:
	if err != nil {
		return nil, err
	}
	config := &AWSTCPDumpConfig{}
	return config, nil
}
func (AwsTCPDump *AWSTCPDump) ParseArgs(flagset *flag.FlagSet, args []string) (*AWSTCPDumpConfig, error) {
	var region *string
	if slices.Contains(args, "-region") {
		region = flagset.String("region", "", "AWS Region")
	} else {
		region = new(string)
	}

	var subnetsString *string
	if slices.Contains(args, "-subnets") {
		subnetsString = flagset.String("subnets", "", "AWS Subnets to listen too")
	} else {
		subnetsString = new(string)
	}

	var profile *string
	if slices.Contains(args, "-profile") {
		profile = flagset.String("profile", "default", "AWS profile")
	} else {
		profile = new(string)
	}

	var live *bool
	if slices.Contains(args, "-live") {
		live = flagset.Bool("live", false, "Live traffic recording")
	} else {
		live = new(bool)
	}

	var filterAddrs *string
	if slices.Contains(args, "-addr") {
		filterAddrs = flagset.String("addr", "", "Filter addresses")
	} else {
		filterAddrs = new(string)
	}

	var configPath *string
	if slices.Contains(args, "-config") {
		configPath = flagset.String("confg", "", "Configuration file path")
	} else {
		configPath = clients.StrPtr("/opt/aws_api/tcpdump/data/tcpdump.conf")
	}

	err := flagset.Parse(args)
	if err != nil {
		return nil, err
	}

	config, err := AwsTCPDump.LoadConfig(*configPath)
	if err != nil {
		return nil, err
	}

	if *subnetsString == "" {
		config.Subnets = []string{}
	} else {
		config.Subnets = strings.Split(*subnetsString, ",")
	}

	if *region != "" {
		config.Region = *region
	}

	if *filterAddrs == "" {
		config.AddrFilters = []string{}
	} else {
		config.AddrFilters = strings.Split(*filterAddrs, ",")
	}

	if *profile != "" {
		config.AWSProfile = *profile
	}

	if *live {
		config.LiveRecording = *live
	}

	return config, nil
}

func (AwsTCPDump *AWSTCPDump) initConfig() error {

	//InterfacesOutputFilePath
	//LogOutputFilePath
	//IamDataDirPath
	//ProcessedOutputFilePath
	//

	flagset := &flag.FlagSet{}
	config, err := AwsTCPDump.ParseArgs(flagset, os.Args)
	if err != nil {
		return err
	}
	AwsTCPDump.Config = config

	if AwsTCPDump.Config.ProcessedOutputFilePath == "" {
		AwsTCPDump.Config.ProcessedOutputFilePath = "/opt/aws_api/tcpdump/output/data.log"

	}

	if _, err := os.Stat(AwsTCPDump.Config.ProcessedOutputFilePath); err == nil {
		os.Truncate(AwsTCPDump.Config.ProcessedOutputFilePath, 0)
	}

	if AwsTCPDump.Config.InterfacesOutputFilePath == "" {
		AwsTCPDump.Config.InterfacesOutputFilePath = "/opt/aws_api/tcpdump/output/interfaces.json"
	}

	if _, err := os.Stat(config.InterfacesOutputFilePath); err == nil {
		os.Truncate(config.InterfacesOutputFilePath, 0)
	} else if !os.IsNotExist(err) {
		fmt.Printf("Error checking file  '%s': %v\n", config.InterfacesOutputFilePath, err)
	}

	if AwsTCPDump.Config.LogOutputFilePath == "" {
		AwsTCPDump.Config.LogOutputFilePath = "/opt/aws_api/tcpdump/output/tcpdump.log"

	}

	if _, err := os.Stat(config.LogOutputFilePath); err == nil {
		os.Truncate(config.LogOutputFilePath, 0)
	}
	lg.FileDst = config.LogOutputFilePath
	return nil
}

func (awsTCPDump *AWSTCPDump) Start() error {
	workPool := make(chan bool, 5)

	subnetLogGroupNames, err := provisionSubnetsFlowLogGroups(awsTCPDump.Config)
	if err != nil {
		return err
	}

	for _, subnetId := range awsTCPDump.Config.Subnets {
		go awsTCPDump.StartSubnetInterfacesRecording(&workPool, subnetId, subnetLogGroupNames[subnetId])
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	fmt.Printf("\nReceived OS signal: %s. Initiating graceful shutdown...\n", sig)

	// Signal the main logic goroutine to stop.
	awsTCPDump.Done = true

	// Give some time for cleanup or final tasks.
	fmt.Println("Performing final cleanup (e.g., closing connections, saving state)...")
	time.Sleep(2 * time.Second) // Simulate cleanup time

	fmt.Println("Program gracefully stopped.")
	return nil

}

func provisionSubnetsFlowLogGroups(config *AWSTCPDumpConfig) (map[string]string, error) {
	iamAPI := clients.IAMAPINew(&config.AWSProfile, &config.IamDataDirPath)

	stsAPI := clients.STSAPINew(&config.AWSProfile)
	accountID, err := stsAPI.GetAccount()
	if err != nil {
		return nil, err
	}
	replacementValues := map[string]string{"STRING_REPLACEMENT_AWS_SERVICE_PRINCIPAL": "vpc-flow-logs.amazonaws.com",
		"STRING_REPLACEMENT_AWS_ACCOUNT_ID": *accountID}
	dstDir := filepath.Join(config.IamDataDirPath, "tmp")
	dstFilePath, err := replacementEngine.ReplaceInTemplateFile(filepath.Join(config.IamDataDirPath, "template_cloudwatch_writer_service_assume_role.json"),
		dstDir, replacementValues)
	if err != nil {
		return nil, err
	}

	assumeDocument, err := os.ReadFile(dstFilePath)
	strAssumeDocument := string(assumeDocument)

	if err != nil {
		fmt.Println("Error Reading file:", err)
		return nil, err
	}

	roleName := "role-aws-tcpdump"
	path := "/test/"
	role, err := iamAPI.ProvisionIamCloudwatchWriterRole(&config.Region, &roleName, &strAssumeDocument, &path)
	if err != nil {
		panic(err)
	}
	ret := make(map[string]string)

	var subnetValues []*string

	for _, subnetString := range config.Subnets {
		subnetValues = append(subnetValues, &subnetString)
	}

	client := clients.GetEC2Client(&config.Region)
	Filters := []*ec2.Filter{{
		Name:   aws.String("resource-id"),
		Values: subnetValues,
	}}

	flowLogObjects := make([]any, 0)
	err = clients.DescribeFlowLogsPages(client, Filters, clients.AggregatorInitializer(&flowLogObjects))
	if err != nil {
		panic(err)
	}

	for _, obj := range flowLogObjects {
		flowLog, ok := obj.(*ec2.FlowLog)
		if !ok {
			panic(obj)
		}
		ret[*flowLog.ResourceId] = *flowLog.LogGroupName
	}
	resourceType := "Subnet"
	trafficType := "ALL"
	for _, subnetId := range config.Subnets {
		api := clients.EC2APINew(&config.Region, &config.AWSProfile)
		_, ok := ret[subnetId]
		if !ok {
			logGroupName := provisionSubnetLogGroup(config, &subnetId)
			_, err := api.ProvisionFlowLog(&logGroupName, &resourceType, &trafficType, []*string{&subnetId}, role.Arn)
			if err != nil {
				panic(err)
			}
			ret[subnetId] = logGroupName
		}
	}
	return ret, nil
}

func provisionSubnetLogGroup(config *AWSTCPDumpConfig, subnetId *string) (logGroupName string) {
	api := clients.CloudwatchLogsAPINew(&config.Region, &config.AWSProfile)
	logGroupName = "tcpdump-" + *subnetId
	existingLogGroup, err := api.GetLogGroup(&logGroupName)

	if err != nil {
		panic(err)
	}

	if existingLogGroup == nil {
		output, err := api.ProvisionLogGroup(logGroupName)
		if err != nil {
			panic(err)
		}
		lg.InfoF("Provision Log group response: %v", output)
	}

	return logGroupName
}

func (awsTCPDump *AWSTCPDump) GetInterfaceChanges(config *AWSTCPDumpConfig, KnownNetworkInterfaces *map[string]context.CancelFunc, subnetId string) ([]string, []string, error) {
	retAdd := []string{}
	retDel := []string{}
	fetchedIds := map[string]bool{}

	fetchedInterfaces := awsTCPDump.GetSubnetInterfacesFromAPI(subnetId)

	for _, ec2Interface := range fetchedInterfaces {
		fetchedIds[*ec2Interface.NetworkInterfaceId] = true

		if _, ok := (*KnownNetworkInterfaces)[*ec2Interface.NetworkInterfaceId]; !ok {
			retAdd = append(retAdd, *ec2Interface.NetworkInterfaceId)
			// 1. Marshal the struct to JSON bytes
			jsonBytes, err := json.Marshal(ec2Interface)
			if err != nil {
				return nil, nil, fmt.Errorf("error marshaling NetworkInterface to JSON: %v", err)
			}

			// 2. Unmarshal the JSON bytes into a map[string]interface{}
			var niMap map[string]interface{}
			err = json.Unmarshal(jsonBytes, &niMap)
			if err != nil {
				return nil, nil, fmt.Errorf("error unmarshaling JSON to map: %v", err)

			}
			awsTCPDump.JsonLogger.InfoM(niMap)
		}
	}

	for ec2InterfaceId := range *KnownNetworkInterfaces {
		if _, ok := fetchedIds[ec2InterfaceId]; !ok {
			retDel = append(retDel, ec2InterfaceId)
		}
	}

	return retAdd, retDel, nil
}

func (awsTCPDump *AWSTCPDump) StartSubnetInterfacesRecording(workPool *chan bool, subnetId string, subnetLogGroupName string) error {
	networkInterfaces := make(map[string]context.CancelFunc)
	for {
		interfacesAdded, InterfacesDeprecated, err := awsTCPDump.GetInterfaceChanges(awsTCPDump.Config, &networkInterfaces, subnetId)
		lg.InfoF("Subnet %s Added intefaces: %d Deprecated interfaces: %d ", subnetId, len(interfacesAdded), len(InterfacesDeprecated))
		if err != nil {
			return err
		}

		if awsTCPDump.Config.LiveRecording {
			for _, ec2InterfaceId := range interfacesAdded {

				ctx, cancel := context.WithCancel(context.Background())
				networkInterfaces[ec2InterfaceId] = cancel
				go awsTCPDump.StartInterfaceRecording(workPool, ec2InterfaceId, subnetId, subnetLogGroupName, &ctx)

			}

			for _, ec2InterfaceId := range InterfacesDeprecated {
				awsTCPDump.StopInterfaceRecording(ec2InterfaceId, &networkInterfaces)
			}
		} else {
			for _, ec2InterfaceId := range interfacesAdded {
				networkInterfaces[ec2InterfaceId] = nil
			}
			for _, interId := range InterfacesDeprecated {
				delete(networkInterfaces, interId)
			}
		}

		lg.InfoF("Subnet %s Current: %d interfaces", subnetId, len(networkInterfaces))

		if awsTCPDump.Done {
			return nil
		}
		time.Sleep(30 * time.Second)
	}
}

func (awsTCPDump *AWSTCPDump) GetSubnetInterfacesFromAPI(subnetId string) []*ec2.NetworkInterface {

	api := clients.EC2APINew(&awsTCPDump.Config.Region, &awsTCPDump.Config.AWSProfile)
	Filters := []*ec2.Filter{{
		Name:   aws.String("subnet-id"), // Filter by resource ID
		Values: []*string{&subnetId},
	}}
	describeNetworkInterfacesInput := ec2.DescribeNetworkInterfacesInput{Filters: Filters}

	objects := make([]any, 0)
	err := api.GetNetworkInterfaces(clients.AggregatorInitializer(&objects), &describeNetworkInterfacesInput)
	if err != nil {
		lg.InfoF("call GetSubnetInterfaceIds(%s, %s)->DescribeNetworkInterfaces %v", awsTCPDump.Config.Region, subnetId, err)
	}
	ret := []*ec2.NetworkInterface{}
	for _, anyObject := range objects {
		nInt, ok := anyObject.(*ec2.NetworkInterface)
		if !ok {
			panic(anyObject)
		}
		ret = append(ret, nInt)

	}
	return ret
}

func PrintInterfaceDescription(config *AWSTCPDumpConfig, interfaceId *string) error {
	api := clients.EC2APINew(&config.Region, &config.AWSProfile)
	values := []*string{interfaceId}

	describeNetworkInterfacesInput := ec2.DescribeNetworkInterfacesInput{NetworkInterfaceIds: values}

	objects := make([]any, 0)
	err := api.GetNetworkInterfaces(clients.AggregatorInitializer(&objects), &describeNetworkInterfacesInput)
	if err != nil {
		return err
	}
	return nil
}

// Log stream per inteface with interface id in the name.
// ${version} ${account-id} ${interface-id} ${srcaddr} ${dstaddr} ${srcport} ${dstport} ${protocol} ${packets} ${bytes} ${start} ${end} ${action} ${log-status}
func (awsTCPDump *AWSTCPDump) StartInterfaceRecording(workPool *chan bool, interId, subnetId, subnetLogGroupName string, ctx *context.Context) error {
	svc := clients.GetCloudwatchLogClient(&awsTCPDump.Config.Region)
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

	nowUTC := time.Now().UTC()
	epochNowSeconds := nowUTC.Unix()
	epochStartMiliSeconds := epochNowSeconds * 1000
	pEpochStartMiliSeconds := &epochStartMiliSeconds

	for {
		if nextToken != nil {
			pEpochStartMiliSeconds = nil
		}
		*workPool <- true
		lastResp, err := clients.YieldCloudwatchLogStream(&awsTCPDump.Config.Region, &subnetLogGroupName, stream.LogStreamName, nextToken, pEpochStartMiliSeconds, nil, awsTCPDump.FlowLogEventsProcessRoutine)
		<-*workPool

		if err != nil {
			lg.InfoF("call StartInterfaceRecording(%s, %s)->YieldCloudwatchLogStream %v", awsTCPDump.Config.Region, subnetLogGroupName, err)
			time.Sleep(5 * time.Second)

			//todo: check this logic:
			awsTCPDump.Done = true
		}

		if lastResp != nil {
			nextToken = lastResp.NextForwardToken
		}

		select {
		case <-(*ctx).Done():
			lg.InfoF("stopped interface recording: %s", interId)
			return nil
		default:
			time.Sleep(5 * time.Second)
		}
	}
}

func (awsTCPDump *AWSTCPDump) StopInterfaceRecording(interId string, networkInterfaces *map[string]context.CancelFunc) {
	lg.InfoF("stopping interface recording using context: %s", interId)
	(*networkInterfaces)[interId]()
	time.Sleep(15 * time.Second)
	delete(*networkInterfaces, interId)
}

func AWSTCPDumpAnalize(filePath string) (string, error) {
	ret := ""
	if _, err := os.Stat(filePath); err != nil {
		if !os.IsNotExist(err) {
			return ret, fmt.Errorf("error checking file  '%s': %v", filePath, err)
		}
		return ret, nil
	}

	jsonLines, err := os.ReadFile(filePath)
	if err != nil {
		return ret, err
	}

	perInterface := make(map[string]*interfaceDataCollector)
	for _, line := range strings.Split(string(jsonLines), "\n") {
		if line == "" {
			continue
		}
		data := new(interfaceDataCollector)
		err = json.Unmarshal([]byte(line), data)
		if err != nil {
			return ret, err
		}
		perInterface[*data.NetworkInterfaceId] = data
	}

	maxInBytes := new(uint64)
	maxInInterface := interfaceDataCollector{}

	maxOutBytes := new(uint64)
	maxOutInterface := interfaceDataCollector{}

	for _, data := range perInterface {
		if data.TotalBytesIn > *maxInBytes {
			maxInInterface = *data
			maxInBytes = &data.TotalBytesIn
		}
		if data.TotalBytesOut > *maxOutBytes {
			maxOutInterface = *data
			maxOutBytes = &data.TotalBytesOut
		}
	}

	if maxInInterface.NetworkInterfaceId != nil {
		lg.InfoF("Max In:  %v: %d , Max Out: %v: %d", *maxInInterface.NetworkInterfaceId, maxInBytes, *maxOutInterface.NetworkInterfaceId, maxOutBytes)
	}
	return ret, nil
}

type interfaceDataCollector struct {
	NetworkInterfaceId     *string
	MaxMinuteBytesIn       uint64
	MinMinuteBytesIn       uint64
	TotalBytesIn           uint64
	TotaldurationSecondsIn uint64

	MaxMinuteBytesOut       uint64
	MinMinuteBytesOut       uint64
	TotalBytesOut           uint64
	TotaldurationSecondsOut uint64
}

// Summarize the sent and received traffic into data collector.
func (awsTCPDump *AWSTCPDump) FlowLogEventsBytesSummarizerHandler(dataCollector *interfaceDataCollector) func(*cloudwatchlogs.OutputLogEvent) error {
	return func(event *cloudwatchlogs.OutputLogEvent) error {
		dataCollectorPrev := *dataCollector
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

		if dataCollectorPrev != *dataCollector {
			lg.InfoF("NetworkInterfaceId %s TotalBytesIn: %d, TotalDurationIn: %d", *dataCollector.NetworkInterfaceId, dataCollector.TotalBytesIn, dataCollector.TotaldurationSecondsIn)
			output := map[string]any{"NetworkInterfaceId": *dataCollector.NetworkInterfaceId,
				"MinMinuteBytesIn":        dataCollector.MinMinuteBytesIn,
				"MaxMinuteBytesIn":        dataCollector.MaxMinuteBytesIn,
				"TotalBytesIn":            dataCollector.TotalBytesIn,
				"TotaldurationSecondsIn":  dataCollector.TotaldurationSecondsIn,
				"MinMinuteBytesOut":       dataCollector.MinMinuteBytesOut,
				"MaxMinuteBytesOut":       dataCollector.MaxMinuteBytesOut,
				"TotalBytesOut":           dataCollector.TotalBytesOut,
				"TotaldurationSecondsOut": dataCollector.TotaldurationSecondsOut,
			}
			awsTCPDump.JsonLogger.InfoM(output)
		}

		return nil
	}
}

type FlowLogEvent struct {
	Version     string
	AccoundID   string
	InterfaceID string
	SrcAddr     net.IP
	DstAddr     net.IP
	SrcPort     int
	DstPort     int
	Protocol    string
	Packets     int
	Bytes       int
	Start       int
	End         int
	Action      string
	LogStatus   string
}

func (awsTCPDump *AWSTCPDump) EventsEchoFilter(event *FlowLogEvent) (*FlowLogEvent, error) {
	return event, nil

}

func (awsTCPDump *AWSTCPDump) FlowLogEventsProcessRoutine(CloudwatchEvent *cloudwatchlogs.OutputLogEvent) error {
	event, err := awsTCPDump.ParseEvent(CloudwatchEvent)
	if err != nil {
		return err
	}

	event, err = awsTCPDump.EventsFilter(event)
	if err != nil {
		return err
	}

	if event != nil {
		return awsTCPDump.EventProcessor(event)
	}
	return nil
}

// Summarize the sent and received traffic into data collector.
func (awsTCPDump *AWSTCPDump) ParseEvent(event *cloudwatchlogs.OutputLogEvent) (*FlowLogEvent, error) {
	// ${version} ${account-id} ${interface-id} ${srcaddr} ${dstaddr} ${srcport} ${dstport} ${protocol} ${packets} ${bytes} ${start} ${end} ${action} ${log-status}

	retFlowEvent := &FlowLogEvent{}
	stringSplit := strings.Split(*event.Message, " ")
	retFlowEvent.Version = stringSplit[0]
	retFlowEvent.AccoundID = stringSplit[1]
	retFlowEvent.InterfaceID = stringSplit[2]
	secondsStart, err := strconv.Atoi(stringSplit[10])
	if err != nil {
		return nil, err
	}
	retFlowEvent.Start = secondsStart

	secondsEnd, err := strconv.Atoi(stringSplit[11])
	if err != nil {
		return nil, err
	}
	retFlowEvent.End = secondsEnd
	retFlowEvent.LogStatus = stringSplit[13]

	if strings.Contains(*event.Message, "NODATA") {
		return retFlowEvent, nil
	}

	//fmt.Println("  ", *event.Message)

	ipSrc := net.ParseIP(stringSplit[3])
	ipDst := net.ParseIP(stringSplit[4])
	if ipSrc == nil || ipDst == nil {
		return nil, fmt.Errorf("srcaddr: %v, dstaddr: %v ", stringSplit[3], stringSplit[4])
	}

	retFlowEvent.SrcAddr = ipSrc
	retFlowEvent.DstAddr = ipDst

	srcPort, err := strconv.Atoi(stringSplit[5])
	if err != nil {
		return nil, err
	}
	retFlowEvent.SrcPort = srcPort

	dstPort, err := strconv.Atoi(stringSplit[6])
	if err != nil {
		return nil, err
	}
	retFlowEvent.DstPort = dstPort

	retFlowEvent.Protocol = stringSplit[7]

	packets, err := strconv.Atoi(stringSplit[8])
	if err != nil {
		return nil, err
	}
	retFlowEvent.Packets = packets

	bytes, err := strconv.Atoi(stringSplit[9])
	if err != nil {
		return nil, err
	}
	retFlowEvent.Bytes = bytes

	retFlowEvent.Action = stringSplit[12]

	return retFlowEvent, nil
}

func (awsTCPDump *AWSTCPDump) EventsEchoWriter(event *FlowLogEvent) error {

	logger := &(logger.Logger{FileDst: awsTCPDump.Config.ProcessedOutputFilePath, AddLogLevel: false})
	jsonBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("error marshaling NetworkInterface to JSON: %v", err)
	}

	var niMap map[string]any
	err = json.Unmarshal(jsonBytes, &niMap)
	if err != nil {
		return fmt.Errorf("error unmarshaling JSON to map: %v", err)

	}
	logger.InfoM(niMap)
	return nil
}

func (awsTCPDump *AWSTCPDump) EventsEchoWriterUTCTime(event *FlowLogEvent) error {

	logger := &(logger.Logger{FileDst: awsTCPDump.Config.ProcessedOutputFilePath, AddLogLevel: false})
	jsonBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("error marshaling NetworkInterface to JSON: %v", err)
	}

	var niMap map[string]any
	err = json.Unmarshal(jsonBytes, &niMap)
	if err != nil {
		return fmt.Errorf("error unmarshaling JSON to map: %v", err)

	}

	float64Start, ok := niMap["Start"].(float64)
	if !ok {
		return fmt.Errorf("was not able to convert time Start %d to int", niMap["Start"])
	}

	float64End, ok := niMap["End"].(float64)
	if !ok {
		return fmt.Errorf("was not able to convert time End %d to int", niMap["End"])
	}

	startTimeUTC := time.Unix(int64(float64Start), 0).UTC()
	endTimeUTC := time.Unix(int64(float64End), 0).UTC()
	niMap["Start"] = startTimeUTC.Format(time.RFC3339)
	niMap["End"] = endTimeUTC.Format(time.RFC3339)
	logger.InfoM(niMap)
	return nil
}

func (awsTCPDump *AWSTCPDump) GenerateSubnetFilter(subnetStrings []string) func(*FlowLogEvent) (*FlowLogEvent, error) {
	return func(event *FlowLogEvent) (*FlowLogEvent, error) {
		CIDRSubnets := []*net.IPNet{}
		for _, subnetString := range subnetStrings {
			_, netSrc, err := net.ParseCIDR(subnetString)

			CIDRSubnets = append(CIDRSubnets, netSrc)
			if err != nil {
				return nil, nil
			}
		}

		for _, CIDRSubnet := range CIDRSubnets {

			if CIDRSubnet.Contains(event.DstAddr) {
				return event, nil
			}
		}
		return event, nil
	}
}
