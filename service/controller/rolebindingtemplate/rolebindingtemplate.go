package rolebindingtemplate

import (
	"github.com/giantswarm/k8sclient/v8/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v7/pkg/controller"
	"github.com/giantswarm/operatorkit/v7/pkg/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/rbac-operator/api/v1alpha1"
	"github.com/giantswarm/rbac-operator/pkg/project"
)

type RoleBindingTemplateConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
}

type RoleBindingTemplate struct {
	*controller.Controller
}

func NewRoleBindingTemplate(config RoleBindingTemplateConfig) (*RoleBindingTemplate, error) {
	var err error

	var resources []resource.Interface
	{
		c := roleBindingTemplateResourcesConfig(config)

		resources, err = newRoleBindingTemplateResources(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var roleBindingTemplateController *controller.Controller
	{
		c := controller.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
			NewRuntimeObjectFunc: func() client.Object {
				return new(v1alpha1.RoleBindingTemplate)
			},
			Resources: resources,

			// Name is used to compute finalizer names. This here results in something
			// like operatorkit.giantswarm.io/rbac-operator-rbac-controller.
			Name: project.Name() + "-rolebindingtemplate-controller",
		}

		roleBindingTemplateController, err = controller.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	c := &RoleBindingTemplate{
		Controller: roleBindingTemplateController,
	}

	return c, nil
}
