package crossplanenamespace

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/service/controller/crossplane/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
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
			// ClusterRoleBinding doesn't exist yet, nothing to do
			return nil
		}
		return microerror.Mask(err)
	}

	// Check if this org namespace's automation SA is already in the subjects
	automationSA := rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      pkgkey.AutomationServiceAccountName,
		Namespace: ns.Name,
	}

	for _, subject := range clusterRoleBinding.Subjects {
		if subject.Kind == automationSA.Kind &&
			subject.Name == automationSA.Name &&
			subject.Namespace == automationSA.Namespace {
			// Already exists, nothing to do
			return nil
		}
	}

	// Add the new automation SA to the subjects
	clusterRoleBinding.Subjects = append(clusterRoleBinding.Subjects, automationSA)

	// Update the ClusterRoleBinding
	err = r.k8sClient.CtrlClient().Update(ctx, clusterRoleBinding)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "info", "message",
		fmt.Sprintf("Added %s:automation to crossplane-edit ClusterRoleBinding for new org namespace", ns.Name))

	return nil
}

func toNamespace(v interface{}) (corev1.Namespace, error) {
	if v == nil {
		return corev1.Namespace{}, microerror.Maskf(wrongTypeError, "expected non-nil, got %#v'", v)
	}

	p, ok := v.(*corev1.Namespace)
	if !ok {
		return corev1.Namespace{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", p, v)
	}

	c := p.DeepCopy()

	return *c, nil
}

var wrongTypeError = &microerror.Error{
	Kind: "wrongTypeError",
}
