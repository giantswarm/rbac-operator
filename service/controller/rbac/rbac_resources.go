package rbac

import (
	"context"
	"time"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v8/pkg/resource"
	"github.com/giantswarm/operatorkit/v8/pkg/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/v8/pkg/resource/wrapper/retryresource"
	"github.com/giantswarm/rbac-operator/service/controller/rbac/resource/automation"
	"github.com/giantswarm/rbac-operator/service/controller/rbac/resource/externalresources"
	"github.com/giantswarm/rbac-operator/service/controller/rbac/resource/namespaceauth"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	clusterAdminAssignments = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cluster_admin_role_assignments",
			Help: "Number of cluster-admin role assignments",
		},
		[]string{"group"},
	)
)

func init() {
	prometheus.MustRegister(clusterAdminAssignments)
}

func newRBACResources(config rbacResourcesConfig) ([]resource.Interface, error) {
	var err error

	var externalResourcesResource resource.Interface
	{
		c := externalresources.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
		}

		externalResourcesResource, err = externalresources.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var automationResource resource.Interface
	{
		c := automation.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
		}

		automationResource, err = automation.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var namespaceAuthResource resource.Interface
	{
		c := namespaceauth.Config{
			K8sClient:              config.K8sClient,
			Logger:                 config.Logger,
			WriteAllCustomerGroups: config.WriteAllCustomerGroups,
		}

		namespaceAuthResource, err = namespaceauth.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []resource.Interface{
		automationResource,
		namespaceAuthResource,
		externalResourcesResource,
	}

	{
		c := retryresource.WrapConfig{
			Logger: config.Logger,
		}

		resources, err = retryresource.Wrap(resources, c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	{
		c := metricsresource.WrapConfig{}

		resources, err = metricsresource.Wrap(resources, c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	// Additional custom metrics for cluster-admin assignments
	go func() {
		for {
			// Logic to update the clusterAdminAssignments metric
			updateClusterAdminAssignments(config.K8sClient)
			time.Sleep(1 * time.Minute)
		}
	}()

	return resources, nil
}

func updateClusterAdminAssignments(k8sClient k8sclient.Interface) {
	clusterRoleBindings, err := k8sClient.RbacV1().ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return
	}

	clusterAdminAssignments.Reset()
	for _, crb := range clusterRoleBindings.Items {
		if crb.RoleRef.Name == "cluster-admin" {
			for _, subject := range crb.Subjects {
				if subject.Kind == "Group" {
					clusterAdminAssignments.WithLabelValues(subject.Name).Set(1)
				}
			}
		}
	}
}
