package aws_api

import (
	"testing"
)

func TestStart(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		config_file_path := "/opt/aws_api_go/AWSTCPDumpConfig.json"
		lg.InfoF("Initializing: %s", config_file_path)
		awsTCPDumpNew, err := AWSTCPDumpNew(config_file_path)
		if err != nil {
			panic(err)
		}
		err = awsTCPDumpNew.Start()

		if err != nil {
			t.Errorf("%v", err)
		}

	})
}
