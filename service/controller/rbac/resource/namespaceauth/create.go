package namespaceauth

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/rbac-operator/service/controller/rbac/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	namespace, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	resources, err := r.k8sClient.Discovery().ServerPreferredResources()
	if err != nil {
		return microerror.Mask(err)
	}

	viewAllRole := role{
		name:        "view-all",
		targetGroup: r.namespaceAuth.ViewAllTargetGroup,
		verbs:       []string{"get", "list", "watch"},
	}
	tenantAdminRole := role{
		name:        "tenant-admin",
		targetGroup: r.namespaceAuth.TenantAdminTargetGroup,
		verbs:       []string{"get", "list", "watch", "create", "update", "patch", "delete"},
	}

	roles := []role{
		viewAllRole,
		tenantAdminRole,
	}

	for _, role := range roles {

		newRole, err := newRole(role.name, resources, role.verbs)
		if err != nil {
			return microerror.Mask(err)
		}

		_, err = r.k8sClient.RbacV1().Roles(namespace.Name).Get(newRole.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating role %#q", newRole.Name))

			_, err := r.k8sClient.RbacV1().Roles(namespace.Name).Create(newRole)
			if apierrors.IsAlreadyExists(err) {
				// do nothing
			} else if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("role %#q has been created", newRole.Name))

		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("role %#q already exists", newRole.Name))
		}

		{
			roleBindingName := fmt.Sprintf("%s-group", role.name)
			newGroupRoleBinding := newGroupRoleBinding(roleBindingName, role.targetGroup, role.name)

			existingRoleBinding, err := r.k8sClient.RbacV1().RoleBindings(namespace.Name).Get(newGroupRoleBinding.Name, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating role binding %#q", newGroupRoleBinding.Name))

				_, err := r.k8sClient.RbacV1().RoleBindings(namespace.Name).Create(newGroupRoleBinding)
				if apierrors.IsAlreadyExists(err) {
					// do nothing
				} else if err != nil {
					return microerror.Mask(err)
				}

				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("role binding %#q has been created", newGroupRoleBinding.Name))

			} else if err != nil {
				return microerror.Mask(err)
			} else if needsUpdate(role, existingRoleBinding) {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating role binding %#q", newGroupRoleBinding.Name))
				_, err := r.k8sClient.RbacV1().RoleBindings(namespace.Name).Update(newGroupRoleBinding)
				if err != nil {
					return microerror.Mask(err)
				}
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("role binding %#q has been updated", newGroupRoleBinding.Name))

			} else {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("role binding %#q already exists", newGroupRoleBinding.Name))
			}
		}

		{
			roleBindingName := fmt.Sprintf("%s-sa", role.name)
			newServiceAccountRoleBinding := newServiceAccountRoleBinding(roleBindingName, automationServiceAccountName, automationServiceAccountNamespace, role.name)

			existingRoleBinding, err := r.k8sClient.RbacV1().RoleBindings(namespace.Name).Get(newServiceAccountRoleBinding.Name, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating role binding %#q", newServiceAccountRoleBinding.Name))

				_, err := r.k8sClient.RbacV1().RoleBindings(namespace.Name).Create(newServiceAccountRoleBinding)
				if apierrors.IsAlreadyExists(err) {
					// do nothing
				} else if err != nil {
					return microerror.Mask(err)
				}

				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("role binding %#q has been created", newServiceAccountRoleBinding.Name))

			} else if err != nil {
				return microerror.Mask(err)
			} else if needsUpdate(role, existingRoleBinding) {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating role binding %#q", newServiceAccountRoleBinding.Name))
				_, err := r.k8sClient.RbacV1().RoleBindings(namespace.Name).Update(newServiceAccountRoleBinding)
				if err != nil {
					return microerror.Mask(err)
				}
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("role binding %#q has been updated", newServiceAccountRoleBinding.Name))

			} else {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("role binding %#q already exists", newServiceAccountRoleBinding.Name))
			}
		}
	}

	return nil
}

func needsUpdate(role role, existingRoleBinding *rbacv1.RoleBinding) bool {
	return role.targetGroup != existingRoleBinding.Subjects[0].Name || role.name != existingRoleBinding.RoleRef.Name
}
