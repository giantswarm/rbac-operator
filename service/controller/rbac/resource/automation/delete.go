package automation

import (
	"context"
	"fmt"
	"slices"

	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"
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

	// remove this org's automation ServiceAccount from the shared `patch-charts`
	// RoleBinding subjects in the `giantswarm` namespace.
	if err := r.removeAutomationSAFromPatchChartsRoleBinding(ctx, ns.Name); err != nil {
		return microerror.Mask(err)
	}

	clusterRoleBindings := []string{
		pkgkey.WriteSilencesAutomationSAinNSRoleBindingName(ns.Name),
		pkgkey.KamajiDatastoreManagerAutomationSAinNSRoleBindingName(ns.Name),
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

// removeAutomationSAFromPatchChartsRoleBinding drops the automation
// ServiceAccount of the given org namespace from the subjects of the shared
// `patch-charts` RoleBinding in the `giantswarm` namespace. The Role and the
// RoleBinding itself are left in place.
func (r *Resource) removeAutomationSAFromPatchChartsRoleBinding(ctx context.Context, namespace string) error {
	subject := rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      pkgkey.AutomationServiceAccountName,
		Namespace: namespace,
	}

	existing, err := r.k8sClient.RbacV1().RoleBindings(pkgkey.GiantSwarmNamespaceName).Get(ctx, pkgkey.PatchChartsPermissionsName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	if !slices.Contains(existing.Subjects, subject) {
		return nil
	}

	existing.Subjects = slices.DeleteFunc(existing.Subjects, func(s rbacv1.Subject) bool {
		return s == subject
	})

	r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("removing automation SA of namespace %s from rolebinding %#q", namespace, pkgkey.PatchChartsPermissionsName))

	_, err = r.k8sClient.RbacV1().RoleBindings(pkgkey.GiantSwarmNamespaceName).Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
