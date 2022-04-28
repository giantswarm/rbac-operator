package clusternamespaceresources

import (
	"context"
	"fmt"
	"testing"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/micrologger/microloggertest"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgofake "k8s.io/client-go/kubernetes/fake"
)

func Test_EnsureDeleted(t *testing.T) {
	tests := []struct {
		name             string
		clusterNamespace string
		namespaces       []*corev1.Namespace
		roleBindings     []*rbacv1.RoleBinding
	}{
		{
			name:             "flawless",
			clusterNamespace: "abc0",
			namespaces: []*corev1.Namespace{
				newClusterNamespace("abc0", "acme"),
			},
			roleBindings: []*rbacv1.RoleBinding{
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
				newRoleBinding(
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
				newRoleBinding(
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
				k8sClientFake = k8sclienttest.NewClients(k8sclienttest.ClientsConfig{
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

			err = clusterns.EnsureDeleted(context.TODO(), tc.namespaces[0])
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			rbList, err := k8sClientFake.K8sClient().
				RbacV1().
				RoleBindings(tc.clusterNamespace).
				List(context.TODO(), metav1.ListOptions{})

			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			if len(rbList.Items) > 0 {
				t.Fatalf("got %d item(s), want 0", len(rbList.Items))
			}
		})
	}
}
