package aws_api

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type ECTestConfiguration struct {
	Region                       string   `json:"Region"`
	LogGroup                     string   `json:"LogGroup"`
	DescribeFlowLogsPagesSubnets []string `json:"DescribeFlowLogsPagesSubnets"`
}

func LoadEC2TestConfig(configFilePath string) (config ECTestConfiguration, err error) {
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

func loadEC2TestRealConfig() ECTestConfiguration {
	os.Getenv("CONFIG_PATH")
	conf_path := "/tmp/ec2.json"
	config, err := LoadEC2TestConfig(conf_path)
	if err != nil {
		log.Fatalf("%v", err)
	}
	return config
}

func TestDescribeNetworkInterfaces(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadEC2TestRealConfig()
		client := getEC2Client(&realConfig.Region)
		err := DescribeNetworkInterfaces(client, func(nInt any) error {
			fmt.Printf("nIntL %v ", nInt)
			return nil
		}, nil)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestRecordNetworkInterfaces(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadEC2TestRealConfig()
		networkInterfaces := make(map[string]ec2.NetworkInterface)
		err := RecordNetworkInterfaces(&realConfig.Region, &networkInterfaces)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestDescribeFlowLogsPagesPrintCallback(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {

		realConfig := loadEC2TestRealConfig()

		var subnetValues []*string
		subnetValues = make([]*string, 0)
		for _, subnet := range realConfig.DescribeFlowLogsPagesSubnets {
			subnetValues = append(subnetValues, &subnet)
		}

		client := getEC2Client(&realConfig.Region)
		Filters := []*ec2.Filter{{
			Name:   aws.String("resource-id"), // Filter by resource ID
			Values: subnetValues,
		}}

		err := DescribeFlowLogsPages(client, Filters, func(flowLog any) error {
			Value, ok := flowLog.(*ec2.FlowLog)
			if !ok {
				t.Errorf("%v", flowLog)
			}

			fmt.Println("flowLog: " + Value.String())
			return nil
		})
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestDescribeFlowLogsPagesAggregatorCallback(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {

		realConfig := loadEC2TestRealConfig()

		var subnetValues []*string
		subnetValues = make([]*string, 0)
		for _, subnet := range realConfig.DescribeFlowLogsPagesSubnets {
			subnetValues = append(subnetValues, &subnet)
		}

		client := getEC2Client(&realConfig.Region)
		Filters := []*ec2.Filter{{
			Name:   aws.String("resource-id"), // Filter by resource ID
			Values: subnetValues,
		}}
		objects := make([]any, 0)
		err := DescribeFlowLogsPages(client, Filters, AggregatorInitializer(&objects))
		if err != nil {
			t.Errorf("%v", err)
		}
		if len(objects) != len(subnetValues) {
			t.Errorf("Expected to fetch %v FlowLogs but fetched %v", len(subnetValues), len(objects))

		}
	})
}
