package core

import (
	"strings"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
)

func MaxInt(x, y int) int {
	if x < y {
		return y
	}
	return x
}

func GenGeneralName(jobName string, rtype commonv1.ReplicaType, index string) string {
	n := jobName + "-" + string(rtype) + "-" + index
	return strings.Replace(n, "/", "-", -1)
}
