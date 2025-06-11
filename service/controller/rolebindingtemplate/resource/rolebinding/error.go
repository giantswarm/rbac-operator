package rolebinding

import (
	"github.com/giantswarm/microerror"
)

var roleBindingCreationFailedError = &microerror.Error{
	Kind: "roleBindingCreationFailedError",
}

// IsRoleBindingCreationFailed asserts roleBindingCreationFailedError.
func IsRoleBindingCreationFailed(err error) bool {
	return microerror.Cause(err) == roleBindingCreationFailedError
}

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}
