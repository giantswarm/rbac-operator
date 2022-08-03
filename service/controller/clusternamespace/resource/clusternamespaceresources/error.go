package clusternamespaceresources

import (
	"github.com/giantswarm/microerror"
)

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var unknownOrganizationNamespaceError = &microerror.Error{
	Kind: "unknownOrganizationNamespaceError",
}

// IsUnknownOrganizationNamespace asserts unknownOrganizationNamespaceError.
func IsUnknownOrganizationNamespace(err error) bool {
	return microerror.Cause(err) == unknownOrganizationNamespaceError
}

// IsUnknownOrganization asserts unknownOrganizationError.
func IsUnknownOrganization(err error) bool {
	return microerror.Cause(err) == unknownOrganizationError
}

var unknownOrganizationError = &microerror.Error{
	Kind: "unknownOrganizationError",
}
