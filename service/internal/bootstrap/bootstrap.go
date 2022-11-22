package bootstrap

import (
	"context"

	"github.com/giantswarm/rbac-operator/service/internal/accessgroup"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/client-go/kubernetes"
)

type Config struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	// internal
	CustomerAdminGroups []accessgroup.AccessGroup
	GSAdminGroups       []accessgroup.AccessGroup
}

type Bootstrap struct {
	k8sClient kubernetes.Interface
	logger    micrologger.Logger

	// internal
	customerAdminGroups []accessgroup.AccessGroup
	gsAdminGroups       []accessgroup.AccessGroup
}

func (b Bootstrap) K8sClient() kubernetes.Interface {
	return b.k8sClient
}

func (b Bootstrap) Logger() micrologger.Logger {
	return b.logger
}

func New(config Config) (*Bootstrap, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if len(config.GSAdminGroups) == 0 {
		return nil, microerror.Maskf(invalidConfigError, "%T.GSAdminGroup must not be empty", config)
	}

	r := &Bootstrap{
		k8sClient: config.K8sClient.K8sClient(),
		logger:    config.Logger,

		customerAdminGroups: config.CustomerAdminGroups,
		gsAdminGroups:       config.GSAdminGroups,
	}

	return r, nil
}

func (b *Bootstrap) Run(ctx context.Context) error {
	var err error

	err = b.createAutomationServiceAccount(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.createReadAllClusterRole(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.createReadClusterNamespaceAppsRole(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.createWriteClusterNamespaceAppsRole(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.createReadAllClusterRoleBindingToAutomationSA(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.createReadReleasesClusterRole(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.createReadDefaultCatalogsRole(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.createWriteAllClusterRoleBindingToGSGroup(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.createWriteAllRoleBindingToAutomationSA(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.createWriteOrganizationsClusterRole(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.createWriteOrganizationsClusterRoleBindingToAutomationSA(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.createWriteFluxResourcesClusterRole(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.createWriteClustersClusterRole(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.createWriteClustersClusterRoleBindingToAutomationSA(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.createWriteNodePoolsClusterRole(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.createWriteNodePoolsClusterRoleBindingToAutomationSA(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.createWriteClientCertsClusterRole(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.createWriteClientCertsClusterRoleBindingToAutomationSA(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.createWriteSilencesClusterRole(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.createWriteSilencesClusterRoleBindingToAutomationSA(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.labelDefaultClusterRoles(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	if len(b.customerAdminGroups) > 0 {
		err = b.createReadAllClusterRoleBindingToCustomerGroup(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		err = b.createWriteAllRoleBindingToCustomerGroup(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		err = b.createWriteOrganizationsClusterRoleBindingToCustomerGroup(ctx)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
