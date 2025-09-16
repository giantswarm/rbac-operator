package crossplane

import (
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v7/pkg/resource"
	"github.com/giantswarm/operatorkit/v7/pkg/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/v7/pkg/resource/wrapper/retryresource"

	"github.com/giantswarm/rbac-operator/service/controller/crossplane/resource/crossplanenamespace"
	"github.com/giantswarm/rbac-operator/service/internal/accessgroup"
)

type crossplaneNamespaceResourcesConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	CustomerAdminGroups                 []accessgroup.AccessGroup
	CrossplaneBindTriggeringClusterRole string
}

func newCrossplaneNamespaceResources(config crossplaneNamespaceResourcesConfig) ([]resource.Interface, error) {
	var err error

	var crossplaneNamespaceResource resource.Interface
	{
		c := crossplanenamespace.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,

			CustomerAdminGroups:                 config.CustomerAdminGroups,
			CrossplaneBindTriggeringClusterRole: config.CrossplaneBindTriggeringClusterRole,
		}

		crossplaneNamespaceResource, err = crossplanenamespace.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []resource.Interface{
		crossplaneNamespaceResource,
	}

	{
		c := retryresource.WrapConfig{
			Logger: config.Logger,
		}

		resources, err = retryresource.Wrap(resources, c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	{
		c := metricsresource.WrapConfig{}

		resources, err = metricsresource.Wrap(resources, c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return resources, nil
}
