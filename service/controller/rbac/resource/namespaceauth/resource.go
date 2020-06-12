package namespaceauth

import (
	"github.com/giantswarm/k8sclient/v3/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/rbac-operator/pkg/label"
	"github.com/giantswarm/rbac-operator/pkg/project"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
)

const (
	Name = "namespaceauth"

	automationServiceAccountName      = "automation"
	automationServiceAccountNamespace = "global"
	tenantAdminRoleName               = "tenant-admin"
	viewAllRoleName                   = "view-all"
)

type Config struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	NamespaceAuth NamespaceAuth
}

type NamespaceAuth struct {
	ViewAllTargetGroup     string
	TenantAdminTargetGroup string
}

type Resource struct {
	k8sClient kubernetes.Interface
	logger    micrologger.Logger

	namespaceAuth NamespaceAuth
}

type role struct {
	name        string
	verbs       []string
	targetGroup string
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
	if config.NamespaceAuth.TenantAdminTargetGroup == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.NamespaceAuth.TenantAdminTargetGroup must not be empty", config)
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

func appendUnique(slice []string, newElement string) []string {
	for _, e := range slice {
		if e == newElement {
			return slice
		}
	}
	return append(slice, newElement)
}

func newInternalRole(name, targetGroup string, verbs []string) role {
	return role{
		name:        name,
		targetGroup: targetGroup,
		verbs:       verbs,
	}
}

func newRole(name string, resources []*metav1.APIResourceList, verbs []string) (*rbacv1.Role, error) {
	var resourceNamesNamespace, apiGroupsNamespace []string
	{
		for _, resource := range resources {

			gv, err := schema.ParseGroupVersion(resource.GroupVersion)
			if err != nil {
				return nil, microerror.Mask(err)
			}

			apiGroupsNamespace = appendUnique(apiGroupsNamespace, gv.Group)

			for _, apiResource := range resource.APIResources {
				if apiResource.Namespaced {
					resourceNamesNamespace = appendUnique(resourceNamesNamespace, apiResource.Name)
				}
			}
		}
	}

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Rules: []rbacv1.PolicyRule{
			rbacv1.PolicyRule{
				APIGroups: apiGroupsNamespace,
				Resources: resourceNamesNamespace,
				Verbs:     verbs,
			},
		},
	}

	return role, nil
}

func newGroupRoleBinding(roleBindingName, targetGroupName, targetRoleName string) *rbacv1.RoleBinding {
	roleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: roleBindingName,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Subjects: []rbacv1.Subject{
			rbacv1.Subject{
				Kind: "Group",
				Name: targetGroupName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     targetRoleName,
		},
	}

	return roleBinding
}

func newServiceAccountRoleBinding(roleBindingName, serviceAccountName, serviceAccountNamespace, targetRoleName string) *rbacv1.RoleBinding {
	roleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: roleBindingName,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Subjects: []rbacv1.Subject{
			rbacv1.Subject{
				Kind:      "ServiceAccount",
				Name:      serviceAccountName,
				Namespace: serviceAccountNamespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     targetRoleName,
		},
	}

	return roleBinding
}
