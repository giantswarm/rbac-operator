package core

import (
	"context"
	"fmt"
	"reflect"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/rbac-operator/pkg/base"
)

func ServiceAccountNeedsUpdate(desiredSA, existingSA *corev1.ServiceAccount) bool {
	desiredSAHasManualAutomountSAToken := desiredSA.AutomountServiceAccountToken != nil && !*desiredSA.AutomountServiceAccountToken
	existingSAHasManualAutomountSAToken := existingSA.AutomountServiceAccountToken != nil && !*existingSA.AutomountServiceAccountToken

	if desiredSAHasManualAutomountSAToken != existingSAHasManualAutomountSAToken {
		return true
	}

	if desiredSAHasManualAutomountSAToken {
		if len(desiredSA.Secrets) != len(existingSA.Secrets) {
			return true
		}

		if !reflect.DeepEqual(desiredSA.Secrets, existingSA.Secrets) {
			return true
		}
	}

	if len(desiredSA.ImagePullSecrets) != len(existingSA.ImagePullSecrets) {
		return true
	}

	return len(desiredSA.ImagePullSecrets) != 0 && !reflect.DeepEqual(desiredSA.ImagePullSecrets, existingSA.ImagePullSecrets)
}

func CreateOrUpdateServiceAccount(c base.K8sClientWithLogging, ctx context.Context, namespace string, desiredSA *corev1.ServiceAccount) error {
	existingSA, err := c.K8sClient().CoreV1().ServiceAccounts(namespace).Get(ctx, desiredSA.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("creating serviceaccount %#q in namespace %s", desiredSA.Name, namespace))

		_, err := c.K8sClient().CoreV1().ServiceAccounts(namespace).Create(ctx, desiredSA, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// do nothing
		} else if err != nil {
			return microerror.Mask(err)
		}

		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("serviceaccount %#q in namespace %s has been created", desiredSA.Name, namespace))

	} else if err != nil {
		return microerror.Mask(err)
	} else if ServiceAccountNeedsUpdate(existingSA, desiredSA) {
		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("updating serviceaccount %#q in namespace %s", desiredSA.Name, namespace))

		_, err := c.K8sClient().CoreV1().ServiceAccounts(namespace).Update(ctx, desiredSA, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}

		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("serviceaccount %#q in namespace %s has been updated", desiredSA.Name, namespace))
	} else {
		c.Logger().LogCtx(ctx, "level", "info", "message", fmt.Sprintf("no need to update serviceaccount %#q in namespace %s", desiredSA.Name, namespace))
	}

	return nil
}
