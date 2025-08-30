package main

import (
	"github.com/AlexeyBeley/go_misc/logger"
	actionManager "github.com/AlexeyBeley/go_misc/action_manager"
	humanAPI "github.com/AlexeyBeley/go_misc/human_api"
	humanAPISlackServer "github.com/AlexeyBeley/go_misc/human_api/slack_server"
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
	(*actionManager).ActionMap = map[string]any{
		"TicketAction": humanApi.TicketAction,
		"server":       humanAPISlackServer.Start}

}
