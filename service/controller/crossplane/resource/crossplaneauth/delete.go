package crossplaneauth

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/rbac-operator/service/controller/crossplane/key"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	_, err := key.ToClusterRole(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
