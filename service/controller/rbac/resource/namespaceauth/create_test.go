package namespaceauth

import (
	"context"
	"reflect"
	"testing"

	"github.com/giantswarm/k8sclient/v8/pkg/k8sclient"
	"github.com/giantswarm/k8sclient/v8/pkg/k8sclienttest"
	"github.com/giantswarm/micrologger/microloggertest"
	security "github.com/giantswarm/organization-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/giantswarm/rbac-operator/service/internal/accessgroup"
	"github.com/giantswarm/rbac-operator/service/test"

	"k8s.io/client-go/kubernetes/scheme"
)

func Test_EnsureCreated(t *testing.T) {

	testCases := []struct {
		name                string
		orgNamespace        *v1.Namespace
		existingResources   []runtime.Object
		customerAdminGroups []accessgroup.AccessGroup
		expectedClusterRole *rbacv1.ClusterRole
		expectedRoleBinding *rbacv1.RoleBinding
	}{
		{
			name:         "case 0: Create a new role binding in case it does not exist",
			orgNamespace: test.NewOrgNamespace("customer"),
			customerAdminGroups: []accessgroup.AccessGroup{
				{Name: "customer:giantswarm:Employees"},
			},
			expectedRoleBinding: test.NewRoleBinding("write-all-customer-group", "org-customer", map[string]string{
				"kind": "ClusterRole",
				"name": "cluster-admin",
			}, []rbacv1.Subject{
				{Kind: "Group", Name: "customer:giantswarm:Employees"},
			}),
		},
		{
			name:         "case 1: Replace subjects in an existing role binding",
			orgNamespace: test.NewOrgNamespace("customer"),
			existingResources: []runtime.Object{
				test.NewRoleBinding("write-all-customer-group", "org-customer", map[string]string{
					"kind": "ClusterRole",
					"name": "cluster-admin",
				}, []rbacv1.Subject{
					{Kind: "Group", Name: "customer:acme:Employees"},
				}),
			},
			customerAdminGroups: []accessgroup.AccessGroup{
				{Name: "customer:giantswarm:Employees"},
			},
			expectedRoleBinding: test.NewRoleBinding("write-all-customer-group", "org-customer", map[string]string{
				"kind": "ClusterRole",
				"name": "cluster-admin",
			}, []rbacv1.Subject{
				{Kind: "Group", Name: "customer:giantswarm:Employees"},
			}),
		},
		{
			name:         "case 2: Add subjects to an existing role binding",
			orgNamespace: test.NewOrgNamespace("customer"),
			existingResources: []runtime.Object{
				test.NewRoleBinding("write-all-customer-group", "org-customer", map[string]string{
					"kind": "ClusterRole",
					"name": "cluster-admin",
				}, []rbacv1.Subject{
					{Kind: "Group", Name: "customer:acme:Employees"},
				}),
			},
			customerAdminGroups: []accessgroup.AccessGroup{
				{Name: "customer:acme:Employees"},
				{Name: "customer:giantswarm:Employees"},
			},
			expectedRoleBinding: test.NewRoleBinding("write-all-customer-group", "org-customer", map[string]string{
				"kind": "ClusterRole",
				"name": "cluster-admin",
			}, []rbacv1.Subject{
				{Kind: "Group", Name: "customer:acme:Employees"},
				{Kind: "Group", Name: "customer:giantswarm:Employees"},
			}),
		},
		{
			name:         "case 3: Remove subjects from an existing role binding",
			orgNamespace: test.NewOrgNamespace("customer"),
			existingResources: []runtime.Object{
				test.NewRoleBinding("write-all-customer-group", "org-customer", map[string]string{
					"kind": "ClusterRole",
					"name": "cluster-admin",
				}, []rbacv1.Subject{
					{Kind: "Group", Name: "customer:acme:Employees"},
					{Kind: "Group", Name: "customer:giantswarm:Employees"},
				}),
			},
			customerAdminGroups: []accessgroup.AccessGroup{
				{Name: "customer:giantswarm:Employees"},
			},
			expectedRoleBinding: test.NewRoleBinding("write-all-customer-group", "org-customer", map[string]string{
				"kind": "ClusterRole",
				"name": "cluster-admin",
			}, []rbacv1.Subject{
				{Kind: "Group", Name: "customer:giantswarm:Employees"},
			}),
		},
		{
			name:         "case 4: Do not overwrite the role binding in case there are no changes",
			orgNamespace: test.NewOrgNamespace("customer"),
			existingResources: []runtime.Object{
				test.NewRoleBinding("write-all-customer-group", "org-customer", map[string]string{
					"kind": "ClusterRole",
					"name": "cluster-admin",
				}, []rbacv1.Subject{
					{Kind: "Group", Name: "customer:acme:Employees"},
					{Kind: "Group", Name: "customer:giantswarm:Employees"},
				}),
			},
			customerAdminGroups: []accessgroup.AccessGroup{
				{Name: "customer:acme:Employees"},
				{Name: "customer:giantswarm:Employees"},
			},
			expectedRoleBinding: test.NewRoleBinding("write-all-customer-group", "org-customer", map[string]string{
				"kind": "ClusterRole",
				"name": "cluster-admin",
			}, []rbacv1.Subject{
				{Kind: "Group", Name: "customer:acme:Employees"},
				{Kind: "Group", Name: "customer:giantswarm:Employees"},
			}),
		},
		{
			name:         "case 5: Create cluster role",
			orgNamespace: test.NewOrgNamespace("customer"),
			expectedClusterRole: test.NewClusterRole(
				"organization-customer-read",
				*test.NewPolicyRule([]string{"get"}, []string{"organizations"}, []string{"customer"}, []string{"security.giantswarm.io"})),
		},
		{
			name:         "case 6: Do not create rolebinding in protected namespace",
			orgNamespace: test.NewOrgNamespace("giantswarm"),
			customerAdminGroups: []accessgroup.AccessGroup{
				{Name: "customer:giantswarm:Employees"},
			},
		},
		{
			name:         "case 7: Delete rolebinding in protected namespace",
			orgNamespace: test.NewOrgNamespace("giantswarm"),
			existingResources: []runtime.Object{
				test.NewRoleBinding("write-all-customer-group", "org-giantswarm", map[string]string{
					"kind": "ClusterRole",
					"name": "cluster-admin",
				}, []rbacv1.Subject{
					{Kind: "Group", Name: "customer:giantswarm:Employees"},
				}),
			},
			customerAdminGroups: []accessgroup.AccessGroup{
				{Name: "customer:giantswarm:Employees"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			var runtimeObjects []runtime.Object
			if tc.orgNamespace != nil {
				runtimeObjects = append(runtimeObjects, tc.orgNamespace)
			}
			runtimeObjects = append(runtimeObjects, tc.existingResources...)

			var k8sClientFake *k8sclienttest.Clients
			{
				schemeBuilder := runtime.SchemeBuilder{
					security.AddToScheme,
				}

				err = schemeBuilder.AddToScheme(scheme.Scheme)
				if err != nil {
					t.Fatal(err)
				}

				k8sClientFake = k8sclienttest.NewClients(k8sclienttest.ClientsConfig{
					CtrlClient: clientfake.NewClientBuilder().
						WithScheme(scheme.Scheme).
						Build(),
					K8sClient: clientgofake.NewSimpleClientset(runtimeObjects...),
				})
			}

			namespaceAuth, err := New(Config{
				K8sClient:              k8sClientFake,
				Logger:                 microloggertest.New(),
				WriteAllCustomerGroups: tc.customerAdminGroups,
			})

			if err != nil {
				t.Fatal(err)
			}

			err = namespaceAuth.EnsureCreated(context.TODO(), tc.orgNamespace)
			if err != nil {
				t.Fatal(err)
			}

			if tc.expectedClusterRole != nil {
				checkClusterRole(t, k8sClientFake, tc.expectedClusterRole)
			}

			if tc.expectedRoleBinding != nil {
				checkRoleBinding(t, k8sClientFake, tc.expectedRoleBinding)
			}
		})
	}
}

func checkClusterRole(t *testing.T, k8sClient k8sclient.Interface, expectedClusterRole *rbacv1.ClusterRole) {
	clusterRole, err := k8sClient.K8sClient().RbacV1().ClusterRoles().Get(context.TODO(), expectedClusterRole.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(expectedClusterRole.Rules, clusterRole.Rules) {
		t.Fatalf("unexpected Rules - expected %s, received %s\n", expectedClusterRole.Rules, clusterRole.Rules)
	}
}

func checkRoleBinding(t *testing.T, k8sClient k8sclient.Interface, expectedRoleBinding *rbacv1.RoleBinding) {
	roleBinding, err := k8sClient.K8sClient().RbacV1().RoleBindings(expectedRoleBinding.Namespace).Get(context.TODO(), expectedRoleBinding.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(expectedRoleBinding.RoleRef, roleBinding.RoleRef) {
		t.Fatalf("unexpected RoleRef - expected %s, received %s\n", expectedRoleBinding.RoleRef, roleBinding.RoleRef)
	}

	if !reflect.DeepEqual(expectedRoleBinding.Subjects, roleBinding.Subjects) {
		t.Fatalf("unexpected Subjects - expected %s, received %s\n", expectedRoleBinding.Subjects, roleBinding.Subjects)
	}
}
