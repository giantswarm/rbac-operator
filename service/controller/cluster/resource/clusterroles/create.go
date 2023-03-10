package clusterroles

import (
	"context"
	"fmt"

	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/pkg/project"
	"github.com/giantswarm/rbac-operator/pkg/rbac"
	"github.com/giantswarm/rbac-operator/service/controller/cluster/key"
)

// EnsureCreated Ensures that ClusterRoles with permissions
// to read and write all common and custom resources
// are created and properly labeled
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	namespace, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if !pkgkey.IsDefaultNamespace(namespace.Name) {
		return nil
	}

	err = r.createReadAllClusterRole(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.createWriteOrganizationsClusterRole(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.createWriteFluxResourcesClusterRole(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.createWriteClustersClusterRole(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.createWriteNodePoolsClusterRole(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.createWriteClientCertsClusterRole(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.createWriteSilencesClusterRole(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.labelDefaultClusterRoles(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// Ensures the ClusterRole 'read-all'.
//
// Purpose if this role is to enable read permissions (get, list, watch)
// for all resources except ConfigMap and Secret.
func (r *Resource) createReadAllClusterRole(ctx context.Context) error {

	lists, err := r.K8sClient().Discovery().ServerPreferredResources()
	if err != nil {
		panic(err)
	}

	var policyRules []rbacv1.PolicyRule
	{
		for _, list := range lists {
			if len(list.APIResources) == 0 {
				continue
			}
			gv, err := schema.ParseGroupVersion(list.GroupVersion)
			if err != nil {
				continue
			}
			for _, resource := range list.APIResources {
				if len(resource.Verbs) == 0 {
					continue
				}
				if isRestrictedResource(resource.Name) {
					continue
				}

				policyRule := rbacv1.PolicyRule{
					APIGroups: []string{gv.Group},
					Resources: []string{resource.Name},
					Verbs:     []string{"get", "list", "watch"},
				}
				policyRules = append(policyRules, policyRule)
			}
			// ServerPreferredResources explicitely ignores any resource containing a '/'
			// but we require this for enabling pods/logs for customer access to
			// kubernetes pod logging. This is appended as a specific rule instead.
			policyRule := rbacv1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"pods/log"},
				Verbs:     []string{"get", "list"},
			}
			policyRules = append(policyRules, policyRule)
		}
	}

	readOnlyClusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: pkgkey.DefaultReadAllPermissionsName,
			Labels: map[string]string{
				label.ManagedBy:              project.Name(),
				label.DisplayInUserInterface: "true",
			},
			Annotations: map[string]string{
				annotation.Notes: "Grants read-only (get, list, watch) permissions to almost all resource types known on the management cluster, with exception of ConfigMap and Secret.",
			},
		},
		Rules: policyRules,
	}

	return rbac.CreateOrUpdateClusterRole(r, ctx, readOnlyClusterRole)
}

// Ensures the ClusterRole 'write-organizations'.
//
// Purpose of this role is to grant all permissions for the
// organizations.security.giantswarm.io resource.
func (r *Resource) createWriteOrganizationsClusterRole(ctx context.Context) error {

	policyRule := rbacv1.PolicyRule{
		APIGroups: []string{"security.giantswarm.io"},
		Resources: []string{"organizations"},
		Verbs:     []string{"*"},
	}

	orgAdminClusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: pkgkey.WriteOrganizationsPermissionsName,
			Labels: map[string]string{
				label.ManagedBy:              project.Name(),
				label.DisplayInUserInterface: "true",
			},
			Annotations: map[string]string{
				annotation.Notes: "Grants full permissions to Organization CRs.",
			},
		},
		Rules: []rbacv1.PolicyRule{policyRule},
	}

	return rbac.CreateOrUpdateClusterRole(r, ctx, orgAdminClusterRole)
}

// Ensures the ClusterRole 'write-flux-resources'.
//
// Purpose of this role is to grant all permissions for certain
// *.toolkit.fluxcd.io resources.
func (r *Resource) createWriteFluxResourcesClusterRole(ctx context.Context) error {
	policyRule := rbacv1.PolicyRule{
		APIGroups: []string{
			"helm.toolkit.fluxcd.io",
			"image.toolkit.fluxcd.io",
			"kustomizations.kustomize.toolkit.fluxcd.io",
			"notification.toolkit.fluxcd.io",
			"source.toolkit.fluxcd.io",
		},
		Resources: []string{
			"alerts",
			"buckets",
			"gitrepositories",
			"helmcharts",
			"helmreleases",
			"helmrepositories",
			"imagepolicies",
			"imagerepositories",
			"imageupdateautomations",
			"kustomizations",
			"providers",
			"receivers",
		},
		Verbs: []string{"*"},
	}

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: pkgkey.WriteFluxResourcesPermissionsName,
			Labels: map[string]string{
				label.ManagedBy:              project.Name(),
				label.DisplayInUserInterface: "true",
			},
			Annotations: map[string]string{
				annotation.Notes: "Grants full permissions to FluxCD related resource types.",
			},
		},
		Rules: []rbacv1.PolicyRule{policyRule},
	}

	return rbac.CreateOrUpdateClusterRole(r, ctx, clusterRole)
}

// Ensures the ClusterRole 'write-clusters'.
//
// Purpose of this role is to grant all permissions needed for
// creating, modifying, and deleting clusters, not including
// node pools.
func (r *Resource) createWriteClustersClusterRole(ctx context.Context) error {
	policyRule := rbacv1.PolicyRule{
		APIGroups: []string{
			"cluster.x-k8s.io",
			"infrastructure.cluster.x-k8s.io",
			"infrastructure.giantswarm.io",
		},
		Resources: []string{
			"awsclusters",
			"awscontrolplanes",
			"azureclusters",
			"azuremachines",
			"clusters",
			"g8scontrolplanes",
		},
		Verbs: []string{"*"},
	}

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: pkgkey.WriteClustersPermissionsName,
			Labels: map[string]string{
				label.ManagedBy:              project.Name(),
				label.DisplayInUserInterface: "true",
			},
			Annotations: map[string]string{
				annotation.Notes: "Grants full permissions to resources for clusters, excluding node pools.",
			},
		},
		Rules: []rbacv1.PolicyRule{policyRule},
	}

	return rbac.CreateOrUpdateClusterRole(r, ctx, clusterRole)
}

// Ensures the ClusterRole 'write-nodepools'.
//
// Purpose of this role is to grant all permissions needed for
// creating, modifying, and deleting node pools.
func (r *Resource) createWriteNodePoolsClusterRole(ctx context.Context) error {
	policyRule := rbacv1.PolicyRule{
		APIGroups: []string{
			"cluster.x-k8s.io",
			"core.giantswarm.io",
			"exp.cluster.x-k8s.io",
			"infrastructure.cluster.x-k8s.io",
			"infrastructure.giantswarm.io",
		},
		Resources: []string{
			"awsmachinedeployments",
			"azuremachinepools",
			"machinedeployments",
			"machinepools",
			"networkpools",
			"sparks",
		},
		Verbs: []string{"*"},
	}

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: pkgkey.WriteNodePoolsPermissionsName,
			Labels: map[string]string{
				label.ManagedBy:              project.Name(),
				label.DisplayInUserInterface: "true",
			},
			Annotations: map[string]string{
				annotation.Notes: "Grants full permissions on resources representing node pools.",
			},
		},
		Rules: []rbacv1.PolicyRule{policyRule},
	}

	return rbac.CreateOrUpdateClusterRole(r, ctx, clusterRole)
}

// Ensures the ClusterRole 'write-client-certificates'.
//
// Purpose of this role is to grant all permissions needed for
// creating client certificates, which happens via the creation
// of certconfigs.core.giantswarm.io resources.
//
// Note: read access to secrets is not included.
func (r *Resource) createWriteClientCertsClusterRole(ctx context.Context) error {
	policyRule := rbacv1.PolicyRule{
		APIGroups: []string{
			"core.giantswarm.io",
		},
		Resources: []string{
			"certconfigs",
		},
		Verbs: []string{"*"},
	}

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: pkgkey.WriteClientCertsPermissionsName,
			Labels: map[string]string{
				label.ManagedBy:              project.Name(),
				label.DisplayInUserInterface: "true",
			},
			Annotations: map[string]string{
				annotation.Notes: "Grants full permissions on certconfigs.core.giantswarm.io resources.",
			},
		},
		Rules: []rbacv1.PolicyRule{policyRule},
	}

	return rbac.CreateOrUpdateClusterRole(r, ctx, clusterRole)
}

// Ensures the ClusterRole 'write-silences'.
//
// Purpose of this role is to grant all permissions needed for
// handling silences.monitoring.giantswarm.io resources.
func (r *Resource) createWriteSilencesClusterRole(ctx context.Context) error {
	policyRule := rbacv1.PolicyRule{
		APIGroups: []string{
			"monitoring.giantswarm.io",
		},
		Resources: []string{
			"silences",
		},
		Verbs: []string{"*"},
	}

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: pkgkey.WriteSilencesPermissionsName,
			Labels: map[string]string{
				label.ManagedBy:              project.Name(),
				label.DisplayInUserInterface: "true",
			},
			Annotations: map[string]string{
				annotation.Notes: "Grants full permissions for silences.monitoring.giantswarm.io resources.",
			},
		},
		Rules: []rbacv1.PolicyRule{policyRule},
	}

	return rbac.CreateOrUpdateClusterRole(r, ctx, clusterRole)
}

// Ensure labels on the ClusterRole 'cluster-admin':
//
// - 'ui.giantswarm.io/display=true'
// - 'giantswarm.io/managed-by=Kubernetes'
func (r *Resource) labelDefaultClusterRoles(ctx context.Context) error {
	labelsToSet := map[string]string{
		label.DisplayInUserInterface: "true",
		label.ManagedBy:              "Kubernetes",
	}

	clusterRoles := key.DefaultClusterRolesToDisplayInUI()

	for _, clusterRole := range clusterRoles {
		clusterRoleToLabel, err := r.K8sClient().RbacV1().ClusterRoles().Get(ctx, clusterRole, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			r.Logger().LogCtx(ctx, "level", "warn", "message", fmt.Sprintf("clusterrole %#q does not exist", clusterRole))
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		needsUpdate := false
		for k, label := range labelsToSet {
			if existingValue, ok := clusterRoleToLabel.Labels[k]; ok && existingValue == label {
				continue
			}

			r.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("adding label %s to clusterrole %#q", label, clusterRole))

			clusterRoleToLabel.Labels[k] = label
			needsUpdate = true
		}

		if !needsUpdate {
			continue
		}

		_, err = r.K8sClient().RbacV1().ClusterRoles().Update(ctx, clusterRoleToLabel, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		r.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrole %#q has been updated with labels", clusterRole))
	}

	return nil
}

func isRestrictedResource(resource string) bool {
	var restrictedResources = []string{"configmaps", "secrets"}

	for _, restrictedResource := range restrictedResources {
		if resource == restrictedResource {
			return true
		}
	}
	return false
}
