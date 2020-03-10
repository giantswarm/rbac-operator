package namespaceauth

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

var wrongTypeError = &microerror.Error{
	Kind: "wrongTypeError",
}

func IsWrongType(err error) bool {
	return microerror.Cause(err) == wrongTypeError
}
