package slack_api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	config_pol "github.com/AlexeyBeley/go_misc/configuration_policy"
)

type Configuration struct {
	BotUserOAuthToken string `json:"BotUserOAuthToken"`
}

type SlackAPI struct {
	Configuration *Configuration
}

type UserProfile struct {
	RealName    string `json:"real_name"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Image72     string `json:"image_72"` // Example of another field you could get
}

// User contains the top-level user object.
type User struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"` // This is the username, e.g., "jdoe"
	Profile UserProfile `json:"profile"`
}

// UserInfoResponse is the top-level structure of the API response.
type UserInfoResponse struct {
	OK    bool   `json:"ok"`
	User  User   `json:"user"`
	Error string `json:"error,omitempty"` // Captures the error message if "ok" is false
}

func SlackAPINew(options ...config_pol.Option) (*SlackAPI, error) {

	slackAPI := &SlackAPI{}
	configuration := &Configuration{}
	for _, option := range options {
		option(slackAPI, configuration)

	}

	return slackAPI, nil
}

func (slackAPI *SlackAPI) SetConfiguration(Config any) error {
	AzureDevopsAPIConfig, ok := Config.(*Configuration)
	if !ok {
		return fmt.Errorf("was not able to convert %v to slackAPIConfig", Config)
	}
	slackAPI.Configuration = AzureDevopsAPIConfig
	return nil
}

func (slackAPI *SlackAPI) GetUser(userID string) (*User, error) {

	url := fmt.Sprintf("https://slack.com/api/users.info?user=%s", userID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("Failed to create HTTP request: %s", err)
	}
	req.Header.Add("Authorization", "Bearer "+slackAPI.Configuration.BotUserOAuthToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to send HTTP request: %s", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %s", err)
	}

	// 6. Decode the JSON response into our Go structs.
	var userInfoResp UserInfoResponse
	if err := json.Unmarshal(body, &userInfoResp); err != nil {
		log.Fatalf("Failed to decode JSON response: %s", err)
	}
	// 7. Check if the API call was successful.
	if !userInfoResp.OK {
		log.Fatalf("Slack API returned an error: %s", userInfoResp.Error)
	}
	user := userInfoResp.User
	return &user, nil
}
