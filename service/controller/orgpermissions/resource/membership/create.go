package membership

import (
	"context"

	"github.com/giantswarm/microerror"
	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/service/controller/orgpermissions/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	var err error

	roleBinding, err := key.ToRoleBinding(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if !pkgkey.IsOrgNamespace(roleBinding.Namespace) || !isTargetRoleBinding(roleBinding) {
		return nil
	}

	// 1. create role to get this organization
	// 2. create rolebinding with copy of subjects

	return nil
}
