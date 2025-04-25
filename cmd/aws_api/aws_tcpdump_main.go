package main

import (
	"github.com/AlexeyBeley/go_common/logger"
	"github.com/AlexeyBeley/go_misc/aws_api"
)

var lg = &(logger.Logger{})

func main() {
	config_file_path := "/tmp/SubnetRecordingConfig.json"
	aws_api.AWSTCPDumpStart(config_file_path)
}
