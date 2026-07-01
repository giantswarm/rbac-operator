package catalog

import (
	"context"
	"testing"

	"github.com/giantswarm/k8sclient/v8/pkg/k8sclienttest"
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
	"github.com/giantswarm/rbac-operator/service/controller/defaultnamespace/defaultnamespacetest"
)

func Test_Catalog(t *testing.T) {
	testCases := []struct {
		Name           string
		InitialObjects []runtime.Object
		ExpectedRoles  []*rbacv1.Role
	}{
		{
			Name: "case0: Create a role with permissions to read catalogs and catalog entries",
			ExpectedRoles: []*rbacv1.Role{
				defaultnamespacetest.NewRole(pkgkey.ReadDefaultCatalogsRole, pkgkey.DefaultNamespaceName, []rbacv1.PolicyRule{
					defaultnamespacetest.NewSingleResourceRule("application.giantswarm.io", "catalogs"),
					defaultnamespacetest.NewSingleResourceRule("application.giantswarm.io", "appcatalogentries"),
				}),
			},
		},
		{
			Name: "case1: Update a role with permissions to read catalogs and catalog entries",
			InitialObjects: []runtime.Object{
				defaultnamespacetest.NewRole(pkgkey.ReadDefaultCatalogsRole, pkgkey.DefaultNamespaceName, []rbacv1.PolicyRule{
					defaultnamespacetest.NewRule([]string{}, []string{}),
				}),
			},
			ExpectedRoles: []*rbacv1.Role{
				defaultnamespacetest.NewRole(pkgkey.ReadDefaultCatalogsRole, pkgkey.DefaultNamespaceName, []rbacv1.PolicyRule{
					defaultnamespacetest.NewSingleResourceRule("application.giantswarm.io", "catalogs"),
					defaultnamespacetest.NewSingleResourceRule("application.giantswarm.io", "appcatalogentries"),
				}),
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

			namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: pkgkey.DefaultNamespaceName}}
			if err == nil {
				err = releases.EnsureCreated(ctx, namespace)
			}

			if err != nil {
				t.Fatalf("received unexpected error: %s", err)
			}

			roleList, err := k8sClientFake.K8sClient().RbacV1().Roles(namespace.Name).List(ctx, metav1.ListOptions{})
			if err != nil {
				t.Fatalf("failed to get roles: %s", err)
			}
			defaultnamespacetest.RolesShouldEqual(t, tc.ExpectedRoles, roleList.Items)
		})
	}
}
