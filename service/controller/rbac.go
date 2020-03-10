package controller

import (
	// If your operator watches a CRD import it here.
	// "github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/giantswarm/rbac-operator/pkg/project"
	"github.com/giantswarm/rbac-operator/service/controller/resource/namespaceauth"
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

	resourceSets, err := newRBACResourceSets(config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var operatorkitController *controller.Controller
	{
		var namespaceLabelValues []string
		namespaceSelector := labels.NewSelector()
		clusterLabelRequirement, err := labels.NewRequirement(nsClusterLabel, selection.Exists, namespaceLabelValues)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		namespaceSelector.Add(*clusterLabelRequirement)
		orgLabelRequirement, err := labels.NewRequirement(nsOrgLabel, selection.Exists, namespaceLabelValues)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		namespaceSelector.Add(*orgLabelRequirement)

		c := controller.Config{
			K8sClient:    config.K8sClient,
			Logger:       config.Logger,
			ResourceSets: resourceSets,
			NewRuntimeObjectFunc: func() runtime.Object {
				return new(corev1.Namespace)
			},
			Selector: namespaceSelector,

			// Name is used to compute finalizer names. This here results in something
			// like operatorkit.giantswarm.io/rbac-operator-RBAC-controller.
			Name: project.Name() + "-rbac-controller",
		}

		operatorkitController, err = controller.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	c := &RBAC{
		Controller: operatorkitController,
	}

	return c, nil
}

func newRBACResourceSets(config RBACConfig) ([]*controller.ResourceSet, error) {
	var err error

	var resourceSet *controller.ResourceSet
	{
		c := RBACResourceSetConfig{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,

			NamespaceAuth: config.NamespaceAuth,
		}

		resourceSet, err = newRBACResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resourceSets := []*controller.ResourceSet{
		resourceSet,
	}

	return resourceSets, nil
}
