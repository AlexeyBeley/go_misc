package main

import (
	actionManager "github.com/AlexeyBeley/go_misc/action_manager"
	config_pol "github.com/AlexeyBeley/go_misc/configuration_policy"
	humanAPI "github.com/AlexeyBeley/go_misc/human_api"
	humanAPISlackServer "github.com/AlexeyBeley/go_misc/human_api/slack_server"
	"github.com/AlexeyBeley/go_misc/logger"
)

var lg = &(logger.Logger{})
var GlobalSlackServerConfigurationFilePath = "/opt/human_api/slack_server_configuration.json"

func main() {

	actionManager, err := actionManager.ActionManagerNew()
	if err != nil {
		panic(err)
	}

	humanApi, err := humanAPI.HumanAPINew()
	if err != nil {
		panic(err)
	}

	server := humanAPISlackServer.SlackServerNew(config_pol.WithConfigurationFile(&GlobalSlackServerConfigurationFilePath))
	(*actionManager).ActionMap = map[string]any{
		"TicketAction":   humanApi.TicketAction,
		"SlackBotServer": server.Start}

	action := "SlackBotServer"
	err = actionManager.RunAction(&action)

	if err != nil {
		panic(err)
	}

}
