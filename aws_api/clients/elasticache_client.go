package aws_api

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elasticache"
)

type ElasticacheAPI struct {
	svc *elasticache.ElastiCache
}

func ElasticacheAPINew(region *string, profileName *string) *ElasticacheAPI {
	if profileName == nil {
		profileNameString := "default"
		profileName = &profileNameString
	}
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            aws.Config{Region: region},
	}))
	ret := ElasticacheAPI{svc: elasticache.New(sess)}
	return &ret
}

func (api *ElasticacheAPI) YieldCacheClusters(callback GenericCallback, Input *elasticache.DescribeCacheClustersInput) error {
	var callbackErr error
	err := api.svc.DescribeCacheClustersPages(Input, func(page *elasticache.DescribeCacheClustersOutput, notHasNextPage bool) bool {
		for _, cluster := range page.CacheClusters {
			if callbackErr = callback(cluster); callbackErr != nil {
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

func (api *ElasticacheAPI) GetCacheClusters(Input *elasticache.DescribeCacheClustersInput) ([]*elasticache.CacheCluster, error) {
	objects := make([]any, 0)

	err := api.YieldCacheClusters(AggregatorInitializer(&objects), nil)
	ret := []*elasticache.CacheCluster{}
	for _, objAny := range objects {
		obj, ok := objAny.(*elasticache.CacheCluster)
		if !ok {
			return nil, fmt.Errorf("casting error: %v ", objAny)
		}
		ret = append(ret, obj)
	}
	return ret, err
}

func (api *ElasticacheAPI) GetTags(cluster *elasticache.CacheCluster) (map[string]*string, error) {

	ret := map[string]*string{}
	output, err := api.svc.ListTagsForResource(&elasticache.ListTagsForResourceInput{ResourceName: cluster.ARN})
	if err != nil {
		return nil, err
	}
	for _, Tag := range output.TagList {
		ret[*Tag.Key] = Tag.Value
	}

	return ret, err
}

func (api *ElasticacheAPI) ProvisionTags(cluster *elasticache.CacheCluster, DesiredTags map[string]*string) error {
	missingTags := map[string]*string{}
	currentTags, err := api.GetTags(cluster)
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

	Tags := []*elasticache.Tag{}
	for tagKey, tagValue := range missingTags {
		Tag := elasticache.Tag{Key: &tagKey, Value: tagValue}
		Tags = append(Tags, &Tag)

	}
	req := elasticache.AddTagsToResourceInput{ResourceName: cluster.ARN, Tags: Tags}
	lg.InfoF("Adding tags: resource: %s, tags: %v, current tags: %v", *cluster.ARN, missingTags, currentTags)
	_, err = api.svc.AddTagsToResource(&req)
	if err != nil {
		return err
	}
	return err
}
