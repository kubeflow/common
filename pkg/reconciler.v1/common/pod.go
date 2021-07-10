package common

import (
	"context"
	"sort"
	"strconv"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	commonutil "github.com/kubeflow/common/pkg/util"
	trainutil "github.com/kubeflow/common/pkg/util/train"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const DefaultContainerName = "kubeflow"

var (
	// Prometheus metrics
	createdPodsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "created_pods_total",
		Help: "The total number of created pods",
	})
	deletedPodsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "deleted_pods_total",
		Help: "The total number of deleted pods",
	})
	failedPodsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "failed_pods_total",
		Help: "The total number of failed pods",
	})
)

func (r *KubeflowReconciler) ReconcilePods(
	job client.Object,
	jobStatus *commonv1.JobStatus,
	pods []*corev1.Pod,
	rtype commonv1.ReplicaType,
	spec *commonv1.ReplicaSpec,
	replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec) error {

	// Convert ReplicaType to lower string.
	logger := commonutil.LoggerForReplica(job, rtype)
	// Get all pods for the type rt.
	pods, err := r.FilterPodsForReplicaType(pods, rtype)
	if err != nil {
		return err
	}
	numReplicas := int(*spec.Replicas)
	var masterRole bool

	InitializeReplicaStatuses(jobStatus, rtype)

	// GetPodSlices will return enough information here to make decision to add/remove/update resources.
	//
	// For example, let's assume we have pods with replica-index 0, 1, 2
	// If replica is 4, return a slice with size 4. [[0],[1],[2],[]], a pod with replica-index 3 will be created.
	//
	// If replica is 1, return a slice with size 3. [[0],[1],[2]], pod with replica-index 1 and 2 are out of range and will be deleted.
	podSlices := r.GetPodSlices(pods, numReplicas, logger)
	for index, podSlice := range podSlices {
		if len(podSlice) > 1 {
			logger.Warningf("We have too many pods for %s %d", rtype, index)
		} else if len(podSlice) == 0 {
			logger.Infof("Need to create new pod: %s-%d", rtype, index)

			// check if this replica is the master role
			masterRole = r.IsMasterRole(replicas, rtype, index)
			err = r.CreateNewPod(job, rtype, strconv.Itoa(index), spec, masterRole, replicas)
			if err != nil {
				return err
			}
		} else {
			// Check the status of the current pod.
			pod := podSlice[0]

			// check if the index is in the valid range, if not, we should kill the pod
			if index < 0 || index >= numReplicas {
				err = r.DeletePod(pod.Namespace, pod.Name, job)
				if err != nil {
					return err
				}
			}

			// Get the exit code of the container.
			var exitCode int32 = 0xbeef // magic number
			for _, status := range pod.Status.ContainerStatuses {
				state := status.State
				if status.Name == r.GetDefaultContainerName() && state.Terminated != nil {
					exitCode = state.Terminated.ExitCode
					logger.Infof("Pod: %v.%v exited with code %v", pod.Namespace, pod.Name, exitCode)
					r.recorder.Eventf(job, corev1.EventTypeNormal, "ExitedWithCode", "Pod: %v.%v exited with code %v", pod.Namespace, pod.Name, exitCode)
				}
			}
			// Check if the pod is retryable.
			if spec.RestartPolicy == commonv1.RestartPolicyExitCode {
				if pod.Status.Phase == corev1.PodFailed && trainutil.IsRetryableExitCode(exitCode) {
					failedPodsCount.Inc()
					logger.Infof("Need to restart the pod: %v.%v", pod.Namespace, pod.Name)
					if err = r.DeletePod(pod.Namespace, pod.Name, job); err != nil {
						return err
					}
				}
			}

			UpdateJobReplicaStatuses(jobStatus, rtype, pod)
		}
	}
	return nil

}

func (r *KubeflowReconciler) CreateNewPod(job client.Object, rt commonv1.ReplicaType, index string,
	spec *commonv1.ReplicaSpec, masterRole bool, replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec) error {

	logger := commonutil.LoggerForReplica(job, rt)

	labels := r.GenLabels(job.GetName())
	labels[commonv1.ReplicaTypeLabel] = string(rt)
	labels[commonv1.ReplicaIndexLabel] = index
	if masterRole {
		labels[commonv1.JobRoleLabel] = "master"
	}

	podTemplate := spec.Template.DeepCopy()

	podTemplate.Name = r.GenPodName(job.GetName(), rt, index)
	if podTemplate.Labels == nil {
		podTemplate.Labels = make(map[string]string)
	}

	for key, value := range labels {
		podTemplate.Labels[key] = value
	}

	// TODO: Customization Phase

	if podTemplate.Spec.RestartPolicy != corev1.RestartPolicy("") {
		errMsg := "Restart policy in pod template will be overwritten by restart policy in replica spec"
		logger.Warning(errMsg)
		r.recorder.Event(job, corev1.EventTypeWarning, "SettedPodTemplateRestartPolicy", errMsg)
	}
	if spec.RestartPolicy == commonv1.RestartPolicyExitCode {
		podTemplate.Spec.RestartPolicy = corev1.RestartPolicyNever
	} else {
		podTemplate.Spec.RestartPolicy = corev1.RestartPolicy(spec.RestartPolicy)
	}

	// TODO: Setting For Gang
	if r.GangSchedulingEnabled() {

	}

	pod := &corev1.Pod{
		ObjectMeta: podTemplate.ObjectMeta,
		Spec:       podTemplate.Spec,
	}

	err := controllerutil.SetControllerReference(job, pod, r.Scheme)
	if err != nil {
		return err
	}

	err = r.Create(context.Background(), pod)
	if err != nil && errors.IsTimeout(err) {
		return nil
	} else if err != nil {
		return err
	}
	createdPodsCount.Inc()
	return nil
}

func (r *KubeflowReconciler) GenPodName(jobName string, rtype commonv1.ReplicaType, index string) string {
	return GenGeneralName(jobName, rtype, index)
}

func (r *KubeflowReconciler) GetDefaultContainerName() string {
	return DefaultContainerName
}

func (r *KubeflowReconciler) GetPodSlices(pods []*corev1.Pod, replicas int, logger *log.Entry) [][]*corev1.Pod {
	podSlices := make([][]*corev1.Pod, calculatePodSliceSize(pods, replicas))
	for _, pod := range pods {
		if _, ok := pod.Labels[commonv1.ReplicaIndexLabel]; !ok {
			logger.Warning("The pod do not have the index label.")
			continue
		}
		index, err := strconv.Atoi(pod.Labels[commonv1.ReplicaIndexLabel])
		if err != nil {
			logger.Warningf("Error when strconv.Atoi: %v", err)
			continue
		}
		if index < 0 || index >= replicas {
			logger.Warningf("The label index is not expected: %d, pod: %s/%s", index, pod.Namespace, pod.Name)
		}

		podSlices[index] = append(podSlices[index], pod)
	}
	return podSlices
}

func MaxInt(x, y int) int {
	if x < y {
		return y
	}
	return x
}

// calculatePodSliceSize compare max pod index with desired replicas and return larger size
func calculatePodSliceSize(pods []*corev1.Pod, replicas int) int {
	size := 0
	for _, pod := range pods {
		if _, ok := pod.Labels[commonv1.ReplicaIndexLabel]; !ok {
			continue
		}
		index, err := strconv.Atoi(pod.Labels[commonv1.ReplicaIndexLabel])
		if err != nil {
			continue
		}
		size = MaxInt(size, index)
	}

	// size comes from index, need to +1 to indicate real size
	return MaxInt(size+1, replicas)
}

func (r *KubeflowReconciler) DeletePod(ns string, name string, job client.Object) error {
	pod := &corev1.Pod{}
	pod.Name = name
	pod.Namespace = ns
	err := r.Delete(context.Background(), pod)
	if err == nil {
		deletedPodsCount.Inc()
	}
	return err
}

func (r *KubeflowReconciler) GetPodsForJob(ctx context.Context, job client.Object) ([]*corev1.Pod, error) {
	podList := &corev1.PodList{}
	err := r.List(ctx, podList, client.MatchingLabels(r.GenLabels(job.GetName())))
	if err != nil {
		return nil, err
	}

	var pods []*corev1.Pod = nil
	for _, pod := range podList.Items {
		pods = append(pods, &pod)
	}

	return pods, nil
	// TODO: (zw0610) adding controller reference management
	//// If any adoptions are attempted, we should first recheck for deletion
	//// with an uncached quorum read sometime after listing Pods (see #42639).
	//canAdoptFunc := RecheckDeletionTimestamp(func() (metav1.Object, error) {
	//	fresh := r.EmptyJob()
	//	err = r.APIReader.Get(ctx, types.NamespacedName{Namespace: job.GetNamespace(), Name: job.GetName()}, fresh)
	//	if err != nil {
	//		return nil, err
	//	}
	//	if fresh.GetUID() != job.GetUID() {
	//		return nil, fmt.Errorf("original Job %v/%v is gone: got uid %v, wanted %v", job.GetNamespace(), job.GetName(), fresh.GetUID(), job.GetUID())
	//	}
	//	return fresh, nil
	//})
	//cm := control.NewPodControllerRefManager(jc.PodControl, job, selector, r.GetAPIGroupVersionKind(), canAdoptFunc)
	//return cm.ClaimPods(pods)
}

func (r *KubeflowReconciler) RecordAbnormalPods(activePods []*corev1.Pod, object client.Object) {
	for _, pod := range activePods {
		// If the pod starts running, should checks the container statuses rather than the conditions.
		recordContainerStatus := func(status *corev1.ContainerStatus) {
			if status.State.Terminated != nil && status.State.Terminated.ExitCode != 0 {
				terminated := status.State.Terminated
				r.recorder.Eventf(object, corev1.EventTypeWarning, terminated.Reason,
					"Error pod %s container %s exitCode: %d terminated message: %s",
					pod.Name, status.Name, terminated.ExitCode, terminated.Message)
			}
			// The terminated state and waiting state don't simultaneously exists, checks them at the same time.
			if status.State.Waiting != nil && status.State.Waiting.Message != "" {
				wait := status.State.Waiting
				r.recorder.Eventf(object, corev1.EventTypeWarning, wait.Reason,
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
		if condition.Status == corev1.ConditionTrue {
			continue
		}
		r.recorder.Eventf(object, corev1.EventTypeWarning, condition.Reason, "Error pod %s condition message: %s", pod.Name, condition.Message)
	}
}

func (r *KubeflowReconciler) FilterPodsForReplicaType(pods []*corev1.Pod, replicaType commonv1.ReplicaType) ([]*corev1.Pod, error) {
	var result []*corev1.Pod

	replicaSelector := &metav1.LabelSelector{
		MatchLabels: make(map[string]string),
	}

	replicaSelector.MatchLabels[commonv1.ReplicaTypeLabel] = string(replicaType)

	for _, pod := range pods {
		selector, err := metav1.LabelSelectorAsSelector(replicaSelector)
		if err != nil {
			return nil, err
		}
		if !selector.Matches(labels.Set(pod.Labels)) {
			continue
		}
		result = append(result, pod)
	}
	return result, nil
}
