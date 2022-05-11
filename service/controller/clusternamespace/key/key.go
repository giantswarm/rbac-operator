package key

import (
	"fmt"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
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

func AppOperatorClusterRoleNameFromNamespace(ns corev1.Namespace) string {
	return fmt.Sprintf("app-operator-%s", ns.Name)
}

func AppOperatorServiceAccountNameFromNamespace(ns corev1.Namespace) string {
	return fmt.Sprintf("app-operator-%s", ns.Name)
}

func AppOperatorRbacOperatorManagedResourceName(ns corev1.Namespace) string {
	return fmt.Sprintf("app-operator-%s-by-rbac-operator", ns.Name)
}
