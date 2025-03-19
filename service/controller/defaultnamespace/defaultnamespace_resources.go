package defaultnamespace

import (
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v8/pkg/resource"
	"github.com/giantswarm/operatorkit/v8/pkg/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/v8/pkg/resource/wrapper/retryresource"

	"github.com/giantswarm/rbac-operator/service/controller/defaultnamespace/resource/automationsa"
	"github.com/giantswarm/rbac-operator/service/controller/defaultnamespace/resource/catalog"
	"github.com/giantswarm/rbac-operator/service/controller/defaultnamespace/resource/clusternamespace"
	"github.com/giantswarm/rbac-operator/service/controller/defaultnamespace/resource/clusterroles"
	"github.com/giantswarm/rbac-operator/service/controller/defaultnamespace/resource/fluxauth"
	"github.com/giantswarm/rbac-operator/service/controller/defaultnamespace/resource/releases"
	"github.com/giantswarm/rbac-operator/service/controller/defaultnamespace/resource/usergroups"
	"github.com/giantswarm/rbac-operator/service/internal/accessgroup"
)

type defaultNamespaceBootstrapResourcesConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	CustomerAdminGroups  []accessgroup.AccessGroup
	CustomerReaderGroups []accessgroup.AccessGroup
	GSAdminGroups        []accessgroup.AccessGroup
}

func newDefaultNamespaceResources(config defaultNamespaceBootstrapResourcesConfig) ([]resource.Interface, error) {
	var err error

	var clusterRolesResource resource.Interface
	{
		c := clusterroles.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
		}

		clusterRolesResource, err = clusterroles.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var automationSAResource resource.Interface
	{
		c := automationsa.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
		}

		automationSAResource, err = automationsa.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var userGroupsResource resource.Interface
	{
		c := usergroups.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,

			CustomerAdminGroups:  config.CustomerAdminGroups,
			CustomerReaderGroups: config.CustomerReaderGroups,
			GSAdminGroups:        config.GSAdminGroups,
		}

		userGroupsResource, err = usergroups.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var releasesResource resource.Interface
	{
		c := releases.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
		}

		releasesResource, err = releases.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var catalogResource resource.Interface
	{
		c := catalog.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
		}

		catalogResource, err = catalog.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var clusterNamespaceResource resource.Interface
	{
		c := clusternamespace.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
		}

		clusterNamespaceResource, err = clusternamespace.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var fluxAuthResource resource.Interface
	{
		c := fluxauth.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
		}

		fluxAuthResource, err = fluxauth.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []resource.Interface{
		clusterRolesResource,
		automationSAResource,
		userGroupsResource,
		releasesResource,
		catalogResource,
		clusterNamespaceResource,
		fluxAuthResource,
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
