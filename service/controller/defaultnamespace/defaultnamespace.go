package defaultnamespace

import (
	"context"
	"fmt"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v7/pkg/controller"
	"github.com/giantswarm/operatorkit/v7/pkg/resource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/pkg/project"
	"github.com/giantswarm/rbac-operator/service/internal/accessgroup"
)

type DefaultNamespaceConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	CustomerAdminGroups  []accessgroup.AccessGroup
	CustomerReaderGroups []accessgroup.AccessGroup
	GSAdminGroups        []accessgroup.AccessGroup

	Provider string
}
type DefaultNamespace struct {
	Controller *controller.Controller
	k8sClient  k8sclient.Interface
	resources  []resource.Interface
}

func NewDefaultNamespace(config DefaultNamespaceConfig) (*DefaultNamespace, error) {

	var err error

	var resources []resource.Interface
	{
		c := defaultNamespaceBootstrapResourcesConfig(config)

		resources, err = newDefaultNamespaceResources(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var defaultNamespaceController *controller.Controller
	{
		//kubernetes.io/metadata.name: default
		//selector, err := labels.Parse("kubernetes.io/metadata.name: default")
		selector, err := labels.Parse(fmt.Sprintf("%s=%s", pkgkey.NameLabel, pkgkey.DefaultNamespaceName))
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
			Name: project.Name() + "-default-namespace-controller",
		}

		defaultNamespaceController, err = controller.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	c := &DefaultNamespace{
		Controller: defaultNamespaceController,
		k8sClient:  config.K8sClient,
		resources:  resources,
	}

	return c, nil
}

func (c *DefaultNamespace) EnsureResourcesCreated(ctx context.Context) error {
	namespace, err := c.k8sClient.K8sClient().CoreV1().Namespaces().Get(ctx, pkgkey.DefaultNamespaceName, metav1.GetOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	for _, res := range c.resources {
		if err := res.EnsureCreated(ctx, namespace); err != nil {
			return microerror.Mask(err)
		}
	}
	return nil
}
