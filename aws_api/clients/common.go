package aws_api

import (
	"fmt"

	"github.com/AlexeyBeley/go_common/logger"
)

type GenericCallback func(any) error
type GenericCallbackNG func(any) (bool, error)

func AggregatorInitializer(objects *[]any) func(any) error {
	return func(object any) error {
		*objects = append(*objects, object)
		return nil
	}
}

func AggregatorInitializerNG(objects *[]any) func(any) (bool, error) {
	return func(object any) (continuePagination bool, err error) {
		*objects = append(*objects, object)
		return true, nil
	}
}

func Counter() func(any) (bool, error) {
	counter := 0
	return func(Something any) (bool, error) {
		counter++
		fmt.Printf("Counter: %d\n", counter)
		return true, nil
	}
}

func CallbackEcho(Something any) error {
	fmt.Printf("raw AWS API response: %v\n", Something)
	return nil
}

var lg = &(logger.Logger{AddDateTime: true})

func IntPtr(i int) *int {
	return &i
}
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
