module github.com/giantswarm/template-operator

go 1.13

require (
	github.com/giantswarm/apiextensions v0.0.0-20200220082851-d6884ee11480 // indirect
	github.com/giantswarm/backoff v0.0.0-20200209120535-b7cb1852522d // indirect
	github.com/giantswarm/exporterkit v0.0.0-20190619131829-9749deade60f
	github.com/giantswarm/k8sclient v0.0.0-20191209120459-6cb127468cd6
	github.com/giantswarm/microendpoint v0.0.0-20200205204116-c2c5b3af4bdb
	github.com/giantswarm/microerror v0.1.1-0.20200205143715-01b76f66cae6
	github.com/giantswarm/microkit v0.0.0-20191023091504-429e22e73d3e
	github.com/giantswarm/micrologger v0.0.0-20200205144836-079154bcae45
	github.com/giantswarm/operatorkit v0.0.0-20200205163802-6b6e6b2c208b
	github.com/giantswarm/versionbundle v0.0.0-20200205145509-6772c2bc7b34
	github.com/gorilla/mux v1.7.4 // indirect
	github.com/prometheus/client_golang v1.0.0
	github.com/spf13/cobra v0.0.6 // indirect
	github.com/spf13/viper v1.6.1
	k8s.io/api v0.16.6
	k8s.io/apimachinery v0.16.6
	k8s.io/client-go v0.16.6
)

replace (
	k8s.io/api v0.0.0 => k8s.io/api v0.16.6
	k8s.io/apiextensions-apiserver v0.0.0 => k8s.io/apiextensions-apiserver v0.16.6
	k8s.io/apimachinery v0.0.0 => k8s.io/apimachinery v0.16.6
	k8s.io/apiserver v0.0.0 => k8s.io/apiserver v0.16.6
	k8s.io/cli-runtime v0.0.0 => k8s.io/cli-runtime v0.16.6
	k8s.io/client-go v0.0.0 => k8s.io/client-go v0.16.6
	k8s.io/cloud-provider v0.0.0 => k8s.io/cloud-provider v0.16.6
	k8s.io/cluster-bootstrap v0.0.0 => k8s.io/cluster-bootstrap v0.16.6
	k8s.io/code-generator v0.0.0 => k8s.io/code-generator v0.16.6
	k8s.io/component-base v0.0.0 => k8s.io/component-base v0.16.6
	k8s.io/cri-api v0.0.0 => k8s.io/cri-api v0.16.6
	k8s.io/csi-translation-lib v0.0.0 => k8s.io/csi-translation-lib v0.16.6
	k8s.io/kube-aggregator v0.0.0 => k8s.io/kube-aggregator v0.16.6
	k8s.io/kube-controller-manager v0.0.0 => k8s.io/kube-controller-manager v0.16.6
	k8s.io/kube-proxy v0.0.0 => k8s.io/kube-proxy v0.16.6
	k8s.io/kube-scheduler v0.0.0 => k8s.io/kube-scheduler v0.16.6
	k8s.io/kubectl v0.0.0 => k8s.io/kubectl v0.16.6
	k8s.io/kubelet v0.0.0 => k8s.io/kubelet v0.16.6
	k8s.io/legacy-cloud-providers v0.0.0 => k8s.io/legacy-cloud-providers v0.16.6
	k8s.io/metrics v0.0.0 => k8s.io/metrics v0.16.6
	k8s.io/sample-apiserver v0.0.0 => k8s.io/sample-apiserver v0.16.6
	k8s.io/utils v0.0.0 => k8s.io/utils v0.0.0-20191114200735-6ca3b61696b6
)
