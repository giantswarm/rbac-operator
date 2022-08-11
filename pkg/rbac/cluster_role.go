package rbac

import (
	"context"
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/rbac-operator/pkg/base"
)

func ClusterRoleNeedsUpdate(desiredClusterRole, existingClusterRole *rbacv1.ClusterRole) bool {
	if len(existingClusterRole.Rules) != len(desiredClusterRole.Rules) {
		return true
	}

	if !reflect.DeepEqual(existingClusterRole, desiredClusterRole) {
		return true
	}

	return false
}

func CreateOrUpdateClusterRole(c base.K8sClientWithLogging, ctx context.Context, clusterRole *rbacv1.ClusterRole) error {
	existingClusterRole, err := c.K8sClient().RbacV1().ClusterRoles().Get(ctx, clusterRole.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating clusterrole %#q", clusterRole.Name))

		_, err := c.K8sClient().RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrole %#q has been created", clusterRole.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else if ClusterRoleNeedsUpdate(clusterRole, existingClusterRole) {
		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating clusterrole %#q", clusterRole.Name))
		_, err := c.K8sClient().RbacV1().ClusterRoles().Update(ctx, clusterRole, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrole %#q has been updated", clusterRole.Name))
	}

	return nil
}

func DeleteClusterRole(c base.K8sClientWithLogging, ctx context.Context, clusterRole string) error {
	var err error

	_, err = c.K8sClient().RbacV1().ClusterRoles().Get(ctx, clusterRole, v1.GetOptions{})
	if errors.IsNotFound(err) {
		// nothing to be done
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("Deleting ClusterRole %s", clusterRole))

		err = c.K8sClient().RbacV1().ClusterRoles().Delete(ctx, clusterRole, v1.DeleteOptions{})
		if errors.IsNotFound(err) {
			// nothing to be done
		} else if err != nil {
			return microerror.Mask(err)
		}
		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("ClusterRole %s has been deleted.", clusterRole))
	}

	return nil
}
