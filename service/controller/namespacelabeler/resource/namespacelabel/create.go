package namespacelabel

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"

	"github.com/giantswarm/rbac-operator/pkg/label"
	"github.com/giantswarm/rbac-operator/service/controller/namespacelabeler/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	namespace, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	changeSet := []struct {
		Src string
		Dst string
	}{
		{
			Src: label.LegacyCluster,
			Dst: label.Cluster,
		},
		{
			Src: label.LegacyCustomer,
			Dst: label.Organization,
		},
	}

	labels := namespace.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}

	needsUpdate := false
	for _, change := range changeSet {
		if _, ok := labels[change.Dst]; ok {
			continue
		}

		needsUpdate = true
		labels[change.Dst] = labels[change.Src]
	}

	if !needsUpdate {
		r.logger.LogCtx(ctx, "level", "debug", "message", "labels are up to date")
		r.logger.LogCtx(ctx, "level", "debug", "message", "cancelling resource")
		return nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "updating labels")
	_, err = r.k8sClient.CoreV1().Namespaces().Update(namespace)
	if err != nil {
		return microerror.Mask(err)
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "updated labels")
	}

	reconciliationcanceledcontext.SetCanceled(true)
	r.logger.LogCtx(ctx, "level", "debug", "message", "object with new labels needs to be reconciled again")
	r.logger.LogCtx(ctx, "level", "debug", "message", "cancelling reconciliation")

	return nil
}
