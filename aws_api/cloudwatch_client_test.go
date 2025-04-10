package aws_api

import (
	"log"
	"os"
	"testing"
	"time"
)

func loadRealConfig() Configuration {
	os.Getenv("CONFIG_PATH")
	conf_path := "/tmp/cloudwatch.json"
	config, err := LoadConfig(conf_path)
	if err != nil {
		log.Fatalf("%v", err)
	}
	return config
}

func TestFetchCloudwatchLogStream(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		err := FetchCloudwatchLogStream(realConfig.Region, realConfig.LogGroup, realConfig.LogGroup)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestLogStreamsCache(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		err := LogStreamsCache(realConfig.Region, realConfig.LogGroup)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestYieldCloudwatchLogStream(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		var epochStartSeconds, epochEndSeconds int64

		streamName := ""
		counter := 0
		realConfig := loadRealConfig()
		nowUTC := time.Now().UTC()
		_ = nowUTC
		epochEndSeconds = nowUTC.Unix()
		epochStartSeconds = epochEndSeconds - 24*60*60
		epochEndMiliSeconds := epochEndSeconds * 1000
		epochStartMiliSeconds := epochStartSeconds * 1000

		err := YieldCloudwatchLogStream(realConfig.Region, realConfig.LogGroup, streamName, &epochStartMiliSeconds, &epochEndMiliSeconds, BytesSummarizer(&counter))
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestSubnetsFlowStreamByteSum(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		err := SubnetsFlowStreamByteSum(realConfig.Region, realConfig.LogGroup)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}
