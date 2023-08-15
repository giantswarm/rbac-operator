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

	namespaces, err := r.getNamespacesFromScope(ctx, template.Spec.Scopes)
	if err != nil {
		return microerror.Mask(err)
	}
	roleBindingName := template.Spec.Template.Spec.Name

	for _, ns := range namespaces {
		if err = rbac.DeleteRoleBinding(r, ctx, ns, roleBindingName); err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
