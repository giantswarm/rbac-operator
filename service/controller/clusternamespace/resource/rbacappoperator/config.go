package rbacappoperator

import (
	corev1 "k8s.io/api/core/v1"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/k8smetadata/pkg/annotation"

	"github.com/giantswarm/rbac-operator/pkg/label"
	"github.com/giantswarm/rbac-operator/pkg/project"
	"github.com/giantswarm/rbac-operator/service/controller/clusternamespace/key"
)

func getAppOperatorClusterRole(ns corev1.Namespace) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: key.AppOperatorRbacOperatorManagedResourceName(ns),
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
			Annotations: map[string]string{
				annotation.Notes: "reduced-cluster-roles-for-app-operator",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"application.giantswarm.io"},
				Resources: []string{"apps"},
				Verbs:     []string{"get", "list", "update", "patch", "watch"},
			},
			{
				APIGroups: []string{"application.giantswarm.io"},
				Resources: []string{"apps/status"},
				Verbs:     []string{"create", "patch", "update"},
			},
			{
				APIGroups: []string{"application.giantswarm.io"},
				Resources: []string{"catalogs"},
				Verbs:     []string{"get", "list", "patch", "watch"},
			},
			{
				APIGroups: []string{"application.giantswarm.io"},
				Resources: []string{"appcatalogs"},
				Verbs:     []string{"create", "delete", "get", "list", "patch", "update", "watch"},
			},
			{
				APIGroups: []string{"application.giantswarm.io"},
				Resources: []string{"appcatalogs/status"},
				Verbs:     []string{"create", "patch", "update"},
			},
			{
				APIGroups: []string{"application.giantswarm.io"},
				Resources: []string{"appcatalogentries"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"list", "watch"},
			},
		},
	}
}

func getAppOperatorCLusterRoleBinding(ns corev1.Namespace, clusterRoleRef *rbacv1.ClusterRole) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: key.AppOperatorRbacOperatorManagedResourceName(ns),
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
			Annotations: map[string]string{
				annotation.Notes: "binding-of-reduced-cluster-roles-for-app-operator",
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      key.AppOperatorServiceAccountNameFromNamespace(ns),
				Namespace: ns.Name,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     clusterRoleRef.Kind,
			Name:     clusterRoleRef.Name,
		},
	}
}

func getAppOperatorCatalogReaderRole(ns corev1.Namespace) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: key.AppOperatorRbacOperatorManagedResourceName(ns),
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
			Annotations: map[string]string{
				annotation.Notes: "role-for-app-operator-to-read-catalogs-in-giantswarm-ns",
			},
			Namespace: "giantswarm",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"get", "patch"},
			},
		},
	}
}

func getAppOperatorCatalogReaderRoleBinding(ns corev1.Namespace, roleRef *rbacv1.Role) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: key.AppOperatorRbacOperatorManagedResourceName(ns),
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
			Annotations: map[string]string{
				annotation.Notes: "binding-of-app-operator-giantswarm-catalog-reader-role",
			},
			Namespace: "giantswarm",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      key.AppOperatorServiceAccountNameFromNamespace(ns),
				Namespace: ns.Name,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     roleRef.Kind,
			Name:     roleRef.Name,
		},
	}
}

func getAppOperatorOwnNamespaceRole(ns corev1.Namespace) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: key.AppOperatorRbacOperatorManagedResourceName(ns),
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
			Annotations: map[string]string{
				annotation.Notes: "role-for-app-operator-to-handle-resources-in-its-namespace",
			},
			Namespace: ns.Name,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"namespaces"},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"create", "patch", "update"},
			},
		},
	}
}

func getAppOperatorOwnNamespaceRoleBinding(ns corev1.Namespace, roleRef *rbacv1.Role) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: key.AppOperatorRbacOperatorManagedResourceName(ns),
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
			Annotations: map[string]string{
				annotation.Notes: "binding-of-app-operator-role-to-handle-its-namespace",
			},
			Namespace: ns.Name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      key.AppOperatorServiceAccountNameFromNamespace(ns),
				Namespace: ns.Name,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     roleRef.Kind,
			Name:     roleRef.Name,
		},
	}
}
