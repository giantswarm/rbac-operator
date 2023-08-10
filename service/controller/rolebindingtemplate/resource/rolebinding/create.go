package rolebinding

import (
	"context"

	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/giantswarm/rbac-operator/api/v1alpha1"
	"github.com/giantswarm/rbac-operator/pkg/rbac"
	"github.com/giantswarm/rbac-operator/service/controller/rolebindingtemplate/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	template, err := key.ToRoleBindingTemplate(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	namespaces, err := r.getNamespacesFromScope(ctx, template.Spec.Scopes)
	if err != nil {
		return microerror.Mask(err)
	}

	for _, ns := range namespaces {
		roleBinding, err := getRoleBindingFromTemplate(template.Spec.Template, ns)
		if err != nil {
			return microerror.Mask(err)
		}

		if err = rbac.CreateOrUpdateRoleBinding(r, ctx, ns, roleBinding); err != nil {
			return microerror.Mask(err)
		}
	}
	//TODO: update status
	template.Status.Namespaces = namespaces

	return nil
}

func getRoleBindingFromTemplate(template v1alpha1.RoleBindingTemplateResource, namespace string) (*rbacv1.RoleBinding, error) {
	//TODO: get binding from template
	return &template.Spec, nil
}
