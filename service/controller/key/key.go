package key

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/giantswarm/microerror"
)

const (
	ViewAllRole = "view-all"
)

func NewViewAllRole(resources []*metav1.APIResourceList) (*rbacv1.Role, error) {
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
			Name: ViewAllRole,
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

func NewViewAllRoleBinding(targetGroupName string) *rbacv1.RoleBinding {
	roleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: ViewAllRole,
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
			Name:     ViewAllRole,
		},
	}

	return roleBinding
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
