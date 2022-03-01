package externalresources

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/service/controller/rbac/key"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	var err error

	ns, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	organization := pkgkey.OrganizationName(ns.Name)

	// Delete RoleBinding for default app catalogs access
	err = r.deleteRoleBinding(ctx, pkgkey.DefaultNamespaceName, pkgkey.OrganizationReadDefaultCatalogsRoleBindingName(organization))
	if err != nil {
		return microerror.Mask(err)
	}

	// Delete ClusterRoleBinding for releases access
	err = r.deleteClusterRoleBinding(ctx, pkgkey.OrganizationReadReleasesClusterRoleBindingName(organization))
	if err != nil {
		return microerror.Mask(err)
	}

	// Delete ClusterRoleBinding to read cluster namespaces
	err = r.deleteClusterRoleBinding(ctx, pkgkey.OrganizationReadClusterNamespaceRoleBindingName(organization))
	if err != nil {
		return microerror.Mask(err)
	}

	// Delete ClusterRoleBinding to write in cluster namespaces
	err = r.deleteClusterRoleBinding(ctx, pkgkey.OrganizationWriteClusterNamespaceRoleBindingName(organization))
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}

func (r *Resource) deleteClusterRoleBinding(ctx context.Context, clusterRoleBinding string) error {
	var err error

	_, err = r.k8sClient.RbacV1().ClusterRoleBindings().Get(ctx, clusterRoleBinding, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		// nothing to be done
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("deleting  clusterrolebinding %#q", clusterRoleBinding))

		err = r.k8sClient.RbacV1().ClusterRoleBindings().Delete(ctx, clusterRoleBinding, metav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			// nothing to be done
		} else if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("clusterrolebinding %#q has been deleted", clusterRoleBinding))
	}
	return nil
}

func (r *Resource) deleteRoleBinding(ctx context.Context, namespace string, roleBinding string) error {
	var err error

	_, err = r.k8sClient.RbacV1().RoleBindings(namespace).Get(ctx, roleBinding, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		// nothing to be done
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("deleting rolebinding %#q from namespace %s", roleBinding, namespace))

		err = r.k8sClient.RbacV1().RoleBindings(namespace).Delete(ctx, roleBinding, metav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			// nothing to be done
		} else if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("rolebinding %#q has been deleted from namespace %s", roleBinding, namespace))
	}
	return nil
}
