package clusternamespaceresources

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/rbac-operator/pkg/rbac"
	"github.com/giantswarm/rbac-operator/service/controller/clusternamespace/key"
)

// EnsureDeleted Ensures that when a cluster is deleted, roles and roleBindings for cluster resource access are deleted as well
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	var err error

	cl, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	// Delete RoleBindings in org Cluster namespace
	for _, referencedRole := range referencedClusterRoles() {
		err = rbac.DeleteRoleBinding(r, ctx, cl.Name, referencedRole.roleBindingName)
		if err != nil {
			return microerror.Mask(err)
		}
		err = rbac.DeleteRole(r, ctx, cl.Name, referencedRole.roleName)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	err = rbac.DeleteRoleBinding(r, ctx, cl.Name, fluxCRDRolePair.roleBindingName)
	if err != nil {
		return microerror.Mask(err)
	}

	err = rbac.DeleteRoleBinding(r, ctx, cl.Name, fluxNSRolePair.roleBindingName)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
