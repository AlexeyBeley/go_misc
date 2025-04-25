package aws_api

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
)

type AutoscalingAPI struct {
	svc *autoscaling.AutoScaling
}

func GetAutoscalingAPI(region *string, profileName *string) *AutoscalingAPI {
	if profileName == nil {
		profileNameString := "default"
		profileName = &profileNameString
	}
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            aws.Config{Region: region},
		Profile:           *profileName,
	}))
	lg.Infof("AWS profile: %s\n", *profileName)
	svc := autoscaling.New(sess)
	ret := AutoscalingAPI{svc: svc}
	return &ret
}

func (api *AutoscalingAPI) DescribeAutoScalingGroups(callback GenericCallback, Input *autoscaling.DescribeAutoScalingGroupsInput) error {
	var callbackErr error
	pageNum := 0
	err := api.svc.DescribeAutoScalingGroupsPages(Input, func(page *autoscaling.DescribeAutoScalingGroupsOutput, notHasNextPage bool) bool {
		pageNum++
		for _, obj := range page.AutoScalingGroups {
			if callbackErr = callback(obj); callbackErr != nil {
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

func (api *AutoscalingAPI) CreateOrUpdateTags(existingTags []*autoscaling.Tag, AddTags map[string]string, resource, resourceType *string, declarative bool) error {
	Tags := []*autoscaling.Tag{}
	propagateAtLaunch := true
	for key, value := range AddTags {
		found := false
		for _, existingTag := range existingTags {
			if *existingTag.Key == key && (*existingTag.Value == value || !declarative) {
				found = true
				break
			}
		}

		if !found {
			Tags = append(Tags, &autoscaling.Tag{Key: &key, Value: &value, ResourceId: resource, ResourceType: resourceType, PropagateAtLaunch: &propagateAtLaunch})
		}
	}

	req := autoscaling.CreateOrUpdateTagsInput{Tags: Tags}
	lg.Infof("Adding tags: resource: %s, tags: %v, Current tags: %v", *resource, Tags, existingTags)
	_, err := api.svc.CreateOrUpdateTags(&req)
	return err
}
