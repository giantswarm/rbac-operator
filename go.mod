module github.com/giantswarm/rbac-operator

go 1.21

toolchain go1.22.6

require (
	github.com/giantswarm/exporterkit v1.2.0
	github.com/giantswarm/k8sclient/v7 v7.2.0
	github.com/giantswarm/k8smetadata v0.25.0
	github.com/giantswarm/microendpoint v1.1.0
	github.com/giantswarm/microerror v0.4.1
	github.com/giantswarm/microkit v1.0.1
	github.com/giantswarm/micrologger v1.1.1
	github.com/giantswarm/operatorkit/v8 v8.0.0
	github.com/giantswarm/organization-operator v1.6.3
	github.com/google/go-cmp v0.6.0
	github.com/prometheus/client_golang v1.19.1
	github.com/spf13/viper v1.18.2
	k8s.io/api v0.28.3
	k8s.io/apimachinery v0.28.3
	k8s.io/client-go v0.28.3
	k8s.io/utils v0.0.0-20240502163921-fe8a2dddb1d0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/emicklei/go-restful/v3 v3.11.0 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.6.0 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/getsentry/sentry-go v0.13.0 // indirect
	github.com/giantswarm/backoff v1.0.0 // indirect
	github.com/giantswarm/to v0.4.0 // indirect
	github.com/giantswarm/versionbundle v1.1.0 // indirect
	github.com/go-kit/kit v0.13.0 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.6.0 // indirect
	github.com/go-logr/logr v1.3.0 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/gnostic-models v0.6.8 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.4.0 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/imdario/mergo v0.3.13 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pelletier/go-toml/v2 v2.1.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.48.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/cobra v1.8.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/exp v0.0.0-20231108232855-2478ac86f678 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/oauth2 v0.16.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
	golang.org/x/term v0.21.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.4.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/resty.v1 v1.12.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiextensions-apiserver v0.28.3 // indirect
	k8s.io/component-base v0.28.3 // indirect
	k8s.io/klog/v2 v2.100.1 // indirect
	k8s.io/kube-openapi v0.0.0-20230717233707-2695361300d9 // indirect
	sigs.k8s.io/controller-runtime v0.16.3
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd //indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

replace (
	github.com/coreos/etcd => go.etcd.io/etcd/v3 v3.5.14
	github.com/dgrijalva/jwt-go => github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/gin-gonic/gin => github.com/gin-gonic/gin v1.10.0
	github.com/hashicorp/consul/api => github.com/hashicorp/consul/api v1.29.1
	github.com/hashicorp/consul/sdk => github.com/hashicorp/consul/sdk v0.16.0
	github.com/kataras/iris/v12 => github.com/kataras/iris/v12 v12.2.11
	github.com/labstack/echo/v4 => github.com/labstack/echo/v4 v4.12.0
	github.com/microcosm-cc/bluemonday => github.com/microcosm-cc/bluemonday v1.0.26
	github.com/nats-io/nats-server/v2 => github.com/nats-io/nats-server/v2 v2.10.17
	github.com/nats-io/nkeys => github.com/nats-io/nkeys v0.4.7
	github.com/valyala/fasthttp => github.com/valyala/fasthttp v1.54.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc => go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.53.0
	golang.org/x/text => golang.org/x/text v0.16.0
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.15.3
)

replace golang.org/x/net => golang.org/x/net v0.26.0
