package key

import (
	rbacv1 "k8s.io/api/rbac/v1"
	"github.com/giantswarm/microerror"
)

func ToClusterRole(v interface{}) (rbacv1.ClusterRole, error) {
	if v == nil {
		return rbacv1.ClusterRole{}, microerror.Maskf(wrongTypeError, "expected non-nil, got %#v'", v)
	}

	p, ok := v.(*rbacv1.ClusterRole)
	if !ok {
		return rbacv1.ClusterRole{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", p, v)
	}

	c := p.DeepCopy()

	return *c, nil
}
