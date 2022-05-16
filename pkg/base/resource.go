package base

import (
	"github.com/giantswarm/micrologger"
	"k8s.io/client-go/kubernetes"
)

type K8sClientWithLogging interface {
	K8sClient() kubernetes.Interface
	Logger() micrologger.Logger
}
