package job_controller

import (
	"testing"

	"github.com/kubeflow/common/operator/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestIsSucceeded(t *testing.T) {
	jobStatus := v1.JobStatus{
		Conditions: []v1.JobCondition{
			{
				Type:   v1.JobSucceeded,
				Status: corev1.ConditionTrue,
			},
		},
	}
	assert.True(t, isSucceeded(jobStatus))
}
func TestUpdateJobConditions(t *testing.T) {
	jobStatus := v1.JobStatus{

	}
	conditionType := v1.JobCreated
	reason := "Job Created"
	message := "Job Created"

	err := updateJobConditions(&jobStatus, conditionType, reason, message)
	if assert.NoError(t, err) {
		conditionInStatus := jobStatus.Conditions[0]
		assert.Equal(t, conditionInStatus.Type, conditionType)
		assert.Equal(t, conditionInStatus.Reason, reason)
		assert.Equal(t, conditionInStatus.Message, message)
	}
}

func TestUpdateJobReplicaStatuses(t *testing.T) {
	jobStatus := v1.JobStatus{}
	initializeReplicaStatuses(&jobStatus, "worker")
	setStatusForTest(&jobStatus, "worker", 2, 3, 1)
	assert.Equal(t, jobStatus.ReplicaStatuses["worker"].Failed, int32(2))
	assert.Equal(t, jobStatus.ReplicaStatuses["worker"].Succeeded, int32(3))
	assert.Equal(t, jobStatus.ReplicaStatuses["worker"].Active, int32(1))
}

func setStatusForTest(jobStatus *v1.JobStatus,  rtype v1.ReplicaType, failed, succeeded, active int32) {
	pod := corev1.Pod{
		Status: corev1.PodStatus{
		},
	}
	var i int32
	for i = 0; i < failed; i++ {
		pod.Status.Phase = corev1.PodFailed
		updateJobReplicaStatuses(jobStatus, rtype, &pod)
	}
	for i = 0; i < succeeded; i++ {
		pod.Status.Phase = corev1.PodSucceeded
		updateJobReplicaStatuses(jobStatus, rtype, &pod)
	}
	for i = 0; i < active; i++ {
		pod.Status.Phase = corev1.PodRunning
		updateJobReplicaStatuses(jobStatus, rtype, &pod)
	}
}
