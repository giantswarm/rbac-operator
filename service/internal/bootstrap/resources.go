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
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating %#q service account", automationSA.Name))

		_, err := b.k8sClient.CoreV1().ServiceAccounts(key.DefaultNamespaceName).Update(ctx, automationSA, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("service account %#q has been updated", automationSA.Name))
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

func (b *Bootstrap) createWriteOrganizationsClusterRole(ctx context.Context) error {

	policyRule := rbacv1.PolicyRule{
		APIGroups: []string{"security.giantswarm.io"},
		Resources: []string{"organizations"},
		Verbs:     []string{"*"},
	}

	orgAdminClusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: key.WriteOrganizationsPermissionsName,
			Labels: map[string]string{
				label.ManagedBy:              project.Name(),
				label.DisplayInUserInterface: "true",
			},
		},
		Rules: []rbacv1.PolicyRule{policyRule},
	}

	_, err := b.k8sClient.RbacV1().ClusterRoles().Get(ctx, orgAdminClusterRole.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating clusterrole %#q", orgAdminClusterRole.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoles().Create(ctx, orgAdminClusterRole, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrole %#q has been created", orgAdminClusterRole.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating clusterrole binding %#q", orgAdminClusterRole.Name))
		_, err := b.k8sClient.RbacV1().ClusterRoles().Update(ctx, orgAdminClusterRole, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrole %#q has been updated", orgAdminClusterRole.Name))
	}

	return nil
}

// Grant customer admin write access to organizations.security.giantswarm.io.
func (b *Bootstrap) createWriteOrganizationsClusterRoleBindingToCustomerGroup(ctx context.Context) error {
	clusterRoleBindingName := key.WriteOrganizationsCustomerGroupClusterRoleBindingName()

	writeOrganizationsClusterRoleBinding := &rbacv1.ClusterRoleBinding{
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
			Name:     key.WriteOrganizationsPermissionsName,
		},
	}

	_, err := b.k8sClient.RbacV1().ClusterRoleBindings().Get(ctx, writeOrganizationsClusterRoleBinding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating clusterrolebinding %#q", writeOrganizationsClusterRoleBinding.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, writeOrganizationsClusterRoleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrolebinding %#q has been created", writeOrganizationsClusterRoleBinding.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating clusterrolebinding %#q", writeOrganizationsClusterRoleBinding.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoleBindings().Update(ctx, writeOrganizationsClusterRoleBinding, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrolebinding %#q has been updated", writeOrganizationsClusterRoleBinding.Name))
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
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating clusterrolebinding %#q", readAllClusterRoleBinding.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoleBindings().Update(ctx, readAllClusterRoleBinding, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrolebinding %#q has been updated", readAllClusterRoleBinding.Name))
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
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating clusterrolebinding %#q", readAllClusterRoleBinding.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoleBindings().Update(ctx, readAllClusterRoleBinding, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrolebinding %#q has been updated", readAllClusterRoleBinding.Name))
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
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating rolebinding %#q/%#q", ns, writeAllRoleBinding.Name))

		_, err := b.k8sClient.RbacV1().RoleBindings(ns).Update(ctx, writeAllRoleBinding, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("rolebinding %#q/%#q has been updated", ns, writeAllRoleBinding.Name))
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

func (b *Bootstrap) createWriteOrganizationsClusterRoleBindingToAutomationSA(ctx context.Context) error {
	clusterRoleBindingName := key.WriteOrganizationsAutomationSARoleBindingName()

	writeOrganizationsClusterRoleBinding := &rbacv1.ClusterRoleBinding{
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
			Name:     key.WriteOrganizationsPermissionsName,
		},
	}

	_, err := b.k8sClient.RbacV1().ClusterRoleBindings().Get(ctx, writeOrganizationsClusterRoleBinding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating clusterrolebinding %#q", writeOrganizationsClusterRoleBinding.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, writeOrganizationsClusterRoleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrolebinding %#q has been created", writeOrganizationsClusterRoleBinding.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating clusterrolebinding %#q", writeOrganizationsClusterRoleBinding.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoleBindings().Update(ctx, writeOrganizationsClusterRoleBinding, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrolebinding %#q has been updated", writeOrganizationsClusterRoleBinding.Name))
	}

	return nil
}

func (b *Bootstrap) createWriteFluxResourcesClusterRole(ctx context.Context) error {
	policyRule := rbacv1.PolicyRule{
		APIGroups: []string{"notification.toolkit.fluxcd.io", "source.toolkit.fluxcd.io", "image.toolkit.fluxcd.io", "helm.toolkit.fluxcd.io", "kustomizations.kustomize.toolkit.fluxcd.io"},
		Resources: []string{"alerts", "providers", "receivers", "buckets", "gitrepositories", "helmcharts", "helmrepositories", "imagepolicies", "imagerepositories", "imageupdateautomations", "helmreleases", "kustomizations"},
		Verbs:     []string{"*"},
	}

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: key.WriteFluxResourcesPermissionsName,
			Labels: map[string]string{
				label.ManagedBy:              project.Name(),
				label.DisplayInUserInterface: "true",
			},
		},
		Rules: []rbacv1.PolicyRule{policyRule},
	}

	_, err := b.k8sClient.RbacV1().ClusterRoles().Get(ctx, clusterRole.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating clusterrole %#q", clusterRole.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// Do nothing.
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrole %#q has been created", clusterRole.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating clusterrole binding %#q", clusterRole.Name))
		_, err := b.k8sClient.RbacV1().ClusterRoles().Update(ctx, clusterRole, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrole %#q has been updated", clusterRole.Name))
	}

	return nil
}

func (b *Bootstrap) createWriteFluxResourcesClusterRoleBindingToAutomationSA(ctx context.Context) error {
	clusterRoleBindingName := key.WriteFluxResourcesAutomationSARoleBindingName()

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
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
			Name:     key.WriteFluxResourcesPermissionsName,
		},
	}

	_, err := b.k8sClient.RbacV1().ClusterRoleBindings().Get(ctx, clusterRoleBinding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating clusterrolebinding %#q", clusterRoleBinding.Name))

		_, err = b.k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// Do nothing.
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrolebinding %#q has been created", clusterRoleBinding.Name))

		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating clusterrolebinding %#q", clusterRoleBinding.Name))

	_, err = b.k8sClient.RbacV1().ClusterRoleBindings().Update(ctx, clusterRoleBinding, metav1.UpdateOptions{})
	if err != nil {
		return microerror.Mask(err)
	}
	b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrolebinding %#q has been updated", clusterRoleBinding.Name))

	return nil
}

func (b *Bootstrap) createWriteClustersClusterRole(ctx context.Context) error {
	policyRule := rbacv1.PolicyRule{
		APIGroups: []string{"cluster.x-k8s.io", "infrastructure.cluster.x-k8s.io", "infrastructure.giantswarm.io"},
		Resources: []string{"clusters", "awsclusters", "awscontrolplanes", "g8scontrolplanes", "azureclusters", "azuremachines"},
		Verbs:     []string{"*"},
	}

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: key.WriteClustersPermissionsName,
			Labels: map[string]string{
				label.ManagedBy:              project.Name(),
				label.DisplayInUserInterface: "true",
			},
		},
		Rules: []rbacv1.PolicyRule{policyRule},
	}

	_, err := b.k8sClient.RbacV1().ClusterRoles().Get(ctx, clusterRole.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating clusterrole %#q", clusterRole.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// Do nothing.
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrole %#q has been created", clusterRole.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating clusterrole binding %#q", clusterRole.Name))
		_, err := b.k8sClient.RbacV1().ClusterRoles().Update(ctx, clusterRole, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrole %#q has been updated", clusterRole.Name))
	}

	return nil
}

func (b *Bootstrap) createWriteClustersClusterRoleBindingToAutomationSA(ctx context.Context) error {
	clusterRoleBindingName := key.WriteClustersAutomationSARoleBindingName()

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
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
			Name:     key.WriteClustersPermissionsName,
		},
	}

	_, err := b.k8sClient.RbacV1().ClusterRoleBindings().Get(ctx, clusterRoleBinding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating clusterrolebinding %#q", clusterRoleBinding.Name))

		_, err = b.k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// Do nothing.
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrolebinding %#q has been created", clusterRoleBinding.Name))

		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating clusterrolebinding %#q", clusterRoleBinding.Name))

	_, err = b.k8sClient.RbacV1().ClusterRoleBindings().Update(ctx, clusterRoleBinding, metav1.UpdateOptions{})
	if err != nil {
		return microerror.Mask(err)
	}
	b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrolebinding %#q has been updated", clusterRoleBinding.Name))

	return nil
}

func (b *Bootstrap) createWriteNodePoolsClusterRole(ctx context.Context) error {
	policyRule := rbacv1.PolicyRule{
		APIGroups: []string{"cluster.x-k8s.io", "exp.cluster.x-k8s.io", "infrastructure.cluster.x-k8s.io", "infrastructure.giantswarm.io", "core.giantswarm.io"},
		Resources: []string{"machinedeployments", "awsmachinedeployments", "networkpools", "machinepools", "azuremachinepools", "sparks"},
		Verbs:     []string{"*"},
	}

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: key.WriteNodePoolsPermissionsName,
			Labels: map[string]string{
				label.ManagedBy:              project.Name(),
				label.DisplayInUserInterface: "true",
			},
		},
		Rules: []rbacv1.PolicyRule{policyRule},
	}

	_, err := b.k8sClient.RbacV1().ClusterRoles().Get(ctx, clusterRole.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating clusterrole %#q", clusterRole.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// Do nothing.
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrole %#q has been created", clusterRole.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating clusterrole binding %#q", clusterRole.Name))
		_, err := b.k8sClient.RbacV1().ClusterRoles().Update(ctx, clusterRole, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrole %#q has been updated", clusterRole.Name))
	}

	return nil
}

func (b *Bootstrap) createWriteNodePoolsClusterRoleBindingToAutomationSA(ctx context.Context) error {
	clusterRoleBindingName := key.WriteNodePoolsAutomationSARoleBindingName()

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
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
			Name:     key.WriteNodePoolsPermissionsName,
		},
	}

	_, err := b.k8sClient.RbacV1().ClusterRoleBindings().Get(ctx, clusterRoleBinding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating clusterrolebinding %#q", clusterRoleBinding.Name))

		_, err = b.k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// Do nothing.
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrolebinding %#q has been created", clusterRoleBinding.Name))

		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating clusterrolebinding %#q", clusterRoleBinding.Name))

	_, err = b.k8sClient.RbacV1().ClusterRoleBindings().Update(ctx, clusterRoleBinding, metav1.UpdateOptions{})
	if err != nil {
		return microerror.Mask(err)
	}
	b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrolebinding %#q has been updated", clusterRoleBinding.Name))

	return nil
}

func (b *Bootstrap) createWriteClientCertsClusterRole(ctx context.Context) error {
	policyRule := rbacv1.PolicyRule{
		APIGroups: []string{"core.giantswarm.io"},
		Resources: []string{"certconfigs"},
		Verbs:     []string{"*"},
	}

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: key.WriteClientCertsPermissionsName,
			Labels: map[string]string{
				label.ManagedBy:              project.Name(),
				label.DisplayInUserInterface: "true",
			},
		},
		Rules: []rbacv1.PolicyRule{policyRule},
	}

	_, err := b.k8sClient.RbacV1().ClusterRoles().Get(ctx, clusterRole.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating clusterrole %#q", clusterRole.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// Do nothing.
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrole %#q has been created", clusterRole.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating clusterrole binding %#q", clusterRole.Name))
		_, err := b.k8sClient.RbacV1().ClusterRoles().Update(ctx, clusterRole, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrole %#q has been updated", clusterRole.Name))
	}

	return nil
}

func (b *Bootstrap) createWriteClientCertsClusterRoleBindingToAutomationSA(ctx context.Context) error {
	clusterRoleBindingName := key.WriteClientCertsAutomationSARoleBindingName()

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
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
			Name:     key.WriteClientCertsPermissionsName,
		},
	}

	_, err := b.k8sClient.RbacV1().ClusterRoleBindings().Get(ctx, clusterRoleBinding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating clusterrolebinding %#q", clusterRoleBinding.Name))

		_, err = b.k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// Do nothing.
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrolebinding %#q has been created", clusterRoleBinding.Name))

		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating clusterrolebinding %#q", clusterRoleBinding.Name))

	_, err = b.k8sClient.RbacV1().ClusterRoleBindings().Update(ctx, clusterRoleBinding, metav1.UpdateOptions{})
	if err != nil {
		return microerror.Mask(err)
	}
	b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrolebinding %#q has been updated", clusterRoleBinding.Name))

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
