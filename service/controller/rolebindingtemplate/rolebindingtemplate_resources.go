package rolebindingtemplate

import (
	"github.com/giantswarm/k8sclient/v8/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v7/pkg/resource"
	"github.com/giantswarm/operatorkit/v7/pkg/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/v7/pkg/resource/wrapper/retryresource"

	"github.com/giantswarm/rbac-operator/service/controller/rolebindingtemplate/resource/rolebinding"
)

type roleBindingTemplateResourcesConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
}

func newRoleBindingTemplateResources(config roleBindingTemplateResourcesConfig) ([]resource.Interface, error) {
	var err error

	var roleBindingResource resource.Interface
	{
		c := rolebinding.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
		}

		roleBindingResource, err = rolebinding.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []resource.Interface{
		roleBindingResource,
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
