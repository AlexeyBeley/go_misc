package main

import (
	"github.com/AlexeyBeley/go_common/logger"
	"github.com/AlexeyBeley/go_misc/aws_api"
)

var lg = &(logger.Logger{})

func main() {
	config_file_path := "/opt/aws_api_go/AWSTCPDumpConfig.json"
	lg.Infof("Initializing: %s", config_file_path)
	err := aws_api.AWSTCPDumpStart(config_file_path)
	if err != nil {
		panic(err)
	}

}
