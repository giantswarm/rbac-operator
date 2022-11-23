package rbac

import (
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v7/pkg/controller"
	"github.com/giantswarm/operatorkit/v7/pkg/resource"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/rbac-operator/service/internal/accessgroup"

	"github.com/giantswarm/rbac-operator/pkg/project"
)

type RBACConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	WriteAllCustomerGroups []accessgroup.AccessGroup
}

type RBAC struct {
	*controller.Controller
}

func NewRBAC(config RBACConfig) (*RBAC, error) {
	var err error

	var resources []resource.Interface
	{
		c := rbacResourcesConfig(config)

		resources, err = newRBACResources(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var namespaceAuthController *controller.Controller
	{
		c := controller.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
			NewRuntimeObjectFunc: func() client.Object {
				return new(corev1.Namespace)
			},
			Resources: resources,

			// Name is used to compute finalizer names. This here results in something
			// like operatorkit.giantswarm.io/rbac-operator-rbac-controller.
			Name: project.Name() + "-rbac-controller",
		}

		namespaceAuthController, err = controller.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	c := &RBAC{
		Controller: namespaceAuthController,
	}

	return c, nil
}
