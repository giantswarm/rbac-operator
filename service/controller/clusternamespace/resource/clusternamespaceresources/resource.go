// clusternamespaceresources package is responsible for managing RBAC resources
// that grant those with access to an organization namespace access to
// namespaces belonging to the organizations clusters
package clusternamespaceresources

import (
	"reflect"

	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Name = "clusternamespaceresources"
)

type Config struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
}

type Resource struct {
	k8sClient k8sclient.Interface
	logger    micrologger.Logger
}

func (r Resource) K8sClient() kubernetes.Interface {
	return r.k8sClient.K8sClient()
}

func (r Resource) Logger() micrologger.Logger {
	return r.logger
}

func New(config Config) (*Resource, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		k8sClient: config.K8sClient,
		logger:    config.Logger,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}

func roleBindingReferencesClusterRole(roleBinding rbacv1.RoleBinding, roleName string) bool {
	if roleBinding.RoleRef.Name == roleName && roleBinding.RoleRef.Kind == "ClusterRole" {
		return true
	}
	return false
}

// RoleBinding needs an update with the list of subjects has changed
func roleBindingNeedsUpdate(desiredRoleBinding, existingRoleBinding *rbacv1.RoleBinding) bool {
	if len(existingRoleBinding.Subjects) < 1 {
		return true
	}

	if !reflect.DeepEqual(desiredRoleBinding.Subjects, existingRoleBinding.Subjects) {
		return true
	}

	return false
}

// Role needs an update if the rules have changed
func roleNeedsUpdate(desiredRole, existingRole *rbacv1.Role) bool {
	if len(existingRole.Rules) < 1 {
		return true
	}

	if !reflect.DeepEqual(desiredRole.Rules, existingRole.Rules) {
		return true
	}

	return false
}

func getRules(resources []metav1.APIResource, verbs []string) []rbacv1.PolicyRule {
	var policyRules []rbacv1.PolicyRule
	for _, resource := range resources {
		policyRule := rbacv1.PolicyRule{
			APIGroups: []string{resource.Group},
			Resources: []string{resource.Name},
			Verbs:     verbs,
		}
		policyRules = append(policyRules, policyRule)
	}
	return policyRules
}
