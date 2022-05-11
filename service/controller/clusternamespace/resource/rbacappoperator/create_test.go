package rbacappoperator

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/giantswarm/rbac-operator/service/controller/clusternamespace/key"
)

func Test_EnsureCreated(t *testing.T) {
	t.Run("Ensure rbac-operator managed app-operator resources", func(t *testing.T) {
		var err error

		// Existing resources

		wcNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "in5m9",
			},
		}

		wcServiceAccount := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("app-operator-%s", wcNamespace.Name),
				Namespace: wcNamespace.Name,
			},
		}

		giantswarmNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "giantswarm",
			},
		}

		// Setup

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
					Build(),
				K8sClient: clientgofake.NewSimpleClientset([]runtime.Object{
					wcNamespace,
					wcServiceAccount,
					giantswarmNamespace,
				}...),
			})

			resource, err := New(Config{
				K8sClient: k8sClientFake,
				Logger:    microloggertest.New(),
			})

			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			// Execute

			err = resource.EnsureCreated(context.TODO(), wcNamespace)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			// Checks
			expectedName := key.AppOperatorRbacOperatorManagedResourceName(*wcNamespace)

			actualClusterRole, err := k8sClientFake.K8sClient().
				RbacV1().
				ClusterRoles().
				Get(context.TODO(), expectedName, metav1.GetOptions{})

			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			expectedClusterRole := getAppOperatorClusterRole(*wcNamespace)
			if !reflect.DeepEqual(actualClusterRole, expectedClusterRole) {
				t.Fatalf("want matching resources \n %s", cmp.Diff(actualClusterRole, expectedClusterRole))
			}
		}
	})
}
