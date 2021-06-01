package expectation

import (
	apiv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"strings"
)

// GenExpectationPodsKey generates an expectation key for pods of a job
func GenExpectationPodsKey(jobKey string, replicaType apiv1.ReplicaType) string {
	return jobKey + "/" + strings.ToLower(string(replicaType)) + "/pods"
}

// GenExpectationPodsKey generates an expectation key for services of a job
func GenExpectationServicesKey(jobKey string, replicaType apiv1.ReplicaType) string {
	return jobKey + "/" + strings.ToLower(string(replicaType)) + "/services"
}
