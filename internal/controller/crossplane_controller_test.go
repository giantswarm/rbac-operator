package controller

import (
	"context"
	"fmt"
	"testing"

	"github.com/giantswarm/k8sclient/v8/pkg/k8sclienttest"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
)

const testCrossplaneClusterRoleName = "crossplane-edit"

var crossplaneEditCR = rbacv1.ClusterRole{
	ObjectMeta: metav1.ObjectMeta{
		Name: testCrossplaneClusterRoleName,
	},
}

var crossplaneClusterRoleBinding = rbacv1.ClusterRoleBinding{
	ObjectMeta: metav1.ObjectMeta{
		Name: pkgkey.CrossplaneBindClusterRoleBindingName(testCrossplaneClusterRoleName),
	},
}

func Test_Reconcile(t *testing.T) {
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
				testScheme := runtime.NewScheme()
				err = corev1.AddToScheme(testScheme)
				if err != nil {
					t.Fatalf("error == %#v, want nil", err)
				}
				err = rbacv1.AddToScheme(testScheme)
				if err != nil {
					t.Fatalf("error == %#v, want nil", err)
				}

				ctrlClientObjs := []runtime.Object{tc.clusterRole}
				for _, crb := range tc.clusterRoleBindings {
					ctrlClientObjs = append(ctrlClientObjs, crb)
				}

				k8sClientFake = k8sclienttest.NewClients(k8sclienttest.ClientsConfig{
					CtrlClient: clientfake.NewClientBuilder().WithScheme(testScheme).WithRuntimeObjects(ctrlClientObjs...).Build(),
					K8sClient:  clientgofake.NewSimpleClientset(k8sObj...),
				})
			}

			r := &CrossplaneReconciler{
				Client:                              k8sClientFake.CtrlClient(),
				Scheme:                              k8sClientFake.CtrlClient().Scheme(),
				CrossplaneBindTriggeringClusterRole: testCrossplaneClusterRoleName,
			}
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			_, err = r.Reconcile(context.TODO(), ctrl.Request{
				NamespacedName: types.NamespacedName{Name: tc.clusterRole.Name},
			})
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			var gotCRB rbacv1.ClusterRoleBinding
			err = k8sClientFake.CtrlClient().Get(context.TODO(), client.ObjectKey{
				Name: pkgkey.CrossplaneBindClusterRoleBindingName(testCrossplaneClusterRoleName),
			}, &gotCRB)

			if errors.IsNotFound(err) {
				t.Fatalf("error == %#v, was not NotFound", err)
			} else if err != nil {
				t.Fatalf("error == %#v, was expecting no error", err)
			}
		})
	}
}
