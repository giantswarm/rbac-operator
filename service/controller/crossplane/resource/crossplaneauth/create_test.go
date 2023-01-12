package crossplaneauth_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/micrologger/microloggertest"

	"github.com/giantswarm/rbac-operator/service/controller/crossplane/key"
	"github.com/giantswarm/rbac-operator/service/controller/crossplane/resource/crossplaneauth"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgofake "k8s.io/client-go/kubernetes/fake"
)

func Test_EnsureCreated(t *testing.T) {
	tests := []struct {
		name                string
		clusterRole         *rbacv1.ClusterRole
		clusterRoleBindings []*rbacv1.ClusterRoleBinding
	}{
		{
			name:                "creates when CRB not present",
			clusterRole:         &crossplaneEditCR,
			clusterRoleBindings: make([]*rbacv1.ClusterRoleBinding, 0),
		},
		{
			name:                "updates when CRB present",
			clusterRole:         &crossplaneEditCR,
			clusterRoleBindings: []*rbacv1.ClusterRoleBinding{&crossplaneClusterRoleBinding},
		},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			var err error

			k8sObj := []runtime.Object{tc.clusterRole}

			var k8sClientFake *k8sclienttest.Clients
			{
				k8sClientFake = k8sclienttest.NewClients(k8sclienttest.ClientsConfig{
					K8sClient: clientgofake.NewSimpleClientset(k8sObj...),
				})
			}

			fakeCrossplaneauth, err := crossplaneauth.New(crossplaneauth.Config{
				K8sClient:                           k8sClientFake,
				Logger:                              microloggertest.New(),
				CrossplaneBindTriggeringClusterRole: testCrossplaneClusterRoleName,
			})
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			err = fakeCrossplaneauth.EnsureCreated(context.TODO(), tc.clusterRole)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			_, err = k8sClientFake.K8sClient().RbacV1().ClusterRoleBindings().Get(context.TODO(),
				key.GetClusterRoleBindingName(testCrossplaneClusterRoleName), metav1.GetOptions{})

			if errors.IsNotFound(err) {
				t.Fatalf("error == %#v, was not NotFound", err)
			} else if err != nil {
				t.Fatalf("error == %#v, was expecting no error", err)
			}
		})
	}
}
