package main

import (
	"github.com/AlexeyBeley/go_common/logger"
	"github.com/AlexeyBeley/go_misc/kub_api"
)

var lg = &(logger.Logger{})

func main() {

	api, err := kub_api.KubAPINew()
	if err != nil {
		panic(err)
	}
	api.GetPods()
}
