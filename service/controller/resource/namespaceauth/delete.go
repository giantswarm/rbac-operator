package namespaceauth

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/rbac-operator/service/controller/key"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	namespace, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	roles := []role{
		viewAllRole,
		tenantAdminRole,
	}

	for _, role := range roles {

		_, err = r.k8sClient.RbacV1().RoleBindings(namespace.Name).Get(role.name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting %#q role binding", role.name))

			err = r.k8sClient.RbacV1().RoleBindings(namespace.Name).Delete(role.name, &metav1.DeleteOptions{})
			if apierrors.IsNotFound(err) {
				// do nothing
			}
			if err != nil {
				return microerror.Mask(err)
			} else {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("role binding %#q has been deleted", role.name))
			}
		}

		_, err = r.k8sClient.RbacV1().Roles(namespace.Name).Get(role.name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting %#q role", role.name))

			err = r.k8sClient.RbacV1().Roles(namespace.Name).Delete(role.name, &metav1.DeleteOptions{})
			if apierrors.IsNotFound(err) {
				// do nothing
			}
			if err != nil {
				return microerror.Mask(err)
			} else {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("role %#q has been deleted", role.name))
			}
		}
	}

	return nil
}
