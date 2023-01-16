package crossplane

import (
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v8/pkg/resource"
	"github.com/giantswarm/operatorkit/v8/pkg/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/v8/pkg/resource/wrapper/retryresource"

	"github.com/giantswarm/rbac-operator/service/controller/crossplane/resource/crossplaneauth"
	"github.com/giantswarm/rbac-operator/service/internal/accessgroup"
)

type crossplaneResourcesConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	CustomerAdminGroups                 []accessgroup.AccessGroup
	CrossplaneBindTriggeringClusterRole string
}

func newCrossplaneResources(config crossplaneResourcesConfig) ([]resource.Interface, error) {
	var err error

	var crossplaneClusterRoleBinderResource resource.Interface
	{
		c := crossplaneauth.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,

			CustomerAdminGroups:                 config.CustomerAdminGroups,
			CrossplaneBindTriggeringClusterRole: config.CrossplaneBindTriggeringClusterRole,
		}

		crossplaneClusterRoleBinderResource, err = crossplaneauth.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []resource.Interface{
		crossplaneClusterRoleBinderResource,
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
