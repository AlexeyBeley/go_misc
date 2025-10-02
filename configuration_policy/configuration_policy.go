package configuration_policy

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
)

type ConfigurationPolicy struct {
	ConfigurationFilePath *string
}

func (Config ConfigurationPolicy) InitFromFile(APIConfigurationPointer any) error {
	if reflect.ValueOf(APIConfigurationPointer).Kind() != reflect.Ptr || reflect.ValueOf(APIConfigurationPointer).Elem().Kind() != reflect.Struct {
		return fmt.Errorf("out parameter must be a pointer to a struct, got %T", APIConfigurationPointer)
	}

	jsonData, err := os.ReadFile(*Config.ConfigurationFilePath)
	if err != nil {
		return err
	}

	err = json.Unmarshal(jsonData, APIConfigurationPointer)
	return err
}

type Configurable interface {
	SetConfiguration(Config any) error
}

type Option func(Configurable, any) error

func WithConfigurationFile(ConfigurationFilePath *string) func(api Configurable, APIConfiguration any) error {

	return func(api Configurable, APIConfiguration any) error {
		err := ConfigurationPolicy{ConfigurationFilePath: ConfigurationFilePath}.InitFromFile(APIConfiguration)
		if err != nil {
			return err
		}
		err = api.SetConfiguration(APIConfiguration)
		if err != nil {
			return err
		}
		return nil
	}
}
