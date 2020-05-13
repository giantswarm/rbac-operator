package namespaceauth

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/rbac-operator/service/controller/key"
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

		newRoleBinding := newRoleBinding(role.name, role.targetGroup)

		existingRoleBinding, err := r.k8sClient.RbacV1().RoleBindings(namespace.Name).Get(newRoleBinding.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating role binding %#q", newRoleBinding.Name))

			_, err := r.k8sClient.RbacV1().RoleBindings(namespace.Name).Create(newRoleBinding)
			if apierrors.IsAlreadyExists(err) {
				// do nothing
			} else if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("role binding %#q has been created", newRoleBinding.Name))

		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("role binding %#q already exists", newRoleBinding.Name))

			if role.targetGroup != existingRoleBinding.Subjects[0].Name {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating role binding %#q", newRoleBinding.Name))
				_, err := r.k8sClient.RbacV1().RoleBindings(namespace.Name).Update(newRoleBinding)
				if err != nil {
					return microerror.Mask(err)
				}
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("role binding %#q has been updated", newRoleBinding.Name))

			}
		}
	}

	return nil
}
