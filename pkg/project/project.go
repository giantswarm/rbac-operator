package project

var (
	bundleVersion = "0.3.0"
	description   = "The rbac-operator manages tenant namespace access in control-plane Kubernetes."
	gitSHA        = "n/a"
	name          = "rbac-operator"
	source        = "https://github.com/giantswarm/rbac-operator"
	version       = "0.3.1"
)

func Description() string {
	return description
}

func GitSHA() string {
	return gitSHA
}

func Name() string {
	return name
}

func Source() string {
	return source
}

func Version() string {
	return version
}
