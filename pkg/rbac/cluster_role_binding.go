package rbac

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/rbac-operator/pkg/base"
)

func CreateOrUpdateClusterRoleBinding(c base.K8sClientWithLogging, ctx context.Context, clusterRoleBinding *rbacv1.ClusterRoleBinding) error {
	_, err := c.K8sClient().RbacV1().ClusterRoleBindings().Get(ctx, clusterRoleBinding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating clusterrolebinding %#q", clusterRoleBinding.Name))

		_, err := c.K8sClient().RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrolebinding %#q has been created", clusterRoleBinding.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating clusterrolebinding %#q", clusterRoleBinding.Name))

		_, err := c.K8sClient().RbacV1().ClusterRoleBindings().Update(ctx, clusterRoleBinding, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrolebinding %#q has been updated", clusterRoleBinding.Name))
	}

	return nil
}
