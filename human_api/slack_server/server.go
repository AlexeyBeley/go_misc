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
	slack_api "github.com/AlexeyBeley/go_misc/slack_api"
)

type Configuration struct {
	MainDirPath                         *string
	SlackBlockKitDirPath                *string
	VerificationToken                   *string
	AzureDevopsAPIConfigurationFilePath *string
	HumanAPIConfigurationFilePath       *string
	SlackAPIConfigurationFilePath       *string
}

type SlackServer struct {
	Configuration *Configuration
	humanAPI      *human_api.HumanAPI
	slackAPI      *slack_api.SlackAPI
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
	request := InteractiveRequest{}
	err := json.Unmarshal([]byte(payload), &request)
	if err != nil {
		return err
	}

	if request.Token != *slackServer.Configuration.VerificationToken {
		log.Printf("error handling request Slack App: Received oken %s does not match expected: %s", request.Token, *slackServer.Configuration.VerificationToken)
		return fmt.Errorf("error handling request Slack App: %s", "Wrong token")
	}

	actionId := ""
	for _, action := range request.Actions {

		if strings.Contains(action.ActionID, "->") {
			if actionId != "" {
				return fmt.Errorf("action already initialized to '%s', trying to init new value '%s'", actionId, action.ActionID)
			}
			actionId = action.ActionID
		}

	}

	if actionId == "" {
		return fmt.Errorf("can not find action ID '%v'", request.Actions)
	}

	var response slackBlockKitResponse

	switch actionId {
	case "main->wobj":
		response, err = slackServer.LoadGenericMenu("slack_wobj.json", nil)
	case "main->wobj->create":
		response, err = slackServer.LoadGenericMenu("slack_wobj_create_new.json", &map[string]string{"STRING_REPLACEMENT_INITIAL_USER": request.User.ID})
	case "main->wobj->create->submit":
		response, err = slackServer.HandleProvisionWobjectRequest(request)
	case "main->help":
		response, err = slackServer.LoadGenericMenu("help.json", &map[string]string{"STRING_REPLACEMENT_INITIAL_USER": request.User.ID})
	default:
		return fmt.Errorf("unknown action ID %s", actionId)
	}

	if err != nil {
		return fmt.Errorf("error handling actionId %s", actionId)
	}

	err = slackServer.sendResponseUrlMessage(request.ResponseURL, response)
	if err != nil {
		return fmt.Errorf("handleInteractivePayload failed to send response %v to url %s, with error: %w ", response, request.ResponseURL, err)
	}
	return nil
}

func (slackServer *SlackServer) HandleProvisionWobjectRequest(request InteractiveRequest) (response slackBlockKitResponse, err error) {
	menu, err := slackServer.LoadIneractiveMenu("slack_wobj_create_new.json", &map[string]string{"STRING_REPLACEMENT_INITIAL_USER": request.User.ID})
	if err != nil {
		return response, err
	}

	mapValues := map[string]string{}
	for blockID, value := range request.State.Values {
		fmt.Printf("Checking block: %s\n", blockID)
		for actionKey, interactiveRequestStateValue := range value {
			switch interactiveRequestStateValue.Type {
			case "static_select":
				mapValues[actionKey] = interactiveRequestStateValue.SelectedOption.Value
			case "plain_text_input":
				mapValues[actionKey] = interactiveRequestStateValue.Value
			case "users_select":
				mapValues[actionKey] = interactiveRequestStateValue.SelectedUser
			default:
				return response, fmt.Errorf("Unknown StateValue.Type: %s", interactiveRequestStateValue.Type)
			}
		}
	}

	for _, defaultBlock := range menu.Blocks {
		if defaultBlock.Type != "input" {
			continue
		}

		_, ok := mapValues[defaultBlock.Element.ActionID]
		if !ok {
			switch defaultBlock.Element.Type {
			case "users_select":
				mapValues[defaultBlock.Element.ActionID] = defaultBlock.Element.InitialUser
			case "static_select":
				mapValues[defaultBlock.Element.ActionID] = defaultBlock.Element.InitialOption.Value
			case "plain_text_input":
				return response, fmt.Errorf("input Element type plain_text_input has to be filled by user, in received block %s", defaultBlock.BlockID)
			default:
				return response, fmt.Errorf("unknown Element type %s in received block %s", defaultBlock.Element.Type, defaultBlock.BlockID)
			}

		}
	}

	err = slackServer.slackAPIInit()
	if err != nil {
		return response, err
	}
	User, err := slackServer.slackAPI.GetUser(mapValues["input_select_wobject_assignee"])
	if err != nil {
		fmt.Printf("Error: was not able to find user: '%s', with error: %v", mapValues["input_select_wobject_assignee"], err)
		return response, err
	}

	fmt.Printf("Request filled mapValues: %v\n", mapValues)
	wobject := human_api_types.Wobject{}
	wobject.WorkerID = User.Name

	wobject.Title = mapValues["input_plain_text_title"]
	wobject.Description = mapValues["input_plain_text_description"]
	wobject.Type = mapValues["input_select_wobject_type"]
	wobject.Status = mapValues["input_select_wobject_status"]

	err = slackServer.ProvisionWobject(&wobject)
	if err != nil {
		log.Printf("Error received when creating the wobject: %v", err)
		return response, err
	}

	ticketLink := fmt.Sprintf("<%s|%s-%s>", wobject.Link, wobject.Type, wobject.Id)
	messageText := fmt.Sprintf("✅ Successfully created wobject!\n*%s: %s*", ticketLink, wobject.Title)

	// 6. Construct the Block Kit payload
	response = slackBlockKitResponse{
		ResponseType: "in_channel", // Makes the message visible to everyone in the channel
		Blocks: []block{
			{
				Type: "section",
				Text: blockText{
					Type: "mrkdwn",
					Text: messageText,
				},
			},
		},
	}
	return response, nil

}

func (slackServer *SlackServer) LoadIneractiveMenu(menuFileName string, replacements *map[string]string) (menu *slackBlockKitResponse, err error) {
	fullPath := filepath.Join(*slackServer.Configuration.SlackBlockKitDirPath, menuFileName)
	menu = &slackBlockKitResponse{}

	jsonDataString, err := slackServer.loadFileWithReplacements(fullPath, replacements)
	if err != nil {
		log.Printf("error loading interactive menu from file %s: %v, file: %s", fullPath, err, jsonDataString)
		return nil, err
	}

	err = json.Unmarshal([]byte(jsonDataString), menu)
	if err != nil {
		log.Printf("Error unmarshalling interactive menu: %v", err)
	}
	return menu, err
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

	tokenAny, ok := data["token"]
	if !ok {
		http.Error(w, "Error handling request, Slack App Any-token iether wrong or does not present", http.StatusBadRequest)
		log.Printf("Error in received AnyToken. Can not convert to string: %v", tokenAny)
		return
	}

	token, ok := tokenAny.(string)
	if !ok {
		http.Error(w, "Error handling request, Slack App String-token", http.StatusBadRequest)
		log.Printf("Error parsing TokenAny: %v, to TokenString %v", tokenAny, token)
		return
	}

	if token != *slackServer.Configuration.VerificationToken {
		http.Error(w, "Error handling request, Slack App String-token iether wrong or does not present", http.StatusBadRequest)
		log.Printf("Error in received StringToken: '%s',  Does not equal configured: '%s'", token, *slackServer.Configuration.VerificationToken)
		return

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
	var response slackBlockKitResponse
	domain := strings.Split(text, " ")[0]
	if text != "" {
		log.Printf("Handling text '%s'", text)
	}
	switch domain {
	case "":
		response, err = slackServer.LoadGenericMenu("slack_main.json", nil)
	case "wobj":
		text = text[len("wobj"):]
		text = strings.TrimLeftFunc(text, unicode.IsSpace)
		response, err = slackServer.wobjHandler(text)
	default:
		http.Error(w, fmt.Sprintf("Unknown domain: %s", domain), http.StatusBadRequest)
		log.Printf("Unknown domain: %s", domain)
		return
	}

	if err != nil {
		response = slackServer.GenerateErrorResponse("Error handling request" + fmt.Sprintf("%v", err))
		statusCode = http.StatusBadRequest
	}
	slackServer.SendResponse(w, statusCode, response)
}

func (slackServer *SlackServer) GenerateErrorResponse(errMessage string) slackBlockKitResponse {

	//ticketLink := fmt.Sprintf("<http://your-ticketing-system.com/tickets/%d|TICKET-%d>", ticketID, ticketID)
	messageText := fmt.Sprintf("❌ Error handlig request!\n*%s*", errMessage)

	// 6. Construct the Block Kit payload
	response := slackBlockKitResponse{
		ResponseType: "in_channel", // Makes the message visible to everyone in the channel
		Blocks: []block{
			{
				Type: "section",
				Text: blockText{
					Type: "mrkdwn",
					Text: messageText,
				},
			},
		},
	}
	return response
}

func (slackServer *SlackServer) SendResponse(w http.ResponseWriter, statusCode int, response slackBlockKitResponse) (err error) {

	log.Printf("Returning response to client statusCode: %d, responseAny: %v", statusCode, response)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Error encoding response", http.StatusBadRequest)
	}
	return err
}

func (slackServer *SlackServer) LoadGenericMenu(fileName string, replacements *map[string]string) (response slackBlockKitResponse, err error) {

	fullPath := filepath.Join(*slackServer.Configuration.SlackBlockKitDirPath, fileName)
	jsonDataString, err := slackServer.loadFileWithReplacements(fullPath, replacements)
	if err != nil {
		log.Printf("Error reading file %s: %v, file: %s", fullPath, err, jsonDataString)
		return response, err
	}
	response = slackBlockKitResponse{}
	err = json.Unmarshal([]byte(jsonDataString), &response)
	if err != nil {
		log.Printf("Error Unmarshalling file: %s", jsonDataString)
		return response, err
	}

	log.Printf("Successfully loaded Generic Menu %s", fileName)
	response.ResponseType = "in_channel"
	return response, nil
}

func (slackServer *SlackServer) wobjHandler(text string) (response slackBlockKitResponse, err error) {
	command := strings.Split(text, " ")[0]
	log.Printf("Handling command '%s'", command)
	if command == "menu" {
		response, err = slackServer.LoadGenericMenu("slack_wobj.json", nil)
	} else {
		err = fmt.Errorf("unknown command: %s", command)
	}

	return response, err
}

func (slackServer *SlackServer) loadFileWithReplacements(fullPath string, replacements *map[string]string) (jsonDataString string, err error) {
	jsonData, err := os.ReadFile(fullPath)
	if err != nil {
		log.Printf("error reading file %s: %v", fullPath, err)
		return "", err
	}

	jsonDataString = string(jsonData)
	if replacements != nil {
		for key, val := range *replacements {
			jsonDataString = strings.ReplaceAll(jsonDataString, key, val)
		}
	}
	return jsonDataString, err
}

func (slackServer *SlackServer) loadJsonFile(fileName string, replacements *map[string]string) (map[string]any, error) {
	fullPath := filepath.Join(*slackServer.Configuration.SlackBlockKitDirPath, fileName)
	jsonDataString, err := slackServer.loadFileWithReplacements(fullPath, replacements)
	if err != nil {
		log.Printf("error reading file %s: %v, file: %s", fullPath, err, jsonDataString)
		return nil, err
	}
	var result map[string]any
	err = json.Unmarshal([]byte(jsonDataString), &result)
	if err != nil {
		log.Printf("error unmarshalling file %s: %v, file: %s", fullPath, err, jsonDataString)
		return nil, err
	}
	return result, nil
}

func (slackServer *SlackServer) sendResponseUrlMessage(responseUrl string, response slackBlockKitResponse) error {
	log.Printf("Sending response using POST: %v", response)
	jsonMessage, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling response: %v", err)
		return err
	}

	// 7. Send the message to the response_url
	resp, err := http.Post(responseUrl, "application/json", bytes.NewBuffer(jsonMessage))
	if err != nil {
		log.Printf("Error sending success response: %v", err)
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (slackServer *SlackServer) sendResponseUrlMessageOld(responseUrl string, payload map[string]any) error {

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

func (slackServer *SlackServer) slackAPIInit() error {
	api, err := slack_api.SlackAPINew(config_pol.WithConfigurationFile(slackServer.Configuration.SlackAPIConfigurationFilePath))

	if err != nil {
		return err
	}
	slackServer.slackAPI = api
	return nil
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

type slackBlockKitResponse struct {
	ResponseType string  `json:"response_type"`
	Blocks       []block `json:"blocks"`
}

type block struct {
	Type     string    `json:"type"`
	BlockID  string    `json:"block_id,omitempty"`
	Optional bool      `json:"optional,omitempty"`
	Label    blockText `json:"label,omitzero"`
	Text     blockText `json:"text,omitzero"`
	Element  Element   `json:"element,omitzero"`
	Elements []Element `json:"elements,omitzero"`
}

type blockText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type Element struct {
	Type          string    `json:"type"`
	Text          blockText `json:"text,omitzero"`
	ActionID      string    `json:"action_id,omitempty"`
	Value         string    `json:"value,omitempty"`
	InitialUser   string    `json:"initial_user,omitempty"`
	InitialOption Option    `json:"initial_option,omitzero"`
	Options       []Option  `json:"options,omitzero"`
}

type Option struct {
	Value string    `json:"value"`
	Text  blockText `json:"text,omitzero"`
}

// sendSuccessResponse builds the Block Kit message and posts it to the response_url.
func (slackServer *SlackServer) sendSuccessResponse(responseURL string, ticketID int, ticketTitle string) {
	// 5. Build the formatted message using markdown
	ticketLink := fmt.Sprintf("<http://your-ticketing-system.com/tickets/%d|TICKET-%d>", ticketID, ticketID)
	messageText := fmt.Sprintf("✅ Successfully created ticket!\n*%s: %s*", ticketLink, ticketTitle)

	// 6. Construct the Block Kit payload
	response := slackBlockKitResponse{
		ResponseType: "in_channel", // Makes the message visible to everyone in the channel
		Blocks: []block{
			{
				Type: "section",
				Text: blockText{
					Type: "mrkdwn",
					Text: messageText,
				},
			},
		},
	}

	jsonMessage, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling response: %v", err)
		return
	}

	// 7. Send the message to the response_url
	resp, err := http.Post(responseURL, "application/json", bytes.NewBuffer(jsonMessage))
	if err != nil {
		log.Printf("Error sending success response: %v", err)
		return
	}
	defer resp.Body.Close()
}

// InteractiveRequest is the top-level struct for the entire request.
type InteractiveRequest struct {
	Token               string                      `json:"token"`
	Type                string                      `json:"type"`
	User                InteractiveRequestUser      `json:"user"`
	APIAppID            string                      `json:"api_app_id"`
	Container           InteractiveRequestContainer `json:"container"`
	TriggerID           string                      `json:"trigger_id"`
	Team                InteractiveRequestTeam      `json:"team"`
	IsEnterpriseInstall bool                        `json:"is_enterprise_install"`
	Channel             InteractiveRequestChannel   `json:"channel"`
	State               InteractiveRequestState     `json:"state"`
	ResponseURL         string                      `json:"response_url"`
	Actions             []InteractiveRequestAction  `json:"actions"`
	Message             any                         `json:"message"`
}

// InteractiveRequestUser represents the user who initiated the action.
type InteractiveRequestUser struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	TeamID string `json:"team_id"`
}

// InteractiveRequestContainer holds information about where the interaction occurred.
type InteractiveRequestContainer struct {
	Type        string `json:"type"`
	MessageTs   string `json:"message_ts"`
	ChannelID   string `json:"channel_id"`
	IsEphemeral bool   `json:"is_ephemeral"`
}

// InteractiveRequestTeam represents the Slack team.
type InteractiveRequestTeam struct {
	ID     string `json:"id"`
	Domain string `json:"domain"`
}

// InteractiveRequestChannel represents the Slack channel.
type InteractiveRequestChannel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// InteractiveRequestState contains the values of all input blocks.
type InteractiveRequestState struct {
	Values map[string]map[string]InteractiveRequestStateValue `json:"values"`
}

// InteractiveRequestStateValue holds the value of a single input element.
type InteractiveRequestStateValue struct {
	Type           string                   `json:"type"`
	SelectedUser   string                   `json:"selected_user,omitempty"`
	SelectedOption InteractiveRequestOption `json:"selected_option,omitzero"`
	Value          string                   `json:"value,omitempty"`
}

type InteractiveRequestOption struct {
	Text  InteractiveRequestText `json:"text"`
	Value string                 `json:"value"`
}

// InteractiveRequestAction represents one of the actions triggered (e.g., a button click).
type InteractiveRequestAction struct {
	ActionID string                 `json:"action_id"`
	BlockID  string                 `json:"block_id"`
	Text     InteractiveRequestText `json:"text"`
	Value    string                 `json:"value"`
	Type     string                 `json:"button"`
	ActionTs string                 `json:"action_ts"`
}

// InteractiveRequestText is a common object for text elements in Block Kit.
type InteractiveRequestText struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Emoji bool   `json:"emoji,omitempty"`
}
