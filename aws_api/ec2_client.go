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

func getEC2Client(region *string) *ec2.EC2 {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            aws.Config{Region: region},
	}))
	return ec2.New(sess)
}

func DescribeNetworkInterfaces(svc *ec2.EC2, callback NetworkInterfaceCallback) error {
	var callbackErr error
	pageNum := 0
	err := svc.DescribeNetworkInterfacesPages(&ec2.DescribeNetworkInterfacesInput{}, func(page *ec2.DescribeNetworkInterfacesOutput, notHasNextPage bool) bool {
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

func cacheNetworkInterfacesGenerator(networkInterfaces *[]ec2.NetworkInterface) func(nInt *ec2.NetworkInterface) error {
	dst_file_path := "/tmp/networkInterfaces.json"
	return func(nInt *ec2.NetworkInterface) error {
		for _, existingnetworkInterface := range *networkInterfaces {
			if *existingnetworkInterface.NetworkInterfaceId == *nInt.NetworkInterfaceId {
				return nil
			}
		}
		fmt.Printf("Recording interface %s\n", *nInt.NetworkInterfaceId)
		*networkInterfaces = append(*networkInterfaces, *nInt)

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

func RecordNetworkInterfaces(region *string) error {
	networkInterfaces := []ec2.NetworkInterface{}
	client := getEC2Client(region)
	for {
		err := DescribeNetworkInterfaces(client, cacheNetworkInterfacesGenerator(&networkInterfaces))
		if err != nil {
			return err
		}
		log.Println("Going to sleep")
		time.Sleep(30 * time.Second)
	}
	return nil
}

//type DescribeFlowLogsCallback func(*ec2.FlowLog) error

type DescribeFlowLogsCallback func(any) error

func AggregatorInitializer(objects *[]any) func(any) error {
	return func(object any) error {
		*objects = append(*objects, object)
		return nil
	}
}

func DescribeFlowLogsPages(svc *ec2.EC2, Filter []*ec2.Filter, callback DescribeFlowLogsCallback) error {
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
