package aws_api

type GenericCallback func(any) error


func AggregatorInitializer(objects *[]any) func(any) error {
	return func(object any) error {
		*objects = append(*objects, object)
		return nil
	}
}