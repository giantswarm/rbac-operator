package externalresources

import (
	"context"

	"github.com/giantswarm/rbac-operator/pkg/rbac"

	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/pkg/label"
	"github.com/giantswarm/rbac-operator/pkg/project"
	"github.com/giantswarm/rbac-operator/service/controller/rbac/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	var err error

	ns, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if !key.HasOrganizationOrCustomerLabel(ns) {
		return nil
	}

	orgNamespace := ns.Name
	if !pkgkey.IsOrgNamespace(orgNamespace) {
		return nil
	}

	// List roleBindings in org-namespace
	orgRoleBindings, err := r.k8sClient.RbacV1().RoleBindings(orgNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return microerror.Mask(err)
	} else if len(orgRoleBindings.Items) == 0 {
		return nil
	}

	// Ensure RoleBinding for default app catalogs access and ClusterRoleBinding for releases access
	// Ensure RoleBinding for access to the organization CR by name
	err = r.ensureAll(ctx, orgNamespace, orgRoleBindings)
	if err != nil {
		return microerror.Mask(err)
	}

	// Ensure RoleBindings for read/write access to cluster namespace resources
	err = r.ensureClusterNamespaceAccess(ctx, orgNamespace, orgRoleBindings)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// Ensures that Subjects with full read access to the org-namespace also have
// - read-access to resources in the org cluster namespaces
// Ensures that Subjects with admin access to the org-namespace also have
// - write-access toresources in the org cluster namespaces
func (r *Resource) ensureClusterNamespaceAccess(ctx context.Context, orgNamespace string, orgRoleBindings *rbacv1.RoleBindingList) error {
	var err error
	organization := pkgkey.OrganizationName(orgNamespace)

	subjects := getUniqueSubjectsWithClusterRoleRef(orgRoleBindings, pkgkey.DefaultReadAllPermissionsName)
	err = r.ensureRoleBindingToClusterRole(ctx, subjects, pkgkey.ReadClusterNamespaceAppsRole, orgNamespace, pkgkey.OrganizationReadClusterNamespaceRoleBindingName(organization))
	if err != nil {
		return microerror.Mask(err)
	}

	subjects = getUniqueSubjectsWithClusterRoleRef(orgRoleBindings, pkgkey.ClusterAdminClusterRoleName)
	err = r.ensureRoleBindingToClusterRole(ctx, subjects, pkgkey.WriteClusterNamespaceAppsRole, orgNamespace, pkgkey.OrganizationWriteClusterNamespaceRoleBindingName(organization))
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) ensureRoleBindingToClusterRole(ctx context.Context, subjects []rbacv1.Subject, clusterRole string, namespace string, name string) error {
	var err error

	roleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
			Namespace: namespace,
		},
		Subjects: subjects,
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterRole,
		},
	}

	if err = rbac.CreateOrUpdateRoleBinding(r, ctx, roleBinding.Namespace, roleBinding); err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// Ensures that Subjects with any sort of access in the organization namespace also have read access to
// - releases (non-namespaced)
// - app catalogs and app catalog entries in the default namespace
// - organization cr by name
func (r *Resource) ensureAll(ctx context.Context, orgNamespace string, orgRoleBindings *rbacv1.RoleBindingList) error {
	var err error

	// Collect the subjects that need access
	subjects := getUniqueSubjects(orgRoleBindings)
	// Ensure RoleBinding for default app catalogs access
	err = r.ensureDefaultCatalogsRoleBinding(ctx, subjects, pkgkey.OrganizationName(orgNamespace))
	if err != nil {
		return microerror.Mask(err)
	}

	// Ensure ClusterRoleBinding for releases access
	err = r.ensureReleasesClusterRoleBinding(ctx, subjects, pkgkey.OrganizationName(orgNamespace))
	if err != nil {
		return microerror.Mask(err)
	}

	// Ensure ClusterRoleBinding for access to organization cr by name
	err = r.ensureOrganizationClusterRoleBinding(ctx, subjects, pkgkey.OrganizationName(orgNamespace))
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) ensureOrganizationClusterRoleBinding(ctx context.Context, subjects []rbacv1.Subject, organization string) error {
	var err error

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: pkgkey.OrganizationReadOrganizationClusterRoleBindingName(organization),
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Subjects: subjects,
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     pkgkey.OrganizationReadClusterRoleName(organization),
		},
	}

	if err = rbac.CreateOrUpdateClusterRoleBinding(r, ctx, clusterRoleBinding); err != nil {
		return microerror.Mask(err)
	}
	return nil
}

func (r *Resource) ensureReleasesClusterRoleBinding(ctx context.Context, subjects []rbacv1.Subject, organization string) error {
	var err error

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: pkgkey.OrganizationReadReleasesClusterRoleBindingName(organization),
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Subjects: subjects,
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     pkgkey.ReadReleasesRole,
		},
	}

	if err = rbac.CreateOrUpdateClusterRoleBinding(r, ctx, clusterRoleBinding); err != nil {
		return microerror.Mask(err)
	}
	return nil
}

func (r *Resource) ensureDefaultCatalogsRoleBinding(ctx context.Context, subjects []rbacv1.Subject, organization string) error {
	var err error

	roleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: pkgkey.OrganizationReadDefaultCatalogsRoleBindingName(organization),
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
			Namespace: pkgkey.DefaultNamespaceName,
		},
		Subjects: subjects,
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     pkgkey.ReadDefaultCatalogsRole,
		},
	}

	if err = rbac.CreateOrUpdateRoleBinding(r, ctx, roleBinding.Namespace, roleBinding); err != nil {
		return microerror.Mask(err)
	}

	return nil
}
