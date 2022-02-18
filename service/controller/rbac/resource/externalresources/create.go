package externalresources

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/pkg/label"
	"github.com/giantswarm/rbac-operator/pkg/project"
	"github.com/giantswarm/rbac-operator/service/controller/rbac/key"
)

// Ensures that Subjects with any sort of access in the organization namespace also have read access to
// - releases (non-namespaced)
// - app catalogs and app catalog entries in the default namespace
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	var err error

	ns, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	orgNamespace := ns.Name

	// List roleBindings in org-namespace
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Listing RoleBindings in namespace %s.", orgNamespace))
	orgRoleBindings, err := r.k8sClient.RbacV1().RoleBindings(orgNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return microerror.Mask(err)
	} else if len(orgRoleBindings.Items) == 0 {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("No RoleBindings found in namespace %s.", orgNamespace))
		return nil
	}

	// Collect the subjects that need access
	var subjects []rbacv1.Subject
	for _, roleBinding := range orgRoleBindings.Items {
		if roleBindingHasReference(roleBinding) && roleBindingHasSubject(roleBinding) {
			subjects = append(subjects, roleBinding.Subjects...)
		}
	}
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

	if err = r.createOrUpdateClusterRoleBinding(ctx, clusterRoleBinding); err != nil {
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

	if err = r.createOrUpdateRoleBinding(ctx, roleBinding.Namespace, roleBinding); err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) createOrUpdateClusterRoleBinding(ctx context.Context, clusterrolebinding *rbacv1.ClusterRoleBinding) error {
	existingClusterRoleBinding, err := r.k8sClient.RbacV1().ClusterRoleBindings().Get(ctx, clusterrolebinding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating clusterrolebinding %#q", clusterrolebinding.Name))

		_, err := r.k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, clusterrolebinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrolebinding %#q has been created", clusterrolebinding.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else if clusterRoleBindingNeedsUpdate(clusterrolebinding, existingClusterRoleBinding) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating clusterrolebinding %#q", clusterrolebinding.Name))
		_, err := r.k8sClient.RbacV1().ClusterRoleBindings().Update(ctx, clusterrolebinding, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrolebinding %#q has been updated", clusterrolebinding.Name))
	}
	return nil
}

func (r *Resource) createOrUpdateRoleBinding(ctx context.Context, namespace string, roleBinding *rbacv1.RoleBinding) error {
	existingRoleBinding, err := r.k8sClient.RbacV1().RoleBindings(namespace).Get(ctx, roleBinding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating rolebinding %#q in namespace %s", roleBinding.Name, namespace))

		_, err := r.k8sClient.RbacV1().RoleBindings(namespace).Create(ctx, roleBinding, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("rolebinding %#q in namespace %s has been created.", roleBinding.Name, namespace))

	} else if err != nil {
		return microerror.Mask(err)
	} else if roleBindingNeedsUpdate(roleBinding, existingRoleBinding) {
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating rolebinding %#q in namespace %s", roleBinding.Name, namespace))
		_, err := r.k8sClient.RbacV1().RoleBindings(namespace).Update(ctx, roleBinding, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("rolebinding %#q in namespace %s has been updated.", roleBinding.Name, namespace))

	}

	return nil
}
