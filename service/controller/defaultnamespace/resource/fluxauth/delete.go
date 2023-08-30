package fluxauth

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/giantswarm/rbac-operator/api/v1alpha1"
	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/service/controller/rbac/key"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	namespace, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if !pkgkey.IsDefaultNamespace(namespace.Name) {
		return nil
	}

	roleBindingTemplates := []string{
		pkgkey.FluxCRDRoleBindingName,
		pkgkey.FluxReconcilerRoleBindingName,
		pkgkey.WriteAllAutomationSARoleBindingName(),
	}

	for _, name := range roleBindingTemplates {
		roleBindingTemplate := &v1alpha1.RoleBindingTemplate{}
		if err := r.k8sClient.CtrlClient().Get(ctx, types.NamespacedName{Name: name}, roleBindingTemplate); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			} else {
				return microerror.Mask(err)
			}
		}
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("deleting role binding template %s", name))

		if err := r.k8sClient.CtrlClient().Delete(ctx, roleBindingTemplate); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			} else {
				return microerror.Mask(err)
			}
		}
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("role binding template %#q has been deleted", name))
	}

	return nil
}
