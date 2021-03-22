package orgpermissions

import (
	"github.com/giantswarm/apiextensions/v3/pkg/label"

	"github.com/giantswarm/k8sclient/v4/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v2/pkg/controller"
	"github.com/giantswarm/operatorkit/v2/pkg/resource"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"

	legacylabel "github.com/giantswarm/rbac-operator/pkg/label"
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
		selector := controller.NewSelector(func(labels controller.Labels) bool {
			return labels.Has(label.Organization) || labels.Has(legacylabel.LegacyCustomer)
		})

		c := controller.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
			NewRuntimeObjectFunc: func() runtime.Object {
				return new(rbacv1.RoleBinding)
			},
			Resources: resources,
			Selector:  selector,

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
