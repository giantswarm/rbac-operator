package controller_test

import (
	"cmp"
	"context"
	"fmt"
	"reflect"
	"slices"
	"testing"

	k8smetadata "github.com/giantswarm/k8smetadata/pkg/label"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/giantswarm/rbac-operator/internal/controller"
	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	pkglabel "github.com/giantswarm/rbac-operator/pkg/label"
	"github.com/giantswarm/rbac-operator/pkg/project"
)

// the Role rules expected for the App to HelmRelease migration.
var expectedPatchChartsRules = []rbacv1.PolicyRule{{
	APIGroups: []string{"application.giantswarm.io"},
	Resources: []string{"charts"},
	Verbs:     []string{"list", "get", "patch"},
}}

// the Role rules expected for managing PolicyExceptions.
var expectedWritePolicyExceptionsRules = []rbacv1.PolicyRule{{
	APIGroups: []string{"kyverno.io"},
	Resources: []string{"policyexceptions"},
	Verbs:     []string{"*"},
}}

// Ported from service/controller/rbac/resource/automation/create_test.go.
//
// Unlike the old resource, which appended this org's subject to whatever the
// RoleBinding held, the reconciler rebuilds the subjects from the org
// namespaces that exist. Orgs referenced by existing subjects therefore have to
// exist in the fixtures.
func TestOrgNamespaceReconciler_PatchCharts(t *testing.T) {
	testCases := []struct {
		name              string
		existingResources []client.Object
		expectedSubjects  []rbacv1.Subject
	}{
		{
			name:             "case 0: create the Role and RoleBinding when they do not exist yet",
			expectedSubjects: []rbacv1.Subject{automationSubject("org-customer")},
		},
		{
			name: "case 1: add the org's automation SA next to the other orgs' ones",
			existingResources: []client.Object{
				orgNamespace("acme"),
				sharedRoleBinding(pkgkey.PatchChartsPermissionsName, pkgkey.GiantSwarmNamespaceName, automationSubject("org-acme")),
			},
			expectedSubjects: []rbacv1.Subject{automationSubject("org-acme"), automationSubject("org-customer")},
		},
		{
			name: "case 2: do not add a duplicate subject when the org's automation SA is already present",
			existingResources: []client.Object{
				orgNamespace("acme"),
				sharedRoleBinding(pkgkey.PatchChartsPermissionsName, pkgkey.GiantSwarmNamespaceName,
					automationSubject("org-acme"), automationSubject("org-customer")),
			},
			expectedSubjects: []rbacv1.Subject{automationSubject("org-acme"), automationSubject("org-customer")},
		},
		{
			name: "case 3: correct the Role's rules when they have drifted from the desired state",
			existingResources: []client.Object{
				sharedRole(pkgkey.PatchChartsPermissionsName, pkgkey.GiantSwarmNamespaceName, []rbacv1.PolicyRule{{
					APIGroups: []string{"application.giantswarm.io"},
					Resources: []string{"charts"},
					Verbs:     []string{"get"},
				}}),
			},
			expectedSubjects: []rbacv1.Subject{automationSubject("org-customer")},
		},
		{
			name: "case 4: drop subjects of orgs that no longer exist",
			existingResources: []client.Object{
				sharedRoleBinding(pkgkey.PatchChartsPermissionsName, pkgkey.GiantSwarmNamespaceName,
					automationSubject("org-gone"), automationSubject("org-customer")),
			},
			expectedSubjects: []rbacv1.Subject{automationSubject("org-customer")},
		},
		{
			name: "case 5: sort subjects that are out of order",
			existingResources: []client.Object{
				orgNamespace("acme"),
				sharedRoleBinding(pkgkey.PatchChartsPermissionsName, pkgkey.GiantSwarmNamespaceName,
					automationSubject("org-customer"), automationSubject("org-acme")),
			},
			expectedSubjects: []rbacv1.Subject{automationSubject("org-acme"), automationSubject("org-customer")},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			objects := append([]client.Object{orgNamespace("customer"), genericNamespace(pkgkey.GiantSwarmNamespaceName)}, tc.existingResources...)
			r, c := newOrgNamespaceReconciler(t, nil, objects...)

			reconcileNamespace(t, r, "org-customer")

			checkRole(t, c, pkgkey.PatchChartsPermissionsName, pkgkey.GiantSwarmNamespaceName, expectedPatchChartsRules)
			checkRoleBinding(t, c, pkgkey.PatchChartsPermissionsName, pkgkey.GiantSwarmNamespaceName,
				roleRef("Role", pkgkey.PatchChartsPermissionsName), tc.expectedSubjects)
		})
	}
}

// Ported from service/controller/rbac/resource/automation/create_test.go.
func TestOrgNamespaceReconciler_WritePolicyExceptions(t *testing.T) {
	testCases := []struct {
		name              string
		existingResources []client.Object
		expectedSubjects  []rbacv1.Subject
	}{
		{
			name:             "case 0: create the Role and RoleBinding in all namespaces when they do not exist yet",
			expectedSubjects: []rbacv1.Subject{automationSubject("org-customer")},
		},
		{
			name: "case 1: add the org's automation SA next to the other orgs' ones",
			existingResources: []client.Object{
				orgNamespace("acme"),
				sharedRoleBinding(pkgkey.WritePolicyExceptionsPermissionsName, pkgkey.GiantSwarmNamespaceName, automationSubject("org-acme")),
				sharedRoleBinding(pkgkey.WritePolicyExceptionsPermissionsName, pkgkey.KubeSystemNamespaceName, automationSubject("org-acme")),
				sharedRoleBinding(pkgkey.WritePolicyExceptionsPermissionsName, pkgkey.PolicyExceptionsNamespaceName, automationSubject("org-acme")),
			},
			expectedSubjects: []rbacv1.Subject{automationSubject("org-acme"), automationSubject("org-customer")},
		},
		{
			name: "case 2: do not add a duplicate subject when the org's automation SA is already present",
			existingResources: []client.Object{
				sharedRoleBinding(pkgkey.WritePolicyExceptionsPermissionsName, pkgkey.GiantSwarmNamespaceName, automationSubject("org-customer")),
				sharedRoleBinding(pkgkey.WritePolicyExceptionsPermissionsName, pkgkey.KubeSystemNamespaceName, automationSubject("org-customer")),
				sharedRoleBinding(pkgkey.WritePolicyExceptionsPermissionsName, pkgkey.PolicyExceptionsNamespaceName, automationSubject("org-customer")),
			},
			expectedSubjects: []rbacv1.Subject{automationSubject("org-customer")},
		},
		{
			name: "case 3: correct the Roles' rules when they have drifted from the desired state",
			existingResources: []client.Object{
				sharedRole(pkgkey.WritePolicyExceptionsPermissionsName, pkgkey.GiantSwarmNamespaceName, []rbacv1.PolicyRule{{
					APIGroups: []string{"kyverno.io"},
					Resources: []string{"policyexceptions"},
					Verbs:     []string{"get"},
				}}),
				sharedRole(pkgkey.WritePolicyExceptionsPermissionsName, pkgkey.KubeSystemNamespaceName, nil),
			},
			expectedSubjects: []rbacv1.Subject{automationSubject("org-customer")},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			objects := []client.Object{orgNamespace("customer")}
			for _, namespace := range pkgkey.WritePolicyExceptionsNamespaces {
				objects = append(objects, genericNamespace(namespace))
			}
			objects = append(objects, tc.existingResources...)
			r, c := newOrgNamespaceReconciler(t, nil, objects...)

			reconcileNamespace(t, r, "org-customer")

			for _, namespace := range pkgkey.WritePolicyExceptionsNamespaces {
				checkRole(t, c, pkgkey.WritePolicyExceptionsPermissionsName, namespace, expectedWritePolicyExceptionsRules)
				checkRoleBinding(t, c, pkgkey.WritePolicyExceptionsPermissionsName, namespace,
					roleRef("Role", pkgkey.WritePolicyExceptionsPermissionsName), tc.expectedSubjects)
			}
		})
	}
}

// Ported from service/controller/rbac/resource/automation/create_test.go.
// Clusters without the policy apps have no `policy-exceptions` namespace. There,
// the reconciler must skip that namespace instead of failing.
func TestOrgNamespaceReconciler_WritePolicyExceptions_MissingNamespace(t *testing.T) {
	r, c := newOrgNamespaceReconciler(t, nil,
		orgNamespace("customer"),
		genericNamespace(pkgkey.GiantSwarmNamespaceName),
		genericNamespace(pkgkey.KubeSystemNamespaceName),
	)

	reconcileNamespace(t, r, "org-customer")

	for _, namespace := range []string{pkgkey.GiantSwarmNamespaceName, pkgkey.KubeSystemNamespaceName} {
		checkRole(t, c, pkgkey.WritePolicyExceptionsPermissionsName, namespace, expectedWritePolicyExceptionsRules)
		checkRoleBinding(t, c, pkgkey.WritePolicyExceptionsPermissionsName, namespace,
			roleRef("Role", pkgkey.WritePolicyExceptionsPermissionsName), []rbacv1.Subject{automationSubject("org-customer")})
	}

	checkNotFound(t, c, &rbacv1.Role{}, pkgkey.WritePolicyExceptionsPermissionsName, pkgkey.PolicyExceptionsNamespaceName)
	checkNotFound(t, c, &rbacv1.RoleBinding{}, pkgkey.WritePolicyExceptionsPermissionsName, pkgkey.PolicyExceptionsNamespaceName)
}

// Ported from the EnsureDeleted tests in
// service/controller/rbac/resource/automation/create_test.go. The reconciler
// drops a namespace's automation SA from the shared RoleBindings both while the
// namespace is terminating and once it is gone.
func TestOrgNamespaceReconciler_SharedRBACOnDeletion(t *testing.T) {
	sharedBindings := []struct{ name, namespace string }{
		{pkgkey.PatchChartsPermissionsName, pkgkey.GiantSwarmNamespaceName},
		{pkgkey.WritePolicyExceptionsPermissionsName, pkgkey.GiantSwarmNamespaceName},
		{pkgkey.WritePolicyExceptionsPermissionsName, pkgkey.KubeSystemNamespaceName},
		{pkgkey.WritePolicyExceptionsPermissionsName, pkgkey.PolicyExceptionsNamespaceName},
	}

	testCases := []struct {
		name             string
		customer         *corev1.Namespace // nil: the namespace is gone
		otherOrgs        []string
		expectedSubjects []rbacv1.Subject
	}{
		{
			name:             "case 0: namespace gone, remove the org's automation SA and keep the others",
			otherOrgs:        []string{"acme"},
			expectedSubjects: []rbacv1.Subject{automationSubject("org-acme")},
		},
		{
			name:             "case 1: namespace gone, leave an empty subject list when the org was the only subject",
			expectedSubjects: nil,
		},
		{
			name:             "case 2: namespace terminating, remove the org's automation SA and keep the others",
			customer:         terminating(orgNamespace("customer"), "test.giantswarm.io/keep"),
			otherOrgs:        []string{"acme"},
			expectedSubjects: []rbacv1.Subject{automationSubject("org-acme")},
		},
		{
			name:             "case 3: namespace terminating, leave an empty subject list when the org was the only subject",
			customer:         terminating(orgNamespace("customer"), "test.giantswarm.io/keep"),
			expectedSubjects: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var objects []client.Object
			for _, namespace := range pkgkey.WritePolicyExceptionsNamespaces {
				objects = append(objects, genericNamespace(namespace))
			}
			subjects := []rbacv1.Subject{automationSubject("org-customer")}
			for _, org := range tc.otherOrgs {
				objects = append(objects, orgNamespace(org))
				subjects = append(subjects, automationSubject("org-"+org))
			}
			for _, b := range sharedBindings {
				objects = append(objects, sharedRoleBinding(b.name, b.namespace, subjects...))
			}
			if tc.customer != nil {
				objects = append(objects, tc.customer)
			}
			r, c := newOrgNamespaceReconciler(t, nil, objects...)

			reconcileNamespace(t, r, "org-customer")

			for _, b := range sharedBindings {
				checkRoleBinding(t, c, b.name, b.namespace, roleRef("Role", b.name), tc.expectedSubjects)
			}
		})
	}
}

// Ported from service/controller/rbac/resource/automation/create_test.go and
// extended to all ClusterRoles bound to the automation SA. The bindings are
// controlled by the org namespace, so they are garbage-collected with it.
func TestOrgNamespaceReconciler_AutomationClusterRoleBindings(t *testing.T) {
	ns := orgNamespace("customer")
	r, c := newOrgNamespaceReconciler(t, nil, ns)

	reconcileNamespace(t, r, "org-customer")

	sa := &corev1.ServiceAccount{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: pkgkey.AutomationServiceAccountName, Namespace: "org-customer"}, sa); err != nil {
		t.Fatalf("expected automation ServiceAccount: %v", err)
	}
	checkManagedAndControlledBy(t, sa, "org-customer")

	for _, binding := range []struct{ name, clusterRole string }{
		{pkgkey.WriteSilencesAutomationSAinNSRoleBindingName("org-customer"), pkgkey.WriteSilencesPermissionsName},
		{pkgkey.KamajiDatastoreManagerAutomationSAinNSRoleBindingName("org-customer"), pkgkey.KamajiDatastoreManagerPermissionsName},
		{pkgkey.ClusterAppChartLookupsAutomationSAinNSRoleBindingName("org-customer"), pkgkey.ClusterAppChartLookupsPermissionsName},
	} {
		crb := checkClusterRoleBinding(t, c, binding.name,
			roleRef("ClusterRole", binding.clusterRole), []rbacv1.Subject{automationSubject("org-customer")})
		checkManagedAndControlledBy(t, crb, "org-customer")
	}
}

// Ported from service/controller/rbac/resource/namespaceauth/create_test.go,
// plus cases for configurations without customer admin groups and for legacy
// customer namespaces.
func TestOrgNamespaceReconciler_CustomerAdminAccess(t *testing.T) {
	acme := rbacv1.Subject{Kind: "Group", Name: "customer:acme:Employees"}
	giantswarm := rbacv1.Subject{Kind: "Group", Name: "customer:giantswarm:Employees"}

	testCases := []struct {
		name                string
		namespace           *corev1.Namespace
		existingResources   []client.Object
		customerAdminGroups []rbacv1.Subject
		expectedSubjects    []rbacv1.Subject // nil: the RoleBinding must not exist
	}{
		{
			name:                "case 0: create a new role binding in case it does not exist",
			namespace:           orgNamespace("customer"),
			customerAdminGroups: []rbacv1.Subject{giantswarm},
			expectedSubjects:    []rbacv1.Subject{giantswarm},
		},
		{
			name:      "case 1: replace subjects in an existing role binding",
			namespace: orgNamespace("customer"),
			existingResources: []client.Object{
				customerAdminRoleBinding("org-customer", acme),
			},
			customerAdminGroups: []rbacv1.Subject{giantswarm},
			expectedSubjects:    []rbacv1.Subject{giantswarm},
		},
		{
			name:      "case 2: add subjects to an existing role binding",
			namespace: orgNamespace("customer"),
			existingResources: []client.Object{
				customerAdminRoleBinding("org-customer", acme),
			},
			customerAdminGroups: []rbacv1.Subject{acme, giantswarm},
			expectedSubjects:    []rbacv1.Subject{acme, giantswarm},
		},
		{
			name:      "case 3: remove subjects from an existing role binding",
			namespace: orgNamespace("customer"),
			existingResources: []client.Object{
				customerAdminRoleBinding("org-customer", acme, giantswarm),
			},
			customerAdminGroups: []rbacv1.Subject{giantswarm},
			expectedSubjects:    []rbacv1.Subject{giantswarm},
		},
		{
			name:      "case 4: do not overwrite the role binding in case there are no changes",
			namespace: orgNamespace("customer"),
			existingResources: []client.Object{
				customerAdminRoleBinding("org-customer", acme, giantswarm),
			},
			customerAdminGroups: []rbacv1.Subject{acme, giantswarm},
			expectedSubjects:    []rbacv1.Subject{acme, giantswarm},
		},
		{
			name:                "case 5: do not create rolebinding in protected namespace",
			namespace:           orgNamespace("giantswarm"),
			customerAdminGroups: []rbacv1.Subject{giantswarm},
		},
		{
			name:      "case 6: delete rolebinding in protected namespace",
			namespace: orgNamespace("giantswarm"),
			existingResources: []client.Object{
				customerAdminRoleBinding("org-giantswarm", giantswarm),
			},
			customerAdminGroups: []rbacv1.Subject{giantswarm},
		},
		{
			name:      "case 7: delete rolebinding when no customer admin groups are configured",
			namespace: orgNamespace("customer"),
			existingResources: []client.Object{
				customerAdminRoleBinding("org-customer", giantswarm),
			},
		},
		{
			name:                "case 8: create role binding in legacy customer namespace",
			namespace:           customerNamespace("legacy"),
			customerAdminGroups: []rbacv1.Subject{giantswarm},
			expectedSubjects:    []rbacv1.Subject{giantswarm},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			objects := append([]client.Object{tc.namespace}, tc.existingResources...)
			r, c := newOrgNamespaceReconciler(t, tc.customerAdminGroups, objects...)

			reconcileNamespace(t, r, tc.namespace.Name)

			name := pkgkey.WriteAllCustomerGroupRoleBindingName()
			if tc.expectedSubjects == nil {
				checkNotFound(t, c, &rbacv1.RoleBinding{}, name, tc.namespace.Name)
				return
			}
			checkRoleBinding(t, c, name, tc.namespace.Name,
				roleRef("ClusterRole", pkgkey.ClusterAdminClusterRoleName), tc.expectedSubjects)
		})
	}
}

// Ported from service/controller/rbac/resource/namespaceauth/create_test.go
// (case 5), extended to the bindings that grant the organization's members
// read access.
func TestOrgNamespaceReconciler_MemberReadAccess(t *testing.T) {
	customerAdmins := rbacv1.Subject{Kind: "Group", Name: "customer-admins"}
	devs := rbacv1.Subject{Kind: "Group", Name: "devs"}
	readers := rbacv1.Subject{Kind: "Group", Name: "readers"}
	admin := rbacv1.Subject{Kind: "User", Name: "admin"}

	r, c := newOrgNamespaceReconciler(t, []rbacv1.Subject{customerAdmins},
		orgNamespace("customer"),
		inputRoleBinding("org-customer", "devs", roleRef("Role", "something"), devs),
		inputRoleBinding("org-customer", "readers", roleRef("ClusterRole", pkgkey.DefaultReadAllPermissionsName), readers),
		inputRoleBinding("org-customer", "admins", roleRef("ClusterRole", pkgkey.ClusterAdminClusterRoleName), admin),
	)

	reconcileNamespace(t, r, "org-customer")

	// The customer-admin binding is created before the RoleBindings are listed,
	// so its subjects count as members in the same pass.
	members := []rbacv1.Subject{customerAdmins, devs, readers, admin}
	sortSubjectsForTest(members)

	cr := &rbacv1.ClusterRole{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: pkgkey.OrganizationReadClusterRoleName("org-customer")}, cr); err != nil {
		t.Fatalf("expected organization read ClusterRole: %v", err)
	}
	expectedRules := []rbacv1.PolicyRule{{
		APIGroups:     []string{"security.giantswarm.io"},
		Resources:     []string{"organizations"},
		ResourceNames: []string{"customer"},
		Verbs:         []string{"get"},
	}}
	if !reflect.DeepEqual(expectedRules, cr.Rules) {
		t.Fatalf("unexpected Rules - expected %v, received %v", expectedRules, cr.Rules)
	}
	checkManagedAndControlledBy(t, cr, "org-customer")

	crb := checkClusterRoleBinding(t, c, pkgkey.OrganizationReadOrganizationClusterRoleBindingName("customer"),
		roleRef("ClusterRole", pkgkey.OrganizationReadClusterRoleName("org-customer")), members)
	checkManagedAndControlledBy(t, crb, "org-customer")

	crb = checkClusterRoleBinding(t, c, pkgkey.OrganizationReadReleasesClusterRoleBindingName("customer"),
		roleRef("ClusterRole", pkgkey.ReadReleasesRole), members)
	checkManagedAndControlledBy(t, crb, "org-customer")

	rb := checkRoleBinding(t, c, pkgkey.OrganizationReadDefaultCatalogsRoleBindingName("customer"), pkgkey.DefaultNamespaceName,
		roleRef("Role", pkgkey.ReadDefaultCatalogsRole), members)
	checkManagedAndControlledBy(t, rb, "org-customer")
}

func TestOrgNamespaceReconciler_ClusterNamespaceAccess(t *testing.T) {
	customerAdmins := rbacv1.Subject{Kind: "Group", Name: "customer-admins"}
	devs := rbacv1.Subject{Kind: "Group", Name: "devs"}
	readers := rbacv1.Subject{Kind: "Group", Name: "readers"}
	admin := rbacv1.Subject{Kind: "User", Name: "admin"}

	r, c := newOrgNamespaceReconciler(t, []rbacv1.Subject{customerAdmins},
		orgNamespace("customer"),
		inputRoleBinding("org-customer", "devs", roleRef("Role", "something"), devs),
		inputRoleBinding("org-customer", "readers", roleRef("ClusterRole", pkgkey.DefaultReadAllPermissionsName), readers),
		inputRoleBinding("org-customer", "admins", roleRef("ClusterRole", pkgkey.ClusterAdminClusterRoleName), admin),
	)

	reconcileNamespace(t, r, "org-customer")

	checkRoleBinding(t, c, pkgkey.OrganizationReadClusterNamespaceRoleBindingName("customer"), "org-customer",
		roleRef("ClusterRole", pkgkey.ReadClusterNamespaceAppsRole), []rbacv1.Subject{readers})

	// customer-admins is bound to cluster-admin by the customer-admin binding.
	writers := []rbacv1.Subject{customerAdmins, admin}
	sortSubjectsForTest(writers)
	checkRoleBinding(t, c, pkgkey.OrganizationWriteClusterNamespaceRoleBindingName("customer"), "org-customer",
		roleRef("ClusterRole", pkgkey.WriteClusterNamespaceAppsRole), writers)
}

// Removing every RoleBinding from the org namespace revokes the derived access.
func TestOrgNamespaceReconciler_RevokesAccess(t *testing.T) {
	devs := rbacv1.Subject{Kind: "Group", Name: "devs"}
	admin := rbacv1.Subject{Kind: "User", Name: "admin"}
	inputs := []client.Object{
		inputRoleBinding("org-customer", "devs", roleRef("Role", "something"), devs),
		inputRoleBinding("org-customer", "admins", roleRef("ClusterRole", pkgkey.ClusterAdminClusterRoleName), admin),
	}
	r, c := newOrgNamespaceReconciler(t, nil, append([]client.Object{orgNamespace("customer")}, inputs...)...)

	reconcileNamespace(t, r, "org-customer")
	for _, obj := range inputs {
		if err := c.Delete(context.Background(), obj); err != nil {
			t.Fatalf("deleting input RoleBinding: %v", err)
		}
	}
	// The derived cluster-namespace RoleBindings live in the org namespace too,
	// so it takes a second pass until the members derived from them are gone.
	reconcileNamespace(t, r, "org-customer")
	reconcileNamespace(t, r, "org-customer")

	checkClusterRoleBinding(t, c, pkgkey.OrganizationReadOrganizationClusterRoleBindingName("customer"),
		roleRef("ClusterRole", pkgkey.OrganizationReadClusterRoleName("org-customer")), nil)
	checkClusterRoleBinding(t, c, pkgkey.OrganizationReadReleasesClusterRoleBindingName("customer"),
		roleRef("ClusterRole", pkgkey.ReadReleasesRole), nil)
	checkRoleBinding(t, c, pkgkey.OrganizationReadDefaultCatalogsRoleBindingName("customer"), pkgkey.DefaultNamespaceName,
		roleRef("Role", pkgkey.ReadDefaultCatalogsRole), nil)
	checkRoleBinding(t, c, pkgkey.OrganizationWriteClusterNamespaceRoleBindingName("customer"), "org-customer",
		roleRef("ClusterRole", pkgkey.WriteClusterNamespaceAppsRole), nil)
}

// A second reconcile without changes must not rewrite any binding.
func TestOrgNamespaceReconciler_IsIdempotent(t *testing.T) {
	r, c := newOrgNamespaceReconciler(t, []rbacv1.Subject{{Kind: "Group", Name: "customer-admins"}},
		orgNamespace("customer"),
		orgNamespace("acme"),
		orgNamespace("zeta"),
		genericNamespace(pkgkey.GiantSwarmNamespaceName),
		inputRoleBinding("org-customer", "devs", roleRef("Role", "something"),
			rbacv1.Subject{Kind: "Group", Name: "zz"}, rbacv1.Subject{Kind: "Group", Name: "aa"}),
	)

	reconcileNamespace(t, r, "org-customer")
	before := resourceVersions(t, c)
	reconcileNamespace(t, r, "org-customer")
	after := resourceVersions(t, c)

	if !reflect.DeepEqual(before, after) {
		t.Fatalf("second reconcile changed objects - before %v, after %v", before, after)
	}
}

// The finalizers of the removed operatorkit controllers would otherwise block
// namespace deletion forever.
func TestOrgNamespaceReconciler_RemovesLegacyFinalizers(t *testing.T) {
	ns := orgNamespace("customer")
	ns.Finalizers = []string{
		"operatorkit.giantswarm.io/rbac-operator-rbac-controller",
		"operatorkit.giantswarm.io/rbac-operator-crossplane-namespace-controller",
		"test.giantswarm.io/keep",
	}
	r, c := newOrgNamespaceReconciler(t, nil, ns)

	reconcileNamespace(t, r, "org-customer")

	got := &corev1.Namespace{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "org-customer"}, got); err != nil {
		t.Fatalf("getting namespace: %v", err)
	}
	if expected := []string{"test.giantswarm.io/keep"}; !reflect.DeepEqual(expected, got.Finalizers) {
		t.Fatalf("unexpected Finalizers - expected %v, received %v", expected, got.Finalizers)
	}
}

// Legacy customer namespaces without the org- prefix have no organization: they
// only get customer-admin access.
func TestOrgNamespaceReconciler_LegacyCustomerNamespace(t *testing.T) {
	r, c := newOrgNamespaceReconciler(t, []rbacv1.Subject{{Kind: "Group", Name: "customer-admins"}},
		customerNamespace("legacy"),
		inputRoleBinding("legacy", "admins", roleRef("ClusterRole", pkgkey.ClusterAdminClusterRoleName), rbacv1.Subject{Kind: "User", Name: "admin"}),
	)

	reconcileNamespace(t, r, "legacy")

	checkRoleBinding(t, c, pkgkey.WriteAllCustomerGroupRoleBindingName(), "legacy",
		roleRef("ClusterRole", pkgkey.ClusterAdminClusterRoleName), []rbacv1.Subject{{Kind: "Group", Name: "customer-admins"}})

	checkNotFound(t, c, &corev1.ServiceAccount{}, pkgkey.AutomationServiceAccountName, "legacy")
	checkNotFound(t, c, &rbacv1.ClusterRoleBinding{}, pkgkey.WriteSilencesAutomationSAinNSRoleBindingName("legacy"), "")
	checkNotFound(t, c, &rbacv1.ClusterRole{}, pkgkey.OrganizationReadClusterRoleName("legacy"), "")
	checkNotFound(t, c, &rbacv1.ClusterRoleBinding{}, pkgkey.OrganizationReadReleasesClusterRoleBindingName("legacy"), "")
	checkNotFound(t, c, &rbacv1.RoleBinding{}, pkgkey.OrganizationReadDefaultCatalogsRoleBindingName("legacy"), pkgkey.DefaultNamespaceName)
	checkNotFound(t, c, &rbacv1.RoleBinding{}, pkgkey.PatchChartsPermissionsName, pkgkey.GiantSwarmNamespaceName)
}

func TestOrgNamespaceReconciler_SkipsNonOrgNamespace(t *testing.T) {
	r, c := newOrgNamespaceReconciler(t, []rbacv1.Subject{{Kind: "Group", Name: "customer-admins"}},
		genericNamespace("unrelated"),
	)

	reconcileNamespace(t, r, "unrelated")

	checkNotFound(t, c, &corev1.ServiceAccount{}, pkgkey.AutomationServiceAccountName, "unrelated")
	checkNotFound(t, c, &rbacv1.ClusterRole{}, pkgkey.OrganizationReadClusterRoleName("unrelated"), "")
}

func newOrgNamespaceReconciler(t *testing.T, writeAllCustomerSubjects []rbacv1.Subject, objects ...client.Object) (*controller.OrgNamespaceReconciler, client.Client) {
	t.Helper()
	c := clientfake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objects...).Build()
	return &controller.OrgNamespaceReconciler{
		Client:                   c,
		Scheme:                   scheme.Scheme,
		WriteAllCustomerSubjects: writeAllCustomerSubjects,
	}, c
}

func reconcileNamespace(t *testing.T, r *controller.OrgNamespaceReconciler, name string) {
	t.Helper()
	if _, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Name: name}}); err != nil {
		t.Fatalf("reconcile %s: %v", name, err)
	}
}

func orgNamespace(org string) *corev1.Namespace {
	return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name:   "org-" + org,
		Labels: map[string]string{k8smetadata.Organization: org},
	}}
}

func customerNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name:   name,
		Labels: map[string]string{pkglabel.LegacyCustomer: name},
	}}
}

func genericNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
}

func terminating(ns *corev1.Namespace, finalizer string) *corev1.Namespace {
	now := metav1.Now()
	ns.DeletionTimestamp = &now
	ns.Finalizers = append(ns.Finalizers, finalizer)
	return ns
}

func automationSubject(namespace string) rbacv1.Subject {
	return rbacv1.Subject{Kind: "ServiceAccount", Name: pkgkey.AutomationServiceAccountName, Namespace: namespace}
}

func roleRef(kind, name string) rbacv1.RoleRef {
	return rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: kind, Name: name}
}

func sharedRole(name, namespace string, rules []rbacv1.PolicyRule) *rbacv1.Role {
	return &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}, Rules: rules}
}

func sharedRoleBinding(name, namespace string, subjects ...rbacv1.Subject) *rbacv1.RoleBinding {
	return inputRoleBinding(namespace, name, roleRef("Role", name), subjects...)
}

func customerAdminRoleBinding(namespace string, subjects ...rbacv1.Subject) *rbacv1.RoleBinding {
	return inputRoleBinding(namespace, pkgkey.WriteAllCustomerGroupRoleBindingName(),
		roleRef("ClusterRole", pkgkey.ClusterAdminClusterRoleName), subjects...)
}

func inputRoleBinding(namespace, name string, ref rbacv1.RoleRef, subjects ...rbacv1.Subject) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		RoleRef:    ref,
		Subjects:   subjects,
	}
}

func checkRole(t *testing.T, c client.Client, name, namespace string, expectedRules []rbacv1.PolicyRule) {
	t.Helper()
	role := &rbacv1.Role{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: name, Namespace: namespace}, role); err != nil {
		t.Fatalf("getting Role %s/%s: %v", namespace, name, err)
	}
	if !reflect.DeepEqual(expectedRules, role.Rules) {
		t.Fatalf("unexpected Rules in Role %s/%s - expected %v, received %v", namespace, name, expectedRules, role.Rules)
	}
}

func checkRoleBinding(t *testing.T, c client.Client, name, namespace string, expectedRoleRef rbacv1.RoleRef, expectedSubjects []rbacv1.Subject) *rbacv1.RoleBinding {
	t.Helper()
	rb := &rbacv1.RoleBinding{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: name, Namespace: namespace}, rb); err != nil {
		t.Fatalf("getting RoleBinding %s/%s: %v", namespace, name, err)
	}
	if !reflect.DeepEqual(expectedRoleRef, rb.RoleRef) {
		t.Fatalf("unexpected RoleRef in RoleBinding %s/%s - expected %v, received %v", namespace, name, expectedRoleRef, rb.RoleRef)
	}
	checkSubjects(t, "RoleBinding "+namespace+"/"+name, expectedSubjects, rb.Subjects)
	return rb
}

func checkClusterRoleBinding(t *testing.T, c client.Client, name string, expectedRoleRef rbacv1.RoleRef, expectedSubjects []rbacv1.Subject) *rbacv1.ClusterRoleBinding {
	t.Helper()
	crb := &rbacv1.ClusterRoleBinding{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: name}, crb); err != nil {
		t.Fatalf("getting ClusterRoleBinding %s: %v", name, err)
	}
	if !reflect.DeepEqual(expectedRoleRef, crb.RoleRef) {
		t.Fatalf("unexpected RoleRef in ClusterRoleBinding %s - expected %v, received %v", name, expectedRoleRef, crb.RoleRef)
	}
	checkSubjects(t, "ClusterRoleBinding "+name, expectedSubjects, crb.Subjects)
	return crb
}

// checkSubjects treats nil and empty subject lists as equal.
func checkSubjects(t *testing.T, what string, expected, received []rbacv1.Subject) {
	t.Helper()
	if len(expected) == 0 && len(received) == 0 {
		return
	}
	if !reflect.DeepEqual(expected, received) {
		t.Fatalf("unexpected Subjects in %s - expected %v, received %v", what, expected, received)
	}
}

func checkNotFound(t *testing.T, c client.Client, obj client.Object, name, namespace string) {
	t.Helper()
	err := c.Get(context.Background(), types.NamespacedName{Name: name, Namespace: namespace}, obj)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected %T %s/%s not to exist, got: %v", obj, namespace, name, err)
	}
}

func checkManagedAndControlledBy(t *testing.T, obj client.Object, namespace string) {
	t.Helper()
	if got := obj.GetLabels()[k8smetadata.ManagedBy]; got != project.Name() {
		t.Fatalf("expected %T %s to be managed by %s, got %q", obj, obj.GetName(), project.Name(), got)
	}
	owner := metav1.GetControllerOf(obj)
	if owner == nil || owner.Kind != "Namespace" || owner.Name != namespace {
		t.Fatalf("expected %T %s to be controlled by namespace %s, got %v", obj, obj.GetName(), namespace, owner)
	}
}

// sortSubjectsForTest mirrors the order the reconciler writes subjects in.
func sortSubjectsForTest(subjects []rbacv1.Subject) {
	slices.SortFunc(subjects, func(a, b rbacv1.Subject) int {
		return cmp.Or(cmp.Compare(a.Kind, b.Kind), cmp.Compare(a.Namespace, b.Namespace), cmp.Compare(a.Name, b.Name))
	})
}

// resourceVersions returns the resourceVersion of every RBAC object and
// ServiceAccount, keyed by kind and name.
func resourceVersions(t *testing.T, c client.Client) map[string]string {
	t.Helper()
	versions := map[string]string{}
	lists := []client.ObjectList{
		&rbacv1.RoleList{}, &rbacv1.RoleBindingList{}, &rbacv1.ClusterRoleList{},
		&rbacv1.ClusterRoleBindingList{}, &corev1.ServiceAccountList{},
	}
	for _, list := range lists {
		if err := c.List(context.Background(), list); err != nil {
			t.Fatalf("listing %T: %v", list, err)
		}
		items, err := meta.ExtractList(list)
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range items {
			obj := item.(client.Object)
			versions[fmt.Sprintf("%T/%s/%s", obj, obj.GetNamespace(), obj.GetName())] = obj.GetResourceVersion()
		}
	}
	return versions
}
