package kub_api

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AccessManager struct {
	KAPI *KubAPI
}

func (accessManager *AccessManager) GenerateJobRunnerRole(RoleName, JobName *string) (*rbacv1.Role, error) {
	roleP := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      *RoleName,
			Namespace: *accessManager.KAPI.Namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{"batch"},
				Resources:     []string{"jobs"},
				Verbs:         []string{"create"},
				ResourceNames: []string{*JobName}, // Limit to the specific Job name
			},
			{
				APIGroups: []string{""}, // Core API group
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list"}, // Needed for Job controller
			},
		},
	}
	return roleP, nil
}

func (accessManager *AccessManager) GenerateJobRunnerServiceAccount(Name *string) (*corev1.ServiceAccount, error) {
	serviceAccountP := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      *Name,
			Namespace: *accessManager.KAPI.Namespace,
		},
	}

	return serviceAccountP, nil
}

func (accessManager *AccessManager) GenerateRoleBinding(RoleBindingName, ServiceAccountName, RoleName *string) (*rbacv1.RoleBinding, error) {
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      *RoleBindingName,
			Namespace: *accessManager.KAPI.Namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      *ServiceAccountName,
				Namespace: *accessManager.KAPI.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     *RoleName,
		},
	}
	return roleBinding, nil
}
