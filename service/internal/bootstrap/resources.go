package bootstrap

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
