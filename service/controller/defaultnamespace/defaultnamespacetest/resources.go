package defaultnamespacetest

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
)

func NewDefaultNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "default",
			Labels: map[string]string{pkgkey.NameLabel: pkgkey.DefaultNamespaceName},
		},
	}
}

func NewClusterAdminRole() *rbacv1.ClusterRole {
	return NewClusterRole(pkgkey.ClusterAdminClusterRoleName, []rbacv1.PolicyRule{
		{
			Verbs:           []string{"*"},
			APIGroups:       []string{"*"},
			Resources:       []string{"*"},
			ResourceNames:   []string{"*"},
			NonResourceURLs: []string{"*"},
		},
	})
}

func NewGroupSubjects(names ...string) []rbacv1.Subject {
	var subjects []rbacv1.Subject
	for _, name := range names {
		subject := rbacv1.Subject{Kind: "Group", Name: name}
		subjects = append(subjects, subject)
	}
	return subjects
}

func NewClusterRoleBinding(name string, subjects []rbacv1.Subject) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Subjects:   subjects,
	}
}

func NewRoleBinding(name, namespace string, subjects []rbacv1.Subject) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Subjects:   subjects,
	}
}

func NewClusterRole(name string, rules []rbacv1.PolicyRule) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{},
		},
		Rules: rules,
	}
}

func NewSingleResourceRule(apiGroup, resource string) rbacv1.PolicyRule {
	return NewRule([]string{apiGroup}, []string{resource})
}

func NewSingletonRulesNoResources() []rbacv1.PolicyRule {
	return NewSingletonRules([]string{}, []string{})
}

func NewSingletonRules(apiGroups []string, resources []string) []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{NewRule(apiGroups, resources)}
}

func NewRule(apiGroups []string, resources []string) rbacv1.PolicyRule {
	return rbacv1.PolicyRule{
		APIGroups: apiGroups,
		Resources: resources,
	}
}

func NewApiResource(group string, version string, kind string, name string, namespaced bool) metav1.APIResource {
	return metav1.APIResource{
		Name:         name,
		SingularName: strings.ToLower(kind),
		Namespaced:   namespaced,
		Group:        group,
		Version:      version,
		Kind:         kind,
		Verbs:        []string{"get", "list", "watch"},
	}
}

func NewRole(name string, namespace string, rules []rbacv1.PolicyRule) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Rules:      rules,
	}
}

func NewServiceAccount(name string, namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
}

func NewSingletonSASubjects(name string, namespace string) []rbacv1.Subject {
	return []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      name,
			Namespace: namespace,
		},
	}
}
