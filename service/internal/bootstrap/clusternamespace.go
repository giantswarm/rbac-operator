// Package bootstrap ensures certain RBAC resources like ClusterRoles,
// ClusterRoleBindings, and ServiceAccounts. If they exist, they will
// be modified according to spec. If not, they are created.
package bootstrap

import (
	"context"

	"github.com/giantswarm/rbac-operator/pkg/rbac"

	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/pkg/project"
)

// Ensures the ClusterRole 'read-cluster-apps'.
//
// Purpose if this role is to enable read permissions (get, list, watch)
// for app resources in the cluster namespace
func (b *Bootstrap) createReadClusterNamespaceAppsRole(ctx context.Context) error {
	var err error

	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: key.ReadClusterNamespaceAppsRole,
			Labels: map[string]string{
				label.ManagedBy:              project.Name(),
				label.DisplayInUserInterface: "true",
			},
			Annotations: map[string]string{
				annotation.Notes: "If referenced within an organization namespace, grants read-only (get, list, watch) permissions to app resources in cluster namespaces belonging to the organization.",
			},
		},
	}

	if err = rbac.CreateOrUpdateClusterRole(b, ctx, role); err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// Ensures the ClusterRole 'write-cluster-apps'.
//
// Purpose if this role is to enable write permissions (get, list, watch, create, update, patch, delete)
// for app resources in the cluster namespace
func (b *Bootstrap) createWriteClusterNamespaceAppsRole(ctx context.Context) error {
	var err error

	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: key.WriteClusterNamespaceAppsRole,
			Labels: map[string]string{
				label.ManagedBy:              project.Name(),
				label.DisplayInUserInterface: "true",
			},
			Annotations: map[string]string{
				annotation.Notes: "If referenced within an organization namespace, grants read and write permissions to app resources in cluster namespaces belonging to the organization.",
			},
		},
	}

	if err = rbac.CreateOrUpdateClusterRole(b, ctx, role); err != nil {
		return microerror.Mask(err)
	}

	return nil
}
