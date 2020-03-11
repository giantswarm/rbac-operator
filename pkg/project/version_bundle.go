package project

import (
	"github.com/giantswarm/versionbundle"
)

func NewVersionBundle() versionbundle.Bundle {
	return versionbundle.Bundle{
		Changelogs: []versionbundle.Changelog{
			{
				Component:   "rbac-operator",
				Description: "Add view-all resources role for customer identity provider's group.",
				Kind:        versionbundle.KindAdded,
			},
			{
				Component:   "rbac-operator",
				Description: "Add tenant-admin resources role for customer identity provider's group.",
				Kind:        versionbundle.KindChanged,
			},
		},
		Components: []versionbundle.Component{},
		Name:       "rbac-operator",
		Version:    BundleVersion(),
	}
}
