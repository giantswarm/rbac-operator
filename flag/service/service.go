package service

import (
	"github.com/giantswarm/operatorkit/flag/service/kubernetes"
	"github.com/giantswarm/rbac-operator/flag/service/namespaceauth"
)

// Service is an intermediate data structure for command line configuration flags.
type Service struct {
	NamespaceAuth namespaceauth.NamespaceAuth
	Kubernetes    kubernetes.Kubernetes
}
