package automation

import (
	"context"
	"reflect"
	"testing"

	"github.com/giantswarm/k8sclient/v8/pkg/k8sclient"
	"github.com/giantswarm/k8sclient/v8/pkg/k8sclienttest"
	"github.com/giantswarm/micrologger/microloggertest"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/giantswarm/rbac-operator/service/test"
)

// the Role rules the resource is expected to grant for the App to HelmRelease migration.
var expectedPatchChartsRules = []rbacv1.PolicyRule{
	{
		APIGroups: []string{"application.giantswarm.io"},
		Resources: []string{"charts"},
		Verbs:     []string{"list", "get", "patch"},
	},
}

func automationSubject(namespace string) rbacv1.Subject {
	return rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      "automation",
		Namespace: namespace,
	}
}

func patchChartsRoleBinding(subjects []rbacv1.Subject) *rbacv1.RoleBinding {
	return test.NewRoleBinding("patch-charts", "giantswarm", map[string]string{
		"kind": "Role",
		"name": "patch-charts",
	}, subjects)
}

func Test_EnsureCreated_PatchCharts(t *testing.T) {
	testCases := []struct {
		name              string
		orgNamespace      string
		existingResources []runtime.Object
		expectedSubjects  []rbacv1.Subject
	}{
		{
			name:             "case 0: create the Role and RoleBinding when they do not exist yet",
			orgNamespace:     "customer",
			expectedSubjects: []rbacv1.Subject{automationSubject("org-customer")},
		},
		{
			name:         "case 1: append the org's automation SA to an existing RoleBinding",
			orgNamespace: "customer",
			existingResources: []runtime.Object{
				patchChartsRoleBinding([]rbacv1.Subject{automationSubject("org-acme")}),
			},
			expectedSubjects: []rbacv1.Subject{
				automationSubject("org-acme"),
				automationSubject("org-customer"),
			},
		},
		{
			name:         "case 2: do not add a duplicate subject when the org's automation SA is already present",
			orgNamespace: "customer",
			existingResources: []runtime.Object{
				patchChartsRoleBinding([]rbacv1.Subject{
					automationSubject("org-acme"),
					automationSubject("org-customer"),
				}),
			},
			expectedSubjects: []rbacv1.Subject{
				automationSubject("org-acme"),
				automationSubject("org-customer"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			orgNamespace := test.NewOrgNamespace(tc.orgNamespace)

			runtimeObjects := []runtime.Object{orgNamespace}
			runtimeObjects = append(runtimeObjects, tc.existingResources...)

			k8sClientFake := newFakeClients(runtimeObjects...)

			r, err := New(Config{
				K8sClient: k8sClientFake,
				Logger:    microloggertest.New(),
			})
			if err != nil {
				t.Fatal(err)
			}

			if err := r.EnsureCreated(context.TODO(), orgNamespace); err != nil {
				t.Fatal(err)
			}

			checkRole(t, k8sClientFake, "patch-charts", "giantswarm", expectedPatchChartsRules)
			checkRoleBindingSubjects(t, k8sClientFake, "patch-charts", "giantswarm", tc.expectedSubjects)
		})
	}
}

func Test_EnsureDeleted_PatchCharts(t *testing.T) {
	testCases := []struct {
		name              string
		orgNamespace      string
		existingResources []runtime.Object
		expectedSubjects  []rbacv1.Subject
	}{
		{
			name:         "case 0: remove the org's automation SA and keep the others",
			orgNamespace: "customer",
			existingResources: []runtime.Object{
				patchChartsRoleBinding([]rbacv1.Subject{
					automationSubject("org-acme"),
					automationSubject("org-customer"),
				}),
			},
			expectedSubjects: []rbacv1.Subject{automationSubject("org-acme")},
		},
		{
			name:         "case 1: leave an empty subject list when the org was the only subject",
			orgNamespace: "customer",
			existingResources: []runtime.Object{
				patchChartsRoleBinding([]rbacv1.Subject{automationSubject("org-customer")}),
			},
			expectedSubjects: []rbacv1.Subject{},
		},
		{
			name:             "case 2: do nothing when the RoleBinding does not exist",
			orgNamespace:     "customer",
			expectedSubjects: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			orgNamespace := test.NewOrgNamespace(tc.orgNamespace)

			runtimeObjects := []runtime.Object{orgNamespace}
			runtimeObjects = append(runtimeObjects, tc.existingResources...)

			k8sClientFake := newFakeClients(runtimeObjects...)

			r, err := New(Config{
				K8sClient: k8sClientFake,
				Logger:    microloggertest.New(),
			})
			if err != nil {
				t.Fatal(err)
			}

			if err := r.EnsureDeleted(context.TODO(), orgNamespace); err != nil {
				t.Fatal(err)
			}

			if tc.existingResources == nil {
				// The RoleBinding never existed and must not be created on deletion.
				_, err := k8sClientFake.K8sClient().RbacV1().RoleBindings("giantswarm").Get(context.TODO(), "patch-charts", metav1.GetOptions{})
				if err == nil {
					t.Fatalf("expected RoleBinding %#q not to exist", "patch-charts")
				}
				return
			}

			checkRoleBindingSubjects(t, k8sClientFake, "patch-charts", "giantswarm", tc.expectedSubjects)
		})
	}
}

func newFakeClients(runtimeObjects ...runtime.Object) *k8sclienttest.Clients {
	return k8sclienttest.NewClients(k8sclienttest.ClientsConfig{
		CtrlClient: clientfake.NewClientBuilder().WithScheme(scheme.Scheme).Build(),
		K8sClient:  clientgofake.NewSimpleClientset(runtimeObjects...),
	})
}

func checkRole(t *testing.T, k8sClient k8sclient.Interface, name, namespace string, expectedRules []rbacv1.PolicyRule) {
	t.Helper()

	role, err := k8sClient.K8sClient().RbacV1().Roles(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(expectedRules, role.Rules) {
		t.Fatalf("unexpected Rules - expected %v, received %v\n", expectedRules, role.Rules)
	}
}

func checkRoleBindingSubjects(t *testing.T, k8sClient k8sclient.Interface, name, namespace string, expectedSubjects []rbacv1.Subject) {
	t.Helper()

	roleBinding, err := k8sClient.K8sClient().RbacV1().RoleBindings(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(expectedSubjects, roleBinding.Subjects) {
		t.Fatalf("unexpected Subjects - expected %v, received %v\n", expectedSubjects, roleBinding.Subjects)
	}
}
