package usergroups

import (
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/rbac-operator/service/internal/accessgroup"
)

const (
	Name = "usergroups"
)

type Config struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	CustomerAdminGroups  []accessgroup.AccessGroup
	CustomerReaderGroups []accessgroup.AccessGroup
	GSAdminGroups        []accessgroup.AccessGroup
}

type Resource struct {
	k8sClient k8sclient.Interface
	logger    micrologger.Logger

	customerAdminGroups  []accessgroup.AccessGroup
	customerReaderGroups []accessgroup.AccessGroup
	gsAdminGroups        []accessgroup.AccessGroup
}

func (r Resource) K8sClient() kubernetes.Interface {
	return r.k8sClient.K8sClient()
}

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
	if !accessgroup.ValidateGroups(config.GSAdminGroups) {
		return nil, microerror.Maskf(invalidConfigError, "%T.GSAdminGroups must not be empty", config)
	}

	r := &Resource{
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		customerAdminGroups:  config.CustomerAdminGroups,
		customerReaderGroups: config.CustomerReaderGroups,
		gsAdminGroups:        config.GSAdminGroups,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}
