module github.com/AlexeyBeley/go_misc

go 1.23.5

require (
	github.com/AlexeyBeley/go_common v0.0.0-20250329144146-1a569c008ae4
	github.com/aws/aws-sdk-go v1.55.6
)

require github.com/jmespath/go-jmespath v0.4.0 // indirect

replace github.com/AlexeyBeley/go_common => ../go_common
