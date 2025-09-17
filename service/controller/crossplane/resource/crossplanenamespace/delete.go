package crossplanenamespace

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/service/controller/crossplane/key"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	ns, err := toNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	// Only care about org namespaces
	if !pkgkey.IsOrgNamespace(ns.Name) {
		return nil
	}

	// Check if the crossplane-edit ClusterRoleBinding exists
	clusterRoleBindingName := key.GetClusterRoleBindingName(r.crossplaneBindTriggeringClusterRole)
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{}

	err = r.k8sClient.CtrlClient().Get(ctx, types.NamespacedName{Name: clusterRoleBindingName}, clusterRoleBinding)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// ClusterRoleBinding doesn't exist, nothing to do
			return nil
		}
		return microerror.Mask(err)
	}

	// Find and remove this org namespace's automation SA from the subjects
	automationSA := rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      pkgkey.AutomationServiceAccountName,
		Namespace: ns.Name,
	}

	var updatedSubjects []rbacv1.Subject
	removed := false
	for _, subject := range clusterRoleBinding.Subjects {
		if subject.Kind == automationSA.Kind &&
			subject.Name == automationSA.Name &&
			subject.Namespace == automationSA.Namespace {
			// Skip this subject (remove it)
			removed = true
			continue
		}
		updatedSubjects = append(updatedSubjects, subject)
	}

	if !removed {
		// Subject wasn't in the binding anyway, nothing to do
		return nil
	}

	// Update the ClusterRoleBinding with the new subjects list
	clusterRoleBinding.Subjects = updatedSubjects
	err = r.k8sClient.CtrlClient().Update(ctx, clusterRoleBinding)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "info", "message",
		fmt.Sprintf("Removed %s:automation from crossplane-edit ClusterRoleBinding for deleted org namespace", ns.Name))

	return nil
}
