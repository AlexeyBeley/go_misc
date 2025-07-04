package aws_api

import (
	"strings"

	clients "github.com/AlexeyBeley/go_misc/aws_api/clients"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/s3"
)

type ModifyTagsConfig struct {
	Region  string
	AddTags map[string]string
}

func (config *ModifyTagsConfig) InitFromM(source any) error {
	mapValues, sucess := source.(map[string]any)
	if !sucess {
		panic(source)
	}

	region, sucess := mapValues["Region"].(string)
	if !sucess {
		panic(source)
	}

	(*config).Region = region

	addTagsInterface, sucess := mapValues["AddTags"].(map[string]any)
	if !sucess {
		panic(source)
	}

	(*config).AddTags = map[string]string{}

	for key, value := range addTagsInterface {
		valueString, sucess := value.(string)
		if !sucess {
			panic(source)
		}
		(*config).AddTags[key] = valueString
	}

	(*config).Region = region
	// = addTags
	return nil
}

func AddTagsNetworkInterfaces(config ModifyTagsConfig) error {
	client := clients.GetEC2Client(&config.Region)
	api := clients.EC2APINew(&config.Region, nil)
	objects := make([]any, 0)
	err := clients.DescribeNetworkInterfaces(client, clients.AggregatorInitializer(&objects), nil)
	if err != nil {
		return err
	}

	for _, anyObject := range objects {
		nInt, ok := anyObject.(*ec2.NetworkInterface)
		if !ok {
			panic(anyObject)
		}

		createTagsOutput, err := api.CreateTags(nInt.TagSet, config.AddTags, nInt.NetworkInterfaceId, false)
		if err != nil {
			ret := err.Error()
			if strings.Contains(ret, "does not exist") {
				continue
			}
			return err
		}
		lg.InfoF("%s", createTagsOutput)

	}
	return nil
}

func AddTagsNatGateways(config ModifyTagsConfig) error {
	client := clients.GetEC2Client(&config.Region)
	api := clients.EC2APINew(&config.Region, nil)
	objects := make([]any, 0)
	err := clients.DescribeNatGateways(client, clients.AggregatorInitializer(&objects), nil)
	if err != nil {
		return err
	}

	for _, anyObject := range objects {
		obj, ok := anyObject.(*ec2.NatGateway)
		if !ok {
			panic(anyObject)
		}

		createTagsOutput, err := api.CreateTags(obj.Tags, config.AddTags, obj.NatGatewayId, false)
		if err != nil {
			ret := err.Error()
			lg.InfoF("%s", ret)
			return err
		}
		lg.InfoF("Create tags response: %s", createTagsOutput)

	}
	return nil
}

func AddTagsInstances(config ModifyTagsConfig) error {
	client := clients.GetEC2Client(&config.Region)
	api := clients.EC2APINew(&config.Region, nil)
	objects := make([]any, 0)
	err := clients.DescribeInstances(client, clients.AggregatorInitializer(&objects), nil)
	if err != nil {
		return err
	}

	for _, anyObject := range objects {
		obj, ok := anyObject.(*ec2.Instance)
		if !ok {
			panic(anyObject)
		}

		createTagsOutput, err := api.CreateTags(obj.Tags, config.AddTags, obj.InstanceId, false)
		if err != nil {
			ret := err.Error()
			lg.InfoF("%s", ret)
			return err
		}
		lg.InfoF("Create tags response: %s", createTagsOutput)

	}
	return nil
}

func AddTagsElasticIps(config ModifyTagsConfig) error {
	api := clients.EC2APINew(&config.Region, nil)
	objects := make([]any, 0)
	err := api.DescribeAddresses(clients.AggregatorInitializer(&objects), nil)
	if err != nil {
		return err
	}

	for _, anyObject := range objects {
		obj, ok := anyObject.(*ec2.Address)
		if !ok {
			panic(anyObject)
		}

		createTagsOutput, err := api.CreateTags(obj.Tags, config.AddTags, obj.AllocationId, false)
		if err != nil {
			ret := err.Error()
			lg.InfoF("%s", ret)
			return err
		}
		lg.InfoF("Create tags response: %s", createTagsOutput)

	}
	return nil
}

func AddTagsVolumes(config ModifyTagsConfig) error {
	api := clients.EC2APINew(&config.Region, nil)
	objects := make([]any, 0)
	err := api.DescribeVolumes(clients.AggregatorInitializer(&objects), nil)
	if err != nil {
		return err
	}

	for _, anyObject := range objects {
		obj, ok := anyObject.(*ec2.Volume)
		if !ok {
			panic(anyObject)
		}

		createTagsOutput, err := api.CreateTags(obj.Tags, config.AddTags, obj.VolumeId, false)
		if err != nil {
			ret := err.Error()
			lg.InfoF("%s", ret)
			return err
		}
		lg.InfoF("Create tags response: %s", createTagsOutput)

	}
	return nil
}

func AddTagsLaunchTemplates(config ModifyTagsConfig) error {
	api := clients.EC2APINew(&config.Region, nil)
	objects := make([]any, 0)
	err := api.DescribeLaunchTemplates(clients.AggregatorInitializer(&objects), nil)
	if err != nil {
		return err
	}

	for _, anyObject := range objects {
		obj, ok := anyObject.(*ec2.LaunchTemplate)
		if !ok {
			panic(anyObject)
		}

		createTagsOutput, err := api.CreateTags(obj.Tags, config.AddTags, obj.LaunchTemplateId, false)
		if err != nil {
			ret := err.Error()
			lg.InfoF("%s", ret)
			return err
		}
		lg.InfoF("Create tags response: %s", createTagsOutput)

	}
	return nil
}

func AddTagsImages(config ModifyTagsConfig) error {
	api := clients.EC2APINew(&config.Region, nil)
	objects := make([]any, 0)
	self := "self"
	input := ec2.DescribeImagesInput{Owners: []*string{&self}}
	err := api.DescribeImages(clients.AggregatorInitializer(&objects), &input)
	if err != nil {
		return err
	}

	for _, anyObject := range objects {
		obj, ok := anyObject.(*ec2.Image)
		if !ok {
			panic(anyObject)
		}

		createTagsOutput, err := api.CreateTags(obj.Tags, config.AddTags, obj.ImageId, false)
		if err != nil {
			ret := err.Error()
			lg.InfoF("%s", ret)
			return err
		}
		lg.InfoF("Create tags response: %s", createTagsOutput)

	}
	return nil
}

func AddTagsSnapshots(config ModifyTagsConfig) error {
	api := clients.EC2APINew(&config.Region, nil)
	objects := make([]any, 0)
	self := "self"
	input := ec2.DescribeSnapshotsInput{OwnerIds: []*string{&self}}
	err := api.DescribeSnapshots(clients.AggregatorInitializer(&objects), &input)
	if err != nil {
		return err
	}

	for _, anyObject := range objects {
		obj, ok := anyObject.(*ec2.Snapshot)
		if !ok {
			panic(anyObject)
		}

		createTagsOutput, err := api.CreateTags(obj.Tags, config.AddTags, obj.SnapshotId, false)
		if err != nil {
			ret := err.Error()
			lg.InfoF("%s", ret)
			return err
		}
		lg.InfoF("Create tags response: %s", createTagsOutput)

	}
	return nil
}

func AddTagsKeyPairs(config ModifyTagsConfig) error {
	api := clients.EC2APINew(&config.Region, nil)
	objects := make([]any, 0)
	err := api.DescribeKeyPairs(clients.AggregatorInitializer(&objects), nil)
	if err != nil {
		return err
	}

	for _, anyObject := range objects {
		obj, ok := anyObject.(*ec2.KeyPairInfo)
		if !ok {
			panic(anyObject)
		}

		createTagsOutput, err := api.CreateTags(obj.Tags, config.AddTags, obj.KeyPairId, false)
		if err != nil {
			ret := err.Error()
			lg.InfoF("%s", ret)
			return err
		}
		lg.InfoF("Create tags response: %s", createTagsOutput)

	}
	return nil
}

func AddTagsSecurityGroups(config ModifyTagsConfig) error {
	api := clients.EC2APINew(&config.Region, nil)
	objects := make([]any, 0)
	err := api.DescribeSecurityGroups(clients.AggregatorInitializer(&objects), nil)
	if err != nil {
		return err
	}

	for _, anyObject := range objects {
		obj, ok := anyObject.(*ec2.SecurityGroup)
		if !ok {
			panic(anyObject)
		}

		createTagsOutput, err := api.CreateTags(obj.Tags, config.AddTags, obj.GroupId, false)
		if err != nil {
			ret := err.Error()
			lg.InfoF("%s", ret)
			return err
		}
		lg.InfoF("Create tags response: %s", createTagsOutput)

	}
	return nil
}

func AddTagsLoadBalancers(config ModifyTagsConfig) error {
	api := clients.ELBV2APINew(&config.Region, nil)
	objects := make([]any, 0)
	err := api.DescribeLoadBalancers(clients.AggregatorInitializer(&objects), nil)
	if err != nil {
		return err
	}

	for _, anyObject := range objects {
		obj, ok := anyObject.(*elbv2.LoadBalancer)
		if !ok {
			panic(anyObject)
		}

		createTagsOutput, err := api.AddTags(config.AddTags, obj.LoadBalancerArn, false)
		if err != nil {
			ret := err.Error()
			lg.InfoF("%s", ret)
			return err
		}
		lg.InfoF("Create tags response: %s", createTagsOutput)

	}
	return nil
}

func AddTagsTargetGroups(config ModifyTagsConfig) error {
	api := clients.ELBV2APINew(&config.Region, nil)
	objects := make([]any, 0)
	err := api.DescribeTargetGroups(clients.AggregatorInitializer(&objects), nil)
	if err != nil {
		return err
	}

	for _, anyObject := range objects {
		obj, ok := anyObject.(*elbv2.TargetGroup)
		if !ok {
			panic(anyObject)
		}

		createTagsOutput, err := api.AddTags(config.AddTags, obj.TargetGroupArn, false)
		if err != nil {
			ret := err.Error()
			lg.InfoF("%s", ret)
			return err
		}
		lg.InfoF("Create tags response: %s", createTagsOutput)

	}
	return nil
}

func AddTagsAutoScalingGroups(config ModifyTagsConfig) error {
	api := clients.GetAutoscalingAPI(&config.Region, nil)
	objects := make([]any, 0)
	err := api.DescribeAutoScalingGroups(clients.AggregatorInitializer(&objects), nil)
	if err != nil {
		return err
	}

	for _, anyObject := range objects {
		obj, ok := anyObject.(*autoscaling.Group)
		if !ok {
			panic(anyObject)
		}

		//todo: refactor to transform obj.Tags (TagDescription) -> []*autoscaling.Tag{}

		objTags := []*autoscaling.Tag{}
		resourceType := "auto-scaling-group"
		err := api.CreateOrUpdateTags(objTags, config.AddTags, obj.AutoScalingGroupName, &resourceType, false)
		if err != nil {
			ret := err.Error()
			lg.InfoF("%s", ret)
			return err
		}

	}
	return nil
}

func AddTagsRDSClusters(config ModifyTagsConfig) error {
	api := clients.RDSAPINew(&config.Region, nil)
	objects := make([]any, 0)
	err := api.DescribeClusters(clients.AggregatorInitializer(&objects), nil)
	if err != nil {
		return err
	}

	for _, anyObject := range objects {
		obj, ok := anyObject.(*rds.DBCluster)
		if !ok {
			panic(anyObject)
		}

		createTagsOutput, err := api.CreateTags(obj.TagList, config.AddTags, obj.DBClusterArn, false)
		if err != nil {
			ret := err.Error()
			lg.InfoF("%s", ret)
			return err
		}
		lg.InfoF("Create tags response: %s", createTagsOutput)

	}
	return nil
}

func AddTagsRDSInstances(config ModifyTagsConfig) error {
	api := clients.RDSAPINew(&config.Region, nil)
	objects := make([]any, 0)
	err := api.DescribeInstances(clients.AggregatorInitializer(&objects), nil)
	if err != nil {
		return err
	}

	for _, anyObject := range objects {
		obj, ok := anyObject.(*rds.DBInstance)
		if !ok {
			panic(anyObject)
		}

		createTagsOutput, err := api.CreateTags(obj.TagList, config.AddTags, obj.DBInstanceArn, false)
		if err != nil {
			ret := err.Error()
			lg.InfoF("%s", ret)
			return err
		}
		lg.InfoF("Create tags response: %s", createTagsOutput)

	}
	return nil
}

func AddTagsS3Buckets(config ModifyTagsConfig) error {
	api := clients.S3APINew(&config.Region, nil)
	objects := make([]any, 0)
	err := api.ListBuckets(clients.AggregatorInitializer(&objects), nil)
	if err != nil {
		return err
	}

	for _, anyObject := range objects {
		obj, ok := anyObject.(*s3.Bucket)
		if !ok {
			panic(anyObject)
		}

		createTagsOutput, err := api.AddTags(config.AddTags, obj, false)
		if err != nil {
			ret := err.Error()
			lg.InfoF("%s", ret)
			return err
		}
		lg.InfoF("Create tags response: %s", createTagsOutput)

	}
	return nil
}

func AnyToStrings(src []any) (dst []*string, err error) {
	for _, srcObj := range src {
		dstObj := srcObj.(*string)
		dst = append(dst, dstObj)
	}
	return dst, err
}

func HandleTaskDefinition(anyObject any) error {
	obj, ok := anyObject.(*ecs.TaskDefinition)
	if !ok {
		panic(anyObject)
	}

	lg.InfoF("Tod: %v", obj)
	return nil
}

func CheckTagsECSTaskDefinitions(config ModifyTagsConfig) error {
	api := clients.ECSAPINew(&config.Region, nil)
	objects := make([]any, 0)
	err := api.GetTaskDefinitionFamilies(clients.AggregatorInitializer(&objects), nil)
	if err != nil {
		return err
	}

	families, err := AnyToStrings(objects)
	if err != nil {
		return err
	}

	for _, familyName := range families {
		objects = objects[:0]
		api.GetTaskDefinitions(clients.AggregatorInitializer(&objects), &ecs.ListTaskDefinitionsInput{FamilyPrefix: familyName, MaxResults: int64Ptr(int64(1)), Sort: strPtr("DESC")})
		lg.InfoF("Fetched %d task definitions family: %s", len(objects), *familyName)
	}
	return nil
}

func int64Ptr(i int64) *int64 {
	return &i
}

func strPtr(src string) *string {
	return &src
}
