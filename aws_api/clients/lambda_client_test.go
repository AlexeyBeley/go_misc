package aws_api

import (
	"log"
	"testing"
)

func loadLambdaTestConfig() Configuration {
	conf_path := "/opt/aws_api_go/test_lambda.json"
	config, err := LoadConfig(conf_path)
	if err != nil {
		log.Fatalf("%v", err)
	}
	return config
}

func TestGetFunctions(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadLambdaTestConfig()
		api := LambdaAPINew(&realConfig.Region, nil)

		ret, err := api.GetFunctions(nil)
		if err != nil {
			t.Errorf("%v", err)
		}
		if len(ret) == 0 {
			t.Errorf("len(ret)  %d", len(ret))
		}
	})
}
