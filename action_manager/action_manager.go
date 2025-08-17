package action_manager

import (
	"fmt"
	"reflect"
)

type ActionManager struct {
	ActionMap map[string]any
}

func ActionManagerNew() (*ActionManager, error) {
	ret := &ActionManager{}
	return ret, nil
}

func (actionManager *ActionManager) RunAction(actionName *string) error {
	fn, ok := actionManager.ActionMap[*actionName]
	if !ok {
		return fmt.Errorf("action '%s' not found", *actionName)
	}

	funcValue := reflect.ValueOf(fn)
	if funcValue.Kind() != reflect.Func {
		return fmt.Errorf("'%s' is not a function", *actionName)
	}

	in := make([]reflect.Value, 0)
	results := funcValue.Call(in)
	if len(results) != 1 {
		return fmt.Errorf("only result acceptable is 'error', recived %d results", len(results))
	}

	result := results[0].Interface()
	if result == nil {
		return nil
	}

	err, ok := result.(error)
	if !ok {
		return fmt.Errorf("action '%s' result expected to be 'error' but received %v", *actionName, result)
	}

	return err
}
