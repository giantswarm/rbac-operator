package rbac

import (
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	"github.com/giantswarm/operatorkit/resource"
	"github.com/giantswarm/operatorkit/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/resource/wrapper/retryresource"

	"github.com/giantswarm/rbac-operator/service/controller/rbac/resource/namespaceauth"
)

type RBACResourceSetConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	NamespaceAuth namespaceauth.NamespaceAuth
}

func newRBACResourceSet(config RBACConfig) (*controller.ResourceSet, error) {
	var err error

	var namespaceAuthResource resource.Interface
	{
		c := namespaceauth.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,

			NamespaceAuth: config.NamespaceAuth,
		}

		namespaceAuthResource, err = namespaceauth.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []resource.Interface{
		namespaceAuthResource,
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

	handlesFunc := func(obj interface{}) bool {
		return true
	}

	var resourceSet *controller.ResourceSet
	{
		c := controller.ResourceSetConfig{
			Handles:   handlesFunc,
			Logger:    config.Logger,
			Resources: resources,
		}

		resourceSet, err = controller.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return resourceSet, nil
}
