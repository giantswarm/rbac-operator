package key

import (
	k8smetadata "github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"

	"github.com/giantswarm/rbac-operator/pkg/label"
)

func ToNamespace(v interface{}) (corev1.Namespace, error) {
	if v == nil {
		return corev1.Namespace{}, microerror.Maskf(wrongTypeError, "expected non-nil, got %#v'", v)
	}

	p, ok := v.(*corev1.Namespace)
	if !ok {
		return corev1.Namespace{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", p, v)
	}

	c := p.DeepCopy()

	return *c, nil
}

func HasOrganizationOrCustomerLabel(namespace corev1.Namespace) bool {
	_, orgLabelPresent := namespace.GetLabels()[k8smetadata.Organization]
	_, customerLabelPresent := namespace.GetLabels()[label.LegacyCustomer]

	return orgLabelPresent || customerLabelPresent
}
