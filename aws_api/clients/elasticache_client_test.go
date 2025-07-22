package aws_api

import (
	"log"
	"testing"
)

func loadElasticacheTestConfig() Configuration {
	conf_path := "/opt/aws_api_go/test_elasticache.json"
	config, err := LoadConfig(conf_path)
	if err != nil {
		log.Fatalf("%v", err)
	}
	return config
}

func TestGetCacheClusters(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadElasticacheTestConfig()
		api := ElasticacheAPINew(&realConfig.Region, nil)

		ret, err := api.GetCacheClusters(nil)
		if err != nil {
			t.Errorf("%v", err)
		}
		if len(ret) == 0 {
			t.Errorf("len(ret)  %d", len(ret))
		}
	})
}
