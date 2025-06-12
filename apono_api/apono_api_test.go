package apono_api

import (
	"encoding/json"
	"log"
	"os"
	"testing"

	json_api "github.com/AlexeyBeley/go_common/json_api"
)

type testData struct {
	APIToken      string   `json:"APIToken"`
	UserID        string   `json:"UserID"`
	ResourceIdPermissionPairs   [][]string `json:"ResourceIdPermissionPairs"`
	IntegrationID string   `json:"IntegrationID"`
	Justification string   `json:"Justification"`
	AwsAppsURL    string   `json:"AwsAppsURL"`
	AwsAccountURL string   `json:"AwsAccountURL"`
}

func TestAponoAPINew(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		jsonData, err := os.ReadFile("/opt/apono/request_access_config.json")
		if err != nil {
			panic(err)
		}

		config := new(testData)
		err = json.Unmarshal([]byte(jsonData), config)
		if err != nil {
			panic(err)
		}

		aponoAPI, err := AponoAPINew(config.APIToken)

		if err != nil || aponoAPI == nil {
			t.Errorf("%v", err)
		}
	})
}

func TestListAccessRequests(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		jsonData, err := os.ReadFile("/opt/apono/request_access_config.json")
		if err != nil {
			panic(err)
		}

		config := new(testData)
		err = json.Unmarshal([]byte(jsonData), config)
		if err != nil {
			panic(err)
		}

		aponoAPI, err := AponoAPINew(config.APIToken)

		if err != nil || aponoAPI == nil {
			t.Errorf("%v", err)
		}
		ret, err := aponoAPI.ListAccessRequests(0)
		if err != nil {
			t.Errorf("%v", err)
		}
		filePath := "/tmp/apono_access_requests.json"
		json_api.WirteToFile(ret, &filePath)
		_ = ret
	})
}

func TestGetAccessRequestEntitlements(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		jsonData, err := os.ReadFile("/opt/apono/request_access_config.json")
		if err != nil {
			panic(err)
		}

		config := new(testData)
		err = json.Unmarshal([]byte(jsonData), config)
		if err != nil {
			panic(err)
		}

		aponoAPI, err := AponoAPINew(config.APIToken)

		if err != nil || aponoAPI == nil {
			t.Errorf("%v", err)
		}
		ret, err := aponoAPI.GetAccessRequestEntitlements(0, "AR-Something")
		if err != nil {
			t.Errorf("%v", err)
		}
		log.Printf("Resource ID: %s: Permission: %s", ret[0].Resource.ID, ret[0].Permission.Name)

	})
}
