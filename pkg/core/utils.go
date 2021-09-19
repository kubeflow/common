package core

import (
	"strings"
)

func MaxInt(x, y int) int {
	if x < y {
		return y
	}
	return x
}

func GenGeneralName(jobName string, rtype string, index string) string {
	n := jobName + "-" + strings.ToLower(rtype) + "-" + index
	return strings.Replace(n, "/", "-", -1)
}
