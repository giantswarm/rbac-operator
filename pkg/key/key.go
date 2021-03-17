package key

import "fmt"

const (
	AutomationServiceAccountName   = "automation"
	ClusterAdminClusterRoleName    = "cluster-admin"
	DefaultReadAllPermissionsName  = "read-all"
	DefaultWriteAllPermissionsName = "write-all"
	DefaultNamespaceName           = "default"
)

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
