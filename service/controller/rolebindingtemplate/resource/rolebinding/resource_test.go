package rolebinding

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/giantswarm/k8sclient/v8/pkg/k8sclienttest"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/micrologger/microloggertest"
	security "github.com/giantswarm/organization-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/giantswarm/rbac-operator/api/v1alpha1"
	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
)

func TestGetNamespacesFromScope(t *testing.T) {

	testCases := []struct {
		Name                 string
		MatchLabels          map[string]string
		MatchExpressions     []metav1.LabelSelectorRequirement
		ExistingOrgStructure []int // each number represents number of cluster ns in an org

		expectedNamespaces []string
		expectError        bool
	}{
		{
			Name:                 "case0: no matcher, no cluster namespaces",
			ExistingOrgStructure: []int{0, 0, 0},
			expectedNamespaces:   []string{"org-organization-0", "org-organization-1", "org-organization-2"},
		},
		{
			Name:                 "case1: no matcher, cluster namespaces",
			ExistingOrgStructure: []int{1, 2},
			expectedNamespaces:   []string{"org-organization-0", "cluster-0-org-0", "org-organization-1", "cluster-0-org-1", "cluster-1-org-1"},
		},
		{
			Name:                 "case2: matcher for uneven orgs, no cluster namespaces",
			MatchLabels:          map[string]string{"key": "value-1"},
			ExistingOrgStructure: []int{0, 0, 0},
			expectedNamespaces:   []string{"org-organization-1"},
		},
		{
			Name:                 "case3: matcher for uneven orgs, cluster namespaces",
			MatchLabels:          map[string]string{"key": "value-1"},
			ExistingOrgStructure: []int{1, 2, 3},
			expectedNamespaces:   []string{"org-organization-1", "cluster-0-org-1", "cluster-1-org-1"},
		},
		{
			Name:                 "case4: no matching orgs",
			MatchLabels:          map[string]string{"key": "value-2"},
			ExistingOrgStructure: []int{1, 2, 3, 1},
			expectedNamespaces:   []string{},
		},
		{
			Name:                 "case5: no orgs",
			ExistingOrgStructure: []int{},
			expectedNamespaces:   []string{},
		},
		{
			Name: "case6: matcher for uneven orgs, no cluster namespaces using matchExpressions",
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      "key",
					Operator: metav1.LabelSelectorOpIn,
					Values:   []string{"value-1"},
				},
			},
			ExistingOrgStructure: []int{0, 0, 0},
			expectedNamespaces:   []string{"org-organization-1"},
		},
		{
			Name: "case7: matcher for uneven orgs, cluster namespaces using matchExpressions",
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      "key",
					Operator: metav1.LabelSelectorOpNotIn,
					Values:   []string{"value-0"},
				},
			},
			ExistingOrgStructure: []int{1, 2, 3},
			expectedNamespaces:   []string{"org-organization-1", "cluster-0-org-1", "cluster-1-org-1"},
		},
		{
			Name: "case8: using matchExpressions with different operators",
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      "key",
					Operator: metav1.LabelSelectorOpNotIn,
					Values:   []string{"value-0"},
				},
				{
					Key:      "key2",
					Operator: metav1.LabelSelectorOpIn,
					Values:   []string{"value-1"},
				},
			},
			ExistingOrgStructure: []int{1, 2, 3},
			expectedNamespaces:   []string{},
		},
		{
			Name:        "case9: using matchExpressions and matchLabels",
			MatchLabels: map[string]string{"key": "value-1"},
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      "key",
					Operator: metav1.LabelSelectorOpIn,
					Values:   []string{"value-1", "value-0"},
				},
			},
			ExistingOrgStructure: []int{1, 2, 3},
			expectedNamespaces:   []string{"org-organization-1", "cluster-0-org-1", "cluster-1-org-1"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {

			scopes := v1alpha1.RoleBindingTemplateScopes{
				OrganizationSelector: v1alpha1.ScopeSelector{
					MatchLabels:      tc.MatchLabels,
					MatchExpressions: tc.MatchExpressions,
				},
			}

			fakeClient, err := getTestClient(tc.ExistingOrgStructure)
			if err != nil {
				t.Fatal(err)
			}

			r := &Resource{
				k8sClient: fakeClient,
				logger:    microloggertest.New(),
			}
			result, err := r.getNamespacesFromScope(context.Background(), scopes)
			if !tc.expectError && err != nil {
				t.Fatalf("Expected success, got error %v", err)
			}
			if tc.expectError && err == nil {
				t.Fatalf("Expected error, got success")
			}

			if !reflect.DeepEqual(result, tc.expectedNamespaces) {
				t.Fatalf("Expected %v to be equal to %v", result, tc.expectedNamespaces)
			}
		})
	}
}

func TestGetLabelSelectorFromScopes(t *testing.T) {
	testCases := []struct {
		Name             string
		Scopes           v1alpha1.RoleBindingTemplateScopes
		ExpectedSelector string
	}{
		{
			Name:             "case0: no matchers",
			Scopes:           v1alpha1.RoleBindingTemplateScopes{},
			ExpectedSelector: "",
		},
		{
			Name: "case1: matchLabels",
			Scopes: v1alpha1.RoleBindingTemplateScopes{
				OrganizationSelector: v1alpha1.ScopeSelector{
					MatchLabels: map[string]string{"key": "value"},
				},
			},
			ExpectedSelector: "key=value",
		},
		{
			Name: "case2: matchExpressions",
			Scopes: v1alpha1.RoleBindingTemplateScopes{
				OrganizationSelector: v1alpha1.ScopeSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "key",
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{"value1", "value2"},
						},
					},
				},
			},
			ExpectedSelector: "key in (value1,value2)",
		},
		{
			Name: "case3: matchLabels and matchExpressions",
			Scopes: v1alpha1.RoleBindingTemplateScopes{
				OrganizationSelector: v1alpha1.ScopeSelector{
					MatchLabels: map[string]string{"key": "value"},
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "key",
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{"value1", "value2"},
						},
					},
				},
			},
			ExpectedSelector: "key=value,key in (value1,value2)",
		},
		{
			Name: "case4: matchLabels and matchExpressions with different operators",
			Scopes: v1alpha1.RoleBindingTemplateScopes{
				OrganizationSelector: v1alpha1.ScopeSelector{
					MatchLabels: map[string]string{"key": "value"},
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "key",
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{"value1", "value2"},
						},
						{
							Key:      "key2",
							Operator: metav1.LabelSelectorOpNotIn,
							Values:   []string{"value3", "value4"},
						},
					},
				},
			},
			ExpectedSelector: "key=value,key in (value1,value2),key2 notin (value3,value4)",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result, err := getLabelSelectorFromScopes(tc.Scopes)
			if err != nil {
				t.Fatalf("Expected success, got error %v", err)
			}
			if result.String() != tc.ExpectedSelector {
				t.Fatalf("Expected %v to be equal to %v", result, tc.ExpectedSelector)
			}
		})
	}
}

func getTestClient(structure []int) (*k8sclienttest.Clients, error) {
	schemeBuilder := runtime.SchemeBuilder{
		security.AddToScheme,
		v1alpha1.AddToScheme,
		corev1.AddToScheme,
	}

	err := schemeBuilder.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}

	orgs, namespaces := getTestObjects(structure)

	return k8sclienttest.NewClients(k8sclienttest.ClientsConfig{
		CtrlClient: clientfake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithRuntimeObjects(orgs...).
			Build(),
		K8sClient: clientgofake.NewSimpleClientset(namespaces...),
	}), nil
}

func getTestObjects(orgs []int) ([]runtime.Object, []runtime.Object) {
	namespaces := []runtime.Object{}
	organizations := []runtime.Object{}
	for org, clusters := range orgs {

		orgNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("org-organization-%v", org),
			},
		}
		namespaces = append(namespaces, orgNamespace)

		organization := &security.Organization{
			ObjectMeta: metav1.ObjectMeta{
				Name: pkgkey.OrganizationName(orgNamespace.Name),
				Labels: map[string]string{
					label.Organization: pkgkey.OrganizationName(orgNamespace.Name),
					"key":              fmt.Sprintf("value-%v", org%2),
				},
			},
			Status: security.OrganizationStatus{
				Namespace: orgNamespace.Name,
			},
		}
		organizations = append(organizations, organization)

		for cluster := 0; cluster < clusters; cluster++ {
			clusterNamespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("cluster-%v-org-%v", cluster, org),
					Labels: map[string]string{
						label.Organization: pkgkey.OrganizationName(orgNamespace.Name),
						label.Cluster:      fmt.Sprintf("cluster-%v-org-%v", cluster, org),
					},
				},
			}
			namespaces = append(namespaces, clusterNamespace)
		}
	}
	return organizations, namespaces
}
