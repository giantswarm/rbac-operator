package crossplaneauth

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/rbac-operator/service/controller/crossplane/key"
	// rbacv1 "k8s.io/api/rbac/v1"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	// var err error

	cr, err := key.ToClusterRole(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	if err == nil {
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("nfo clusterrole %s", cr.Name))
	}

	return nil
}
