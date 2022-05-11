package rbacappoperator

import (
	"context"
	"fmt"

	"github.com/giantswarm/rbac-operator/pkg/rbac"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"

	"github.com/giantswarm/rbac-operator/service/controller/rbac/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cl, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(
		ctx, "level", "info",
		"message", fmt.Sprintf("Reconciling cluster namespace: %s.", cl.Name),
	)

	// Allow working with some generic resources across namespaces
	err = r.CreateClusterRoleAndBinding(ctx, cl)
	if err != nil {
		return err
	}

	// Allow getting catalog configmaps in giantswarm namespace
	err = r.CreateCatalogReaderRoleAndBinding(ctx, cl)
	if err != nil {
		return err
	}

	// Allow working with stuff in its own namespace
	err = r.CreateOwnNamespaceRoleAndBinding(ctx, cl)
	if err != nil {
		return err
	}

	return nil
}

func (r *Resource) CreateClusterRoleAndBinding(ctx context.Context, cl corev1.Namespace) error {
	var clusterRole = getAppOperatorClusterRole(cl.Name)

	if err := rbac.CreateOrUpdateClusterRole(r, ctx, clusterRole); err != nil {
		return microerror.Mask(err)
	}

	clusterRoleBinding := getAppOperatorCLusterRoleBinding(cl.Name, clusterRole.Name)

	if err := rbac.CreateOrUpdateClusterRoleBinding(r, ctx, clusterRoleBinding); err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) CreateCatalogReaderRoleAndBinding(ctx context.Context, cl corev1.Namespace) error {
	var catalogReaderRole = getAppOperatorCatalogReaderRole(cl.Name)

	// TODO Move namespace to keys
	if err := rbac.CreateOrUpdateRole(r, ctx, "giantswarm", catalogReaderRole); err != nil {
		return microerror.Mask(err)
	}

	catalogReaderRoleBinding := getAppOperatorCatalogReaderRoleBinding(cl.Name, fmt.Sprintf("app-operator-%s", cl.Name), catalogReaderRole)

	if err := rbac.CreateOrUpdateRoleBinding(r, ctx, "giantswarm", catalogReaderRoleBinding); err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) CreateOwnNamespaceRoleAndBinding(ctx context.Context, cl corev1.Namespace) error {
	var ownNamespaceRole = getAppOperatorOwnNamespaceRole(cl.Name)

	if err := rbac.CreateOrUpdateRole(r, ctx, cl.Name, ownNamespaceRole); err != nil {
		return microerror.Mask(err)
	}

	ownNamespaceRoleBinding := getAppOperatorOwnNamespaceRoleBinding(cl.Name, ownNamespaceRole)

	if err := rbac.CreateOrUpdateRoleBinding(r, ctx, cl.Name, ownNamespaceRoleBinding); err != nil {
		return microerror.Mask(err)
	}

	return nil
}
