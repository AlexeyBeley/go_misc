package main

import (
	"os"

	actionManager "github.com/AlexeyBeley/go_misc/action_manager"
	common_utils "github.com/AlexeyBeley/go_misc/common_utils"
	humanAPI "github.com/AlexeyBeley/go_misc/human_api"
	humanAPISlackServer "github.com/AlexeyBeley/go_misc/human_api/slack_server"
	"github.com/AlexeyBeley/go_misc/logger"
)

var lg = &(logger.Logger{})

func main() {

	actionManager, err := actionManager.ActionManagerNew()
	if err != nil {
		panic(err)
	}

	humanApi, err := humanAPI.HumanAPINew()
	if err != nil {
		panic(err)
	}

	token := os.Getenv("SLACK_APP_TOKEN")

	server := humanAPISlackServer.SlackServerNew(nil, common_utils.StrPTR(token))
	(*actionManager).ActionMap = map[string]any{
		"TicketAction":   humanApi.TicketAction,
		"SlackBotServer": server.Start}

	action := "SlackBotServer"
	err = actionManager.RunAction(&action)

	if err != nil {
		panic(err)
	}

}
