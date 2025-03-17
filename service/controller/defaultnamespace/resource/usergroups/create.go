package usergroups

import (
	"context"

	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/pkg/project"
	"github.com/giantswarm/rbac-operator/pkg/rbac"
	"github.com/giantswarm/rbac-operator/service/controller/defaultnamespace/key"
	"github.com/giantswarm/rbac-operator/service/internal/accessgroup"
)

// EnsureCreated Ensures that ClusterRoleBindings and RoleBindings
// to roles with permissions to read and write all common and custom resources
// are created for customer and giantswarm groups
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	namespace, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if !pkgkey.IsDefaultNamespace(namespace.Name) {
		return nil
	}

	err = r.createWriteAllClusterRoleBindingToGSGroup(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	if len(r.customerAdminGroups) > 0 {
		err = r.createReadAllClusterRoleBindingToCustomerGroup(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		err = r.createWriteAllRoleBindingToCustomerGroup(ctx, namespace.Name)
		if err != nil {
			return microerror.Mask(err)
		}

		err = r.createWriteOrganizationsClusterRoleBindingToCustomerGroup(ctx)
		if err != nil {
			return microerror.Mask(err)
		}
	} else if len(r.customerReaderGroups) > 0 {
		err = r.createReadAllClusterRoleBindingToCustomerGroup(ctx)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

// Ensures the ClusterRoleBinding 'write-organizations-customer-group' between
// ClusterRole 'write-organizations' and the customer admin group.
func (r *Resource) createWriteOrganizationsClusterRoleBindingToCustomerGroup(ctx context.Context) error {
	subjects := accessgroup.GroupsToSubjects(r.customerAdminGroups)
	if len(subjects) == 0 {
		return microerror.Maskf(invalidConfigError, "empty customer admin group name given")
	}

	clusterRoleBindingName := pkgkey.WriteOrganizationsCustomerGroupClusterRoleBindingName()

	writeOrganizationsClusterRoleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleBindingName,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Subjects: subjects,
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     pkgkey.WriteOrganizationsPermissionsName,
		},
	}

	return rbac.CreateOrUpdateClusterRoleBinding(r, ctx, writeOrganizationsClusterRoleBinding)
}

// Ensures the ClusterRoleBinding 'read-all-customer-group' between
// ClusterRole 'read-all' and the customer admin group.
func (r *Resource) createReadAllClusterRoleBindingToCustomerGroup(ctx context.Context) error {
	subjects := accessgroup.GroupsToSubjects(r.customerAdminGroups)
	if len(r.customerReaderGroups) > 0 {
		subjects = append(subjects, accessgroup.GroupsToSubjects(r.customerReaderGroups)...)
	}

	if len(subjects) == 0 {
		return microerror.Maskf(invalidConfigError, "empty customer admin group name given")
	}

	clusterRoleBindingName := pkgkey.ReadAllCustomerGroupClusterRoleBindingName()

	readAllClusterRoleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleBindingName,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Subjects: subjects,
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     pkgkey.DefaultReadAllPermissionsName,
		},
	}

	return rbac.CreateOrUpdateClusterRoleBinding(r, ctx, readAllClusterRoleBinding)
}

// Ensures the ClusterRoleBinding 'write-all-customer-group' between
// ClusterRole 'cluster-admin' and the customer admin group.
func (r *Resource) createWriteAllRoleBindingToCustomerGroup(ctx context.Context, namespace string) error {
	subjects := accessgroup.GroupsToSubjects(r.customerAdminGroups)
	if len(subjects) == 0 {
		return microerror.Maskf(invalidConfigError, "empty customer admin group name given")
	}

	roleBindingName := pkgkey.WriteAllCustomerGroupRoleBindingName()

	writeAllRoleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: roleBindingName,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Subjects: subjects,
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     pkgkey.ClusterAdminClusterRoleName,
		},
	}

	return rbac.CreateOrUpdateRoleBinding(r, ctx, namespace, writeAllRoleBinding)
}

// Ensures the ClusterRoleBinding 'write-all-giantswarm-group' between
// ClusterRole 'cluster-admin' and the Giant Swarm admin group.
func (r *Resource) createWriteAllClusterRoleBindingToGSGroup(ctx context.Context) error {
	clusterRoleBindingName := pkgkey.WriteAllGSGroupClusterRoleBindingName()

	readAllClusterRoleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleBindingName,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Subjects: accessgroup.GroupsToSubjects(r.gsAdminGroups),
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     pkgkey.ClusterAdminClusterRoleName,
		},
	}

	return rbac.CreateOrUpdateClusterRoleBinding(r, ctx, readAllClusterRoleBinding)
}
