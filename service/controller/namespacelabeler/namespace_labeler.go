package namespacelabeler

import (
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
)

type NamespaceLabelerConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
}

type NamespaceLabeler struct {
	*controller.Controller
}

func NewNamespaceLabeler(config NamespaceLabelerConfig) (*NamespaceLabeler, error) {
	var err error

	resourceSet, err := NewNamespaceLabelerResourceSet(config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var namespaceLabelerController *controller.Controller
	{

		namespaceSelectorQuery := fmt.Sprintf("%s=,%s=,%s!=,%s!=", label.LegacyCluster, label.LegacyCustomer, label.Cluster, label.Organization)
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

			Name: project.Name() + "-namespace-labeler-controller",
		}

		namespaceLabelerController, err = controller.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	c := &NamespaceLabeler{
		Controller: namespaceLabelerController,
	}

	return c, nil
}
