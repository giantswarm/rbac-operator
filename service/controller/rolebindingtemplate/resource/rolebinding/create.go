package rolebinding

import (
	"context"
	"fmt"
	"strings"

	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/rbac-operator/api/v1alpha1"
	"github.com/giantswarm/rbac-operator/pkg/project"
	"github.com/giantswarm/rbac-operator/pkg/rbac"
	"github.com/giantswarm/rbac-operator/service/controller/rolebindingtemplate/key"
)

const (
	// MaxDetailedErrors is the maximum number of detailed error messages to include
	// in the error response to avoid excessively long error messages
	MaxDetailedErrors = 5
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	template, err := key.ToRoleBindingTemplate(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	namespaces, err := r.getNamespacesFromScope(ctx, template.Spec.Scopes)
	if err != nil {
		return microerror.Mask(err)
	}

	status := []string{}
	var roleBindingErrors []string
	successCount := 0
	totalOperations := 0

	// Create or update role bindings in target namespaces
	for _, ns := range namespaces {
		totalOperations++
		roleBinding, err := getRoleBindingFromTemplate(template, ns)
		if err != nil {
			return microerror.Mask(err)
		}

		if err = rbac.CreateOrUpdateRoleBinding(r, ctx, ns, roleBinding); err != nil {
			r.logger.Errorf(ctx, err, "Failed to apply roleBinding %s to namespace %s", roleBinding.Name, ns)
			roleBindingErrors = append(roleBindingErrors, fmt.Sprintf("namespace %s: %v", ns, err))
			continue
		}

		successCount++
		r.logger.Debugf(ctx, "Successfully applied roleBinding %s to namespace %s", roleBinding.Name, ns)
		status = append(status, ns)
	}

	// Cleanup role bindings from namespaces no longer in scope
	var deletionErrors []string
	for _, ns := range template.Status.Namespaces {
		if !contains(status, ns) {
			totalOperations++
			roleBindingName := getRoleBindingNameFromTemplate(template)
			if err = rbac.DeleteRoleBinding(r, ctx, ns, roleBindingName); err != nil {
				r.logger.Errorf(ctx, err, "Failed to delete roleBinding %s from namespace %s", roleBindingName, ns)
				deletionErrors = append(deletionErrors, fmt.Sprintf("namespace %s: %v", ns, err))
				continue
			}
			successCount++
			r.logger.Debugf(ctx, "Successfully deleted roleBinding %s from namespace %s", roleBindingName, ns)
		}
	}

	// Update template status with current namespace list
	template.Status.Namespaces = status
	if err := r.k8sClient.CtrlClient().Status().Update(ctx, &template); err != nil {
		r.logger.Errorf(ctx, err, "Failed to update template status")
		return microerror.Maskf(namespaceUpdateFailedError, "failed to update template status: %v", err)
	}

	// Return combined errors if any operations failed
	allErrors := append(roleBindingErrors, deletionErrors...)
	if len(allErrors) > 0 {
		// Log summary for operations visibility
		r.logger.Debugf(ctx, "RoleBindingTemplate %s operation summary: %d/%d successful (%d creation errors, %d deletion errors)",
			template.Name, successCount, totalOperations, len(roleBindingErrors), len(deletionErrors))

		// Limit the number of detailed error messages to avoid excessive verbosity
		detailedErrors := allErrors
		if len(allErrors) > MaxDetailedErrors {
			detailedErrors = allErrors[:MaxDetailedErrors]
			detailedErrors = append(detailedErrors, fmt.Sprintf("and %d more errors", len(allErrors)-MaxDetailedErrors))
		}

		// Determine which error type to return based on what failed
		if len(roleBindingErrors) > 0 && len(deletionErrors) > 0 {
			return microerror.Maskf(roleBindingCreationFailedError,
				"completed %d/%d operations successfully; failed to apply/delete role bindings: %s",
				successCount, totalOperations, strings.Join(detailedErrors, "; "))
		} else if len(roleBindingErrors) > 0 {
			return microerror.Maskf(roleBindingCreationFailedError,
				"completed %d/%d operations successfully; failed to apply role bindings: %s",
				successCount, totalOperations, strings.Join(detailedErrors, "; "))
		} else {
			return microerror.Maskf(roleBindingDeletionFailedError,
				"completed %d/%d operations successfully; failed to delete role bindings: %s",
				successCount, totalOperations, strings.Join(detailedErrors, "; "))
		}
	}

	r.logger.Debugf(ctx, "Successfully completed all %d role binding operations for template %s", totalOperations, template.Name)
	return nil
}

func getRoleBindingFromTemplate(template v1alpha1.RoleBindingTemplate, namespace string) (*rbacv1.RoleBinding, error) {
	objectMeta := template.Spec.Template.ObjectMeta
	{
		// ensure namespaced name
		objectMeta.Name = getRoleBindingNameFromTemplate(template)
		objectMeta.Namespace = namespace
		// add labels and annotations
		labels := objectMeta.GetLabels()
		if labels == nil {
			labels = map[string]string{}
		}
		labels[label.ManagedBy] = project.Name()
		objectMeta.SetLabels(labels)
		annotations := objectMeta.GetAnnotations()
		if annotations == nil {
			annotations = map[string]string{}
		}
		if annotations[annotation.Notes] == "" {
			annotations[annotation.Notes] = fmt.Sprintf("Generated based on RoleBindingTemplate %s", template.Name)
		}
		objectMeta.SetAnnotations(annotations)
	}

	// ensure role reference
	roleRef := template.Spec.Template.RoleRef
	{
		if incompleteRoleRef(roleRef) {
			return nil, microerror.Maskf(invalidConfigError, "RoleBindingTemplate %s has incomplete roleRef %v", template.Name, roleRef)
		}
		if roleRef.APIGroup == "" {
			roleRef.APIGroup = "rbac.authorization.k8s.io"
		}
	}

	// ensure subjects
	var subjects []rbacv1.Subject
	{
		for _, subject := range template.Spec.Template.Subjects {
			if subject.Kind == rbacv1.ServiceAccountKind && subject.Namespace == "" {
				subject.Namespace = namespace
			}
			subjects = append(subjects, subject)
		}
	}

	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: objectMeta,
		RoleRef:    roleRef,
		Subjects:   subjects,
	}, nil
}

func incompleteRoleRef(roleRef rbacv1.RoleRef) bool {
	if roleRef.Name == "" {
		return true
	}
	if roleRef.Kind != "Role" && roleRef.Kind != "ClusterRole" {
		return true
	}
	return false
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
