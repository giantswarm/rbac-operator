package bootstrap

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/pkg/project"
)

// Ensures the Role 'read-default-catalogs'.
//
// Purpose if this role is to enable read permissions (get, list, watch)
// for catalog resources which are in the default namespace
func (b *Bootstrap) createReadDefaultCatalogsRole(ctx context.Context) error {
	var err error

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.ReadDefaultCatalogsRole,
			Namespace: key.DefaultNamespaceName,
			Labels: map[string]string{
				label.ManagedBy:              project.Name(),
				label.DisplayInUserInterface: "false",
			},
			Annotations: map[string]string{
				annotation.Notes: "Grants permissions needed for fetching Catalog and App Catalog Entry CRs in the default namespace. Supposed to be bound via RoleBinding.",
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

	if err = b.createOrUpdateRole(ctx, role, role.Namespace); err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (b *Bootstrap) createOrUpdateRole(ctx context.Context, role *rbacv1.Role, namespace string) error {
	var err error
	_, err = b.k8sClient.RbacV1().Roles(namespace).Get(ctx, role.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating role %#q", role.Name))

		_, err := b.k8sClient.RbacV1().Roles(namespace).Create(ctx, role, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("role %#q has been created", role.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating role %#q", role.Name))
		_, err := b.k8sClient.RbacV1().Roles(namespace).Update(ctx, role, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("role %#q has been updated", role.Name))
	}

	return nil
}
