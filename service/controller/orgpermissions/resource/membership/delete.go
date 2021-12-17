package membership

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/service/controller/orgpermissions/key"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	var err error

	roleBinding, err := key.ToRoleBinding(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if !pkgkey.IsOrgNamespace(roleBinding.Namespace) || !isTargetRoleBinding(roleBinding) {
		return nil
	}

	orgName := pkgkey.OrganizationName(roleBinding.Namespace)

	clusterRoleBindingsToDelete := []string{
		pkgkey.OrganizationReadClusterRoleBindingName(roleBinding.Name, orgName),
	}

	for _, crb := range clusterRoleBindingsToDelete {
		_, err = r.k8sClient.RbacV1().ClusterRoleBindings().Get(ctx, crb, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			continue
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting %#q clusterrolebinding", crb))

			err = r.k8sClient.RbacV1().ClusterRoleBindings().Delete(ctx, crb, metav1.DeleteOptions{})
			if apierrors.IsNotFound(err) {
				continue
			}
			if err != nil {
				return microerror.Mask(err)
			} else {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrolebinding %#q has been deleted", crb))
			}
		}
	}

	return nil
}
