package aws_api

import (
	"testing"
)

func TestYieldSecrets(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		api := SecretsmanagerAPINew(StrPtr("us-west-2"), StrPtr("default"))

		ret, err := api.YieldSecrets(nil, nil)
		if err != nil {
			t.Errorf("%v", err)
			return
		}

		if len(ret) == 0 {
			t.Errorf("Found '%d' secrets", len(ret))
		}

		lg.InfoF("Found '%d' secrets", len(ret))
	})
}
