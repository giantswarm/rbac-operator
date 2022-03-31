// fluxauth package is responsible for setting up RBAC required by
// customer-facing Flux, operating in flux-system namespace.
// This includes creating RoleBindings to upstream Flux roles in Organization
// namespaces, RoleBindings allowing default Flux ServiceAccounts access to
// said namespaces, as well as creating an `automation` ServiceAccount in every
// Organization namespace, with enough permissions to reconcile Flux resources
// there.
package fluxauth

import (
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"k8s.io/client-go/kubernetes"
)

const (
	Name = "fluxauth"
)

type Config struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
}

type Resource struct {
	k8sClient kubernetes.Interface
	logger    micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		k8sClient: config.K8sClient.K8sClient(),
		logger:    config.Logger,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}
