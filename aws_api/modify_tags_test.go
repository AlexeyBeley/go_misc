package aws_api

import (
	"encoding/json"
	"log"
	"os"
	"testing"
)

func LoadDynamicConfig(configFilePath string) (config any, err error) {
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}
	return config, nil
}

func loadRealConfig() ModifyTagsConfig {
	conf_path := "/tmp/ModifyTagsConfig.json"
	config, err := LoadDynamicConfig(conf_path)
	if err != nil {
		log.Fatalf("%v", err)
	}

	modifyTagsConfig := ModifyTagsConfig{}
	err = modifyTagsConfig.InitFromM(config)
	if err != nil {
		log.Fatalf("%v", err)
	}
	return modifyTagsConfig
}

func TestAddTagNetworkInterfaces(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		err := AddTagNetworkInterfaces(realConfig)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestAddTagNatGateways(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		err := AddTagNatGateways(realConfig)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestAddTagInstances(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		err := AddTagInstances(realConfig)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestAddTagElasticIps(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		err := AddTagElasticIps(realConfig)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}
