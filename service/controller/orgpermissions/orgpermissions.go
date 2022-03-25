package orgpermissions

import (
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v7/pkg/controller"
	"github.com/giantswarm/operatorkit/v7/pkg/resource"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/rbac-operator/pkg/project"
)

type OrgPermissionsConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
}

type OrgPermissions struct {
	*controller.Controller
}

func NewOrgPermissions(config OrgPermissionsConfig) (*OrgPermissions, error) {
	var err error

	var resources []resource.Interface
	{
		c := orgPermissionsResourcesConfig(config)

		resources, err = newOrgPermissionsResources(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var organizationPermissionsController *controller.Controller
	{
		c := controller.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
			NewRuntimeObjectFunc: func() client.Object {
				return new(rbacv1.RoleBinding)
			},
			Resources: resources,

			Name: project.Name() + "-orgpermissions-controller",
		}

		organizationPermissionsController, err = controller.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	c := &OrgPermissions{
		Controller: organizationPermissionsController,
	}

	return c, nil
}
