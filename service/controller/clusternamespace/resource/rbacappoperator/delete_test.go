package rbacappoperator

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/micrologger/microloggertest"
	security "github.com/giantswarm/organization-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/giantswarm/rbac-operator/service/controller/clusternamespace/key"
)

func Test_EnsureDeleted(t *testing.T) {
	t.Run("Ensure rbac-operator managed app-operator resources are deleted", func(t *testing.T) {
		var err error

		// Existing resources

		wcNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "in5m9",
			},
		}

		wcServiceAccount := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.AppOperatorServiceAccountNameFromNamespace(*wcNamespace),
				Namespace: wcNamespace.Name,
			},
		}

		giantswarmNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "giantswarm",
			},
		}

		clusterRole := getAppOperatorClusterRole(*wcNamespace)
		clusterRoleBinding := getAppOperatorCLusterRoleBinding(*wcNamespace, clusterRole)
		ownNamespaceRole := getAppOperatorOwnNamespaceRole(*wcNamespace)
		ownNamespaceRoleBinding := getAppOperatorOwnNamespaceRoleBinding(*wcNamespace, ownNamespaceRole)
		catalogReaderRole := getAppOperatorCatalogReaderRole(*wcNamespace)
		catalogReaderRoleBinding := getAppOperatorCatalogReaderRoleBinding(*wcNamespace, catalogReaderRole)

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
					clusterRole, clusterRoleBinding,
					ownNamespaceRole, ownNamespaceRoleBinding,
					catalogReaderRole, catalogReaderRoleBinding,
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

			err = resource.EnsureDeleted(context.TODO(), wcNamespace)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			// Checks
			_, err = k8sClientFake.K8sClient().RbacV1().ClusterRoles().Get(context.TODO(), clusterRole.Name, metav1.GetOptions{})
			if !errors.IsNotFound(err) {
				t.Fatalf("The self managed app-operator cluster role should be deleted, but error is: %#v", err)
			}

			_, err = k8sClientFake.K8sClient().RbacV1().ClusterRoleBindings().Get(context.TODO(), clusterRoleBinding.Name, metav1.GetOptions{})
			if !errors.IsNotFound(err) {
				t.Fatalf("The self managed app-operator cluster role binding should be deleted, but error is: %#v", err)
			}

			_, err = k8sClientFake.K8sClient().RbacV1().Roles(ownNamespaceRole.Namespace).Get(context.TODO(), ownNamespaceRole.Name, metav1.GetOptions{})
			if !errors.IsNotFound(err) {
				t.Fatalf("The self managed app-operator role for own namespace should be deleted, but error is: %#v", err)
			}

			_, err = k8sClientFake.K8sClient().RbacV1().RoleBindings(ownNamespaceRoleBinding.Namespace).Get(context.TODO(), ownNamespaceRoleBinding.Name, metav1.GetOptions{})
			if !errors.IsNotFound(err) {
				t.Fatalf("The self managed app-operator role binding for own namespace should be deleted, but error is: %#v", err)
			}

			_, err = k8sClientFake.K8sClient().RbacV1().Roles(catalogReaderRole.Namespace).Get(context.TODO(), catalogReaderRole.Name, metav1.GetOptions{})
			if !errors.IsNotFound(err) {
				t.Fatalf("The self managed app-operator role for catalog reading should be deleted, but error is: %#v", err)
			}

			_, err = k8sClientFake.K8sClient().RbacV1().RoleBindings(catalogReaderRoleBinding.Namespace).Get(context.TODO(), catalogReaderRoleBinding.Name, metav1.GetOptions{})
			if !errors.IsNotFound(err) {
				t.Fatalf("The self managed app-operator role binding for catalog reading should be deleted, but error is: %#v", err)
			}
		}
	})
}
