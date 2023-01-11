package crossplaneauth

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/rbac-operator/service/controller/crossplane/key"
	pkgrbac "github.com/giantswarm/rbac-operator/pkg/rbac"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	cr, err := key.ToClusterRole(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	if cr.Name != key.CrossplaneEditClusterRole() {
		return nil
	}

	// Delete ClusterRoleBinding for customer's admin access to crossplane-edit ClusterRole
	err = pkgrbac.DeleteClusterRoleBinding(r, ctx, key.GetClusterRoleBindingName())
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}
