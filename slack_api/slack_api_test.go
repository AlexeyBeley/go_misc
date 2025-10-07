package slack_api

import (
	"log"
	"testing"

	config_pol "github.com/AlexeyBeley/go_misc/configuration_policy"
)

var GlobaSlackAPIConfigurationFilePath = "/opt/slack_api/configuration.json"

func TestLoadConfiguration(t *testing.T) {
	t.Run("Init test", func(t *testing.T) {

		api, err := SlackAPINew(config_pol.WithConfigurationFile(&GlobaSlackAPIConfigurationFilePath))
		if err != nil {
			log.Fatalf("%v", err)
		}
		ret, err := api.GetUser("")
		if err != nil {
			log.Fatalf("%v", err)
		}
		_ = ret

	})
}
