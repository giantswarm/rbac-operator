package controller

import (
	"context"
	"reflect"
	"testing"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	security "github.com/giantswarm/organization-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/giantswarm/rbac-operator/api/v1alpha1"
	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/pkg/project"
	"github.com/giantswarm/rbac-operator/service/controller/defaultnamespace/defaultnamespacetest"
)

func TestGetRoleBindingFromTemplate(t *testing.T) {

	testCases := []struct {
		Name         string
		Template     *rbacv1.RoleBinding
		TemplateName string
		Namespaces   []string

		expectedRoleBindings []*rbacv1.RoleBinding
		expectError          bool
	}{
		{
			Name:         "case0: no changes",
			Template:     getTestRoleBinding(),
			TemplateName: "something",
			Namespaces:   []string{"org-example"},

			expectedRoleBindings: []*rbacv1.RoleBinding{getTestRoleBinding()},
		},

		{
			Name: "case1: add defaults",
			Template: &rbacv1.RoleBinding{
				RoleRef: rbacv1.RoleRef{
					Name: "example",
					Kind: "ClusterRole",
				},
				Subjects: []rbacv1.Subject{
					{Kind: "Group", Name: "test-group"},
					{Kind: "ServiceAccount", Name: "test-SA"},
				},
			},
			TemplateName: "something",
			Namespaces:   []string{"org-example"},

			expectedRoleBindings: []*rbacv1.RoleBinding{getTestRoleBinding()},
		},

		{
			Name: "case2: forbidden roleRef kind",
			Template: &rbacv1.RoleBinding{
				RoleRef: rbacv1.RoleRef{
					Name: "example",
					Kind: "AnotherKind",
				},
				Subjects: []rbacv1.Subject{
					{Kind: "Group", Name: "test-group"},
					{Kind: "ServiceAccount", Name: "test-SA"},
				},
			},
			TemplateName: "something",
			Namespaces:   []string{"org-example"},

			expectError: true,
		},

		{
			Name: "case3: no roleRef",
			Template: &rbacv1.RoleBinding{
				Subjects: []rbacv1.Subject{
					{Kind: "Group", Name: "test-group"},
					{Kind: "ServiceAccount", Name: "test-SA"},
				},
			},
			TemplateName: "something",
			Namespaces:   []string{"org-example"},

			expectError: true,
		},

		{
			Name: "case4: retain values",
			Template: &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "org-example",
					Labels: map[string]string{
						"the-label": "the-value",
					},
					Annotations: map[string]string{
						annotation.Notes: "There is already a note here",
					},
				},
				RoleRef: rbacv1.RoleRef{
					Name:     "example",
					Kind:     "ClusterRole",
					APIGroup: "rbac.authorization.k8s.io",
				},
				Subjects: []rbacv1.Subject{
					{Kind: "Group", Name: "test-group"},
					{Kind: "ServiceAccount", Name: "test-SA", Namespace: "org-example"},
					{Kind: "ServiceAccount", Name: "test-SA"},
				},
			},
			TemplateName: "another-name",
			Namespaces:   []string{"another-namespace"},

			expectedRoleBindings: []*rbacv1.RoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "name",
						Namespace: "another-namespace",
						Labels: map[string]string{
							"the-label":     "the-value",
							label.ManagedBy: project.Name(),
						},
						Annotations: map[string]string{
							annotation.Notes: "There is already a note here",
						},
					},
					RoleRef: rbacv1.RoleRef{
						Name:     "example",
						Kind:     "ClusterRole",
						APIGroup: "rbac.authorization.k8s.io",
					},
					Subjects: []rbacv1.Subject{
						{Kind: "Group", Name: "test-group"},
						{Kind: "ServiceAccount", Name: "test-SA", Namespace: "org-example"},
						{Kind: "ServiceAccount", Name: "test-SA", Namespace: "another-namespace"},
					},
				},
			},
		},

		{
			Name: "case5: apply template to multiple namespaces",
			Template: &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "org-example",
					Labels: map[string]string{
						"the-label": "the-value",
					},
					Annotations: map[string]string{
						annotation.Notes: "There is already a note here",
					},
				},
				RoleRef: rbacv1.RoleRef{
					Name:     "example",
					Kind:     "ClusterRole",
					APIGroup: "rbac.authorization.k8s.io",
				},
				Subjects: []rbacv1.Subject{
					{Kind: "Group", Name: "test-group"},
					{Kind: "ServiceAccount", Name: "test-SA", Namespace: "org-example"},
					{Kind: "ServiceAccount", Name: "test-SA"},
				},
			},
			TemplateName: "another-name",
			Namespaces:   []string{"another-namespace-1", "another-namespace-2"},

			expectedRoleBindings: []*rbacv1.RoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "name",
						Namespace: "another-namespace-1",
						Labels: map[string]string{
							"the-label":     "the-value",
							label.ManagedBy: project.Name(),
						},
						Annotations: map[string]string{
							annotation.Notes: "There is already a note here",
						},
					},
					RoleRef: rbacv1.RoleRef{
						Name:     "example",
						Kind:     "ClusterRole",
						APIGroup: "rbac.authorization.k8s.io",
					},
					Subjects: []rbacv1.Subject{
						{Kind: "Group", Name: "test-group"},
						{Kind: "ServiceAccount", Name: "test-SA", Namespace: "org-example"},
						{Kind: "ServiceAccount", Name: "test-SA", Namespace: "another-namespace-1"},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "name",
						Namespace: "another-namespace-2",
						Labels: map[string]string{
							"the-label":     "the-value",
							label.ManagedBy: project.Name(),
						},
						Annotations: map[string]string{
							annotation.Notes: "There is already a note here",
						},
					},
					RoleRef: rbacv1.RoleRef{
						Name:     "example",
						Kind:     "ClusterRole",
						APIGroup: "rbac.authorization.k8s.io",
					},
					Subjects: []rbacv1.Subject{
						{Kind: "Group", Name: "test-group"},
						{Kind: "ServiceAccount", Name: "test-SA", Namespace: "org-example"},
						{Kind: "ServiceAccount", Name: "test-SA", Namespace: "another-namespace-2"},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {

			template := v1alpha1.RoleBindingTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: tc.TemplateName,
				},
				Spec: v1alpha1.RoleBindingTemplateSpec{
					Template: v1alpha1.RoleBindingTemplateResource{
						ObjectMeta: tc.Template.ObjectMeta,
						RoleRef:    tc.Template.RoleRef,
						Subjects:   tc.Template.Subjects,
					},
				},
			}

			var results []*rbacv1.RoleBinding

			for _, namespace := range tc.Namespaces {
				result := getRoleBindingFromTemplate(&template, namespace)

				results = append(results, result)
			}

			if !tc.expectError {
				if len(tc.expectedRoleBindings) != len(results) {
					t.Fatalf("Expected %d role bindings, got %d\n", len(tc.expectedRoleBindings), len(results))
				}

				for _, expected := range tc.expectedRoleBindings {
					hasEqualResult := false
					for _, result := range results {
						if reflect.DeepEqual(expected, result) {
							hasEqualResult = true
							break
						}
					}
					if !hasEqualResult {
						t.Fatalf("Did not find expected role binding\n%v\n\n ...in:\n%v\n", expected, results)
					}
				}
			}
		})
	}
}

func TestEnsureCreated(t *testing.T) {
	testCases := []struct {
		Name          string
		Template      *v1alpha1.RoleBindingTemplate
		Organizations []string

		expectedRoleBindings []*rbacv1.RoleBinding
		expectError          bool
	}{
		{
			Name: "case1: create role binding in all orgs",
			Template: &v1alpha1.RoleBindingTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "something",
				},
				Spec: v1alpha1.RoleBindingTemplateSpec{
					Template: v1alpha1.RoleBindingTemplateResource{
						ObjectMeta: metav1.ObjectMeta{
							Name: "something",
						},
						RoleRef: rbacv1.RoleRef{
							Name: "example",
							Kind: "ClusterRole",
						},
						Subjects: []rbacv1.Subject{
							{Kind: "Group", Name: "test-group"},
							{Kind: "ServiceAccount", Name: "test-SA"},
						},
					},
				},
			},
			Organizations:        []string{"example", "example-2"},
			expectedRoleBindings: []*rbacv1.RoleBinding{getTestRoleBinding(), getTestRoleBindingInNS("org-example-2")},
		},
		{
			Name: "case2: create role binding in one org",
			Template: &v1alpha1.RoleBindingTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "something",
				},
				Spec: v1alpha1.RoleBindingTemplateSpec{
					Template: v1alpha1.RoleBindingTemplateResource{
						ObjectMeta: metav1.ObjectMeta{
							Name: "something",
						},
						RoleRef: rbacv1.RoleRef{
							Name: "example",
							Kind: "ClusterRole",
						},
						Subjects: []rbacv1.Subject{
							{Kind: "Group", Name: "test-group"},
							{Kind: "ServiceAccount", Name: "test-SA"},
						},
					},
					Scopes: v1alpha1.RoleBindingTemplateScopes{
						OrganizationSelector: metav1.LabelSelector{
							MatchLabels: map[string]string{"name": "example"},
						},
					},
				},
			},
			Organizations:        []string{"example", "example-2"},
			expectedRoleBindings: []*rbacv1.RoleBinding{getTestRoleBinding()},
		},
		{
			Name: "case3: no matching orgs",
			Template: &v1alpha1.RoleBindingTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "something",
				},
				Spec: v1alpha1.RoleBindingTemplateSpec{
					Template: v1alpha1.RoleBindingTemplateResource{
						ObjectMeta: metav1.ObjectMeta{
							Name: "something",
						},
						RoleRef: rbacv1.RoleRef{
							Name: "example",
							Kind: "ClusterRole",
						},
						Subjects: []rbacv1.Subject{
							{Kind: "Group", Name: "test-group"},
							{Kind: "ServiceAccount", Name: "test-SA"},
						},
					},
					Scopes: v1alpha1.RoleBindingTemplateScopes{
						OrganizationSelector: metav1.LabelSelector{
							MatchLabels: map[string]string{"name": "example-3"},
						},
					},
				},
			},
			Organizations:        []string{"example", "example-2"},
			expectedRoleBindings: []*rbacv1.RoleBinding{},
		},
		{
			Name: "case4: do not create rolebinding in org-giantswarm namespace",
			Template: &v1alpha1.RoleBindingTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "something",
				},
				Spec: v1alpha1.RoleBindingTemplateSpec{
					Template: v1alpha1.RoleBindingTemplateResource{
						ObjectMeta: metav1.ObjectMeta{
							Name: "something",
						},
						RoleRef: rbacv1.RoleRef{
							Name: "example",
							Kind: "ClusterRole",
						},
						Subjects: []rbacv1.Subject{
							{Kind: "Group", Name: "test-group"},
						},
					},
				},
			},
			Organizations: []string{"example", "giantswarm"},
			expectedRoleBindings: []*rbacv1.RoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "something",
						Namespace: "org-example",
						Labels: map[string]string{
							label.ManagedBy: project.Name(),
						},
						Annotations: map[string]string{
							annotation.Notes: "Generated based on RoleBindingTemplate something",
						},
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "RoleBinding",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					RoleRef: rbacv1.RoleRef{
						Name:     "example",
						Kind:     "ClusterRole",
						APIGroup: "rbac.authorization.k8s.io",
					},
					Subjects: []rbacv1.Subject{
						{Kind: "Group", Name: "test-group"},
					},
				},
			},
		},
		{
			Name: "case5: create rolebinding in org-giantswarm namespace",
			Template: &v1alpha1.RoleBindingTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "something",
				},
				Spec: v1alpha1.RoleBindingTemplateSpec{
					Template: v1alpha1.RoleBindingTemplateResource{
						ObjectMeta: metav1.ObjectMeta{
							Name: "something",
						},
						RoleRef: rbacv1.RoleRef{
							Name: "example",
							Kind: "ClusterRole",
						},
						Subjects: []rbacv1.Subject{
							{Kind: "ServiceAccount", Name: "test-SA", Namespace: pkgkey.FluxNamespaceName},
						},
					},
				},
			},
			Organizations: []string{"example", "giantswarm"},
			expectedRoleBindings: []*rbacv1.RoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "something",
						Namespace: "org-giantswarm",
						Labels: map[string]string{
							label.ManagedBy: project.Name(),
						},
						Annotations: map[string]string{
							annotation.Notes: "Generated based on RoleBindingTemplate something",
						},
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "RoleBinding",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					RoleRef: rbacv1.RoleRef{
						Name:     "example",
						Kind:     "ClusterRole",
						APIGroup: "rbac.authorization.k8s.io",
					},
					Subjects: []rbacv1.Subject{
						{Kind: "ServiceAccount", Name: "test-SA", Namespace: pkgkey.FluxNamespaceName},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "something",
						Namespace: "org-example",
						Labels: map[string]string{
							label.ManagedBy: project.Name(),
						},
						Annotations: map[string]string{
							annotation.Notes: "Generated based on RoleBindingTemplate something",
						},
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "RoleBinding",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					RoleRef: rbacv1.RoleRef{
						Name:     "example",
						Kind:     "ClusterRole",
						APIGroup: "rbac.authorization.k8s.io",
					},
					Subjects: []rbacv1.Subject{
						{Kind: "ServiceAccount", Name: "test-SA", Namespace: pkgkey.FluxNamespaceName},
					},
				},
			},
		},
		{
			Name: "case6: create rolebinding in org-giantswarm namespace and remove invalid subjects",
			Template: &v1alpha1.RoleBindingTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "something",
				},
				Spec: v1alpha1.RoleBindingTemplateSpec{
					Template: v1alpha1.RoleBindingTemplateResource{
						ObjectMeta: metav1.ObjectMeta{
							Name: "something",
						},
						RoleRef: rbacv1.RoleRef{
							Name: "example",
							Kind: "ClusterRole",
						},
						Subjects: []rbacv1.Subject{
							{Kind: "ServiceAccount", Name: "test-SA-1", Namespace: pkgkey.FluxNamespaceName},
							{Kind: "ServiceAccount", Name: "test-SA-2", Namespace: "org-example"},
							{Kind: "ServiceAccount", Name: "test-SA-3", Namespace: "org-giantswarm"},
							{Kind: "ServiceAccount", Name: "test-SA-4"},
							{Kind: "Group", Name: "test-group"},
						},
					},
				},
			},
			Organizations: []string{"example", "giantswarm"},
			expectedRoleBindings: []*rbacv1.RoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "something",
						Namespace: "org-example",
						Labels: map[string]string{
							label.ManagedBy: project.Name(),
						},
						Annotations: map[string]string{
							annotation.Notes: "Generated based on RoleBindingTemplate something",
						},
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "RoleBinding",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					RoleRef: rbacv1.RoleRef{
						Name:     "example",
						Kind:     "ClusterRole",
						APIGroup: "rbac.authorization.k8s.io",
					},
					Subjects: []rbacv1.Subject{
						{Kind: "ServiceAccount", Name: "test-SA-1", Namespace: pkgkey.FluxNamespaceName},
						{Kind: "ServiceAccount", Name: "test-SA-2", Namespace: "org-example"},
						{Kind: "ServiceAccount", Name: "test-SA-3", Namespace: "org-giantswarm"},
						{Kind: "ServiceAccount", Name: "test-SA-4", Namespace: "org-example"},
						{Kind: "Group", Name: "test-group"},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "something",
						Namespace: "org-giantswarm",
						Labels: map[string]string{
							label.ManagedBy: project.Name(),
						},
						Annotations: map[string]string{
							annotation.Notes: "Generated based on RoleBindingTemplate something",
						},
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "RoleBinding",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					RoleRef: rbacv1.RoleRef{
						Name:     "example",
						Kind:     "ClusterRole",
						APIGroup: "rbac.authorization.k8s.io",
					},
					Subjects: []rbacv1.Subject{
						{Kind: "ServiceAccount", Name: "test-SA-1", Namespace: pkgkey.FluxNamespaceName},
						{Kind: "ServiceAccount", Name: "test-SA-3", Namespace: "org-giantswarm"},
						{Kind: "ServiceAccount", Name: "test-SA-4", Namespace: "org-giantswarm"},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			objects := []runtime.Object{tc.Template}
			namespaces := []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "example",
					},
				},
			}
			for _, org := range tc.Organizations {
				objects = append(objects, getTestOrganization(org))
				namespaces = append(namespaces, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "org-" + org,
					},
				})
			}
			var k8sClientFake *k8sclienttest.Clients
			{
				schemeBuilder := runtime.SchemeBuilder{
					security.AddToScheme,
					v1alpha1.AddToScheme,
				}
				if err := schemeBuilder.AddToScheme(scheme.Scheme); err != nil {
					t.Fatal(err)
				}

				k8sClientFake = k8sclienttest.NewClients(k8sclienttest.ClientsConfig{
					CtrlClient: clientfake.NewClientBuilder().
						WithScheme(scheme.Scheme).
						WithRuntimeObjects(objects...).
						WithStatusSubresource(&v1alpha1.RoleBindingTemplate{}).
						Build(),
					K8sClient: clientgofake.NewSimpleClientset(namespaces...),
				})
			}

			r := &RoleBindingTemplateReconciler{
				Client: k8sClientFake.CtrlClient(),
				Scheme: k8sClientFake.CtrlClient().Scheme(),
			}

			ctx := context.Background()
			_, err := r.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: tc.Template.Name},
			})
			if !tc.expectError && err != nil {
				t.Fatalf("Expected success, got error %v", err)
			}
			if tc.expectError && err == nil {
				t.Fatalf("Expected error, got success")
			}
			roleBindingList := &rbacv1.RoleBindingList{}
			if err := k8sClientFake.CtrlClient().List(ctx, roleBindingList); err != nil {
				t.Fatalf("failed to get role bindings: %s", err)
			}
			defaultnamespacetest.RoleBindingsShouldEqual(t, tc.expectedRoleBindings, roleBindingList.Items)
		})
	}
}

func getTestOrganization(name string) *security.Organization {
	return &security.Organization{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"name": name,
			},
		},
		Status: security.OrganizationStatus{
			Namespace: "org-" + name,
		},
	}
}

func getTestRoleBinding() *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "something",
			Namespace: "org-example",
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
			Annotations: map[string]string{
				annotation.Notes: "Generated based on RoleBindingTemplate something",
			},
		},
		RoleRef: rbacv1.RoleRef{
			Name:     "example",
			Kind:     "ClusterRole",
			APIGroup: "rbac.authorization.k8s.io",
		},
		Subjects: []rbacv1.Subject{
			{Kind: "Group", Name: "test-group"},
			{Kind: "ServiceAccount", Name: "test-SA", Namespace: "org-example"},
		},
	}
}

func getTestRoleBindingInNS(namespace string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "something",
			Namespace: "org-example",
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
			Annotations: map[string]string{
				annotation.Notes: "Generated based on RoleBindingTemplate something",
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		RoleRef: rbacv1.RoleRef{
			Name:     "example",
			Kind:     "ClusterRole",
			APIGroup: "rbac.authorization.k8s.io",
		},
		Subjects: []rbacv1.Subject{
			{Kind: "Group", Name: "test-group"},
			{Kind: "ServiceAccount", Name: "test-SA", Namespace: "org-example"},
		},
	}
}
