package rolebinding

import (
	"context"
	"fmt"

	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/rbac-operator/api/v1alpha1"
	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/pkg/project"
	"github.com/giantswarm/rbac-operator/pkg/rbac"
	"github.com/giantswarm/rbac-operator/service/controller/rolebindingtemplate/key"
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
	for _, ns := range namespaces {
		roleBinding, err := getRoleBindingFromTemplate(template, ns)
		if err != nil {
			return microerror.Mask(err)
		}

		roleBinding = cleanSubjects(roleBinding, ns)
		if len(roleBinding.Subjects) > 0 {
			if err = rbac.CreateOrUpdateRoleBinding(r, ctx, ns, roleBinding); err != nil {
				r.logger.Errorf(ctx, err, "Failed to apply roleBinding %s to namespace %s", roleBinding.Name, ns)
				return microerror.Maskf(roleBindingCreationFailedError, "failed to apply role binding to namespace %s: %v", ns, err)
			}
			r.logger.Debugf(ctx, "Successfully applied roleBinding %s to namespace %s", roleBinding.Name, ns)
			status = append(status, ns)
		}
	}

	// go through old list of namespaces and compare for scope changes
	for _, ns := range template.Status.Namespaces {
		if !contains(status, ns) {
			if err = rbac.DeleteRoleBinding(r, ctx, ns, getRoleBindingNameFromTemplate(template)); err != nil {
				r.logger.Errorf(ctx, err, "Failed to delete roleBinding from namespace %s", ns)
				return microerror.Mask(err)
			}
			r.logger.Debugf(ctx, "Successfully deleted roleBinding from namespace %s", ns)
		}
	}

	template.Status.Namespaces = status
	if err := r.k8sClient.CtrlClient().Status().Update(ctx, &template); err != nil {
		r.logger.Errorf(ctx, err, "Failed to update template status")
		return microerror.Mask(err)
	}

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

func cleanSubjects(roleBinding *rbacv1.RoleBinding, namespace string) *rbacv1.RoleBinding {
	// if the rolebinding is in a protected namespace, subjects can only be serviceAccounts in flux namespace or the same namespace
	if !pkgkey.IsProtectedNamespace(namespace) {
		return roleBinding
	}
	var validSubjects []rbacv1.Subject
	for _, subject := range roleBinding.Subjects {
		if subject.Kind != rbacv1.ServiceAccountKind {
			continue
		}
		if subject.Namespace != pkgkey.FluxNamespaceName && subject.Namespace != namespace {
			continue
		}
		validSubjects = append(validSubjects, subject)
	}
	roleBinding.Subjects = validSubjects
	return roleBinding
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
