package namespacelabel

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/rbac-operator/pkg/label"
	"github.com/giantswarm/rbac-operator/service/controller/namespacelabeler/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	namespace, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	namespace.ObjectMeta.Labels[label.Cluster] = namespace.ObjectMeta.Labels[label.LegacyCluster]
	namespace.ObjectMeta.Labels[label.Organization] = namespace.ObjectMeta.Labels[label.LegacyCustomer]

	r.logger.LogCtx(ctx, "level", "debug", "message", "applying new labels to namespace")
	_, err = r.k8sClient.CoreV1().Namespaces().Update(namespace)
	if err != nil {
		return microerror.Mask(err)
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "new labels has been applied")
	}

	return nil
}
