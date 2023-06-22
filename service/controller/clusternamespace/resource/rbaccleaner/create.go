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

	r.logger.LogCtx(
		ctx,
		"level",
		"debug",
		"message",
		fmt.Sprintf("running resource rbacappoperator ensurecreated on namespace %s", obj),
	)

	cl, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	namesToDelete := []string{
		key.AppOperatorClusterRoleNameFromNamespace(cl),
		key.AppOperatorChartClusterRoleNameFromNamespace(cl),
	}

	errorOccurred := false
	for _, name := range namesToDelete {
		err = rbac.DeleteClusterRoleBinding(r, ctx, name)
		if err != nil {
			r.logger.Errorf(ctx, err, "Failed to delete app-operator managed cluster role binding: %s", name)
			errorOccurred = true
		}

		err = rbac.DeleteClusterRole(r, ctx, name)
		if err != nil {
			r.logger.Errorf(ctx, err, "Failed to delete app-operator managed cluster role: %s", name)
			errorOccurred = true
		}
	}

	if errorOccurred {
		return microerror.Maskf(executionFailedError, "Failed to clean up one or more original app-operator rbac resources for: %s", cl.Name)
	}

	return nil
}
