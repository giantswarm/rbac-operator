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
	if err := r.ensurePatchChartsRole(ctx); err != nil {
		return microerror.Mask(err)
	}

	if err := r.addAutomationSAToPatchChartsRoleBinding(ctx, ns.Name); err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// ensurePatchChartsRole makes sure the shared `patch-charts` Role exists in the
// `giantswarm` namespace, granting list/get/patch on Chart resources.
func (r *Resource) ensurePatchChartsRole(ctx context.Context) error {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pkgkey.PatchChartsPermissionsName,
			Namespace: pkgkey.GiantSwarmNamespaceName,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"application.giantswarm.io"},
				Resources: []string{"charts"},
				Verbs:     []string{"list", "get", "patch"},
			},
		},
	}

	_, err := r.k8sClient.RbacV1().Roles(pkgkey.GiantSwarmNamespaceName).Get(ctx, role.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating role %#q in namespace %s", role.Name, pkgkey.GiantSwarmNamespaceName))

		_, err := r.k8sClient.RbacV1().Roles(pkgkey.GiantSwarmNamespaceName).Create(ctx, role, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("role %#q in namespace %s has been created", role.Name, pkgkey.GiantSwarmNamespaceName))
	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// addAutomationSAToPatchChartsRoleBinding makes sure the shared `patch-charts`
// RoleBinding exists in the `giantswarm` namespace and that the automation
// ServiceAccount of the given org namespace is listed in its subjects.
func (r *Resource) addAutomationSAToPatchChartsRoleBinding(ctx context.Context, namespace string) error {
	subject := rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      pkgkey.AutomationServiceAccountName,
		Namespace: namespace,
	}

	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pkgkey.PatchChartsPermissionsName,
			Namespace: pkgkey.GiantSwarmNamespaceName,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Subjects: []rbacv1.Subject{subject},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     pkgkey.PatchChartsPermissionsName,
		},
	}

	existing, err := r.k8sClient.RbacV1().RoleBindings(pkgkey.GiantSwarmNamespaceName).Get(ctx, roleBinding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating rolebinding %#q in namespace %s", roleBinding.Name, pkgkey.GiantSwarmNamespaceName))

		_, err := r.k8sClient.RbacV1().RoleBindings(pkgkey.GiantSwarmNamespaceName).Create(ctx, roleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("rolebinding %#q in namespace %s has been created", roleBinding.Name, pkgkey.GiantSwarmNamespaceName))
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	if slices.Contains(existing.Subjects, subject) {
		return nil
	}

	existing.Subjects = append(existing.Subjects, subject)
	r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("adding automation SA of namespace %s to rolebinding %#q", namespace, roleBinding.Name))

	_, err = r.k8sClient.RbacV1().RoleBindings(pkgkey.GiantSwarmNamespaceName).Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return microerror.Mask(err)
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
