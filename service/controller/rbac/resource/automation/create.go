package automation

import (
	"context"
	"fmt"
	"reflect"
	"slices"

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

	// create a ClusterRoleBinding granting :
	// - write-silences access for "automation" ServiceAccount *in this org namespace*
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: pkgkey.WriteSilencesAutomationSAinNSRoleBindingName(ns.Name),
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      pkgkey.AutomationServiceAccountName,
			Namespace: ns.Name}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     pkgkey.WriteSilencesPermissionsName,
		},
	}

	if err := r.createOrUpdateClusterRoleBinding(ctx, ns, clusterRoleBinding); err != nil {
		return microerror.Mask(err)
	}

	// create a ClusterRoleBinding granting :
	// - kamaji datastore management for "automation" ServiceAccount *in this org namespace*
	// The referenced ClusterRole is provisioned by the global Kamaji app; the binding is
	// inert on clusters where that role doesn't exist.
	kamajiDatastoreBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: pkgkey.KamajiDatastoreManagerAutomationSAinNSRoleBindingName(ns.Name),
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      pkgkey.AutomationServiceAccountName,
			Namespace: ns.Name}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     pkgkey.KamajiDatastoreManagerPermissionsName,
		},
	}

	if err := r.createOrUpdateClusterRoleBinding(ctx, ns, kamajiDatastoreBinding); err != nil {
		return microerror.Mask(err)
	}

	// create the shared `patch-charts` Role and RoleBinding in the `giantswarm`
	// namespace and add this org's automation ServiceAccount to the RoleBinding
	// subjects. This is required for the App to HelmRelease migration and is
	// expected to be removed once that migration is complete.
	if err := r.ensureRole(ctx, pkgkey.PatchChartsPermissionsName, pkgkey.GiantSwarmNamespaceName, patchChartsRules); err != nil {
		return microerror.Mask(err)
	}

	if err := r.ensureRoleBinding(ctx, pkgkey.PatchChartsPermissionsName, pkgkey.GiantSwarmNamespaceName, ns.Name); err != nil {
		return microerror.Mask(err)
	}

	// create the shared `write-policy-exceptions` Role and RoleBinding in every
	// namespace in which PolicyExceptions are managed and add this org's
	// automation ServiceAccount to the RoleBinding subjects.
	for _, namespace := range pkgkey.WritePolicyExceptionsNamespaces {
		// The `policy-exceptions` namespace is created by the policy apps and is
		// absent on clusters which don't run them, so skip namespaces that don't
		// exist instead of failing the whole reconciliation.
		exists, err := r.namespaceExists(ctx, namespace)
		if err != nil {
			return microerror.Mask(err)
		} else if !exists {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("namespace %s does not exist, not ensuring role %#q in it", namespace, pkgkey.WritePolicyExceptionsPermissionsName))
			continue
		}

		if err := r.ensureRole(ctx, pkgkey.WritePolicyExceptionsPermissionsName, namespace, writePolicyExceptionsRules); err != nil {
			return microerror.Mask(err)
		}

		if err := r.ensureRoleBinding(ctx, pkgkey.WritePolicyExceptionsPermissionsName, namespace, ns.Name); err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (r *Resource) namespaceExists(ctx context.Context, namespace string) (bool, error) {
	_, err := r.k8sClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, microerror.Mask(err)
	}

	return true, nil
}

var (
	// patchChartsRules are the rules of the shared `patch-charts` Role, granting
	// list/get/patch on Chart resources.
	patchChartsRules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{"application.giantswarm.io"},
			Resources: []string{"charts"},
			Verbs:     []string{"list", "get", "patch"},
		},
	}

	// writePolicyExceptionsRules are the rules of the shared
	// `write-policy-exceptions` Role, granting all verbs on Kyverno
	// PolicyExceptions.
	writePolicyExceptionsRules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{"kyverno.io"},
			Resources: []string{"policyexceptions"},
			Verbs:     []string{"*"},
		},
	}
)

// ensureRole makes sure a shared Role with the given name, granting the given
// rules, exists in the given namespace.
func (r *Resource) ensureRole(ctx context.Context, name string, namespace string, rules []rbacv1.PolicyRule) error {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Rules: rules,
	}

	existing, err := r.k8sClient.RbacV1().Roles(namespace).Get(ctx, role.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating role %#q in namespace %s", role.Name, namespace))

		_, err := r.k8sClient.RbacV1().Roles(namespace).Create(ctx, role, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("role %#q in namespace %s has been created", role.Name, namespace))
	} else if err != nil {
		return microerror.Mask(err)
	} else if !reflect.DeepEqual(role.Rules, existing.Rules) {
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating role %#q in namespace %s", role.Name, namespace))

		existing.Rules = role.Rules

		_, err := r.k8sClient.RbacV1().Roles(namespace).Update(ctx, existing, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("role %#q in namespace %s has been updated", role.Name, namespace))
	}

	return nil
}

// ensureRoleBinding makes sure a shared RoleBinding with the given name,
// referencing the Role of the same name, exists in the given namespace and that
// the automation ServiceAccount of the given org namespace is listed in its
// subjects.
func (r *Resource) ensureRoleBinding(ctx context.Context, name string, namespace string, orgNamespace string) error {
	subject := rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      pkgkey.AutomationServiceAccountName,
		Namespace: orgNamespace,
	}

	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Subjects: []rbacv1.Subject{subject},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     name,
		},
	}

	existing, err := r.k8sClient.RbacV1().RoleBindings(namespace).Get(ctx, roleBinding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating rolebinding %#q in namespace %s", roleBinding.Name, namespace))

		_, err := r.k8sClient.RbacV1().RoleBindings(namespace).Create(ctx, roleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("rolebinding %#q in namespace %s has been created", roleBinding.Name, namespace))
	} else if err != nil {
		return microerror.Mask(err)
	} else if !slices.Contains(existing.Subjects, subject) {
		existing.Subjects = append(existing.Subjects, subject)
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("adding automation SA of namespace %s to rolebinding %#q in namespace %s", orgNamespace, roleBinding.Name, namespace))

		_, err := r.k8sClient.RbacV1().RoleBindings(namespace).Update(ctx, existing, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (r *Resource) createOrUpdateClusterRoleBinding(ctx context.Context, ns corev1.Namespace, clusterRoleBinding *rbacv1.ClusterRoleBinding) error {
	existingClusterRoleBinding, err := r.k8sClient.RbacV1().ClusterRoleBindings().Get(ctx, clusterRoleBinding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating clusterrolebinding %#q for Automation SA in namespace %s", clusterRoleBinding.Name, ns.Name))

		_, err := r.k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrolebinding %#q for Automation SA in namespace %s has been created", clusterRoleBinding.Name, ns.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else if needsUpdateClusterRoleBinding(clusterRoleBinding, existingClusterRoleBinding) {
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating cluster role binding %#q for Automation SA in namespace %s", clusterRoleBinding.Name, ns.Name))
		_, err := r.k8sClient.RbacV1().ClusterRoleBindings().Update(ctx, clusterRoleBinding, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("cluster role binding %#q for Automation SA in namespace %s has been updated", clusterRoleBinding.Name, ns.Name))

	}

	return nil
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
