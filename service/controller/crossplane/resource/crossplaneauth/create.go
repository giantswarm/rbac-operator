package crossplaneauth

import (
	"context"
	// "fmt"
	"github.com/giantswarm/microerror"
	// "github.com/giantswarm/rbac-operator/service/controller/rbac/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	var err error

	// _, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	// r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating rolebinding %#q in namespace %s", roleBinding.Name, ns.Name))

	return nil
}
