package automation

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

	if !pkgkey.IsOrgNamespace(ns.Name) {
		return nil
	}

	clusterRoleBindings := []string{
		pkgkey.WriteSilencesAutomationSAinNSRoleBindingName(ns.Name),
	}

	for _, clusterRoleBinding := range clusterRoleBindings {
		_, err = r.k8sClient.RbacV1().ClusterRoleBindings().Get(ctx, clusterRoleBinding, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			continue
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("deleting %#q clusterrolebinding", clusterRoleBinding))

			err = r.k8sClient.RbacV1().ClusterRoleBindings().Delete(ctx, clusterRoleBinding, metav1.DeleteOptions{})
			if apierrors.IsNotFound(err) {
				continue
			}
			if err != nil {
				return microerror.Mask(err)
			} else {
				r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("rolebinding %#q has been deleted ", clusterRoleBinding))
			}
		}
	}

	serviceAccountName := pkgkey.AutomationServiceAccountName
	_, err = r.k8sClient.CoreV1().ServiceAccounts(ns.Name).Get(ctx, serviceAccountName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		// pass
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("deleting serviceaccount %#q from namespace %s", serviceAccountName, ns.Name))

		err = r.k8sClient.CoreV1().ServiceAccounts(ns.Name).Delete(ctx, serviceAccountName, metav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			// pass
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("serviceaccount %#q has been deleted from namespace %s", serviceAccountName, ns.Name))
		}
	}

	return nil
}
