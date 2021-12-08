package rbac

import (
	// If your operator watches a CRD import it here.
	// "github.com/giantswarm/apiextensions/v2/pkg/apis/application/v1alpha1"

	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v5/pkg/controller"
	"github.com/giantswarm/operatorkit/v5/pkg/resource"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	legacylabel "github.com/giantswarm/rbac-operator/pkg/label"
	"github.com/giantswarm/rbac-operator/pkg/project"
)

type RBACConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	WriteAllCustomerGroup string
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
		selector := newWrongSelector(func(labels labels.Labels) bool {
			return labels.Has(label.Organization) || labels.Has(legacylabel.LegacyCustomer)
		})

		c := controller.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
			NewRuntimeObjectFunc: func() runtime.Object {
				return new(corev1.Namespace)
			},
			Resources: resources,
			Selector:  selector,

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
