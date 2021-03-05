package bootstrap

import (
	"github.com/giantswarm/k8sclient/v4/pkg/k8sclient"
	"github.com/giantswarm/micrologger"
	"k8s.io/client-go/kubernetes"
)

/*
Create during boot:

1. ClusterRole read-all -> introduce code for this - create during boot
2. Rolebinding to cluster-admin in default namespace
3. Rolebinding to cluster-admin in global namespace (create global namespace if doesn't exist)
4. ClusterRole list clusterroles
5. ClusterRole list organizations
6. ClusterRoleBinding to clusterrole 4
7. ClusterRoleBinding to clusterrole 5


------------------------

out of scope of this package:

controller updates:

Per namespace:
Rolebinding for read-all
Rolebinding for cluster-admin
*/

type Config struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
}

type Bootstrap struct {
	k8sClient kubernetes.Interface
	logger    micrologger.Logger
}

func (b *Bootstrap) Do() error {
	return nil
}
