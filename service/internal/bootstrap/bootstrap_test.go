package bootstrap

import (
	"context"
	"testing"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/k8sclient/v7/pkg/k8scrdclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/service/internal/accessgroup"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	fakek8s "k8s.io/client-go/kubernetes/fake"
)

func Test_Bootstrap(t *testing.T) {

	testCases := []struct {
		Name                        string
		InitialObjects              []interface{}
		WriteAllCustomerGroups      []accessgroup.AccessGroup
		WriteAllGiantswarmGroups    []accessgroup.AccessGroup
		ExpectedRoleBindings        []rbacv1.RoleBinding
		ExpectedClusterRoleBindings []rbacv1.ClusterRoleBinding
		ExpectedError               error
	}{
		{
			Name:                     "case 0: Add new bindings with multiple subjects",
			WriteAllCustomerGroups:   []accessgroup.AccessGroup{{Name: "customers1"}, {Name: "customers2"}},
			WriteAllGiantswarmGroups: []accessgroup.AccessGroup{{Name: "giantswarm1"}, {Name: "giantswarm2"}},
			ExpectedRoleBindings: []rbacv1.RoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{Name: key.WriteAllCustomerGroupRoleBindingName()},
					Subjects:   []rbacv1.Subject{{Kind: "Group", Name: "customers1"}, {Kind: "Group", Name: "customers2"}},
				},
			},
			ExpectedClusterRoleBindings: []rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{Name: key.WriteOrganizationsCustomerGroupClusterRoleBindingName()},
					Subjects:   []rbacv1.Subject{{Kind: "Group", Name: "customers1"}, {Kind: "Group", Name: "customers2"}},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: key.ReadAllCustomerGroupClusterRoleBindingName()},
					Subjects:   []rbacv1.Subject{{Kind: "Group", Name: "customers1"}, {Kind: "Group", Name: "customers2"}},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: key.WriteAllGSGroupClusterRoleBindingName()},
					Subjects:   []rbacv1.Subject{{Kind: "Group", Name: "giantswarm1"}, {Kind: "Group", Name: "giantswarm2"}},
				},
			},
		},
		{
			Name: "case 1: Add multiple subjects to existing bindings",
			InitialObjects: []interface{}{
				rbacv1.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{Name: key.WriteAllCustomerGroupRoleBindingName(), Namespace: key.DefaultNamespaceName},
					Subjects:   []rbacv1.Subject{{Kind: "Group", Name: "customers"}},
				},
				rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{Name: key.WriteOrganizationsCustomerGroupClusterRoleBindingName()},
					Subjects:   []rbacv1.Subject{{Kind: "Group", Name: "customers"}},
				},
				rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{Name: key.ReadAllCustomerGroupClusterRoleBindingName()},
					Subjects:   []rbacv1.Subject{{Kind: "Group", Name: "customers"}},
				},
				rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{Name: key.WriteAllGSGroupClusterRoleBindingName()},
					Subjects:   []rbacv1.Subject{{Kind: "Group", Name: "giantswarm"}},
				},
			},
			WriteAllCustomerGroups:   []accessgroup.AccessGroup{{Name: "customers1"}, {Name: "customers2"}},
			WriteAllGiantswarmGroups: []accessgroup.AccessGroup{{Name: "giantswarm1"}, {Name: "giantswarm2"}},
			ExpectedRoleBindings: []rbacv1.RoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{Name: key.WriteAllCustomerGroupRoleBindingName()},
					Subjects:   []rbacv1.Subject{{Kind: "Group", Name: "customers1"}, {Kind: "Group", Name: "customers2"}},
				},
			},
			ExpectedClusterRoleBindings: []rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{Name: key.WriteOrganizationsCustomerGroupClusterRoleBindingName()},
					Subjects:   []rbacv1.Subject{{Kind: "Group", Name: "customers1"}, {Kind: "Group", Name: "customers2"}},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: key.ReadAllCustomerGroupClusterRoleBindingName()},
					Subjects:   []rbacv1.Subject{{Kind: "Group", Name: "customers1"}, {Kind: "Group", Name: "customers2"}},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: key.WriteAllGSGroupClusterRoleBindingName()},
					Subjects:   []rbacv1.Subject{{Kind: "Group", Name: "giantswarm1"}, {Kind: "Group", Name: "giantswarm2"}},
				},
			},
		},
		{
			Name:          "case 2: Fail in attempt to create/update bindings with empty subjects",
			ExpectedError: invalidConfigError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.TODO()

			loggerConfig := micrologger.Config{}
			logger, err := micrologger.New(loggerConfig)

			if err != nil {
				t.Fatalf("Failed to init logger: %s", err)
			}

			k8sClient, err := FakeK8sClient()
			if err != nil {
				t.Fatalf("Failed to init k8s client: %s", err)
			}

			config := Config{
				Logger:              logger,
				K8sClient:           k8sClient,
				CustomerAdminGroups: tc.WriteAllCustomerGroups,
				GSAdminGroups:       tc.WriteAllGiantswarmGroups,
			}

			if len(tc.InitialObjects) > 0 {
				for _, initialObject := range tc.InitialObjects {
					if clusterRoleBinding, ok := initialObject.(rbacv1.ClusterRoleBinding); ok {
						_, err = k8sClient.K8sClient().RbacV1().ClusterRoleBindings().Create(ctx, &clusterRoleBinding, metav1.CreateOptions{})
					} else if roleBinding, ok := initialObject.(rbacv1.RoleBinding); ok {
						_, err = k8sClient.K8sClient().RbacV1().RoleBindings(roleBinding.Namespace).Create(ctx, &roleBinding, metav1.CreateOptions{})
					}
					if err != nil {
						t.Fatalf("Failed to create initial object %v", initialObject)
					}
				}
			}

			bootstrap, err := New(config)

			if err == nil {
				err = bootstrap.Run(ctx)
			}

			if tc.ExpectedError != nil && err == nil {
				t.Fatalf("Did not receive an expected error: %s", tc.ExpectedError)
			} else if err != nil && err != tc.ExpectedError && microerror.Cause(err) != tc.ExpectedError {
				t.Fatalf("Received an unexpected error: %s", err)
			}

			clusterRoleBindingList, err := k8sClient.K8sClient().RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
			if err != nil {
				t.Fatalf("Failed to get cluster role bindings: %s", err)
			}
			clusterRoleBindingsShouldContain(t, tc.ExpectedClusterRoleBindings, clusterRoleBindingList.Items)

			roleBindingList, err := k8sClient.K8sClient().RbacV1().RoleBindings(key.DefaultNamespaceName).List(ctx, metav1.ListOptions{})
			if err != nil {
				t.Fatalf("Failed to get role bindings: %s", err)
			}
			roleBindingsShouldContain(t, tc.ExpectedRoleBindings, roleBindingList.Items)
		})
	}
}

func clusterRoleBindingsShouldContain(t *testing.T, expected []rbacv1.ClusterRoleBinding, actual []rbacv1.ClusterRoleBinding) {
	for _, expectedItem := range expected {
		hasItem := false
		for _, actualItem := range actual {
			if expectedItem.ObjectMeta.Name == actualItem.ObjectMeta.Name {
				hasItem = true
				subjectsShouldEqual(t, actualItem.Kind, actualItem.Name, expectedItem.Subjects, actualItem.Subjects)
				break
			}
		}
		if !hasItem {
			t.Fatalf("Missing cluster role binding %v", expectedItem)
		}
	}
}

func roleBindingsShouldContain(t *testing.T, expected []rbacv1.RoleBinding, actual []rbacv1.RoleBinding) {
	for _, expectedItem := range expected {
		hasItem := false
		for _, actualItem := range actual {
			if expectedItem.ObjectMeta.Name == actualItem.ObjectMeta.Name {
				hasItem = true
				subjectsShouldEqual(t, actualItem.Kind, actualItem.Name, expectedItem.Subjects, actualItem.Subjects)
				break
			}
		}
		if !hasItem {
			t.Fatalf("Missing role binding %v", expectedItem)
		}
	}
}

func subjectsShouldEqual(t *testing.T, kind string, name string, expected []rbacv1.Subject, actual []rbacv1.Subject) {
	if len(expected) != len(actual) {
		t.Fatalf("Subjects don't equal. Expected length %d, received %d", len(expected), len(actual))
	}
	for _, expectedSubject := range expected {
		hasSubject := false
		for _, actualSubject := range actual {
			if expectedSubject.Kind == actualSubject.Kind && expectedSubject.Name == actualSubject.Name {
				hasSubject = true
				break
			}
		}
		if !hasSubject {
			t.Fatalf("Missing subject %v in %s %s", expectedSubject, kind, name)
		}
	}
}

type fakeK8sClient struct {
	ctrlClient client.Client
	k8sClient  kubernetes.Interface
	scheme     *runtime.Scheme
}

func FakeK8sClient() (k8sclient.Interface, error) {
	var err error
	var k8sClient k8sclient.Interface
	{
		scheme := runtime.NewScheme()
		err = v1.AddToScheme(scheme)
		if err != nil {
			return nil, err
		}

		k8sClient = &fakeK8sClient{
			ctrlClient: fake.NewClientBuilder().WithObjects().Build(),
			k8sClient:  fakek8s.NewSimpleClientset(),
			scheme:     scheme,
		}
	}

	return k8sClient, nil
}

func (f *fakeK8sClient) CRDClient() k8scrdclient.Interface {
	return nil
}

func (f *fakeK8sClient) CtrlClient() client.Client {
	return f.ctrlClient
}

func (f *fakeK8sClient) DynClient() dynamic.Interface {
	return nil
}

func (f *fakeK8sClient) ExtClient() clientset.Interface {
	return nil
}

func (f *fakeK8sClient) K8sClient() kubernetes.Interface {
	return f.k8sClient
}

func (f *fakeK8sClient) RESTClient() rest.Interface {
	return nil
}

func (f *fakeK8sClient) RESTConfig() *rest.Config {
	return nil
}

func (f *fakeK8sClient) Scheme() *runtime.Scheme {
	return f.scheme
}
