package usergroups

import (
	"context"
	"testing"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger/microloggertest"
	security "github.com/giantswarm/organization-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/service/controller/cluster/clustertest"
	"github.com/giantswarm/rbac-operator/service/internal/accessgroup"
)

func Test_UserGroups(t *testing.T) {
	testCases := []struct {
		Name                        string
		InitialObjects              []runtime.Object
		CustomerAdminGroups         []accessgroup.AccessGroup
		GSAdminGroups               []accessgroup.AccessGroup
		ExpectedRoleBindings        []*rbacv1.RoleBinding
		ExpectedClusterRoleBindings []*rbacv1.ClusterRoleBinding
		ExpectedError               error
	}{
		{
			Name:                "case 0: Add new bindings with multiple subjects",
			CustomerAdminGroups: []accessgroup.AccessGroup{{Name: "customers1"}, {Name: "customers2"}},
			GSAdminGroups:       []accessgroup.AccessGroup{{Name: "giantswarm1"}, {Name: "giantswarm2"}},
			ExpectedRoleBindings: []*rbacv1.RoleBinding{
				clustertest.NewRoleBinding(
					pkgkey.WriteAllCustomerGroupRoleBindingName(),
					pkgkey.DefaultNamespaceName,
					clustertest.NewGroupSubjects("customers1", "customers2"),
				),
			},
			ExpectedClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
				clustertest.NewClusterRoleBinding(
					pkgkey.WriteOrganizationsCustomerGroupClusterRoleBindingName(),
					clustertest.NewGroupSubjects("customers1", "customers2"),
				),
				clustertest.NewClusterRoleBinding(
					pkgkey.ReadAllCustomerGroupClusterRoleBindingName(),
					clustertest.NewGroupSubjects("customers1", "customers2"),
				),
				clustertest.NewClusterRoleBinding(
					pkgkey.WriteAllGSGroupClusterRoleBindingName(),
					clustertest.NewGroupSubjects("giantswarm1", "giantswarm2"),
				),
			},
		},
		{
			Name: "case 1: Add multiple subjects to existing bindings",
			InitialObjects: []runtime.Object{
				clustertest.NewRoleBinding(
					pkgkey.WriteAllCustomerGroupRoleBindingName(),
					pkgkey.DefaultNamespaceName,
					clustertest.NewGroupSubjects("customers"),
				),
				clustertest.NewClusterRoleBinding(
					pkgkey.WriteOrganizationsCustomerGroupClusterRoleBindingName(),
					clustertest.NewGroupSubjects("customers"),
				),
				clustertest.NewClusterRoleBinding(
					pkgkey.ReadAllCustomerGroupClusterRoleBindingName(),
					clustertest.NewGroupSubjects("customers"),
				),
				clustertest.NewClusterRoleBinding(
					pkgkey.WriteAllGSGroupClusterRoleBindingName(),
					clustertest.NewGroupSubjects("giantswarm"),
				),
			},
			CustomerAdminGroups: []accessgroup.AccessGroup{{Name: "customers1"}, {Name: "customers2"}},
			GSAdminGroups:       []accessgroup.AccessGroup{{Name: "giantswarm1"}, {Name: "giantswarm2"}},
			ExpectedRoleBindings: []*rbacv1.RoleBinding{
				clustertest.NewRoleBinding(
					pkgkey.WriteAllCustomerGroupRoleBindingName(),
					pkgkey.DefaultNamespaceName,
					clustertest.NewGroupSubjects("customers1", "customers2"),
				),
			},
			ExpectedClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
				clustertest.NewClusterRoleBinding(
					pkgkey.WriteOrganizationsCustomerGroupClusterRoleBindingName(),
					clustertest.NewGroupSubjects("customers1", "customers2"),
				),
				clustertest.NewClusterRoleBinding(
					pkgkey.ReadAllCustomerGroupClusterRoleBindingName(),
					clustertest.NewGroupSubjects("customers1", "customers2"),
				),
				clustertest.NewClusterRoleBinding(
					pkgkey.WriteAllGSGroupClusterRoleBindingName(),
					clustertest.NewGroupSubjects("giantswarm1", "giantswarm2"),
				),
			},
		},
		{
			Name:          "case 2: Fail in attempt to create/update bindings with empty subjects",
			ExpectedError: invalidConfigError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			var err error

			ctx := context.TODO()

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
						WithRuntimeObjects().
						Build(),
					K8sClient: clientgofake.NewSimpleClientset(tc.InitialObjects...),
				})
			}

			userGroups, err := New(Config{
				K8sClient:           k8sClientFake,
				Logger:              microloggertest.New(),
				CustomerAdminGroups: tc.CustomerAdminGroups,
				GSAdminGroups:       tc.GSAdminGroups,
			})

			if err == nil {
				namespace := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: pkgkey.DefaultNamespaceName}}
				err = userGroups.EnsureCreated(ctx, namespace)
			}

			if tc.ExpectedError != nil && err == nil {
				t.Fatalf("did not receive an expected error: %s", tc.ExpectedError)
			} else if err != nil && err != tc.ExpectedError && microerror.Cause(err) != tc.ExpectedError {
				t.Fatalf("received an unexpected error: %s", err)
			}

			clusterRoleBindingList, err := k8sClientFake.K8sClient().RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
			if err != nil {
				t.Fatalf("failed to get cluster role bindings: %s", err)
			}
			clustertest.ClusterRoleBindingsShouldEqual(t, tc.ExpectedClusterRoleBindings, clusterRoleBindingList.Items)

			roleBindingList, err := k8sClientFake.K8sClient().RbacV1().RoleBindings(pkgkey.DefaultNamespaceName).List(ctx, metav1.ListOptions{})
			if err != nil {
				t.Fatalf("failed to get role bindings: %s", err)
			}
			clustertest.RoleBindingsShouldEqual(t, tc.ExpectedRoleBindings, roleBindingList.Items)
		})
	}
}
