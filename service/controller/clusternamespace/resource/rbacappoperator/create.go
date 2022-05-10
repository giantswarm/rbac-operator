package rbacappoperator

import (
	"context"
	"fmt"

	"github.com/giantswarm/rbac-operator/pkg/rbac"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/rbac-operator/pkg/label"
	"github.com/giantswarm/rbac-operator/pkg/project"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/rbac-operator/service/controller/rbac/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cl, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(
		ctx, "level", "info",
		"message", fmt.Sprintf("Reconciling cluster namespace: %s.", cl.Name),
	)

	// Allow working with some generic resources across namespaces
	err = r.CreateClusterRoleAndBinding(ctx, cl)
	if err != nil {
		return err
	}

	// Allow getting catalog configmaps in giantswarm namespace
	err = r.CreateCatalogReaderRoleAndBinding(ctx, cl)
	if err != nil {
		return err
	}

	// Allow working with stuff in its own namespace
	err = r.CreateOwnNamespaceRoleAndBinding(ctx, cl)
	if err != nil {
		return err
	}

	return nil
}

func (r *Resource) CreateClusterRoleAndBinding(ctx context.Context, cl corev1.Namespace) error {
	var clusterRole = &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("app-operator-%s", cl.Name),
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"application.giantswarm.io"},
				Resources: []string{"apps"},
				Verbs:     []string{"get", "list", "update", "patch", "watch"},
			},
			{
				APIGroups: []string{"application.giantswarm.io"},
				Resources: []string{"apps/status"},
				Verbs:     []string{"create", "patch", "update"},
			},
			{
				APIGroups: []string{"application.giantswarm.io"},
				Resources: []string{"catalogs"},
				Verbs:     []string{"get", "list", "patch", "watch"},
			},
			{
				APIGroups: []string{"application.giantswarm.io"},
				Resources: []string{"appcatalogs"},
				Verbs:     []string{"create", "delete", "get", "list", "patch", "update", "watch"},
			},
			{
				APIGroups: []string{"application.giantswarm.io"},
				Resources: []string{"appcatalogs/status"},
				Verbs:     []string{"create", "patch", "update"},
			},
			{
				APIGroups: []string{"application.giantswarm.io"},
				Resources: []string{"appcatalogentries"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"list", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"watch"},
			},
		},
	}

	if err := rbac.CreateOrUpdateClusterRole(r, ctx, clusterRole); err != nil {
		return microerror.Mask(err)
	}

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("app-operator-%s", cl.Name),
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      fmt.Sprintf("app-operator-%s", cl.Name),
				Namespace: cl.Name,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterRole.Name,
		},
	}

	if err := rbac.CreateOrUpdateClusterRoleBinding(r, ctx, clusterRoleBinding); err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) CreateCatalogReaderRoleAndBinding(ctx context.Context, cl corev1.Namespace) error {
	var catalogReaderRole = &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("app-operator-%s", cl.Name),
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
			Namespace: "giantswarm",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"get", "patch"},
			},
		},
	}

	if err := rbac.CreateOrUpdateRole(r, ctx, "giantswarm", catalogReaderRole); err != nil {
		return microerror.Mask(err)
	}

	catalogReaderRoleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("app-operator-%s", cl.Name),
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
			Namespace: "giantswarm",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      fmt.Sprintf("app-operator-%s", cl.Name),
				Namespace: cl.Name,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     catalogReaderRole.Kind,
			Name:     catalogReaderRole.Name,
		},
	}

	if err := rbac.CreateOrUpdateRoleBinding(r, ctx, "giantswarm", catalogReaderRoleBinding); err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) CreateOwnNamespaceRoleAndBinding(ctx context.Context, cl corev1.Namespace) error {
	var ownNamespaceRole = &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("app-operator-%s", cl.Name),
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
			Namespace: cl.Name,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"namespaces"},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"*"},
			},
		},
	}

	if err := rbac.CreateOrUpdateRole(r, ctx, cl.Name, ownNamespaceRole); err != nil {
		return microerror.Mask(err)
	}

	ownNamespaceRoleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("app-operator-%s", cl.Name),
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
			Namespace: cl.Name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      fmt.Sprintf("app-operator-%s", cl.Name),
				Namespace: cl.Name,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     ownNamespaceRole.Kind,
			Name:     ownNamespaceRole.Name,
		},
	}

	if err := rbac.CreateOrUpdateRoleBinding(r, ctx, cl.Name, ownNamespaceRoleBinding); err != nil {
		return microerror.Mask(err)
	}

	return nil
}
