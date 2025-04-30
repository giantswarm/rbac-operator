package rbac

import (
	"context"
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/rbac-operator/pkg/base"
)

// ClusterRoleBindingNeedsUpdate ClusterRoleBinding needs an update with the list of subjects has changed
func ClusterRoleBindingNeedsUpdate(desiredRoleBinding, existingRoleBinding *rbacv1.ClusterRoleBinding) bool {
	if len(existingRoleBinding.Subjects) != len(desiredRoleBinding.Subjects) {
		return true
	}

	if !reflect.DeepEqual(desiredRoleBinding.Subjects, existingRoleBinding.Subjects) {
		return true
	}

	return false
}

func CreateOrUpdateClusterRoleBinding(c base.K8sClientWithLogging, ctx context.Context, clusterRoleBinding *rbacv1.ClusterRoleBinding) error {
	existingClusterRoleBinding, err := c.K8sClient().RbacV1().ClusterRoleBindings().Get(ctx, clusterRoleBinding.Name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating clusterrolebinding %#q", clusterRoleBinding.Name))

		_, err := c.K8sClient().RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
		if errors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrolebinding %#q has been created", clusterRoleBinding.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else if ClusterRoleBindingNeedsUpdate(clusterRoleBinding, existingClusterRoleBinding) {
		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating clusterrolebinding %#q", clusterRoleBinding.Name))

		_, err := c.K8sClient().RbacV1().ClusterRoleBindings().Update(ctx, clusterRoleBinding, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrolebinding %#q has been updated", clusterRoleBinding.Name))
	}

	return nil
}

func DeleteClusterRoleBinding(c base.K8sClientWithLogging, ctx context.Context, clusterRoleBinding string) error {
	var err error

	_, err = c.K8sClient().RbacV1().ClusterRoleBindings().Get(ctx, clusterRoleBinding, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		// nothing to be done
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("Deleting ClusterRoleBinding %s", clusterRoleBinding))

		err = c.K8sClient().RbacV1().ClusterRoleBindings().Delete(ctx, clusterRoleBinding, metav1.DeleteOptions{})
		if errors.IsNotFound(err) {
			// nothing to be done
		} else if err != nil {
			return microerror.Mask(err)
		}
		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("ClusterRoleBinding %s has been deleted.", clusterRoleBinding))
	}

	return nil
}
