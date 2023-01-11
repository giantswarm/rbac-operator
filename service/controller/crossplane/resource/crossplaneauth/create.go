package crossplaneauth

import (
	"context"
	"fmt"

	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/pkg/project"
	pkgrbac "github.com/giantswarm/rbac-operator/pkg/rbac"
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

	err = pkgrbac.CreateOrUpdateClusterRoleBinding(r, ctx, clusterRoleBinding)

	if apierrors.IsAlreadyExists(err) {
		// do nothing
	} else if err != nil {
		return microerror.Mask(err)
	}
	r.logger.LogCtx(ctx, "level", "info", "message",
		fmt.Sprintf("ClusterRoleBinding %#q between customer's admin group and rbac-manager of Crossplane been created",
			key.CrossplaneEditClusterRole()))

	return nil
}
