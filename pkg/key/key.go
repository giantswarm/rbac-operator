package key

import (
	"fmt"
	"strings"

	"github.com/giantswarm/k8smetadata/pkg/label"

	"github.com/giantswarm/rbac-operator/pkg/annotation"
	rbacLabel "github.com/giantswarm/rbac-operator/pkg/label"
)

const (
	AutomationServiceAccountName         = "automation"
	ClusterAdminClusterRoleName          = "cluster-admin"
	DefaultReadAllPermissionsName        = "read-all"
	DefaultWriteAllPermissionsName       = "write-all"
	DefaultNamespaceName                 = "default"
	NameLabel                            = "kubernetes.io/metadata.name"
	FluxCRDRoleBindingName               = "flux-crd-controller"
	FluxNamespaceName                    = "flux-system"
	FluxReconcilerRoleBindingName        = "flux-namespace-reconciler"
	ReadClusterNamespaceAppsRoleBinding  = "read-in-cluster-ns"
	ReadClusterNamespaceAppsRole         = "read-in-cluster-ns"
	ReadDefaultCatalogsRole              = "read-default-catalogs"
	ReadReleasesRole                     = "read-releases"
	UpstreamFluxCRDClusterRole           = "crd-controller"
	WriteClusterNamespaceAppsRoleBinding = "write-in-cluster-ns"
	WriteClusterNamespaceAppsRole        = "write-in-cluster-ns"
	WriteOrganizationsPermissionsName    = "write-organizations"
	WriteFluxResourcesPermissionsName    = "write-flux-resources"
	WriteClustersPermissionsName         = "write-clusters"
	WriteNodePoolsPermissionsName        = "write-nodepools"
	WriteClientCertsPermissionsName      = "write-client-certificates"
	WriteSilencesPermissionsName         = "write-silences"
)

var (
	// FluxCrdServiceAccounts Upstream Flux ServiceAccounts which need permissions for
	// "*.toolkit.fluxcd.io" resources in Organization namespace
	// see: https://github.com/fluxcd/flux2/blob/main/manifests/rbac/controller.yaml
	FluxCrdServiceAccounts = []string{
		"helm-controller",
		"image-automation-controller",
		"image-reflector-controller",
		"kustomize-controller",
		"notification-controller",
		"source-controller",
	}
	// FluxReconcilerServiceAccounts Upstream Flux ServiceAccounts which need cluster-admin access to
	// reconcile resources in Organization namespace
	// see: https://github.com/fluxcd/flux2/blob/main/manifests/rbac/reconciler.yaml
	FluxReconcilerServiceAccounts = []string{
		"helm-controller",
		"kustomize-controller",
	}
)

func IsDefaultNamespace(ns string) bool {
	return ns == DefaultNamespaceName
}

func IsOrgNamespace(ns string) bool {
	return strings.HasPrefix(ns, "org-")
}

func Organization(getter rbacLabel.LabelsGetter) string {
	return getter.GetLabels()[label.Organization]
}

func GetLegacyOrganization(getter annotation.AnnotationsGetter) string {
	annotations := getter.GetAnnotations()
	if annotations == nil {
		return ""
	}

	return annotations[annotation.LegacyOrganization]
}

func OrganizationName(ns string) string {
	return strings.TrimPrefix(ns, "org-")
}

func OrganizationReadClusterRoleName(ns string) string {
	return fmt.Sprintf("organization-%s-read", strings.TrimPrefix(ns, "org-"))
}

func OrganizationReadDefaultCatalogsRoleBindingName(organization string) string {
	return fmt.Sprintf("default-catalogs-organization-%s-read", organization)
}

func OrganizationReadClusterNamespaceRoleBindingName(organization string) string {
	return fmt.Sprintf("cluster-ns-organization-%s-read", organization)
}

func OrganizationReadReleasesClusterRoleBindingName(organization string) string {
	return fmt.Sprintf("releases-organization-%s-read", organization)
}

func OrganizationReadOrganizationClusterRoleBindingName(organization string) string {
	return fmt.Sprintf("organization-organization-%s-read", organization)
}

func OrganizationWriteClusterNamespaceRoleBindingName(organization string) string {
	return fmt.Sprintf("cluster-ns-organization-%s-write", organization)
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

func WriteSilencesAutomationSAinNSRoleBindingName(namespace string) string {
	return fmt.Sprintf("%s-customer-sa-ns-%s", WriteSilencesPermissionsName, namespace)
}
