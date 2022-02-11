package key

import (
	"github.com/giantswarm/microerror"
	cluster "sigs.k8s.io/cluster-api/api/v1alpha3"
)

func ToCluster(v interface{}) (cluster.Cluster, error) {
	if v == nil {
		return cluster.Cluster{}, microerror.Maskf(wrongTypeError, "expected non-nil, got %#v'", v)
	}

	p, ok := v.(*cluster.Cluster)
	if !ok {
		return cluster.Cluster{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", p, v)
	}

	c := p.DeepCopy()

	return *c, nil
}
