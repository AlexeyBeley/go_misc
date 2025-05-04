package aws_api

import (
	"testing"
)

func TestProvisionIamCloudwatchWriterRole(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		assumeDocument := "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":{\"Service\":\"vpc-flow-logs.amazonaws.com\"},\"Action\":\"sts:AssumeRole\",\"Condition\":{}}]}"

		dir := "/opt/aws_api_go/IamDataDirPath"
		api := IAMAPINew(nil, &dir)
		region := "us-west-2"
		roleName := "role-test"
		path := "/test/"
		_, err := api.ProvisionIamCloudwatchWriterRole(&region, &roleName, &assumeDocument, &path)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestUpdateRoleInfo(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {

		dir := "/opt/aws_api_go/IamDataDirPath"
		api := IAMAPINew(nil, &dir)

		roleName := "none-existing-role"
		path := "/test/"
		role := Role{Name: &roleName, Path: &path}
		ret, err := api.UpdateRoleInfo(&role)
		if err != nil {
			t.Errorf("%v", err)
		}
		if ret {
			t.Errorf("expected false, received %v", ret)
		}
	})
}
