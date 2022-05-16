package bootstrap

import (
	"context"

	"github.com/giantswarm/rbac-operator/pkg/rbac"

	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"
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

	if err = rbac.CreateOrUpdateClusterRole(b, ctx, role); err != nil {
		return microerror.Mask(err)
	}

	return nil
}
