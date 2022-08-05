package common

import (
	"fmt"
	"reflect"
	"time"

	"sigs.k8s.io/scheduler-plugins/apis/scheduling/v1alpha1"

	apiv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/controller.v1/expectation"
	"github.com/kubeflow/common/pkg/core"
	commonutil "github.com/kubeflow/common/pkg/util"
	"github.com/kubeflow/common/pkg/util/k8sutil"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"volcano.sh/apis/pkg/apis/scheduling/v1beta1"
)

func (jc *JobController) DeletePodsAndServices(runPolicy *apiv1.RunPolicy, job interface{}, pods []*v1.Pod) error {
	if len(pods) == 0 {
		return nil
	}

	// Delete nothing when the cleanPodPolicy is None.
	if *runPolicy.CleanPodPolicy == apiv1.CleanPodPolicyNone {
		return nil
	}

	for _, pod := range pods {
		// Note that pending pod will turn into running once schedulable,
		// not cleaning it may leave orphan running pod in the future,
		// we should treat it equivalent to running phase here.
		if *runPolicy.CleanPodPolicy == apiv1.CleanPodPolicyRunning && pod.Status.Phase != v1.PodRunning && pod.Status.Phase != v1.PodPending {
			continue
		}
		if err := jc.PodControl.DeletePod(pod.Namespace, pod.Name, job.(runtime.Object)); err != nil {
			return err
		}
		// Pod and service have the same name, thus the service could be deleted using pod's name.
		if err := jc.ServiceControl.DeleteService(pod.Namespace, pod.Name, job.(runtime.Object)); err != nil {
			return err
		}
	}
	return nil
}

// recordAbnormalPods records the active pod whose latest condition is not in True status.
func (jc *JobController) recordAbnormalPods(activePods []*v1.Pod, object runtime.Object) {
	core.RecordAbnormalPods(activePods, object, jc.Recorder)
}

// ReconcileJobs checks and updates replicas for each given ReplicaSpec.
// It will requeue the job in case of an error while creating/deleting pods/services.
func (jc *JobController) ReconcileJobs(
	job interface{},
	replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec,
	jobStatus apiv1.JobStatus,
	runPolicy *apiv1.RunPolicy) error {

	metaObject, ok := job.(metav1.Object)
	jobName := metaObject.GetName()
	if !ok {
		return fmt.Errorf("job is not of type metav1.Object")
	}
	runtimeObject, ok := job.(runtime.Object)
	if !ok {
		return fmt.Errorf("job is not of type runtime.Object")
	}
	jobKey, err := KeyFunc(job)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for job object %#v: %v", job, err))
		return err
	}
	// Reset expectations
	// 1. Since `ReconcileJobs` is called, we expect that previous expectations are all satisfied,
	//    and it's safe to reset the expectations
	// 2. Reset expectations can avoid dirty data such as `expectedDeletion = -1`
	//    (pod or service was deleted unexpectedly)
	jc.ResetExpectations(jobKey, replicas)

	log.Infof("Reconciling for job %s", metaObject.GetName())
	pods, err := jc.Controller.GetPodsForJob(job)
	if err != nil {
		log.Warnf("GetPodsForJob error %v", err)
		return err
	}

	services, err := jc.Controller.GetServicesForJob(job)
	if err != nil {
		log.Warnf("GetServicesForJob error %v", err)
		return err
	}

	oldStatus := jobStatus.DeepCopy()
	if commonutil.IsSucceeded(jobStatus) || commonutil.IsFailed(jobStatus) {
		// If the Job is succeed or failed, delete all pods and services.
		if err := jc.DeletePodsAndServices(runPolicy, job, pods); err != nil {
			return err
		}

		if jc.Config.EnableGangScheduling() {
			jc.Recorder.Event(runtimeObject, v1.EventTypeNormal, "JobTerminated", "Job has been terminated. Deleting PodGroup")
			if err := jc.DeletePodGroup(metaObject); err != nil {
				jc.Recorder.Eventf(runtimeObject, v1.EventTypeWarning, "FailedDeletePodGroup", "Error deleting: %v", err)
				return err
			} else {
				jc.Recorder.Eventf(runtimeObject, v1.EventTypeNormal, "SuccessfulDeletePodGroup", "Deleted PodGroup: %v", jobName)
			}
		}

		if err := jc.CleanupJob(runPolicy, jobStatus, job); err != nil {
			return err
		}

		// At this point the pods may have been deleted.
		// 1) If the job succeeded, we manually set the replica status.
		// 2) If any replicas are still active, set their status to succeeded.
		if commonutil.IsSucceeded(jobStatus) {
			for rtype := range jobStatus.ReplicaStatuses {
				jobStatus.ReplicaStatuses[rtype].Succeeded += jobStatus.ReplicaStatuses[rtype].Active
				jobStatus.ReplicaStatuses[rtype].Active = 0
			}
		}

		// No need to update the job status if the status hasn't changed since last time.
		if !reflect.DeepEqual(*oldStatus, jobStatus) {
			return jc.Controller.UpdateJobStatusInApiServer(job, &jobStatus)
		}

		return nil
	}

	// retrieve the previous number of retry
	previousRetry := jc.WorkQueue.NumRequeues(jobKey)

	activePods := k8sutil.FilterActivePods(pods)

	jc.recordAbnormalPods(activePods, runtimeObject)

	active := int32(len(activePods))
	failed := k8sutil.FilterPodCount(pods, v1.PodFailed)
	totalReplicas := k8sutil.GetTotalReplicas(replicas)
	prevReplicasFailedNum := k8sutil.GetTotalFailedReplicas(jobStatus.ReplicaStatuses)

	var failureMessage string
	jobExceedsLimit := false
	exceedsBackoffLimit := false
	pastBackoffLimit := false

	if runPolicy.BackoffLimit != nil {
		jobHasNewFailure := failed > prevReplicasFailedNum
		// new failures happen when status does not reflect the failures and active
		// is different than parallelism, otherwise the previous controller loop
		// failed updating status so even if we pick up failure it is not a new one
		exceedsBackoffLimit = jobHasNewFailure && (active != totalReplicas) &&
			(int32(previousRetry)+1 > *runPolicy.BackoffLimit)

		pastBackoffLimit, err = jc.PastBackoffLimit(jobName, runPolicy, replicas, pods)
		if err != nil {
			return err
		}
	}

	if exceedsBackoffLimit || pastBackoffLimit {
		// check if the number of pod restart exceeds backoff (for restart OnFailure only)
		// OR if the number of failed jobs increased since the last syncJob
		jobExceedsLimit = true
		failureMessage = fmt.Sprintf("Job %s has failed because it has reached the specified backoff limit", jobName)
	} else if jc.PastActiveDeadline(runPolicy, jobStatus) {
		failureMessage = fmt.Sprintf("Job %s has failed because it was active longer than specified deadline", jobName)
		jobExceedsLimit = true
	}

	if jobExceedsLimit {
		// Set job completion time before resource cleanup
		if jobStatus.CompletionTime == nil {
			now := metav1.Now()
			jobStatus.CompletionTime = &now
		}

		// If the Job exceeds backoff limit or is past active deadline
		// delete all pods and services, then set the status to failed
		if err := jc.DeletePodsAndServices(runPolicy, job, pods); err != nil {
			return err
		}

		if err := jc.CleanupJob(runPolicy, jobStatus, job); err != nil {
			return err
		}

		if jc.Config.EnableGangScheduling() {
			jc.Recorder.Event(runtimeObject, v1.EventTypeNormal, "JobTerminated", "Job has been terminated. Deleting PodGroup")
			if err := jc.DeletePodGroup(metaObject); err != nil {
				jc.Recorder.Eventf(runtimeObject, v1.EventTypeWarning, "FailedDeletePodGroup", "Error deleting: %v", err)
				return err
			} else {
				jc.Recorder.Eventf(runtimeObject, v1.EventTypeNormal, "SuccessfulDeletePodGroup", "Deleted PodGroup: %v", jobName)
			}
		}

		jc.Recorder.Event(runtimeObject, v1.EventTypeNormal, commonutil.JobFailedReason, failureMessage)

		if err := commonutil.UpdateJobConditions(&jobStatus, apiv1.JobFailed, commonutil.JobFailedReason, failureMessage); err != nil {
			log.Infof("Append job condition error: %v", err)
			return err
		}

		return jc.Controller.UpdateJobStatusInApiServer(job, &jobStatus)
	} else {
		// General cases which need to reconcile
		if jc.Config.EnableGangScheduling() {
			minMember := totalReplicas
			queue := ""
			priorityClass := ""
			var minResources *v1.ResourceList

			if runPolicy.SchedulingPolicy != nil {
				if runPolicy.SchedulingPolicy.MinAvailable != nil {
					minMember = *runPolicy.SchedulingPolicy.MinAvailable
				}

				if runPolicy.SchedulingPolicy.Queue != "" {
					queue = runPolicy.SchedulingPolicy.Queue
				}

				if runPolicy.SchedulingPolicy.PriorityClass != "" {
					priorityClass = runPolicy.SchedulingPolicy.PriorityClass
				}

				if runPolicy.SchedulingPolicy.MinResources != nil {
					minResources = runPolicy.SchedulingPolicy.MinResources
				}
			}

			if minResources == nil {
				minResources = jc.calcPGMinResources(minMember, replicas)
			}

			var pgSpecFill FillPodGroupSpecFunc = nil
			switch jc.Config.GangScheduling {
			case GangSchedulerVolcano:
				pgSpecFill = func(pg metav1.Object) error {
					volcanoPodGroup := pg.(*v1beta1.PodGroup)
					volcanoPodGroup.Spec = v1beta1.PodGroupSpec{
						MinMember:         minMember,
						Queue:             queue,
						PriorityClassName: priorityClass,
						MinResources:      minResources,
					}
					pg = volcanoPodGroup
					return nil
				}
			case GangSchedulerSchedulerPlugins:
				pgSpecFill = func(pg metav1.Object) error {
					schedulerPluginsPodGroup := pg.(*v1alpha1.PodGroup)
					schedulerPluginsPodGroup.Spec = v1alpha1.PodGroupSpec{
						MinMember:              minMember,
						MinResources:           minResources,
						ScheduleTimeoutSeconds: nil,
					}
					pg = schedulerPluginsPodGroup
					return nil
				}
			}

			syncReplicas := true
			pg, err := jc.SyncPodGroup(metaObject, pgSpecFill)
			if err != nil {
				log.Warnf("Sync PodGroup %v: %v", jobKey, err)
				syncReplicas = false
			}

			// Delay pods creation until PodGroup status is Inqueue
			if jc.PodGroupControl.DelayPodCreationDueToPodGroup(pg) {
				log.Warnf("PodGroup %v unschedulable", jobKey)
				syncReplicas = false
			}

			if !syncReplicas {
				now := metav1.Now()
				jobStatus.LastReconcileTime = &now

				// Update job status here to trigger a new reconciliation
				return jc.Controller.UpdateJobStatusInApiServer(job, &jobStatus)
			}
		}

		// Diff current active pods/services with replicas.
		for rtype, spec := range replicas {
			err := jc.Controller.ReconcilePods(metaObject, &jobStatus, pods, rtype, spec, replicas)
			if err != nil {
				log.Warnf("ReconcilePods error %v", err)
				return err
			}

			err = jc.Controller.ReconcileServices(metaObject, services, rtype, spec)

			if err != nil {
				log.Warnf("ReconcileServices error %v", err)
				return err
			}
		}
	}

	err = jc.Controller.UpdateJobStatus(job, replicas, &jobStatus)
	if err != nil {
		log.Warnf("UpdateJobStatus error %v", err)
		return err
	}
	// No need to update the job status if the status hasn't changed since last time.
	if !reflect.DeepEqual(*oldStatus, jobStatus) {
		return jc.Controller.UpdateJobStatusInApiServer(job, &jobStatus)
	}
	return nil
}

// ResetExpectations reset the expectation for creates and deletes of pod/service to zero.
func (jc *JobController) ResetExpectations(jobKey string, replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec) {
	for rtype := range replicas {
		expectationPodsKey := expectation.GenExpectationPodsKey(jobKey, string(rtype))
		jc.Expectations.SetExpectations(expectationPodsKey, 0, 0)
		expectationServicesKey := expectation.GenExpectationServicesKey(jobKey, string(rtype))
		jc.Expectations.SetExpectations(expectationServicesKey, 0, 0)
	}
}

// PastActiveDeadline checks if job has ActiveDeadlineSeconds field set and if it is exceeded.
func (jc *JobController) PastActiveDeadline(runPolicy *apiv1.RunPolicy, jobStatus apiv1.JobStatus) bool {
	return core.PastActiveDeadline(runPolicy, jobStatus)
}

// PastBackoffLimit checks if container restartCounts sum exceeds BackoffLimit
// this method applies only to pods when restartPolicy is one of OnFailure, Always or ExitCode
func (jc *JobController) PastBackoffLimit(jobName string, runPolicy *apiv1.RunPolicy,
	replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec, pods []*v1.Pod) (bool, error) {
	return core.PastBackoffLimit(jobName, runPolicy, replicas, pods, jc.FilterPodsForReplicaType)
}

func (jc *JobController) CleanupJob(runPolicy *apiv1.RunPolicy, jobStatus apiv1.JobStatus, job interface{}) error {
	currentTime := time.Now()
	metaObject, _ := job.(metav1.Object)
	ttl := runPolicy.TTLSecondsAfterFinished
	if ttl == nil {
		return nil
	}
	duration := time.Second * time.Duration(*ttl)
	if jobStatus.CompletionTime == nil {
		return fmt.Errorf("job completion time is nil, cannot cleanup")
	}
	finishTime := jobStatus.CompletionTime
	expireTime := finishTime.Add(duration)
	if currentTime.After(expireTime) {
		err := jc.Controller.DeleteJob(job)
		if err != nil {
			commonutil.LoggerForJob(metaObject).Warnf("Cleanup Job error: %v.", err)
			return err
		}
		return nil
	} else {
		if finishTime.After(currentTime) {
			commonutil.LoggerForJob(metaObject).Warnf("Found Job finished in the future. This is likely due to time skew in the cluster. Job cleanup will be deferred.")
		}
		remaining := expireTime.Sub(currentTime)
		key, err := KeyFunc(job)
		if err != nil {
			commonutil.LoggerForJob(metaObject).Warnf("Couldn't get key for job object: %v", err)
			return err
		}
		jc.WorkQueue.AddAfter(key, remaining)
		return nil
	}
}

func (jc *JobController) calcPGMinResources(minMember int32, replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec) *v1.ResourceList {
	return CalcPGMinResources(minMember, replicas, jc.PriorityClassLister.Get)
}
