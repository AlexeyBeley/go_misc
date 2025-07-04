package aws_api

import (
	"fmt"

	"github.com/AlexeyBeley/go_common/logger"
)

type GenericCallback func(any) error

func AggregatorInitializer(objects *[]any) func(any) error {
	return func(object any) error {
		*objects = append(*objects, object)
		return nil
	}
}

func Counter() func(any) error {
	counter := 0
	return func(Something any) error {
		counter++
		fmt.Printf("Counter: %d\n", counter)
		return nil
	}
}


func Echo(Something any) error {
		fmt.Printf("raw AWS API response: %v\n", Something)
		return nil
	}

var lg = &(logger.Logger{})
