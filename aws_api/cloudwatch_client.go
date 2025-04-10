package aws_api

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

type Configuration struct {
	Region   string `json:"Region"`
	LogGroup string `json:"LogGroup"`
}

func LoadConfig(configFilePath string) (config Configuration, err error) {
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}
	return config, nil
}

func FetchCloudwatchLogStream(region, logGroupName, streamName string) error {
	limit := int64(100)
	if logGroupName == "" || streamName == "" {
		fmt.Println("You must supply a log group name and log stream name")
		return nil
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            aws.Config{Region: &region},
	}))

	var nextToken *string
	var gotToken *string
	var StartTime, EndTime *int64
	for {
		resp, err := GetLogEventsRaw(sess, &limit, &logGroupName, &streamName, nextToken, StartTime, EndTime)
		if err != nil {
			fmt.Println("Got error getting log events:")
			fmt.Println(err)
			return nil
		}

		for _, event := range resp.Events {
			fmt.Println("  ", *event.Message)
		}

		gotToken = resp.NextForwardToken

		if nextToken == nil {
			nextToken = gotToken
			continue
		}

		if *gotToken == *nextToken {
			break
		}

		nextToken = gotToken
		fmt.Printf("Fetched")

	}

	return nil
}

func GetLogEventsRaw(sess *session.Session, limit *int64, logGroupName *string, logStreamName *string, NextToken *string, StartTime, EndTime *int64) (*cloudwatchlogs.GetLogEventsOutput, error) {
	svc := cloudwatchlogs.New(sess)
	StartFromHead := true
	Unmask := true

	resp, err := svc.GetLogEvents(&cloudwatchlogs.GetLogEventsInput{
		Limit:         limit,
		LogGroupName:  logGroupName,
		LogStreamName: logStreamName,
		NextToken:     NextToken,
		StartFromHead: &StartFromHead,
		Unmask:        &Unmask,
		StartTime: StartTime,
		EndTime: EndTime,
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

type EventHandlerCallback func(*cloudwatchlogs.OutputLogEvent) error

func YieldCloudwatchLogStream(region, logGroupName, streamName string, startTime, endTime *int64, callback EventHandlerCallback) error {
	limit := int64(100)
	counter := 0

	if logGroupName == "" || streamName == "" {
		fmt.Println("You must supply a log group name (-g LOG-GROUP) and log stream name (-s LOG-STREAM)")
		return nil
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            aws.Config{Region: &region},
	}))

	var nextToken *string
	var gotToken *string

	for {
		resp, err := GetLogEventsRaw(sess, &limit, &logGroupName, &streamName, nextToken, startTime, endTime)
		if err != nil {
			fmt.Printf("Got error getting log events: %s\n", err)
			return err
		}

		for _, event := range resp.Events {
			counter++
			callback(event)
		}

		gotToken = resp.NextForwardToken
		if gotToken == nil{
			if len(resp.Events) > 0 {
				return fmt.Errorf("unexpected state: gotToken is nil while len(resp.Events)>0 for stream: %s", streamName)
			}
			return nil
		}

		if nextToken == nil {
			nextToken = gotToken
			continue
		}

		if *gotToken == *nextToken {
			break
		}

		nextToken = gotToken
		fmt.Printf("Fetched events: %d\n", counter)

	}

	return nil
}

func BytesSummarizer(aggregator *int) func(*cloudwatchlogs.OutputLogEvent) error {
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
		*aggregator += bytes
		return nil
	}
}

func SubnetFlowStreamByteSumCallback(region, logGroupName string) func(LogStream *cloudwatchlogs.LogStream) error {
	return func(LogStream *cloudwatchlogs.LogStream) error {
		sum := 0

		nowUTC := time.Now().UTC()
		epochEndSeconds := nowUTC.Unix()
		epochStartSeconds := epochEndSeconds - 24*60*60
		//LogStream.FirstEventTimestamp
		epochEndMiliSeconds := epochEndSeconds * 1000
		epochStartMiliSeconds := epochStartSeconds * 1000
		
		if *LogStream.LastEventTimestamp < epochStartSeconds {
			return nil
		}

		YieldCloudwatchLogStream(region, logGroupName, *LogStream.LogStreamName, &epochStartMiliSeconds, &epochEndMiliSeconds, BytesSummarizer(&sum))

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

func SubnetsFlowStreamByteSum(region, logGroupName string) error {
	return YieldCloudwatchLogStreams(region, logGroupName, SubnetFlowStreamByteSumCallback(region, logGroupName))
}

type StringCallback func(string) error
type LogStreamCallback func(*cloudwatchlogs.LogStream) error

func GetLogStreamsRaw(sess *session.Session, limit *int64, logGroupName *string, callback LogStreamCallback) error {
	svc := cloudwatchlogs.New(sess)
	var callbackErr error
	pageNum := 0
	err := svc.DescribeLogStreamsPages(&cloudwatchlogs.DescribeLogStreamsInput{
		Limit:        limit,
		LogGroupName: logGroupName,
	}, func(page *cloudwatchlogs.DescribeLogStreamsOutput, notHasNextPage bool) bool {
		// stop when returns False
		pageNum++
		for _, logStream := range page.LogStreams {
			if callbackErr = callback(logStream); callbackErr != nil {
				return false
			}
			fmt.Printf("logStream: %v\n", logStream)

		}
		return !notHasNextPage
	})
	if callbackErr != nil {
		return callbackErr
	}

	return err
}

func YieldCloudwatchLogStreams(region, logGroupName string, callback LogStreamCallback) error {
	limit := int64(50)

	if logGroupName == "" {
		return fmt.Errorf("you must supply a log group name: '%s'", logGroupName)
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            aws.Config{Region: &region},
	}))

	err := GetLogStreamsRaw(sess, &limit, &logGroupName, callback)
	if err != nil {
		fmt.Println("Got error getting log events:")
		fmt.Println(err)
		return err
	}

	return nil
}

func LogStreamsCacheCallback(LogStream *cloudwatchlogs.LogStream) error {
	return nil
}

func Cacher() func(*cloudwatchlogs.LogStream) error {
	counter := 0
	return func(LogStream *cloudwatchlogs.LogStream) error {
		counter++
		fmt.Printf("Counter: %d\n", counter)
		return nil
	}
}

func LogStreamsCache(region, logGroupName string) error {
	return YieldCloudwatchLogStreams(region, logGroupName, Cacher())
}
