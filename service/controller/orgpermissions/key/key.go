package key

import (
	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"
)

// Casts the given value to a rbacv1.RoleBinding.
func ToRoleBinding(v interface{}) (rbacv1.RoleBinding, error) {
	if v == nil {
		return rbacv1.RoleBinding{}, microerror.Maskf(wrongTypeError, "expected non-nil, got %#v'", v)
	}

	p, ok := v.(*rbacv1.RoleBinding)
	if !ok {
		return rbacv1.RoleBinding{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", p, v)
	}

	c := p.DeepCopy()

	return *c, nil
}
