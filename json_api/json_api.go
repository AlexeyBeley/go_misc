package json_api

import (
	"encoding/json"
	"os"
	hLogger "github.com/AlexeyBeley/go_misc/logger"
)

var lg = hLogger.Logger{}

func WirteToFile(Data any, DstFilePath *string) error {
	jsonData, err := json.MarshalIndent(Data, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(*DstFilePath, jsonData, 0644)
	if err != nil {
		return err
	}
	lg.InfoF("Wrote JSON to file: '%s'", *DstFilePath)
	return nil

}
