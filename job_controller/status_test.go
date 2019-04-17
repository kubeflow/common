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

func TestIsFailed(t *testing.T) {
	jobStatus := v1.JobStatus{
		Conditions: []v1.JobCondition{
			{
				Type:   v1.JobFailed,
				Status: corev1.ConditionTrue,
			},
		},
	}
	assert.True(t, isFailed(jobStatus))
}

func TestUpdateJobConditions(t *testing.T) {
	jobStatus := v1.JobStatus{}
	conditionType := v1.JobCreated
	reason := "Job Created"
	message := "Job Created"

	err := updateJobConditions(&jobStatus, conditionType, reason, message)
	if assert.NoError(t, err) {
		// Check JobCreated condition is appended
		conditionInStatus := jobStatus.Conditions[0]
		assert.Equal(t, conditionInStatus.Type, conditionType)
		assert.Equal(t, conditionInStatus.Reason, reason)
		assert.Equal(t, conditionInStatus.Message, message)
	}

	conditionType = v1.JobRunning
	reason = "Job Running"
	message = "Job Running"
	err = updateJobConditions(&jobStatus, conditionType, reason, message)
	if assert.NoError(t, err) {
		// Check JobRunning condition is appended
		conditionInStatus := jobStatus.Conditions[1]
		assert.Equal(t, conditionInStatus.Type, conditionType)
		assert.Equal(t, conditionInStatus.Reason, reason)
		assert.Equal(t, conditionInStatus.Message, message)
	}

	conditionType = v1.JobRestarting
	reason = "Job Restarting"
	message = "Job Restarting"
	err = updateJobConditions(&jobStatus, conditionType, reason, message)
	if assert.NoError(t, err) {
		// Check JobRunning condition is filtered out and JobRestarting state is appended
		conditionInStatus := jobStatus.Conditions[1]
		assert.Equal(t, conditionInStatus.Type, conditionType)
		assert.Equal(t, conditionInStatus.Reason, reason)
		assert.Equal(t, conditionInStatus.Message, message)
	}

	conditionType = v1.JobRunning
	reason = "Job Running"
	message = "Job Running"
	err = updateJobConditions(&jobStatus, conditionType, reason, message)
	if assert.NoError(t, err) {
		// Again, Check JobRestarting condition is filtered and JobRestarting is appended
		conditionInStatus := jobStatus.Conditions[1]
		assert.Equal(t, conditionInStatus.Type, conditionType)
		assert.Equal(t, conditionInStatus.Reason, reason)
		assert.Equal(t, conditionInStatus.Message, message)
	}

	conditionType = v1.JobFailed
	reason = "Job Failed"
	message = "Job Failed"
	err = updateJobConditions(&jobStatus, conditionType, reason, message)
	if assert.NoError(t, err) {
		// Check JobRunning condition is set to false
		jobRunningCondition := jobStatus.Conditions[1]
		assert.Equal(t, jobRunningCondition.Type, v1.JobRunning)
		assert.Equal(t, jobRunningCondition.Status, corev1.ConditionFalse)
		// Check JobFailed state is appended
		conditionInStatus := jobStatus.Conditions[2]
		assert.Equal(t, conditionInStatus.Type, conditionType)
		assert.Equal(t, conditionInStatus.Reason, reason)
		assert.Equal(t, conditionInStatus.Message, message)
	}
}

func TestUpdateJobReplicaStatuses(t *testing.T) {
	jobStatus := v1.JobStatus{}
	initializeReplicaStatuses(&jobStatus, "worker")
	_, ok := jobStatus.ReplicaStatuses["worker"]
	// assert ReplicaStatus for "worker" exists
	assert.True(t, ok)
	setStatusForTest(&jobStatus, "worker", 2, 3, 1)
	assert.Equal(t, jobStatus.ReplicaStatuses["worker"].Failed, int32(2))
	assert.Equal(t, jobStatus.ReplicaStatuses["worker"].Succeeded, int32(3))
	assert.Equal(t, jobStatus.ReplicaStatuses["worker"].Active, int32(1))
}

func setStatusForTest(jobStatus *v1.JobStatus, rtype v1.ReplicaType, failed, succeeded, active int32) {
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
