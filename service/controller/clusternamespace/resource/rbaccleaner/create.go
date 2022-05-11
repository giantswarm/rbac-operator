package rbaccleaner

import (
	"context"

	"github.com/giantswarm/rbac-operator/pkg/rbac"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/rbac-operator/service/controller/clusternamespace/key"
)

// EnsureCreated Ensures that ClusterRoleBinding and ClusterRole deployed by legacy
// app-operators to Cluster namespaces are *deleted* when a cluster namespace is created.
// We will be using other roles and cluster roles created by another resource.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	var err error

	cl, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	nameToDelete := key.AppOperatorClusterRoleNameFromNamespace(cl)

	err = rbac.DeleteClusterRole(r, ctx, nameToDelete)
	if err != nil {
		r.logger.Errorf(ctx, err, "Failed to delete app-operator managed cluster role: %s", nameToDelete)
	}

	err = rbac.DeleteClusterRoleBinding(r, ctx, nameToDelete)
	if err != nil {
		r.logger.Errorf(ctx, err, "Failed to delete app-operator managed cluster role binding: %s", nameToDelete)
	}

	return nil
}
