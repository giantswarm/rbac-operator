package namespacelabel

import (
	"context"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"

	"github.com/giantswarm/rbac-operator/pkg/label"
	"github.com/giantswarm/rbac-operator/service/controller/namespacelabeler/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	namespace, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if needsUpdate(namespace) {
		namespace.ObjectMeta.Labels[label.Cluster] = ns.ObjectMeta.Labels[label.LegacyCluster]
		namespace.ObjectMeta.Labels[label.Organization] = ns.ObjectMeta.Labels[label.LegacyCustomer]

		r.logger.LogCtx(ctx, "level", "debug", "message", "applying new labels to namespace")
		_, err = r.k8sClient.CoreV1().Namespaces().Update(namespace)
		if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", "new labels has been applied")
		}
	}

	return nil
}

func needsUpdate(ns *corev1.Namespace) bool {
	_, clusterOK := ns.ObjectMeta.Labels[label.Cluster]
	_, organizationOK := ns.ObjectMeta.Labels[label.Organization]

	return !(clusterOK && organizationOK)
}
