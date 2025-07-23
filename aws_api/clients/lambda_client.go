package aws_api

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
)

type LambdaAPI struct {
	svc *lambda.Lambda
}

func LambdaAPINew(region *string, profileName *string) *LambdaAPI {
	if profileName == nil {
		profileNameString := "default"
		profileName = &profileNameString
	}
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            aws.Config{Region: region},
	}))
	ret := LambdaAPI{svc: lambda.New(sess)}
	return &ret
}

func (api *LambdaAPI) YieldFunctions(callback GenericCallback, Input *lambda.ListFunctionsInput) error {
	var callbackErr error
	pageNum := 0
	err := api.svc.ListFunctionsPages(Input, func(page *lambda.ListFunctionsOutput, notHasNextPage bool) bool {
		pageNum++
		for _, object := range page.Functions {

			if callbackErr = callback(object); callbackErr != nil {
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

func (api *LambdaAPI) GetFunctions(Input *lambda.ListFunctionsInput) ([]*lambda.FunctionConfiguration, error) {
	objects := make([]any, 0)

	err := api.YieldFunctions(AggregatorInitializer(&objects), nil)
	ret := []*lambda.FunctionConfiguration{}
	for _, objAny := range objects {
		obj, ok := objAny.(*lambda.FunctionConfiguration)
		if !ok {
			return nil, fmt.Errorf("casting error: %v ", objAny)
		}
		ret = append(ret, obj)
	}
	return ret, err
}
