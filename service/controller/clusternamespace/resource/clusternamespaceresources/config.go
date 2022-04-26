package clusternamespaceresources

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
)

type rolePair struct {
	roleName        string
	roleBindingName string
	policyRules     []rbacv1.PolicyRule
}

// List of roles and roleBinding pairs that should be ensured as well as the granted permissions
func referencedClusterRoles() []rolePair {
	return []rolePair{
		{
			roleName:        pkgkey.ReadClusterNamespaceAppsRole,
			roleBindingName: pkgkey.ReadClusterNamespaceAppsRoleBinding,
			policyRules:     readClusterAppsRules(),
		},
		{
			roleName:        pkgkey.WriteClusterNamespaceAppsRole,
			roleBindingName: pkgkey.WriteClusterNamespaceAppsRoleBinding,
			policyRules:     writeClusterAppsRules(),
		},
	}
}

func fluxRoleBindings() []rolePair {
	return []rolePair{
		{
			roleName:        pkgkey.ClusterAdminClusterRoleName,
			roleBindingName: pkgkey.FluxReconcilerRoleBindingName,
		},
		{
			roleName:        pkgkey.UpstreamFluxCRDClusterRole,
			roleBindingName: pkgkey.FluxCRDRoleBindingName,
		},
	}
}

func readClusterAppsRules() []rbacv1.PolicyRule {
	return getRules(clusterNamespaceResources(), readAccess())
}
func writeClusterAppsRules() []rbacv1.PolicyRule {
	return getRules(clusterNamespaceResources(), writeAccess())
}

// List of org cluster resources we want to grant access to
func clusterNamespaceResources() []metav1.APIResource {
	return []metav1.APIResource{
		{
			Name:  "apps",
			Group: "application.giantswarm.io",
		},
		{
			Name: "configmaps",
		},
		{
			Name: "secrets",
		},
	}
}

// Actions that are allowed for read access
func readAccess() []string {
	return []string{"get", "list", "watch"}
}

// Actions that are allowed for write access
func writeAccess() []string {
	return []string{"get", "list", "watch", "create", "update", "patch", "delete"}
}
