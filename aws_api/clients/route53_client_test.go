package aws_api

import (
	"testing"
)

func TestYieldHostedZones(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		api := Route53APINew(StrPtr("default"))

		ret, err := api.YieldHostedZones(nil, nil)
		if err != nil {
			t.Errorf("%v", err)
			return
		}

		if len(ret) == 0 {
			t.Errorf("Found '%d' hosted zones", len(ret))
		}

		lg.InfoF("Found '%d' hosted zones", len(ret))
	})
}

func TestYieldResourceRecordSets(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		api := Route53APINew(StrPtr("prod"))

		ret, err := api.YieldResourceRecordSets(nil)
		if err != nil {
			t.Errorf("%v", err)
			return
		}

		if len(ret) == 0 {
			t.Errorf("Found '%d' records", len(ret))
		}

		lg.InfoF("Found '%d' records", len(ret))
	})
}
