package fluxauth

import (
	"context"
	"fmt"
	"reflect"

	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/pkg/project"
	"github.com/giantswarm/rbac-operator/service/controller/rbac/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	var err error

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

	// create "automation" ServiceAccount in org namespace
	{
		serviceAccount := &corev1.ServiceAccount{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ServiceAccount",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: pkgkey.AutomationServiceAccountName,
				Labels: map[string]string{
					label.ManagedBy: project.Name(),
				},
				Namespace: ns.Name,
			},
		}

		_, err := r.k8sClient.CoreV1().ServiceAccounts(ns.Name).Get(ctx, serviceAccount.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating serviceaccount %#q in namespace %s", serviceAccount.Name, ns.Name))

			_, err := r.k8sClient.CoreV1().ServiceAccounts(ns.Name).Create(ctx, serviceAccount, metav1.CreateOptions{})
			if apierrors.IsAlreadyExists(err) {
				// do nothing
			} else if err != nil {
				return microerror.Mask(err)
			}
			r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("serviceaccount %#q in namespace %s has been created", serviceAccount.Name, ns.Name))
		}
	}

	// create a RoleBinding granting :
	// - cluster-admin access for "automation" ServiceAccount *in this org namespace*
	// - cluster-admin access for "automation" ServiceAccount *in default namespace*
	// cluster-admin permissions are limited in scope to org namespace
	roleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: pkgkey.WriteAllAutomationSARoleBindingName(),
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
			Namespace: ns.Name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      pkgkey.AutomationServiceAccountName,
				Namespace: ns.Name,
			},
			{
				Kind:      "ServiceAccount",
				Name:      pkgkey.AutomationServiceAccountName,
				Namespace: pkgkey.DefaultNamespaceName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     pkgkey.ClusterAdminClusterRoleName,
		},
	}

	if err := r.createOrUpdateRoleBinding(ctx, ns, roleBinding); err != nil {
		return microerror.Mask(err)
	}

	// create a RoleBinding granting :
	// - write-silences access for "automation" ServiceAccount *in this org namespace*
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: pkgkey.WriteSilencesAutomationSARoleBindingName(),
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
			Namespace: ns.Name,
		},
		Subjects: []rbacv1.Subject{
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     pkgkey.WriteSilencesPermissionsName,
		},
	}

	if err := r.createOrUpdateClusterRoleBinding(ctx, ns, clusterRoleBinding); err != nil {
		return microerror.Mask(err)
	}

	// create a RoleBinding allowing ServiceAccounts in flux-system to access
	// Flux CRs in org namespace
	roleBinding = &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: pkgkey.FluxCRDRoleBindingName,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
			Namespace: ns.Name,
		},
		Subjects: []rbacv1.Subject{},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     pkgkey.UpstreamFluxCRDClusterRole,
		},
	}

	for _, serviceAccount := range pkgkey.FluxCrdServiceAccounts {
		roleBinding.Subjects = append(roleBinding.Subjects,
			rbacv1.Subject{
				Kind:      "ServiceAccount",
				Name:      serviceAccount,
				Namespace: pkgkey.FluxNamespaceName,
			},
		)
	}

	if err := r.createOrUpdateRoleBinding(ctx, ns, roleBinding); err != nil {
		return microerror.Mask(err)
	}

	// create a RoleBinding allowing *some* ServiceAccounts in flux-system to
	// reconcile (read, write) Flux CRs in org namespace
	roleBinding = &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: pkgkey.FluxReconcilerRoleBindingName,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
			Namespace: ns.Name,
		},
		Subjects: []rbacv1.Subject{},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     pkgkey.ClusterAdminClusterRoleName,
		},
	}

	for _, serviceAccount := range pkgkey.FluxReconcilerServiceAccounts {
		roleBinding.Subjects = append(roleBinding.Subjects,
			rbacv1.Subject{
				Kind:      "ServiceAccount",
				Name:      serviceAccount,
				Namespace: pkgkey.FluxNamespaceName,
			},
		)
	}

	if err := r.createOrUpdateRoleBinding(ctx, ns, roleBinding); err != nil {
		return microerror.Mask(err)
	}
	return nil
}

func (r *Resource) createOrUpdateRoleBinding(ctx context.Context, ns corev1.Namespace, roleBinding *rbacv1.RoleBinding) error {
	existingRoleBinding, err := r.k8sClient.RbacV1().RoleBindings(ns.Name).Get(ctx, roleBinding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating rolebinding %#q in namespace %s", roleBinding.Name, ns.Name))

		_, err := r.k8sClient.RbacV1().RoleBindings(ns.Name).Create(ctx, roleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("rolebinding %#q in namespace %s has been created", roleBinding.Name, ns.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else if needsUpdate(roleBinding, existingRoleBinding) {
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating role binding %#q in namespace %s", roleBinding.Name, ns.Name))
		_, err := r.k8sClient.RbacV1().RoleBindings(ns.Name).Update(ctx, roleBinding, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("role binding %#q in namespace %s has been updated", roleBinding.Name, ns.Name))

	}

	return nil
}

func (r *Resource) createOrUpdateClusterRoleBinding(ctx context.Context, ns corev1.Namespace, clusterRoleBinding *rbacv1.ClusterRoleBinding) error {
	existingClusterRoleBinding, err := r.k8sClient.RbacV1().ClusterRoleBindings().Get(ctx, clusterRoleBinding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating clusterrolebinding %#q in namespace %s", clusterRoleBinding.Name, ns.Name))

		_, err := r.k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrolebinding %#q in namespace %s has been created", clusterRoleBinding.Name, ns.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else if needsUpdateClusterRoleBinding(clusterRoleBinding, existingClusterRoleBinding) {
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating cluster role binding %#q in namespace %s", clusterRoleBinding.Name, ns.Name))
		clusterRoleBinding.Subjects = append(existingClusterRolebinding.Subjects,
			rbacv1.Subject{
				Kind:      "ServiceAccount",
				Name:      pkgkey.AutomationServiceAccountName,
				Namespace: ns.Name,
			},
		)
		_, err := r.k8sClient.RbacV1().ClusterRoleBindings().Update(ctx, clusterRoleBinding, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("cluster role binding %#q in namespace %s has been updated", clusterRoleBinding.Name, ns.Name))

	}

	return nil
}

func needsUpdate(desiredRoleBinding, existingRoleBinding *rbacv1.RoleBinding) bool {
	if len(existingRoleBinding.Subjects) < 1 {
		return true
	}

	if !reflect.DeepEqual(desiredRoleBinding.Subjects, existingRoleBinding.Subjects) {
		return true
	}

	return false
}

func needsUpdateClusterRoleBinding(desiredClusterRoleBinding, existingClusterRoleBinding *rbacv1.ClusterRoleBinding) bool {
	if len(existingClusterRoleBinding.Subjects) < 1 {
		return true
	}

	if !reflect.DeepEqual(desiredClusterRoleBinding.Subjects, existingClusterRoleBinding.Subjects) {
		return true
	}

	return false
}
