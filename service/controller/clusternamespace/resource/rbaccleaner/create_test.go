package rbaccleaner

import (
	"context"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/giantswarm/k8smetadata/pkg/label"

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

	"github.com/giantswarm/rbac-operator/pkg/project"
	"github.com/giantswarm/rbac-operator/service/controller/clusternamespace/key"
)

func Test_EnsureCreated(t *testing.T) {
	t.Run("Ensure app-operator managed rbac resources are deleted", func(t *testing.T) {
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

		appOperatorManagedName := key.AppOperatorClusterRoleNameFromNamespace(*wcNamespace)

		clusterRole := &rbacv1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ClusterRole",
				APIVersion: "rbac.authorization.k8s.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: appOperatorManagedName,
				Labels: map[string]string{
					label.ManagedBy: project.Name(),
				},
			},
			Rules: []rbacv1.PolicyRule{},
		}

		clusterRoleBinding := &rbacv1.ClusterRoleBinding{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ClusterRole",
				APIVersion: "rbac.authorization.k8s.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: appOperatorManagedName,
				Labels: map[string]string{
					label.ManagedBy: project.Name(),
				},
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      wcServiceAccount.Name,
					Namespace: wcServiceAccount.Namespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     clusterRole.Kind,
				Name:     clusterRole.Name,
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
					clusterRole, clusterRoleBinding,
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
			_, err = k8sClientFake.K8sClient().RbacV1().ClusterRoles().Get(context.TODO(), clusterRole.Name, metav1.GetOptions{})
			if !errors.IsNotFound(err) {
				t.Fatalf("The app-operator managed cluster role should be deleted, but error is: %#v", err)
			}

			_, err = k8sClientFake.K8sClient().RbacV1().ClusterRoleBindings().Get(context.TODO(), clusterRoleBinding.Name, metav1.GetOptions{})
			if !errors.IsNotFound(err) {
				t.Fatalf("The app-operator managed cluster role binding should be deleted, but error is: %#v", err)
			}
		}
	})
}
