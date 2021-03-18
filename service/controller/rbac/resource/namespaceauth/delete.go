package namespaceauth

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/rbac-operator/service/controller/rbac/key"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	ns, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	roles := []string{
		pkgkey.WriteAllCustomerGroupRoleBindingName(),
		pkgkey.WriteAllAutomationSARoleBindingName(),
	}

	for _, role := range roles {

		_, err = r.k8sClient.RbacV1().RoleBindings(ns.Name).Get(ctx, role, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting %#q role binding", role))

			err = r.k8sClient.RbacV1().RoleBindings(ns.Name).Delete(ctx, role, metav1.DeleteOptions{})
			if apierrors.IsNotFound(err) {
				// do nothing
			}
			if err != nil {
				return microerror.Mask(err)
			} else {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("role binding %#q has been deleted", role))
			}
		}

		_, err = r.k8sClient.RbacV1().Roles(ns.Name).Get(ctx, role, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting %#q role", role))

			err = r.k8sClient.RbacV1().Roles(ns.Name).Delete(ctx, role, metav1.DeleteOptions{})
			if apierrors.IsNotFound(err) {
				// do nothing
			}
			if err != nil {
				return microerror.Mask(err)
			} else {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("role %#q has been deleted", role))
			}
		}
	}

	return nil
}
