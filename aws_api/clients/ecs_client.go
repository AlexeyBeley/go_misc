package aws_api

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
)

type ECSAPI struct {
	svc         *ecs.ECS
	region      *string
	profileName *string
}

func ECSAPINew(region *string, profileName *string) *ECSAPI {
	if profileName == nil {
		profileNameString := "default"
		profileName = &profileNameString
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            aws.Config{Region: region},
		Profile:           *profileName,
	}))

	lg.InfoF("AWS profile: %s\n", *profileName)
	svc := ecs.New(sess)
	ret := ECSAPI{svc: svc, region: region, profileName: profileName}
	return &ret
}

func (api *ECSAPI) GetTaskDefinitions(callback GenericCallback, Input *ecs.ListTaskDefinitionsInput) error {
	var callbackErr error
	pageNum := 0
	err := api.svc.ListTaskDefinitionsPages(Input, func(page *ecs.ListTaskDefinitionsOutput, notHasNextPage bool) bool {

		pageNum++
		for _, arn := range page.TaskDefinitionArns {
			response, err := api.svc.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{TaskDefinition: arn})
			if err != nil {
				return false
			}

			tags, err := api.svc.ListTagsForResource(&ecs.ListTagsForResourceInput{ResourceArn: arn})
			if err != nil {
				return false
			}
			fmt.Printf("todo: handle tags in GetTaskDefinitions %s", tags)
			if callbackErr = callback(response.TaskDefinition); callbackErr != nil {
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

func (api *ECSAPI) GetTaskDefinitionFamilies(callback GenericCallback, Input *ecs.ListTaskDefinitionFamiliesInput) error {
	var callbackErr error
	pageNum := 0
	err := api.svc.ListTaskDefinitionFamiliesPages(Input, func(page *ecs.ListTaskDefinitionFamiliesOutput, notHasNextPage bool) bool {

		pageNum++
		for _, arn := range page.Families {
			if callbackErr = callback(arn); callbackErr != nil {
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
