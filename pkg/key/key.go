package key

import (
	"fmt"
	"strings"
)

const (
	AutomationServiceAccountName      = "automation"
	ClusterAdminClusterRoleName       = "cluster-admin"
	DefaultReadAllPermissionsName     = "read-all"
	DefaultWriteAllPermissionsName    = "write-all"
	DefaultNamespaceName              = "default"
	WriteOrganizationsPermissionsName = "write-organizations"
)

func DefaultClusterRolesToDisplayInUI() []string {
	return []string{
		"cluster-admin",
	}
}

func IsOrgNamespace(ns string) bool {
	return strings.HasPrefix(ns, "org-")
}

func OrganizationName(ns string) string {
	return strings.TrimPrefix(ns, "org-")
}

func OrganizationReadClusterRoleName(ns string) string {
	return fmt.Sprintf("organization-%s-read", strings.TrimPrefix(ns, "org-"))
}

func OrganizationReadClusterRoleBindingName(roleBindingName, organization string) string {
	return fmt.Sprintf("%s-organization-%s-read", roleBindingName, organization)
}

func ReadAllCustomerGroupClusterRoleBindingName() string {
	return fmt.Sprintf("%s-customer-group", DefaultReadAllPermissionsName)
}

func ReadAllAutomationSAClusterRoleBindingName() string {
	return fmt.Sprintf("%s-customer-sa", DefaultReadAllPermissionsName)
}

func WriteAllCustomerGroupRoleBindingName() string {
	return fmt.Sprintf("%s-customer-group", DefaultWriteAllPermissionsName)
}

func WriteAllAutomationSARoleBindingName() string {
	return fmt.Sprintf("%s-customer-sa", DefaultWriteAllPermissionsName)
}

func WriteAllGSGroupClusterRoleBindingName() string {
	return fmt.Sprintf("%s-giantswarm-group", DefaultWriteAllPermissionsName)
}

func WriteOrganizationsCustomerGroupClusterRoleBindingName() string {
	return fmt.Sprintf("%s-customer-group", WriteOrganizationsPermissionsName)
}
