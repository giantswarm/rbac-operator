package catalog

import (
	"context"
	"fmt"

	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/pkg/project"
	"github.com/giantswarm/rbac-operator/service/controller/cluster/key"
)

// EnsureCreated Ensures the Role 'read-default-catalogs'.
//
// Purpose if this role is to enable read permissions (get, list, watch)
// for catalog resources which are in the default namespace
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	namespace, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if !pkgkey.IsDefaultNamespace(namespace.Name) {
		return nil
	}

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pkgkey.ReadDefaultCatalogsRole,
			Namespace: namespace.Name,
			Labels: map[string]string{
				label.ManagedBy:              project.Name(),
				label.DisplayInUserInterface: "false",
			},
			Annotations: map[string]string{
				annotation.Notes: "Grants permissions needed for fetching Catalog and AppCatalogEntry CRs in the default namespace. Will be granted automatically to any subject bound to an Organization namespace.",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"application.giantswarm.io"},
				Resources: []string{"catalogs"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"application.giantswarm.io"},
				Resources: []string{"appcatalogentries"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}

	_, err = r.K8sClient().RbacV1().Roles(role.Namespace).Get(ctx, role.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		r.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating role %#q in namespace %s", role.Name, role.Namespace))

		_, err := r.K8sClient().RbacV1().Roles(role.Namespace).Create(ctx, role, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("role %#q in namespace %s has been created", role.Name, role.Namespace))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		r.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating role %#q in namespace %s", role.Name, role.Namespace))
		_, err := r.K8sClient().RbacV1().Roles(role.Namespace).Update(ctx, role, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		r.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("role %#q in namespace %s has been updated", role.Name, role.Namespace))
	}

	return nil
}
