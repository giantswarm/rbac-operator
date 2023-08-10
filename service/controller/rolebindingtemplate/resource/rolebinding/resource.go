// rolebinding package is repsonsible for creating rolebindings based on roleBindingTemplate CRs
// this allows for the dynamic granting of permissions inside all or specific organizations
package rolebinding

import (
	"context"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	security "github.com/giantswarm/organization-operator/api/v1alpha1"
	"github.com/giantswarm/rbac-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

const (
	Name = "rolebinding"
)

type Config struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
}

type Resource struct {
	k8sClient k8sclient.Interface
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
		k8sClient: config.K8sClient,
		logger:    config.Logger,
	}

	return r, nil
}

func (r Resource) K8sClient() kubernetes.Interface {
	return r.k8sClient.K8sClient()
}

func (r Resource) Logger() micrologger.Logger {
	return r.logger
}

func (r *Resource) Name() string {
	return Name
}

func (r *Resource) getNamespacesFromScope(ctx context.Context, scopes v1alpha1.RoleBindingTemplateScopes) ([]string, error) {
	organizations := &security.OrganizationList{}
	{
		labelSelector := labels.SelectorFromSet(scopes.OrganizationSelector.MatchLabels)
		if err := r.k8sClient.CtrlClient().List(ctx, organizations, &client.ListOptions{LabelSelector: labelSelector}); err != nil {
			if apierrors.IsNotFound(err) {
				r.logger.Debugf(ctx, "No namespaces in organization scope %s", labelSelector.String())
				return []string{}, nil
			}
			return nil, microerror.Mask(err)
		}
		if len(organizations.Items) == 0 {
			r.logger.Debugf(ctx, "No namespaces in organization scope %s", labelSelector.String())
			return []string{}, nil
		}
	}
	namespaces := []string{}
	{
		for _, o := range organizations.Items {
			namespaces = append(namespaces, o.Namespace)
		}
		// TODO: org-namespaces
	}
	return namespaces, nil
}
