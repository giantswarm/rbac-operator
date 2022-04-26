package clusternamespaceresources

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/rbac-operator/service/controller/clusternamespace/key"
)

// Ensures that when a cluster is deleted, roles and roleBindings for cluster resource access are deleted as well
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	var err error

	cl, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	// Delete RoleBindings in org Cluster namespace
	for _, referencedRole := range referencedClusterRoles() {
		err = r.deleteRoleBinding(ctx, cl.Name, referencedRole.roleBindingName)
		if err != nil {
			return microerror.Mask(err)
		}
		err = r.deleteRole(ctx, cl.Name, referencedRole.roleName)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	err = r.deleteRoleBinding(ctx, cl.Name, fluxCRDRolePair.roleBindingName)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.deleteRoleBinding(ctx, cl.Name, fluxNSRolePair.roleBindingName)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) deleteRoleBinding(ctx context.Context, namespace string, roleBinding string) error {
	var err error

	_, err = r.k8sClient.K8sClient().RbacV1().RoleBindings(namespace).Get(ctx, roleBinding, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		// nothing to be done
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("Deleting RoleBinding %#q in namespace %s.", roleBinding, namespace))

		err = r.k8sClient.K8sClient().RbacV1().RoleBindings(namespace).Delete(ctx, roleBinding, metav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			// nothing to be done
		} else if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("RoleBinding %#q in namespace %s has been deleted.", roleBinding, namespace))
	}
	return nil
}

func (r *Resource) deleteRole(ctx context.Context, namespace string, role string) error {
	var err error

	_, err = r.k8sClient.K8sClient().RbacV1().Roles(namespace).Get(ctx, role, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		// nothing to be done
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("Deleting Role %#q in namespace %s.", role, namespace))

		err = r.k8sClient.K8sClient().RbacV1().Roles(namespace).Delete(ctx, role, metav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			// nothing to be done
		} else if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("Role %#q in namespace %s has been deleted.", role, namespace))
	}
	return nil
}
