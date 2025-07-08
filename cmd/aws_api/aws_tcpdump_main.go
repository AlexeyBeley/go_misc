package main

import (
	"github.com/AlexeyBeley/go_common/logger"
	"github.com/AlexeyBeley/go_misc/aws_api"
)

var lg = &(logger.Logger{})

var actions map[string]any

func main() {

	//addresses := flag.String("address-filter", "", "comma Subnets")
	// flag.Int(name, defaultValue, usage) returns an *int
	//age := flag.Int("age", 0, "Your age")
	// flag.Bool(name, defaultValue, usage) returns a *bool
	//verbose := flag.Bool("verbose", false, "Enable verbose output")

	actions = make(map[string]any, 0)

	config_file_path := "/opt/aws_api_go/AWSTCPDumpConfig.json"
	lg.InfoF("Initializing: %s", config_file_path)
	awsTCPDumpNew, err := aws_api.AWSTCPDumpNew(config_file_path)
	if err != nil {
		panic(err)
	}

	actions["start"] = awsTCPDumpNew.Start

	err = awsTCPDumpNew.Start()
	if err != nil {
		panic(err)
	}

}
