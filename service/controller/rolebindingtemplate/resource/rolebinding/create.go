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

	for _, ns := range namespaces {
		roleBinding, err := getRoleBindingFromTemplate(template, ns)
		if err != nil {
			return microerror.Mask(err)
		}

		if err = rbac.CreateOrUpdateRoleBinding(r, ctx, ns, roleBinding); err != nil {
			return microerror.Mask(err)
		}
	}

	template.Status.Namespaces = namespaces
	if err := r.k8sClient.CtrlClient().Status().Update(ctx, &template); err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func getRoleBindingFromTemplate(template v1alpha1.RoleBindingTemplate, namespace string) (*rbacv1.RoleBinding, error) {
	roleBinding := &template.Spec.Template.Spec

	// ensure namespaced name
	if roleBinding.Name == "" {
		roleBinding.Name = template.Name
	}
	roleBinding.Namespace = namespace

	// ensure type meta
	roleBinding.TypeMeta = metav1.TypeMeta{
		Kind:       "RoleBinding",
		APIVersion: "rbac.authorization.k8s.io/v1",
	}

	// add labels and annotations
	labels := roleBinding.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[label.ManagedBy] = project.Name()
	roleBinding.SetLabels(labels)
	annotations := roleBinding.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	if annotations[annotation.Notes] == "" {
		annotations[annotation.Notes] = fmt.Sprintf("Generated based on RoleBindingTemplate %s", template.Name)
	}
	roleBinding.SetAnnotations(annotations)

	// ensure role reference
	if incompleteRoleRef(roleBinding.RoleRef) {
		return nil, microerror.Maskf(invalidConfigError, "RoleBindingTemplate %s has incomplete roleRef %v", template.Name, roleBinding.RoleRef)
	}
	if roleBinding.RoleRef.APIGroup == "" {
		roleBinding.RoleRef.APIGroup = "rbac.authorization.k8s.io"
	}

	for i, s := range roleBinding.Subjects {
		if s.Kind == rbacv1.ServiceAccountKind && s.Namespace == "" {
			roleBinding.Subjects[i].Namespace = namespace
		}
	}

	return roleBinding, nil
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
