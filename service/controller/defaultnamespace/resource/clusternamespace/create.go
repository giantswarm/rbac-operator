package clusternamespace

import (
	"context"

	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/pkg/project"
	"github.com/giantswarm/rbac-operator/pkg/rbac"
	"github.com/giantswarm/rbac-operator/service/controller/defaultnamespace/key"
)

// EnsureCreated Ensures that ClusterRoles with read and write permissions
// for app resources are created
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	namespace, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if !pkgkey.IsDefaultNamespace(namespace.Name) {
		return nil
	}

	err = r.createReadClusterNamespaceAppsRole(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.createWriteClusterNamespaceAppsRole(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// Ensures the ClusterRole 'read-cluster-apps'.
//
// Purpose if this role is to enable read permissions (get, list, watch)
// for app resources in the cluster namespace
func (r *Resource) createReadClusterNamespaceAppsRole(ctx context.Context) error {
	var err error

	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: pkgkey.ReadClusterNamespaceAppsRole,
			Labels: map[string]string{
				label.ManagedBy:              project.Name(),
				label.DisplayInUserInterface: "true",
			},
			Annotations: map[string]string{
				annotation.Notes: "If referenced within an organization namespace, grants read-only (get, list, watch) permissions to app resources in cluster namespaces belonging to the organization.",
			},
		},
	}

	if err = rbac.CreateOrUpdateClusterRole(r, ctx, role); err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// Ensures the ClusterRole 'write-cluster-apps'.
//
// Purpose if this role is to enable write permissions (get, list, watch, create, update, patch, delete)
// for app resources in the cluster namespace
func (r *Resource) createWriteClusterNamespaceAppsRole(ctx context.Context) error {
	var err error

	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: pkgkey.WriteClusterNamespaceAppsRole,
			Labels: map[string]string{
				label.ManagedBy:              project.Name(),
				label.DisplayInUserInterface: "true",
			},
			Annotations: map[string]string{
				annotation.Notes: "If referenced within an organization namespace, grants read and write permissions to app resources in cluster namespaces belonging to the organization.",
			},
		},
	}

	if err = rbac.CreateOrUpdateClusterRole(r, ctx, role); err != nil {
		return microerror.Mask(err)
	}

	return nil
}
