package slack_server

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	common_utils "github.com/AlexeyBeley/go_misc/common_utils"
)

var GlobalHumanAPIConfigurationFilePath = "/opt/human_api/human_api_config.json"
var GlobalAzureDevopsAPIConfigurationFilePath = "/opt/azure_devops_api/configuration.json"

type ResponseWriterMock struct {
	DataCollector *ResponseWriterDataCollector
}

func (rwm ResponseWriterMock) Header() http.Header {
	return http.Header{}
}

func (rwm ResponseWriterMock) Write(data []byte) (int, error) {
	*(*rwm.DataCollector).Response = data
	return len(data), nil
}

func (rwm ResponseWriterMock) WriteHeader(statusCode int) {
	*(*rwm.DataCollector).StatusCode = statusCode
}

type ResponseWriterDataCollector struct {
	StatusCode *int
	Response   *[]byte
}

type BodyMock struct {
	Data []byte
}

func (bodyMock BodyMock) Close() error {
	return nil
}

func (bodyMock BodyMock) Read(p []byte) (n int, err error) {
	p = append(p, bodyMock.Data...)
	return len(p), nil
}

func NewDataCollector() *ResponseWriterDataCollector {
	ret := ResponseWriterDataCollector{}
	ret.Response = new([]byte)
	ret.StatusCode = new(int)
	return &ret
}

// curl -X POST -d "key1=value1&key2=value2" http://127.0.0.1:8080/ticket
func TestHapiMain(t *testing.T) {
	t.Run("Init test", func(t *testing.T) {
		w := ResponseWriterMock{DataCollector: NewDataCollector()}

		baseMap := map[string]string{"api_app_id": "APIAPPID012",
			"channel_id":            "CHANID01234",
			"channel_name":          "directmessage",
			"command":               "/hapi",
			"is_enterprise_install": "false",
			"response_url":          "https://hooks.slack.com/commands/SOMETHINGONE/12345678910/SONETHINGTWO",
			"team_domain":           "horeydomain",
			"team_id":               "horeyteam",
			"text":                  "wobj init",
			"token":                 "SECRETTOKEN",
			"trigger_id":            "12345678910.12345678910.12345678910HOREY",
			"user_id":               "USERIDHOREY",
			"user_name":             "horeyname.horeyfamily"}

		request := url.Values{}
		for key, val := range baseMap {
			request[key] = []string{val}
		}

		r := &http.Request{
			URL:      &url.URL{Path: "bla.com"},
			PostForm: request,
			Header:   http.Header{"Content-Type": []string{"application/x-www-form-urlencoded"}}}
		slackServer := &SlackServer{}
		slackServer.hapiMain(w, r)
		SCExpected := http.StatusOK

		if *w.DataCollector.StatusCode != SCExpected {
			t.Fatalf("Status code expected %d, received %d with data: %v", SCExpected, *w.DataCollector.StatusCode, string(*w.DataCollector.Response))
		}
	})
}

func TestSendResponseUrlMessageRaw(t *testing.T) {
	t.Run("Init test", func(t *testing.T) {

		url := ""

		response := map[string]any{"1": "2"}
		slackServer := &SlackServer{}
		err := slackServer.sendResponseUrlMessage(url, response)
		if err != nil {
			t.Fatalf("Failed with error: %v", err)
		}

	})
}

func TestHandleInteractivePayload(t *testing.T) {
	t.Run("Init test", func(t *testing.T) {

		dir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed with error: %v", err)
		}

		testPayloadFilePath := filepath.Join(dir, "payloads", "main_wobj.json")
		paylod, err := os.ReadFile(testPayloadFilePath)

		slackServer := SlackServerNew(common_utils.StrPTR(filepath.Join(dir, "../../cmd/human_api")), common_utils.StrPTR("TESTTOKEN"))
		err = slackServer.handleInteractivePayload(string(paylod))

		if err != nil {
			t.Fatalf("Failed with error: %v", err)
		}

	})
}

func TestSendResponseUrlMessageFromFile(t *testing.T) {
	t.Run("Init test", func(t *testing.T) {
		responseUrl := ""
		dir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed with error: %v", err)
		}

		slackServer := SlackServerNew(common_utils.StrPTR(filepath.Join(dir, "../../cmd/human_api")), common_utils.StrPTR("TESTTOKEN"))

		response, err := slackServer.LoadGenericMenu("slack_wobj_create_new.json")
		if err != nil {
			t.Fatalf("Failed with error: %v", err)
		}
		err = slackServer.sendResponseUrlMessage(responseUrl, response)
		if err != nil {
			t.Fatalf("Failed with error: %v", err)
		}
	})
}
