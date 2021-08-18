package core

import (
	"sort"
	"time"

	log "github.com/sirupsen/logrus"

	apiv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

// RecordAbnormalPods records the active pod whose latest condition is not in True status.
func RecordAbnormalPods(activePods []*v1.Pod, object runtime.Object, recorder record.EventRecorder) {
	for _, pod := range activePods {
		// If the pod starts running, should checks the container statuses rather than the conditions.
		recordContainerStatus := func(status *v1.ContainerStatus) {
			if status.State.Terminated != nil && status.State.Terminated.ExitCode != 0 {
				terminated := status.State.Terminated
				recorder.Eventf(object, v1.EventTypeWarning, terminated.Reason,
					"Error pod %s container %s exitCode: %d terminated message: %s",
					pod.Name, status.Name, terminated.ExitCode, terminated.Message)
			}
			// The terminated state and waiting state don't simultaneously exists, checks them at the same time.
			if status.State.Waiting != nil && status.State.Waiting.Message != "" {
				wait := status.State.Waiting
				recorder.Eventf(object, v1.EventTypeWarning, wait.Reason,
					"Error pod %s container %s waiting message: %s", pod.Name, status.Name, wait.Message)
			}
		}
		if len(pod.Status.ContainerStatuses) != 0 {
			for _, status := range pod.Status.ContainerStatuses {
				recordContainerStatus(&status)
			}
			// If the pod has container status info, that means the init container statuses are normal.
			continue
		}
		if len(pod.Status.InitContainerStatuses) != 0 {
			for _, status := range pod.Status.InitContainerStatuses {
				recordContainerStatus(&status)
			}
			continue
		}
		if len(pod.Status.Conditions) == 0 {
			continue
		}
		// Should not modify the original pod which is stored in the informer cache.
		status := pod.Status.DeepCopy()
		sort.Slice(status.Conditions, func(i, j int) bool {
			return status.Conditions[i].LastTransitionTime.After(status.Conditions[j].LastTransitionTime.Time)
		})
		condition := status.Conditions[0]
		if condition.Status == v1.ConditionTrue {
			continue
		}
		recorder.Eventf(object, v1.EventTypeWarning, condition.Reason, "Error pod %s condition message: %s", pod.Name, condition.Message)
	}
}

// PastActiveDeadline checks if job has ActiveDeadlineSeconds field set and if it is exceeded.
func PastActiveDeadline(runPolicy *apiv1.RunPolicy, jobStatus apiv1.JobStatus) bool {
	if runPolicy.ActiveDeadlineSeconds == nil || jobStatus.StartTime == nil {
		return false
	}
	now := metav1.Now()
	start := jobStatus.StartTime.Time
	duration := now.Time.Sub(start)
	allowedDuration := time.Duration(*runPolicy.ActiveDeadlineSeconds) * time.Second
	return duration >= allowedDuration
}

// PastBackoffLimit checks if container restartCounts sum exceeds BackoffLimit
// this method applies only to pods with restartPolicy == OnFailure or Always
func PastBackoffLimit(jobName string, runPolicy *apiv1.RunPolicy,
	replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec, pods []*v1.Pod,
	podFilterFunc func(pods []*v1.Pod, replicaType apiv1.ReplicaType) ([]*v1.Pod, error)) (bool, error) {
	if runPolicy.BackoffLimit == nil {
		return false, nil
	}
	result := int32(0)
	for rtype, spec := range replicas {
		if spec.RestartPolicy != apiv1.RestartPolicyOnFailure && spec.RestartPolicy != apiv1.RestartPolicyAlways {
			log.Warnf("The restart policy of replica %v of the job %v is not OnFailure or Always. Not counted in backoff limit.", rtype, jobName)
			continue
		}
		// Convert ReplicaType to lower string.
		pods, err := podFilterFunc(pods, rtype)
		if err != nil {
			return false, err
		}
		for i := range pods {
			po := pods[i]
			if po.Status.Phase != v1.PodRunning {
				continue
			}
			for j := range po.Status.InitContainerStatuses {
				stat := po.Status.InitContainerStatuses[j]
				result += stat.RestartCount
			}
			for j := range po.Status.ContainerStatuses {
				stat := po.Status.ContainerStatuses[j]
				result += stat.RestartCount
			}
		}
	}

	if *runPolicy.BackoffLimit == 0 {
		return result > 0, nil
	}
	return result >= *runPolicy.BackoffLimit, nil
}
