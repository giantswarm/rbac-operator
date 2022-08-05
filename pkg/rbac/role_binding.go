package rbac

import (
	"context"
	"fmt"
	"reflect"

	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/rbac-operator/pkg/base"
)

// RoleBindingNeedsUpdate RoleBinding needs an update with the list of subjects has changed
func RoleBindingNeedsUpdate(desiredRoleBinding, existingRoleBinding *rbacv1.RoleBinding) bool {
	if len(existingRoleBinding.Subjects) != len(desiredRoleBinding.Subjects) {
		return true
	}

	if !reflect.DeepEqual(desiredRoleBinding.Subjects, existingRoleBinding.Subjects) {
		return true
	}

	return false
}

func CreateOrUpdateRoleBinding(c base.K8sClientWithLogging, ctx context.Context, namespace string, roleBinding *rbacv1.RoleBinding) error {
	existingRoleBinding, err := c.K8sClient().RbacV1().RoleBindings(namespace).Get(ctx, roleBinding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("Creating RoleBinding %#q in namespce %s.", roleBinding.Name, namespace))

		_, err := c.K8sClient().RbacV1().RoleBindings(namespace).Create(ctx, roleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("RoleBinding %#q in namespace %s has been created.", roleBinding.Name, namespace))

	} else if err != nil {
		return microerror.Mask(err)
	} else if RoleBindingNeedsUpdate(roleBinding, existingRoleBinding) {
		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("Updating RoleBinding %#q in namespace %s.", roleBinding.Name, namespace))
		_, err := c.K8sClient().RbacV1().RoleBindings(namespace).Update(ctx, roleBinding, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("RoleBinding %#q in namespace %s has been updated.", roleBinding.Name, namespace))

	}

	return nil
}

func DeleteRoleBinding(c base.K8sClientWithLogging, ctx context.Context, namespace string, roleBinding string) error {
	var err error

	_, err = c.K8sClient().RbacV1().RoleBindings(namespace).Get(ctx, roleBinding, v1.GetOptions{})
	if errors.IsNotFound(err) {
		// nothing to be done
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("Deleting RoleBinding %#q in namespace %s.", roleBinding, namespace))

		err = c.K8sClient().RbacV1().RoleBindings(namespace).Delete(ctx, roleBinding, v1.DeleteOptions{})
		if errors.IsNotFound(err) {
			// nothing to be done
		} else if err != nil {
			return microerror.Mask(err)
		}
		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("RoleBinding %#q in namespace %s has been deleted.", roleBinding, namespace))
	}
	return nil
}
