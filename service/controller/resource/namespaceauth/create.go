package namespaceauth

import (
	"context"

	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/rbac-operator/service/controller/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	namespace, err := key.ToNamespace(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	resources, err := r.k8sClient.Discovery().ServerPreferredResources()
	if err != nil {
		return microerror.Mask(err)
	}

	viewAllRole, err := newViewAllRole(resources)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "creating view role %#q in namespace %#q", viewAllRole.Name, namespace.Name)

	_, err = r.k8sClient.RbacV1().Roles(namespace.Name).Get(viewAllRole.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err := r.k8sClient.RbacV1().Roles(namespace.Name).Create(viewAllRole)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "created view role")

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "view role already exists")	
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "view role exists")

	viewAllRoleBinding := newViewAllRoleBinding(r.namespaceAuth.ViewAllTargetGroup)

	r.logger.LogCtx(ctx, "level", "debug", "message", "creating view role binding")

	_, err = r.k8sClient.RbacV1().RoleBindings(namespace.Name).Get(viewAllRoleBinding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err := r.k8sClient.RbacV1().RoleBindings(namespace.Name).Create(viewAllRoleBinding)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "created view role binding")

	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "view role binding exists")

	return nil
}
