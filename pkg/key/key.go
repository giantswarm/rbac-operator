package key

import (
	"fmt"
	"strings"
)

const (
	LabelManagedBy = "giantswarm.io/managed-by"
)

const (
	AutomationServiceAccountName      = "automation"
	ClusterAdminClusterRoleName       = "cluster-admin"
	DefaultReadAllPermissionsName     = "read-all"
	DefaultWriteAllPermissionsName    = "write-all"
	DefaultNamespaceName              = "default"
	WriteOrganizationsPermissionsName = "write-organizations"
	WriteFluxResourcesPermissionsName = "write-flux-resources"
	WriteClustersPermissionsName      = "write-clusters"
	WriteNodePoolsPermissionsName     = "write-nodepools"
	WriteClientCertsPermissionsName   = "write-client-certificates"
	WriteSilencesPermissionsName      = "write-silences"
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

func WriteOrganizationsAutomationSARoleBindingName() string {
	return fmt.Sprintf("%s-customer-sa", WriteOrganizationsPermissionsName)
}

func WriteAllGSGroupClusterRoleBindingName() string {
	return fmt.Sprintf("%s-giantswarm-group", DefaultWriteAllPermissionsName)
}

func WriteOrganizationsCustomerGroupClusterRoleBindingName() string {
	return fmt.Sprintf("%s-customer-group", WriteOrganizationsPermissionsName)
}

func WriteFluxResourcesAutomationSARoleBindingName() string {
	return fmt.Sprintf("%s-customer-sa", WriteFluxResourcesPermissionsName)
}

func WriteClustersAutomationSARoleBindingName() string {
	return fmt.Sprintf("%s-customer-sa", WriteClustersPermissionsName)
}

func WriteNodePoolsAutomationSARoleBindingName() string {
	return fmt.Sprintf("%s-customer-sa", WriteNodePoolsPermissionsName)
}

func WriteClientCertsAutomationSARoleBindingName() string {
	return fmt.Sprintf("%s-customer-sa", WriteClientCertsPermissionsName)
}

func WriteSilencesAutomationSARoleBindingName() string {
	return fmt.Sprintf("%s-customer-sa", WriteSilencesPermissionsName)
}
