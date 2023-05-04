package namespaceauth

import (
	"context"
	"fmt"
	"reflect"

	k8smetadata "github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/rbac-operator/service/internal/accessgroup"

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

	// Create ClusterRole allowing 'get' access to Organization CR
	{
		orgReadClusterRoleName := pkgkey.OrganizationReadClusterRoleName(ns.Name)

		orgReadClusterRole := &rbacv1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ClusterRole",
				APIVersion: "rbac.authorization.k8s.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: orgReadClusterRoleName,
				Labels: map[string]string{
					k8smetadata.ManagedBy: project.Name(),
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups:     []string{"security.giantswarm.io"},
					Resources:     []string{"organizations"},
					ResourceNames: []string{pkgkey.OrganizationName(ns.Name)},
					Verbs:         []string{"get"},
				},
			},
		}

		_, err = r.k8sClient.RbacV1().ClusterRoles().Get(ctx, orgReadClusterRole.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating clusterrole %#q", orgReadClusterRole.Name))

			_, err := r.k8sClient.RbacV1().ClusterRoles().Create(ctx, orgReadClusterRole, metav1.CreateOptions{})
			if apierrors.IsAlreadyExists(err) {
				// do nothing
			} else if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrole %#q has been created", orgReadClusterRole.Name))

		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	// Bind the ClusterRole created before to the writeAllCustomerGroup (if set)
	writeAllCustomerGroupSubjects := accessgroup.GroupsToSubjects(r.writeAllCustomerGroups)
	if len(writeAllCustomerGroupSubjects) > 0 {
		roleBindingToCustomerGroupName := pkgkey.WriteAllCustomerGroupRoleBindingName()

		writeAllRoleBindingToCustomerGroup := &rbacv1.RoleBinding{
			TypeMeta: metav1.TypeMeta{
				Kind:       "RoleBinding",
				APIVersion: "rbac.authorization.k8s.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: roleBindingToCustomerGroupName,
				Labels: map[string]string{
					k8smetadata.ManagedBy: project.Name(),
				},
			},
			Subjects: writeAllCustomerGroupSubjects,
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     pkgkey.ClusterAdminClusterRoleName,
			},
		}

		roleBinding := writeAllRoleBindingToCustomerGroup
		existingRoleBinding, err := r.k8sClient.RbacV1().RoleBindings(ns.Name).Get(ctx, roleBinding.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating rolebinding %#q in namespace %s", roleBinding.Name, ns.Name))

			_, err := r.k8sClient.RbacV1().RoleBindings(ns.Name).Create(ctx, roleBinding, metav1.CreateOptions{})
			if apierrors.IsAlreadyExists(err) {
				// do nothing
			} else if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("rolebinding %#q in namespace %s has been created", roleBinding.Name, ns.Name))

		} else if err != nil {
			return microerror.Mask(err)
		} else if needsUpdate(roleBinding, existingRoleBinding) {
			r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating rolebinding %#q in namespace %s", roleBinding.Name, ns.Name))
			_, err := r.k8sClient.RbacV1().RoleBindings(ns.Name).Update(ctx, roleBinding, metav1.UpdateOptions{})
			if err != nil {
				return microerror.Mask(err)
			}
			r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("rolebinding %#q in namespace %s has been updated", roleBinding.Name, ns.Name))
		}
	}

	// Bind ClusterRole 'cluster-admin' to the 'automation' SA in the 'default' namespace.
	{
		roleBindingToAutomationSAName := pkgkey.WriteAllAutomationSARoleBindingName()

		writeAllRoleBindingToAutomationSA := &rbacv1.RoleBinding{
			TypeMeta: metav1.TypeMeta{
				Kind:       "RoleBinding",
				APIVersion: "rbac.authorization.k8s.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: roleBindingToAutomationSAName,
				Labels: map[string]string{
					k8smetadata.ManagedBy: project.Name(),
				},
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      pkgkey.AutomationServiceAccountName,
					Namespace: pkgkey.DefaultNamespaceName,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     pkgkey.ClusterAdminClusterRoleName,
			},
		}

		roleBinding := writeAllRoleBindingToAutomationSA
		_, err = r.k8sClient.RbacV1().RoleBindings(ns.Name).Get(ctx, roleBinding.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating rolebinding %#q in namespace %s", roleBinding.Name, ns.Name))

			_, err := r.k8sClient.RbacV1().RoleBindings(ns.Name).Create(ctx, roleBinding, metav1.CreateOptions{})
			if apierrors.IsAlreadyExists(err) {
				// do nothing
			} else if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("rolebinding %#q in namespace %s has been created", roleBinding.Name, ns.Name))

		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func needsUpdate(desiredRoleBinding, existingRoleBinding *rbacv1.RoleBinding) bool {
	if len(existingRoleBinding.Subjects) < 1 {
		return true
	}

	if !reflect.DeepEqual(existingRoleBinding.RoleRef, desiredRoleBinding.RoleRef) {
		return true
	}

	return !reflect.DeepEqual(existingRoleBinding.Subjects, desiredRoleBinding.Subjects)
}
