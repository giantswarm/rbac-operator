package controller

import (
	"context"
	"fmt"

	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/pkg/project"
)

// CrossplaneReconciler manages the ClusterRoleBinding that grants customer admin groups and
// the automation ServiceAccount from every org namespace access to the crossplane-edit ClusterRole.
// Merges the two old operatorkit crossplane sub-controllers into one authoritative reconcile.
type CrossplaneReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	CustomerAdminGroups                 []rbacv1.Subject
	CrossplaneBindTriggeringClusterRole string
}

// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterroles,verbs=get;list;watch
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterrolebindings,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch

func (r *CrossplaneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	cr := &rbacv1.ClusterRole{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	log.Info("reconciling crossplane ClusterRoleBinding", "clusterRole", cr.Name)

	subjects, err := r.buildSubjects(ctx)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("build subjects: %w", err)
	}

	crbName := pkgkey.CrossplaneBindClusterRoleBindingName(r.CrossplaneBindTriggeringClusterRole)
	crb := &rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: crbName}}
	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, crb, func() error {
		if crb.Labels == nil {
			crb.Labels = map[string]string{}
		}
		crb.Labels[label.ManagedBy] = project.Name()
		if crb.Annotations == nil {
			crb.Annotations = map[string]string{}
		}
		crb.Annotations[annotation.Notes] = "Grants customer's cluster-admin permissions to use crossplane rbac-manager managed crossplane:edit ClusterRole"
		crb.Subjects = subjects
		crb.RoleRef = rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     r.CrossplaneBindTriggeringClusterRole,
		}
		return nil
	}); err != nil {
		return ctrl.Result{}, fmt.Errorf("create/update ClusterRoleBinding %s: %w", crbName, err)
	}

	return ctrl.Result{}, nil
}

// buildSubjects collects: automation SA from default namespace, automation SAs from all org
// namespaces, and all customer admin groups.
func (r *CrossplaneReconciler) buildSubjects(ctx context.Context) ([]rbacv1.Subject, error) {
	subjects := []rbacv1.Subject{{
		Kind:      "ServiceAccount",
		Name:      pkgkey.AutomationServiceAccountName,
		Namespace: pkgkey.DefaultNamespaceName,
	}}

	nsList := &corev1.NamespaceList{}
	if err := r.List(ctx, nsList, client.HasLabels{label.Organization}); err != nil {
		return nil, err
	}
	for _, ns := range nsList.Items {
		subjects = append(subjects, rbacv1.Subject{
			Kind:      "ServiceAccount",
			Name:      pkgkey.AutomationServiceAccountName,
			Namespace: ns.Name,
		})
	}

	subjects = append(subjects, r.CustomerAdminGroups...)

	return subjects, nil
}

// SetupWithManager registers the reconciler. Watches the triggering ClusterRole as primary.
// When an org namespace is created or deleted, the ClusterRole is enqueued so the
// ClusterRoleBinding is rebuilt with the current set of org automation SAs. The org-namespace
// filter is applied on the watch itself via nsPredicate, not inside the mapper.
func (r *CrossplaneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	crPredicate := predicate.NewPredicateFuncs(func(obj client.Object) bool {
		return obj.GetName() == r.CrossplaneBindTriggeringClusterRole
	})

	nsPredicate, err := predicate.LabelSelectorPredicate(metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: label.Organization, Operator: metav1.LabelSelectorOpExists},
		},
	})
	if err != nil {
		return fmt.Errorf("build namespace label selector predicate: %w", err)
	}

	nsMapper := handler.MapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
		return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: r.CrossplaneBindTriggeringClusterRole}}}
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&rbacv1.ClusterRole{}, builder.WithPredicates(crPredicate)).
		Watches(&corev1.Namespace{}, handler.EnqueueRequestsFromMapFunc(nsMapper), builder.WithPredicates(nsPredicate)).
		Complete(r)
}
