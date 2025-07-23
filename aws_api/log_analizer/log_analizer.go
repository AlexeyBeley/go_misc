package aws_api

import (
	"flag"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/AlexeyBeley/go_common/logger"
	clients "github.com/AlexeyBeley/go_misc/aws_api/clients"
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
	Region         *string
	LambdaName     *string
	OutputFilePath *string
	AWSProfile     *string
}

func ParseArgsLambdaLogsAction(flagset *flag.FlagSet, args []string) (*LambdaLogsActionInput, error) {
	ret := &LambdaLogsActionInput{}

	//required
	ret.Region = flagset.String("Region", "", "AWS Region")
	ret.LambdaName = flagset.String("LambdaName", "", "AWS Region")
	ret.AWSProfile = flagset.String("AWSProfile", "default", "AWS profile")
	ret.OutputFilePath = flagset.String("OutputFilePath", "/opt/aws_api/log_analyzer/output/out", "Output file path")

	err := flagset.Parse(args)
	if err != nil {
		return nil, err
	}

	if *ret.Region == "" {
		return nil, fmt.Errorf("region is required in %s", args)
	}

	return ret, nil
}

func (AwsLogAnalizer *AWSLogAnalizer) LambdaLogsAction() error {
	flagset := flag.NewFlagSet("LambdaLogsActionFlagSet", flag.ExitOnError)

	input, err := ParseArgsLambdaLogsAction(flagset, os.Args[1:])
	if err != nil {
		return err
	}
	lg.InfoF("Parsed the args to input: %s", input)
	return nil
}

func (AwsLogAnalizer *AWSLogAnalizer) GetLambdaLogs(lambdaName *string, dateTime *time.Time) ([]*string, error) {
	clients.LambdaAPINew(nil, nil)
	return nil, nil
}

type ActionManager struct {
	ActionMap map[string]any
}

func (AwsLogAnalizer *AWSLogAnalizer) GenerateActionManager() (*ActionManager, error) {
	ret := &ActionManager{ActionMap: map[string]any{"GetLamdaLogs": AwsLogAnalizer.LambdaLogsAction}}
	return ret, nil
}

func (actionManager *ActionManager) RunAction(actionName *string) error {
	fn, ok := actionManager.ActionMap[*actionName]
	if !ok {
		return fmt.Errorf("action '%s' not found", *actionName)
	}

	funcValue := reflect.ValueOf(fn)
	if funcValue.Kind() != reflect.Func {
		return fmt.Errorf("'%s' is not a function", *actionName)
	}

	in := make([]reflect.Value, 0)
	results := funcValue.Call(in)
	if len(results) != 1 {
		return fmt.Errorf("only result acceptable is 'error', recived %d results", len(results))
	}

	result := results[0].Interface()
	err, ok := result.(error)
	if !ok {
		return fmt.Errorf("action '%s' result expected to be 'error' but received %v", *actionName, result)
	}

	return err
}
