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
	Region  string `json:"Region"`
	Profile string `json:"Profile"`
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

func (api *CloudwatchLogsAPI) YieldCloudwatchLogStreams(input *cloudwatchlogs.DescribeLogStreamsInput, callback GenericCallbackNG) error {

	if *input.LogGroupName == "" {
		return fmt.Errorf("you must supply a log group name: '%s'", *input.LogGroupName)
	}
	if input.Limit == nil {
		input.Limit = Int64Ptr(50)
	}

	var callbackErr error
	pageNum := 0
	err := api.svc.DescribeLogStreamsPages(input, func(page *cloudwatchlogs.DescribeLogStreamsOutput, notHasNextPage bool) bool {
		// stop when returns False
		lg.DebugF("DescribeLogStreamsPages, page %d", pageNum)
		pageNum++
		for _, logStream := range page.LogStreams {
			if continuePagination, err := callback(logStream); !continuePagination {
				callbackErr = err
				return false
			}
		}
		return !notHasNextPage
	})

	if err != nil {
		return err
	}

	if callbackErr != nil {
		return callbackErr
	}

	return nil
}

func (api *CloudwatchLogsAPI) YieldStreamEvents(input *cloudwatchlogs.GetLogEventsInput, callback GenericCallbackNG) error {

	if *input.LogGroupName == "" {
		return fmt.Errorf("you must supply a log group name: '%s'", *input.LogGroupName)
	}
	if input.Limit == nil {
		input.Limit = Int64Ptr(10000)
	}

	var callbackErr error
	pageNum := 0
	err := api.svc.GetLogEventsPages(input, func(page *cloudwatchlogs.GetLogEventsOutput, notHasNextPage bool) bool {
		// stop when returns False
		lg.DebugF("GetLogEventsPages, page %d", pageNum)
		pageNum++
		for _, event := range page.Events {
			if continuePagination, err := callback(event); !continuePagination {
				callbackErr = err
				return false
			}
		}
		return !notHasNextPage
	})

	if err != nil {
		return err
	}

	if callbackErr != nil {
		return callbackErr
	}

	return nil
}

func (api *CloudwatchLogsAPI) GetLogGroup(name *string) (logGroup *cloudwatchlogs.LogGroup, err error) {
	input := cloudwatchlogs.DescribeLogGroupsInput{LogGroupNamePattern: name}
	response, err := api.svc.DescribeLogGroups(&input)
	if len(response.LogGroups) > 1 {
		return logGroup, fmt.Errorf("found %d log groups by name: %s", len(response.LogGroups), *name)
	}
	if err != nil {
		return nil, err
	}

	if len(response.LogGroups) == 1 {
		return response.LogGroups[0], nil
	}
	return logGroup, err
}

func (api *CloudwatchLogsAPI) YieldLogGroups(callback GenericCallbackNG, Input *cloudwatchlogs.DescribeLogGroupsInput) error {
	var callbackErr error
	pageNum := 0
	err := api.svc.DescribeLogGroupsPages(Input, func(page *cloudwatchlogs.DescribeLogGroupsOutput, notHasNextPage bool) bool {

		pageNum++
		for _, logGroup := range page.LogGroups {
			if continuePagination, err := callback(logGroup); !continuePagination {
				callbackErr = err
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

func (api *CloudwatchLogsAPI) GetLogGroups(Input *cloudwatchlogs.DescribeLogGroupsInput) ([]*cloudwatchlogs.LogGroup, error) {
	objects := []any{}
	err := api.YieldLogGroups(AggregatorInitializerNG(&objects), nil)
	if err != nil {
		return nil, err
	}
	logGroups := []*cloudwatchlogs.LogGroup{}
	for _, objAny := range objects {
		obj, ok := objAny.(*cloudwatchlogs.LogGroup)
		if !ok {
			return nil, fmt.Errorf("cast error: %v", objAny)
		}
		logGroups = append(logGroups, obj)
	}

	return logGroups, nil
}

func (api *CloudwatchLogsAPI) GetTags(Input *cloudwatchlogs.ListTagsForResourceInput) (map[string]*string, error) {
	response, err := api.svc.ListTagsForResource(Input)
	if err != nil {
		return nil, err
	}

	return response.Tags, nil
}

func (api *CloudwatchLogsAPI) ProvisionTags(targetGroup *cloudwatchlogs.LogGroup, DesiredTags map[string]*string) error {
	missingTags := map[string]*string{}
	currentTags, err := api.GetTags(&cloudwatchlogs.ListTagsForResourceInput{ResourceArn: targetGroup.LogGroupArn})
	if err != nil {
		return nil
	}

	for desiredKey, desiredValue := range DesiredTags {
		if currentValue, found := currentTags[desiredKey]; !found || *currentValue != *desiredValue {
			missingTags[desiredKey] = desiredValue
		}
	}

	if len(missingTags) == 0 {
		return nil
	}
	req := cloudwatchlogs.TagResourceInput{ResourceArn: targetGroup.LogGroupArn, Tags: missingTags}
	lg.InfoF("Adding tags: resource: %s, tags: %v, current tags: %v", *targetGroup.LogGroupArn, missingTags, currentTags)
	_, err = api.svc.TagResource(&req)
	if err != nil {
		return err
	}
	return err
}

func (api *CloudwatchLogsAPI) DisposeStream(Input *cloudwatchlogs.DeleteLogStreamInput) (*cloudwatchlogs.DeleteLogStreamOutput, error) {
	lg.InfoF("Disposing log stream %s -> %s", *Input.LogGroupName, *Input.LogStreamName)
	response, err := api.svc.DeleteLogStream(Input)
	if err != nil {
		return nil, err
	}

	return response, nil
}
