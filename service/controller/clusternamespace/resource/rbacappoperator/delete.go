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

	errorOccurred := false

	var clusterRole = getAppOperatorClusterRole(ns)
	err = rbac.DeleteClusterRole(r, ctx, clusterRole.Name)
	if err != nil {
		r.logger.Errorf(ctx, err, "Failed to delete app-operator cluster role: %s", clusterRole.Name)
		errorOccurred = true
	}

	var clusterRoleBinding = getAppOperatorCLusterRoleBinding(ns, clusterRole)
	err = rbac.DeleteClusterRoleBinding(r, ctx, clusterRoleBinding.Name)
	if err != nil {
		r.logger.Errorf(ctx, err, "Failed to delete app-operator cluster role binding: %s", clusterRoleBinding.Name)
		errorOccurred = true
	}

	var catalogReaderRole = getAppOperatorCatalogReaderRole(ns)
	err = rbac.DeleteRole(r, ctx, catalogReaderRole.Namespace, catalogReaderRole.Name)
	if err != nil {
		r.logger.Errorf(ctx, err, "Failed to delete app-operator catalog reader role: %s", catalogReaderRole.Name)
		errorOccurred = true
	}

	var catalogReaderRoleBinding = getAppOperatorCatalogReaderRoleBinding(ns, catalogReaderRole)
	err = rbac.DeleteRoleBinding(r, ctx, catalogReaderRoleBinding.Namespace, catalogReaderRoleBinding.Name)
	if err != nil {
		r.logger.Errorf(ctx, err, "Failed to delete app-operator catalog reader role binding: %s", catalogReaderRoleBinding.Name)
		errorOccurred = true
	}

	var ownNamespaceRole = getAppOperatorOwnNamespaceRole(ns)
	err = rbac.DeleteRole(r, ctx, ownNamespaceRole.Namespace, ownNamespaceRole.Name)
	if err != nil {
		r.logger.Errorf(ctx, err, "Failed to delete app-operator own namespace role: %s", ownNamespaceRole.Name)
		errorOccurred = true
	}

	var ownNamespaceRoleBinding = getAppOperatorOwnNamespaceRoleBinding(ns, ownNamespaceRole)
	err = rbac.DeleteRoleBinding(r, ctx, ownNamespaceRoleBinding.Namespace, ownNamespaceRoleBinding.Name)
	if err != nil {
		r.logger.Errorf(ctx, err, "Failed to delete app-operator own namespace role biding: %s", ownNamespaceRoleBinding.Name)
		errorOccurred = true
	}

	if errorOccurred {
		return microerror.Maskf(executionFailedError, "Failed to clean up one or more app-operator rbac resources for: %s", ns.Name)
	}

	return nil
}
