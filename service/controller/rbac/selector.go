package rbac

import (
	"k8s.io/apimachinery/pkg/labels"
)

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
