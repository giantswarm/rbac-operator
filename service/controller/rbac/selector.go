package rbac

import (
	"k8s.io/apimachinery/pkg/labels"
)

/*
	For some reason selector package was moved under internal package in
	operatorkit@v4.3.1 and is not accessible outside. This is an adapted
	copy/paste from the operatorkit package.
*/

type selectorImpl struct {
	matchesFn func(labels labels.Labels) bool
}

func newSelector(matchesFn func(labels labels.Labels) bool) *selectorImpl {
	return &selectorImpl{
		matchesFn: matchesFn,
	}
}

func (s *selectorImpl) Matches(labels labels.Labels) bool {
	return s.matchesFn(labels)
}
