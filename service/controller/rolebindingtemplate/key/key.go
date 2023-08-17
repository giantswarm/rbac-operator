package key

import (
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/rbac-operator/api/v1alpha1"
)

func ToRoleBindingTemplate(v interface{}) (v1alpha1.RoleBindingTemplate, error) {
	if v == nil {
		return v1alpha1.RoleBindingTemplate{}, microerror.Maskf(wrongTypeError, "expected non-nil, got %#v'", v)
	}

	p, ok := v.(*v1alpha1.RoleBindingTemplate)
	if !ok {
		return v1alpha1.RoleBindingTemplate{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", p, v)
	}

	c := p.DeepCopy()

	return *c, nil
}
