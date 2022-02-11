// Package service implements business logic to create Kubernetes resources
// against the Kubernetes API.
package service

import (
	"context"
	"sync"

	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/k8sclient/v5/pkg/k8srestconfig"
	"github.com/giantswarm/microendpoint/service/version"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/viper"
	"k8s.io/client-go/rest"
	cluster "sigs.k8s.io/cluster-api/api/v1alpha3"

	"github.com/giantswarm/rbac-operator/flag"
	"github.com/giantswarm/rbac-operator/pkg/project"
	"github.com/giantswarm/rbac-operator/service/collector"
	"github.com/giantswarm/rbac-operator/service/controller/orgcluster"
	"github.com/giantswarm/rbac-operator/service/controller/orgpermissions"
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

	bootOnce                 sync.Once
	bootstrapRunner          *bootstrap.Bootstrap
	rbacController           *rbac.RBAC
	orgPermissionsController *orgpermissions.OrgPermissions
	orgClusterController     *orgcluster.OrgCluster
	operatorCollector        *collector.Set
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
	if config.Flag.Service.Kubernetes.KubeConfig == "" {
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
			KubeConfig: config.Viper.GetString(config.Flag.Service.Kubernetes.KubeConfig),
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
				cluster.AddToScheme,
			},
			RestConfig: restConfig,
		}

		k8sClient, err = k8sclient.NewClients(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var bootstrapRunner *bootstrap.Bootstrap
	{
		c := bootstrap.Config{
			Logger:    config.Logger,
			K8sClient: k8sClient,

			CustomerAdminGroup: config.Viper.GetString(config.Flag.Service.WriteAllCustomerGroup),
			GSAdminGroup:       config.Viper.GetString(config.Flag.Service.WriteAllGiantswarmGroup),
		}

		bootstrapRunner, err = bootstrap.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var orgClusterController *orgcluster.OrgCluster
	{

		c := orgcluster.OrgClusterConfig{
			K8sClient: k8sClient,
			Logger:    config.Logger,
		}

		orgClusterController, err = orgcluster.NewOrgCluster(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var rbacController *rbac.RBAC
	{

		c := rbac.RBACConfig{
			K8sClient: k8sClient,
			Logger:    config.Logger,

			WriteAllCustomerGroup: config.Viper.GetString(config.Flag.Service.WriteAllCustomerGroup),
		}

		rbacController, err = rbac.NewRBAC(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var orgPermissionsController *orgpermissions.OrgPermissions
	{

		c := orgpermissions.OrgPermissionsConfig{
			K8sClient: k8sClient,
			Logger:    config.Logger,
		}

		orgPermissionsController, err = orgpermissions.NewOrgPermissions(c)
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

		bootOnce:                 sync.Once{},
		bootstrapRunner:          bootstrapRunner,
		rbacController:           rbacController,
		orgPermissionsController: orgPermissionsController,
		orgClusterController:     orgClusterController,
		operatorCollector:        operatorCollector,
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

		go s.orgPermissionsController.Boot(ctx)
	})
}
