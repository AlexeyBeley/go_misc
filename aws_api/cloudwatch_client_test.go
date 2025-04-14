package aws_api

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
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

		err := YieldCloudwatchLogStream(realConfig.Region, realConfig.LogGroup, streamName, &epochStartMiliSeconds, &epochEndMiliSeconds, BytesSummarizerInt(&counter))
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func BytesSummarizerInt(aggregator *int) func(*cloudwatchlogs.OutputLogEvent) error {
	return func(event *cloudwatchlogs.OutputLogEvent) error {
		if strings.Contains(*event.Message, "NODATA") {
			return nil
		}
		//fmt.Println("  ", *event.Message)
		stringSplit := strings.Split(*event.Message, " ")
		srcaddr := stringSplit[3]
		dstaddr := stringSplit[4]
		ipSrc := net.ParseIP(srcaddr)
		ipDst := net.ParseIP(dstaddr)
		if ipSrc == nil || ipDst == nil {
			return fmt.Errorf("srcaddr: %v, dstaddr: %v ", srcaddr, dstaddr)
		}

		if ipSrc.IsPrivate() && ipDst.IsPrivate() {
			return nil
		}

		bytes, err := strconv.Atoi(stringSplit[9])
		if err != nil {
			return err
		}
		*aggregator += bytes
		return nil
	}
}
