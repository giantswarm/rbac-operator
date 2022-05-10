package rbac

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/rbac-operator/pkg/base"
)

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
