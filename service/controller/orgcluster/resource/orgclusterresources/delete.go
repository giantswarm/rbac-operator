package orgclusterresources

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkgkey "github.com/giantswarm/rbac-operator/pkg/key"
	"github.com/giantswarm/rbac-operator/service/controller/orgcluster/key"
)

// Ensures that when a cluster is deleted, roles and roleBindings for cluster resource access are deleted as well
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	var err error

	cl, err := key.ToCluster(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	orgNamespace := cl.Namespace
	if !pkgkey.IsOrgNamespace(orgNamespace) {
		return nil
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
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Deleting %#q roleBinding.", roleBinding))

		err = r.k8sClient.K8sClient().RbacV1().RoleBindings(namespace).Delete(ctx, roleBinding, metav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			// nothing to be done
		}
		if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("RoleBinding %#q has been deleted.", roleBinding))
		}
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
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Deleting %#q role.", role))

		err = r.k8sClient.K8sClient().RbacV1().Roles(namespace).Delete(ctx, role, metav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			// nothing to be done
		}
		if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Role %#q has been deleted.", role))
		}
	}
	return nil
}
