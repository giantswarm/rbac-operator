package namespacelabeler

import (
	"github.com/giantswarm/k8sclient/v3/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/resource"
	"github.com/giantswarm/operatorkit/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/resource/wrapper/retryresource"

	"github.com/giantswarm/rbac-operator/service/controller/namespacelabeler/resource/namespacelabel"
)

type namespaceLabelerResourcesConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
}

func newNamespaceLabelerResources(config namespaceLabelerResourcesConfig) ([]resource.Interface, error) {
	var err error

	var namespaceLabelResource resource.Interface
	{
		c := namespacelabel.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
		}

		namespaceLabelResource, err = namespacelabel.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []resource.Interface{
		namespaceLabelResource,
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
