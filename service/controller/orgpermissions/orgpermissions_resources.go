package orgpermissions

import (
	"github.com/giantswarm/k8sclient/v4/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v2/pkg/resource"
	"github.com/giantswarm/operatorkit/v2/pkg/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/v2/pkg/resource/wrapper/retryresource"

	"github.com/giantswarm/rbac-operator/service/controller/orgpermissions/resource/membership"
)

type orgPermissionsResourcesConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
}

func newOrgPermissionsResources(config orgPermissionsResourcesConfig) ([]resource.Interface, error) {
	var err error

	var membershipResource resource.Interface
	{
		c := membership.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
		}

		membershipResource, err = membership.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []resource.Interface{
		membershipResource,
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
