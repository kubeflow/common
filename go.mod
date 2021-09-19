module github.com/kubeflow/common

go 1.16

require (
	github.com/go-logr/logr v0.3.0
	github.com/go-openapi/spec v0.20.3
	github.com/prometheus/client_golang v1.10.0
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.6.1
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4 // indirect
	golang.org/x/sys v0.0.0-20210510120138-977fb7262007 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	k8s.io/api v0.19.9
	k8s.io/apimachinery v0.19.9
	k8s.io/client-go v0.19.9
	k8s.io/code-generator v0.19.9
	k8s.io/kube-openapi v0.0.0-20200805222855-6aeccd4b50c6
	sigs.k8s.io/controller-runtime v0.7.2
	volcano.sh/apis v1.2.0-k8s1.19.6
)

replace (
	k8s.io/api => k8s.io/api v0.19.9
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.9
	k8s.io/client-go => k8s.io/client-go v0.19.9
	k8s.io/code-generator => k8s.io/code-generator v0.19.9
)
