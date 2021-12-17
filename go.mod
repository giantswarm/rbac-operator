module github.com/giantswarm/rbac-operator

go 1.16

require (
	github.com/giantswarm/apiextensions/v3 v3.39.0
	github.com/giantswarm/exporterkit v0.2.1
	github.com/giantswarm/k8sclient/v5 v5.12.0
	github.com/giantswarm/k8smetadata v0.7.1
	github.com/giantswarm/microendpoint v1.0.0
	github.com/giantswarm/microerror v0.4.0
	github.com/giantswarm/microkit v1.0.0
	github.com/giantswarm/micrologger v0.6.0
	github.com/giantswarm/operatorkit/v4 v4.3.1
	github.com/prometheus/client_golang v1.11.0
	github.com/spf13/viper v1.10.1
	k8s.io/api v0.18.19
	k8s.io/apimachinery v0.18.19
	k8s.io/client-go v0.18.19
)

replace (
	github.com/coreos/etcd => go.etcd.io/etcd/v3 v3.5.1
	github.com/dgrijalva/jwt-go => github.com/dgrijalva/jwt-go/v4 v4.0.0-preview1
	sigs.k8s.io/cluster-api => github.com/giantswarm/cluster-api v0.3.13-gs
)
