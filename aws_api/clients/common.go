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

func CallbackEcho(Something any) error {
	fmt.Printf("raw AWS API response: %v\n", Something)
	return nil
}

var lg = &(logger.Logger{})

func Int32Ptr(i int32) *int32 {
	return &i
}

func Int64Ptr(i int64) *int64 {
	return &i
}

func StrPtr(src string) *string {
	return &src
}

func BoolPtr(src bool) *bool {
	return &src
}
