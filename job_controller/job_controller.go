package job_controller

import (
	"errors"
	"fmt"
	commonv1 "github.com/kubeflow/common/operator/v1"
	"github.com/kubeflow/common/util/k8sutil"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"reflect"
	"strings"
	"time"

	"github.com/kubernetes-sigs/kube-batch/pkg/apis/scheduling/v1alpha1"
	kubebatchclient "github.com/kubernetes-sigs/kube-batch/pkg/client/clientset/versioned"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	kubeinformers "k8s.io/client-go/informers"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/kubernetes/pkg/controller"
)

var (
	// KeyFunc is the short name to DeletionHandlingMetaNamespaceKeyFunc.
	// IndexerInformer uses a delta queue, therefore for deletes we have to use this
	// key function but it should be just fine for non delete events.
	KeyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc
)

// Common Interface to be implemented by all operators.
type ControllerInterface interface {

	// Returns the Controller name
	ControllerName() string

	// Returns the GroupVersionKind of the API
	GetAPIGroupVersionKind() schema.GroupVersionKind

	// Returns the GroupVersion of the API
	GetAPIGroupVersion() schema.GroupVersion

	// Returns the Group Name(key) in the labels of the job
	GetGroupNameLabelKey() string

	// Returns the Job Name(key) in the labels of the job
	GetJobNameLabelKey() string

	// Returns the Group Name(value) in the labels of the job
	GetGroupNameLabelValue() string

	// Returns the Replica Type(key) in the labels of the job
	GetReplicaTypeLabelKey() string

	// Returns the Replica Index(value) in the labels of the job
	GetReplicaIndexLabelKey() string

	// Returns the Job Role(key) in the labels of the job
	GetJobRoleKey() string

	// Returns the Job from Informer Cache
	GetJobFromInformerCache(namespace, name string) (metav1.Object, error)

	// Returns the Job from API server
	GetJobFromAPIClient(namespace, name string) (metav1.Object, error)
}

// JobControllerConfiguration contains configuration of operator.
type JobControllerConfiguration struct {
	// ReconcilerSyncLoopPeriod is the amount of time the reconciler sync states loop
	// wait between two reconciler sync.
	// It is set to 15 sec by default.
	// TODO(cph): maybe we can let it grows by multiple in the future
	// and up to 5 minutes to reduce idle loop.
	// e.g. 15s, 30s, 60s, 120s...
	ReconcilerSyncLoopPeriod metav1.Duration

	// Enable gang scheduling by kube-batch
	EnableGangScheduling bool
}

// JobController abstracts other operators to manage the lifecycle of Jobs.
type JobController struct {
	Controller ControllerInterface

	Config JobControllerConfiguration

	// podControl is used to add or delete pods.
	PodControl controller.PodControlInterface

	// serviceControl is used to add or delete services.
	ServiceControl ServiceControlInterface

	// kubeClientSet is a standard kubernetes clientset.
	KubeClientSet kubeclientset.Interface

	//KubeBatchClientSet is a standard kube-batch clientset.
	KubeBatchClientSet kubebatchclient.Interface

	// podLister can list/get pods from the shared informer's store.
	PodLister corelisters.PodLister

	// serviceLister can list/get services from the shared informer's store.
	ServiceLister corelisters.ServiceLister

	// podInformerSynced returns true if the pod store has been synced at least once.
	PodInformerSynced cache.InformerSynced

	// serviceInformerSynced returns true if the service store has been synced at least once.
	ServiceInformerSynced cache.InformerSynced

	// A TTLCache of pod/services creates/deletes each job expects to see
	// We use Job namespace/name + ReplicaType + pods/services as an expectation key,
	// For example, there is a TFJob with namespace "tf-operator" and name "tfjob-abc":
	// {
	//     "PS": {
	//         "Replicas": 2,
	//     },
	//     "Worker": {
	//         "Replicas": 4,
	//     }
	// }
	// We will create 4 expectations:
	// - "tf-operator/tfjob-abc/ps/services", expects 2 adds.
	// - "tf-operator/tfjob-abc/ps/pods", expects 2 adds.
	// - "tf-operator/tfjob-abc/worker/services", expects 4 adds.
	// - "tf-operator/tfjob-abc/worker/pods", expects 4 adds.
	Expectations controller.ControllerExpectationsInterface

	// workQueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	WorkQueue workqueue.RateLimitingInterface

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	Recorder record.EventRecorder

	// deleteJobHandler deletes the job
	deleteJobHandler func(job interface{}) error

	// updateJobStatusHandler updates the job status
	updateJobStatusHandler func(job interface{}) error

	// createServiceHandler creates the service
	createServiceHandler func(job interface{}, rtype commonv1.ReplicaType, spec *commonv1.ReplicaSpec, index string) error

	// reconcilePods reconciles the pods for the job
	reconcilePods func(
		job metav1.Object,
		pods []*v1.Pod,
		rtype commonv1.ReplicaType,
		spec *commonv1.ReplicaSpec, rstatus map[string]v1.PodPhase) error
}

func NewJobController(
	controllerImpl ControllerInterface,
	reconcilerSyncPeriod metav1.Duration,
	enableGangScheduling bool,
	kubeClientSet kubeclientset.Interface,
	kubeBatchClientSet kubebatchclient.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	workQueueName string) JobController {

	log.Debug("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(log.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClientSet.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: controllerImpl.ControllerName()})

	realPodControl := RealPodControl{
		KubeClient: kubeClientSet,
		Recorder:   eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: controllerImpl.ControllerName()}),
	}

	realServiceControl := RealServiceControl{
		KubeClient: kubeClientSet,
		Recorder:   eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: controllerImpl.ControllerName()}),
	}

	jobControllerConfig := JobControllerConfiguration{
		ReconcilerSyncLoopPeriod: reconcilerSyncPeriod,
		EnableGangScheduling:     enableGangScheduling,
	}

	jc := JobController{
		Controller:         controllerImpl,
		Config:             jobControllerConfig,
		PodControl:         realPodControl,
		ServiceControl:     realServiceControl,
		KubeClientSet:      kubeClientSet,
		KubeBatchClientSet: kubeBatchClientSet,
		Expectations:       controller.NewControllerExpectations(),
		WorkQueue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), workQueueName),
		Recorder:           recorder,
	}
	return jc

}

func (jc *JobController) GenOwnerReference(obj metav1.Object) *metav1.OwnerReference {
	boolPtr := func(b bool) *bool { return &b }
	controllerRef := &metav1.OwnerReference{
		APIVersion:         jc.Controller.GetAPIGroupVersion().String(),
		Kind:               jc.Controller.GetAPIGroupVersionKind().Kind,
		Name:               obj.GetName(),
		UID:                obj.GetUID(),
		BlockOwnerDeletion: boolPtr(true),
		Controller:         boolPtr(true),
	}

	return controllerRef
}

func (jc *JobController) GenLabels(jobName string) map[string]string {
	labelGroupName := jc.Controller.GetGroupNameLabelKey()
	labelJobName := jc.Controller.GetJobNameLabelKey()
	groupName := jc.Controller.GetGroupNameLabelValue()
	return map[string]string{
		labelGroupName: groupName,
		labelJobName:   strings.Replace(jobName, "/", "-", -1),
	}
}

func (jc *JobController) SyncPodGroup(job metav1.Object, minAvailableReplicas int32) (*v1alpha1.PodGroup, error) {

	kubeBatchClientInterface := jc.KubeBatchClientSet
	// Check whether podGroup exists or not
	podGroup, err := kubeBatchClientInterface.SchedulingV1alpha1().PodGroups(job.GetNamespace()).Get(job.GetName(), metav1.GetOptions{})
	if err == nil {
		return podGroup, nil
	}

	// create podGroup for gang scheduling by kube-batch
	minAvailable := intstr.FromInt(int(minAvailableReplicas))
	createPodGroup := &v1alpha1.PodGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: job.GetName(),
			OwnerReferences: []metav1.OwnerReference{
				*jc.GenOwnerReference(job),
			},
		},
		Spec: v1alpha1.PodGroupSpec{
			MinMember: minAvailable.IntVal,
		},
	}
	return kubeBatchClientInterface.SchedulingV1alpha1().PodGroups(job.GetNamespace()).Create(createPodGroup)
}

// SyncPdb will create a PDB for gang scheduling by kube-batch.
func (jc *JobController) SyncPdb(job metav1.Object, minAvailableReplicas int32) (*v1beta1.PodDisruptionBudget, error) {
	labelJobName := jc.Controller.GetJobNameLabelKey()

	// Check the pdb exist or not
	pdb, err := jc.KubeClientSet.PolicyV1beta1().PodDisruptionBudgets(job.GetNamespace()).Get(job.GetName(), metav1.GetOptions{})
	if err == nil || !k8serrors.IsNotFound(err) {
		if err == nil {
			err = errors.New(string(metav1.StatusReasonAlreadyExists))
		}
		return pdb, err
	}

	// Create pdb for gang scheduling by kube-batch
	minAvailable := intstr.FromInt(int(minAvailableReplicas))
	createPdb := &v1beta1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name: job.GetName(),
			OwnerReferences: []metav1.OwnerReference{
				*jc.GenOwnerReference(job),
			},
		},
		Spec: v1beta1.PodDisruptionBudgetSpec{
			MinAvailable: &minAvailable,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					labelJobName: job.GetName(),
				},
			},
		},
	}
	return jc.KubeClientSet.PolicyV1beta1().PodDisruptionBudgets(job.GetNamespace()).Create(createPdb)
}

func (jc *JobController) DeletePodGroup(job metav1.Object) error {
	kubeBatchClientInterface := jc.KubeBatchClientSet

	//check whether podGroup exists or not
	_, err := kubeBatchClientInterface.SchedulingV1alpha1().PodGroups(job.GetNamespace()).Get(job.GetName(), metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		return nil
	}

	log.Infof("Deleting PodGroup %s", job.GetName())

	//delete podGroup
	err = kubeBatchClientInterface.SchedulingV1alpha1().PodGroups(job.GetNamespace()).Delete(job.GetName(), &metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("unable to delete PodGroup: %v", err)
	}
	return nil
}

func (jc *JobController) DeletePdb(job metav1.Object) error {

	// Check the pdb exist or not
	_, err := jc.KubeClientSet.PolicyV1beta1().PodDisruptionBudgets(job.GetNamespace()).Get(job.GetName(), metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		return nil
	}

	msg := fmt.Sprintf("Deleting pdb %s", job.GetName())
	log.Info(msg)

	if err := jc.KubeClientSet.PolicyV1beta1().PodDisruptionBudgets(job.GetNamespace()).Delete(job.GetName(), &metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("unable to delete pdb: %v", err)
	}
	return nil
}

// resolveControllerRef returns the job referenced by a ControllerRef,
// or nil if the ControllerRef could not be resolved to a matching job
// of the correct Kind.
func (jc *JobController) resolveControllerRef(namespace string, controllerRef *metav1.OwnerReference) metav1.Object {
	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it's the wrong Kind.
	if controllerRef.Kind != jc.Controller.GetAPIGroupVersionKind().Kind {
		return nil
	}
	job, err := jc.Controller.GetJobFromInformerCache(namespace, controllerRef.Name)
	if err != nil {
		return nil
	}
	if job.GetUID() != controllerRef.UID {
		// The controller we found with this Name is not the same one that the
		// ControllerRef points to.
		return nil
	}
	return job
}

// reconcileJobs checks and updates replicas for each given ReplicaSpec.
// It will requeue the job in case of an error while creating/deleting pods/services.
func (jc *JobController) reconcileJobs(job interface{}, replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec,
	jobStatus commonv1.JobStatus, runPolicy *commonv1.RunPolicy) error {
	metaObject, ok := job.(metav1.Object)
	jobName := metaObject.GetName()
	if !ok {
		return fmt.Errorf("job is not a metav1.Object type")
	}
	runtimeObject, ok := job.(runtime.Object)
	if !ok {
		return fmt.Errorf("job is not a runtime.Object type")
	}
	jobKey, err := KeyFunc(job)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for job object %#v: %v", job, err))
		return err
	}
	log.Infof("Reconcile Jobs %s", metaObject.GetName())

	oldStatus := jobStatus.DeepCopy()

	pods, err := jc.GetPodsForJob(metaObject)

	if err != nil {
		log.Warnf("getPodsForJob error %v", err)
		return err
	}

	services, err := jc.GetServicesForJob(metaObject)

	if err != nil {
		log.Warnf("getServicesForJob error %v", err)
		return err
	}

	// retrieve the previous number of retry
	previousRetry := jc.WorkQueue.NumRequeues(jobKey)

	activePods := k8sutil.FilterActivePods(pods)
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

		pastBackoffLimit, err = jc.pastBackoffLimit(jobName, runPolicy, replicas, pods)
		if err != nil {
			return err
		}
	}

	if exceedsBackoffLimit || pastBackoffLimit {
		// check if the number of pod restart exceeds backoff (for restart OnFailure only)
		// OR if the number of failed jobs increased since the last syncJob
		jobExceedsLimit = true
		failureMessage = fmt.Sprintf("Job %s has failed because it has reached the specified backoff limit", jobName)
	} else if jc.pastActiveDeadline(runPolicy, jobStatus) {
		failureMessage = fmt.Sprintf("Job %s has failed because it was active longer than specified deadline", jobName)
		jobExceedsLimit = true
	}

	// If the TFJob is terminated, delete all pods and services.
	if isSucceeded(jobStatus) || isFailed(jobStatus) || jobExceedsLimit {
		if err := jc.deletePodsAndServices(runPolicy, &runtimeObject, pods); err != nil {
			return err
		}

		if err := jc.cleanupJob(runPolicy, jobStatus, &runtimeObject); err != nil {
			return err
		}

		if jc.Config.EnableGangScheduling {
			jc.Recorder.Event(runtimeObject, v1.EventTypeNormal, "JobTerminated", "Job is terminated, deleting PodGroup")
			if err := jc.DeletePodGroup(metaObject); err != nil {
				jc.Recorder.Eventf(runtimeObject, v1.EventTypeWarning, "FailedDeletePodGroup", "Error deleting: %v", err)
				return err
			} else {
				jc.Recorder.Eventf(runtimeObject, v1.EventTypeNormal, "SuccessfulDeletePodGroup", "Deleted pdb: %v", jobName)

			}
		}

		if jobExceedsLimit {
			jc.Recorder.Event(runtimeObject, v1.EventTypeNormal, JobFailedReason, failureMessage)
			if jobStatus.CompletionTime == nil {
				now := metav1.Now()
				jobStatus.CompletionTime = &now
			}
			err := updateJobConditions(&jobStatus, commonv1.JobFailed, JobFailedReason, failureMessage)
			if err != nil {
				log.Infof("Append job condition error: %v", err)
				return err
			}
		}

		// At this point the pods may have been deleted, so if the job succeeded, we need to manually set the replica status.
		// If any replicas are still Active, set their status to succeeded.
		if isSucceeded(jobStatus) {
			for rtype := range jobStatus.ReplicaStatuses {
				jobStatus.ReplicaStatuses[rtype].Succeeded += jobStatus.ReplicaStatuses[rtype].Active
				jobStatus.ReplicaStatuses[rtype].Active = 0
			}
		}
		return jc.updateJobStatusHandler(job)
	}

	// Save the current state of the replicas
	replicasStatus := make(map[string]v1.PodPhase)

	// Diff current active pods/services with replicas.
	for rtype, spec := range replicas {
		err = jc.reconcilePods(metaObject, pods, rtype, spec, replicasStatus)
		if err != nil {
			log.Warnf("reconcilePods error %v", err)
			return err
		}

		err = jc.reconcileServices(metaObject, services, rtype, spec, jc.createServiceHandler)

		if err != nil {
			log.Warnf("reconcileServices error %v", err)
			return err
		}
	}

	// no need to update the tfjob if the status hasn't changed since last time.
	if !reflect.DeepEqual(*oldStatus, jobStatus) {
		return jc.updateJobStatusHandler(job)
	}
	return nil
}

// pastActiveDeadline checks if job has ActiveDeadlineSeconds field set and if it is exceeded.
func (jc *JobController) pastActiveDeadline(runPolicy *commonv1.RunPolicy, jobStatus commonv1.JobStatus) bool {
	if runPolicy.ActiveDeadlineSeconds == nil || jobStatus.StartTime == nil {
		return false
	}
	now := metav1.Now()
	start := jobStatus.StartTime.Time
	duration := now.Time.Sub(start)
	allowedDuration := time.Duration(*runPolicy.ActiveDeadlineSeconds) * time.Second
	return duration >= allowedDuration
}

// pastBackoffLimitOnFailure checks if container restartCounts sum exceeds BackoffLimit
// this method applies only to pods with restartPolicy == OnFailure or Always
func (jc *JobController) pastBackoffLimit(jobName string, runPolicy *commonv1.RunPolicy,
	replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, pods []*v1.Pod) (bool, error) {
	if runPolicy.BackoffLimit == nil {
		return false, nil
	}
	result := int32(0)
	for rtype, spec := range replicas {
		if spec.RestartPolicy != commonv1.RestartPolicyOnFailure && spec.RestartPolicy != commonv1.RestartPolicyAlways {
			log.Warnf("The restart policy of replica %v of the job %v is not OnFailure or Always. Not counted in backoff limit.", rtype, jobName)
			continue
		}
		// Convert ReplicaType to lower string.
		rt := strings.ToLower(string(rtype))
		pods, err := jc.FilterPodsForReplicaType(pods, rt)
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
