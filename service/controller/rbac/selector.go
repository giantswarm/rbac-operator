package rbac

import (
	"k8s.io/apimachinery/pkg/labels"
)

// wrongSelector is named like that because it only partially implements
// labels.Selector interface and relies on the fact that operatorkit/v5 calls
// only `Matches` method. This feels wrong. Hence the name. If you want to fix
// it then you are welcome to do so. If the underlying implementation changes
// this will result in a panic. This is to make it work with operatorkit/v5.
// Kubernetes in general doesn't support label selector with OR condition and
// we need that here.
type wrongSelector struct {
	labels.Selector
	matchesFn func(labels.Labels) bool
}

func newWrongSelector(matchesFn func(labels.Labels) bool) wrongSelector {
	return wrongSelector{
		matchesFn: matchesFn,
	}
}

func (s wrongSelector) Matches(labels labels.Labels) bool {
	return s.matchesFn(labels)
}
