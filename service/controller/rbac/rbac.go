package rbac

import (
	// If your operator watches a CRD import it here.
	// "github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"fmt"

	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/giantswarm/rbac-operator/pkg/label"
	"github.com/giantswarm/rbac-operator/pkg/project"
	"github.com/giantswarm/rbac-operator/service/controller/rbac/resource/namespaceauth"
)

type RBACConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	NamespaceAuth namespaceauth.NamespaceAuth
}

type RBAC struct {
	*controller.Controller
}

func NewRBAC(config RBACConfig) (*RBAC, error) {
	var err error

	resourceSet, err := newRBACResourceSet(config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var namespaceAuthController *controller.Controller
	{

		namespaceSelectorQuery := fmt.Sprintf("%s,%s", label.Cluster, label.Organization)
		namespaceSelector, err := labels.Parse(namespaceSelectorQuery)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		c := controller.Config{
			K8sClient:    config.K8sClient,
			Logger:       config.Logger,
			ResourceSets: []*controller.ResourceSet{resourceSet},
			NewRuntimeObjectFunc: func() runtime.Object {
				return new(corev1.Namespace)
			},
			Selector: namespaceSelector,

			// Name is used to compute finalizer names. This here results in something
			// like operatorkit.giantswarm.io/rbac-operator-RBAC-controller.
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
