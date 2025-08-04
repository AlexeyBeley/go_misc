package aws_api

import (
	"encoding/json"
	"os"
	"testing"

	clients "github.com/AlexeyBeley/go_misc/aws_api/clients"
)

type GetLambdaLogsTestsData struct {
	Region string `json:"Region"`
	Lambda string `json:"Lambda"`
}

func LoadGetLambdaLogsTestConfig() (*GetLambdaLogsTestsData, error) {
	configFilePath := "/opt/aws_hapi/log_analyzer/input/GetLambdaLogsTestsData.json"
	var config *GetLambdaLogsTestsData
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

func TestGetLambdaLogs(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		config, err := LoadGetLambdaLogsTestConfig()
		if err != nil {
			t.Errorf("%v", err)
		}

		os.Args = []string{"program_name", "--LambdaName", config.Lambda, "--Region", config.Region, "--PointInTime", "2025-07-24T21:11:47.147Z"}
		awsLogAnalizer, err := AWSLogAnalizerNew()

		if err != nil {
			t.Errorf("%v", err)
		}
		actionManager, err := awsLogAnalizer.GenerateActionManager()

		if err != nil {
			t.Errorf("%v", err)
		}

		err = actionManager.RunAction(clients.StrPtr("GetLamdaLogs"))

		if err != nil {
			t.Errorf("%v", err)
		}

	})
}

func TestGetAllLambdasLogs(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		config, err := LoadGetLambdaLogsTestConfig()
		if err != nil {
			t.Errorf("%v", err)
		}

		os.Args = []string{"program_name", "--LambdaName", config.Lambda, "--Region", config.Region}
		awsLogAnalizer, err := AWSLogAnalizerNew()

		if err != nil {
			t.Errorf("%v", err)
		}
		actionManager, err := awsLogAnalizer.GenerateActionManager()

		if err != nil {
			t.Errorf("%v", err)
		}

		err = actionManager.RunAction(clients.StrPtr("GetAllLamdasLogs"))

		if err != nil {
			t.Errorf("%v", err)
		}

	})
}
