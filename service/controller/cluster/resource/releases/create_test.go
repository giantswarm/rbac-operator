package releases

import (
	"context"
	"testing"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/micrologger/microloggertest"
	security "github.com/giantswarm/organization-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/service/controller/cluster/clustertest"
)

func Test_Releases(t *testing.T) {
	testCases := []struct {
		Name                 string
		InitialObjects       []runtime.Object
		ExpectedClusterRoles []*rbacv1.ClusterRole
	}{
		{
			Name: "case0: Create cluster role for fetching Release CRs",
			ExpectedClusterRoles: []*rbacv1.ClusterRole{
				clustertest.NewClusterRole(
					pkgkey.ReadReleasesRole,
					clustertest.NewSingletonRules([]string{"release.giantswarm.io"}, []string{"releases"}),
				),
			},
		},
		{
			Name: "case1: Update cluster role for fetching Release CRs",
			InitialObjects: []runtime.Object{
				clustertest.NewClusterRole(
					pkgkey.ReadReleasesRole,
					clustertest.NewSingletonRules([]string{}, []string{}),
				),
			},
			ExpectedClusterRoles: []*rbacv1.ClusterRole{
				clustertest.NewClusterRole(
					pkgkey.ReadReleasesRole,
					clustertest.NewSingletonRules([]string{"release.giantswarm.io"}, []string{"releases"}),
				),
			},
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

			releases, err := New(Config{
				K8sClient: k8sClientFake,
				Logger:    microloggertest.New(),
			})

			if err == nil {
				namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: pkgkey.DefaultNamespaceName}}
				err = releases.EnsureCreated(ctx, namespace)
			}

			if err != nil {
				t.Fatalf("received unexpected error: %s", err)
			}

			clusterRoleList, err := k8sClientFake.K8sClient().RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
			if err != nil {
				t.Fatalf("failed to get cluster roles: %s", err)
			}
			clustertest.ClusterRolesShouldEqual(t, tc.ExpectedClusterRoles, clusterRoleList.Items)
		})
	}
}
