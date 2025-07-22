package aws_api

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type DynamoDBAPI struct {
	svc *dynamodb.DynamoDB
}

func DynamoDBAPINew(region *string, profileName *string) *DynamoDBAPI {
	if profileName == nil {
		profileNameString := "default"
		profileName = &profileNameString
	}
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            aws.Config{Region: region},
	}))
	ret := DynamoDBAPI{svc: dynamodb.New(sess)}
	return &ret
}

func (api *DynamoDBAPI) YieldTables(callback GenericCallback, Input *dynamodb.ListTablesInput) error {
	var callbackErr error
	pageNum := 0
	err := api.svc.ListTablesPages(Input, func(page *dynamodb.ListTablesOutput, notHasNextPage bool) bool {

		pageNum++
		for _, objName := range page.TableNames {
			obj, err := api.svc.DescribeTable(&dynamodb.DescribeTableInput{TableName: objName})
			if err != nil {
				callbackErr = err
				return false
			}

			if callbackErr = callback(obj.Table); callbackErr != nil {
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

func (api *DynamoDBAPI) GetTables(Input *dynamodb.ListTablesInput) ([]*dynamodb.TableDescription, error) {
	objects := make([]any, 0)

	err := api.YieldTables(AggregatorInitializer(&objects), nil)
	ret := []*dynamodb.TableDescription{}
	for _, objAny := range objects {
		obj, ok := objAny.(*dynamodb.TableDescription)
		if !ok {
			return nil, fmt.Errorf("Casting error: %v ", objAny)
		}
		ret = append(ret, obj)
	}
	return ret, err
}

func (api *DynamoDBAPI) GetTags(table *dynamodb.TableDescription) (map[string]*string, error) {

	ret := map[string]*string{}
	output, err := api.svc.ListTagsOfResource(&dynamodb.ListTagsOfResourceInput{ResourceArn: table.TableArn})
	if err != nil {
		return nil, err
	}
	for _, Tag := range output.Tags {
		ret[*Tag.Key] = Tag.Value
	}

	return ret, err
}

func (api *DynamoDBAPI) ProvisionTags(table *dynamodb.TableDescription, DesiredTags map[string]*string) error {
	missingTags := map[string]*string{}
	currentTags, err := api.GetTags(table)
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

	Tags := []*dynamodb.Tag{}
	for tagKey, tagValue := range missingTags {
		Tag := dynamodb.Tag{Key: &tagKey, Value: tagValue}
		Tags = append(Tags, &Tag)

	}
	req := dynamodb.TagResourceInput{ResourceArn: table.TableArn, Tags: Tags}
	lg.InfoF("Adding tags: resource: %s, tags: %v, current tags: %v", *table.TableArn, missingTags, currentTags)
	_, err = api.svc.TagResource(&req)
	if err != nil {
		return err
	}
	return err
}
