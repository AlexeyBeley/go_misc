package aws_api

import (
	"encoding/json"
	"fmt"
	"os"

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

type CloudwatchLogsAPI struct {
	svc *cloudwatchlogs.CloudWatchLogs
}

func CloudwatchLogsAPINew(region *string, profileName *string) *CloudwatchLogsAPI {
	if profileName == nil {
		profileNameString := "default"
		profileName = &profileNameString
	}
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            aws.Config{Region: region},
	}))
	ret := CloudwatchLogsAPI{svc: cloudwatchlogs.New(sess)}
	return &ret
}

func (api *CloudwatchLogsAPI) ProvisionLogGroup(logGroupName string) (*cloudwatchlogs.CreateLogGroupOutput, error) {
	input := cloudwatchlogs.CreateLogGroupInput{LogGroupName: &logGroupName}
	response, err := api.svc.CreateLogGroup(&input)
	return response, err
}

func GetCloudwatchLogClient(region *string) *cloudwatchlogs.CloudWatchLogs {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            aws.Config{Region: region},
	}))
	return cloudwatchlogs.New(sess)
}

func FetchCloudwatchLogStream(region, logGroupName, streamName string) error {
	limit := int64(100)
	if logGroupName == "" || streamName == "" {
		fmt.Println("You must supply a log group name and log stream name")
		return nil
	}

	svc := GetCloudwatchLogClient(&region)

	var nextToken *string
	var gotToken *string
	var StartTime, EndTime *int64
	for {
		resp, err := GetLogEventsRaw(svc, &limit, &logGroupName, &streamName, nextToken, StartTime, EndTime)
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

func GetLogEventsRaw(svc *cloudwatchlogs.CloudWatchLogs, limit *int64, logGroupName *string, logStreamName *string, NextToken *string, StartTime, EndTime *int64) (*cloudwatchlogs.GetLogEventsOutput, error) {

	StartFromHead := true
	Unmask := true

	resp, err := svc.GetLogEvents(&cloudwatchlogs.GetLogEventsInput{
		Limit:         limit,
		LogGroupName:  logGroupName,
		LogStreamName: logStreamName,
		NextToken:     NextToken,
		StartFromHead: &StartFromHead,
		Unmask:        &Unmask,
		StartTime:     StartTime,
		EndTime:       EndTime,
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

type EventHandlerCallback func(*cloudwatchlogs.OutputLogEvent) error

func YieldCloudwatchLogStream(region, logGroupName, streamName, nextToken *string, startTime, endTime *int64, callback EventHandlerCallback) (*cloudwatchlogs.GetLogEventsOutput, error) {
	limit := int64(100)

	if *logGroupName == "" || *streamName == "" {
		return nil, fmt.Errorf("you must supply a log group name and log stream name: %s, %s", *logGroupName, *streamName)
	}

	var gotToken *string
	svc := GetCloudwatchLogClient(region)
	for {
		resp, err := GetLogEventsRaw(svc, &limit, logGroupName, streamName, nextToken, startTime, endTime)
		if err != nil {
			fmt.Printf("Got error getting log events: %s\n", err)
			return nil, err
		}

		for _, event := range resp.Events {
			callback(event)
		}

		gotToken = resp.NextForwardToken
		if gotToken == nil {
			if len(resp.Events) > 0 {
				return resp, fmt.Errorf("unexpected state: gotToken is nil while len(resp.Events)>0 for stream: %s", *streamName)
			}
			return resp, nil
		}

		if nextToken == nil {
			nextToken = gotToken
			continue
		}

		if *gotToken == *nextToken {
			return resp, nil
		}

		nextToken = gotToken
	}
}

type StringCallback func(string) error

func GetLogStreamsRaw(svc *cloudwatchlogs.CloudWatchLogs, limit *int64, logGroupName, LogStreamNamePrefix *string, callback GenericCallback) error {
	var callbackErr error
	pageNum := 0
	err := svc.DescribeLogStreamsPages(&cloudwatchlogs.DescribeLogStreamsInput{
		Limit:               limit,
		LogGroupName:        logGroupName,
		LogStreamNamePrefix: LogStreamNamePrefix,
	}, func(page *cloudwatchlogs.DescribeLogStreamsOutput, notHasNextPage bool) bool {
		// stop when returns False
		pageNum++
		for _, logStream := range page.LogStreams {
			if callbackErr = callback(logStream); callbackErr != nil {
				return false
			}
		}
		return !notHasNextPage
	})
	if callbackErr != nil {
		return callbackErr
	}

	return err
}

func YieldCloudwatchLogStreams(region, logGroupName string, callback GenericCallback) error {
	limit := int64(50)

	if logGroupName == "" {
		return fmt.Errorf("you must supply a log group name: '%s'", logGroupName)
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            aws.Config{Region: &region},
	}))
	svc := cloudwatchlogs.New(sess)
	err := GetLogStreamsRaw(svc, &limit, &logGroupName, nil, callback)
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

func LogStreamsCache(region, logGroupName string) error {
	return YieldCloudwatchLogStreams(region, logGroupName, Counter())
}

func (api *CloudwatchLogsAPI) GetLogGroup(name *string) (logGroup *cloudwatchlogs.LogGroup, err error) {
	input := cloudwatchlogs.DescribeLogGroupsInput{LogGroupNamePattern: name}
	response, err := api.svc.DescribeLogGroups(&input)
	if len(response.LogGroups) > 1 {
		return logGroup, fmt.Errorf("found %d log groups by name: %s", len(response.LogGroups), *name)
	}
	if len(response.LogGroups) == 1 {
		return response.LogGroups[0], nil
	}
	return logGroup, err
}
