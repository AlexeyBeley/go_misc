package aws_api

import (
	"log"
	"testing"
)

func loadDynamoDBTestConfig() Configuration {
	conf_path := "/opt/aws_api_go/test_dynamodb.json"
	config, err := LoadConfig(conf_path)
	if err != nil {
		log.Fatalf("%v", err)
	}
	return config
}

func TestGetTables(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadDynamoDBTestConfig()
		api := DynamoDBAPINew(&realConfig.Region, nil)

		ret, err := api.GetTables(nil)
		if err != nil {
			t.Errorf("%v", err)
		}
		if len(ret) == 0 {
			t.Errorf("len(ret)  %d", len(ret))
		}
	})
}
