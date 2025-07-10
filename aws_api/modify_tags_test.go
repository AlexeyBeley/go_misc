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
	conf_path := "/opt/ModifyTagsConfig.json"
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

func TestAddTagsNetworkInterfaces(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		err := AddTagsNetworkInterfaces(realConfig)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestAddTagsNatGateways(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		err := AddTagsNatGateways(realConfig)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestAddTagsInstances(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		err := AddTagsInstances(realConfig)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestAddTagsElasticIps(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		err := AddTagsElasticIps(realConfig)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestAddTagsVolumes(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		err := AddTagsVolumes(realConfig)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestAddTagsLaunchTemplates(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		err := AddTagsLaunchTemplates(realConfig)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestAddTagsImages(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		err := AddTagsImages(realConfig)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestAddTagsSnapshots(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		err := AddTagsSnapshots(realConfig)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestAddTagsKeyPairs(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		err := AddTagsKeyPairs(realConfig)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestAddTagsSecurityGroups(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		err := AddTagsSecurityGroups(realConfig)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestAddTagsLoadBalancers(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		err := AddTagsLoadBalancers(realConfig)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestAddTagsTargetGroups(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		err := AddTagsTargetGroups(realConfig)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestAddTagsAutoScalingGroups(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		err := AddTagsAutoScalingGroups(realConfig)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestAddTagsRDSClusters(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		err := AddTagsRDSClusters(realConfig)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestAddTagsRDSInstances(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		err := AddTagsRDSInstances(realConfig)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestAddTagsS3Buckets(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		err := AddTagsS3Buckets(realConfig)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestAddTagsECSTaskdefinitions(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		err := AddTagsECSTaskdefinitions(realConfig)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestAddTagsCloudwatchLogGroups(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		err := AddTagsCloudwatchLogGroups(realConfig)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}
