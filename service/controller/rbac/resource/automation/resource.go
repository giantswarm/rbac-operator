// automation package is responsible for setting up RBAC required by
// customer-facing Flux, operating in flux-system namespace.
// This includes creating RoleBindings to upstream Flux roles in Organization
// namespaces, RoleBindings allowing default Flux ServiceAccounts access to
// said namespaces, as well as creating an `automation` ServiceAccount in every
// Organization namespace, with enough permissions to reconcile Flux resources
// there.
package automation

import (
	"github.com/giantswarm/k8sclient/v8/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"k8s.io/client-go/kubernetes"
)

const (
	Name = "automation"
)

type Config struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
}

type Resource struct {
	k8sClient kubernetes.Interface
	logger    micrologger.Logger
}

// K8sClient returns the Kubernetes client, so the resource can be passed to
// the helpers in pkg/rbac.
func (r Resource) K8sClient() kubernetes.Interface {
	return r.k8sClient
}

// Logger returns the logger, so the resource can be passed to the helpers in
// pkg/rbac.
func (r Resource) Logger() micrologger.Logger {
	return r.logger
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
