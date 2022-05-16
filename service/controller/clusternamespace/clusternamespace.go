package clusternamespace

import (
	"fmt"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v7/pkg/controller"
	"github.com/giantswarm/operatorkit/v7/pkg/resource"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/rbac-operator/pkg/project"
)

type ClusterNamespaceConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
}
type ClusterNamespace struct {
	*controller.Controller
}

func NewClusterNamespace(config ClusterNamespaceConfig) (*ClusterNamespace, error) {
	var err error

	var resources []resource.Interface
	{
		c := clusterNamespaceResourcesConfig(config)

		resources, err = newClusterNamespaceResources(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var clusterNamespaceController *controller.Controller
	{
		selector, err := labels.Parse(fmt.Sprintf("%s,%s", label.Organization, label.Cluster))
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

			// Name is used to compute finalizer names. This here results in something
			// like operatorkit.giantswarm.io/rbac-operator-cluster-namespace-controller.
			Name: project.Name() + "-cluster-namespace-controller",
		}

		clusterNamespaceController, err = controller.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	c := &ClusterNamespace{
		Controller: clusterNamespaceController,
	}

	return c, nil
}
