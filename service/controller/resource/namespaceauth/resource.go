package namespaceauth

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"k8s.io/client-go/kubernetes"
)

const (
	Name = "namespaceauth"
)

type NamespaceAuth struct {
	ViewAllTargetGroup string
}

type Config struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	NamespaceAuth NamespaceAuth
}

type Resource struct {
	k8sClient kubernetes.Interface
	logger    micrologger.Logger

	namespaceAuth NamespaceAuth
}

func New(config Config) (*Resource, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.NamespaceAuth.ViewAllTargetGroup == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.NamespaceAuth.ViewAllTargetGroup must not be empty", config)
	}

	r := &Resource{
		k8sClient: config.K8sClient.K8sClient(),
		logger:    config.Logger,

		namespaceAuth: config.NamespaceAuth,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}

func ToNamespace(v interface{}) (corev1.Namespace, error) {
	if v == nil {
		return corev1.Namespace{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &corev1.Namespace{}, v)
	}

	p, ok := v.(*corev1.Namespace)
	if !ok {
		return corev1.Namespace{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &corev1.Namespace{}, v)
	}

	c := p.DeepCopy()

	return *c, nil
}
