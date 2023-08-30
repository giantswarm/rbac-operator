package clusternamespaceresources

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/giantswarm/rbac-operator/service/test"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/micrologger/microloggertest"
	security "github.com/giantswarm/organization-operator/api/v1alpha1"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_EnsureCreated(t *testing.T) {
	tests := []struct {
		name                    string
		namespaces              []*corev1.Namespace
		organization            *security.Organization
		roleBindings            []*rbacv1.RoleBinding
		expectedRoleBindings    []*rbacv1.RoleBinding
		expectedRoleBindingsNum map[string]int
	}{
		{
			name: "flawless",
			namespaces: []*corev1.Namespace{
				test.NewOrgNamespace("acme"),
				test.NewClusterNamespace("abc0", "acme"),
				test.NewGenericNamespace("giantswarm"),
			},
			organization: test.NewOrganization("acme"),
			roleBindings: []*rbacv1.RoleBinding{
				test.NewRoleBinding(
					"cluster-ns-organization-acme-write",
					"org-acme",
					map[string]string{
						"kind": "ClusterRole",
						"name": "write-in-cluster-ns",
					},
					[]rbacv1.Subject{
						{Kind: "Group", Name: "customer:acme:Employees"},
					},
				),
				test.NewRoleBinding(
					"cluster-ns-organization-acme-read",
					"org-acme",
					map[string]string{
						"kind": "ClusterRole",
						"name": "read-in-cluster-ns",
					},
					[]rbacv1.Subject{
						{Kind: "Group", Name: "customer:acme:Employees"},
					},
				),
			},
			expectedRoleBindings: []*rbacv1.RoleBinding{
				test.NewRoleBinding(
					"write-in-cluster-ns",
					"abc0",
					map[string]string{
						"kind": "Role",
						"name": "write-in-cluster-ns",
					},
					[]rbacv1.Subject{
						{Kind: "Group", Name: "customer:acme:Employees"},
					},
				),
				test.NewRoleBinding(
					"read-in-cluster-ns",
					"abc0",
					map[string]string{
						"kind": "Role",
						"name": "read-in-cluster-ns",
					},
					[]rbacv1.Subject{
						{Kind: "Group", Name: "customer:acme:Employees"},
					},
				),
			},
			expectedRoleBindingsNum: map[string]int{
				"abc0":       2,
				"org-acme":   2,
				"giantswarm": 0,
			},
		},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			var err error

			k8sObj := make([]runtime.Object, 0)
			for _, ns := range tc.namespaces {
				k8sObj = append(k8sObj, ns)
			}

			for _, rb := range tc.roleBindings {
				k8sObj = append(k8sObj, rb)
			}

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
						WithRuntimeObjects([]runtime.Object{tc.organization}...).
						Build(),
					K8sClient: clientgofake.NewSimpleClientset(k8sObj...),
				})
			}

			clusterns, err := New(Config{
				K8sClient: k8sClientFake,
				Logger:    microloggertest.New(),
			})
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			err = clusterns.EnsureCreated(context.TODO(), tc.namespaces[1])
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			for _, rb := range tc.expectedRoleBindings {
				r, err := k8sClientFake.K8sClient().
					RbacV1().
					RoleBindings(rb.ObjectMeta.Namespace).
					Get(context.TODO(), rb.ObjectMeta.Name, metav1.GetOptions{})

				if err != nil {
					t.Fatalf("error == %#v, want nil", err)
				}

				if !reflect.DeepEqual(r, rb) {
					t.Fatalf("want matching resources \n %s", cmp.Diff(r, rb))
				}
			}

			for ns, c := range tc.expectedRoleBindingsNum {
				r, err := k8sClientFake.K8sClient().
					RbacV1().
					RoleBindings(ns).
					List(context.TODO(), metav1.ListOptions{})

				if err != nil {
					t.Fatalf("error == %#v, want nil", err)
				}

				if len(r.Items) != c {
					t.Fatalf("got %d item(s) in %s namespace, want %d", len(r.Items), ns, c)
				}
			}
		})
	}
}
