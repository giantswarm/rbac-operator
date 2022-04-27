package clusternamespaceresources

import (
	"context"
	"fmt"
	"reflect"
	"testing"

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
		name                 string
		namespaces           []*corev1.Namespace
		organization         *security.Organization
		roleBindings         []*rbacv1.RoleBinding
		expectedRoleBindings []*rbacv1.RoleBinding
	}{
		{
			name: "flawless",
			namespaces: []*corev1.Namespace{
				newOrgNamespace("acme"),
				newClusterNamespace("abc0", "acme"),
			},
			organization: newOrganization("acme"),
			roleBindings: []*rbacv1.RoleBinding{
				newRoleBinding(
					"flux-crd-controller",
					"org-acme",
					map[string]string{
						"kind": "ClusterRole",
						"name": "crd-controller",
					},
					[]rbacv1.Subject{
						{Kind: "ServiceAccount", Name: "helm-controller", Namespace: "flux-system"},
						{Kind: "ServiceAccount", Name: "image-automation-controller", Namespace: "flux-system"},
						{Kind: "ServiceAccount", Name: "image-reflector-controller", Namespace: "flux-system"},
						{Kind: "ServiceAccount", Name: "kustomize-controller", Namespace: "flux-system"},
						{Kind: "ServiceAccount", Name: "notification-controller", Namespace: "flux-system"},
						{Kind: "ServiceAccount", Name: "source-controller", Namespace: "flux-system"},
					},
				),
				newRoleBinding(
					"flux-namespace-reconciler",
					"org-acme",
					map[string]string{
						"kind": "ClusterRole",
						"name": "cluster-admin",
					},
					[]rbacv1.Subject{
						{Kind: "ServiceAccount", Name: "helm-controller", Namespace: "flux-system"},
						{Kind: "ServiceAccount", Name: "kustomize-controller", Namespace: "flux-system"},
					},
				),
			},
			expectedRoleBindings: []*rbacv1.RoleBinding{
				newRoleBinding(
					"flux-crd-controller",
					"abc0",
					map[string]string{
						"kind": "ClusterRole",
						"name": "crd-controller",
					},
					[]rbacv1.Subject{
						{Kind: "ServiceAccount", Name: "helm-controller", Namespace: "flux-system"},
						{Kind: "ServiceAccount", Name: "image-automation-controller", Namespace: "flux-system"},
						{Kind: "ServiceAccount", Name: "image-reflector-controller", Namespace: "flux-system"},
						{Kind: "ServiceAccount", Name: "kustomize-controller", Namespace: "flux-system"},
						{Kind: "ServiceAccount", Name: "notification-controller", Namespace: "flux-system"},
						{Kind: "ServiceAccount", Name: "source-controller", Namespace: "flux-system"},
					},
				),
				newRoleBinding(
					"flux-namespace-reconciler",
					"abc0",
					map[string]string{
						"kind": "ClusterRole",
						"name": "cluster-admin",
					},
					[]rbacv1.Subject{
						{Kind: "ServiceAccount", Name: "helm-controller", Namespace: "flux-system"},
						{Kind: "ServiceAccount", Name: "kustomize-controller", Namespace: "flux-system"},
					},
				),
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
		})
	}
}
