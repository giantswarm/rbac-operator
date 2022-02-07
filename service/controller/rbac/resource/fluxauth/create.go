package fluxauth

import (
	"context"
	"fmt"

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

const fluxNamespace = "flux-system"

var (
	// upstream Flux ServiceAccounts which need permissions for
	// "*.toolkit.fluxcd.io" resources in Organization namespace
	// see: https://github.com/fluxcd/flux2/blob/main/manifests/rbac/controller.yaml
	crdServiceAccounts = []string{
		"helm-controller",
		"image-automation-controller",
		"image-reflector-controller",
		"kustomize-controller",
		"notification-controller",
		"source-controller",
	}
	// upstream Flux ServiceAccounts which need cluster-admin access to
	// reconcile resources in Organization namespace
	// see: https://github.com/fluxcd/flux2/blob/main/manifests/rbac/reconciler.yaml
	reconcilerServiceAccounts = []string{
		"helm-controller",
		"kustomize-controller",
	}
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	var err error

	ns, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
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
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating serviceaccount %#q", serviceAccount.Name))

			_, err := r.k8sClient.CoreV1().ServiceAccounts(ns.Name).Create(ctx, serviceAccount, metav1.CreateOptions{})
			if apierrors.IsAlreadyExists(err) {
				// do nothing
			} else if err != nil {
				return microerror.Mask(err)
			}
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("serviceaccount %#q has been created", serviceAccount.Name))
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

	for _, serviceAccount := range crdServiceAccounts {
		roleBinding.Subjects = append(roleBinding.Subjects,
			rbacv1.Subject{
				Kind:      "ServiceAccount",
				Name:      serviceAccount,
				Namespace: fluxNamespace,
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

	for _, serviceAccount := range reconcilerServiceAccounts {
		roleBinding.Subjects = append(roleBinding.Subjects,
			rbacv1.Subject{
				Kind:      "ServiceAccount",
				Name:      serviceAccount,
				Namespace: fluxNamespace,
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
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating rolebinding %#q", roleBinding.Name))

		_, err := r.k8sClient.RbacV1().RoleBindings(ns.Name).Create(ctx, roleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("rolebinding %#q has been created", roleBinding.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else if needsUpdate(roleBinding, existingRoleBinding) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating role binding %#q", roleBinding.Name))
		_, err := r.k8sClient.RbacV1().RoleBindings(ns.Name).Update(ctx, roleBinding, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("role binding %#q has been updated", roleBinding.Name))

	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("rolebinding %#q already exists", roleBinding.Name))
	}

	return nil
}

func needsUpdate(desiredRoleBinding, existingRoleBinding *rbacv1.RoleBinding) bool {
	if len(existingRoleBinding.Subjects) < 1 {
		return true
	}

	if desiredRoleBinding.Subjects[0].Name != existingRoleBinding.Subjects[0].Name {
		return true
	}

	if desiredRoleBinding.Subjects[0].Namespace != existingRoleBinding.Subjects[0].Namespace {
		return true
	}

	if desiredRoleBinding.RoleRef.Name != existingRoleBinding.RoleRef.Name {
		return true
	}

	return false
}
