package aws_api

import (
	"encoding/json"
	"os"
	"testing"
)

func LoadGetLambdaLogsTestConfig() (*CleanerConfig, error) {
	configFilePath := "/opt/aws_hapi/cleaner/input/CleanerConfig.json"
	var config *CleanerConfig
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func TestCleanLogGroupExpired(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		config, err := LoadGetLambdaLogsTestConfig()
		if err != nil {
			t.Errorf("%v", err)
		}

		awsCleaner, err := CleanerNew(config)

		if err != nil {
			t.Errorf("%v", err)
		}

		err = awsCleaner.CleanLogGroupsExpired()
		if err != nil {
			t.Errorf("%v", err)
		}

	})
}
