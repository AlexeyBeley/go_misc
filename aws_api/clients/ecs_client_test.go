package aws_api

import (
	"testing"
)

type Stringer interface {
	String() string
}

func TestGetTaskDefinitions(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		api := ECSAPINew(nil, nil)

		err := api.GetTaskDefinitions(CallbackEcho, nil)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestGetTaskDefinitionFamilies(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		api := ECSAPINew(nil, nil)

		err := api.GetTaskDefinitionFamilies(CallbackEcho, nil)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}
