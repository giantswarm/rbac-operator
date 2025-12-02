/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	. "github.com/giantswarm/rbac-operator/api/v1alpha1"

	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	security "github.com/giantswarm/organization-operator/api/v1alpha1"
	"github.com/giantswarm/organization-operator/pkg/project"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// RoleBindingTemplateReconciler reconciles a RoleBindingTemplate object
type RoleBindingTemplateReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	errGetRBT           = "could not get RoleBindingTemplate"
	errUpdateStatus     = "unable to update status"
	errNamespacesFailed = "one or more namespaces failed"
	errSetOwnerRef      = "could not set owner reference on RoleBinding"
	errGetNamespaces    = "unable to get namespaces from scope"
	errGetRB            = "could not get existing RoleBindings"
	errCreateRB         = "could not create RoleBinding"
	errUpdateRB         = "could not update RoleBinding"
	errDeleteRB         = "could not delete RoleBinding"

	msgReconcileStarted   = "starting reconciliation"
	msgReconcileSucceeded = "reconciliation succeeded"
)

const (
	roleBindingOwnerKey = ".metadata.controller"
)

// +kubebuilder:rbac:groups=auth.giantswarm.io,resources=rolebindingtemplates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=auth.giantswarm.io,resources=rolebindingtemplates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=auth.giantswarm.io,resources=rolebindingtemplates/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups="security.giantswarm.io",resources=organizations,verbs=get;list;watch
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=rolebindings,verbs=*

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.1/pkg/reconcile
func (r *RoleBindingTemplateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	template := &RoleBindingTemplate{}
	if err := r.Get(ctx, req.NamespacedName, template); err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			return ctrl.Result{}, nil
		}
		log.Error(err, errGetRBT)
		return ctrl.Result{}, err
	}

	// Ensure status is updated at the end of reconciliation
	patch := client.MergeFrom(template.DeepCopy())
	defer func() {
		if updateErr := r.Status().Patch(ctx, template, patch); updateErr != nil {
			log.Error(updateErr, errUpdateStatus)
		}

		new := &RoleBindingTemplate{}
		if getErr := r.Get(ctx, req.NamespacedName, new); getErr != nil {
			log.Error(getErr, errGetRBT)
			return
		}
	}()

	if len(template.Status.Conditions) == 0 {
		meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
			Type:    ReadyCondition,
			Status:  metav1.ConditionUnknown,
			Reason:  ProgressingReason,
			Message: msgReconcileStarted,
		})
		if err := r.Status().Patch(ctx, template, patch); err != nil {
			log.Error(err, errUpdateStatus)
			return ctrl.Result{}, err
		}
	}

	namespaces, err := r.getNamespacesFromScope(ctx, template.Spec.Scopes)
	if err != nil {
		meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
			Type:    ReadyCondition,
			Status:  metav1.ConditionFalse,
			Reason:  FailedReason,
			Message: errGetNamespaces,
		})
		return ctrl.Result{}, err
	}

	// List existing RoleBindings created from this template
	ownedBindings := &rbacv1.RoleBindingList{}
	if err := r.List(ctx, ownedBindings, &client.MatchingFields{roleBindingOwnerKey: string(template.UID)}); err != nil {
		log.Error(err, errGetRB)
		meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
			Type:    ReadyCondition,
			Status:  metav1.ConditionFalse,
			Reason:  FailedReason,
			Message: errGetRB,
		})
		return ctrl.Result{}, err
	}

	provisionedNamespaces := []string{}
	failedNamespaces := map[string]string{}
	for _, ns := range namespaces {
		var rolebinding *rbacv1.RoleBinding
		// Check if RoleBinding already exists in the list of owned bindings
		for _, rb := range ownedBindings.Items {
			if rb.Namespace == ns {
				rolebinding = &rb
				break
			}
		}

		if rolebinding == nil {
			// Create new RoleBinding
			rolebinding = getRoleBindingFromTemplate(template, ns)
			rolebinding = cleanSubjects(rolebinding, ns)
			if len(rolebinding.Subjects) == 0 {
				log.Info("skipping RoleBinding creation due to empty subjects after cleaning", "namespace", ns, "rolebinding", rolebinding.Name)
				continue
			}

			if err := ctrl.SetControllerReference(template, rolebinding, r.Scheme); err != nil {
				log.Error(err, errSetOwnerRef, "namespace", ns, "rolebinding", rolebinding.Name)
				failedNamespaces[ns] = errSetOwnerRef
				continue
			}

			if err := r.Create(ctx, rolebinding); err != nil {
				log.Error(err, errCreateRB, "namespace", ns, "rolebinding", rolebinding.Name)
				failedNamespaces[ns] = errCreateRB
				continue
			}
		} else if !reflect.DeepEqual(rolebinding.Subjects, template.Spec.Template.Subjects) || !reflect.DeepEqual(rolebinding.RoleRef, template.Spec.Template.RoleRef) {
			// Update existing RoleBinding
			rolebinding.Subjects = template.Spec.Template.Subjects
			rolebinding.RoleRef = template.Spec.Template.RoleRef
			rolebinding = cleanSubjects(rolebinding, ns)
			if len(rolebinding.Subjects) == 0 {
				log.Info("removing RoleBinding due to empty subjects after cleaning", "namespace", ns, "rolebinding", rolebinding.Name)
				if err := r.Delete(ctx, rolebinding); err != nil {
					log.Error(err, errDeleteRB, "namespace", ns, "rolebinding", rolebinding.Name)
					failedNamespaces[ns] = errDeleteRB
					continue
				}
			}
			if err := r.Patch(ctx, rolebinding, patch); err != nil {
				log.Error(err, errUpdateRB, "namespace", ns, "rolebinding", rolebinding.Name)
				failedNamespaces[ns] = errUpdateRB
				continue
			}
		}
		provisionedNamespaces = append(provisionedNamespaces, ns)
	}

	template.Status.ProvisionedNamespaces = provisionedNamespaces
	for ns, reason := range failedNamespaces {
		template.Status.FailedNamespaces = append(template.Status.FailedNamespaces, RoleBindingTemplateNamespaceFailure{
			Namespace: ns,
			Reason:    reason,
		})
	}

	if len(failedNamespaces) > 0 {
		err := fmt.Errorf(errNamespacesFailed)
		log.Error(err, strings.Join(slices.Collect(maps.Keys(failedNamespaces)), ", "))
		meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
			Type:    ReadyCondition,
			Status:  metav1.ConditionFalse,
			Reason:  FailedReason,
			Message: errNamespacesFailed,
		})
		return ctrl.Result{}, err
	}

	log.Info("Reconciliation successful", "name", req.Name, "provisionedNamespaces", provisionedNamespaces, "failedNamespaces", failedNamespaces)
	meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
		Type:    ReadyCondition,
		Status:  metav1.ConditionTrue,
		Reason:  SucceededReason,
		Message: msgReconcileSucceeded,
	})
	return ctrl.Result{}, nil
}

func getRoleBindingFromTemplate(template *RoleBindingTemplate, namespace string) *rbacv1.RoleBinding {
	roleBinding := rbacv1.RoleBinding{
		ObjectMeta: template.Spec.Template.ObjectMeta,
		Subjects:   template.Spec.Template.Subjects,
		RoleRef:    template.Spec.Template.RoleRef,
	}
	if roleBinding.Name == "" {
		roleBinding.Name = template.Name
	}
	roleBinding.Namespace = namespace

	if roleBinding.Labels == nil {
		roleBinding.Labels = map[string]string{}
	}
	roleBinding.Labels[label.ManagedBy] = project.Name()

	if roleBinding.Annotations == nil {
		roleBinding.Annotations = map[string]string{}
	}
	if roleBinding.Annotations[annotation.Notes] == "" {
		roleBinding.Annotations[annotation.Notes] = fmt.Sprintf("Generated based on RoleBindingTemplate %s", template.Name)
	}

	return &roleBinding
}

func (r *RoleBindingTemplateReconciler) getNamespacesFromScope(ctx context.Context, scopes RoleBindingTemplateScopes) ([]string, error) {
	labelSelector, err := metav1.LabelSelectorAsSelector(&scopes.OrganizationSelector)
	if err != nil {
		return nil, err
	}

	var organizations security.OrganizationList
	if err := r.List(ctx, &organizations, &client.ListOptions{LabelSelector: labelSelector}); err != nil {
		return nil, err
	}

	namespaces, err := r.getNamespacesFromOrganizations(ctx, &organizations)
	if err != nil {
		return nil, err
	}

	return namespaces, nil
}

func (r *RoleBindingTemplateReconciler) getNamespacesFromOrganizations(ctx context.Context, organizations *security.OrganizationList) ([]string, error) {
	namespaces := []string{}

	for _, o := range organizations.Items {
		if o.Status.Namespace != "" {
			// get the org namespace
			namespaces = append(namespaces, o.Status.Namespace)
		}

		// get the cluster namespaces that belong to the org namespace
		labelSelector, err := labels.Parse(fmt.Sprintf("%s=%s,%s", label.Organization, o.Name, label.Cluster))
		if err != nil {
			return nil, err
		}

		clusterNamespaces := &corev1.NamespaceList{}
		if err := r.List(ctx, clusterNamespaces, &client.ListOptions{LabelSelector: labelSelector}); err != nil {
			return nil, err
		}

		for _, cns := range clusterNamespaces.Items {
			namespaces = append(namespaces, cns.Name)
		}
	}
	return namespaces, nil
}

func isProtectedNamespace(ns string) bool {
	return ns == "org-giantswarm"
}

const (
	fluxNamespaceName = "flux-system"
)

// TODO: could be configured from the template spec instead of hardcoded conditions
func cleanSubjects(roleBinding *rbacv1.RoleBinding, namespace string) *rbacv1.RoleBinding {
	// if the rolebinding is in a protected namespace, subjects can only be serviceAccounts in flux namespace or the same namespace
	if !isProtectedNamespace(namespace) {
		return roleBinding
	}
	var validSubjects []rbacv1.Subject
	for _, subject := range roleBinding.Subjects {
		if subject.Kind == rbacv1.ServiceAccountKind && (subject.Namespace == fluxNamespaceName || subject.Namespace == namespace) {
			validSubjects = append(validSubjects, subject)
		}
	}
	roleBinding.Subjects = validSubjects
	return roleBinding
}

// SetupWithManager sets up the controller with the Manager.
func (r *RoleBindingTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Index child objects by owner
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &rbacv1.RoleBinding{}, roleBindingOwnerKey, func(obj client.Object) []string {
		owner := metav1.GetControllerOf(obj)
		if owner == nil {
			return nil
		}
		if owner.APIVersion != GroupVersion.String() || owner.Kind != "RoleBindingTemplate" {
			return nil
		}
		return []string{string(owner.UID)}
	},
	); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&RoleBindingTemplate{}).
		Owns(&rbacv1.RoleBinding{}).
		Named("rolebindingtemplate").
		Complete(r)
}
