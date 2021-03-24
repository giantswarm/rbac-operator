package bootstrap

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/pkg/project"
)

func (b *Bootstrap) createAutomationServiceAccount(ctx context.Context) error {

	automationSA := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: key.AutomationServiceAccountName,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
	}

	_, err := b.k8sClient.CoreV1().ServiceAccounts(key.DefaultNamespaceName).Get(ctx, automationSA.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating service account %#q", automationSA.Name))

		_, err := b.k8sClient.CoreV1().ServiceAccounts(key.DefaultNamespaceName).Create(ctx, automationSA, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("service account %#q has been created", automationSA.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("service account %#q already exists", automationSA.Name))
	}

	return nil
}

func (b *Bootstrap) createReadAllClusterRole(ctx context.Context) error {

	lists, err := b.k8sClient.Discovery().ServerPreferredResources()
	if err != nil {
		panic(err)
	}

	var policyRules []rbacv1.PolicyRule
	{
		for _, list := range lists {
			if len(list.APIResources) == 0 {
				continue
			}
			gv, err := schema.ParseGroupVersion(list.GroupVersion)
			if err != nil {
				continue
			}
			for _, resource := range list.APIResources {
				if len(resource.Verbs) == 0 {
					continue
				}
				if isRestrictedResource(resource.Name) {
					continue
				}

				policyRule := rbacv1.PolicyRule{
					APIGroups: []string{gv.Group},
					Resources: []string{resource.Name},
					Verbs:     []string{"get", "list", "watch"},
				}
				policyRules = append(policyRules, policyRule)
			}
		}
	}

	readOnlyClusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: key.DefaultReadAllPermissionsName,
			Labels: map[string]string{
				label.ManagedBy:              project.Name(),
				label.DisplayInUserInterface: "true",
			},
		},
		Rules: policyRules,
	}

	_, err = b.k8sClient.RbacV1().ClusterRoles().Get(ctx, readOnlyClusterRole.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating clusterrole %#q", readOnlyClusterRole.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoles().Create(ctx, readOnlyClusterRole, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrole %#q has been created", readOnlyClusterRole.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating clusterrole binding %#q", readOnlyClusterRole.Name))
		_, err := b.k8sClient.RbacV1().ClusterRoles().Update(ctx, readOnlyClusterRole, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrole %#q has been updated", readOnlyClusterRole.Name))
	}

	return nil
}

// Grant customer admin read access to everything except configmap/secrets.
func (b *Bootstrap) createReadAllClusterRoleBindingToCustomerGroup(ctx context.Context) error {
	clusterRoleBindingName := key.ReadAllCustomerGroupClusterRoleBindingName()

	readAllClusterRoleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleBindingName,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "Group",
				Name: b.customerAdminGroup,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     key.DefaultReadAllPermissionsName,
		},
	}

	_, err := b.k8sClient.RbacV1().ClusterRoleBindings().Get(ctx, readAllClusterRoleBinding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating clusterrolebinding %#q", readAllClusterRoleBinding.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, readAllClusterRoleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrolebinding %#q has been created", readAllClusterRoleBinding.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrolebinding %#q already exists", readAllClusterRoleBinding.Name))
	}

	return nil
}

// Grant automation service account read access to everything except configmap/secrets.
func (b *Bootstrap) createReadAllClusterRoleBindingToAutomationSA(ctx context.Context) error {
	clusterRoleBindingName := key.ReadAllAutomationSAClusterRoleBindingName()

	readAllClusterRoleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleBindingName,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      key.AutomationServiceAccountName,
				Namespace: key.DefaultNamespaceName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     key.DefaultReadAllPermissionsName,
		},
	}

	_, err := b.k8sClient.RbacV1().ClusterRoleBindings().Get(ctx, readAllClusterRoleBinding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating clusterrolebinding %#q", readAllClusterRoleBinding.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, readAllClusterRoleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrolebinding %#q has been created", readAllClusterRoleBinding.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrolebinding %#q already exists", readAllClusterRoleBinding.Name))
	}

	return nil
}

// Grant cluster-admin access to giantswarm admin group.
func (b *Bootstrap) createWriteAllClusterRoleBindingToGSGroup(ctx context.Context) error {
	clusterRoleBindingName := key.WriteAllGSGroupClusterRoleBindingName()

	readAllClusterRoleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleBindingName,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "Group",
				Name: b.gsAdminGroup,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     key.ClusterAdminClusterRoleName,
		},
	}

	_, err := b.k8sClient.RbacV1().ClusterRoleBindings().Get(ctx, readAllClusterRoleBinding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating clusterrolebinding %#q", readAllClusterRoleBinding.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, readAllClusterRoleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrolebinding %#q has been created", readAllClusterRoleBinding.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrolebinding %#q already exists", readAllClusterRoleBinding.Name))
	}

	return nil
}

// Grant cluster-admin access for customer admin group to default namespace.
func (b *Bootstrap) createWriteAllRoleBindingToCustomerGroup(ctx context.Context) error {
	roleBindingName := key.WriteAllCustomerGroupRoleBindingName()

	writeAllRoleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: roleBindingName,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "Group",
				Name: b.customerAdminGroup,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     key.ClusterAdminClusterRoleName,
		},
	}

	ns := key.DefaultNamespaceName
	_, err := b.k8sClient.RbacV1().RoleBindings(ns).Get(ctx, writeAllRoleBinding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating rolebinding %#q/%#q", ns, writeAllRoleBinding.Name))

		_, err := b.k8sClient.RbacV1().RoleBindings(ns).Create(ctx, writeAllRoleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("rolebinding %#q/%#q has been created", ns, writeAllRoleBinding.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("rolebinding %#q/%#q already exists", ns, writeAllRoleBinding.Name))
	}

	return nil
}

// Grant cluster-admin access for automation service account to default namespace.
func (b *Bootstrap) createWriteAllRoleBindingToAutomationSA(ctx context.Context) error {
	roleBindingName := key.WriteAllAutomationSARoleBindingName()

	writeAllRoleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: roleBindingName,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      key.AutomationServiceAccountName,
				Namespace: key.DefaultNamespaceName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     key.ClusterAdminClusterRoleName,
		},
	}

	ns := key.DefaultNamespaceName
	_, err := b.k8sClient.RbacV1().RoleBindings(ns).Get(ctx, writeAllRoleBinding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating rolebinding %#q/%#q", ns, writeAllRoleBinding.Name))

		_, err := b.k8sClient.RbacV1().RoleBindings(ns).Create(ctx, writeAllRoleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("rolebinding %#q/%#q has been created", ns, writeAllRoleBinding.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("rolebinding %#q/%#q already exists", ns, writeAllRoleBinding.Name))
	}

	return nil
}

// Grant cluster-admin access for automation service account to default namespace.
func (b *Bootstrap) labelDefaultClusterRoles(ctx context.Context) error {
	clusterRoles := key.DefaultClusterRolesToDisplayInUI()

	for _, clusterRole := range clusterRoles {

		clusterRoleToLabel, err := b.k8sClient.RbacV1().ClusterRoles().Get(ctx, clusterRole, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrole %#q doesn't exist", clusterRole))
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			displayLabel, displayLabelExists := clusterRoleToLabel.Labels[label.DisplayInUserInterface]
			if displayLabelExists && displayLabel == "true" {
				b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrole %#q already labeled", clusterRole))
				return nil
			}
			b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("labeling clusterrole %#q", clusterRole))

			clusterRoleToLabel.Labels[label.DisplayInUserInterface] = "true"

			_, err := b.k8sClient.RbacV1().ClusterRoles().Update(ctx, clusterRoleToLabel, metav1.UpdateOptions{})
			if err != nil {
				return microerror.Mask(err)
			}
			b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrole %#q has been labeled", clusterRole))
		}
	}

	return nil
}

func isRestrictedResource(resource string) bool {
	var restrictedResources = []string{"configmaps", "secrets"}

	for _, restrictedResource := range restrictedResources {
		if resource == restrictedResource {
			return true
		}
	}
	return false
}
