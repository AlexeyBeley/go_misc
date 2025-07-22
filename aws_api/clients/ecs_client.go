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

	err := api.svc.ListTaskDefinitionsPages(Input, func(page *ecs.ListTaskDefinitionsOutput, notHasNextPage bool) bool {

		for _, arn := range page.TaskDefinitionArns {
			response, err := api.svc.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{TaskDefinition: arn})
			if err != nil {
				return false
			}

			//tags, err := api.svc.ListTagsForResource(&ecs.ListTagsForResourceInput{ResourceArn: arn})
			//if err != nil {
			//	return false
			//}
			//fmt.Printf("todo: handle tags in GetTaskDefinitions %s", tags)
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

func (api *ECSAPI) GetTags(resource *string) ([]*ecs.Tag, error) {
	ListTagsForResourceOutput, err := api.svc.ListTagsForResource(&ecs.ListTagsForResourceInput{ResourceArn: resource})

	if err != nil {
		return nil, err
	}
	ret := []*ecs.Tag{}

	ret = append(ret, ListTagsForResourceOutput.Tags...)

	return ret, nil
}

func (api *ECSAPI) GetTasks(Input *ecs.ListTasksInput) ([]*ecs.Task, error) {
	var callbackErr error
	objects := make([]any, 0)
	callback := AggregatorInitializer(&objects)

	err := api.svc.ListTasksPages(Input, func(page *ecs.ListTasksOutput, notHasNextPage bool) bool {
		if len(page.TaskArns) == 100 {
			callbackErr = fmt.Errorf("not implemented len(page.TaskArns)= %d", len(page.TaskArns))
			return false
		}

		response, err := api.svc.DescribeTasks(&ecs.DescribeTasksInput{Tasks: page.TaskArns, Cluster: Input.Cluster})
		if err != nil {
			return false
		}

		for _, task := range response.Tasks {
			if callbackErr = callback(task); callbackErr != nil {
				return false
			}
		}

		return !notHasNextPage
	})

	if callbackErr != nil {
		return nil, callbackErr
	}

	if err != nil {
		return nil, err
	}
	retTasks := []*ecs.Task{}
	for _, anyTask := range objects {
		task, ok := anyTask.(*ecs.Task)
		if !ok {
			return nil, fmt.Errorf("ecs.Task cast error: %v", anyTask)
		}
		retTasks = append(retTasks, task)
	}
	return retTasks, nil
}

func (api *ECSAPI) ProvisionTags(task *ecs.Task, DesiredTags map[string]*string) error {
	missingTags := []*ecs.Tag{}
	currentTagsobj, err := api.GetTags(task.TaskArn)
	if err != nil {
		return nil
	}

	currentTags := map[string]*string{}
	for _, currentTag := range currentTagsobj {
		currentTags[*currentTag.Key] = currentTag.Value

	}

	for desiredKey, desiredValue := range DesiredTags {
		if currentValue, found := currentTags[desiredKey]; !found || *currentValue != *desiredValue {
			Tag := &ecs.Tag{Key: &desiredKey, Value: desiredValue}
			missingTags = append(missingTags, Tag)
		}
	}

	if len(missingTags) == 0 {
		return nil
	}
	req := ecs.TagResourceInput{ResourceArn: task.TaskArn, Tags: missingTags}
	lg.InfoF("Adding tags: resource: %s, tags: %v, current tags: %v", *task.TaskArn, missingTags, currentTags)
	_, err = api.svc.TagResource(&req)
	if err != nil {
		return err
	}
	return err
}

func (api *ECSAPI) IterClusters(Input *ecs.ListClustersInput) ([]*ecs.Cluster, error) {
	var callbackErr error
	objects := make([]any, 0)
	callback := AggregatorInitializer(&objects)

	err := api.svc.ListClustersPages(Input, func(page *ecs.ListClustersOutput, notHasNextPage bool) bool {
		if len(page.ClusterArns) == 100 {
			callbackErr = fmt.Errorf("not implemented len(page.ClusterArns)= %d", len(page.ClusterArns))
			return false
		}

		for _, arn := range page.ClusterArns {
			cluster := &ecs.Cluster{ClusterArn: arn}
			if callbackErr = callback(cluster); callbackErr != nil {
				return false
			}
		}

		return !notHasNextPage
	})

	if callbackErr != nil {
		return nil, callbackErr
	}

	if err != nil {
		return nil, err
	}

	retClusters := []*ecs.Cluster{}
	for _, anyCluster := range objects {
		task, ok := anyCluster.(*ecs.Cluster)
		if !ok {
			return nil, fmt.Errorf("ecs.Cluster cast error: %v", anyCluster)
		}
		retClusters = append(retClusters, task)
	}
	return retClusters, nil
}

func (api *ECSAPI) ListClusters(Input *ecs.ListClustersInput) ([]*string, error) {
	var callbackErr error
	ret := []*string{}

	err := api.svc.ListClustersPages(Input, func(page *ecs.ListClustersOutput, notHasNextPage bool) bool {
		if len(page.ClusterArns) == 100 {
			callbackErr = fmt.Errorf("not implemented len(page.ClusterArns)= %d", len(page.ClusterArns))
			return false
		}

		ret = append(ret, page.ClusterArns...)

		return !notHasNextPage
	})

	if callbackErr != nil {
		return nil, callbackErr
	}

	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (api *ECSAPI) GetClusters(Input *ecs.ListClustersInput) (ret []*ecs.Cluster, err error) {

	clustersArns, err := api.ListClusters(&ecs.ListClustersInput{})
	if err != nil {
		return nil, err
	}

	ret = make([]*ecs.Cluster, 0, len(clustersArns))

	bulks, err := api.SplitToBulks(clustersArns, 100)
	if err != nil {
		return nil, err
	}

	for _, bulk := range bulks {
		output, err := api.svc.DescribeClusters(&ecs.DescribeClustersInput{Clusters: bulk})
		if err != nil {
			return nil, err
		}

		ret = append(ret, output.Clusters...)
	}
	return ret, nil
}

func (api *ECSAPI) SplitToBulks(src []*string, size int) (ret [][]*string, err error) {
	ret = [][]*string{}
	var start int
	var end int
	for i := range len(src) / size {
		start = i * size
		end = min((start + size), len(src))
		if start == end {
			break
		}
		bulk := make([]*string, 0, size)
		bulk = append(bulk, src[start:end]...)
		ret = append(ret, bulk)
	}

	if end < len(src) {
		bulk := make([]*string, 0, len(src)-end)
		bulk = append(bulk, src[end:]...)
		ret = append(ret, bulk)
	}

	return ret, nil
}
