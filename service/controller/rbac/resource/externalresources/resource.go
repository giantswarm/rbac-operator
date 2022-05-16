// externalresources package is responsible for managing RBAC resources
// that grant those with access to an organization namespace access to
// namespaces belonging to the organizations clusters
package externalresources

import (
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	Name = "externalresources"
)

type Config struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
}

type Resource struct {
	k8sClient kubernetes.Interface
	logger    micrologger.Logger
}

func (r Resource) K8sClient() kubernetes.Interface {
	return r.k8sClient
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
		k8sClient: config.K8sClient.K8sClient(),
		logger:    config.Logger,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}

func getUniqueSubjectsWithClusterRoleRef(orgRoleBindings *rbacv1.RoleBindingList, clusterRole string) []rbacv1.Subject {
	readAllSubjects := make(map[string]rbacv1.Subject)
	for _, roleBinding := range orgRoleBindings.Items {
		if roleBindingReferencesClusterRole(roleBinding, clusterRole) && roleBindingHasSubject(roleBinding) {
			for _, subject := range roleBinding.Subjects {
				readAllSubjects[subject.Kind+subject.Name] = subject
			}
		}
	}
	var subjects []rbacv1.Subject
	for _, value := range readAllSubjects {
		subjects = append(subjects, value)
	}
	return subjects
}

func getUniqueSubjects(orgRoleBindings *rbacv1.RoleBindingList) []rbacv1.Subject {
	uniqueSubjects := make(map[string]rbacv1.Subject)
	for _, roleBinding := range orgRoleBindings.Items {
		if roleBindingHasReference(roleBinding) && roleBindingHasSubject(roleBinding) {
			for _, subject := range roleBinding.Subjects {
				uniqueSubjects[subject.Kind+subject.Name] = subject
			}
		}
	}
	var subjects []rbacv1.Subject
	for _, value := range uniqueSubjects {
		subjects = append(subjects, value)
	}
	return subjects
}

func roleBindingHasReference(roleBinding rbacv1.RoleBinding) bool {
	if roleBinding.RoleRef.Name != "" && roleBinding.RoleRef.Kind != "" {
		return true
	}
	return false
}

func roleBindingHasSubject(roleBinding rbacv1.RoleBinding) bool {
	for _, subject := range roleBinding.Subjects {
		if (subject.Kind == "Group" || subject.Kind == "User") && subject.Name != "" {
			return true
		}
	}
	return false
}

func roleBindingReferencesClusterRole(roleBinding rbacv1.RoleBinding, roleName string) bool {
	if roleBinding.RoleRef.Name == roleName && roleBinding.RoleRef.Kind == "ClusterRole" {
		return true
	}
	return false
}
