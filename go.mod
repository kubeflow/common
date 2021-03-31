module github.com/kubeflow/common

go 1.13

require (
	github.com/go-openapi/spec v0.19.2
	github.com/golang/groupcache v0.0.0-20190129154638-5b532d6fd5ef // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/prometheus/client_golang v1.5.1
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.4.0
	k8s.io/api v0.16.15
	k8s.io/apimachinery v0.16.15
	k8s.io/client-go v0.16.15
	k8s.io/code-generator v0.16.15
	k8s.io/kube-openapi v0.0.0-20200410163147-594e756bea31
	volcano.sh/apis v1.2.0-k8s1.16.15
)

replace (
	k8s.io/api => k8s.io/api v0.16.9
	k8s.io/apimachinery => k8s.io/apimachinery v0.16.10-beta.0
	k8s.io/client-go => k8s.io/client-go v0.16.9
	k8s.io/code-generator => k8s.io/code-generator v0.16.10-beta.0
)
