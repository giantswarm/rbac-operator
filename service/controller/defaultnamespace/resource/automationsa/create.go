package automationsa

import (
	"context"

	"github.com/giantswarm/rbac-operator/pkg/core"

	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/pkg/project"
	"github.com/giantswarm/rbac-operator/pkg/rbac"
	"github.com/giantswarm/rbac-operator/service/controller/defaultnamespace/key"
)

// EnsureCreated Ensures that the automation service account is created in the default namespace,
// and it has all the necessary RoleBindings and ClusterRoleBindings
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	namespace, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if !pkgkey.IsDefaultNamespace(namespace.Name) {
		return nil
	}

	err = r.createAutomationServiceAccount(ctx, namespace.Name)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.createReadAllClusterRoleBindingToAutomationSA(ctx, namespace.Name)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.createWriteAllRoleBindingToAutomationSA(ctx, namespace.Name)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.createWriteOrganizationsClusterRoleBindingToAutomationSA(ctx, namespace.Name)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.createWriteClientCertsClusterRoleBindingToAutomationSA(ctx, namespace.Name)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.createWriteSilencesClusterRoleBindingToAutomationSA(ctx, namespace.Name)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.createWriteAWSClusterRoleIdentityClusterRoleBindingToAutomationSA(ctx, namespace.Name)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// Ensures the 'automation' service account in the default namespace.
func (r *Resource) createAutomationServiceAccount(ctx context.Context, namespace string) error {

	automationSA := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: pkgkey.AutomationServiceAccountName,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
	}

	return core.CreateOrUpdateServiceAccount(r, ctx, namespace, automationSA)
}

// Ensures the ClusterRoleBinding 'read-all-customer-sa' between
// ClusterRole 'read-all' and the ServiceAccount 'automation'.
func (r *Resource) createReadAllClusterRoleBindingToAutomationSA(ctx context.Context, namespace string) error {
	clusterRoleBindingName := pkgkey.ReadAllAutomationSAClusterRoleBindingName()

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
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      pkgkey.AutomationServiceAccountName,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     pkgkey.DefaultReadAllPermissionsName,
		},
	}

	return rbac.CreateOrUpdateClusterRoleBinding(r, ctx, readAllClusterRoleBinding)
}

// Ensures a RoleBinding 'write-all-customer-sa' between
// ClusterRole 'cluster-admin' and ServiceAccount 'automation'
// in namespace 'default'.
func (r *Resource) createWriteAllRoleBindingToAutomationSA(ctx context.Context, namespace string) error {
	roleBindingName := pkgkey.WriteAllAutomationSARoleBindingName()

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
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      pkgkey.AutomationServiceAccountName,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     pkgkey.ClusterAdminClusterRoleName,
		},
	}

	return rbac.CreateOrUpdateRoleBinding(r, ctx, namespace, writeAllRoleBinding)
}

// Ensures the ClusterRoleBinding 'write-organizations-customer-sa' between
// ClusterRole 'write-organizations' and ServiceAccount 'automation'.
func (r *Resource) createWriteOrganizationsClusterRoleBindingToAutomationSA(ctx context.Context, namespace string) error {
	clusterRoleBindingName := pkgkey.WriteOrganizationsAutomationSARoleBindingName()

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
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      pkgkey.AutomationServiceAccountName,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     pkgkey.WriteOrganizationsPermissionsName,
		},
	}

	return rbac.CreateOrUpdateClusterRoleBinding(r, ctx, writeOrganizationsClusterRoleBinding)
}

// Ensures the ClusterRoleBinding 'write-client-certificates-customer-sa' between
// ClusterRole 'write-client-certificates' and ServiceAccount 'automation'.
func (r *Resource) createWriteClientCertsClusterRoleBindingToAutomationSA(ctx context.Context, namespace string) error {
	clusterRoleBindingName := pkgkey.WriteClientCertsAutomationSARoleBindingName()

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
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
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      pkgkey.AutomationServiceAccountName,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     pkgkey.WriteClientCertsPermissionsName,
		},
	}

	return rbac.CreateOrUpdateClusterRoleBinding(r, ctx, clusterRoleBinding)
}

// Ensures the ClusterRoleBinding 'write-silences-customer-sa' between
// ClusterRole 'write-silences' and ServiceAccount 'automation'.
func (r *Resource) createWriteSilencesClusterRoleBindingToAutomationSA(ctx context.Context, namespace string) error {
	clusterRoleBindingName := pkgkey.WriteSilencesAutomationSARoleBindingName()

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
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
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      pkgkey.AutomationServiceAccountName,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     pkgkey.WriteSilencesPermissionsName,
		},
	}

	return rbac.CreateOrUpdateClusterRoleBinding(r, ctx, clusterRoleBinding)
}

// Ensures the ClusterRoleBinding 'write-aws-cluster-role-identity-customer-sa' between
// ClusterRole 'write-aws-cluster-role-identity' and ServiceAccount 'automation'.
func (r *Resource) createWriteAWSClusterRoleIdentityClusterRoleBindingToAutomationSA(ctx context.Context, ns string) error {
	automationSA := pkgkey.AutomationServiceAccountName

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: pkgkey.WriteAWSClusterRoleIdentityAutomationSARoleBindingName(),
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      automationSA,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     pkgkey.WriteAWSClusterRoleIdentityPermissionsName,
		},
	}

	return rbac.CreateOrUpdateClusterRoleBinding(r, ctx, clusterRoleBinding)
}
