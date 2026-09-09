package controller

import (
	"context"
	"fmt"

	k8smetadata "github.com/giantswarm/k8smetadata/pkg/label"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	pkglabel "github.com/giantswarm/rbac-operator/pkg/label"
	"github.com/giantswarm/rbac-operator/pkg/project"
)

// OrgNamespaceReconciler manages RBAC resources in namespaces that belong to an organization.
type OrgNamespaceReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	// WriteAllCustomerSubjects are the RBAC subjects that receive write-all access in org namespaces.
	WriteAllCustomerSubjects []rbacv1.Subject
	// ReadAllCustomerSubjects are the RBAC subjects that receive read-all access in org namespaces.
	ReadAllCustomerSubjects []rbacv1.Subject
}

const (
	// legacyRBACControllerFinalizer is the finalizer the old operatorkit-based
	// "rbac" controller (removed in favor of OrgNamespaceReconciler) added to
	// these namespaces.
	legacyRBACControllerFinalizer = "operatorkit.giantswarm.io/rbac-operator-rbac-controller"

	// legacyCrossplaneNamespaceControllerFinalizer is the finalizer the
	// operatorkit-based "crossplane-namespace" controller adds to org
	// namespaces. That controller is being migrated away separately, so we strip
	// its finalizer here.
	legacyCrossplaneNamespaceControllerFinalizer = "operatorkit.giantswarm.io/rbac-operator-crossplane-namespace-controller"
)

func (r *OrgNamespaceReconciler) removeLegacyFinalizers(ctx context.Context, ns *corev1.Namespace) error {
	removedLegacyRBAC := controllerutil.RemoveFinalizer(ns, legacyRBACControllerFinalizer)
	removedLegacyCrossplane := controllerutil.RemoveFinalizer(ns, legacyCrossplaneNamespaceControllerFinalizer)
	if removedLegacyRBAC || removedLegacyCrossplane {
		if err := r.Update(ctx, ns); err != nil {
			return fmt.Errorf("updating namespace %s: %w", ns.Name, err)
		}
	}

	return nil
}

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;update
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=roles,verbs=get;list;watch;create;update;patch;bind;escalate
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterroles,verbs=get;list;watch;create;update;patch;bind;escalate
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterrolebindings,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=rolebindings,verbs=get;list;watch;create;update;patch;delete

func (r *OrgNamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ns := &corev1.Namespace{}
	if err := r.Get(ctx, req.NamespacedName, ns); err != nil {
		if apierrors.IsNotFound(err) {
			// Rebuild shared bindings to drop the stale automation ServiceAccount.
			if err := r.ensureAutomationSharedRBAC(ctx); err != nil {
				return ctrl.Result{}, fmt.Errorf("reconciling shared automation RBAC: %w", err)
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("getting namespace: %w", err)
	}

	if err := r.removeLegacyFinalizers(ctx, ns); err != nil {
		return ctrl.Result{}, fmt.Errorf("removing legacy finalizers: %w", err)
	}
	if !ns.DeletionTimestamp.IsZero() {
		// Drop this namespace's automation ServiceAccount from the shared bindings
		if err := r.ensureAutomationSharedRBAC(ctx); err != nil {
			return ctrl.Result{}, fmt.Errorf("reconciling shared automation RBAC: %w", err)
		}
		return ctrl.Result{}, nil
	}

	// Must run before the RoleBindings are listed below, so the customer-admin
	// subjects count as members and writers in the same pass.
	if err := r.ensureCustomerAdminAccess(ctx, ns); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconciling customer-admin access: %w", err)
	}

	// TODO: Preserved from old controller, might not be required
	if !pkgkey.IsOrgNamespace(ns.Name) {
		return ctrl.Result{}, nil
	}

	if err := r.ensureAutomationControlledRBAC(ctx, ns); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconciling automation RBAC: %w", err)
	}
	if err := r.ensureAutomationSharedRBAC(ctx); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconciling shared automation RBAC: %w", err)
	}

	rbs := &rbacv1.RoleBindingList{}
	if err := r.List(ctx, rbs, client.InNamespace(ns.Name)); err != nil {
		return ctrl.Result{}, fmt.Errorf("listing RoleBindings: %w", err)
	}
	// Any subject bound to anything in the org namespace is considered a member.
	members := uniqueSubjectsFromRBs(rbs.Items)
	readers := subjectsFromRBsWithClusterRoleRef(rbs.Items, pkgkey.DefaultReadAllPermissionsName)
	writers := subjectsFromRBsWithClusterRoleRef(rbs.Items, pkgkey.ClusterAdminClusterRoleName)

	if err := r.ensureMemberReadAccess(ctx, ns, members); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconciling member read access: %w", err)
	}
	if err := r.ensureClusterNamespaceAccess(ctx, ns, readers, writers); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconciling cluster namespace access: %w", err)
	}

	return ctrl.Result{}, nil
}

// ensureAutomationControlledRBAC creates the automation ServiceAccount in the org
// namespace and binds it to the "write-silences", "kamaji-datastore-manager" and
// "cluster-app-chart-lookups" ClusterRoles. All of these are controlled by the
// org namespace.
func (r *OrgNamespaceReconciler) ensureAutomationControlledRBAC(ctx context.Context, ns *corev1.Namespace) error {
	sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
		Name: pkgkey.AutomationServiceAccountName, Namespace: ns.Name,
	}}
	_, err := ctrl.CreateOrUpdate(ctx, r.Client, sa, func() error {
		metav1.SetMetaDataLabel(&sa.ObjectMeta, k8smetadata.ManagedBy, project.Name())
		return controllerutil.SetControllerReference(ns, sa, r.Scheme)
	})
	if err != nil {
		return fmt.Errorf("creating or updating ServiceAccount %s/%s: %w", sa.Namespace, sa.Name, err)
	}

	// Cluster-wide permissions needed by the automation SA.
	automationCRBs := []struct {
		name        string
		clusterRole string
	}{
		// ClusterRole is provisioned by the default-namespace controller.
		{pkgkey.WriteSilencesAutomationSAinNSRoleBindingName(ns.Name), pkgkey.WriteSilencesPermissionsName},
		// ClusterRole is provisioned by the global Kamaji app.
		{pkgkey.KamajiDatastoreManagerAutomationSAinNSRoleBindingName(ns.Name), pkgkey.KamajiDatastoreManagerPermissionsName},
		// ClusterRole is provisioned by the default-namespace controller.
		// The purpose of this role is to grant read-only access to the cluster-app-chart-lookups
		// resources, which are created by the cluster-* (release-*) charts.
		{pkgkey.ClusterAppChartLookupsAutomationSAinNSRoleBindingName(ns.Name), pkgkey.ClusterAppChartLookupsPermissionsName},
	}
	for _, binding := range automationCRBs {
		crb := &rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: binding.name}}
		_, err = ctrl.CreateOrUpdate(ctx, r.Client, crb, func() error {
			metav1.SetMetaDataLabel(&crb.ObjectMeta, k8smetadata.ManagedBy, project.Name())
			crb.Subjects = []rbacv1.Subject{{
				Kind: "ServiceAccount", Name: pkgkey.AutomationServiceAccountName, Namespace: ns.Name,
			}}
			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     binding.clusterRole,
			}
			return controllerutil.SetControllerReference(ns, crb, r.Scheme)
		})
		if err != nil {
			return fmt.Errorf("creating or updating ClusterRoleBinding %s: %w", crb.Name, err)
		}
	}
	return nil
}

// TODO: Move to a dedicated reconciler with a fixed request key, triggered by
// org namespace create/delete and policy-exceptions namespace creation. Merge
// the per-org automation ClusterRoleBindings into one ClusterRoleBinding per
// ClusterRole there, cleaning up the old per-org bindings, and watch the shared
// Roles/RoleBindings there for drift.
// ensureAutomationSharedRBAC binds the automation ServiceAccounts of all org
// namespaces to the "patch-charts" and "write-policy-exceptions" Roles. These
// resources are shared across org namespaces and must be recomputed whenever an
// org namespace is created or deleted.
func (r *OrgNamespaceReconciler) ensureAutomationSharedRBAC(ctx context.Context) error {
	automationSASubjects, err := r.allOrgAutomationSubjects(ctx)
	if err != nil {
		return err
	}

	// Create the shared `patch-charts` Role and RoleBinding in the `giantswarm`
	// namespace, bound to every org's automation ServiceAccount. This is required
	// for the App to HelmRelease migration and is expected to be removed once
	// that migration is complete.
	// TODO: Role is not dependent on org namespace, could be managed elsewhere
	patchChartsRole := &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{
		Name: pkgkey.PatchChartsPermissionsName, Namespace: pkgkey.GiantSwarmNamespaceName,
	}}
	_, err = ctrl.CreateOrUpdate(ctx, r.Client, patchChartsRole, func() error {
		metav1.SetMetaDataLabel(&patchChartsRole.ObjectMeta, k8smetadata.ManagedBy, project.Name())
		patchChartsRole.Rules = []rbacv1.PolicyRule{{
			APIGroups: []string{"application.giantswarm.io"},
			Resources: []string{"charts"},
			Verbs:     []string{"list", "get", "patch"},
		}}
		return nil
	})
	if err != nil {
		return fmt.Errorf("creating or updating Role %s/%s: %w", patchChartsRole.Namespace, patchChartsRole.Name, err)
	}

	patchChartsRB := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{
		Name: pkgkey.PatchChartsPermissionsName, Namespace: pkgkey.GiantSwarmNamespaceName,
	}}
	_, err = ctrl.CreateOrUpdate(ctx, r.Client, patchChartsRB, func() error {
		metav1.SetMetaDataLabel(&patchChartsRB.ObjectMeta, k8smetadata.ManagedBy, project.Name())
		patchChartsRB.Subjects = automationSASubjects
		patchChartsRB.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     pkgkey.PatchChartsPermissionsName,
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("creating or updating RoleBinding %s/%s: %w", patchChartsRB.Namespace, patchChartsRB.Name, err)
	}

	// Create the shared `write-policy-exceptions` Role and RoleBinding in every
	// namespace in which PolicyExceptions are managed, bound to every org's
	// automation ServiceAccount.
	for _, namespace := range pkgkey.WritePolicyExceptionsNamespaces {
		// The `policy-exceptions` namespace is created by the policy apps and is
		// absent on clusters which don't run them, so skip namespaces that don't
		// exist instead of failing the whole reconciliation.
		ns := &corev1.Namespace{}
		if err := r.Get(ctx, types.NamespacedName{Name: namespace}, ns); err != nil {
			// TODO: Currently it will not reconcile if the namespace is created later until some other event triggers a reconcile
			if apierrors.IsNotFound(err) {
				continue
			}
			return fmt.Errorf("getting namespace %s: %w", namespace, err)
		}

		// TODO: Role is not dependent on org namespace, could be managed elsewhere
		role := &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{
			Name: pkgkey.WritePolicyExceptionsPermissionsName, Namespace: namespace,
		}}
		_, err = ctrl.CreateOrUpdate(ctx, r.Client, role, func() error {
			metav1.SetMetaDataLabel(&role.ObjectMeta, k8smetadata.ManagedBy, project.Name())
			role.Rules = []rbacv1.PolicyRule{{
				APIGroups: []string{"kyverno.io"},
				Resources: []string{"policyexceptions"},
				Verbs:     []string{"*"},
			}}
			return nil
		})
		if err != nil {
			return fmt.Errorf("creating or updating Role %s/%s: %w", role.Namespace, role.Name, err)
		}

		rb := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{
			Name: pkgkey.WritePolicyExceptionsPermissionsName, Namespace: namespace,
		}}
		_, err = ctrl.CreateOrUpdate(ctx, r.Client, rb, func() error {
			metav1.SetMetaDataLabel(&rb.ObjectMeta, k8smetadata.ManagedBy, project.Name())
			rb.Subjects = automationSASubjects
			rb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "Role",
				Name:     pkgkey.WritePolicyExceptionsPermissionsName,
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("creating or updating RoleBinding %s/%s: %w", rb.Namespace, rb.Name, err)
		}
	}

	return nil
}

// ensureCustomerAdminAccess binds the write-all customer groups to cluster-admin
// in the namespace. The binding is removed in protected namespaces and when no
// write-all customer groups are configured.
func (r *OrgNamespaceReconciler) ensureCustomerAdminAccess(ctx context.Context, ns *corev1.Namespace) error {
	rb := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{
		Name: pkgkey.WriteAllCustomerGroupRoleBindingName(), Namespace: ns.Name,
	}}
	if len(r.WriteAllCustomerSubjects) == 0 || pkgkey.IsProtectedNamespace(ns.Name) {
		if err := r.Delete(ctx, rb); client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("deleting RoleBinding %s/%s: %w", rb.Namespace, rb.Name, err)
		}
		return nil
	}
	_, err := ctrl.CreateOrUpdate(ctx, r.Client, rb, func() error {
		metav1.SetMetaDataLabel(&rb.ObjectMeta, k8smetadata.ManagedBy, project.Name())
		rb.Subjects = r.WriteAllCustomerSubjects
		// NOTE: cluster-admin is the built-in Kubernetes ClusterRole
		rb.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     pkgkey.ClusterAdminClusterRoleName,
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("creating or updating RoleBinding %s/%s: %w", rb.Namespace, rb.Name, err)
	}
	return nil
}

// ensureMemberReadAccess gives members of the org namespace read access to
//   - the organization's own Organization CR,
//   - releases (cluster-scoped),
//   - app catalogs and app catalog entries in the default namespace.
func (r *OrgNamespaceReconciler) ensureMemberReadAccess(ctx context.Context, ns *corev1.Namespace, members []rbacv1.Subject) error {
	org := pkgkey.OrganizationName(ns.Name)

	cr := &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{
		Name: pkgkey.OrganizationReadClusterRoleName(ns.Name),
	}}
	_, err := ctrl.CreateOrUpdate(ctx, r.Client, cr, func() error {
		metav1.SetMetaDataLabel(&cr.ObjectMeta, k8smetadata.ManagedBy, project.Name())
		cr.Rules = []rbacv1.PolicyRule{{
			APIGroups:     []string{"security.giantswarm.io"},
			Resources:     []string{"organizations"},
			ResourceNames: []string{org},
			Verbs:         []string{"get"},
		}}
		return controllerutil.SetControllerReference(ns, cr, r.Scheme)
	})
	if err != nil {
		return fmt.Errorf("creating or updating ClusterRole %s: %w", cr.Name, err)
	}

	organizationReadCRB := &rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{
		Name: pkgkey.OrganizationReadOrganizationClusterRoleBindingName(org),
	}}
	_, err = ctrl.CreateOrUpdate(ctx, r.Client, organizationReadCRB, func() error {
		metav1.SetMetaDataLabel(&organizationReadCRB.ObjectMeta, k8smetadata.ManagedBy, project.Name())
		organizationReadCRB.Subjects = members
		organizationReadCRB.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     cr.Name,
		}
		return controllerutil.SetControllerReference(ns, organizationReadCRB, r.Scheme)
	})
	if err != nil {
		return fmt.Errorf("creating or updating ClusterRoleBinding %s: %w", organizationReadCRB.Name, err)
	}

	releasesCRB := &rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{
		Name: pkgkey.OrganizationReadReleasesClusterRoleBindingName(org),
	}}
	_, err = ctrl.CreateOrUpdate(ctx, r.Client, releasesCRB, func() error {
		metav1.SetMetaDataLabel(&releasesCRB.ObjectMeta, k8smetadata.ManagedBy, project.Name())
		releasesCRB.Subjects = members
		// NOTE: ClusterRole is provisioned by default-namespace controller
		releasesCRB.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     pkgkey.ReadReleasesRole,
		}
		return controllerutil.SetControllerReference(ns, releasesCRB, r.Scheme)
	})
	if err != nil {
		return fmt.Errorf("creating or updating ClusterRoleBinding %s: %w", releasesCRB.Name, err)
	}

	catalogsRB := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{
		Name: pkgkey.OrganizationReadDefaultCatalogsRoleBindingName(org), Namespace: pkgkey.DefaultNamespaceName,
	}}
	_, err = ctrl.CreateOrUpdate(ctx, r.Client, catalogsRB, func() error {
		metav1.SetMetaDataLabel(&catalogsRB.ObjectMeta, k8smetadata.ManagedBy, project.Name())
		catalogsRB.Subjects = members
		// NOTE: Role is provisioned by default-namespace controller
		catalogsRB.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     pkgkey.ReadDefaultCatalogsRole,
		}
		return controllerutil.SetControllerReference(ns, catalogsRB, r.Scheme)
	})
	if err != nil {
		return fmt.Errorf("creating or updating RoleBinding %s/%s: %w", catalogsRB.Namespace, catalogsRB.Name, err)
	}

	return nil
}

// ensureClusterNamespaceAccess gives subjects with read-all access to the org
// namespace read access, and subjects with cluster-admin access write access
// to resources in the organization's cluster namespaces.
func (r *OrgNamespaceReconciler) ensureClusterNamespaceAccess(ctx context.Context, ns *corev1.Namespace, readers, writers []rbacv1.Subject) error {
	org := pkgkey.OrganizationName(ns.Name)

	readClusterNamespaceRB := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{
		Name: pkgkey.OrganizationReadClusterNamespaceRoleBindingName(org), Namespace: ns.Name,
	}}
	_, err := ctrl.CreateOrUpdate(ctx, r.Client, readClusterNamespaceRB, func() error {
		metav1.SetMetaDataLabel(&readClusterNamespaceRB.ObjectMeta, k8smetadata.ManagedBy, project.Name())
		readClusterNamespaceRB.Subjects = readers
		// NOTE: ClusterRole is provisioned by default-namespace controller
		readClusterNamespaceRB.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     pkgkey.ReadClusterNamespaceAppsRole,
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("creating or updating RoleBinding %s/%s: %w", readClusterNamespaceRB.Namespace, readClusterNamespaceRB.Name, err)
	}

	writeClusterNamespaceRB := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{
		Name: pkgkey.OrganizationWriteClusterNamespaceRoleBindingName(org), Namespace: ns.Name,
	}}
	_, err = ctrl.CreateOrUpdate(ctx, r.Client, writeClusterNamespaceRB, func() error {
		metav1.SetMetaDataLabel(&writeClusterNamespaceRB.ObjectMeta, k8smetadata.ManagedBy, project.Name())
		writeClusterNamespaceRB.Subjects = writers
		// NOTE: ClusterRole is provisioned by default-namespace controller
		writeClusterNamespaceRB.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     pkgkey.WriteClusterNamespaceAppsRole,
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("creating or updating RoleBinding %s/%s: %w", writeClusterNamespaceRB.Namespace, writeClusterNamespaceRB.Name, err)
	}

	return nil
}

// allOrgAutomationSubjects returns a Subject for the automation SA in every org namespace.
func (r *OrgNamespaceReconciler) allOrgAutomationSubjects(ctx context.Context) ([]rbacv1.Subject, error) {
	nsList := &corev1.NamespaceList{}
	if err := r.List(ctx, nsList, client.HasLabels{k8smetadata.Organization}); err != nil {
		return nil, fmt.Errorf("listing organization namespaces: %w", err)
	}
	var subjects []rbacv1.Subject
	for _, ns := range nsList.Items {
		if !ns.DeletionTimestamp.IsZero() {
			continue
		}
		subjects = append(subjects, rbacv1.Subject{
			Kind: "ServiceAccount", Name: pkgkey.AutomationServiceAccountName, Namespace: ns.Name,
		})
	}
	sortSubjects(subjects)
	return subjects, nil
}

func HasOrganizationOrCustomerLabel(ns *corev1.Namespace) bool {
	_, org := ns.Labels[k8smetadata.Organization]
	_, cust := ns.Labels[pkglabel.LegacyCustomer]
	return org || cust
}

// SetupWithManager registers the reconciler. Org- or customer-labeled namespaces are the
// primary trigger. RoleBinding changes in those namespaces also enqueue the namespace, so
// changes to its bindings are propagated immediately to derived resources.
func (r *OrgNamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// TODO: Identify org namespaces via Organization CRs (status.namespace)
	nsPredicate := predicate.NewPredicateFuncs(func(obj client.Object) bool {
		ns, ok := obj.(*corev1.Namespace)
		return ok && HasOrganizationOrCustomerLabel(ns)
	})

	// Only RoleBindings inside org or customer namespaces are inputs to Reconcile.
	rbPredicate := predicate.NewPredicateFuncs(func(obj client.Object) bool {
		ns := &corev1.Namespace{}
		if err := r.Get(context.Background(), types.NamespacedName{Name: obj.GetNamespace()}, ns); err != nil {
			return false
		}
		return HasOrganizationOrCustomerLabel(ns)
	})

	// Enqueue the namespace owning a RoleBinding when it changes.
	rbMapper := handler.MapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
		return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: obj.GetNamespace()}}}
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}, builder.WithPredicates(nsPredicate)).
		Watches(&rbacv1.RoleBinding{}, handler.EnqueueRequestsFromMapFunc(rbMapper), builder.WithPredicates(rbPredicate)).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Owns(&rbacv1.RoleBinding{}).
		Owns(&corev1.ServiceAccount{}).
		Complete(r)
}
