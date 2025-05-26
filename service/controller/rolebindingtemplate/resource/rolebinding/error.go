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

var roleBindingDeletionFailedError = &microerror.Error{
	Kind: "roleBindingDeletionFailedError",
}

// IsRoleBindingDeletionFailed asserts roleBindingDeletionFailedError.
func IsRoleBindingDeletionFailed(err error) bool {
	return microerror.Cause(err) == roleBindingDeletionFailedError
}

var namespaceUpdateFailedError = &microerror.Error{
	Kind: "namespaceUpdateFailedError",
}

// IsNamespaceUpdateFailed asserts namespaceUpdateFailedError.
func IsNamespaceUpdateFailed(err error) bool {
	return microerror.Cause(err) == namespaceUpdateFailedError
}

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}
