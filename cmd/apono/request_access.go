package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/AlexeyBeley/go_misc/logger"
	apono_api "github.com/AlexeyBeley/go_misc/apono_api"
)

var lg = &(logger.Logger{})

type testData struct {
	APIToken                  string     `json:"APIToken"`
	UserID                    string     `json:"UserID"`
	ResourceIdPermissionPairs [][]string `json:"ResourceIdPermissionPairs"`
	IntegrationID             string     `json:"IntegrationID"`
	Justification             string     `json:"Justification"`
	AwsAppsURL                string     `json:"AwsAppsURL"`
	AwsAccountURL             string     `json:"AwsAccountURL"`
}

func main() {

	jsonData, err := os.ReadFile("/opt/apono/request_access_config.json")
	if err != nil {
		panic(err)
	}

	config := new(testData)
	err = json.Unmarshal([]byte(jsonData), config)
	if err != nil {
		panic(err)
	}

	aponoAPI, err := apono_api.AponoAPINew(config.APIToken)
	aponoAPI.RequestAccess(config.UserID, config.ResourceIdPermissionPairs, config.IntegrationID, config.Justification)
	if err != nil {
		panic(err)
	}

	url := config.AwsAccountURL
	lg.InfoF("GOTO: %s", url)
	cmd := exec.Command("open", "-a", "Google Chrome", url)
	err = cmd.Start()
	if err != nil {
		panic(fmt.Errorf("failed to open URL in Chrome (%s): %w", runtime.GOOS, err))
	}

	lg.InfoF("AWS landing page: %s\n", config.AwsAppsURL)

}
