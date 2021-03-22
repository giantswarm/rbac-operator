package namespaceauth

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"

	"github.com/giantswarm/rbac-operator/service/controller/rbac/key"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	ns, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	clusterRoles := []string{
		pkgkey.OrganizationReadClusterRoleName(ns.Name),
	}

	for _, clusterRole := range clusterRoles {

		_, err = r.k8sClient.RbacV1().ClusterRoles().Get(ctx, clusterRole, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting %#q clusterrole", clusterRole))

			err = r.k8sClient.RbacV1().ClusterRoles().Delete(ctx, clusterRole, metav1.DeleteOptions{})
			if apierrors.IsNotFound(err) {
				// do nothing
			}
			if err != nil {
				return microerror.Mask(err)
			} else {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrole %#q has been deleted", clusterRole))
			}
		}
	}

	roleBindings := []string{
		pkgkey.WriteAllCustomerGroupRoleBindingName(),
		pkgkey.WriteAllAutomationSARoleBindingName(),
	}

	for _, roleBinding := range roleBindings {

		_, err = r.k8sClient.RbacV1().RoleBindings(ns.Name).Get(ctx, roleBinding, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting %#q role binding", roleBinding))

			err = r.k8sClient.RbacV1().RoleBindings(ns.Name).Delete(ctx, roleBinding, metav1.DeleteOptions{})
			if apierrors.IsNotFound(err) {
				// do nothing
			}
			if err != nil {
				return microerror.Mask(err)
			} else {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("role binding %#q has been deleted", roleBinding))
			}
		}
	}

	return nil
}
