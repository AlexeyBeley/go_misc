package aws_api

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
)

type RDSAPI struct {
	svc *rds.RDS
}

func RDSAPINew(region *string, profileName *string) *RDSAPI {
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
	svc := rds.New(sess)
	ret := RDSAPI{svc: svc}
	return &ret
}

func (api *RDSAPI) DescribeClusters(callback GenericCallback, Input *rds.DescribeDBClustersInput) error {
	var callbackErr error
	pageNum := 0
	err := api.svc.DescribeDBClustersPages(Input, func(page *rds.DescribeDBClustersOutput, notHasNextPage bool) bool {
		pageNum++
		for _, obj := range page.DBClusters {
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

func (api *RDSAPI) DescribeInstances(callback GenericCallback, Input *rds.DescribeDBInstancesInput) error {
	var callbackErr error
	pageNum := 0
	err := api.svc.DescribeDBInstancesPages(Input, func(page *rds.DescribeDBInstancesOutput, notHasNextPage bool) bool {
		pageNum++
		for _, obj := range page.DBInstances {
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

func (api *RDSAPI) CreateTags(existingTags []*rds.Tag, AddTags map[string]string, resource *string, declarative bool) (*rds.AddTagsToResourceOutput, error) {
	Tags := []*rds.Tag{}
	for key, value := range AddTags {
		found := false
		for _, existingTag := range existingTags {
			if *existingTag.Key == key && (*existingTag.Value == value || !declarative) {
				found = true
				break
			}
		}

		if !found {
			Tags = append(Tags, &rds.Tag{Key: &key, Value: &value})
		}
	}
	if len(Tags) == 0 {
		return nil, nil
	}
	lg.InfoF("Adding tags: resource: %s, tags: %v, Current tags: %v", *resource, Tags, existingTags)
	createTagsOutput, err := api.svc.AddTagsToResource(&rds.AddTagsToResourceInput{ResourceName: resource, Tags: Tags})
	return createTagsOutput, err
}
