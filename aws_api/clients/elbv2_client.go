package aws_api

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
)

type ELBV2API struct {
	svc *elbv2.ELBV2
}

func ELBV2APINew(region *string, profileName *string) *ELBV2API {
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
	svc := elbv2.New(sess)
	ret := ELBV2API{svc: svc}
	return &ret
}

func (api *ELBV2API) DescribeLoadBalancers(callback GenericCallback, Input *elbv2.DescribeLoadBalancersInput) error {
	var callbackErr error
	pageNum := 0
	err := api.svc.DescribeLoadBalancersPages(Input, func(page *elbv2.DescribeLoadBalancersOutput, notHasNextPage bool) bool {
		pageNum++
		for _, obj := range page.LoadBalancers {
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

func (api *ELBV2API) DescribeTargetGroups(callback GenericCallback, Input *elbv2.DescribeTargetGroupsInput) error {
	var callbackErr error
	pageNum := 0
	err := api.svc.DescribeTargetGroupsPages(Input, func(page *elbv2.DescribeTargetGroupsOutput, notHasNextPage bool) bool {
		pageNum++
		for _, obj := range page.TargetGroups {
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

func (api *ELBV2API) AddTags(AddTags map[string]string, resource *string, declarative bool) (*elbv2.AddTagsOutput, error) {
	existingTags, err := api.GetTags(resource)
	if err != nil {
		return nil, err
	}

	Tags := []*elbv2.Tag{}
	for key, value := range AddTags {
		found := false
		for _, existingTag := range existingTags {
			if *existingTag.Key == key && (*existingTag.Value == value || !declarative) {
				found = true
				break
			}
		}

		if !found {
			Tags = append(Tags, &elbv2.Tag{Key: &key, Value: &value})
		}
	}
	if len(Tags) == 0 {
		return nil, nil
	}
	lg.Infof("Adding tags: resource: %s, tags: %v, Current tags: %v", *resource, Tags, existingTags)
	createTagsOutput, err := api.svc.AddTags(&elbv2.AddTagsInput{ResourceArns: []*string{resource}, Tags: Tags})
	return createTagsOutput, err
}

func (api *ELBV2API) GetTags(resource *string) ([]*elbv2.Tag, error) {
	describeTagsOutput, err := api.svc.DescribeTags(&elbv2.DescribeTagsInput{ResourceArns: []*string{resource}})

	if err != nil {
		return nil, err
	}
	ret := []*elbv2.Tag{}
	for _, tag := range describeTagsOutput.TagDescriptions {
		ret = append(ret, tag.Tags...)
	}
	return ret, nil
}
