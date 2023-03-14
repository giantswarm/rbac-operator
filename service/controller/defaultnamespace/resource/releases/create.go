package releases

import (
	"context"

	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/pkg/project"
	"github.com/giantswarm/rbac-operator/pkg/rbac"
	"github.com/giantswarm/rbac-operator/service/controller/defaultnamespace/key"
)

// EnsureCreated Ensures the ClusterRole 'read-releases'.
//
// Purpose if this role is to enable read permissions (get, list, watch)
// for release resources which are not in a namespace
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	namespace, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if !pkgkey.IsDefaultNamespace(namespace.Name) {
		return nil
	}

	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: pkgkey.ReadReleasesRole,
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

	if err = rbac.CreateOrUpdateClusterRole(r, ctx, role); err != nil {
		return microerror.Mask(err)
	}

	return nil
}
