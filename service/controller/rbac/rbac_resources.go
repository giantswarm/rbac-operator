package rbac

import (
	"github.com/giantswarm/k8sclient/v8/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v7/pkg/resource"
	"github.com/giantswarm/operatorkit/v7/pkg/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/v7/pkg/resource/wrapper/retryresource"

	"github.com/giantswarm/rbac-operator/service/internal/accessgroup"

	"github.com/giantswarm/rbac-operator/service/controller/rbac/resource/automation"
	"github.com/giantswarm/rbac-operator/service/controller/rbac/resource/externalresources"
	"github.com/giantswarm/rbac-operator/service/controller/rbac/resource/namespaceauth"
)

type rbacResourcesConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	WriteAllCustomerGroups []accessgroup.AccessGroup
	ReadAllCustomerGroups  []accessgroup.AccessGroup
}

func newRBACResources(config rbacResourcesConfig) ([]resource.Interface, error) {
	var err error

	var externalResourcesResource resource.Interface
	{
		c := externalresources.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
		}

		externalResourcesResource, err = externalresources.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var automationResource resource.Interface
	{
		c := automation.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
		}

		automationResource, err = automation.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var namespaceAuthResource resource.Interface
	{
		c := namespaceauth.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,

			WriteAllCustomerGroups: config.WriteAllCustomerGroups,
			ReadAllCustomerGroups:  config.ReadAllCustomerGroups,
		}

		namespaceAuthResource, err = namespaceauth.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []resource.Interface{
		automationResource,
		namespaceAuthResource,
		externalResourcesResource,
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
