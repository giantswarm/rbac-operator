package controller

import (
	"context"
	"fmt"
	"slices"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/giantswarm/rbac-operator/api/v1alpha1"
)

// TODO: Remove this file once every cluster runs a release without the
// operatorkit controllers.
//
// operatorkit adds its finalizer to every object matched by a controller's
// selector, and some of the old controllers had no selector. The finalizers
// are therefore on objects the kubebuilder controllers never reconcile, and
// would block their deletion forever. Ownership is now handled by
// ownerReferences, so the finalizers can simply be dropped.
var (
	legacyNamespaceFinalizers = []string{
		// No selector: every namespace.
		"operatorkit.giantswarm.io/rbac-operator-rbac-controller",
		// Namespaces with the organization label.
		"operatorkit.giantswarm.io/rbac-operator-crossplane-namespace-controller",
		// Namespaces with the organization and cluster labels.
		"operatorkit.giantswarm.io/rbac-operator-cluster-namespace-controller",
		// The default namespace.
		"operatorkit.giantswarm.io/rbac-operator-default-namespace-controller",
	}
	legacyClusterRoleFinalizers = []string{
		// No selector: every ClusterRole.
		"operatorkit.giantswarm.io/rbac-operator-crossplane-controller",
	}
	legacyRoleBindingTemplateFinalizers = []string{
		// No selector: every RoleBindingTemplate.
		"operatorkit.giantswarm.io/rbac-operator-rolebindingtemplate-controller",
	}
)

// LegacyFinalizerReconciler removes the finalizers that the operatorkit
// controllers of this operator added to objects of one kind.
type LegacyFinalizerReconciler struct {
	client.Client

	// Name is the controller name; it must be unique within the manager.
	Name string
	// NewObject returns an empty object of the kind to clean up.
	NewObject func() client.Object
	// Finalizers are the legacy finalizers to remove.
	Finalizers []string
}

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;update
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterroles,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=auth.giantswarm.io,resources=rolebindingtemplates,verbs=get;list;watch;update

func (r *LegacyFinalizerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	obj := r.NewObject()
	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Objects being deleted are handled too: that is where a leftover
	// finalizer does the damage.
	var removed []string
	for _, f := range r.Finalizers {
		if controllerutil.RemoveFinalizer(obj, f) {
			removed = append(removed, f)
		}
	}
	if len(removed) == 0 {
		return ctrl.Result{}, nil
	}

	if err := r.Update(ctx, obj); err != nil {
		return ctrl.Result{}, fmt.Errorf("removing legacy finalizers from %s: %w", req.NamespacedName, err)
	}
	logf.FromContext(ctx).Info("removed legacy operatorkit finalizers", "finalizers", removed)

	return ctrl.Result{}, nil
}

// SetupWithManager registers the reconciler. Only objects that carry one of
// the legacy finalizers are reconciled.
func (r *LegacyFinalizerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	hasLegacyFinalizer := predicate.NewPredicateFuncs(func(obj client.Object) bool {
		return slices.ContainsFunc(r.Finalizers, func(f string) bool {
			return controllerutil.ContainsFinalizer(obj, f)
		})
	})

	return ctrl.NewControllerManagedBy(mgr).
		Named(r.Name).
		For(r.NewObject(), builder.WithPredicates(hasLegacyFinalizer)).
		Complete(r)
}

// LegacyFinalizerReconcilers returns one LegacyFinalizerReconciler per kind
// that the operatorkit controllers added finalizers to.
func LegacyFinalizerReconcilers(c client.Client) []*LegacyFinalizerReconciler {
	return []*LegacyFinalizerReconciler{
		{
			Client:     c,
			Name:       "legacy-finalizer-namespace",
			NewObject:  func() client.Object { return &corev1.Namespace{} },
			Finalizers: legacyNamespaceFinalizers,
		},
		{
			Client:     c,
			Name:       "legacy-finalizer-clusterrole",
			NewObject:  func() client.Object { return &rbacv1.ClusterRole{} },
			Finalizers: legacyClusterRoleFinalizers,
		},
		{
			Client:     c,
			Name:       "legacy-finalizer-rolebindingtemplate",
			NewObject:  func() client.Object { return &v1alpha1.RoleBindingTemplate{} },
			Finalizers: legacyRoleBindingTemplateFinalizers,
		},
	}
}
