package namespaceauth

import (
	"context"

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

	_, err = r.k8sClient.RbacV1().RoleBindings(namespace.Name).Get(viewAllRole, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		// do nothing
	} else {
		if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", "deleting view role binding")

			err = r.k8sClient.RbacV1().RoleBindings(namespace.Name).Delete(viewAllRole, &metav1.DeleteOptions{})
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", "role binding has been deleted")
		}
	}

	_, err = r.k8sClient.RbacV1().Roles(namespace.Name).Get(viewAllRole, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		// do nothing
	} else {
		if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", "deleting view role")

			err = r.k8sClient.RbacV1().Roles(namespace.Name).Delete(viewAllRole, &metav1.DeleteOptions{})
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", "view role has been deleted")
		}
	}

	return nil
}
