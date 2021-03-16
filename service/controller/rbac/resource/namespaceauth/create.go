package namespaceauth

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/microerror"
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

	roleBindingToCustomerGroupName := pkgkey.WriteAllCustomerGroupRoleBindingName()

	writeAllRoleBindingToCustomerGroup := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: roleBindingToCustomerGroupName,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "Group",
				Name: r.writeAllCustomerGroup,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     pkgkey.ClusterAdminClusterRoleName,
		},
	}

	roleBinding := writeAllRoleBindingToCustomerGroup
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

	roleBindingToAutomationSAName := pkgkey.WriteAllAutomationSARoleBindingName()

	writeAllRoleBindingToAutomationSA := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: roleBindingToAutomationSAName,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Subjects: []rbacv1.Subject{
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

	roleBinding = writeAllRoleBindingToAutomationSA
	_, err = r.k8sClient.RbacV1().RoleBindings(ns.Name).Get(ctx, roleBinding.Name, metav1.GetOptions{})
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
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("rolebinding %#q already exists", roleBinding.Name))
	}

	return nil
}

func needsUpdate(desiredRoleBinding, existingRoleBinding *rbacv1.RoleBinding) bool {
	if len(existingRoleBinding.Subjects) < 1 {
		return true
	}

	return desiredRoleBinding.Subjects[0].Name != existingRoleBinding.Subjects[0].Name || desiredRoleBinding.Subjects[0].Namespace != existingRoleBinding.Subjects[0].Namespace || desiredRoleBinding.RoleRef.Name != existingRoleBinding.RoleRef.Name
}
