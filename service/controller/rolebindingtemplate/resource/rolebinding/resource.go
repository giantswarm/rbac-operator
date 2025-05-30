// rolebinding package is repsonsible for creating rolebindings based on roleBindingTemplate CRs
// this allows for the dynamic granting of permissions inside all or specific organizations
package rolebinding

import (
	"context"
	"fmt"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	security "github.com/giantswarm/organization-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/rbac-operator/api/v1alpha1"
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

func getLabelSelectorFromScopes(scopes v1alpha1.RoleBindingTemplateScopes) (labels.Selector, error) {
	return metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels:      scopes.OrganizationSelector.MatchLabels,
		MatchExpressions: scopes.OrganizationSelector.MatchExpressions,
	})
}

func (r *Resource) getOrganizationsForLabelSelector(ctx context.Context, labelSelector labels.Selector) (*security.OrganizationList, error) {
	organizations := &security.OrganizationList{}

	if err := r.k8sClient.CtrlClient().List(ctx, organizations, &client.ListOptions{LabelSelector: labelSelector}); err != nil {
		if apierrors.IsNotFound(err) {
			r.logger.Debugf(ctx, "No organizations in organization scope %s", labelSelector.String())
			return organizations, nil
		}
		return nil, microerror.Mask(err)
	}
	if len(organizations.Items) == 0 {
		r.logger.Debugf(ctx, "No organizations in organization scope %s", labelSelector.String())
	}
	return organizations, nil
}

func (r *Resource) getNamespacesFromOrganizations(ctx context.Context, organizations *security.OrganizationList) ([]string, error) {
	namespaces := []string{}

	for _, o := range organizations.Items {
		// get the org namespace
		namespaces = append(namespaces, o.Status.Namespace)

		// get the cluster namespaces that belong to the org namespace
		labelSelector, err := labels.Parse(fmt.Sprintf("%s=%s,%s", label.Organization, o.Name, label.Cluster))
		if err != nil {
			return nil, microerror.Mask(err)
		}
		clusterNamespaces, err := r.k8sClient.K8sClient().CoreV1().Namespaces().List(ctx, metav1.ListOptions{LabelSelector: labelSelector.String()})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return nil, microerror.Mask(err)
			}
		}
		for _, cns := range clusterNamespaces.Items {
			namespaces = append(namespaces, cns.Name)
		}
	}
	return namespaces, nil
}

func (r *Resource) getNamespacesFromScope(ctx context.Context, scopes v1alpha1.RoleBindingTemplateScopes) ([]string, error) {
	labelSelector, err := getLabelSelectorFromScopes(scopes)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	organizations, err := r.getOrganizationsForLabelSelector(ctx, labelSelector)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	namespaces, err := r.getNamespacesFromOrganizations(ctx, organizations)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	scope := []string{}
	for _, ns := range namespaces {
		namespace, err := r.k8sClient.K8sClient().CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
		if err != nil {
			continue
		} else if namespace.DeletionTimestamp != nil {
			continue
		}
		scope = append(scope, ns)
	}

	return scope, nil
}

func getRoleBindingNameFromTemplate(template v1alpha1.RoleBindingTemplate) string {
	roleBindingName := template.Spec.Template.Name
	if roleBindingName == "" {
		roleBindingName = template.Name
	}
	return roleBindingName
}
