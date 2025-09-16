package crossplaneauth

import (
	"context"
	"fmt"

	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/pkg/project"
	pkgrbac "github.com/giantswarm/rbac-operator/pkg/rbac"
	"github.com/giantswarm/rbac-operator/service/controller/crossplane/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToClusterRole(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	if cr.Name != r.crossplaneBindTriggeringClusterRole {
		return nil
	}

	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      pkgkey.AutomationServiceAccountName,
			Namespace: pkgkey.DefaultNamespaceName,
		},
	}
	
	// Add automation ServiceAccounts from all org namespaces
	orgAutomationSAs, err := r.getOrgAutomationServiceAccounts(ctx)
	if err != nil {
		return microerror.Mask(err)
	}
	subjects = append(subjects, orgAutomationSAs...)
	
	for _, group := range r.customerAdminGroups {
		subjects = append(subjects, rbacv1.Subject{
			APIGroup:  "rbac.authorization.k8s.io",
			Kind:      "Group",
			Name:      group.Name,
			Namespace: pkgkey.DefaultNamespaceName,
		})
	}

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: key.GetClusterRoleBindingName(r.crossplaneBindTriggeringClusterRole),
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
			Annotations: map[string]string{
				annotation.Notes: "Grants customer's cluster-admin permissions to use crossplane rbac-manager managed crossplane:edit ClusterRole",
			},
		},
		Subjects: subjects,
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     r.crossplaneBindTriggeringClusterRole,
		},
	}

	err = pkgrbac.CreateOrUpdateClusterRoleBinding(r, ctx, clusterRoleBinding)

	if apierrors.IsAlreadyExists(err) {
		// do nothing
	} else if err != nil {
		return microerror.Mask(err)
	}
	r.logger.LogCtx(ctx, "level", "debug", "message",
		fmt.Sprintf("ClusterRoleBinding %#q between customer's admin group and rbac-manager of Crossplane has been checked",
			r.crossplaneBindTriggeringClusterRole))

	return nil
}

func (r *Resource) getOrgAutomationServiceAccounts(ctx context.Context) ([]rbacv1.Subject, error) {
	var subjects []rbacv1.Subject
	
	// Get all namespaces that start with "org-" pattern
	namespaces := &corev1.NamespaceList{}
	if err := r.k8sClient.CtrlClient().List(ctx, namespaces); err != nil {
		return nil, microerror.Mask(err)
	}
	
	for _, ns := range namespaces.Items {
		if pkgkey.IsOrgNamespace(ns.Name) {
			subjects = append(subjects, rbacv1.Subject{
				Kind:      "ServiceAccount", 
				Name:      pkgkey.AutomationServiceAccountName,
				Namespace: ns.Name,
			})
		}
	}
	
	return subjects, nil
}
