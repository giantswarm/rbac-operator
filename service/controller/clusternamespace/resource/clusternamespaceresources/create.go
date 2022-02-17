package clusternamespaceresources

import (
	"context"
	"fmt"

	securityv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/security/v1alpha1"
	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/pkg/label"
	"github.com/giantswarm/rbac-operator/pkg/project"
	"github.com/giantswarm/rbac-operator/service/controller/clusternamespace/key"
)

// Ensures that
// - Roles for read/write access to org cluster resources are ensured in each cluster namespace
// - For each RoleBinding in an org-namespace that references the read/write org cluster resource clusterRole,
//   RoleBindings are created in the organizations cluster namespaces which reference above Role
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	var err error

	cl, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	// Fetch the organization
	organization := securityv1alpha1.Organization{}
	err = r.k8sClient.CtrlClient().Get(ctx, types.NamespacedName{Name: pkgkey.Organization(&cl)}, &organization)
	if err != nil {
		return microerror.Mask(err)
	}
	orgNamespace := organization.Status.Namespace
	if len(orgNamespace) < 1 {
		return microerror.Maskf(unknownOrganizationNamespaceError, "Could not find the namespace for organization %s.", pkgkey.Organization(&cl))
	}

	// List roleBindings in org-namespace
	orgRoleBindings, err := r.k8sClient.K8sClient().RbacV1().RoleBindings(orgNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return microerror.Mask(err)
	} else if len(orgRoleBindings.Items) == 0 {
		return nil
	}

	for _, referencedRole := range referencedClusterRoles() {

		// Ensure Role in cluster namespace
		err = r.ensureClusterNamespaceNSRole(ctx, cl.Name, referencedRole.roleName, referencedRole.policyRules)
		if err != nil {
			return microerror.Mask(err)
		}

		// Collect the subjects that need access to org cluster resources
		var subjects []rbacv1.Subject
		for _, roleBinding := range orgRoleBindings.Items {
			if roleBindingReferencesClusterRole(roleBinding, referencedRole.roleName) {
				subjects = append(subjects, roleBinding.Subjects...)
			}
		}
		// Ensure RoleBinding in cluster namespace
		err = r.ensureClusterNamespaceNSRoleBinding(ctx, subjects, cl.Name, referencedRole)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (r *Resource) ensureClusterNamespaceNSRole(ctx context.Context, clusterNamespace string, referencedRole string, rules []rbacv1.PolicyRule) error {
	var err error

	role := &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: referencedRole,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
			Namespace: clusterNamespace,
		},
		Rules: rules,
	}

	if err = r.createOrUpdateRole(ctx, clusterNamespace, role); err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) ensureClusterNamespaceNSRoleBinding(ctx context.Context, subjects []rbacv1.Subject, clusterNamespace string, referencedRole rolePair) error {
	var err error

	roleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: referencedRole.roleBindingName,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
			Namespace: clusterNamespace,
		},
		Subjects: subjects,
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     referencedRole.roleName,
		},
	}

	if err = r.createOrUpdateRoleBinding(ctx, clusterNamespace, roleBinding); err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) createOrUpdateRole(ctx context.Context, namespace string, role *rbacv1.Role) error {
	existingRole, err := r.k8sClient.K8sClient().RbacV1().Roles(namespace).Get(ctx, role.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("Creating Role %#q in namespace %s.", role.Name, namespace))

		_, err := r.k8sClient.K8sClient().RbacV1().Roles(namespace).Create(ctx, role, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("Role %#q has been created.", role.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else if roleNeedsUpdate(role, existingRole) {
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("Updating Role %#q.", role.Name))
		_, err := r.k8sClient.K8sClient().RbacV1().Roles(namespace).Update(ctx, role, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("Role %#q has been updated.", role.Name))

	}

	return nil
}

func (r *Resource) createOrUpdateRoleBinding(ctx context.Context, namespace string, roleBinding *rbacv1.RoleBinding) error {
	existingRoleBinding, err := r.k8sClient.K8sClient().RbacV1().RoleBindings(namespace).Get(ctx, roleBinding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("Creating RoleBinding %#q in namespce %s.", roleBinding.Name, namespace))

		_, err := r.k8sClient.K8sClient().RbacV1().RoleBindings(namespace).Create(ctx, roleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("RoleBinding %#q in namespace %s has been created.", roleBinding.Name, namespace))

	} else if err != nil {
		return microerror.Mask(err)
	} else if roleBindingNeedsUpdate(roleBinding, existingRoleBinding) {
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("Updating RoleBinding %#q in namespace %s.", roleBinding.Name, namespace))
		_, err := r.k8sClient.K8sClient().RbacV1().RoleBindings(namespace).Update(ctx, roleBinding, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("RoleBinding %#q in namespace %s has been updated.", roleBinding.Name, namespace))

	}

	return nil
}
