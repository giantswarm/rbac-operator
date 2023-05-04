package test

import (
	"fmt"

	k8smetadata "github.com/giantswarm/k8smetadata/pkg/label"
	security "github.com/giantswarm/organization-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewClusterNamespace(name, organization string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				k8smetadata.Cluster:      name,
				k8smetadata.Organization: organization,
			},
			Name: name,
		},
	}
}

func NewGenericNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func NewOrganization(name string) *security.Organization {
	return &security.Organization{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Status: security.OrganizationStatus{
			Namespace: fmt.Sprintf("org-%s", name),
		},
	}
}

func NewOrgNamespace(orgName string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				k8smetadata.ManagedBy:    "organization-operator",
				k8smetadata.Organization: orgName,
			},
			Name: fmt.Sprintf("org-%s", orgName),
		},
	}
}

func NewRoleBinding(name, namespace string, roleRef map[string]string, subjects []rbacv1.Subject) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				k8smetadata.ManagedBy: "rbac-operator",
			},
			Name:      name,
			Namespace: namespace,
		},
		Subjects: subjects,
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     roleRef["kind"],
			Name:     roleRef["name"],
		},
	}
}

func NewClusterRole(name string, rules ...rbacv1.PolicyRule) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{},
			Name:   name,
		},
		Rules: rules,
	}
}

func NewPolicyRule(verbs, resources, resourceNames, apiGroups []string) *rbacv1.PolicyRule {
	return &rbacv1.PolicyRule{
		Verbs:         verbs,
		Resources:     resources,
		ResourceNames: resourceNames,
		APIGroups:     apiGroups,
	}
}
