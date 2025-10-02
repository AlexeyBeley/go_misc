package azure_devops_api

import (
	"fmt"
	"log"
	"os"
	"testing"

	config_pol "github.com/AlexeyBeley/go_misc/configuration_policy"
)

var GlobalAzureDevopsAPIConfigurationFilePath = "/opt/azure_devops_api/configuration.json"

func loadRealConfig() Configuration {
	os.Getenv("CONFIG_PATH")
	conf_path := "/opt/azure_devops_api/configuration.json"
	config, err := LoadConfig(conf_path)
	if err != nil {
		log.Fatalf("%v", err)
	}
	return config
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

func TestGetRepositories(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {

		api, err := AzureDevopsAPINew(config_pol.WithConfigurationFile(&GlobalAzureDevopsAPIConfigurationFilePath))
		if err != nil {
			log.Fatalf("%v", err)
		}

		ret, err := api.GetRepositories()
		if err != nil {
			log.Fatalf("%v", err)
		}

		for _, rep := range ret {
			fmt.Printf("Repo: %s\n", *rep.Name)
		}
	})
}

func TestGetPipelineDefinitions(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		api, err := AzureDevopsAPINew(config_pol.WithConfigurationFile(&GlobalAzureDevopsAPIConfigurationFilePath))
		if err != nil {
			log.Fatalf("%v", err)
		}

		ret, err := api.GetPipelineDefinitions()
		if err != nil {
			log.Fatalf("%v", err)
		}
		_ = ret

	})
}

func TestGetPipelineDefinition(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {

		api, err := AzureDevopsAPINew(config_pol.WithConfigurationFile(&GlobalAzureDevopsAPIConfigurationFilePath))
		if err != nil {
			log.Fatalf("%v", err)
		}

		ret, err := api.GetPipelineDefinition()
		if err != nil {
			log.Fatalf("%v", err)
		}
		_ = ret

	})
}
