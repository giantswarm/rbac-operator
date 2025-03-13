package namespaceauth

import (
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/rbac-operator/service/internal/accessgroup"

	"k8s.io/client-go/kubernetes"
)

const (
	Name = "namespaceauth"
)

type Config struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	WriteAllCustomerGroups []accessgroup.AccessGroup
	ReadAllCustomerGroups  []accessgroup.AccessGroup
}

type Resource struct {
	k8sClient kubernetes.Interface
	logger    micrologger.Logger

	writeAllCustomerGroups []accessgroup.AccessGroup
	ReadAllCustomerGroups  []accessgroup.AccessGroup
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

		writeAllCustomerGroups: config.WriteAllCustomerGroups,
		ReadAllCustomerGroups:  config.ReadAllCustomerGroups,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}
