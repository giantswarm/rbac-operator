package membership

import (
	"github.com/giantswarm/k8sclient/v4/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	rbacv1 "k8s.io/api/rbac/v1"

	"k8s.io/client-go/kubernetes"
)

const (
	Name = "membership"
)

type Config struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
}

type Resource struct {
	k8sClient kubernetes.Interface
	logger    micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		k8sClient: config.K8sClient.K8sClient(),
		logger:    config.Logger,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}

func isTargetRoleBinding(roleBinding rbacv1.RoleBinding) bool {
	for _, subject := range roleBinding.Subjects {
		if (subject.Kind == "Group" || subject.Kind == "User") && subject.Name != "" {
			return true
		}
	}
	return false
}
