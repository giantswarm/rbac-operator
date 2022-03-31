package clusternamespace

import (
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v7/pkg/resource"
	"github.com/giantswarm/operatorkit/v7/pkg/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/v7/pkg/resource/wrapper/retryresource"

	"github.com/giantswarm/rbac-operator/service/controller/clusternamespace/resource/clusternamespaceresources"
)

type clusterNamespaceResourcesConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
}

func newClusterNamespaceResources(config clusterNamespaceResourcesConfig) ([]resource.Interface, error) {
	var err error

	var clusterNamespaceResourcesResource resource.Interface
	{
		c := clusternamespaceresources.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
		}

		clusterNamespaceResourcesResource, err = clusternamespaceresources.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []resource.Interface{
		clusterNamespaceResourcesResource,
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
