package rbac

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/rbac-operator/pkg/base"
)

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
