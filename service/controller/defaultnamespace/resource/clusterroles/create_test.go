package clusterroles

import (
	"context"
	"testing"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/micrologger/microloggertest"
	security "github.com/giantswarm/organization-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/service/controller/defaultnamespace/defaultnamespacetest"
	"github.com/giantswarm/rbac-operator/service/controller/defaultnamespace/key"
)

func Test_ClusterRoleCreation(t *testing.T) {

	testCases := []struct {
		Name                 string
		Provider             string
		InitialObjects       []runtime.Object
		InitialResources     []metav1.APIResource
		ExpectedClusterRoles []*rbacv1.ClusterRole
	}{
		{
			Name:                 "case0: Create static cluster roles on AWS",
			Provider:             "aws",
			ExpectedClusterRoles: newExpectedClusterRoles([]rbacv1.PolicyRule{}, true),
		},
		{
			Name:     "case1: Update static cluster roles on AWS",
			Provider: "aws",
			InitialObjects: []runtime.Object{
				defaultnamespacetest.NewClusterRole(pkgkey.DefaultReadAllPermissionsName, []rbacv1.PolicyRule{}),
				defaultnamespacetest.NewClusterRole(pkgkey.WriteOrganizationsPermissionsName, defaultnamespacetest.NewSingletonRulesNoResources()),
				defaultnamespacetest.NewClusterRole(pkgkey.WriteFluxResourcesPermissionsName, defaultnamespacetest.NewSingletonRulesNoResources()),
				defaultnamespacetest.NewClusterRole(pkgkey.WriteClientCertsPermissionsName, defaultnamespacetest.NewSingletonRulesNoResources()),
				defaultnamespacetest.NewClusterRole(pkgkey.WriteSilencesPermissionsName, defaultnamespacetest.NewSingletonRulesNoResources()),
				defaultnamespacetest.NewClusterRole(pkgkey.WriteAWSClusterRoleIdentityPermissionsName, defaultnamespacetest.NewSingletonRulesNoResources()),
			},
			ExpectedClusterRoles: newExpectedClusterRoles([]rbacv1.PolicyRule{}, true),
		},
		{
			Name:     "case2: Update read-all cluster role with new resources on AWS",
			Provider: "aws",
			InitialObjects: []runtime.Object{
				defaultnamespacetest.NewClusterRole(pkgkey.DefaultReadAllPermissionsName, []rbacv1.PolicyRule{
					defaultnamespacetest.NewSingleResourceRule("security.giantswarm.io", "organizations"),
					defaultnamespacetest.NewSingleResourceRule("", "pods/log"),
				}),
			},
			InitialResources: []metav1.APIResource{
				defaultnamespacetest.NewApiResource("release.giantswarm.io", "v1alpha1", "Release", "releases", true),
			},
			ExpectedClusterRoles: newExpectedClusterRoles([]rbacv1.PolicyRule{
				defaultnamespacetest.NewSingleResourceRule("release.giantswarm.io", "releases"),
			}, true),
		},
		{
			Name:                 "case3: Create static cluster roles on non-AWS provider",
			Provider:             "azure",
			ExpectedClusterRoles: newExpectedClusterRoles([]rbacv1.PolicyRule{}, false),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.TODO()

			var err error

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
					K8sClient: defaultnamespacetest.NewClientSet(tc.InitialObjects...).WithResources(tc.InitialResources...),
				})
			}

			clusterRoles, err := New(Config{
				K8sClient: k8sClientFake,
				Logger:    microloggertest.New(),
				Provider:  tc.Provider,
			})

			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: pkgkey.DefaultNamespaceName}}
			err = clusterRoles.EnsureCreated(context.TODO(), namespace)
			if err != nil {
				t.Fatalf("failed to ensure creation of cluster roles: %s", err)
			}

			clusterRoleList, err := k8sClientFake.K8sClient().RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}
			defaultnamespacetest.ClusterRolesShouldEqual(t, tc.ExpectedClusterRoles, clusterRoleList.Items)
		})
	}
}

func Test_ClusterRoleLabeling(t *testing.T) {
	testCases := []struct {
		Name                        string
		Provider                    string
		ExpectedLabeledClusterRoles []string
		ExpectedLabels              map[string]string
	}{
		{
			Name:                        "case0: Check labeling of cluster roles visible to the UI",
			Provider:                    "aws",
			ExpectedLabeledClusterRoles: key.DefaultClusterRolesToDisplayInUI(),
			ExpectedLabels: map[string]string{
				label.DisplayInUserInterface: "true",
				label.ManagedBy:              "Kubernetes",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.TODO()

			var err error

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
					K8sClient: defaultnamespacetest.NewClientSet(defaultnamespacetest.NewClusterAdminRole()),
				})
			}

			clusterRoles, err := New(Config{
				K8sClient: k8sClientFake,
				Logger:    microloggertest.New(),
				Provider:  tc.Provider,
			})

			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			err = clusterRoles.EnsureCreated(ctx, defaultnamespacetest.NewDefaultNamespace())
			if err != nil {
				t.Fatal("failed to ensure creation of cluster roles")
			}

			for _, clusterRoleName := range tc.ExpectedLabeledClusterRoles {
				clusterRole, err := k8sClientFake.K8sClient().RbacV1().ClusterRoles().Get(ctx, clusterRoleName, metav1.GetOptions{})
				if err != nil {
					t.Fatalf("error == %#v, want nil", err)
				}
				for labelKey, labelValue := range tc.ExpectedLabels {
					if clusterRole.Labels[labelKey] != labelValue {
						t.Fatalf("missing label %s=%s in cluster role %s", labelKey, labelValue, clusterRoleName)
					}
				}
			}
		})
	}
}

func newExpectedClusterRoles(readAllRules []rbacv1.PolicyRule, includeAWS bool) []*rbacv1.ClusterRole {
	roles := []*rbacv1.ClusterRole{
		defaultnamespacetest.NewClusterRole(pkgkey.DefaultReadAllPermissionsName, append(readAllRules, defaultnamespacetest.NewSingleResourceRule(
			"", "pods/log",
		))),
		defaultnamespacetest.NewClusterRole(pkgkey.WriteOrganizationsPermissionsName, defaultnamespacetest.NewSingletonRules(
			[]string{"security.giantswarm.io"},
			[]string{"organizations"},
		)),
		defaultnamespacetest.NewClusterRole(pkgkey.WriteFluxResourcesPermissionsName, defaultnamespacetest.NewSingletonRules(
			[]string{
				"helm.toolkit.fluxcd.io",
				"image.toolkit.fluxcd.io",
				"kustomizations.kustomize.toolkit.fluxcd.io",
				"notification.toolkit.fluxcd.io",
				"source.toolkit.fluxcd.io",
			},
			[]string{
				"alerts",
				"buckets",
				"gitrepositories",
				"helmcharts",
				"helmreleases",
				"helmrepositories",
				"imagepolicies",
				"imagerepositories",
				"imageupdateautomations",
				"kustomizations",
				"providers",
				"receivers",
			},
		)),
		defaultnamespacetest.NewClusterRole(pkgkey.WriteClientCertsPermissionsName, defaultnamespacetest.NewSingletonRules(
			[]string{"core.giantswarm.io"},
			[]string{"certconfigs"},
		)),
		defaultnamespacetest.NewClusterRole(pkgkey.WriteSilencesPermissionsName, defaultnamespacetest.NewSingletonRules(
			[]string{"monitoring.giantswarm.io"},
			[]string{"silences"},
		)),
	}

	if includeAWS {
		roles = append(roles, defaultnamespacetest.NewClusterRole(pkgkey.WriteAWSClusterRoleIdentityPermissionsName, defaultnamespacetest.NewSingletonRules(
			[]string{"infrastructure.cluster.x-k8s.io"},
			[]string{"awsclusterroleidentities"},
		)))
	}

	return roles
}
