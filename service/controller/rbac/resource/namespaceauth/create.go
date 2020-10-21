package namespaceauth

import (
	"context"
	"fmt"
	"reflect"
	"sort"

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

		existingRole, err := r.k8sClient.RbacV1().Roles(namespace.Name).Get(ctx, newRole.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating role %#q", newRole.Name))

			_, err := r.k8sClient.RbacV1().Roles(namespace.Name).Create(ctx, newRole, metav1.CreateOptions{})
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

			if !areRolesEqual(newRole, existingRole) {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("rules in role %#q need to be updated", newRole.Name))
				_, err := r.k8sClient.RbacV1().Roles(namespace.Name).Update(ctx, newRole, metav1.UpdateOptions{})
				if err != nil {
					return microerror.Mask(err)
				}
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("role %#q has been updated", newRole.Name))
			}
		}

		{
			roleBindingName := fmt.Sprintf("%s-group", role.name)
			newGroupRoleBinding := newGroupRoleBinding(roleBindingName, role.targetGroup, role.name)

			existingRoleBinding, err := r.k8sClient.RbacV1().RoleBindings(namespace.Name).Get(ctx, newGroupRoleBinding.Name, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating role binding %#q", newGroupRoleBinding.Name))

				_, err := r.k8sClient.RbacV1().RoleBindings(namespace.Name).Create(ctx, newGroupRoleBinding, metav1.CreateOptions{})
				if apierrors.IsAlreadyExists(err) {
					// do nothing
				} else if err != nil {
					return microerror.Mask(err)
				}

				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("role binding %#q has been created", newGroupRoleBinding.Name))

			} else if err != nil {
				return microerror.Mask(err)
			} else if needsUpdate(newGroupRoleBinding, existingRoleBinding) {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating role binding %#q", newGroupRoleBinding.Name))
				_, err := r.k8sClient.RbacV1().RoleBindings(namespace.Name).Update(ctx, newGroupRoleBinding, metav1.UpdateOptions{})
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

			existingRoleBinding, err := r.k8sClient.RbacV1().RoleBindings(namespace.Name).Get(ctx, newServiceAccountRoleBinding.Name, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating role binding %#q", newServiceAccountRoleBinding.Name))

				_, err := r.k8sClient.RbacV1().RoleBindings(namespace.Name).Create(ctx, newServiceAccountRoleBinding, metav1.CreateOptions{})
				if apierrors.IsAlreadyExists(err) {
					// do nothing
				} else if err != nil {
					return microerror.Mask(err)
				}

				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("role binding %#q has been created", newServiceAccountRoleBinding.Name))

			} else if err != nil {
				return microerror.Mask(err)
			} else if needsUpdate(newServiceAccountRoleBinding, existingRoleBinding) {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating role binding %#q", newServiceAccountRoleBinding.Name))
				_, err := r.k8sClient.RbacV1().RoleBindings(namespace.Name).Update(ctx, newServiceAccountRoleBinding, metav1.UpdateOptions{})
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

func needsUpdate(desiredRoleBinding, existingRoleBinding *rbacv1.RoleBinding) bool {
	if len(existingRoleBinding.Subjects) < 1 {
		return true
	}

	return desiredRoleBinding.Subjects[0].Name != existingRoleBinding.Subjects[0].Name || desiredRoleBinding.Subjects[0].Namespace != existingRoleBinding.Subjects[0].Namespace || desiredRoleBinding.RoleRef.Name != existingRoleBinding.RoleRef.Name
}

func areRolesEqual(role1, role2 *rbacv1.Role) bool {
	if len(role1.Rules) < 1 || len(role2.Rules) < 1 {
		return false
	}

	sort.Strings(role1.Rules[0].Resources)
	sort.Strings(role1.Rules[0].APIGroups)
	sort.Strings(role1.Rules[0].Verbs)

	sort.Strings(role2.Rules[0].Resources)
	sort.Strings(role2.Rules[0].APIGroups)
	sort.Strings(role2.Rules[0].Verbs)

	return reflect.DeepEqual(role1.Rules, role2.Rules)
}
