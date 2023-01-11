// crossplane package is responsible for setting up RBAC required by
// crossplane's rbac-manager. The rbac-manager creates all the necessary
// (Cluster)Roles for crossplane resources, but doesn't bind them to Users
// or Groups. This controller is responsible for that.
package crossplaneauth

import (
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/rbac-operator/service/internal/accessgroup"

	"k8s.io/client-go/kubernetes"
)

const (
	Name = "crossplaneauth"
)

type Config struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	CustomerAdminGroups []accessgroup.AccessGroup
}

type Resource struct {
	k8sClient kubernetes.Interface
	logger    micrologger.Logger
	customerAdminGroups []accessgroup.AccessGroup
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
		customerAdminGroups: config.CustomerAdminGroups,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}
