package defaultnamespace

import (
	"context"
	"testing"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/micrologger/microloggertest"
	security "github.com/giantswarm/organization-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/giantswarm/rbac-operator/api/v1alpha1"
	"github.com/giantswarm/rbac-operator/service/controller/defaultnamespace/defaultnamespacetest"
	"github.com/giantswarm/rbac-operator/service/internal/accessgroup"
)

func Test_DefaultNamespaceController(t *testing.T) {
	testCases := []struct {
		Name                         string
		CustomerAdminGroups          []accessgroup.AccessGroup
		GSAdminGroup                 []accessgroup.AccessGroup
		ExpectedClusterRoles         int
		ExpectedClusterRoleBindings  int
		ExpectedRoles                int
		ExpectedRoleBindings         int
		ExpectedRoleBindingTemplates int
	}{
		{
			Name:                         "case0: Check that all resources are ensured created",
			CustomerAdminGroups:          []accessgroup.AccessGroup{{Name: "customer"}},
			GSAdminGroup:                 []accessgroup.AccessGroup{{Name: "giantswarm"}},
			ExpectedClusterRoles:         9,
			ExpectedClusterRoleBindings:  7,
			ExpectedRoles:                1,
			ExpectedRoleBindings:         2,
			ExpectedRoleBindingTemplates: 4,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.TODO()

			var err error

			defaultNamespace := defaultnamespacetest.NewDefaultNamespace()
			k8sValues := []runtime.Object{
				defaultNamespace,
				defaultnamespacetest.NewClusterAdminRole(),
			}

			var k8sClientFake *k8sclienttest.Clients
			{
				schemeBuilder := runtime.SchemeBuilder{
					security.AddToScheme,
					v1alpha1.AddToScheme,
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
					K8sClient: clientgofake.NewSimpleClientset(k8sValues...),
				})
			}

			defaultNamespaceController, err := NewDefaultNamespace(DefaultNamespaceConfig{
				K8sClient:            k8sClientFake,
				Logger:               microloggertest.New(),
				CustomerAdminGroups:  tc.CustomerAdminGroups,
				CustomerReaderGroups: tc.CustomerAdminGroups,
				GSAdminGroups:        tc.GSAdminGroup,
			})

			if err != nil {
				t.Fatalf("received unexpected error %s", err)
			}

			err = defaultNamespaceController.EnsureResourcesCreated(ctx)

			if err != nil {
				t.Fatalf("received unexpected error %s", err)
			}

			clusterRoles, err := k8sClientFake.K8sClient().RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
			if err != nil {
				t.Fatalf("received unexpected error %s", err)
			}
			shouldContainNumberOfResources(t, "ClusterRole", len(clusterRoles.Items), tc.ExpectedClusterRoles)

			clusterRoleBindings, err := k8sClientFake.K8sClient().RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
			if err != nil {
				t.Fatalf("received unexpected error %s", err)
			}
			shouldContainNumberOfResources(t, "ClusterRoleBinding", len(clusterRoleBindings.Items), tc.ExpectedClusterRoleBindings)

			roles, err := k8sClientFake.K8sClient().RbacV1().Roles(defaultNamespace.Name).List(ctx, metav1.ListOptions{})
			if err != nil {
				t.Fatalf("received unexpected error %s", err)
			}
			shouldContainNumberOfResources(t, "Role", len(roles.Items), tc.ExpectedRoles)

			roleBindings, err := k8sClientFake.K8sClient().RbacV1().RoleBindings(defaultNamespace.Name).List(ctx, metav1.ListOptions{})
			if err != nil {
				t.Fatalf("received unexpected error %s", err)
			}
			shouldContainNumberOfResources(t, "roleBindings.Kind", len(roleBindings.Items), tc.ExpectedRoleBindings)

			roleBindingTemplates := v1alpha1.RoleBindingTemplateList{}
			err = k8sClientFake.CtrlClient().List(ctx, &roleBindingTemplates)
			if err != nil {
				t.Fatalf("received unexpected error %s", err)
			}
			shouldContainNumberOfResources(t, "roleBindingTemplates.Kind", len(roleBindingTemplates.Items), tc.ExpectedRoleBindingTemplates)
		})
	}
}

func shouldContainNumberOfResources(t *testing.T, kind string, count int, expectedCount int) {
	if count != expectedCount {
		t.Fatalf("incorrect number of %s: expected %d, actual %d", kind, expectedCount, count)
	}
}
