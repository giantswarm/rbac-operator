package membership

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/pkg/label"
	"github.com/giantswarm/rbac-operator/pkg/project"
	"github.com/giantswarm/rbac-operator/service/controller/orgpermissions/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	var err error

	roleBinding, err := key.ToRoleBinding(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if !pkgkey.IsOrgNamespace(roleBinding.Namespace) || !isTargetRoleBinding(roleBinding) {
		return nil
	}

	orgName := pkgkey.OrganizationName(roleBinding.Namespace)
	orgReadClusterRoleBindingName := pkgkey.OrganizationReadClusterRoleBindingName(roleBinding.Name, orgName)
	orgReadClusterRoleName := pkgkey.OrganizationReadClusterRoleName(roleBinding.Namespace)

	orgReadClusterRoleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: orgReadClusterRoleBindingName,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Subjects: roleBinding.Subjects,
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     orgReadClusterRoleName,
		},
	}

	_, err = r.k8sClient.RbacV1().ClusterRoleBindings().Get(ctx, orgReadClusterRoleBinding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating clusterrolebinding %#q", orgReadClusterRoleBinding.Name))

		_, err := r.k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, orgReadClusterRoleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrolebinding %#q has been created", orgReadClusterRoleBinding.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrolebinding %#q already exists", orgReadClusterRoleBinding.Name))
	}

	return nil
}
