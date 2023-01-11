package crossplaneauth

import (
	"context"
	"fmt"

	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/pkg/project"
	"github.com/giantswarm/rbac-operator/service/controller/crossplane/key"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToClusterRole(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	if cr.Name != key.CrossplaneEditClusterRole() {
		return nil
	}

	// check if we already have the clusterrolebinding
	_, err = r.k8sClient.RbacV1().ClusterRoleBindings().Get(ctx, key.GetClusterRoleBindingName(), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "info", "message",
			fmt.Sprintf("ClusterRole '%s' is detected, creating ClusterRoleBinding '%s' for Crossplane rbac-manager",
				key.CrossplaneEditClusterRole(), key.GetClusterRoleBindingName()))

		subjects := []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      pkgkey.AutomationServiceAccountName,
				Namespace: pkgkey.DefaultNamespaceName,
			},
		}
		for _, group := range r.customerAdminGroups {
			subjects = append(subjects, rbacv1.Subject{
				Kind:      "Group",
				Name:      group.Name,
				Namespace: pkgkey.DefaultNamespaceName,
			})
		}

		clusterRoleBinding := &rbacv1.ClusterRoleBinding{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ClusterRoleBinding",
				APIVersion: "rbac.authorization.k8s.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: key.GetClusterRoleBindingName(),
				Labels: map[string]string{
					label.ManagedBy: project.Name(),
				},
			},
			Subjects: subjects,
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     key.CrossplaneEditClusterRole(),
			},
		}
		_, err := r.k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "info", "message",
			fmt.Sprintf("ClusterRoleBinding %#q between customer's admin group and rbac-manager of Crossplane been created",
				key.CrossplaneEditClusterRole()))
	}

	return nil
}
