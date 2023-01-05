package crossplaneauth

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/rbac-operator/service/controller/rbac/key"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	_, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
