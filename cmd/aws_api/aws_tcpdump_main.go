package main

import (
	"github.com/AlexeyBeley/go_common/logger"
	"github.com/AlexeyBeley/go_misc/aws_api"
)

var lg = &(logger.Logger{})

func main() {

	//_, err := aws_api.AWSTCPDumpAnalize("/tmp/nat_analyzer.log")

	config_file_path := "/opt/aws_api_go/AWSTCPDumpConfig.json"
	lg.InfoF("Initializing: %s", config_file_path)
	awsTCPDumpNew, err := aws_api.AWSTCPDumpNew(config_file_path)
	if err != nil {
		panic(err)
	}
	err = awsTCPDumpNew.Start()
	if err != nil {
		panic(err)
	}

}
