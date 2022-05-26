package fluxauth

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

	roleBindings := []string{
		pkgkey.FluxCRDRoleBindingName,
		pkgkey.FluxReconcilerRoleBindingName,
		pkgkey.WriteAllAutomationSARoleBindingName(),
	}

	for _, roleBinding := range roleBindings {
		_, err = r.k8sClient.RbacV1().RoleBindings(ns.Name).Get(ctx, roleBinding, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			continue
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("deleting %#q rolebinding from namespace %s", roleBinding, ns.Name))

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

	clusterRoleBindings := []string{
		pkgkey.WriteSilencesAutomationSARoleBindingName(),
	}

	for _, clusterRoleBinding := range clusterRolesBindings {
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
