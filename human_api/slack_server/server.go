package slack_server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
	"unicode"

	"github.com/AlexeyBeley/go_misc/azure_devops_api"
	config_pol "github.com/AlexeyBeley/go_misc/configuration_policy"
	human_api "github.com/AlexeyBeley/go_misc/human_api"
	human_api_types "github.com/AlexeyBeley/go_misc/human_api_types/v1"
)

type Configuration struct {
	MainDirPath                         *string
	SlackBlockKitDirPath                *string
	VerificationToken                   *string
	AzureDevopsAPIConfigurationFilePath *string
	HumanAPIConfigurationFilePath       *string
}

type SlackServer struct {
	Configuration *Configuration
	humanAPI      *human_api.HumanAPI
}

func SlackServerNew(options ...config_pol.Option) *SlackServer {

	slackServer := &SlackServer{}

	configuration := &Configuration{}
	for _, option := range options {
		option(slackServer, configuration)
	}

	if configuration.MainDirPath == nil {
		configuration.MainDirPath = new(string)
		*configuration.MainDirPath = "/opt/human_api/"
	}

	if configuration.SlackBlockKitDirPath == nil {
		configuration.SlackBlockKitDirPath = new(string)
		*configuration.SlackBlockKitDirPath = filepath.Join(*configuration.MainDirPath, "slack_server_static_files")
	}

	if configuration.VerificationToken == nil || *configuration.VerificationToken == "" {
		panic("The environemnt variabele value of SLACK_APP_TOKEN is an empty string, so it's not set.\n")
	}

	return slackServer
}

func (slackServer *SlackServer) SetConfiguration(ConfigAny any) error {
	Config, ok := ConfigAny.(*Configuration)
	if !ok {
		return fmt.Errorf("was not able to convert %v to slackServer Configuration", ConfigAny)
	}
	slackServer.Configuration = Config
	return nil
}

func (slackServer *SlackServer) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// Example of a struct to be returned as JSON
	fmt.Fprint(w, "OK")
}

func (slackServer *SlackServer) Start() error {
	// Register the handler functions for different paths
	http.HandleFunc("/hapi", slackServer.hapiMain)
	http.HandleFunc("/interactive", slackServer.hapiInteractive)
	http.HandleFunc("/health-check", slackServer.healthCheckHandler)

	// Start the server on port 8080
	log.Println("Starting server on port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
	return nil
}

func (slackServer *SlackServer) hapiInteractive(w http.ResponseWriter, r *http.Request) {
	// #todo: refactor this function to handle interactive replies.
	// it has to open goroutine and wait for result.
	// after 1 second send async reply 200 in case result is not ready
	//

	contentType := r.Header.Get("Content-Type")
	// Log the received data

	if !strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		http.Error(w, "Bad Request. Expected Content-Type 'application/x-www-form-urlencoded', received: "+contentType, http.StatusBadRequest)
		log.Println("Received POST request with no data.")
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request: Failed to parse form data", http.StatusBadRequest)
		log.Printf("Error parsing form: %v", err)
		return
	}

	var payload string
	for key, values := range r.Form {
		if key != "payload" {
			log.Printf("key %s is not 'payload'", key)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if len(values) != 1 {
			log.Printf("expected single value, recived %v", values)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		payload = values[0]

	}

	timestamp := time.Now().UTC().Format(time.RFC3339)
	log.Printf("Received hapi interactive command at %s (Content-Type: %s): %+v", timestamp, contentType, payload)

	w.WriteHeader(http.StatusAccepted)
	err := slackServer.handleInteractivePayload(payload)
	if err != nil {
		log.Printf("Recieved error hadnling handleInteractivePayload: %v", err)
	}

}

func (slackServer *SlackServer) handleInteractivePayload(payload string) error {
	request := new(map[string]any)
	json.Unmarshal([]byte(payload), request)

	if token, ok := (*request)["token"]; !ok || token != slackServer.Configuration.VerificationToken {
		return fmt.Errorf("Error handling request Slack App: %s", "token iether wrong or does not present")

	}

	responseUrlAny := (*request)["response_url"]
	responseUrl, ok := responseUrlAny.(string)
	if !ok {
		return fmt.Errorf("response_url is not a valid string %v", *request)
	}
	actionsAny, ok := (*request)["actions"]
	if !ok {
		return fmt.Errorf("can not find key 'actions' in %v", *request)
	}

	actionList, ok := actionsAny.([]any)

	if !ok {
		return fmt.Errorf("interface conversion: actionsAny is %T, not []map[string]any", actionsAny)
	}

	actions := []map[string]any{}
	for _, actionListItemAny := range actionList {
		actionListItem, ok := actionListItemAny.(map[string]any)
		if !ok {
			return fmt.Errorf("was not able to convert %v", actionListItemAny)
		}
		actions = append(actions, actionListItem)

	}

	actionId := ""
	currentUser := (*request)["user"]
	currentUserMap, ok := currentUser.(map[string]any)
	if !ok {
		return fmt.Errorf("currentUser %T, not map", currentUser)
	}

	currentUserID, ok := currentUserMap["id"].(string)
	if !ok {
		return fmt.Errorf("interface conversion: currentUserMap[id] is %T, not string", currentUserMap)
	}

	for _, action := range actions {
		actionIdAny, ok := action["action_id"]
		if !ok {
			return fmt.Errorf("can not find key 'action_id' in %v", action)
		}

		actionIdTmp, ok := actionIdAny.(string)

		if !ok {
			return fmt.Errorf("interface conversion: actionIdAny is %T, not string", actionIdAny)
		}

		if strings.Contains(actionIdTmp, "->") {
			if actionId != "" {
				return fmt.Errorf("action already initialized to '%s', trying to init new value '%s'", actionId, actionIdTmp)
			}
			actionId = actionIdTmp
		}

	}

	if actionId == "" {
		return fmt.Errorf("can not find action ID '%s'", actions)
	}

	var response map[string]any
	var err error

	if actionId == "main->wobj" {
		response, err = slackServer.LoadGenericMenu("slack_wobj.json")
	} else if actionId == "main->wobj->create" {
		response, err = slackServer.LoadGenericMenu("slack_wobj_create_new.json")
	} else if actionId == "main->wobj->create->submit" {
		response, err = slackServer.LoadGenericMenu("slack_wobj_create_submit.json")
	} else if actionId == "main->help" {
		response, err = slackServer.LoadGenericMenu("tmp.json", currentUserID)
	} else {
		return fmt.Errorf("unknown action ID %s", actionId)
	}

	if err != nil {
		return fmt.Errorf("error handling actionId %s", actionId)
	}
	//response["trigger_id"] = triggerID
	err = slackServer.sendResponseUrlMessage(responseUrl, response)
	if err != nil {
		return fmt.Errorf("handleInteractivePayload failed to send response %v to url %s, with error: %w ", response, responseUrl, err)
	}
	return nil
}

func (slackServer *SlackServer) hapiMain(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	// Log the received data

	if !strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		http.Error(w, "Bad Request. Expected Content-Type 'application/x-www-form-urlencoded', received: "+contentType, http.StatusBadRequest)
		log.Println("Received POST request with no data.")
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request: Failed to parse form data", http.StatusBadRequest)
		log.Printf("Error parsing form: %v", err)
		return
	}

	data := make(map[string]any)
	for key, values := range r.Form {
		data[key] = values[0] // Take the first value for each key
	}

	statusCode := http.StatusOK
	var responseAny map[string]any

	if token, ok := data["token"]; !ok || token != slackServer.Configuration.VerificationToken {
		responseAny = map[string]any{"Error handling request": "Slack App token iether wrong or does not present"}
		statusCode = http.StatusBadRequest
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)
	log.Printf("Received hapi command at %s (Content-Type: %s): %+v", timestamp, contentType, data)

	if len(data) == 0 {
		http.Error(w, "Bad request: Empty data", http.StatusBadRequest)
		log.Printf("Bad request: Empty data")
		return
	}

	text := data["text"].(string)
	text = strings.TrimLeftFunc(text, unicode.IsSpace)

	var err error

	domain := strings.Split(text, " ")[0]
	if text != "" {
		log.Printf("Handling text '%s'", text)
	}
	switch domain {
	case "":
		responseAny, err = slackServer.LoadGenericMenu("slack_main.json")
	case "wobj":
		text = text[len("wobj"):]
		text = strings.TrimLeftFunc(text, unicode.IsSpace)
		responseAny, err = slackServer.wobjHandler(text)
	case "help":
		responseAny = map[string]any{"help": "Show this menu",
			"wobj init": "Init wobject sample for submitting"}
	default:
		err = fmt.Errorf("unknown request: %s", text)
		responseAny = map[string]any{"error": fmt.Sprintf("Unknown domain: %s", domain)}
	}

	if err != nil {
		responseAny = map[string]any{"Error handling request": fmt.Sprintf("%v", err)}
		statusCode = http.StatusBadRequest
	}

	log.Printf("Returning response to client statusCode: %d, responseAny: %v", statusCode, responseAny)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(responseAny); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Error encoding response", http.StatusBadRequest)
	}

}

func (slackServer *SlackServer) LoadGenericMenu(fileName string, replacements ...any) (response map[string]any, err error) {
	response, err = slackServer.loadJsonFile(fileName, replacements...)
	if err != nil {
		return nil, err

	}
	log.Printf("Successfully loaded Generic Menu %s", fileName)
	return response, nil
}

func (slackServer *SlackServer) wobjHandler(text string) (response map[string]any, err error) {
	command := strings.Split(text, " ")[0]
	log.Printf("Handling command '%s'", command)
	if command == "menu" {
		// todo: response, err = slackServer.LoadGenericMenu("slack_wobj.json")
		response, err = slackServer.LoadGenericMenu("slack_wobj.json")
	} else {
		err = fmt.Errorf("unknown command: %s", command)
	}

	return response, err
}

func (slackServer *SlackServer) loadJsonFile(fileName string, replacements ...any) (map[string]any, error) {
	fullPath := filepath.Join(*slackServer.Configuration.SlackBlockKitDirPath, fileName)
	jsonData, err := os.ReadFile(fullPath)
	if err != nil {
		log.Printf("error reading file %s: %v", fullPath, err)
		return nil, err
	}

	jsonDataString := fmt.Sprintf(string(jsonData), replacements...)

	var result map[string]any
	err = json.Unmarshal([]byte(jsonDataString), &result)
	if err != nil {
		log.Printf("error unmarshalling file %s: %v", fullPath, err)
		return nil, err
	}
	return result, nil
}

func (slackServer *SlackServer) sendResponseUrlMessage(responseUrl string, payload map[string]any) error {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON payload: %w", err)
	}

	req, err := http.NewRequest("POST", responseUrl, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to response_url: %w", err)
	}
	defer resp.Body.Close()

	bodyStr := []byte{}
	resp.Body.Read(bodyStr)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-OK status code %d from response_url: %s, body: %s",
			resp.StatusCode,
			responseUrl,
			string(bodyStr))
	}

	log.Printf("Sucessfully sent %v to %s, bytes: %s", payload, responseUrl, string(bodyStr))

	return nil
}

// BlockKitMessage is a struct representing a simplified Block Kit message
type BlockKitMessage struct {
	Blocks []map[string]any `json:"blocks"`
}

func createResponseUrlPayload(actionID string, userSelection string) *BlockKitMessage {
	headerBlock := map[string]any{
		"type": "header",
		"text": map[string]any{
			"type": "plain_text",
			"text": "Your selection has been submitted.",
		},
	}

	sectionBlock := map[string]any{
		"type": "section",
		"text": map[string]any{
			"type": "mrkdwn",
			"text": fmt.Sprintf("You selected `%s` from action `%s`.", userSelection, actionID),
		},
	}

	return &BlockKitMessage{
		Blocks: []map[string]any{headerBlock, sectionBlock},
	}
}

// Ticket represents the data for a submission modal.
type Ticket struct {
	Type        string
	Title       string
	Description string
}

func (slackServer *SlackServer) WobjectModalHandler(w http.ResponseWriter, r *http.Request) bool {
	// Create a new ticket with the default type set to "bug".
	log.Printf("WobjectModalHandler started %s", "Now")
	data := Ticket{
		Type: "bug",
	}

	tmpl, err := template.ParseFiles(filepath.Join(*slackServer.Configuration.SlackBlockKitDirPath, "templates", "wobject_create.html"))
	if err != nil {
		log.Printf("Error loading wobject_create: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	log.Printf("WobjectModalHandler created template %v", tmpl)

	err = tmpl.Execute(w, data)
	if err != nil {
		log.Printf("Error loading wobject_create: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	log.Printf("WobjectModalHandler sent template %v", tmpl)
	return true
}

func (slackServer *SlackServer) humanAPIInit() error {
	ProjectManagerAPI, err := azure_devops_api.AzureDevopsAPINew(config_pol.WithConfigurationFile(slackServer.Configuration.AzureDevopsAPIConfigurationFilePath))
	if err != nil {
		return err
	}

	api, err := human_api.HumanAPINew(config_pol.WithConfigurationFile(slackServer.Configuration.HumanAPIConfigurationFilePath),
		human_api.WithProjectManagerAPI(ProjectManagerAPI))
	if err != nil {
		return err
	}
	slackServer.humanAPI = api
	return nil
}

func (slackServer *SlackServer) ProvisionWobject(WorkObject *human_api_types.Wobject) error {
	err := slackServer.humanAPIInit()
	if err != nil {
		return err
	}

	err = slackServer.humanAPI.ProvisionWobject(WorkObject)
	if err != nil {
		return err
	}
	return nil

}
