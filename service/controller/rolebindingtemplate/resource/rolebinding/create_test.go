package rolebinding

import (
	"reflect"
	"testing"

	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/rbac-operator/api/v1alpha1"
	"github.com/giantswarm/rbac-operator/pkg/project"
)

func TestGetRoleBindingFromTemplate(t *testing.T) {

	testCases := []struct {
		Name         string
		Template     *rbacv1.RoleBinding
		TemplateName string
		Namespace    string

		expectedRoleBinding *rbacv1.RoleBinding
		expectError         bool
	}{
		{
			Name:         "case0: no changes",
			Template:     getTestRoleBinding(),
			TemplateName: "something",
			Namespace:    "org-example",

			expectedRoleBinding: getTestRoleBinding(),
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
			Namespace:    "org-example",

			expectedRoleBinding: getTestRoleBinding(),
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
			Namespace:    "org-example",

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
			Namespace:    "org-example",

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
			Namespace:    "another-namespace",

			expectedRoleBinding: &rbacv1.RoleBinding{
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
					{Kind: "ServiceAccount", Name: "test-SA", Namespace: "another-namespace"},
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
			result, err := getRoleBindingFromTemplate(template, tc.Namespace)
			if !tc.expectError && err != nil {
				t.Fatalf("Expected success, got error %v", err)
			}
			if tc.expectError && err == nil {
				t.Fatalf("Expected error, got success")
			}

			if !reflect.DeepEqual(result, tc.expectedRoleBinding) {
				t.Fatalf("Expected %v to be equal to %v", result, tc.expectedRoleBinding)
			}
		})
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
