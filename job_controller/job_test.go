package job_controller

import (
	"testing"
	"time"

	common "github.com/kubeflow/common/operator/v1"
	"github.com/kubeflow/common/test_job/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeletePodsAndServices(T *testing.T) {
	type testCase struct {
		cleanPodPolicy               common.CleanPodPolicy
		deleteRunningPodAndService   bool
		deleteSucceededPodAndService bool
	}

	var testcase = []testCase{
		{
			common.CleanPodPolicyRunning,
			true,
			false,
		},
		{
			common.CleanPodPolicyAll,
			true,
			true,
		},
		{
			common.CleanPodPolicyNone,
			false,
			false,
		},
	}

	for _, tc := range testcase {
		runningPod := newPod("runningPod", corev1.PodRunning)
		succeededPod := newPod("succeededPod", corev1.PodSucceeded)
		allPods := []*corev1.Pod{runningPod, succeededPod}
		runningPodService := newService("runningPod")
		succeededPodService := newService("succeededPod")
		allServices := []*corev1.Service{runningPodService, succeededPodService}

		testJobController := TestJobController{
			pods:     allPods,
			services: allServices,
		}

		mainJobController := JobController{
			Controller: &testJobController,
		}
		runPolicy := common.RunPolicy{
			CleanPodPolicy: &tc.cleanPodPolicy,
		}

		var job interface{}
		err := mainJobController.deletePodsAndServices(&runPolicy, job, allPods)

		if assert.NoError(T, err) {
			if tc.deleteRunningPodAndService {
				// should delete the running pod and its service
				assert.NotContains(T, testJobController.pods, runningPod)
				assert.NotContains(T, testJobController.services, runningPodService)
			} else {
				// should NOT delete the running pod and its service
				assert.Contains(T, testJobController.pods, runningPod)
				assert.Contains(T, testJobController.services, runningPodService)
			}

			if tc.deleteSucceededPodAndService {
				// should delete the SUCCEEDED pod and its service
				assert.NotContains(T, testJobController.pods, succeededPod)
				assert.NotContains(T, testJobController.services, succeededPodService)
			} else {
				// should NOT delete the SUCCEEDED pod and its service
				assert.Contains(T, testJobController.pods, succeededPod)
				assert.Contains(T, testJobController.services, succeededPodService)
			}
		}
	}
}

func TestCleanupJobIfTTL(T *testing.T) {
	ttl := int32(0)
	runPolicy := common.RunPolicy{
		TTLSecondsAfterFinished: &ttl,
	}
	oneDayAgo := time.Now()
	// one day ago
	oneDayAgo.AddDate(0, 0, -1)
	jobStatus := common.JobStatus{
		CompletionTime: &metav1.Time{
			Time: oneDayAgo,
		},
	}

	testJobController := &TestJobController{
		job: &v1.TestJob{},
	}
	mainJobController := JobController{
		Controller: testJobController,
	}

	var job interface{}
	err := mainJobController.cleanupJobIfTTL(&runPolicy, jobStatus, job)
	if assert.NoError(T, err) {
		// job field is zeroed
		assert.Empty(T, testJobController.job)
	}
}

func newPod(name string, phase corev1.PodPhase) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Status: corev1.PodStatus{
			Phase: phase,
		},
	}
	return pod
}

func newService(name string) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return service
}
