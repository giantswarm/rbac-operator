package bootstrap

import (
	"context"
	"fmt"

	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/pkg/project"
)

// Ensures the ClusterRole 'read-releases'.
//
// Purpose if this role is to enable read permissions (get, list, watch)
// for release resources which are not in a namespace
func (b *Bootstrap) createReadReleasesClusterRole(ctx context.Context) error {
	var err error

	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: key.ReadReleasesRole,
			Labels: map[string]string{
				label.ManagedBy:              project.Name(),
				label.DisplayInUserInterface: "false",
			},
			Annotations: map[string]string{
				annotation.Notes: "Grants permissions needed for fetching Release CRs, which are cluster scoped. Will be granted automatically to any subject bound in an Organization namespace.",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"release.giantswarm.io"},
				Resources: []string{"releases"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}

	if err = b.createOrUpdateClusterRole(ctx, role); err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (b *Bootstrap) createOrUpdateClusterRole(ctx context.Context, role *rbacv1.ClusterRole) error {
	var err error
	_, err = b.k8sClient.RbacV1().ClusterRoles().Get(ctx, role.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating clusterrole %#q", role.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoles().Create(ctx, role, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrole %#q has been created", role.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating clusterrole %#q", role.Name))
		_, err := b.k8sClient.RbacV1().ClusterRoles().Update(ctx, role, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		b.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrole %#q has been updated", role.Name))
	}

	return nil
}
