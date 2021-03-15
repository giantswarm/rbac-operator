package bootstrap

import (
	"context"

	"github.com/giantswarm/k8sclient/v4/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/client-go/kubernetes"
)

type Config struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	// internal
	CustomerAdminGroup string
	GSAdminGroup       string
}

type Bootstrap struct {
	k8sClient kubernetes.Interface
	logger    micrologger.Logger

	// internal
	customerAdminGroup string
	gsAdminGroup       string
}

func New(config Config) (*Bootstrap, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.CustomerAdminGroup == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.CustomerAdminGroup must not be empty", config)
	}

	if config.GSAdminGroup == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.GSAdminGroup must not be empty", config)
	}

	r := &Bootstrap{
		k8sClient: config.K8sClient.K8sClient(),
		logger:    config.Logger,

		customerAdminGroup: config.CustomerAdminGroup,
		gsAdminGroup:       config.GSAdminGroup,
	}

	return r, nil
}

func (b *Bootstrap) Run() error {
	var err error

	ctx := context.Background()

	err = b.createAutomationServiceAccount(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.createReadAllClusterRole(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.createReadAllClusterRoleBindingToCustomerGroup(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.createReadAllClusterRoleBindingToAutomationSA(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.createWriteAllClusterRoleBindingToGSGroup(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.createWriteAllRoleBindingToCustomerGroup(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.createWriteAllRoleBindingToAutomationSA(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
