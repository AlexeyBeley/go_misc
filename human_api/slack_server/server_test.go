package slack_server

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	common_utils "github.com/AlexeyBeley/go_misc/common_utils"
	config_pol "github.com/AlexeyBeley/go_misc/configuration_policy"
	human_api_types "github.com/AlexeyBeley/go_misc/human_api_types/v1"
)

var GlobalHumanAPIConfigurationFilePath = "/opt/human_api/human_api_config.json"
var GlobalSlackServerConfigurationFilePath = "/opt/human_api/slack_server_configuration.json"

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
func TestHapiMainWobjInit(t *testing.T) {
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
			"token":                 "TESTTOKEN",
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

		dir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed with error: %v", err)
		}

		slackServer := SlackServerNew(config_pol.WithConfigurationFile(&GlobalSlackServerConfigurationFilePath))
		slackServer.Configuration.MainDirPath = common_utils.StrPTR(filepath.Join(dir, "../../cmd/human_api"))
		*slackServer.Configuration.MainDirPath = filepath.Join(*slackServer.Configuration.MainDirPath, "slack_server_static_files")
		*slackServer.Configuration.SlackBlockKitDirPath = *slackServer.Configuration.MainDirPath
		slackServer.Configuration.VerificationToken = common_utils.StrPTR("TESTTOKEN")

		slackServer.hapiMain(w, r)
		SCExpected := http.StatusOK

		if *w.DataCollector.StatusCode != SCExpected {
			t.Fatalf("Status code expected %d, received %d with data: %v", SCExpected, *w.DataCollector.StatusCode, string(*w.DataCollector.Response))
		}
	})
}

func TestHapiMainMenu(t *testing.T) {
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
			"text":                  "",
			"token":                 "TESTTOKEN",
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

		dir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed with error: %v", err)
		}

		slackServer := SlackServerNew(config_pol.WithConfigurationFile(&GlobalSlackServerConfigurationFilePath))
		slackServer.Configuration.MainDirPath = common_utils.StrPTR(filepath.Join(dir, "../../cmd/human_api"))
		*slackServer.Configuration.MainDirPath = filepath.Join(*slackServer.Configuration.MainDirPath, "slack_server_static_files")
		*slackServer.Configuration.SlackBlockKitDirPath = *slackServer.Configuration.MainDirPath
		slackServer.Configuration.VerificationToken = common_utils.StrPTR("TESTTOKEN")

		slackServer.hapiMain(w, r)
		SCExpected := http.StatusOK

		if *w.DataCollector.StatusCode != SCExpected {
			t.Fatalf("Status code expected %d, received %d with data: %v", SCExpected, *w.DataCollector.StatusCode, string(*w.DataCollector.Response))
		}
	})
}

func TestLoadGenericMenuCreateWobjectNew(t *testing.T) {
	t.Run("Init test", func(t *testing.T) {

		dir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed with error: %v", err)
		}

		slackServer := SlackServerNew(config_pol.WithConfigurationFile(&GlobalSlackServerConfigurationFilePath))
		slackServer.Configuration.MainDirPath = common_utils.StrPTR(filepath.Join(dir, "../../cmd/human_api"))
		*slackServer.Configuration.MainDirPath = filepath.Join(*slackServer.Configuration.MainDirPath, "slack_server_static_files")
		*slackServer.Configuration.SlackBlockKitDirPath = *slackServer.Configuration.MainDirPath
		slackServer.Configuration.VerificationToken = common_utils.StrPTR("TESTTOKEN")

		fileName := "slack_wobj_create_new.json"
		response, err := slackServer.LoadGenericMenu(fileName, &map[string]string{"STRING_REPLACEMENT_INITIAL_USER": "YourSlackUserID"})

		if err != nil {
			t.Fatalf("Recieved error: %V ", err)
		}
		if len(response.Blocks) == 0 {
			t.Fatalf("Expected > 0 blocks in  %s", fileName)
		}

	})
}

func TestSendResponseUrlMessageRaw(t *testing.T) {
	t.Run("Init test", func(t *testing.T) {

		url := ""

		response := slackBlockKitResponse{}
		slackServer := &SlackServer{}
		err := slackServer.sendResponseUrlMessage(url, response)
		if err != nil {
			t.Fatalf("Failed with error: %v", err)
		}

	})
}

func TestHandleInteractivePayloadMain(t *testing.T) {
	t.Run("Init test", func(t *testing.T) {

		dir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed with error: %v", err)
		}

		testPayloadFilePath := filepath.Join(dir, "payloads", "main_wobj.json")
		paylod, err := os.ReadFile(testPayloadFilePath)

		if err != nil {
			t.Fatalf("Failed with error: %v", err)
		}

		slackServer := SlackServerNew(config_pol.WithConfigurationFile(&GlobalSlackServerConfigurationFilePath))

		//common_utils.StrPTR(filepath.Join(dir, "../../cmd/human_api")) )
		err = slackServer.handleInteractivePayload(string(paylod))

		if err != nil {
			t.Fatalf("Failed with error: %v", err)
		}
	})
}

func TestHandleInteractivePayloadWobjCreateSubmit(t *testing.T) {
	t.Run("Init test", func(t *testing.T) {

		dir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed with error: %v", err)
		}

		testPayloadFilePath := filepath.Join(dir, "payloads", "wobject_create_submit.json")
		paylod, err := os.ReadFile(testPayloadFilePath)

		if err != nil {
			t.Fatalf("Failed with error: %v", err)
		}

		slackServer := SlackServerNew(config_pol.WithConfigurationFile(&GlobalSlackServerConfigurationFilePath))
		slackServer.Configuration.MainDirPath = common_utils.StrPTR(filepath.Join(dir, "../../cmd/human_api"))
		*slackServer.Configuration.MainDirPath = filepath.Join(*slackServer.Configuration.MainDirPath, "slack_server_static_files")
		*slackServer.Configuration.SlackBlockKitDirPath = *slackServer.Configuration.MainDirPath
		slackServer.Configuration.VerificationToken = common_utils.StrPTR("TESTTOKEN")

		//common_utils.StrPTR(filepath.Join(dir, "../../cmd/human_api")) )
		err = slackServer.handleInteractivePayload(string(paylod))

		if err != nil {
			t.Fatalf("Failed with error: %v", err)
		}

	})
}

func TestHandleInteractivePayloadWobjCreateSubmitnon_default(t *testing.T) {
	t.Run("Init test", func(t *testing.T) {

		dir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed with error: %v", err)
		}

		testPayloadFilePath := filepath.Join(dir, "payloads", "wobject_create_submit_non_default.json")
		paylod, err := os.ReadFile(testPayloadFilePath)

		if err != nil {
			t.Fatalf("Failed with error: %v", err)
		}

		slackServer := SlackServerNew(config_pol.WithConfigurationFile(&GlobalSlackServerConfigurationFilePath))
		slackServer.Configuration.MainDirPath = common_utils.StrPTR(filepath.Join(dir, "../../cmd/human_api"))
		*slackServer.Configuration.MainDirPath = filepath.Join(*slackServer.Configuration.MainDirPath, "slack_server_static_files")
		*slackServer.Configuration.SlackBlockKitDirPath = *slackServer.Configuration.MainDirPath
		slackServer.Configuration.VerificationToken = common_utils.StrPTR("TESTTOKEN")

		//common_utils.StrPTR(filepath.Join(dir, "../../cmd/human_api")) )
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

		slackServer := SlackServerNew(config_pol.WithConfigurationFile(&GlobalSlackServerConfigurationFilePath))
		slackServer.Configuration.MainDirPath = common_utils.StrPTR(filepath.Join(dir, "../../cmd/human_api"))
		*slackServer.Configuration.MainDirPath = filepath.Join(*slackServer.Configuration.MainDirPath, "slack_server_static_files")
		*slackServer.Configuration.SlackBlockKitDirPath = *slackServer.Configuration.MainDirPath

		fileName := "help.json"
		//fileName = "new_tmp.json"
		response, err := slackServer.LoadGenericMenu(fileName, &map[string]string{"STRING_REPLACEMENT_INITIAL_USER": "YourSlackUserID"})
		if err != nil {
			t.Fatalf("Failed with error: %v", err)
		}
		err = slackServer.sendResponseUrlMessage(responseUrl, response)
		if err != nil {
			t.Fatalf("Failed with error: %v", err)
		}
	})
}

func TestProvisionBugWobject(t *testing.T) {
	t.Run("Init test", func(t *testing.T) {

		dir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed with error: %v", err)
		}

		slackServer := SlackServerNew(config_pol.WithConfigurationFile(&GlobalSlackServerConfigurationFilePath))
		slackServer.Configuration.MainDirPath = common_utils.StrPTR(filepath.Join(dir, "../../cmd/human_api"))
		*slackServer.Configuration.MainDirPath = filepath.Join(*slackServer.Configuration.MainDirPath, "slack_server_static_files")

		workObject := &human_api_types.Wobject{}

		workObject.Id = ""
		workObject.ParentID = ""
		workObject.Priority = 1
		workObject.Title = "test hry"
		workObject.Description = "test hry desc"
		workObject.LeftTime = 4
		workObject.InvestedTime = 1
		workObject.WorkerID = "al"
		workObject.ChildrenIDs = &[]string{}
		workObject.Sprint = ""
		workObject.Status = "Closed"
		workObject.Type = "Bug"

		err = slackServer.ProvisionWobject(workObject)
		if err != nil {
			t.Fatalf("Failed with error: %v", err)
		}

		if err != nil {
			t.Fatalf("Failed with error: %v", err)
		}
	})
}

func TestProvisionTaskWobject(t *testing.T) {
	t.Run("Init test", func(t *testing.T) {

		dir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed with error: %v", err)
		}

		slackServer := SlackServerNew(config_pol.WithConfigurationFile(&GlobalSlackServerConfigurationFilePath))
		slackServer.Configuration.MainDirPath = common_utils.StrPTR(filepath.Join(dir, "../../cmd/human_api"))
		*slackServer.Configuration.MainDirPath = filepath.Join(*slackServer.Configuration.MainDirPath, "slack_server_static_files")

		workObject := &human_api_types.Wobject{}

		workObject.Id = ""
		workObject.ParentID = ""
		workObject.Priority = 1
		workObject.Title = "test hry"
		workObject.Description = "test hry desc"
		workObject.LeftTime = 4
		workObject.InvestedTime = 1
		workObject.WorkerID = "hry"
		workObject.ChildrenIDs = &[]string{}
		workObject.Sprint = ""
		workObject.Status = "Closed"
		workObject.Type = "Task"

		err = slackServer.ProvisionWobject(workObject)
		if err != nil {
			t.Fatalf("Failed with error: %v", err)
		}

		if err != nil {
			t.Fatalf("Failed with error: %v", err)
		}
	})
}
