package bootstrap

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (b *Bootstrap) createGlobalNamespace(ctx context.Context) error {

	globalNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "global",
			Labels: map[string]string{},
		},
	}

	_, err := b.k8sClient.CoreV1().Namespaces().Get(ctx, globalNS.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating namespace %#q", globalNS.Name))

		_, err := b.k8sClient.CoreV1().Namespaces().Create(ctx, globalNS, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("namespace %#q has been created", globalNS.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("namespace %#q already exists", globalNS.Name))
	}

	return nil
}

func (b *Bootstrap) createReadAllClusterRole(ctx context.Context) error {

	lists, err := b.k8sClient.Discovery().ServerPreferredResources()
	if err != nil {
		panic(err)
	}

	var policyRules []rbacv1.PolicyRule
	{
		for _, list := range lists {
			if len(list.APIResources) == 0 {
				continue
			}
			gv, err := schema.ParseGroupVersion(list.GroupVersion)
			if err != nil {
				continue
			}
			for _, resource := range list.APIResources {
				if len(resource.Verbs) == 0 {
					continue
				}
				if isRestrictedResource(resource.Name) {
					continue
				}

				policyRule := rbacv1.PolicyRule{
					APIGroups: []string{gv.Group},
					Resources: []string{resource.Name},
					Verbs:     []string{"get", "list", "watch"},
				}
				policyRules = append(policyRules, policyRule)
			}
		}
	}

	readOnlyClusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "view-all",
		},
		Rules: policyRules,
	}

	_, err = b.k8sClient.RbacV1().ClusterRoles().Get(ctx, readOnlyClusterRole.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating clusterrole %#q", readOnlyClusterRole.Name))

		_, err := b.k8sClient.RbacV1().ClusterRoles().Create(ctx, readOnlyClusterRole, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrole %#q has been created", readOnlyClusterRole.Name))

	} else if err != nil {
		return microerror.Mask(err)
	} else {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clusterrole %#q already exists", readOnlyClusterRole.Name))
	}

	return nil
}

func (b *Bootstrap) createRoleBindings(ctx context.Context) error {
	// rolebindings in default and global namespaces
	// for view-all and admin groups

	return nil
}

func isRestrictedResource(resource string) bool {
	var restrictedResources = []string{"configmaps", "secrets"}

	for _, restrictedResource := range restrictedResources {
		if resource == restrictedResource {
			return true
		}
	}
	return false
}
