package crossplaneauth_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/micrologger/microloggertest"

	"github.com/giantswarm/rbac-operator/service/controller/crossplane/key"
	"github.com/giantswarm/rbac-operator/service/controller/crossplane/resource/crossplaneauth"

	// corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgofake "k8s.io/client-go/kubernetes/fake"
)

func Test_EnsureDeleted(t *testing.T) {
	tests := []struct {
		name                string
		clusterRole         *rbacv1.ClusterRole
		clusterRoleBindings []*rbacv1.ClusterRoleBinding
	}{
		{
			name:                "doesn't delete when CRB not present",
			clusterRole:         &crossplaneEditCR,
			clusterRoleBindings: make([]*rbacv1.ClusterRoleBinding, 0),
		},
		{
			name:                "deletes when CRB present",
			clusterRole:         &crossplaneEditCR,
			clusterRoleBindings: []*rbacv1.ClusterRoleBinding{&crossplaneClusterRoleBinding},
		},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			var err error

			k8sObj := []runtime.Object{tc.clusterRole}

			for _, crb := range tc.clusterRoleBindings {
				k8sObj = append(k8sObj, crb)
			}

			var k8sClientFake *k8sclienttest.Clients
			{
				k8sClientFake = k8sclienttest.NewClients(k8sclienttest.ClientsConfig{
					K8sClient: clientgofake.NewSimpleClientset(k8sObj...),
				})
			}

			fakeCrossplaneauth, err := crossplaneauth.New(crossplaneauth.Config{
				K8sClient: k8sClientFake,
				Logger:    microloggertest.New(),
			})
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			err = fakeCrossplaneauth.EnsureDeleted(context.TODO(), tc.clusterRole)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			_, err = k8sClientFake.K8sClient().RbacV1().ClusterRoleBindings().Get(context.TODO(),
				key.GetClusterRoleBindingName("crossplane-edit"), metav1.GetOptions{})

			if errors.IsNotFound(err) == false {
				t.Fatalf("error == %#v, want NotFound", err)
			}
		})
	}
}
