package aws_api

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
)

type IAMAPI struct {
	svc *iam.IAM
}

func IAMAPINew(profileName *string) *IAMAPI {
	if profileName == nil {
		profileNameString := "default"
		profileName = &profileNameString
	}
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Profile:           *profileName,
	}))
	lg.Infof("AWS profile: %s\n", *profileName)
	svc := iam.New(sess)
	ret := IAMAPI{svc: svc}
	return &ret
}

func (api *IAMAPI) ProvisionIamRole(policy map[string]any, roleName *string) (*iam.CreateRoleOutput, error) {
	Input := iam.CreateRoleInput{RoleName: roleName}
	output, err := api.svc.CreateRole(&Input)
	return output, err
}
