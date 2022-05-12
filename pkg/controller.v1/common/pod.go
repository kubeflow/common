// Copyright 2019 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	apiv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/controller.v1/control"
	"github.com/kubeflow/common/pkg/controller.v1/expectation"
	"github.com/kubeflow/common/pkg/core"
	commonutil "github.com/kubeflow/common/pkg/util"
	utillabels "github.com/kubeflow/common/pkg/util/labels"
	trainutil "github.com/kubeflow/common/pkg/util/train"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
)

const (
	// gang scheduler name.
	gangSchedulerName = "volcano"
	// podTemplateRestartPolicyReason is the warning reason when the restart
	// policy is set in pod template.
	podTemplateRestartPolicyReason = "SettedPodTemplateRestartPolicy"
	// exitedWithCodeReason is the normal reason when the pod is exited because of the exit code.
	exitedWithCodeReason = "ExitedWithCode"
	// podTemplateSchedulerNameReason is the warning reason when other scheduler name is set
	// in pod templates with gang-scheduling enabled
	podTemplateSchedulerNameReason = "SettedPodTemplateSchedulerName"
	// gangSchedulingPodGroupAnnotation is the annotation key used by batch schedulers
	gangSchedulingPodGroupAnnotation = "scheduling.k8s.io/group-name"
)

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

// When a pod is created, enqueue the job that manages it and update its expectations.
func (jc *JobController) AddPod(obj interface{}) {
	pod := obj.(*v1.Pod)
	if pod.DeletionTimestamp != nil {
		// on a restart of the controller controller, it's possible a new pod shows up in a state that
		// is already pending deletion. Prevent the pod from being a creation observation.
		// jc.deletePod(pod)
		return
	}

	// If it has a ControllerRef, that's all that matters.
	if controllerRef := metav1.GetControllerOf(pod); controllerRef != nil {
		job := jc.resolveControllerRef(pod.Namespace, controllerRef)

		logger := commonutil.LoggerForPod(pod, jc.Controller.GetAPIGroupVersionKind().Kind)

		if job == nil {
			if utillabels.HasKnownLabels(pod.Labels, jc.Controller.GetGroupNameLabelValue()) {
				logger.Info("This pod's job does not exist")
			}
			return
		}

		jobKey, err := KeyFunc(job)
		if err != nil {
			logger.Infof("Failed to get the jobkey: %v", err)
			return
		}

		rType, err := utillabels.ReplicaType(pod.Labels)
		if err != nil {
			logger.Infof("This pod maybe not created by %v", jc.Controller.ControllerName())
			return
		}

		expectationPodsKey := expectation.GenExpectationPodsKey(jobKey, string(rType))

		jc.Expectations.CreationObserved(expectationPodsKey)
		// TODO: we may need add backoff here
		jc.WorkQueue.Add(jobKey)

		return
	}

}

// When a pod is updated, figure out what job is managing it and wake it up.
// If the labels of the pod have changed we need to awaken both the old
// and new replica set. old and cur must be *v1.Pod types.
func (jc *JobController) UpdatePod(old, cur interface{}) {
	curPod := cur.(*v1.Pod)
	oldPod := old.(*v1.Pod)
	if curPod.ResourceVersion == oldPod.ResourceVersion {
		// Periodic resync will send update events for all known pods.
		// Two different versions of the same pod will always have different RVs.
		return
	}

	logger := commonutil.LoggerForPod(curPod, jc.Controller.GetAPIGroupVersionKind().Kind)
	curControllerRef := metav1.GetControllerOf(curPod)
	oldControllerRef := metav1.GetControllerOf(oldPod)
	controllerRefChanged := !reflect.DeepEqual(curControllerRef, oldControllerRef)
	if controllerRefChanged && oldControllerRef != nil {
		// The ControllerRef was changed. Sync the old controller, if any.
		if job := jc.resolveControllerRef(oldPod.Namespace, oldControllerRef); job != nil {
			logger.Infof("pod ControllerRef updated: %v, %v", curPod, oldPod)
			jobKey, err := KeyFunc(job)
			if err != nil {
				return
			}
			// TODO: we may need add backoff here
			jc.WorkQueue.Add(jobKey)
		}
	}

	// If it has a ControllerRef, that's all that matters.
	if curControllerRef != nil {
		job := jc.resolveControllerRef(curPod.Namespace, curControllerRef)
		if job == nil {
			return
		}
		logger.Debugf("pod has a ControllerRef: %v, %v", curPod, oldPod)
		jobKey, err := KeyFunc(job)
		if err != nil {
			return
		}
		// TODO: we may need add backoff here
		jc.WorkQueue.Add(jobKey)
		return
	}
}

// When a pod is deleted, enqueue the job that manages the pod and update its expectations.
// obj could be an *v1.Pod, or a DeletionFinalStateUnknown marker item.
func (jc *JobController) DeletePod(obj interface{}) {
	pod, ok := obj.(*v1.Pod)

	logger := commonutil.LoggerForPod(pod, jc.Controller.GetAPIGroupVersionKind().Kind)

	// When a delete is dropped, the relist will notice a pod in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the pod
	// changed labels the new job will not be woken up till the periodic resync.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("couldn't get object from tombstone %+v", obj))
			return
		}
		pod, ok = tombstone.Obj.(*v1.Pod)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("tombstone contained object that is not a pod %+v", obj))
			return
		}
	}

	controllerRef := metav1.GetControllerOf(pod)
	if controllerRef == nil {
		// No controller should care about orphans being deleted.
		return
	}
	job := jc.resolveControllerRef(pod.Namespace, controllerRef)
	if job == nil {
		return
	}
	jobKey, err := KeyFunc(job)
	if err != nil {
		return
	}

	rType, err := utillabels.ReplicaType(pod.Labels)
	if err != nil {
		logger.Infof("This pod maybe not created by %v", jc.Controller.ControllerName())
		return
	}

	expectationPodsKey := expectation.GenExpectationPodsKey(jobKey, string(rType))

	jc.Expectations.DeletionObserved(expectationPodsKey)
	deletedPodsCount.Inc()
	// TODO: we may need add backoff here
	jc.WorkQueue.Add(jobKey)
}

// getPodsForJob returns the set of pods that this job should manage.
// It also reconciles ControllerRef by adopting/orphaning.
// Note that the returned Pods are pointers into the cache.
func (jc *JobController) GetPodsForJob(jobObject interface{}) ([]*v1.Pod, error) {
	job, ok := jobObject.(metav1.Object)
	if !ok {
		return nil, fmt.Errorf("job is not of type metav1.Object")
	}

	// Create selector.
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: jc.GenLabels(job.GetName()),
	})

	if err != nil {
		return nil, fmt.Errorf("couldn't convert Job selector: %v", err)
	}
	// List all pods to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	pods, err := jc.PodLister.Pods(job.GetNamespace()).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	// If any adoptions are attempted, we should first recheck for deletion
	// with an uncached quorum read sometime after listing Pods (see #42639).
	canAdoptFunc := RecheckDeletionTimestamp(func() (metav1.Object, error) {
		fresh, err := jc.Controller.GetJobFromAPIClient(job.GetNamespace(), job.GetName())
		if err != nil {
			return nil, err
		}
		if fresh.GetUID() != job.GetUID() {
			return nil, fmt.Errorf("original Job %v/%v is gone: got uid %v, wanted %v", job.GetNamespace(), job.GetName(), fresh.GetUID(), job.GetUID())
		}
		return fresh, nil
	})
	cm := control.NewPodControllerRefManager(jc.PodControl, job, selector, jc.Controller.GetAPIGroupVersionKind(), canAdoptFunc)
	return cm.ClaimPods(pods)
}

// FilterPodsForReplicaType returns pods belong to a replicaType.
func (jc *JobController) FilterPodsForReplicaType(pods []*v1.Pod, replicaType string) ([]*v1.Pod, error) {
	return core.FilterPodsForReplicaType(pods, replicaType)
}

// getPodSlices returns a slice, which element is the slice of pod.
// It gives enough information to caller to make decision to up/down scale resources.
func (jc *JobController) GetPodSlices(pods []*v1.Pod, replicas int, logger *log.Entry) [][]*v1.Pod {
	return core.GetPodSlices(pods, replicas, logger)
}

// ReconcilePods checks and updates pods for each given ReplicaSpec.
// It will requeue the job in case of an error while creating/deleting pods.
func (jc *JobController) ReconcilePods(
	job interface{},
	jobStatus *apiv1.JobStatus,
	pods []*v1.Pod,
	rType apiv1.ReplicaType,
	spec *apiv1.ReplicaSpec,
	replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec,
	runPolicy *apiv1.RunPolicy) error {

	rt := strings.ToLower(string(rType))
	metaObject, ok := job.(metav1.Object)
	if !ok {
		return fmt.Errorf("job is not a metav1.Object type")
	}
	runtimeObject, ok := job.(runtime.Object)
	if !ok {
		return fmt.Errorf("job is not a runtime.Object type")
	}
	jobKey, err := KeyFunc(metaObject)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for job object %#v: %v", job, err))
		return err
	}
	expectationPodsKey := expectation.GenExpectationPodsKey(jobKey, rt)

	// Convert ReplicaType to lower string.
	logger := commonutil.LoggerForReplica(metaObject, rt)
	// Get all pods for the type rt.
	pods, err = jc.FilterPodsForReplicaType(pods, rt)
	if err != nil {
		return err
	}
	numReplicas := int(*spec.Replicas)
	var masterRole bool

	initializeReplicaStatuses(jobStatus, rType)

	// GetPodSlices will return enough information here to make decision to add/remove/update resources.
	//
	// For example, let's assume we have pods with replica-index 0, 1, 2
	// If replica is 4, return a slice with size 4. [[0],[1],[2],[]], a pod with replica-index 3 will be created.
	//
	// If replica is 1, return a slice with size 3. [[0],[1],[2]], pod with replica-index 1 and 2 are out of range and will be deleted.
	podSlices := jc.GetPodSlices(pods, numReplicas, logger)
	for index, podSlice := range podSlices {
		if len(podSlice) > 1 {
			logger.Warningf("We have too many pods for %s %d", rt, index)
		} else if len(podSlice) == 0 {
			logger.Infof("Need to create new pod: %s-%d", rt, index)

			if JobSuspended(runPolicy) || commonutil.IsSuspended(*jobStatus) {
				logger.Warningf("job is Suspended %s/%s", metaObject.GetNamespace(), metaObject.GetName())
				continue
			}
			// check if this replica is the master role
			masterRole = jc.Controller.IsMasterRole(replicas, rType, index)
			err = jc.createNewPod(job, rt, index, spec, masterRole, replicas)
			if err != nil {
				return err
			}
		} else {
			// Check the status of the current pod.
			pod := podSlice[0]

			// check if the index is in the valid range, if not, we should kill the pod
			if index < 0 || index >= numReplicas || JobSuspended(runPolicy) {
				err = jc.PodControl.DeletePod(pod.Namespace, pod.Name, runtimeObject)
				if err != nil {
					return err
				}
				// Deletion is expected
				jc.Expectations.RaiseExpectations(expectationPodsKey, 0, 1)
			}

			// Get the exit code of the container.
			var exitCode int32 = 0xbeef // magic number
			for _, status := range pod.Status.ContainerStatuses {
				state := status.State
				if status.Name == jc.Controller.GetDefaultContainerName() && state.Terminated != nil {
					exitCode = state.Terminated.ExitCode
					logger.Infof("Pod: %v.%v exited with code %v", pod.Namespace, pod.Name, exitCode)
					jc.Recorder.Eventf(runtimeObject, v1.EventTypeNormal, exitedWithCodeReason, "Pod: %v.%v exited with code %v", pod.Namespace, pod.Name, exitCode)
				}
			}
			// Check if the pod is retryable.
			if spec.RestartPolicy == apiv1.RestartPolicyExitCode {
				if pod.Status.Phase == v1.PodFailed && trainutil.IsRetryableExitCode(exitCode) {
					failedPodsCount.Inc()
					logger.Infof("Need to restart the pod: %v.%v", pod.Namespace, pod.Name)
					if err := jc.PodControl.DeletePod(pod.Namespace, pod.Name, runtimeObject); err != nil {
						return err
					}
					// Deletion is expected
					jc.Expectations.RaiseExpectations(expectationPodsKey, 0, 1)
				}
			}

			updateJobReplicaStatuses(jobStatus, rType, pod)
		}
	}
	return nil
}

// createNewPod creates a new pod for the given index and type.
func (jc *JobController) createNewPod(job interface{}, rt string, index int, spec *apiv1.ReplicaSpec, masterRole bool,
	replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec) error {

	metaObject, ok := job.(metav1.Object)
	if !ok {
		return fmt.Errorf("job is not a metav1.Object type")
	}
	runtimeObject, ok := job.(runtime.Object)
	if !ok {
		return fmt.Errorf("job is not a runtime.Object type")
	}
	jobKey, err := KeyFunc(metaObject)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for job object %#v: %v", job, err))
		return err
	}
	logger := commonutil.LoggerForReplica(metaObject, rt)

	// Set type and index for the worker.
	labels := jc.GenLabels(metaObject.GetName())
	utillabels.SetReplicaType(labels, rt)
	utillabels.SetReplicaIndex(labels, index)

	if masterRole {
		utillabels.SetJobRole(labels, "master")
	}

	podTemplate := spec.Template.DeepCopy()

	idxStr := strconv.Itoa(index)
	// Set name for the template.
	podTemplate.Name = GenGeneralName(metaObject.GetName(), rt, idxStr)

	if podTemplate.Labels == nil {
		podTemplate.Labels = make(map[string]string)
	}

	for key, value := range labels {
		podTemplate.Labels[key] = value
	}

	if err := jc.Controller.SetClusterSpec(job, podTemplate, rt, idxStr); err != nil {
		return err
	}

	// Submit a warning event if the user specifies restart policy for
	// the pod template. We recommend to set it from the replica level.
	if podTemplate.Spec.RestartPolicy != v1.RestartPolicy("") {
		errMsg := "Restart policy in pod template will be overwritten by restart policy in replica spec"
		logger.Warning(errMsg)
		jc.Recorder.Event(runtimeObject, v1.EventTypeWarning, podTemplateRestartPolicyReason, errMsg)
	}
	core.SetRestartPolicy(podTemplate, spec)

	// if gang-scheduling is enabled:
	// 1. if user has specified other scheduler, we report a warning without overriding any fields.
	// 2. if no SchedulerName is set for pods, then we set the SchedulerName to "volcano".
	if jc.Config.EnableGangScheduling {
		if isNonGangSchedulerSet(replicas) {
			errMsg := "Another scheduler is specified when gang-scheduling is enabled and it will not be overwritten"
			logger.Warning(errMsg)
			jc.Recorder.Event(runtimeObject, v1.EventTypeWarning, podTemplateSchedulerNameReason, errMsg)
		} else {
			podTemplate.Spec.SchedulerName = gangSchedulerName
		}

		if podTemplate.Annotations == nil {
			podTemplate.Annotations = map[string]string{}
		}

		if jc.Config.EnableGangScheduling {
			podTemplate.Annotations[gangSchedulingPodGroupAnnotation] = metaObject.GetName()
		}
	}

	// Creation is expected when there is no error returned
	// We use `RaiseExpectations` here to accumulate expectations since `SetExpectations` has no such kind of ability
	expectationPodsKey := expectation.GenExpectationPodsKey(jobKey, rt)
	jc.Expectations.RaiseExpectations(expectationPodsKey, 1, 0)

	controllerRef := jc.GenOwnerReference(metaObject)
	err = jc.PodControl.CreatePodsWithControllerRef(metaObject.GetNamespace(), podTemplate, runtimeObject, controllerRef)
	if err != nil && errors.IsTimeout(err) {
		// Pod is created but its initialization has timed out.
		// If the initialization is successful eventually, the
		// controller will observe the creation via the informer.
		// If the initialization fails, or if the pod keeps
		// uninitialized for a long time, the informer will not
		// receive any update, and the controller will create a new
		// pod when the expectation expires.
		return nil
	} else if err != nil {
		// Since error occurred(the informer won't observe this pod),
		// we decrement the expected number of creates
		// and wait until next reconciliation
		jc.Expectations.CreationObserved(expectationPodsKey)
		return err
	}
	createdPodsCount.Inc()
	return nil
}

func isNonGangSchedulerSet(replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec) bool {
	for _, spec := range replicas {
		if spec.Template.Spec.SchedulerName != "" && spec.Template.Spec.SchedulerName != gangSchedulerName {
			return true
		}
	}
	return false
}
