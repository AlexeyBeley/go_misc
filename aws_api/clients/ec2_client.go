package aws_api

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type EC2Configuration struct {
	Region   string `json:"Region"`
	LogGroup string `json:"LogGroup"`
}

func LoadEC2Config(configFilePath string) (config EC2Configuration, err error) {
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}
	return config, nil
}

type NetworkInterfaceCallback func(*ec2.NetworkInterface) error

func GetEC2Client(region *string) *ec2.EC2 {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            aws.Config{Region: region},
	}))
	return ec2.New(sess)
}

func DescribeNetworkInterfaces(svc *ec2.EC2, callback GenericCallback, describeNetworkInterfacesInput *ec2.DescribeNetworkInterfacesInput) error {
	var callbackErr error
	pageNum := 0
	err := svc.DescribeNetworkInterfacesPages(describeNetworkInterfacesInput, func(page *ec2.DescribeNetworkInterfacesOutput, notHasNextPage bool) bool {
		// stop when returns False
		pageNum++
		for _, nInt := range page.NetworkInterfaces {
			if callbackErr = callback(nInt); callbackErr != nil {
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

func DescribeNatGateways(svc *ec2.EC2, callback GenericCallback, describeInput *ec2.DescribeNatGatewaysInput) error {
	var callbackErr error
	pageNum := 0
	err := svc.DescribeNatGatewaysPages(describeInput, func(page *ec2.DescribeNatGatewaysOutput, notHasNextPage bool) bool {
		// stop when returns False
		pageNum++
		for _, nInt := range page.NatGateways {
			if callbackErr = callback(nInt); callbackErr != nil {
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

func DescribeInstances(svc *ec2.EC2, callback GenericCallback, describeInput *ec2.DescribeInstancesInput) error {
	var callbackErr error
	pageNum := 0
	err := svc.DescribeInstancesPages(describeInput, func(page *ec2.DescribeInstancesOutput, notHasNextPage bool) bool {
		// stop when returns False
		pageNum++
		for _, reservation := range page.Reservations {
			for _, instance := range reservation.Instances {
				if callbackErr = callback(instance); callbackErr != nil {
					return false
				}
			}
		}
		return !notHasNextPage
	})
	if callbackErr != nil {
		return callbackErr
	}

	return err
}

func cacheNetworkInterfacesGenerator(networkInterfaces *map[string]ec2.NetworkInterface) func(nInt any) error {
	dst_file_path := "/tmp/networkInterfaces.json"
	return func(anyInt any) error {
		nInt, ok := anyInt.(*ec2.NetworkInterface)
		if !ok {
			panic(anyInt)
		}

		_, exists := (*networkInterfaces)[*nInt.NetworkInterfaceId]
		if exists {
			return nil
		}

		fmt.Printf("Recording interface %s\n", *nInt.NetworkInterfaceId)
		(*networkInterfaces)[*nInt.NetworkInterfaceId] = *nInt

		jsonData, err := json.MarshalIndent(*networkInterfaces, "", "  ")
		if err != nil {
			return err
		}

		err = os.WriteFile(dst_file_path, jsonData, 0644)
		if err != nil {
			return err
		}
		return nil
	}

}

func RecordNetworkInterfaces(region *string, networkInterfaces *map[string]ec2.NetworkInterface) error {

	client := GetEC2Client(region)
	for {
		err := DescribeNetworkInterfaces(client, cacheNetworkInterfacesGenerator(networkInterfaces), nil)
		if err != nil {
			return err
		}
		log.Println("Going to sleep")
		time.Sleep(30 * time.Second)
	}
}

func DescribeFlowLogsPages(svc *ec2.EC2, Filter []*ec2.Filter, callback GenericCallback) error {
	var callbackErr error
	pageNum := 0
	err := svc.DescribeFlowLogsPages(&ec2.DescribeFlowLogsInput{Filter: Filter}, func(page *ec2.DescribeFlowLogsOutput, notHasNextPage bool) bool {
		// stop when returns False
		pageNum++
		for _, fLog := range page.FlowLogs {
			if callbackErr = callback(fLog); callbackErr != nil {
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

type EC2API struct {
	svc *ec2.EC2
}

func EC2APINew(region *string, profileName *string) *EC2API {
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
	svc := ec2.New(sess)
	ret := EC2API{svc: svc}
	return &ret
}

func (api *EC2API) ProvisionFlowLog(logGroupName, resourceType, trafficType *string, resourceIds []*string, roleArn *string) (*ec2.CreateFlowLogsOutput, error) {
	input := ec2.CreateFlowLogsInput{LogGroupName: logGroupName,
		ResourceIds:              resourceIds,
		ResourceType:             resourceType,
		TrafficType:              trafficType,
		DeliverLogsPermissionArn: roleArn}
	reponse, err := api.svc.CreateFlowLogs(&input)
	return reponse, err
}

func (api *EC2API) DescribeVpcEndpointsPages(callback GenericCallback, describeNetworkInterfacesInput *ec2.DescribeVpcEndpointsInput) error {
	var callbackErr error
	pageNum := 0
	err := api.svc.DescribeVpcEndpointsPages(describeNetworkInterfacesInput, func(page *ec2.DescribeVpcEndpointsOutput, notHasNextPage bool) bool {
		pageNum++
		for _, obj := range page.VpcEndpoints {
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

func (api *EC2API) CreateTags(existingTags []*ec2.Tag, AddTags map[string]string, resource *string, declarative bool) (*ec2.CreateTagsOutput, error) {
	Tags := []*ec2.Tag{}
	for key, value := range AddTags {
		found := false
		for _, existingTag := range existingTags {
			if *existingTag.Key == key && (*existingTag.Value == value || !declarative) {
				found = true
				break
			}
		}

		if !found {
			Tags = append(Tags, &ec2.Tag{Key: &key, Value: &value})
		}
	}
	if len(Tags) == 0 {
		return nil, nil
	}
	lg.InfoF("Adding tags: resource: %s, tags: %v, Current tags: %v", *resource, Tags, existingTags)
	createTagsOutput, err := api.svc.CreateTags(&ec2.CreateTagsInput{Resources: []*string{resource}, Tags: Tags})
	return createTagsOutput, err
}

func (api *EC2API) DescribeAddresses(callback GenericCallback, describeInput *ec2.DescribeAddressesInput) error {
	output, err := api.svc.DescribeAddresses(describeInput)
	if err != nil {
		return err
	}

	for _, obj := range output.Addresses {
		if callbackErr := callback(obj); callbackErr != nil {
			return callbackErr
		}
	}

	return nil
}

func (api *EC2API) DescribeVolumes(callback GenericCallback, Input *ec2.DescribeVolumesInput) error {
	var callbackErr error
	pageNum := 0
	err := api.svc.DescribeVolumesPages(Input, func(page *ec2.DescribeVolumesOutput, notHasNextPage bool) bool {
		pageNum++
		for _, obj := range page.Volumes {
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

func (api *EC2API) DescribeLaunchTemplates(callback GenericCallback, Input *ec2.DescribeLaunchTemplatesInput) error {
	var callbackErr error
	pageNum := 0
	err := api.svc.DescribeLaunchTemplatesPages(Input, func(page *ec2.DescribeLaunchTemplatesOutput, notHasNextPage bool) bool {
		pageNum++
		for _, obj := range page.LaunchTemplates {
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

func (api *EC2API) DescribeImages(callback GenericCallback, Input *ec2.DescribeImagesInput) error {
	var callbackErr error
	pageNum := 0
	err := api.svc.DescribeImagesPages(Input, func(page *ec2.DescribeImagesOutput, notHasNextPage bool) bool {
		pageNum++
		for _, obj := range page.Images {
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

func (api *EC2API) DescribeSnapshots(callback GenericCallback, Input *ec2.DescribeSnapshotsInput) error {
	var callbackErr error
	pageNum := 0
	err := api.svc.DescribeSnapshotsPages(Input, func(page *ec2.DescribeSnapshotsOutput, notHasNextPage bool) bool {
		pageNum++
		for _, obj := range page.Snapshots {
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

func (api *EC2API) DescribeKeyPairs(callback GenericCallback, Input *ec2.DescribeKeyPairsInput) error {
	var callbackErr error

	response, err := api.svc.DescribeKeyPairs(Input)
	if err != nil {
		return err
	}

	for _, obj := range response.KeyPairs {
		if callbackErr = callback(obj); callbackErr != nil {
			return callbackErr
		}
	}

	return nil
}

func (api *EC2API) DescribeSecurityGroups(callback GenericCallback, Input *ec2.DescribeSecurityGroupsInput) error {
	var callbackErr error
	pageNum := 0
	err := api.svc.DescribeSecurityGroupsPages(Input, func(page *ec2.DescribeSecurityGroupsOutput, notHasNextPage bool) bool {
		pageNum++
		for _, obj := range page.SecurityGroups {
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

func (api *EC2API) GetNetworkInterfaces(callback GenericCallback, describeNetworkInterfacesInput *ec2.DescribeNetworkInterfacesInput) error {
	var callbackErr error
	pageNum := 0
	err := api.svc.DescribeNetworkInterfacesPages(describeNetworkInterfacesInput, func(page *ec2.DescribeNetworkInterfacesOutput, notHasNextPage bool) bool {
		// stop when returns False
		pageNum++
		for _, nInt := range page.NetworkInterfaces {
			if callbackErr = callback(nInt); callbackErr != nil {
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
