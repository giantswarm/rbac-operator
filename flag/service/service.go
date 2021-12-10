package service

import (
	"github.com/giantswarm/operatorkit/v4/pkg/flag/service/kubernetes"
)

// Service is an intermediate data structure for command line configuration flags.
type Service struct {
	Kubernetes kubernetes.Kubernetes

	WriteAllCustomerGroup   string
	WriteAllGiantswarmGroup string
}
