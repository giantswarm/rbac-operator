module github.com/giantswarm/rbac-operator

go 1.13

require (
	github.com/giantswarm/apiextensions/v3 v3.22.0
	github.com/giantswarm/exporterkit v0.2.1
	github.com/giantswarm/k8sclient/v4 v4.1.0
	github.com/giantswarm/microendpoint v0.2.0
	github.com/giantswarm/microerror v0.3.0
	github.com/giantswarm/microkit v0.2.2
	github.com/giantswarm/micrologger v0.5.0
	github.com/giantswarm/operatorkit/v2 v2.0.2
	github.com/prometheus/client_golang v1.10.0
	github.com/spf13/viper v1.7.1
	k8s.io/api v0.18.18
	k8s.io/apimachinery v0.18.18
	k8s.io/client-go v0.18.18
)

replace sigs.k8s.io/cluster-api => github.com/giantswarm/cluster-api v0.3.13-gs
