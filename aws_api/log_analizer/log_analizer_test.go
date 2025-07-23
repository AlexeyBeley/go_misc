package aws_api

import (
	"os"
	"testing"

	clients "github.com/AlexeyBeley/go_misc/aws_api/clients"
)

func TestGetLambdaLogs(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {

		os.Args = []string{"program_name", "--LambdaName", "test", "--Region", "us-west-2"}
		awsLogAnalizer, err := AWSLogAnalizerNew()

		if err != nil {
			t.Errorf("%v", err)
		}
		actionManager, err := awsLogAnalizer.GenerateActionManager()

		if err != nil {
			t.Errorf("%v", err)
		}

		err = actionManager.RunAction(clients.StrPtr("GetLamdaLogs"))

		if err != nil {
			t.Errorf("%v", err)
		}

	})
}
