module github.com/dailymotion-oss/osiris

go 1.15

require (
	github.com/go-logr/logr v0.2.1 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/googleapis/gnostic v0.5.1 // indirect
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.10.0
	github.com/satori/go.uuid v1.2.0
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/contrib/detectors/gcp v0.16.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.16.0
	go.opentelemetry.io/otel v0.16.0
	go.opentelemetry.io/otel/exporters/otlp v0.16.0
	go.opentelemetry.io/otel/sdk v0.16.0
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	k8s.io/api v0.19.1
	k8s.io/apimachinery v0.19.1
	k8s.io/client-go v1.5.1
	k8s.io/klog/v2 v2.3.0 // indirect
	k8s.io/kubernetes v1.19.1
	k8s.io/utils v0.0.0-20200821003339-5e75c0163111 // indirect
)

replace (
	k8s.io/api => k8s.io/api v0.19.1
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.1
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.1
	k8s.io/apiserver => k8s.io/apiserver v0.19.1
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.19.1
	k8s.io/client-go => k8s.io/client-go v0.19.1
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.19.1
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.19.1
	k8s.io/code-generator => k8s.io/code-generator v0.19.1
	k8s.io/component-base => k8s.io/component-base v0.19.1
	k8s.io/cri-api => k8s.io/cri-api v0.19.1
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.19.1
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.19.1
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.19.1
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.19.1
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.19.1
	k8s.io/kubectl => k8s.io/kubectl v0.19.1
	k8s.io/kubelet => k8s.io/kubelet v0.19.1
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.19.1
	k8s.io/metrics => k8s.io/metrics v0.19.1
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.19.1
)
