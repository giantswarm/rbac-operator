package rbac

import (
	"context"
	"fmt"
	"reflect"

	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/rbac-operator/pkg/base"
)

// RoleNeedsUpdate Role needs an update if the rules have changed
func RoleNeedsUpdate(desiredRole, existingRole *rbacv1.Role) bool {
	if len(existingRole.Rules) < 1 {
		return true
	}

	if !reflect.DeepEqual(desiredRole.Rules, existingRole.Rules) {
		return true
	}

	return false
}

func CreateOrUpdateRole(c base.K8sClientWithLogging, ctx context.Context, namespace string, role *rbacv1.Role) error {
	existingRole, err := c.K8sClient().RbacV1().Roles(namespace).Get(ctx, role.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("Creating Role %#q in namespace %s.", role.Name, namespace))

		_, err := c.K8sClient().RbacV1().Roles(namespace).Create(ctx, role, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("Role %#q in namespace %s has been created.", role.Name, namespace))

	} else if err != nil {
		return microerror.Mask(err)
	} else if RoleNeedsUpdate(role, existingRole) {
		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("Updating Role %#q in namespace %s.", role.Name, namespace))
		_, err := c.K8sClient().RbacV1().Roles(namespace).Update(ctx, role, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("Role %#q in namespace %s has been updated.", role.Name, namespace))

	}

	return nil
}

func DeleteRole(c base.K8sClientWithLogging, ctx context.Context, namespace string, role string) error {
	var err error

	_, err = c.K8sClient().RbacV1().Roles(namespace).Get(ctx, role, v1.GetOptions{})
	if errors.IsNotFound(err) {
		// nothing to be done
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("Deleting Role %#q in namespace %s.", role, namespace))

		err = c.K8sClient().RbacV1().Roles(namespace).Delete(ctx, role, v1.DeleteOptions{})
		if errors.IsNotFound(err) {
			// nothing to be done
		} else if err != nil {
			return microerror.Mask(err)
		}
		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("Role %#q in namespace %s has been deleted.", role, namespace))
	}
	return nil
}
