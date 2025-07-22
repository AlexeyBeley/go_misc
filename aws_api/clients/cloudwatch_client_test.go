package aws_api

import (
	"fmt"
	"testing"
)

func TestGetMetricAlarms(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {

		realConfig := loadRealConfig()
		api := CloudwatchAPINew(&realConfig.Region, nil)

		objects := make([]any, 0)
		err := api.GetMetricAlarms(CallbackEcho, nil)
		if err != nil {
			t.Errorf("%v", err)
		}
		fmt.Printf("Todo: refactor Counter to Cacher Objects: %d\n", len(objects))
	})
}
