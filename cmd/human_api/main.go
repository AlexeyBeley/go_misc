package main

import (
	"github.com/AlexeyBeley/go_common/logger"
	actionManager "github.com/AlexeyBeley/go_misc/action_manager"
	humanAPI "github.com/AlexeyBeley/go_misc/human_api"
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
	(*actionManager).ActionMap = map[string]any{"GetLamdaLogs": humanApi.TicketAction}

}
