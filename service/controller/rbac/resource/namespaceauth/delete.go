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

	if !key.HasOrganizationOrCustomerLabel(ns) {
		return nil
	}

	clusterRoles := []string{
		pkgkey.OrganizationReadClusterRoleName(ns.Name),
	}

	for _, clusterRole := range clusterRoles {
		_, err = r.k8sClient.RbacV1().ClusterRoles().Get(ctx, clusterRole, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			continue
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("deleting clusterrole %#q", clusterRole))

			err = r.k8sClient.RbacV1().ClusterRoles().Delete(ctx, clusterRole, metav1.DeleteOptions{})
			if apierrors.IsNotFound(err) {
				continue
			}
			if err != nil {
				return microerror.Mask(err)
			} else {
				r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrole %#q has been deleted", clusterRole))
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
			continue
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("deleting rolebinding %#q from namespace %s", roleBinding, ns.Name))

			err = r.k8sClient.RbacV1().RoleBindings(ns.Name).Delete(ctx, roleBinding, metav1.DeleteOptions{})
			if apierrors.IsNotFound(err) {
				continue
			}
			if err != nil {
				return microerror.Mask(err)
			} else {
				r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("rolebinding %#q has been deleted from namespace %s", roleBinding, ns.Name))
			}
		}
	}

	return nil
}
