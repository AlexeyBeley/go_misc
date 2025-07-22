package aws_api

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

type SecretsmanagerAPI struct {
	svc         *secretsmanager.SecretsManager
	profileName *string
}

func SecretsmanagerAPINew(region, profileName *string) *SecretsmanagerAPI {
	if profileName == nil {
		profileNameString := "default"
		profileName = &profileNameString
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            aws.Config{Region: region},
		Profile:           *profileName,
	}))

	lg.InfoF("AWS profile: %s\n", *profileName)
	svc := secretsmanager.New(sess)
	ret := SecretsmanagerAPI{svc: svc, profileName: profileName}
	return &ret
}

// Up to 100 per page
func (api *SecretsmanagerAPI) YieldSecrets(Input *secretsmanager.ListSecretsInput, callbackFilter GenericCallback) ([]*secretsmanager.SecretListEntry, error) {
	var callbackErr error
	ret := []*secretsmanager.SecretListEntry{}

	err := api.svc.ListSecretsPages(Input, func(page *secretsmanager.ListSecretsOutput, notHasNextPage bool) bool {
		for _, secret := range page.SecretList {
			if callbackFilter != nil {
				if callbackErr = callbackFilter(secret); callbackErr != nil {
					return false
				}
			}
			ret = append(ret, secret)
		}
		return true
	})

	if callbackErr != nil {
		return nil, callbackErr
	}
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (api *SecretsmanagerAPI) ProvisionTags(secret *secretsmanager.SecretListEntry, DesiredTags map[string]*string) error {
	missingTags := []*secretsmanager.Tag{}

	currentTags := map[string]*string{}
	for _, currentTag := range secret.Tags {
		currentTags[*currentTag.Key] = currentTag.Value

	}

	for desiredKey, desiredValue := range DesiredTags {
		if currentValue, found := currentTags[desiredKey]; !found || *currentValue != *desiredValue {
			Tag := &secretsmanager.Tag{Key: &desiredKey, Value: desiredValue}
			missingTags = append(missingTags, Tag)
		}
	}

	if len(missingTags) == 0 {
		return nil
	}
	req := secretsmanager.TagResourceInput{SecretId: secret.ARN, Tags: missingTags}
	lg.InfoF("Adding tags: resource: %s, tags: %v, current tags: %v", *secret.ARN, missingTags, currentTags)
	_, err := api.svc.TagResource(&req)
	if err != nil {
		return err
	}
	return err
}
