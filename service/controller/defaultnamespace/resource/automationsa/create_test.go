package automationsa

import (
	"context"
	"testing"

	"k8s.io/utils/ptr"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
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

func Test_AutomationSA(t *testing.T) {
	testCases := []struct {
		Name                        string
		InitialObjects              []runtime.Object
		ExpectedSAs                 []*corev1.ServiceAccount
		ExpectedRoleBindings        []*rbacv1.RoleBinding
		ExpectedClusterRoleBindings []*rbacv1.ClusterRoleBinding
	}{
		{
			Name: "case0: Create automation service account and bindings",
			ExpectedSAs: []*corev1.ServiceAccount{
				defaultnamespacetest.NewServiceAccount(pkgkey.AutomationServiceAccountName, pkgkey.DefaultNamespaceName),
			},
			ExpectedRoleBindings: []*rbacv1.RoleBinding{
				defaultnamespacetest.NewRoleBinding(
					pkgkey.WriteAllAutomationSARoleBindingName(),
					pkgkey.DefaultNamespaceName,
					defaultnamespacetest.NewSingletonSASubjects(pkgkey.AutomationServiceAccountName, pkgkey.DefaultNamespaceName),
				),
			},
			ExpectedClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
				defaultnamespacetest.NewClusterRoleBinding(
					pkgkey.ReadAllAutomationSAClusterRoleBindingName(),
					defaultnamespacetest.NewSingletonSASubjects(pkgkey.AutomationServiceAccountName, pkgkey.DefaultNamespaceName),
				),
				defaultnamespacetest.NewClusterRoleBinding(
					pkgkey.WriteOrganizationsAutomationSARoleBindingName(),
					defaultnamespacetest.NewSingletonSASubjects(pkgkey.AutomationServiceAccountName, pkgkey.DefaultNamespaceName),
				),
				defaultnamespacetest.NewClusterRoleBinding(
					pkgkey.WriteClientCertsAutomationSARoleBindingName(),
					defaultnamespacetest.NewSingletonSASubjects(pkgkey.AutomationServiceAccountName, pkgkey.DefaultNamespaceName),
				),
				defaultnamespacetest.NewClusterRoleBinding(
					pkgkey.WriteSilencesAutomationSARoleBindingName(),
					defaultnamespacetest.NewSingletonSASubjects(pkgkey.AutomationServiceAccountName, pkgkey.DefaultNamespaceName),
				),
			},
		},
		{
			Name: "case1: Update automation service account and bindings",
			InitialObjects: []runtime.Object{
				defaultnamespacetest.NewServiceAccount(pkgkey.AutomationServiceAccountName, pkgkey.DefaultNamespaceName),
				defaultnamespacetest.NewRoleBinding(pkgkey.WriteAllAutomationSARoleBindingName(), pkgkey.DefaultNamespaceName, []rbacv1.Subject{}),
				defaultnamespacetest.NewClusterRoleBinding(pkgkey.ReadAllAutomationSAClusterRoleBindingName(), []rbacv1.Subject{}),
				defaultnamespacetest.NewClusterRoleBinding(pkgkey.WriteOrganizationsAutomationSARoleBindingName(), []rbacv1.Subject{}),
				defaultnamespacetest.NewClusterRoleBinding(pkgkey.WriteClientCertsAutomationSARoleBindingName(), []rbacv1.Subject{}),
				defaultnamespacetest.NewClusterRoleBinding(pkgkey.WriteSilencesAutomationSARoleBindingName(), []rbacv1.Subject{}),
			},
			ExpectedSAs: []*corev1.ServiceAccount{
				defaultnamespacetest.NewServiceAccount(pkgkey.AutomationServiceAccountName, pkgkey.DefaultNamespaceName),
			},
			ExpectedRoleBindings: []*rbacv1.RoleBinding{
				defaultnamespacetest.NewRoleBinding(
					pkgkey.WriteAllAutomationSARoleBindingName(),
					pkgkey.DefaultNamespaceName,
					defaultnamespacetest.NewSingletonSASubjects(pkgkey.AutomationServiceAccountName, pkgkey.DefaultNamespaceName),
				),
			},
			ExpectedClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
				defaultnamespacetest.NewClusterRoleBinding(
					pkgkey.ReadAllAutomationSAClusterRoleBindingName(),
					defaultnamespacetest.NewSingletonSASubjects(pkgkey.AutomationServiceAccountName, pkgkey.DefaultNamespaceName),
				),
				defaultnamespacetest.NewClusterRoleBinding(
					pkgkey.WriteOrganizationsAutomationSARoleBindingName(),
					defaultnamespacetest.NewSingletonSASubjects(pkgkey.AutomationServiceAccountName, pkgkey.DefaultNamespaceName),
				),
				defaultnamespacetest.NewClusterRoleBinding(
					pkgkey.WriteClientCertsAutomationSARoleBindingName(),
					defaultnamespacetest.NewSingletonSASubjects(pkgkey.AutomationServiceAccountName, pkgkey.DefaultNamespaceName),
				),
				defaultnamespacetest.NewClusterRoleBinding(
					pkgkey.WriteSilencesAutomationSARoleBindingName(),
					defaultnamespacetest.NewSingletonSASubjects(pkgkey.AutomationServiceAccountName, pkgkey.DefaultNamespaceName),
				),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			var err error

			ctx := context.TODO()

			namespace := defaultnamespacetest.NewDefaultNamespace()

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

			automationSA, err := New(Config{
				K8sClient: k8sClientFake,
				Logger:    microloggertest.New(),
			})

			if err == nil {
				err = automationSA.EnsureCreated(ctx, namespace)
			}

			if err != nil {
				t.Fatalf("received an unexpected error: %s", err)
			}

			serviceAccountList, err := k8sClientFake.K8sClient().CoreV1().ServiceAccounts(namespace.Name).List(ctx, metav1.ListOptions{})
			if err != nil {
				t.Fatalf("failed to get service accounts: %s", err)
			}
			defaultnamespacetest.ServiceAccountsShouldEqual(t, tc.ExpectedSAs, serviceAccountList.Items)

			clusterRoleBindingList, err := k8sClientFake.K8sClient().RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
			if err != nil {
				t.Fatalf("failed to get cluster role bindings: %s", err)
			}
			defaultnamespacetest.ClusterRoleBindingsShouldEqual(t, tc.ExpectedClusterRoleBindings, clusterRoleBindingList.Items)

			roleBindingList, err := k8sClientFake.K8sClient().RbacV1().RoleBindings(namespace.Name).List(ctx, metav1.ListOptions{})
			if err != nil {
				t.Fatalf("failed to get role bindings: %s", err)
			}
			defaultnamespacetest.RoleBindingsShouldEqual(t, tc.ExpectedRoleBindings, roleBindingList.Items)
		})
	}
}

func Test_AutomationSAUpdate(t *testing.T) {
	testCases := []struct {
		name           string
		InitialObjects []runtime.Object
		ExpectedSAs    []*corev1.ServiceAccount
	}{
		{
			name: "case 0: Create a new Automation SA if it does not exist",
			ExpectedSAs: []*corev1.ServiceAccount{
				defaultnamespacetest.NewServiceAccount(pkgkey.AutomationServiceAccountName, pkgkey.DefaultNamespaceName),
			},
		},
		{
			name: "case 1: Do not update existing SA in case there are no changes",
			InitialObjects: []runtime.Object{
				&corev1.ServiceAccount{
					ObjectMeta:                   metav1.ObjectMeta{Name: pkgkey.AutomationServiceAccountName, Namespace: pkgkey.DefaultNamespaceName},
					AutomountServiceAccountToken: ptr.To(true),
					Secrets: []corev1.ObjectReference{
						{Name: "automation-token-123456", Kind: "Secret"},
					},
				},
			},
			ExpectedSAs: []*corev1.ServiceAccount{
				{
					ObjectMeta:                   metav1.ObjectMeta{Name: pkgkey.AutomationServiceAccountName, Namespace: pkgkey.DefaultNamespaceName},
					AutomountServiceAccountToken: ptr.To(true),
					Secrets: []corev1.ObjectReference{
						{Name: "automation-token-123456", Kind: "Secret"},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var err error

			ctx := context.TODO()

			namespace := defaultnamespacetest.NewDefaultNamespace()

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

			automationSA, err := New(Config{
				K8sClient: k8sClientFake,
				Logger:    microloggertest.New(),
			})

			if err == nil {
				err = automationSA.EnsureCreated(ctx, namespace)
			}

			if err != nil {
				t.Fatalf("received an unexpected error: %s", err)
			}

			serviceAccountList, err := k8sClientFake.K8sClient().CoreV1().ServiceAccounts(namespace.Name).List(ctx, metav1.ListOptions{})
			if err != nil {
				t.Fatalf("failed to get service accounts: %s", err)
			}
			defaultnamespacetest.ServiceAccountsShouldEqualDeep(t, tc.ExpectedSAs, serviceAccountList.Items)
		})
	}
}
