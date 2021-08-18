package common

import (
	"testing"

	apiv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/core"
	testjobv1 "github.com/kubeflow/common/test_job/apis/test_job/v1"
	v12 "github.com/kubeflow/common/test_job/test_util/v1"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSetRestartPolicy(t *testing.T) {
	type tc struct {
		testJob               *testjobv1.TestJob
		expectedRestartPolicy v1.RestartPolicy
		expectedType          testjobv1.TestReplicaType
	}
	testCase := []tc{
		func() tc {
			tj := v12.NewTestJob(2)
			tj.Spec.TestReplicaSpecs[testjobv1.TestReplicaTypeWorker].RestartPolicy = apiv1.RestartPolicyExitCode
			return tc{
				testJob:               tj,
				expectedRestartPolicy: v1.RestartPolicyNever,
				expectedType:          testjobv1.TestReplicaTypeWorker,
			}
		}(),
		func() tc {
			tj := v12.NewTestJob(2)
			tj.Spec.TestReplicaSpecs[testjobv1.TestReplicaTypeWorker].RestartPolicy = apiv1.RestartPolicyNever
			return tc{
				testJob:               tj,
				expectedRestartPolicy: v1.RestartPolicyNever,
				expectedType:          testjobv1.TestReplicaTypeWorker,
			}
		}(),
		func() tc {
			tj := v12.NewTestJob(2)
			tj.Spec.TestReplicaSpecs[testjobv1.TestReplicaTypeWorker].RestartPolicy = apiv1.RestartPolicyAlways
			return tc{
				testJob:               tj,
				expectedRestartPolicy: v1.RestartPolicyAlways,
				expectedType:          testjobv1.TestReplicaTypeWorker,
			}
		}(),
		func() tc {
			tj := v12.NewTestJob(2)
			tj.Spec.TestReplicaSpecs[testjobv1.TestReplicaTypeWorker].RestartPolicy = apiv1.RestartPolicyOnFailure
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
		core.SetRestartPolicy(&podTemplate, spec)
		if podTemplate.Spec.RestartPolicy != c.expectedRestartPolicy {
			t.Errorf("Expected %s, got %s", c.expectedRestartPolicy, podTemplate.Spec.RestartPolicy)
		}
	}
}

func TestIsNonGangSchedulerSet(t *testing.T) {
	replicaSpecs := map[apiv1.ReplicaType]*apiv1.ReplicaSpec{}
	assert.False(t, isNonGangSchedulerSet(replicaSpecs))

	replicaSpecs[apiv1.ReplicaType(testjobv1.TestReplicaTypeWorker)] = &apiv1.ReplicaSpec{
		Template: v1.PodTemplateSpec{
			Spec: v1.PodSpec{
				SchedulerName: gangSchedulerName,
			},
		},
	}
	assert.False(t, isNonGangSchedulerSet(replicaSpecs))

	replicaSpecs[apiv1.ReplicaType(testjobv1.TestReplicaTypeWorker)] = &apiv1.ReplicaSpec{
		Template: v1.PodTemplateSpec{
			Spec: v1.PodSpec{
				SchedulerName: "other-scheduler",
			},
		},
	}
	assert.True(t, isNonGangSchedulerSet(replicaSpecs))
}

func TestCalculatePodSliceSize(t *testing.T) {
	type testCase struct {
		pods         []*v1.Pod
		replicas     int
		expectedSize int
	}

	pods := []*v1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{apiv1.ReplicaIndexLabel: "0"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{apiv1.ReplicaIndexLabel: "1"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{apiv1.ReplicaIndexLabel: "2"},
			},
		},
	}

	var testCases = []testCase{
		{
			pods:         pods,
			replicas:     3,
			expectedSize: 3,
		},
		{
			pods:         pods,
			replicas:     4,
			expectedSize: 4,
		},
		{
			pods:         pods,
			replicas:     2,
			expectedSize: 3,
		},
		{
			pods: append(pods, &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{apiv1.ReplicaIndexLabel: "4"},
				},
			}),
			replicas:     3,
			expectedSize: 5,
		},
	}

	for _, tc := range testCases {
		result := core.CalculatePodSliceSize(tc.pods, tc.replicas)
		assert.Equal(t, tc.expectedSize, result)
	}
}

func TestFilterPodsForReplicaType(t *testing.T) {
	pods := []*v1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "a",
				Labels: map[string]string{apiv1.ReplicaTypeLabel: "foo"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "b",
				Labels: map[string]string{apiv1.ReplicaTypeLabel: "bar"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "c",
				Labels: map[string]string{apiv1.ReplicaTypeLabelDeprecated: "foo"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "d",
				Labels: map[string]string{apiv1.ReplicaTypeLabelDeprecated: "bar"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "e",
				Labels: map[string]string{
					apiv1.ReplicaTypeLabel:           "foo",
					apiv1.ReplicaTypeLabelDeprecated: "bar",
				},
			},
		},
	}
	c := &JobController{}
	got, err := c.FilterPodsForReplicaType(pods, "foo")
	if err != nil {
		t.Fatalf("FilterPodsForReplicaType returned error: %v", err)
	}
	want := []*v1.Pod{pods[0], pods[2], pods[4]}
	assert.Equal(t, want, got)
}
