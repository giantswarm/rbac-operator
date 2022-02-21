// Package bootstrap ensures certain RBAC resources like ClusterRoles,
// ClusterRoleBindings, and ServiceAccounts. If they exist, they will
// be modified according to spec. If not, they are created.
package bootstrap

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/pkg/project"
)

// Ensures the 'automation' service account in the default namespace.
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
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating serviceaccount %#q in namespace %s", automationSA.Name, key.DefaultNamespaceName))

		_, err := b.k8sClient.CoreV1().ServiceAccounts(key.DefaultNamespaceName).Create(ctx, automationSA, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("serviceaccount %#q in namespace %s has been created", automationSA.Name, key.DefaultNamespaceName))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating serviceaccount %#q in namespace %s", automationSA.Name, key.DefaultNamespaceName))

		_, err := b.k8sClient.CoreV1().ServiceAccounts(key.DefaultNamespaceName).Update(ctx, automationSA, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("serviceaccount %#q in namespace %s has been updated", automationSA.Name, key.DefaultNamespaceName))
	}

	return nil
}

// Ensures the ClusterRole 'read-all'.
//
// Purpose if this role is to enable read permissions (get, list, watch)
// for all resources except ConfigMap and Secret.
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
			Annotations: map[string]string{
				annotation.Notes: "Grants read-only (get, list, watch) permissions to almost all resource types known on the management cluster, with exception of ConfigMap and Secret.",
			},
		},
		Rules: policyRules,
	}

	_, err = b.k8sClient.RbacV1().ClusterRoles().Get(ctx, readOnlyClusterRole.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating clusterrole %#q", readOnlyClusterRole.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoles().Create(ctx, readOnlyClusterRole, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrole %#q has been created", readOnlyClusterRole.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating clusterrole %#q", readOnlyClusterRole.Name))
		_, err := b.k8sClient.RbacV1().ClusterRoles().Update(ctx, readOnlyClusterRole, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrole %#q has been updated", readOnlyClusterRole.Name))
	}

	return nil
}

// Ensures the ClusterRole 'write-organizations'.
//
// Purpose of this role is to grant all permissions for the
// organizations.security.giantswarm.io resource.
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
			Annotations: map[string]string{
				annotation.Notes: "Grants full permissions to Organization CRs.",
			},
		},
		Rules: []rbacv1.PolicyRule{policyRule},
	}

	_, err := b.k8sClient.RbacV1().ClusterRoles().Get(ctx, orgAdminClusterRole.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating clusterrole %#q", orgAdminClusterRole.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoles().Create(ctx, orgAdminClusterRole, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrole %#q has been created", orgAdminClusterRole.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating clusterrole %#q", orgAdminClusterRole.Name))
		_, err := b.k8sClient.RbacV1().ClusterRoles().Update(ctx, orgAdminClusterRole, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrole %#q has been updated", orgAdminClusterRole.Name))
	}

	return nil
}

// Ensures the ClusterRoleBinding 'write-organizations-customer-group' between
// ClusterRole 'write-organizations' and the customer admin group.
func (b *Bootstrap) createWriteOrganizationsClusterRoleBindingToCustomerGroup(ctx context.Context) error {
	if b.customerAdminGroup == "" {
		return microerror.Maskf(invalidConfigError, "empty customer admin group name given")
	}

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
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating clusterrolebinding %#q", writeOrganizationsClusterRoleBinding.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, writeOrganizationsClusterRoleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrolebinding %#q has been created", writeOrganizationsClusterRoleBinding.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating clusterrolebinding %#q", writeOrganizationsClusterRoleBinding.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoleBindings().Update(ctx, writeOrganizationsClusterRoleBinding, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrolebinding %#q has been updated", writeOrganizationsClusterRoleBinding.Name))
	}

	return nil
}

// Ensures the ClusterRoleBinding 'read-all-customer-group' between
// ClusterRole 'read-all' and the customer admin group.
func (b *Bootstrap) createReadAllClusterRoleBindingToCustomerGroup(ctx context.Context) error {
	if b.customerAdminGroup == "" {
		return microerror.Maskf(invalidConfigError, "empty customer admin group name given")
	}

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
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating clusterrolebinding %#q", readAllClusterRoleBinding.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, readAllClusterRoleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrolebinding %#q has been created", readAllClusterRoleBinding.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating clusterrolebinding %#q", readAllClusterRoleBinding.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoleBindings().Update(ctx, readAllClusterRoleBinding, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrolebinding %#q has been updated", readAllClusterRoleBinding.Name))
	}

	return nil
}

// Ensures the ClusterRoleBinding 'read-all-customer-sa' between
// ClusterRole 'read-all' and the ServiceAccount 'automation'.
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
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating clusterrolebinding %#q", readAllClusterRoleBinding.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, readAllClusterRoleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrolebinding %#q has been created", readAllClusterRoleBinding.Name))

	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// Ensures the ClusterRoleBinding 'write-all-giantswarm-group' between
// ClusterRole 'cluster-admin' and the Giant Swarm admin group.
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
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating clusterrolebinding %#q", readAllClusterRoleBinding.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, readAllClusterRoleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrolebinding %#q has been created", readAllClusterRoleBinding.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating clusterrolebinding %#q", readAllClusterRoleBinding.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoleBindings().Update(ctx, readAllClusterRoleBinding, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrolebinding %#q has been updated", readAllClusterRoleBinding.Name))
	}

	return nil
}

// Ensures the ClusterRoleBinding 'write-all-customer-group' between
// ClusterRole 'cluster-admin' and the customer admin group.
func (b *Bootstrap) createWriteAllRoleBindingToCustomerGroup(ctx context.Context) error {
	if b.customerAdminGroup == "" {
		return microerror.Maskf(invalidConfigError, "empty customer admin group name given")
	}

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
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating rolebinding %#q in namespace %s", writeAllRoleBinding.Name, ns))

		_, err := b.k8sClient.RbacV1().RoleBindings(ns).Create(ctx, writeAllRoleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("rolebinding %#q in namespace %s has been created", writeAllRoleBinding.Name, ns))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating rolebinding %#q in namespace %s", writeAllRoleBinding.Name, ns))

		_, err := b.k8sClient.RbacV1().RoleBindings(ns).Update(ctx, writeAllRoleBinding, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("rolebinding %#q in namespace %s has been updated", writeAllRoleBinding.Name, ns))
	}

	return nil
}

// Ensures a RoleBinding 'write-all-customer-sa' between
// ClusterRole 'cluster-admin' and ServiceAccount 'automation'
// in namespace 'default'.
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
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating rolebinding %#q in namespace %s", writeAllRoleBinding.Name, ns))

		_, err := b.k8sClient.RbacV1().RoleBindings(ns).Create(ctx, writeAllRoleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("rolebinding %#q in namespace %s has been created", writeAllRoleBinding.Name, ns))

	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// Ensures the ClusterRoleBinding 'write-organizations-customer-sa' between
// ClusterRole 'write-organizations' and ServiceAccount 'automation'.
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
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating clusterrolebinding %#q", writeOrganizationsClusterRoleBinding.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, writeOrganizationsClusterRoleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrolebinding %#q has been created", writeOrganizationsClusterRoleBinding.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating clusterrolebinding %#q", writeOrganizationsClusterRoleBinding.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoleBindings().Update(ctx, writeOrganizationsClusterRoleBinding, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrolebinding %#q has been updated", writeOrganizationsClusterRoleBinding.Name))
	}

	return nil
}

// Ensures the ClusterRole 'write-flux-resources'.
//
// Purpose of this role is to grant all permissions for certain
// *.toolkit.fluxcd.io resources.
func (b *Bootstrap) createWriteFluxResourcesClusterRole(ctx context.Context) error {
	policyRule := rbacv1.PolicyRule{
		APIGroups: []string{
			"helm.toolkit.fluxcd.io",
			"image.toolkit.fluxcd.io",
			"kustomizations.kustomize.toolkit.fluxcd.io",
			"notification.toolkit.fluxcd.io",
			"source.toolkit.fluxcd.io",
		},
		Resources: []string{
			"alerts",
			"buckets",
			"gitrepositories",
			"helmcharts",
			"helmreleases",
			"helmrepositories",
			"imagepolicies",
			"imagerepositories",
			"imageupdateautomations",
			"kustomizations",
			"providers",
			"receivers",
		},
		Verbs: []string{"*"},
	}

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: key.WriteFluxResourcesPermissionsName,
			Labels: map[string]string{
				label.ManagedBy:              project.Name(),
				label.DisplayInUserInterface: "true",
			},
			Annotations: map[string]string{
				annotation.Notes: "Grants full permissions to FluxCD related resource types.",
			},
		},
		Rules: []rbacv1.PolicyRule{policyRule},
	}

	_, err := b.k8sClient.RbacV1().ClusterRoles().Get(ctx, clusterRole.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating clusterrole %#q", clusterRole.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// Do nothing.
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrole %#q has been created", clusterRole.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating clusterrole %#q", clusterRole.Name))
		_, err := b.k8sClient.RbacV1().ClusterRoles().Update(ctx, clusterRole, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrole %#q has been updated", clusterRole.Name))
	}

	return nil
}

// Ensures the ClusterRoleBinding 'write-flux-resources-customer-sa' between
// ClusterRole 'write-flux-resources' and ServiceAccount 'automation'.
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
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating clusterrolebinding %#q", clusterRoleBinding.Name))

		_, err = b.k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// Do nothing.
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrolebinding %#q has been created", clusterRoleBinding.Name))

		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating clusterrolebinding %#q", clusterRoleBinding.Name))

	_, err = b.k8sClient.RbacV1().ClusterRoleBindings().Update(ctx, clusterRoleBinding, metav1.UpdateOptions{})
	if err != nil {
		return microerror.Mask(err)
	}
	b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrolebinding %#q has been updated", clusterRoleBinding.Name))

	return nil
}

// Ensures the ClusterRole 'write-clusters'.
//
// Purpose of this role is to grant all permissions needed for
// creating, modifying, and deleting clusters, not including
// node pools.
func (b *Bootstrap) createWriteClustersClusterRole(ctx context.Context) error {
	policyRule := rbacv1.PolicyRule{
		APIGroups: []string{
			"cluster.x-k8s.io",
			"infrastructure.cluster.x-k8s.io",
			"infrastructure.giantswarm.io",
		},
		Resources: []string{
			"awsclusters",
			"awscontrolplanes",
			"azureclusters",
			"azuremachines",
			"clusters",
			"g8scontrolplanes",
		},
		Verbs: []string{"*"},
	}

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: key.WriteClustersPermissionsName,
			Labels: map[string]string{
				label.ManagedBy:              project.Name(),
				label.DisplayInUserInterface: "true",
			},
			Annotations: map[string]string{
				annotation.Notes: "Grants full permissions to resources for clusters, excluding node pools.",
			},
		},
		Rules: []rbacv1.PolicyRule{policyRule},
	}

	_, err := b.k8sClient.RbacV1().ClusterRoles().Get(ctx, clusterRole.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating clusterrole %#q", clusterRole.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// Do nothing.
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrole %#q has been created", clusterRole.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating clusterrole %#q", clusterRole.Name))
		_, err := b.k8sClient.RbacV1().ClusterRoles().Update(ctx, clusterRole, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrole %#q has been updated", clusterRole.Name))
	}

	return nil
}

// Ensures the ClusterRoleBinding 'write-clusters-customer-sa' between
// ClusterRole 'write-clusters' and ServiceAccount 'automation'.
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
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating clusterrolebinding %#q", clusterRoleBinding.Name))

		_, err = b.k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// Do nothing.
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrolebinding %#q has been created", clusterRoleBinding.Name))

		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating clusterrolebinding %#q", clusterRoleBinding.Name))

	_, err = b.k8sClient.RbacV1().ClusterRoleBindings().Update(ctx, clusterRoleBinding, metav1.UpdateOptions{})
	if err != nil {
		return microerror.Mask(err)
	}
	b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrolebinding %#q has been updated", clusterRoleBinding.Name))

	return nil
}

// Ensures the ClusterRole 'write-nodepools'.
//
// Purpose of this role is to grant all permissions needed for
// creating, modifying, and deleting node pools.
func (b *Bootstrap) createWriteNodePoolsClusterRole(ctx context.Context) error {
	policyRule := rbacv1.PolicyRule{
		APIGroups: []string{
			"cluster.x-k8s.io",
			"core.giantswarm.io",
			"exp.cluster.x-k8s.io",
			"infrastructure.cluster.x-k8s.io",
			"infrastructure.giantswarm.io",
		},
		Resources: []string{
			"awsmachinedeployments",
			"azuremachinepools",
			"machinedeployments",
			"machinepools",
			"networkpools",
			"sparks",
		},
		Verbs: []string{"*"},
	}

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: key.WriteNodePoolsPermissionsName,
			Labels: map[string]string{
				label.ManagedBy:              project.Name(),
				label.DisplayInUserInterface: "true",
			},
			Annotations: map[string]string{
				annotation.Notes: "Grants full permissions on resources representing node pools.",
			},
		},
		Rules: []rbacv1.PolicyRule{policyRule},
	}

	_, err := b.k8sClient.RbacV1().ClusterRoles().Get(ctx, clusterRole.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating clusterrole %#q", clusterRole.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// Do nothing.
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrole %#q has been created", clusterRole.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating clusterrole %#q", clusterRole.Name))
		_, err := b.k8sClient.RbacV1().ClusterRoles().Update(ctx, clusterRole, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrole %#q has been updated", clusterRole.Name))
	}

	return nil
}

// Ensures the ClusterRoleBinding 'write-nodepools-customer-sa' between
// ClusterRole 'write-nodepools' and ServiceAccount 'automation'.
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
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating clusterrolebinding %#q", clusterRoleBinding.Name))

		_, err = b.k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// Do nothing.
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrolebinding %#q has been created", clusterRoleBinding.Name))

		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating clusterrolebinding %#q", clusterRoleBinding.Name))

	_, err = b.k8sClient.RbacV1().ClusterRoleBindings().Update(ctx, clusterRoleBinding, metav1.UpdateOptions{})
	if err != nil {
		return microerror.Mask(err)
	}
	b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrolebinding %#q has been updated", clusterRoleBinding.Name))

	return nil
}

// Ensures the ClusterRole 'write-client-certificates'.
//
// Purpose of this role is to grant all permissions needed for
// creating client certificates, which happens via the creation
// of certconfigs.core.giantswarm.io resources.
//
// Note: read access to secrets is not included.
func (b *Bootstrap) createWriteClientCertsClusterRole(ctx context.Context) error {
	policyRule := rbacv1.PolicyRule{
		APIGroups: []string{
			"core.giantswarm.io",
		},
		Resources: []string{
			"certconfigs",
		},
		Verbs: []string{"*"},
	}

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: key.WriteClientCertsPermissionsName,
			Labels: map[string]string{
				label.ManagedBy:              project.Name(),
				label.DisplayInUserInterface: "true",
			},
			Annotations: map[string]string{
				annotation.Notes: "Grants full permissions on certconfigs.core.giantswarm.io resources.",
			},
		},
		Rules: []rbacv1.PolicyRule{policyRule},
	}

	_, err := b.k8sClient.RbacV1().ClusterRoles().Get(ctx, clusterRole.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating clusterrole %#q", clusterRole.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// Do nothing.
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrole %#q has been created", clusterRole.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating clusterrole %#q", clusterRole.Name))
		_, err := b.k8sClient.RbacV1().ClusterRoles().Update(ctx, clusterRole, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrole %#q has been updated", clusterRole.Name))
	}

	return nil
}

// Ensures the ClusterRoleBinding 'write-client-certificates-customer-sa' between
// ClusterRole 'write-client-certificates' and ServiceAccount 'automation'.
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
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating clusterrolebinding %#q", clusterRoleBinding.Name))

		_, err = b.k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// Do nothing.
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrolebinding %#q has been created", clusterRoleBinding.Name))

		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating clusterrolebinding %#q", clusterRoleBinding.Name))

	_, err = b.k8sClient.RbacV1().ClusterRoleBindings().Update(ctx, clusterRoleBinding, metav1.UpdateOptions{})
	if err != nil {
		return microerror.Mask(err)
	}
	b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrolebinding %#q has been updated", clusterRoleBinding.Name))

	return nil
}

// Ensures the ClusterRole 'write-silences'.
//
// Purpose of this role is to grant all permissions needed for
// handling silences.monitoring.giantswarm.io resources.
func (b *Bootstrap) createWriteSilencesClusterRole(ctx context.Context) error {
	policyRule := rbacv1.PolicyRule{
		APIGroups: []string{
			"monitoring.giantswarm.io",
		},
		Resources: []string{
			"silences",
		},
		Verbs: []string{"*"},
	}

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: key.WriteSilencesPermissionsName,
			Labels: map[string]string{
				label.ManagedBy:              project.Name(),
				label.DisplayInUserInterface: "true",
			},
			Annotations: map[string]string{
				annotation.Notes: "Grants full permissions for silences.monitoring.giantswarm.io resources.",
			},
		},
		Rules: []rbacv1.PolicyRule{policyRule},
	}

	_, err := b.k8sClient.RbacV1().ClusterRoles().Get(ctx, clusterRole.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating clusterrole %#q", clusterRole.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// Do nothing.
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrole %#q has been created", clusterRole.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating clusterrole binding %#q", clusterRole.Name))
		_, err := b.k8sClient.RbacV1().ClusterRoles().Update(ctx, clusterRole, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrole %#q has been updated", clusterRole.Name))
	}

	return nil
}

// Ensures the ClusterRoleBinding 'write-silences-customer-sa' between
// ClusterRole 'write-silences' and ServiceAccount 'automation'.
func (b *Bootstrap) createWriteSilencesClusterRoleBindingToAutomationSA(ctx context.Context) error {
	clusterRoleBindingName := key.WriteSilencesAutomationSARoleBindingName()

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
			Name:     key.WriteSilencesPermissionsName,
		},
	}

	_, err := b.k8sClient.RbacV1().ClusterRoleBindings().Get(ctx, clusterRoleBinding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating clusterrolebinding %#q", clusterRoleBinding.Name))

		_, err = b.k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// Do nothing.
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrolebinding %#q has been created", clusterRoleBinding.Name))

		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating clusterrolebinding %#q", clusterRoleBinding.Name))

	_, err = b.k8sClient.RbacV1().ClusterRoleBindings().Update(ctx, clusterRoleBinding, metav1.UpdateOptions{})
	if err != nil {
		return microerror.Mask(err)
	}
	b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrolebinding %#q has been updated", clusterRoleBinding.Name))

	return nil
}

// Ensure labels on the ClusterRole 'cluster-admin':
//
// - 'ui.giantswarm.io/display=true'
// - 'giantswarm.io/managed-by=Kubernetes'
//
func (b *Bootstrap) labelDefaultClusterRoles(ctx context.Context) error {
	labelsToSet := map[string]string{
		label.DisplayInUserInterface: "true",
		key.LabelManagedBy:           "Kubernetes",
	}

	clusterRoles := key.DefaultClusterRolesToDisplayInUI()

	for _, clusterRole := range clusterRoles {
		clusterRoleToLabel, err := b.k8sClient.RbacV1().ClusterRoles().Get(ctx, clusterRole, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			b.logger.LogCtx(ctx, "level", "warn", "message", fmt.Sprintf("clusterrole %#q does not exist", clusterRole))
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		needsUpdate := false
		for k, label := range labelsToSet {
			if existingValue, ok := clusterRoleToLabel.Labels[k]; ok && existingValue == label {
				continue
			}

			b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("adding label %s to clusterrole %#q", label, clusterRole))

			clusterRoleToLabel.Labels[k] = label
			needsUpdate = true
		}

		if !needsUpdate {
			continue
		}

		_, err = b.k8sClient.RbacV1().ClusterRoles().Update(ctx, clusterRoleToLabel, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrole %#q has been updated with labels", clusterRole))
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
