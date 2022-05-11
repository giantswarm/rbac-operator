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
	err = rbac.DeleteClusterRole(r, ctx, clusterRole.Name)
	if err != nil {
		r.logger.Errorf(ctx, err, "Failed to delete app-operator cluster role: %s", clusterRole.Name)
	}

	var clusterRoleBinding = getAppOperatorCLusterRoleBinding(ns, clusterRole)
	err = rbac.DeleteClusterRoleBinding(r, ctx, clusterRoleBinding.Name)
	if err != nil {
		r.logger.Errorf(ctx, err, "Failed to delete app-operator cluster role binding: %s", clusterRoleBinding.Name)
	}

	var catalogReaderRole = getAppOperatorCatalogReaderRole(ns)
	err = rbac.DeleteRole(r, ctx, catalogReaderRole.Namespace, catalogReaderRole.Name)
	if err != nil {
		r.logger.Errorf(ctx, err, "Failed to delete app-operator catalog reader role: %s", catalogReaderRole.Name)
	}

	var catalogReaderRoleBinding = getAppOperatorCatalogReaderRoleBinding(ns, catalogReaderRole)
	err = rbac.DeleteRoleBinding(r, ctx, catalogReaderRoleBinding.Namespace, catalogReaderRoleBinding.Name)
	if err != nil {
		r.logger.Errorf(ctx, err, "Failed to delete app-operator catalog reader role binding: %s", catalogReaderRoleBinding.Name)
	}

	var ownNamespaceRole = getAppOperatorOwnNamespaceRole(ns)
	err = rbac.DeleteRole(r, ctx, ownNamespaceRole.Namespace, ownNamespaceRole.Name)
	if err != nil {
		r.logger.Errorf(ctx, err, "Failed to delete app-operator own namespace role: %s", ownNamespaceRole.Name)
	}

	var ownNamespaceRoleBinding = getAppOperatorOwnNamespaceRoleBinding(ns, ownNamespaceRole)
	err = rbac.DeleteRoleBinding(r, ctx, ownNamespaceRoleBinding.Namespace, ownNamespaceRoleBinding.Name)
	if err != nil {
		r.logger.Errorf(ctx, err, "Failed to delete app-operator own namespace role biding: %s", ownNamespaceRoleBinding.Name)
	}
	git
	return nil
}
