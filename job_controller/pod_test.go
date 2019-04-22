package job_controller

import (
	"testing"

	common "github.com/kubeflow/common/operator/v1"
	testjobv1 "github.com/kubeflow/common/test_job/v1"
	testutilv1 "github.com/kubeflow/common/test_util/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
)

func TestSetRestartPolicy(t *testing.T) {
	type tc struct {
		testJob               *testjobv1.TestJob
		expectedRestartPolicy v1.RestartPolicy
		expectedType          testjobv1.TestReplicaType
	}
	testCase := []tc{
		func() tc {
			tj := testutilv1.NewTestJob(2)
			tj.Spec.TestReplicaSpecs[testjobv1.TestReplicaTypeWorker].RestartPolicy = common.RestartPolicyExitCode
			return tc{
				testJob:               tj,
				expectedRestartPolicy: v1.RestartPolicyNever,
				expectedType:          testjobv1.TestReplicaTypeWorker,
			}
		}(),
		func() tc {
			tj := testutilv1.NewTestJob(2)
			tj.Spec.TestReplicaSpecs[testjobv1.TestReplicaTypeWorker].RestartPolicy = common.RestartPolicyNever
			return tc{
				testJob:               tj,
				expectedRestartPolicy: v1.RestartPolicyNever,
				expectedType:          testjobv1.TestReplicaTypeWorker,
			}
		}(),
		func() tc {
			tj := testutilv1.NewTestJob(2)
			tj.Spec.TestReplicaSpecs[testjobv1.TestReplicaTypeWorker].RestartPolicy = common.RestartPolicyAlways
			return tc{
				testJob:               tj,
				expectedRestartPolicy: v1.RestartPolicyAlways,
				expectedType:          testjobv1.TestReplicaTypeWorker,
			}
		}(),
		func() tc {
			tj := testutilv1.NewTestJob(2)
			tj.Spec.TestReplicaSpecs[testjobv1.TestReplicaTypeWorker].RestartPolicy = common.RestartPolicyOnFailure
			return tc{
				testJob:               tj,
				expectedRestartPolicy: v1.RestartPolicyOnFailure,
				expectedType:          testjobv1.TestReplicaTypeWorker,
			}
		}(),
	}
	for _, c := range testCase {
		spec := c.testJob.Spec.TestReplicaSpecs[c.expectedType]
		podTemplate := spec.Template
		setRestartPolicy(&podTemplate, spec)
		if podTemplate.Spec.RestartPolicy != c.expectedRestartPolicy {
			t.Errorf("Expected %s, got %s", c.expectedRestartPolicy, podTemplate.Spec.RestartPolicy)
		}
	}
}

func TestIsNonGangSchedulerSet(t *testing.T) {
	replicaSpecs := map[common.ReplicaType]*common.ReplicaSpec{}
	assert.False(t, isNonGangSchedulerSet(replicaSpecs))

	replicaSpecs[common.ReplicaType(testjobv1.TestReplicaTypeWorker)] = &common.ReplicaSpec{
		Template: v1.PodTemplateSpec{
			Spec: v1.PodSpec{
				SchedulerName: gangSchedulerName,
			},
		},
	}
	assert.False(t, isNonGangSchedulerSet(replicaSpecs))

	replicaSpecs[common.ReplicaType(testjobv1.TestReplicaTypeWorker)] = &common.ReplicaSpec{
		Template: v1.PodTemplateSpec{
			Spec: v1.PodSpec{
				SchedulerName: "other-scheduler",
			},
		},
	}
	assert.True(t, isNonGangSchedulerSet(replicaSpecs))
}
