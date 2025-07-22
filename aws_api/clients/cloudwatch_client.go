package aws_api

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

type CloudwatchAPI struct {
	svc *cloudwatch.CloudWatch
}

func CloudwatchAPINew(region, profileName *string) *CloudwatchAPI {
	if profileName == nil {
		profileNameString := "default"
		profileName = &profileNameString
	}
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            aws.Config{Region: region},
	}))
	ret := CloudwatchAPI{svc: cloudwatch.New(sess)}
	return &ret
}

func (api *CloudwatchAPI) GetMetricAlarms(callback GenericCallback, Input *cloudwatch.DescribeAlarmsInput) error {
	var callbackErr error
	pageNum := 0
	err := api.svc.DescribeAlarmsPages(Input, func(page *cloudwatch.DescribeAlarmsOutput, notHasNextPage bool) bool {

		pageNum++
		for _, object := range page.MetricAlarms {
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

func (api *CloudwatchAPI) GetTags(Input *cloudwatch.ListTagsForResourceInput) (map[string]*string, error) {
	ret := map[string]*string{}
	response, err := api.svc.ListTagsForResource(Input)
	if err != nil {
		return nil, err
	}
	for _, tag := range response.Tags {
		ret[*tag.Key] = tag.Value
	}
	return ret, nil
}

func (api *CloudwatchAPI) ProvisionTags(alarm *cloudwatch.MetricAlarm, DesiredTags map[string]*string) error {
	missingTags := map[string]*string{}
	currentTags, err := api.GetTags(&cloudwatch.ListTagsForResourceInput{ResourceARN: alarm.AlarmArn})
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

	Tags := []*cloudwatch.Tag{}
	for tagKey, tagValue := range missingTags {
		Tag := cloudwatch.Tag{Key: &tagKey, Value: tagValue}
		Tags = append(Tags, &Tag)

	}
	req := cloudwatch.TagResourceInput{ResourceARN: alarm.AlarmArn, Tags: Tags}
	lg.InfoF("Adding tags: resource: %s, tags: %v, current tags: %v", *alarm.AlarmArn, missingTags, currentTags)
	_, err = api.svc.TagResource(&req)
	if err != nil {
		return err
	}
	return err
}
