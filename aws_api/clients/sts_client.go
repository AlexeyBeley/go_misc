package aws_api

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
)

type STSAPI struct {
	svc         *sts.STS
	ProfileName *string
}

func STSAPINew(profileName *string) *STSAPI {
	if profileName == nil {
		profileNameString := "default"
		profileName = &profileNameString
	}
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Profile:           *profileName,
	}))
	lg.InfoF("AWS profile: %s\n", *profileName)
	svc := sts.New(sess)
	ret := STSAPI{svc: svc, ProfileName: profileName}
	return &ret
}

func (api *STSAPI) GetAccount() (*string, error) {
	Input := sts.GetCallerIdentityInput{}
	output, err := api.svc.GetCallerIdentity(&Input)
	return output.Account, err
}
