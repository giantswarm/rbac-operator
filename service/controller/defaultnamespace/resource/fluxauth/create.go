package fluxauth

import (
	"context"
	"fmt"
	"reflect"

	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/giantswarm/rbac-operator/api/v1alpha1"
	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/pkg/project"
	"github.com/giantswarm/rbac-operator/service/controller/rbac/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	namespace, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if !pkgkey.IsDefaultNamespace(namespace.Name) {
		return nil
	}

	// create a RoleBinding granting :
	// - cluster-admin access for "automation" ServiceAccount *in org namespace*
	// - cluster-admin access for "automation" ServiceAccount *in default namespace*
	// cluster-admin permissions are limited in scope to org namespaces

	roleBindingTemplate := &v1alpha1.RoleBindingTemplate{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBindingTemplate",
			APIVersion: "auth.giantswarm.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: pkgkey.WriteAllAutomationSARoleBindingName(),
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Spec: v1alpha1.RoleBindingTemplateSpec{
			Template: v1alpha1.RoleBindingTemplateResource{
				ObjectMeta: metav1.ObjectMeta{
					Name: pkgkey.WriteAllAutomationSARoleBindingName(),
				},
				Subjects: []rbacv1.Subject{
					{
						Kind: "ServiceAccount",
						Name: pkgkey.AutomationServiceAccountName,
					},
					{
						Kind:      "ServiceAccount",
						Name:      pkgkey.AutomationServiceAccountName,
						Namespace: namespace.Name,
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     pkgkey.ClusterAdminClusterRoleName,
				},
			},
		},
	}

	if err := r.createOrUpdateRoleBindingTemplate(ctx, roleBindingTemplate); err != nil {
		return microerror.Mask(err)
	}

	// create a RoleBinding allowing ServiceAccounts in flux-system to access
	// Flux CRs in org namespaces
	roleBindingTemplate = &v1alpha1.RoleBindingTemplate{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBindingTemplate",
			APIVersion: "auth.giantswarm.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: pkgkey.FluxCRDRoleBindingName,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Spec: v1alpha1.RoleBindingTemplateSpec{
			Template: v1alpha1.RoleBindingTemplateResource{
				ObjectMeta: metav1.ObjectMeta{
					Name: pkgkey.FluxCRDRoleBindingName,
				},
				Subjects: []rbacv1.Subject{},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     pkgkey.UpstreamFluxCRDClusterRole,
				},
			},
		},
	}

	for _, serviceAccount := range pkgkey.FluxCrdServiceAccounts {
		roleBindingTemplate.Spec.Template.Subjects = append(roleBindingTemplate.Spec.Template.Subjects,
			rbacv1.Subject{
				Kind:      "ServiceAccount",
				Name:      serviceAccount,
				Namespace: pkgkey.FluxNamespaceName,
			},
		)
	}

	if err := r.createOrUpdateRoleBindingTemplate(ctx, roleBindingTemplate); err != nil {
		return microerror.Mask(err)
	}

	// create a RoleBinding allowing *some* ServiceAccounts in flux-system to
	// reconcile (read, write) Flux CRs in org namespace
	roleBindingTemplate = &v1alpha1.RoleBindingTemplate{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBindingTemplate",
			APIVersion: "auth.giantswarm.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: pkgkey.FluxReconcilerRoleBindingName,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Spec: v1alpha1.RoleBindingTemplateSpec{
			Template: v1alpha1.RoleBindingTemplateResource{
				ObjectMeta: metav1.ObjectMeta{
					Name: pkgkey.FluxReconcilerRoleBindingName,
				},
				Subjects: []rbacv1.Subject{},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     pkgkey.ClusterAdminClusterRoleName,
				},
			},
		},
	}

	for _, serviceAccount := range pkgkey.FluxReconcilerServiceAccounts {
		roleBindingTemplate.Spec.Template.Subjects = append(roleBindingTemplate.Spec.Template.Subjects,
			rbacv1.Subject{
				Kind:      "ServiceAccount",
				Name:      serviceAccount,
				Namespace: pkgkey.FluxNamespaceName,
			},
		)
	}

	if err := r.createOrUpdateRoleBindingTemplate(ctx, roleBindingTemplate); err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) createOrUpdateRoleBindingTemplate(ctx context.Context, roleBindingTemplate *v1alpha1.RoleBindingTemplate) error {
	existingRoleBindingTemplate := &v1alpha1.RoleBindingTemplate{}
	if err := r.k8sClient.CtrlClient().Get(ctx, types.NamespacedName{Name: roleBindingTemplate.Name}, existingRoleBindingTemplate); err != nil {
		if apierrors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating role binding template %#q", roleBindingTemplate.Name))

			if err := r.k8sClient.CtrlClient().Create(ctx, roleBindingTemplate); err != nil {
				if apierrors.IsAlreadyExists(err) {
					// do nothing
				} else {
					return microerror.Mask(err)
				}
			}

			r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("role binding template %#q has been created", roleBindingTemplate.Name))

		} else {
			return microerror.Mask(err)
		}
	} else if needsUpdate(roleBindingTemplate, existingRoleBindingTemplate) {
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating role binding template %#q", roleBindingTemplate.Name))
		existingRoleBindingTemplate.Spec = roleBindingTemplate.Spec
		if err := r.k8sClient.CtrlClient().Update(ctx, existingRoleBindingTemplate); err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("role binding template %#q has been updated", roleBindingTemplate.Name))

	}

	return nil
}

func needsUpdate(desiredRoleBindingTemplate, existingRoleBindingTemplate *v1alpha1.RoleBindingTemplate) bool {
	return !reflect.DeepEqual(desiredRoleBindingTemplate.Spec, existingRoleBindingTemplate.Spec)
}
