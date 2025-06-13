package kub_api

import (
	"testing"
)

const jobName string = "job-test"
const roleName string = "role-job-runner"
const serviceAccountName string = "service-account-job-runner"
const roleBindingName string = "role-binding-test"

func TestGenerateJobRunnerRole(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {

		realConfig := loadRealConfig()
		_ = realConfig

		api, err := KubAPINew()
		if err != nil {
			t.Errorf("%v", err)
		}
		api.Namespace = realConfig.Namespace

		accessMAnager := AccessManager{KAPI: api}

		roleNameVar := roleName
		jobNameVar := jobName

		role, err := accessMAnager.GenerateJobRunnerRole(&roleNameVar, &jobNameVar)
		if err != nil {
			t.Errorf("%v", err)
		}
		err = api.ProvisionRole(role)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestCreateJobRunnerServiceAccount(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {

		realConfig := loadRealConfig()
		_ = realConfig

		api, err := KubAPINew()
		if err != nil {
			t.Errorf("%v", err)
		}
		api.Namespace = realConfig.Namespace

		accessMAnager := AccessManager{KAPI: api}

		serviceAccountNameVar := serviceAccountName

		svcAccount, err := accessMAnager.GenerateJobRunnerServiceAccount(&serviceAccountNameVar)
		if err != nil {
			t.Errorf("%v", err)
		}
		err = api.ProvisionServiceAccount(svcAccount)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestCreateJobRunnerRoleBinding(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {

		realConfig := loadRealConfig()
		_ = realConfig

		api, err := KubAPINew()
		if err != nil {
			t.Errorf("%v", err)
		}
		api.Namespace = realConfig.Namespace

		accessMAnager := AccessManager{KAPI: api}

		serviceAccountNameVar := serviceAccountName
		roleNameVar := roleName
		roleBindingNameVar := roleBindingName

		binding, err := accessMAnager.GenerateRoleBinding(&roleBindingNameVar, &serviceAccountNameVar, &roleNameVar)
		if err != nil {
			t.Errorf("%v", err)
		}
		err = api.ProvisionRoleBinding(binding)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}
