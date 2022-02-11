package orgcluster

import (
	// If your operator watches a CRD import it here.
	// "github.com/giantswarm/apiextensions/v2/pkg/apis/application/v1alpha1"

	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v4/pkg/controller"
	"github.com/giantswarm/operatorkit/v4/pkg/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	cluster "sigs.k8s.io/cluster-api/api/v1alpha3"

	"github.com/giantswarm/rbac-operator/pkg/project"
)

type OrgClusterConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
}
type OrgCluster struct {
	*controller.Controller
}

func NewOrgCluster(config OrgClusterConfig) (*OrgCluster, error) {
	var err error

	var resources []resource.Interface
	{
		c := orgClusterResourcesConfig(config)

		resources, err = newOrgClusterResources(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var orgClusterController *controller.Controller
	{
		selector := newSelector(func(labels labels.Labels) bool {
			return labels.Has(label.Organization)
		})

		c := controller.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
			NewRuntimeObjectFunc: func() runtime.Object {
				return new(cluster.Cluster)
			},
			Resources: resources,
			Selector:  selector,

			// Name is used to compute finalizer names. This here results in something
			// like operatorkit.giantswarm.io/rbac-operator-org-cluster-controller.
			Name: project.Name() + "-org-cluster-controller",
		}

		orgClusterController, err = controller.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	c := &OrgCluster{
		Controller: orgClusterController,
	}

	return c, nil
}
