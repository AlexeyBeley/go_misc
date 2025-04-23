package aws_api

import (
	"strings"

	clients "github.com/AlexeyBeley/go_misc/aws_api/clients"
	"github.com/aws/aws-sdk-go/service/ec2"
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

func AddTagNetworkInterfaces(config ModifyTagsConfig) error {
	client := clients.GetEC2Client(&config.Region)
	api := clients.GetEC2API(&config.Region, nil)
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
		lg.Infof("%s", createTagsOutput)

	}
	return nil
}

func AddTagNatGateways(config ModifyTagsConfig) error {
	client := clients.GetEC2Client(&config.Region)
	api := clients.GetEC2API(&config.Region, nil)
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
			lg.Infof("%s", ret)
			return err
		}
		lg.Infof("Create tags response: %s", createTagsOutput)

	}
	return nil
}

func AddTagInstances(config ModifyTagsConfig) error {
	client := clients.GetEC2Client(&config.Region)
	api := clients.GetEC2API(&config.Region, nil)
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
			lg.Infof("%s", ret)
			return err
		}
		lg.Infof("Create tags response: %s", createTagsOutput)

	}
	return nil
}

func AddTagElasticIps(config ModifyTagsConfig) error {
	api := clients.GetEC2API(&config.Region, nil)
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
			lg.Infof("%s", ret)
			return err
		}
		lg.Infof("Create tags response: %s", createTagsOutput)

	}
	return nil
}

/*
volumes
launch templates
amis
snapshots
load balancers
target groups
key pairs
auto-scaling groups
security groups
*/
