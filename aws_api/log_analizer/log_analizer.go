package aws_api

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/AlexeyBeley/go_misc/logger"
	actionManager "github.com/AlexeyBeley/go_misc/action_manager"
	clients "github.com/AlexeyBeley/go_misc/aws_api/clients"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/lambda"
)

var lg = &(logger.Logger{})

type AWSLogAnalizerConfig struct {
	Region string `json:"Region"`
}

type AWSLogAnalizer struct{}

func AWSLogAnalizerNew() (*AWSLogAnalizer, error) {
	new := &AWSLogAnalizer{}

	return new, nil
}

type LambdaLogsActionInput struct {
	Region               *string
	LambdaName           *string
	OutputFilePath       *string
	AWSProfile           *string
	PointInTime          *int64
	LogGroup             *string
	OutputFolderPath     *string
	GroupRetentionInDays *int64
}

func ParseArgsLambdaLogsAction(flagset *flag.FlagSet, args []string) (*LambdaLogsActionInput, error) {
	ret := &LambdaLogsActionInput{}

	//required
	ret.Region = flagset.String("Region", "", "AWS Region")
	ret.LambdaName = flagset.String("LambdaName", "", "AWS Region")
	ret.AWSProfile = flagset.String("AWSProfile", "default", "AWS profile")
	dateTimeString := flagset.String("PointInTime", "", "PointInTime")
	ret.OutputFilePath = flagset.String("OutputFilePath", "/opt/aws_hapi/log_analyzer/output/out", "Output file path")

	err := flagset.Parse(args)
	if err != nil {
		return nil, err
	}

	if *ret.Region == "" {
		return nil, fmt.Errorf("region is required in %s", args)
	}

	if *dateTimeString != "" {
		parsedTime, err := time.Parse(time.RFC3339Nano, *dateTimeString)
		if err != nil {
			return nil, fmt.Errorf("error parsing time string: %v", err)
		}
		ret.PointInTime = clients.Int64Ptr(parsedTime.Unix() * 1000)
	}

	outputFolderPath := "/opt/aws_hapi/log_analyzer/output_" + *ret.Region
	ret.OutputFolderPath = flagset.String("OutputFolderPath", outputFolderPath, "Output folder path")

	err = os.Mkdir(outputFolderPath, 0755)
	if err != nil {
		if os.IsExist(err) {
			fmt.Printf("Directory '%s' already exists (Method 1).\n", outputFolderPath)
		} else {
			log.Fatalf("Error creating directory '%s' (Method 1): %v\n", outputFolderPath, err)
		}
	} else {
		fmt.Printf("Directory '%s' created successfully (Method 1).\n", outputFolderPath)
	}

	return ret, nil
}

func (AwsLogAnalizer *AWSLogAnalizer) LambdaLogsAction() error {
	flagset := flag.NewFlagSet("LambdaLogsActionFlagSet", flag.ExitOnError)

	input, err := ParseArgsLambdaLogsAction(flagset, os.Args[1:])
	if err != nil {
		return err
	}
	if input.PointInTime == nil {
		return fmt.Errorf("PointInTime is required in: %v", os.Args)
	}
	lg.InfoF("Parsed the args to input: %s", input)

	logGroup, err := AwsLogAnalizer.InitLogGroup(input)
	if err != nil {
		return err
	}
	input.LogGroup = logGroup
	ret, err := AwsLogAnalizer.GetLambdaPointInTimeLogs(input)
	if err != nil {
		return err
	}
	//todo:
	_ = ret

	return nil
}

func (AwsLogAnalizer *AWSLogAnalizer) InitLogGroup(input *LambdaLogsActionInput) (*string, error) {
	lambda_api := clients.LambdaAPINew(input.Region, input.AWSProfile)
	var foundLambda *lambda.FunctionConfiguration
	Finder := func(lammbdaAny any) (continuePagination bool, err error) {
		lambda, ok := lammbdaAny.(*lambda.FunctionConfiguration)
		if !ok {
			return false, fmt.Errorf("cast error %v ", lammbdaAny)
		}

		if *lambda.FunctionName == *input.LambdaName {
			foundLambda = lambda
			return false, nil
		}
		return true, nil
	}

	err := lambda_api.YieldFunctions(Finder, &lambda.ListFunctionsInput{})
	if err != nil {
		return nil, err
	}

	logs_api := clients.CloudwatchLogsAPINew(input.Region, input.AWSProfile)
	logGroup, err := logs_api.GetLogGroup(foundLambda.LoggingConfig.LogGroup)
	if err != nil {
		return nil, err
	}

	if logGroup.RetentionInDays == nil {
		logGroup.RetentionInDays = clients.Int64Ptr(60)
		//return nil, fmt.Errorf("does not support log group without retention")
	}

	input.GroupRetentionInDays = logGroup.RetentionInDays

	return foundLambda.LoggingConfig.LogGroup, nil
}

func GetSingleLambdaRunLogsInitializer(objects *[]any, input *LambdaLogsActionInput) func(any) (bool, error) {

	return func(streamAny any) (bool, error) {
		stream, ok := streamAny.(*cloudwatchlogs.LogStream)
		if !ok {
			return false, fmt.Errorf("cast error: %v", streamAny)
		}

		//descending means when seeing old stream - stop the pagination.
		if *stream.LastEventTimestamp < *input.PointInTime {
			return false, nil
		}

		//descending means new streams should be ignored
		if *stream.FirstEventTimestamp > *input.PointInTime {
			return true, nil
		}

		*objects = append(*objects, stream)

		return true, nil
	}
}

func (awsLogAnalizer *AWSLogAnalizer) GetLambdaPointInTimeLogs(input *LambdaLogsActionInput) ([]*string, error) {
	logs_api := clients.CloudwatchLogsAPINew(input.Region, input.AWSProfile)
	objects := make([]any, 0)

	err := logs_api.YieldCloudwatchLogStreams(&cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: input.LogGroup,
		OrderBy:      clients.StrPtr("LastEventTime"),
		Descending:   clients.BoolPtr(true),
	}, GetSingleLambdaRunLogsInitializer(&objects, input))
	if err != nil {
		return nil, err
	}

	for _, logStreamAny := range objects {
		logStream, ok := logStreamAny.(*cloudwatchlogs.LogStream)
		if !ok {
			return nil, fmt.Errorf("cast error: %v", logStreamAny)
		}
		output, err := awsLogAnalizer.GetLambdaPointInTimeLogsFromLogStream(logs_api, logStream, input)
		if err != nil {
			return nil, err
		}
		// todo:
		_ = output

	}
	return nil, nil
}

func (awsLogAnalizer *AWSLogAnalizer) GetLambdaPointInTimeLogsFromLogStream(logs_api *clients.CloudwatchLogsAPI, logStream *cloudwatchlogs.LogStream, input *LambdaLogsActionInput) ([]*string, error) {
	objects := []any{}

	logs_api.YieldStreamEvents(&cloudwatchlogs.GetLogEventsInput{
		StartFromHead: clients.BoolPtr(true),
		LogGroupName:  input.LogGroup,
		LogStreamName: logStream.LogStreamName,
		EndTime:       clients.Int64Ptr(*input.PointInTime + 15*1000*60),
		StartTime:     clients.Int64Ptr(*input.PointInTime - 15*1000*60)}, clients.AggregatorInitializerNG(&objects))

	lambdaStarted := false
	lambdaEvents := []*cloudwatchlogs.OutputLogEvent{}
	for _, eventAny := range objects {
		event, ok := eventAny.(*cloudwatchlogs.OutputLogEvent)
		if !ok {
			return nil, fmt.Errorf("cast error: %v", eventAny)
		}

		if strings.Contains(*event.Message, "START RequestId") {
			if lambdaStarted {
				return nil, fmt.Errorf("unsupported state: recording started lambda received a start lambda line: %s", *event.Message)
			}
			lambdaEvents = lambdaEvents[:0]
		}

		lambdaEvents = append(lambdaEvents, event)

		if lambdaStarted {
			if strings.Contains(*event.Message, "REPORT RequestId") {
				break
			}
			continue
		}

		if *event.Timestamp >= *input.PointInTime {
			lambdaStarted = true
		}
	}

	//*"START RequestId: f875b1f4-0646-46f5-93d6-a16d8d96b092 Version: $LATEST\n"
	startRequestId := strings.Split((*lambdaEvents[0].Message), " ")[2]

	//Why? That's WHY!!!
	// lambda start request 'f875b1f4-0646-46f5-93d6-a16d8d96b092' does not match the end request 'f875b1f4-0646-46f5-93d6-a16d8d96b092	Duration:'
	newLastEventString := strings.ReplaceAll(*lambdaEvents[len(lambdaEvents)-1].Message, "\t", " ")

	endRequestId := strings.Split(newLastEventString, " ")[2]

	if startRequestId != endRequestId {
		return nil, fmt.Errorf("lambda start request '%s' does not match the end request '%s'", startRequestId, endRequestId)
	}
	lg.InfoF("Fetched %d events", len(lambdaEvents))

	jsonData, err := json.MarshalIndent(lambdaEvents, "", "  ")
	if err != nil {
		return nil, err
	}

	err = os.WriteFile("/tmp/output.json", jsonData, 0644)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (AwsLogAnalizer *AWSLogAnalizer) GenerateActionManager() (*actionManager.ActionManager, error) {
	actionManager, err := actionManager.ActionManagerNew()
	if err != nil {
		return nil, err
	}
	(*actionManager).ActionMap = map[string]any{"GetLamdaLogs": AwsLogAnalizer.LambdaLogsAction,
		"GetAllLamdasLogs": AwsLogAnalizer.GetAllLamdasLogsAction}

	return actionManager, nil
}

func (AwsLogAnalizer *AWSLogAnalizer) GetAllLamdasLogsAction() error {
	flagset := flag.NewFlagSet("LambdaLogsActionFlagSet", flag.ExitOnError)

	input, err := ParseArgsLambdaLogsAction(flagset, os.Args[1:])
	if err != nil {
		return err
	}
	lg.InfoF("Parsed the args to input: %s", input)

	logGroup, err := AwsLogAnalizer.InitLogGroup(input)
	if err != nil {
		return err
	}
	input.LogGroup = logGroup

	ret, err := AwsLogAnalizer.GetAllLambdasLogs(input)
	if err != nil {
		return err
	}
	//todo:
	_ = ret

	return nil
}

func SaveAllLamdasLogs(logs_api *clients.CloudwatchLogsAPI, input *LambdaLogsActionInput) func(any) (bool, error) {
	var lambdaStarted = false
	lambdaEvents := []*cloudwatchlogs.OutputLogEvent{}
	StreamsCounter := 0
	return func(streamAny any) (bool, error) {
		logStream, ok := streamAny.(*cloudwatchlogs.LogStream)
		if !ok {
			return false, fmt.Errorf("cast error: %v", streamAny)
		}

		// Todo: remove
		// Skip old streams
		if *logStream.FirstEventTimestamp < 1754334000000-30*60*1000 {
			return true, nil
		}

		lg.InfoF("Fetched %d streams", StreamsCounter)

		StreamsCounter++

		err := logs_api.YieldStreamEvents(&cloudwatchlogs.GetLogEventsInput{
			StartFromHead: clients.BoolPtr(true),
			LogGroupName:  input.LogGroup,
			LogStreamName: logStream.LogStreamName,
			StartTime:     clients.Int64Ptr((time.Now().UTC().Unix() - *input.GroupRetentionInDays*24*60*60) * 1000),
		},
			func(eventAny any) (bool, error) {
				event, ok := eventAny.(*cloudwatchlogs.OutputLogEvent)
				if !ok {
					return false, fmt.Errorf("cast error: %v", eventAny)
				}

				if strings.Contains(*event.Message, "START RequestId") {
					if lambdaStarted {
						if *lambdaEvents[len(lambdaEvents)-1].Message != *event.Message {
							return false, fmt.Errorf("unsupported state: recording started lambda received a start lambda line: %s", *event.Message)
						}
					}

					lambdaStarted = true
				} else if !lambdaStarted {
					return true, nil
				}

				lambdaEvents = append(lambdaEvents, event)

				// end
				if strings.Contains(*event.Message, "REPORT RequestId") {
					filePath := *input.OutputFolderPath + "/" + strconv.Itoa(int(*lambdaEvents[0].Timestamp)) + ".json"
					jsonData, err := json.MarshalIndent(lambdaEvents, "", "  ")
					if err != nil {
						return false, err
					}

					err = os.WriteFile(filePath, jsonData, 0644)
					if err != nil {
						return false, err
					}

					lambdaEvents = lambdaEvents[:0]
					lambdaStarted = false
				}
				return true, nil

			})
		if err != nil {
			return false, nil
		}

		return true, nil
	}
}
func (awsLogAnalizer *AWSLogAnalizer) GetAllLambdasLogs(input *LambdaLogsActionInput) ([]*string, error) {
	logs_api := clients.CloudwatchLogsAPINew(input.Region, input.AWSProfile)

	err := logs_api.YieldCloudwatchLogStreams(&cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: input.LogGroup,
		OrderBy:      clients.StrPtr("LastEventTime"),
		Descending:   clients.BoolPtr(false),
	}, SaveAllLamdasLogs(logs_api, input))
	if err != nil {
		return nil, err
	}
	return nil, nil
}
