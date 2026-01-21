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
	"errors"
	"fmt"
	"slices"

	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	security "github.com/giantswarm/organization-operator/api/v1alpha1"
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

	"github.com/giantswarm/rbac-operator/api/v1alpha1"
	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/pkg/project"
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
	errGetNamespaces    = "unable to get namespaces from scope"
	errCreateOrUpdateRB = "could not create or update RoleBinding"
	errDeleteRB         = "could not delete RoleBinding"

	msgReconcileStarted   = "starting reconciliation"
	msgReconcileSucceeded = "reconciliation succeeded"
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

	// Get the RoleBindingTemplate resource
	template := &v1alpha1.RoleBindingTemplate{}
	if err := r.Get(ctx, req.NamespacedName, template); err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			return ctrl.Result{}, nil
		}
		log.Error(err, errGetRBT)
		return ctrl.Result{}, err
	}

	// Initialize status conditions if not already set
	if len(template.Status.Conditions) == 0 {
		meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
			Type:    v1alpha1.ReadyCondition,
			Status:  metav1.ConditionUnknown,
			Reason:  v1alpha1.ProgressingReason,
			Message: msgReconcileStarted,
		})
		template.Status.ProvisionedNamespaces = []string{}
		template.Status.FailedNamespaces = []v1alpha1.RoleBindingTemplateNamespaceFailure{}
		if err := r.Status().Update(ctx, template); err != nil {
			log.Error(err, errUpdateStatus)
			return ctrl.Result{}, err
		}
	}

	// Get namespaces from the specified scopes
	namespaces, err := r.getNamespacesFromScope(ctx, template.Spec.Scopes)
	if err != nil {
		meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
			Type:    v1alpha1.ReadyCondition,
			Status:  metav1.ConditionFalse,
			Reason:  v1alpha1.FailedReason,
			Message: errGetNamespaces,
		})
		if updateErr := r.Status().Update(ctx, template); updateErr != nil {
			log.Error(updateErr, errUpdateStatus)
		}
		return ctrl.Result{}, err
	}

	// Remove RoleBindings from namespaces that are no longer in scope
	// TODO: Use owner references to clean up RoleBindings
	for _, ns := range template.Status.ProvisionedNamespaces {
		if !slices.Contains(namespaces, ns) {
			rb := &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      getRoleBindingNameFromTemplate(template),
					Namespace: ns,
				},
			}
			if err = r.Delete(ctx, rb); client.IgnoreNotFound(err) != nil {
				log.Error(err, errDeleteRB, "namespace", ns, "rolebinding", rb.Name)
			}
		}
	}

	// Reconcile RoleBindings for each namespace in scope
	provisionedNamespaces := []string{}
	failedNamespaces := map[string]string{}
	for _, ns := range namespaces {
		rolebinding := cleanSubjects(getRoleBindingFromTemplate(template, ns), ns)
		if len(rolebinding.Subjects) == 0 {
			// If there are no subjects after cleaning, delete the RoleBinding
			log.Info("removing RoleBinding due to empty subjects after cleaning", "namespace", ns, "rolebinding", rolebinding.Name)
			if err = r.Delete(ctx, rolebinding); client.IgnoreNotFound(err) != nil {
				log.Error(err, errDeleteRB, "namespace", ns, "rolebinding", rolebinding.Name)
				failedNamespaces[ns] = errDeleteRB
			} else {
				// Treat as provisioned namespace
				provisionedNamespaces = append(provisionedNamespaces, ns)
			}
		} else {
			// Create or update the RoleBinding
			expectedRB := &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rolebinding.Name,
					Namespace: ns,
				},
			}
			_, err := ctrl.CreateOrUpdate(ctx, r.Client, expectedRB, func() error {
				expectedRB.ObjectMeta = rolebinding.ObjectMeta
				expectedRB.Subjects = rolebinding.Subjects
				expectedRB.RoleRef = rolebinding.RoleRef
				return ctrl.SetControllerReference(template, expectedRB, r.Scheme)
			})
			if err != nil {
				log.Error(err, errCreateOrUpdateRB, "namespace", ns, "rolebinding", rolebinding.Name)
				failedNamespaces[ns] = errCreateOrUpdateRB
			} else {
				provisionedNamespaces = append(provisionedNamespaces, ns)
			}
		}
	}

	template.Status.ProvisionedNamespaces = provisionedNamespaces
	for ns, reason := range failedNamespaces {
		template.Status.FailedNamespaces = append(template.Status.FailedNamespaces, v1alpha1.RoleBindingTemplateNamespaceFailure{
			Namespace: ns,
			Reason:    reason,
		})
	}

	// > 0 failed namespaces means reconciliation was not successful
	if len(template.Status.FailedNamespaces) > 0 {
		err := errors.New(errNamespacesFailed)
		log.Error(err, "failedNamespaces", template.Status.FailedNamespaces)
		meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
			Type:    v1alpha1.ReadyCondition,
			Status:  metav1.ConditionFalse,
			Reason:  v1alpha1.FailedReason,
			Message: errNamespacesFailed,
		})
		if updateErr := r.Status().Update(ctx, template); updateErr != nil {
			log.Error(updateErr, errUpdateStatus)
		}
		return ctrl.Result{}, err
	}

	log.Info(msgReconcileSucceeded, "name", req.Name, "provisionedNamespaces", provisionedNamespaces, "failedNamespaces", failedNamespaces)
	meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
		Type:    v1alpha1.ReadyCondition,
		Status:  metav1.ConditionTrue,
		Reason:  v1alpha1.SucceededReason,
		Message: msgReconcileSucceeded,
	})
	if updateErr := r.Status().Update(ctx, template); updateErr != nil {
		log.Error(updateErr, errUpdateStatus)
	}
	return ctrl.Result{}, nil
}

func (r *RoleBindingTemplateReconciler) getNamespacesFromScope(ctx context.Context, scopes v1alpha1.RoleBindingTemplateScopes) ([]string, error) {
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

func getRoleBindingNameFromTemplate(template *v1alpha1.RoleBindingTemplate) string {
	roleBindingName := template.Spec.Template.Metadata.Name
	if roleBindingName == "" {
		roleBindingName = template.Name
	}
	return roleBindingName
}

func getRoleBindingFromTemplate(template *v1alpha1.RoleBindingTemplate, namespace string) *rbacv1.RoleBinding {
	roleBinding := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      template.Spec.Template.Metadata.Labels,
			Annotations: template.Spec.Template.Metadata.Annotations,
			Finalizers:  template.Spec.Template.Metadata.Finalizers,
		},
		RoleRef: template.Spec.Template.RoleRef,
	}
	roleBinding.Name = getRoleBindingNameFromTemplate(template)
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

	if roleBinding.RoleRef.APIGroup == "" {
		roleBinding.RoleRef.APIGroup = "rbac.authorization.k8s.io"
	}

	subjects := []rbacv1.Subject{}
	{
		for _, subject := range template.Spec.Template.Subjects {
			if subject.Kind == rbacv1.ServiceAccountKind && subject.Namespace == "" {
				subject.Namespace = namespace
			}
			subjects = append(subjects, subject)
		}
	}
	roleBinding.Subjects = subjects

	return &roleBinding
}

func cleanSubjects(roleBinding *rbacv1.RoleBinding, namespace string) *rbacv1.RoleBinding {
	// if the rolebinding is in a protected namespace, subjects can only be serviceAccounts in flux namespace or the same namespace
	if !pkgkey.IsProtectedNamespace(namespace) {
		return roleBinding
	}
	var validSubjects []rbacv1.Subject
	for _, subject := range roleBinding.Subjects {
		if subject.Kind == rbacv1.ServiceAccountKind && (subject.Namespace == pkgkey.FluxNamespaceName || subject.Namespace == namespace) {
			validSubjects = append(validSubjects, subject)
		}
	}
	roleBinding.Subjects = validSubjects
	return roleBinding
}

// SetupWithManager sets up the controller with the Manager.
func (r *RoleBindingTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.RoleBindingTemplate{}).
		Owns(&rbacv1.RoleBinding{}).
		Complete(r)
}
