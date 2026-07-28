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

// the Role rules the resource is expected to grant for managing PolicyExceptions.
var expectedWritePolicyExceptionsRules = []rbacv1.PolicyRule{
	{
		APIGroups: []string{"kyverno.io"},
		Resources: []string{"policyexceptions"},
		Verbs:     []string{"*"},
	},
}

// the namespaces the `write-policy-exceptions` Role and RoleBinding are expected in.
var writePolicyExceptionsNamespaces = []string{"kube-system", "giantswarm", "policy-exceptions"}

func automationSubject(namespace string) rbacv1.Subject {
	return rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      "automation",
		Namespace: namespace,
	}
}

func sharedRoleBinding(name, namespace string, subjects []rbacv1.Subject) *rbacv1.RoleBinding {
	return test.NewRoleBinding(name, namespace, map[string]string{
		"kind": "Role",
		"name": name,
	}, subjects)
}

func patchChartsRoleBinding(subjects []rbacv1.Subject) *rbacv1.RoleBinding {
	return sharedRoleBinding("patch-charts", "giantswarm", subjects)
}

func sharedRole(name, namespace string, rules []rbacv1.PolicyRule) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Rules: rules,
	}
}

func patchChartsRole(rules []rbacv1.PolicyRule) *rbacv1.Role {
	return sharedRole("patch-charts", "giantswarm", rules)
}

// the namespaces the `write-policy-exceptions` Role is ensured in have to exist
// for the resource to act on them.
func writePolicyExceptionsNamespaceObjects() []runtime.Object {
	var objects []runtime.Object
	for _, namespace := range writePolicyExceptionsNamespaces {
		objects = append(objects, test.NewGenericNamespace(namespace))
	}

	return objects
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
		{
			name:         "case 3: correct the Role's rules when they have drifted from the desired state",
			orgNamespace: "customer",
			existingResources: []runtime.Object{
				patchChartsRole([]rbacv1.PolicyRule{
					{
						APIGroups: []string{"application.giantswarm.io"},
						Resources: []string{"charts"},
						Verbs:     []string{"get"},
					},
				}),
			},
			expectedSubjects: []rbacv1.Subject{automationSubject("org-customer")},
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

func Test_EnsureCreated_WritePolicyExceptions(t *testing.T) {
	testCases := []struct {
		name              string
		orgNamespace      string
		existingResources []runtime.Object
		expectedSubjects  []rbacv1.Subject
	}{
		{
			name:             "case 0: create the Role and RoleBinding in all namespaces when they do not exist yet",
			orgNamespace:     "customer",
			expectedSubjects: []rbacv1.Subject{automationSubject("org-customer")},
		},
		{
			name:         "case 1: append the org's automation SA to existing RoleBindings",
			orgNamespace: "customer",
			existingResources: []runtime.Object{
				sharedRoleBinding("write-policy-exceptions", "giantswarm", []rbacv1.Subject{automationSubject("org-acme")}),
				sharedRoleBinding("write-policy-exceptions", "kube-system", []rbacv1.Subject{automationSubject("org-acme")}),
				sharedRoleBinding("write-policy-exceptions", "policy-exceptions", []rbacv1.Subject{automationSubject("org-acme")}),
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
				sharedRoleBinding("write-policy-exceptions", "giantswarm", []rbacv1.Subject{automationSubject("org-customer")}),
				sharedRoleBinding("write-policy-exceptions", "kube-system", []rbacv1.Subject{automationSubject("org-customer")}),
				sharedRoleBinding("write-policy-exceptions", "policy-exceptions", []rbacv1.Subject{automationSubject("org-customer")}),
			},
			expectedSubjects: []rbacv1.Subject{automationSubject("org-customer")},
		},
		{
			name:         "case 3: correct the Roles' rules when they have drifted from the desired state",
			orgNamespace: "customer",
			existingResources: []runtime.Object{
				sharedRole("write-policy-exceptions", "giantswarm", []rbacv1.PolicyRule{
					{
						APIGroups: []string{"kyverno.io"},
						Resources: []string{"policyexceptions"},
						Verbs:     []string{"get"},
					},
				}),
				sharedRole("write-policy-exceptions", "kube-system", nil),
			},
			expectedSubjects: []rbacv1.Subject{automationSubject("org-customer")},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			orgNamespace := test.NewOrgNamespace(tc.orgNamespace)

			runtimeObjects := []runtime.Object{orgNamespace}
			runtimeObjects = append(runtimeObjects, writePolicyExceptionsNamespaceObjects()...)
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

			for _, namespace := range writePolicyExceptionsNamespaces {
				checkRole(t, k8sClientFake, "write-policy-exceptions", namespace, expectedWritePolicyExceptionsRules)
				checkRoleBindingSubjects(t, k8sClientFake, "write-policy-exceptions", namespace, tc.expectedSubjects)
			}
		})
	}
}

// Clusters without the policy apps have no `policy-exceptions` namespace. There,
// the resource must skip that namespace instead of failing.
func Test_EnsureCreated_WritePolicyExceptions_MissingNamespace(t *testing.T) {
	orgNamespace := test.NewOrgNamespace("customer")

	k8sClientFake := newFakeClients(
		orgNamespace,
		test.NewGenericNamespace("giantswarm"),
		test.NewGenericNamespace("kube-system"),
	)

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

	expectedSubjects := []rbacv1.Subject{automationSubject("org-customer")}
	for _, namespace := range []string{"giantswarm", "kube-system"} {
		checkRole(t, k8sClientFake, "write-policy-exceptions", namespace, expectedWritePolicyExceptionsRules)
		checkRoleBindingSubjects(t, k8sClientFake, "write-policy-exceptions", namespace, expectedSubjects)
	}

	if _, err := k8sClientFake.K8sClient().RbacV1().Roles("policy-exceptions").Get(context.TODO(), "write-policy-exceptions", metav1.GetOptions{}); err == nil {
		t.Fatalf("expected Role %#q not to exist in namespace %s", "write-policy-exceptions", "policy-exceptions")
	}
}

func Test_EnsureDeleted_WritePolicyExceptions(t *testing.T) {
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
				sharedRoleBinding("write-policy-exceptions", "giantswarm", []rbacv1.Subject{
					automationSubject("org-acme"),
					automationSubject("org-customer"),
				}),
				sharedRoleBinding("write-policy-exceptions", "kube-system", []rbacv1.Subject{
					automationSubject("org-customer"),
					automationSubject("org-acme"),
				}),
				sharedRoleBinding("write-policy-exceptions", "policy-exceptions", []rbacv1.Subject{
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
				sharedRoleBinding("write-policy-exceptions", "giantswarm", []rbacv1.Subject{automationSubject("org-customer")}),
				sharedRoleBinding("write-policy-exceptions", "kube-system", []rbacv1.Subject{automationSubject("org-customer")}),
				sharedRoleBinding("write-policy-exceptions", "policy-exceptions", []rbacv1.Subject{automationSubject("org-customer")}),
			},
			expectedSubjects: []rbacv1.Subject{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			orgNamespace := test.NewOrgNamespace(tc.orgNamespace)

			runtimeObjects := []runtime.Object{orgNamespace}
			runtimeObjects = append(runtimeObjects, writePolicyExceptionsNamespaceObjects()...)
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

			for _, namespace := range writePolicyExceptionsNamespaces {
				checkRoleBindingSubjects(t, k8sClientFake, "write-policy-exceptions", namespace, tc.expectedSubjects)
			}
		})
	}
}

// The RoleBindings must not be created on deletion when they never existed.
func Test_EnsureDeleted_WritePolicyExceptions_NoRoleBindings(t *testing.T) {
	orgNamespace := test.NewOrgNamespace("customer")

	runtimeObjects := []runtime.Object{orgNamespace}
	runtimeObjects = append(runtimeObjects, writePolicyExceptionsNamespaceObjects()...)

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

	for _, namespace := range writePolicyExceptionsNamespaces {
		_, err := k8sClientFake.K8sClient().RbacV1().RoleBindings(namespace).Get(context.TODO(), "write-policy-exceptions", metav1.GetOptions{})
		if err == nil {
			t.Fatalf("expected RoleBinding %#q not to exist in namespace %s", "write-policy-exceptions", namespace)
		}
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
