package common

import (
	"errors"
	"fmt"
	v1 "k8s.io/api/core/v1"
	kubeinformers "k8s.io/client-go/informers"
	kubeclientset "k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"strings"

	apiv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	log "github.com/sirupsen/logrus"
	policyapi "k8s.io/api/policy/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/cache"
	"volcano.sh/volcano/pkg/apis/scheduling/v1beta1"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/kubernetes/pkg/controller"
	volcanoclient "volcano.sh/volcano/pkg/client/clientset/versioned"
)

var (
	// KeyFunc is the short name to DeletionHandlingMetaNamespaceKeyFunc.
	// IndexerInformer uses a delta queue, therefore for deletes we have to use this
	// key function but it should be just fine for non delete events.
	KeyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc
)

// JobControllerConfiguration contains configuration of operator.
type JobControllerConfiguration struct {
	// ReconcilerSyncLoopPeriod is the amount of time the reconciler sync states loop
	// wait between two reconciler sync.
	// It is set to 15 sec by default.
	// TODO(cph): maybe we can let it grows by multiple in the future
	// and up to 5 minutes to reduce idle loop.
	// e.g. 15s, 30s, 60s, 120s...
	ReconcilerSyncLoopPeriod metav1.Duration

	// Enable gang scheduling by volcano
	EnableGangScheduling bool
}

// JobController abstracts other operators to manage the lifecycle of Jobs.
// User need to first implement the ControllerInterface(objectA) and then initialize a JobController(objectB) struct with objectA
// as the parameter.
// And then call objectB.ReconcileJobs as mentioned below, the ReconcileJobs method is the entrypoint to trigger the
// reconcile logic of the job controller
//
// ReconcileJobs(
//		job interface{},
//		replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec,
//		jobStatus apiv1.JobStatus,
//		runPolicy *apiv1.RunPolicy) error
type JobController struct {
	Controller apiv1.ControllerInterface

	Config JobControllerConfiguration

	// KubeClientSet is a standard kubernetes clientset.
	KubeClientSet kubeclientset.Interface

	// VolcanoClientSet is a standard volcano clientset.
	VolcanoClientSet volcanoclient.Interface

	// PodLister can list/get pods from the shared informer's store.
	PodLister corelisters.PodLister

	// ServiceLister can list/get services from the shared informer's store.
	ServiceLister corelisters.ServiceLister

	// PodInformerSynced returns true if the pod store has been synced at least once.
	PodInformerSynced cache.InformerSynced

	// ServiceInformerSynced returns true if the service store has been synced at least once.
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

	// WorkQueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	WorkQueue workqueue.RateLimitingInterface

	// Recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	Recorder record.EventRecorder
}

func NewJobController(
	controllerImpl apiv1.ControllerInterface,
	reconcilerSyncPeriod metav1.Duration,
	enableGangScheduling bool,
	kubeClientSet kubeclientset.Interface,
	volcanoClientSet volcanoclient.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	workQueueName string) JobController {

	log.Debug("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(log.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClientSet.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: controllerImpl.ControllerName()})

	jobControllerConfig := JobControllerConfiguration{
		ReconcilerSyncLoopPeriod: reconcilerSyncPeriod,
		EnableGangScheduling:     enableGangScheduling,
	}

	jc := JobController{
		Controller:       controllerImpl,
		Config:           jobControllerConfig,
		KubeClientSet:    kubeClientSet,
		VolcanoClientSet: volcanoClientSet,
		Expectations:     controller.NewControllerExpectations(),
		WorkQueue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), workQueueName),
		Recorder:         recorder,
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
	labelGroupName := apiv1.GroupNameLabel
	labelJobName := apiv1.JobNameLabel
	groupName := jc.Controller.GetGroupNameLabelValue()
	return map[string]string{
		labelGroupName: groupName,
		labelJobName:   strings.Replace(jobName, "/", "-", -1),
	}
}

func (jc *JobController) SyncPodGroup(job metav1.Object, minAvailableReplicas int32) (*v1beta1.PodGroup, error) {

	volcanoClientSet := jc.VolcanoClientSet
	// Check whether podGroup exists or not
	podGroup, err := volcanoClientSet.SchedulingV1beta1().PodGroups(job.GetNamespace()).Get(job.GetName(), metav1.GetOptions{})
	if err == nil {
		return podGroup, nil
	}

	// create podGroup for gang scheduling by volcano
	minAvailable := intstr.FromInt(int(minAvailableReplicas))
	createPodGroup := &v1beta1.PodGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: job.GetName(),
			OwnerReferences: []metav1.OwnerReference{
				*jc.GenOwnerReference(job),
			},
		},
		Spec: v1beta1.PodGroupSpec{
			MinMember: minAvailable.IntVal,
		},
	}
	return volcanoClientSet.SchedulingV1beta1().PodGroups(job.GetNamespace()).Create(createPodGroup)
}

// SyncPdb will create a PDB for gang scheduling by volcano.
func (jc *JobController) SyncPdb(job metav1.Object, minAvailableReplicas int32) (*policyapi.PodDisruptionBudget, error) {
	labelJobName := apiv1.JobNameLabel

	// Check the pdb exist or not
	pdb, err := jc.KubeClientSet.PolicyV1beta1().PodDisruptionBudgets(job.GetNamespace()).Get(job.GetName(), metav1.GetOptions{})
	if err == nil || !k8serrors.IsNotFound(err) {
		if err == nil {
			err = errors.New(string(metav1.StatusReasonAlreadyExists))
		}
		return pdb, err
	}

	// Create pdb for gang scheduling by volcano
	minAvailable := intstr.FromInt(int(minAvailableReplicas))
	createPdb := &policyapi.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name: job.GetName(),
			OwnerReferences: []metav1.OwnerReference{
				*jc.GenOwnerReference(job),
			},
		},
		Spec: policyapi.PodDisruptionBudgetSpec{
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
	volcanoClientSet := jc.VolcanoClientSet

	//check whether podGroup exists or not
	_, err := volcanoClientSet.SchedulingV1beta1().PodGroups(job.GetNamespace()).Get(job.GetName(), metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		return nil
	}

	log.Infof("Deleting PodGroup %s", job.GetName())

	//delete podGroup
	err = volcanoClientSet.SchedulingV1beta1().PodGroups(job.GetNamespace()).Delete(job.GetName(), &metav1.DeleteOptions{})
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
