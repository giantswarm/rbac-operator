package rbacappoperator

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/rbac-operator/pkg/rbac"
	"github.com/giantswarm/rbac-operator/service/controller/rbac/key"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	ns, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(
		ctx, "level", "info",
		"message", fmt.Sprintf("Cleaning up app-operator resources for cluster namespace: %s.", ns.Name),
	)

	var clusterRole = getAppOperatorClusterRole(ns)
	_ = rbac.DeleteClusterRole(r, ctx, clusterRole.Name)

	var clusterRoleBinding = getAppOperatorCLusterRoleBinding(ns, clusterRole)
	_ = rbac.DeleteClusterRoleBinding(r, ctx, clusterRoleBinding.Name)

	var catalogReaderRole = getAppOperatorCatalogReaderRole(ns)
	_ = rbac.DeleteRole(r, ctx, catalogReaderRole.Namespace, catalogReaderRole.Name)

	var catalogReaderRoleBinding = getAppOperatorCatalogReaderRoleBinding(ns, catalogReaderRole)
	_ = rbac.DeleteRoleBinding(r, ctx, catalogReaderRoleBinding.Namespace, catalogReaderRoleBinding.Name)

	var ownNamespaceRole = getAppOperatorOwnNamespaceRole(ns)
	_ = rbac.DeleteRole(r, ctx, ownNamespaceRole.Namespace, ownNamespaceRole.Name)

	var ownNamespaceRoleBinding = getAppOperatorOwnNamespaceRoleBinding(ns, ownNamespaceRole)
	_ = rbac.DeleteRoleBinding(r, ctx, ownNamespaceRoleBinding.Namespace, ownNamespaceRoleBinding.Name)

	return nil
}
