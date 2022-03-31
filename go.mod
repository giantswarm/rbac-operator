module github.com/giantswarm/rbac-operator

go 1.16

require (
	github.com/giantswarm/exporterkit v1.0.0
	github.com/giantswarm/k8sclient/v7 v7.0.1
	github.com/giantswarm/k8smetadata v0.10.1
	github.com/giantswarm/microendpoint v1.0.0
	github.com/giantswarm/microerror v0.4.0
	github.com/giantswarm/microkit v1.0.0
	github.com/giantswarm/micrologger v0.6.0
	github.com/giantswarm/operatorkit/v7 v7.0.1
	github.com/giantswarm/organization-operator v1.0.0
	github.com/prometheus/client_golang v1.12.1
	github.com/spf13/viper v1.10.1
	k8s.io/api v0.20.15
	k8s.io/apimachinery v0.20.15
	k8s.io/client-go v0.20.15
	sigs.k8s.io/controller-runtime v0.8.3
)

replace (
	github.com/coreos/etcd => go.etcd.io/etcd/v3 v3.5.1
	github.com/dgrijalva/jwt-go => github.com/dgrijalva/jwt-go/v4 v4.0.0-preview1
)
