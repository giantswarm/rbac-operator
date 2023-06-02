package defaultnamespacetest

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

func ServiceAccountsShouldEqual(t *testing.T, expected []*corev1.ServiceAccount, actual []corev1.ServiceAccount) {
	if len(expected) != len(actual) {
		t.Fatalf("service accounts do not equal: expected length %d, actual %d", len(expected), len(actual))
	}
	for _, expectedItem := range expected {
		hasItem := false
		for _, actualItem := range actual {
			if expectedItem.Name == actualItem.Name || expectedItem.Namespace == actualItem.Namespace {
				hasItem = true
				break
			}
		}
		if !hasItem {
			t.Fatalf("missing service account %v", expectedItem)
		}
	}
}

func ServiceAccountsShouldEqualDeep(t *testing.T, expected []*corev1.ServiceAccount, actual []corev1.ServiceAccount) {
	if len(expected) != len(actual) {
		t.Fatalf("service accounts do not equal: expected length %d, actual %d", len(expected), len(actual))
	}
	for _, expectedItem := range expected {
		for _, actualItem := range actual {
			if expectedItem.Name != actualItem.Name || expectedItem.Namespace != actualItem.Namespace {
				t.Fatalf("expected service accounts %v do not equal actual service accounts %v\n", expected, actual)
			}
			if !((expectedItem.AutomountServiceAccountToken == nil && actualItem.AutomountServiceAccountToken == nil) || (expectedItem.AutomountServiceAccountToken != nil && actualItem.AutomountServiceAccountToken != nil && *expectedItem.AutomountServiceAccountToken == *actualItem.AutomountServiceAccountToken)) {
				t.Fatalf("expected service accounts %v do not equal actual service accounts %v\n", expected, actual)
			}
			if !reflect.DeepEqual(expectedItem.ImagePullSecrets, actualItem.ImagePullSecrets) {
				t.Fatalf("expected service accounts %v do not equal actual service accounts %v\n", expected, actual)
			}
			if !reflect.DeepEqual(expectedItem.Secrets, actualItem.Secrets) {
				t.Fatalf("expected service accounts %v do not equal actual service accounts %v\n", expected, actual)
			}
		}
	}

}

func ClusterRoleBindingsShouldEqual(t *testing.T, expected []*rbacv1.ClusterRoleBinding, actual []rbacv1.ClusterRoleBinding) {
	if len(expected) != len(actual) {
		t.Fatalf("cluster role bindings do not equal: expected length %d, actual %d", len(expected), len(actual))
	}
	for _, expectedItem := range expected {
		hasItem := false
		for _, actualItem := range actual {
			if expectedItem.ObjectMeta.Name == actualItem.ObjectMeta.Name {
				hasItem = true
				SubjectsShouldEqual(t, actualItem.Kind, actualItem.Name, expectedItem.Subjects, actualItem.Subjects)
				break
			}
		}
		if !hasItem {
			t.Fatalf("missing cluster role binding %v", expectedItem)
		}
	}
}

func RoleBindingsShouldEqual(t *testing.T, expected []*rbacv1.RoleBinding, actual []rbacv1.RoleBinding) {
	if len(expected) != len(actual) {
		t.Fatalf("role bindings do not equal: expected length %d, actual %d", len(expected), len(actual))
	}
	for _, expectedItem := range expected {
		hasItem := false
		for _, actualItem := range actual {
			if expectedItem.ObjectMeta.Name == actualItem.ObjectMeta.Name {
				hasItem = true
				SubjectsShouldEqual(t, actualItem.Kind, actualItem.Name, expectedItem.Subjects, actualItem.Subjects)
				break
			}
		}
		if !hasItem {
			t.Fatalf("missing role binding %v", expectedItem)
		}
	}
}

func SubjectsShouldEqual(t *testing.T, kind string, name string, expected []rbacv1.Subject, actual []rbacv1.Subject) {
	if len(expected) != len(actual) {
		t.Fatalf("incorrect number of subjects in %s %s: expected %d, actual %d", kind, name, len(expected), len(actual))
	}
	for _, expectedSubject := range expected {
		hasSubject := false
		for _, actualSubject := range actual {
			if expectedSubject.Kind == actualSubject.Kind && expectedSubject.Name == actualSubject.Name {
				hasSubject = true
				break
			}
		}
		if !hasSubject {
			t.Fatalf("missing subject %v in %s %s", expectedSubject, kind, name)
		}
	}
}

func RolesShouldEqual(t *testing.T, expected []*rbacv1.Role, actual []rbacv1.Role) {
	if len(expected) != len(actual) {
		t.Fatalf("incorrect number of roles: expected %d, actual %d", len(expected), len(actual))
	}
	for _, expectedItem := range expected {
		hasItem := false
		for _, actualItem := range actual {
			if expectedItem.ObjectMeta.Name == actualItem.ObjectMeta.Name && expectedItem.ObjectMeta.Namespace == actualItem.ObjectMeta.Namespace {
				hasItem = true
				RulesShouldEqual(t, actualItem.Kind, actualItem.Name, expectedItem.Rules, actualItem.Rules)
				break
			}
		}
		if !hasItem {
			t.Fatalf("missing role binding %v", expectedItem)
		}
	}
}

func ClusterRolesShouldEqual(t *testing.T, expected []*rbacv1.ClusterRole, actual []rbacv1.ClusterRole) {
	if len(expected) != len(actual) {
		t.Fatalf("incorrect number of cluster roles: expected %d, actual %d", len(expected), len(actual))
	}
	for _, expectedItem := range expected {
		hasItem := false
		for _, actualItem := range actual {
			if expectedItem.ObjectMeta.Name == actualItem.ObjectMeta.Name {
				hasItem = true
				RulesShouldEqual(t, actualItem.Kind, actualItem.Name, expectedItem.Rules, actualItem.Rules)
				break
			}
		}
		if !hasItem {
			t.Fatalf("missing role binding %v", expectedItem)
		}
	}
}

func RulesShouldEqual(t *testing.T, kind string, name string, expected []rbacv1.PolicyRule, actual []rbacv1.PolicyRule) {
	if len(expected) != len(actual) {
		t.Fatalf("incorrect number of rules in %s %s: expected %d, actual %d", kind, name, len(expected), len(actual))
	}
	for _, expectedItem := range expected {
		hasRule := false
		for _, actualItem := range actual {
			if reflect.DeepEqual(expectedItem.APIGroups, actualItem.APIGroups) && reflect.DeepEqual(expectedItem.Resources, actualItem.Resources) {
				hasRule = true
				break
			}
		}
		if !hasRule {
			t.Fatalf("missing rule %v in %s %s", expectedItem, kind, name)
		}
	}
}
