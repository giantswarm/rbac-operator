package rbaccleaner

import (
	"context"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/rbac-operator/service/controller/clusternamespace/key"
)

// Ensures that ClusterRoleBinding and ClusterRole deployed by legacy
// app-operators to Cluster namespaces are *deleted*. We will be using another
// ClusterRole and RoleBinding created by another resource.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	var err error

	cl, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	nameToDelete := key.ClusterRoleNameFromNamespace(cl)

	err = r.k8sClient.K8sClient().RbacV1().ClusterRoleBindings().Delete(ctx, nameToDelete, metav1.DeleteOptions{})
	if errors.IsNotFound(err) {
		// all good
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	err = r.k8sClient.K8sClient().RbacV1().ClusterRoles().Delete(ctx, nameToDelete, metav1.DeleteOptions{})
	if errors.IsNotFound(err) {
		// all good
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
