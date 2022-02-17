package externalresources

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/rbac-operator/pkg/label"
	"github.com/giantswarm/rbac-operator/pkg/project"
	"github.com/giantswarm/rbac-operator/service/controller/clusternamespace/key"
)

// Ensures that Subjects with any sort of access in the organization namespace also have read access to
// - releases (non-namespaced)
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	var err error

	ns, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	orgNamespace := ns.name

	// List roleBindings in org-namespace
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Listing RoleBindings in namespace %s.", orgNamespace))
	orgRoleBindings, err := r.k8sClient.K8sClient().RbacV1().RoleBindings(orgNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return microerror.Mask(err)
	} else if len(orgRoleBindings.Items) == 0 {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("No RoleBindings found in namespace %s.", orgNamespace))
		return nil
	}

	// Collect the subjects that need access
	var subjects []rbacv1.Subject
	for _, roleBinding := range orgRoleBindings.Items {
		if roleBindingHasReference(roleBinding) {
			subjects = append(subjects, roleBinding.Subjects...)
		}
	}
	// Ensure RoleBinding for release access
	err = r.ensureReleasesRoleBinding(ctx, subjects, orgNamespace)
	if err != nil {
		return microerror.Mask(err)
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
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Creating Role %#q.", role.Name))

		_, err := r.k8sClient.K8sClient().RbacV1().Roles(namespace).Create(ctx, role, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Role %#q has been created.", role.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else if roleNeedsUpdate(role, existingRole) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Updating Role %#q.", role.Name))
		_, err := r.k8sClient.K8sClient().RbacV1().Roles(namespace).Update(ctx, role, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Role %#q has been updated.", role.Name))

	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Role %#q already exists.", role.Name))
	}

	return nil
}

func (r *Resource) createOrUpdateRoleBinding(ctx context.Context, namespace string, roleBinding *rbacv1.RoleBinding) error {
	existingRoleBinding, err := r.k8sClient.K8sClient().RbacV1().RoleBindings(namespace).Get(ctx, roleBinding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Creating RoleBinding %#q.", roleBinding.Name))

		_, err := r.k8sClient.K8sClient().RbacV1().RoleBindings(namespace).Create(ctx, roleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("RoleBinding %#q has been created.", roleBinding.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else if roleBindingNeedsUpdate(roleBinding, existingRoleBinding) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Updating RoleBinding %#q.", roleBinding.Name))
		_, err := r.k8sClient.K8sClient().RbacV1().RoleBindings(namespace).Update(ctx, roleBinding, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("RoleBinding %#q has been updated.", roleBinding.Name))

	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("RoleBinding %#q already exists.", roleBinding.Name))
	}

	return nil
}
