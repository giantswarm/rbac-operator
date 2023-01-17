// Package service implements business logic to create Kubernetes resources
// against the Kubernetes API.
package service

import (
	"context"
	"sync"

	"github.com/giantswarm/rbac-operator/service/internal/accessgroup"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/k8sclient/v7/pkg/k8srestconfig"
	"github.com/giantswarm/microendpoint/service/version"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	security "github.com/giantswarm/organization-operator/api/v1alpha1"
	"github.com/spf13/viper"
	"k8s.io/client-go/rest"

	"github.com/giantswarm/rbac-operator/flag"
	"github.com/giantswarm/rbac-operator/pkg/project"
	"github.com/giantswarm/rbac-operator/service/collector"
	"github.com/giantswarm/rbac-operator/service/controller/clusternamespace"
	"github.com/giantswarm/rbac-operator/service/controller/crossplane"
	"github.com/giantswarm/rbac-operator/service/controller/rbac"
	"github.com/giantswarm/rbac-operator/service/internal/bootstrap"
)

// Config represents the configuration used to create a new service.
type Config struct {
	Logger micrologger.Logger

	Flag  *flag.Flag
	Viper *viper.Viper
}

type Service struct {
	Version *version.Service

	bootOnce                   sync.Once
	bootstrapRunner            *bootstrap.Bootstrap
	rbacController             *rbac.RBAC
	clusterNamespaceController *clusternamespace.ClusterNamespace
	crossplaneController       *crossplane.Crossplane
	operatorCollector          *collector.Set
}

// New creates a new configured service object.
func New(config Config) (*Service, error) {
	var serviceAddress string
	// Settings.
	if config.Flag == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Flag must not be empty")
	}
	if config.Viper == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Viper must not be empty")
	}

	kubeConfig := config.Viper.GetString(config.Flag.Service.Kubernetes.KubeConfig)
	if kubeConfig == "" {
		serviceAddress = config.Viper.GetString(config.Flag.Service.Kubernetes.Address)
	} else {
		serviceAddress = ""
	}

	// Dependencies.
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "logger must not be empty")
	}

	var err error

	var restConfig *rest.Config
	{
		c := k8srestconfig.Config{
			Logger: config.Logger,

			Address:    serviceAddress,
			InCluster:  config.Viper.GetBool(config.Flag.Service.Kubernetes.InCluster),
			KubeConfig: kubeConfig,
			TLS: k8srestconfig.ConfigTLS{
				CAFile:  config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CAFile),
				CrtFile: config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CrtFile),
				KeyFile: config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.KeyFile),
			},
		}

		restConfig, err = k8srestconfig.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var k8sClient k8sclient.Interface
	{
		c := k8sclient.ClientsConfig{
			Logger: config.Logger,
			SchemeBuilder: k8sclient.SchemeBuilder{
				security.AddToScheme,
			},
			RestConfig: restConfig,
		}

		k8sClient, err = k8sclient.NewClients(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var accessGroups accessgroup.AccessGroups
	{
		err = config.Viper.UnmarshalKey(config.Flag.Service.AccessGroups, &accessGroups)
		if err != nil {
			// TODO: Log error
			accessGroups = accessgroup.AccessGroups{}
		}

		legacyCustomerAdminGroup := config.Viper.GetString(config.Flag.Service.WriteAllCustomerGroup)
		accessGroups.AddLegacyCustomerAdminGroup(legacyCustomerAdminGroup)

		legacyGiantswarmAdminGroup := config.Viper.GetString(config.Flag.Service.WriteAllGiantswarmGroup)
		accessGroups.AddLegacyGiantswarmAdminGroup(legacyGiantswarmAdminGroup)

		if !accessGroups.HasValidWriteAllGiantswarmAdminGroups() {
			return nil, microerror.Maskf(invalidConfigError, "Giantswarm Write All Admin groups must not be empty")
		}
	}

	var bootstrapRunner *bootstrap.Bootstrap
	{
		c := bootstrap.Config{
			Logger:    config.Logger,
			K8sClient: k8sClient,

			CustomerAdminGroups: accessGroups.WriteAllCustomerGroups,
			GSAdminGroups:       accessGroups.WriteAllGiantswarmGroups,
		}

		bootstrapRunner, err = bootstrap.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var clusterNamespaceController *clusternamespace.ClusterNamespace
	{

		c := clusternamespace.ClusterNamespaceConfig{
			K8sClient: k8sClient,
			Logger:    config.Logger,
		}

		clusterNamespaceController, err = clusternamespace.NewClusterNamespace(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var rbacController *rbac.RBAC
	{

		c := rbac.RBACConfig{
			K8sClient: k8sClient,
			Logger:    config.Logger,

			WriteAllCustomerGroups: accessGroups.WriteAllCustomerGroups,
		}

		rbacController, err = rbac.NewRBAC(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var crossplaneController *crossplane.Crossplane
	{
		c := crossplane.CrossplaneConfig{
			K8sClient: k8sClient,
			Logger:    config.Logger,

			CustomerAdminGroups:                 accessGroups.WriteAllCustomerGroups,
			CrossplaneBindTriggeringClusterRole: config.Viper.GetString(config.Flag.Service.CrossplaneBindTriggeringClusterRoleName),
		}

		crossplaneController, err = crossplane.NewCrossplane(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var operatorCollector *collector.Set
	{
		c := collector.SetConfig{
			K8sClient: k8sClient.K8sClient(),
			Logger:    config.Logger,
		}

		operatorCollector, err = collector.NewSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var versionService *version.Service
	{
		c := version.Config{
			Description: project.Description(),
			GitCommit:   project.GitSHA(),
			Name:        project.Name(),
			Source:      project.Source(),
			Version:     project.Version(),
		}

		versionService, err = version.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	s := &Service{
		Version: versionService,

		bootOnce:                   sync.Once{},
		bootstrapRunner:            bootstrapRunner,
		rbacController:             rbacController,
		clusterNamespaceController: clusterNamespaceController,
		operatorCollector:          operatorCollector,
		crossplaneController:       crossplaneController,
	}

	return s, nil
}

func (s *Service) Boot(ctx context.Context) {
	s.bootOnce.Do(func() {

		err := s.bootstrapRunner.Run(ctx)
		if err != nil {
			panic(microerror.JSON(microerror.Mask(err)))
		}

		go func() {
			err := s.operatorCollector.Boot(ctx)
			if err != nil {
				panic(microerror.JSON(microerror.Mask(err)))
			}
		}()

		go s.rbacController.Boot(ctx)

		go s.clusterNamespaceController.Boot(ctx)

		go s.crossplaneController.Boot(ctx)
	})
}
