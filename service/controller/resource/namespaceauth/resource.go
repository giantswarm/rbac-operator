package namespaceauth

import (
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
)

const (
	Name = "namespaceauth"

	viewAllRole = "view-all"
)

type NamespaceAuth struct {
	ViewAllTargetGroup string
}

type Config struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	NamespaceAuth NamespaceAuth
}

type Resource struct {
	k8sClient kubernetes.Interface
	logger    micrologger.Logger

	namespaceAuth NamespaceAuth
}

func New(config Config) (*Resource, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.NamespaceAuth.ViewAllTargetGroup == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.NamespaceAuth.ViewAllTargetGroup must not be empty", config)
	}

	r := &Resource{
		k8sClient: config.K8sClient.K8sClient(),
		logger:    config.Logger,

		namespaceAuth: config.NamespaceAuth,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}

func appendUnique(slice, exceptions []string, newElement string) []string {
	for _, e := range slice {
		for _, exception := range exceptions {
			if e == newElement || newElement == exception {
				return slice
			}
		}
	}
	return append(slice, newElement)
}

func newViewAllRole(resources []*metav1.APIResourceList) (*rbacv1.Role, error) {
	var resourceNamesNamespace, apiGroupsNamespace []string
	{
		apiGroupExceptions := []string{}
		resourceExceptions := []string{}

		for _, resource := range resources {

			gv, err := schema.ParseGroupVersion(resource.GroupVersion)
			if err != nil {
				return nil, microerror.Mask(err)
			}

			apiGroupsNamespace = appendUnique(apiGroupsNamespace, apiGroupExceptions, gv.Group)

			for _, apiResource := range resource.APIResources {
				if apiResource.Namespaced {
					resourceNamesNamespace = appendUnique(resourceNamesNamespace, resourceExceptions, apiResource.Name)
				}
			}
		}
	}

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: viewAllRole,
		},
		Rules: []rbacv1.PolicyRule{
			rbacv1.PolicyRule{
				APIGroups: apiGroupsNamespace,
				Resources: resourceNamesNamespace,
				Verbs:     []string{"get", "list", "watch", "create", "update"},
			},
		},
	}

	return role, nil
}

func newViewAllRoleBinding(targetGroupName string) *rbacv1.RoleBinding {
	roleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: viewAllRole,
		},
		Subjects: []rbacv1.Subject{
			rbacv1.Subject{
				Kind: "User",
				Name: targetGroupName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     viewAllRole,
		},
	}

	return roleBinding
}
