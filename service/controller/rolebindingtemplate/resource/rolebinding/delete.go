package rolebinding

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/rbac-operator/pkg/rbac"
	"github.com/giantswarm/rbac-operator/service/controller/rolebindingtemplate/key"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	template, err := key.ToRoleBindingTemplate(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	roleBindingName := getRoleBindingNameFromTemplate(template)

	for _, ns := range template.Status.Namespaces {
		if err = rbac.DeleteRoleBinding(r, ctx, ns, roleBindingName); err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
