package crossplane

import (
	"context"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v7/pkg/controller"
	"github.com/giantswarm/operatorkit/v7/pkg/resource"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/rbac-operator/pkg/project"
	"github.com/giantswarm/rbac-operator/service/internal/accessgroup"
)

type CrossplaneConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	CustomerAdminGroups                 []accessgroup.AccessGroup
	CustomerReaderGroups                []accessgroup.AccessGroup
	CrossplaneBindTriggeringClusterRole string
}

type Crossplane struct {
	ClusterRoleController *controller.Controller
	NamespaceController   *controller.Controller
}

func NewCrossplane(config CrossplaneConfig) (*Crossplane, error) {
	var err error

	var resources []resource.Interface
	{
		c := crossplaneResourcesConfig(config)

		resources, err = newCrossplaneResources(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var clusterRoleAuthController *controller.Controller
	{
		c := controller.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
			NewRuntimeObjectFunc: func() client.Object {
				return new(rbacv1.ClusterRole)
			},
			Resources: resources,

			Name: project.Name() + "-crossplane-controller",
		}

		clusterRoleAuthController, err = controller.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var namespaceController *controller.Controller
	{
		crossplaneNamespace, err := NewCrossplaneNamespace(CrossplaneNamespaceConfig{
			K8sClient:                           config.K8sClient,
			Logger:                              config.Logger,
			CustomerAdminGroups:                 config.CustomerAdminGroups,
			CrossplaneBindTriggeringClusterRole: config.CrossplaneBindTriggeringClusterRole,
		})
		if err != nil {
			return nil, microerror.Mask(err)
		}
		namespaceController = crossplaneNamespace.Controller
	}

	c := &Crossplane{
		ClusterRoleController: clusterRoleAuthController,
		NamespaceController:   namespaceController,
	}

	return c, nil
}

func (c *Crossplane) Boot(ctx context.Context) error {
	go c.ClusterRoleController.Boot(ctx)
	go c.NamespaceController.Boot(ctx)
	return nil
}
