package membership

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/service/controller/orgpermissions/key"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	var err error

	roleBinding, err := key.ToRoleBinding(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if !pkgkey.IsOrgNamespace(roleBinding.Namespace) || !isTargetRoleBinding(roleBinding) {
		return nil
	}

	roleBindingsToDelete := []string{
		pkgkey.OrganizationReadRoleBindingName(roleBinding.Name),
	}

	for _, rb := range roleBindingsToDelete {

		_, err = r.k8sClient.RbacV1().RoleBindings(roleBinding.Namespace).Get(ctx, rb, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting %#q role binding", rb))

			err = r.k8sClient.RbacV1().RoleBindings(roleBinding.Namespace).Delete(ctx, rb, metav1.DeleteOptions{})
			if apierrors.IsNotFound(err) {
				// do nothing
			}
			if err != nil {
				return microerror.Mask(err)
			} else {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("role binding %#q has been deleted", rb))
			}
		}
	}

	return nil
}
