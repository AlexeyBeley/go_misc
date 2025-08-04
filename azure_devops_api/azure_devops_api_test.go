package azure_devops_api

import (
	"fmt"
	"log"
	"os"
	"testing"
)

func loadRealConfig() Configuration {
	os.Getenv("CONFIG_PATH")
	conf_path := "/tmp/azure_devops_api_configuration_values.json"
	config, err := LoadConfig(conf_path)
	if err != nil {
		log.Fatalf("%v", err)
	}
	return config
}

func TestHoreyClient(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		err := HoreyClient(realConfig)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestGetAllWits(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		err := GetAllWits(realConfig)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestDownloadAllWits(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		err := DownloadAllWits(realConfig, "/tmp/wit.json")
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestSubmitSprintStatus(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		fmt.Print("todo:\n")
	})
}

func TestGetTeamUuid(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		ret, err := GetTeamUuid(realConfig)
		if err != nil {
			log.Fatalf("%v", err)
		}
		if len(ret) == 0 {
			t.Errorf("%v", ret)
		}
	})
}

func TestGetIteration(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		ret, err := GetIteration(realConfig)
		if err != nil {
			log.Fatalf("%v", err)
		}
		if len(ret.Id) == 0 {
			t.Errorf("%v", ret)
		}
	})
}
