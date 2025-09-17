// crossplanenamespace package handles updating the crossplane-edit ClusterRoleBinding
// when new org namespaces are created or deleted, ensuring their automation
// ServiceAccounts have the necessary crossplane permissions.
package crossplanenamespace

import (
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/rbac-operator/service/internal/accessgroup"

	"k8s.io/client-go/kubernetes"
)

const (
	Name = "crossplanenamespace"
)

type Config struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	CustomerAdminGroups                 []accessgroup.AccessGroup
	CrossplaneBindTriggeringClusterRole string
}

type Resource struct {
	k8sClient                           k8sclient.Interface
	logger                              micrologger.Logger
	customerAdminGroups                 []accessgroup.AccessGroup
	crossplaneBindTriggeringClusterRole string
}

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

func New(config Config) (*Resource, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		k8sClient:                           config.K8sClient,
		logger:                              config.Logger,
		customerAdminGroups:                 config.CustomerAdminGroups,
		crossplaneBindTriggeringClusterRole: config.CrossplaneBindTriggeringClusterRole,
	}

	return r, nil
}

func (r Resource) K8sClient() kubernetes.Interface {
	return r.k8sClient.K8sClient()
}

func (r Resource) Logger() micrologger.Logger {
	return r.logger
}

func (r *Resource) Name() string {
	return Name
}
