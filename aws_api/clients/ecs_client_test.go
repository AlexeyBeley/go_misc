package aws_api

import (
	"fmt"
	"testing"
)

type Stringer interface {
	String() string
}

func CallbackEcho(obj any) error {
	Value, ok := obj.(Stringer)
	var dst string
	if !ok {
		ValueString, ok := obj.(*string)
		if !ok {
			return fmt.Errorf("error parsing results: %v", obj)
		} else {
			dst = *ValueString
		}
	} else {
		dst = Value.String()
	}

	fmt.Println(dst)
	return nil
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
