package crossplane

import (
	"fmt"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	k8smetadata "github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v7/pkg/controller"
	"github.com/giantswarm/operatorkit/v7/pkg/resource"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/rbac-operator/pkg/project"
	"github.com/giantswarm/rbac-operator/service/internal/accessgroup"
)

type CrossplaneNamespaceConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	CustomerAdminGroups                 []accessgroup.AccessGroup
	CrossplaneBindTriggeringClusterRole string
}

type CrossplaneNamespace struct {
	*controller.Controller
}

func NewCrossplaneNamespace(config CrossplaneNamespaceConfig) (*CrossplaneNamespace, error) {
	var err error

	var resources []resource.Interface
	{
		c := crossplaneNamespaceResourcesConfig{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,

			CustomerAdminGroups:                 config.CustomerAdminGroups,
			CrossplaneBindTriggeringClusterRole: config.CrossplaneBindTriggeringClusterRole,
		}

		resources, err = newCrossplaneNamespaceResources(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var namespaceController *controller.Controller
	{
		// Only watch org namespaces (those starting with "org-")
		selector, err := labels.Parse(fmt.Sprintf("%s", k8smetadata.Organization))
		if err != nil {
			return nil, microerror.Mask(err)
		}

		c := controller.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
			NewRuntimeObjectFunc: func() client.Object {
				return new(corev1.Namespace)
			},
			Resources: resources,
			Selector:  selector,

			Name: project.Name() + "-crossplane-namespace-controller",
		}

		namespaceController, err = controller.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	c := &CrossplaneNamespace{
		Controller: namespaceController,
	}

	return c, nil
}
