package aws_api

import "testing"

func TestGetAccount(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		api := STSAPINew(nil)
		_, err := api.GetAccount()
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}
