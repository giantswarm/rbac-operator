// Package bootstrap ensures certain RBAC resources like ClusterRoles,
// ClusterRoleBindings, and ServiceAccounts. If they exist, they will
// be modified according to spec. If not, they are created.
package bootstrap

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/pkg/project"
)

// Ensures the ClusterRole 'read-cluster-apps'.
//
// Purpose if this role is to enable read permissions (get, list, watch)
// for app resources in the cluster namespace
func (b *Bootstrap) createReadOrgClusterAppsRole(ctx context.Context) error {
	var err error

	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: key.ReadOrgClusterAppsRole,
			Labels: map[string]string{
				label.ManagedBy:              project.Name(),
				label.DisplayInUserInterface: "true",
			},
			Annotations: map[string]string{
				annotation.Notes: "If referenced within an organization namespace, grants read-only (get, list, watch) permissions to app resources in cluster namespaces belonging to the organization.",
			},
		},
	}

	if err = b.createOrUpdateClusterRole(ctx, role); err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// Ensures the ClusterRole 'write-cluster-apps'.
//
// Purpose if this role is to enable write permissions (get, list, watch, create, update, patch, delete)
// for app resources in the cluster namespace
func (b *Bootstrap) createWriteOrgClusterAppsRole(ctx context.Context) error {
	var err error

	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: key.WriteOrgClusterAppsRole,
			Labels: map[string]string{
				label.ManagedBy:              project.Name(),
				label.DisplayInUserInterface: "true",
			},
			Annotations: map[string]string{
				annotation.Notes: "If referenced within an organization namespace, grants read and write permissions to app resources in cluster namespaces belonging to the organization.",
			},
		},
	}

	if err = b.createOrUpdateClusterRole(ctx, role); err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (b *Bootstrap) createOrUpdateClusterRole(ctx context.Context, role *rbacv1.ClusterRole) error {
	var err error
	_, err = b.k8sClient.RbacV1().ClusterRoles().Get(ctx, role.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating clusterrole %#q", role.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoles().Create(ctx, role, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrole %#q has been created", role.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating clusterrole %#q", role.Name))
		_, err := b.k8sClient.RbacV1().ClusterRoles().Update(ctx, role, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrole %#q has been updated", role.Name))
	}

	return nil
}
