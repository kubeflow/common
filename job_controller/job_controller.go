package job_controller

import (
	"errors"
	"fmt"
	"strings"

	commonv1 "github.com/kubeflow/common/operator/v1"
	"github.com/kubeflow/common/util"
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

	// Returns the Group Name(value) in the labels of the job
	GetGroupNameLabelValue() string

	// Returns the Job from Informer Cache
	GetJobFromInformerCache(namespace, name string) (metav1.Object, error)

	// Returns the Job from API server
	GetJobFromAPIClient(namespace, name string) (metav1.Object, error)

	// GetPodsForJob returns the pods managed by the job. This can be achieved by selecting pods using label key "job-name"
	// i.e. all pods created by the job will come with label "job-name" = <this_job_name>
	GetPodsForJob(job interface{}) ([]*v1.Pod, error)

	// GetServicesForJob returns the services managed by the job. This can be achieved by selecting services using label key "job-name"
	// i.e. all services created by the job will come with label "job-name" = <this_job_name>
	GetServicesForJob(job interface{}) ([]*v1.Service, error)

	// DeleteJob deletes the job
	DeleteJob(job interface{}) error

	// UpdateJobStatus updates the job status and job conditions
	UpdateJobStatus(job interface{}, replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, jobStatus commonv1.JobStatus, restart bool) error

	// UpdateJobStatusInApiServer updates the job status in API server
	UpdateJobStatusInApiServer(job interface{}, jobStatus *commonv1.JobStatus) error

	// CreateService creates the service
	CreateService(job interface{}, service *v1.Service) error

	// DeleteService deletes the service
	DeleteService(job interface{}, name string, namespace string) error

	// CreatePod creates the pod
	CreatePod(job interface{}, pod *v1.Pod) error

	// DeletePod deletes the pod
	DeletePod(job interface{}, pod *v1.Pod) error

	// SetClusterSpec sets the cluster spec for the pod
	SetClusterSpec(job interface{}, podTemplate *v1.PodTemplateSpec, rtype, index string) error

	// Returns the default container name in pod
	GetDefaultContainerName() string

	// Get the default container port number
	GetDefaultContainerPortNumber() string

	// Returns if this replica type with index specified is a master role.
	// MasterRole pod will have "job-role=master" set in its label
	IsMasterRole(replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, rtype commonv1.ReplicaType, index int) bool
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

	// PodControl is used to add or delete pods.
	PodControl controller.PodControlInterface

	// ServiceControl is used to add or delete services.
	ServiceControl ServiceControlInterface

	// KubeClientSet is a standard kubernetes clientset.
	KubeClientSet kubeclientset.Interface

	// KubeBatchClientSet is a standard kube-batch clientset.
	KubeBatchClientSet kubebatchclient.Interface

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

	jobControllerConfig := JobControllerConfiguration{
		ReconcilerSyncLoopPeriod: reconcilerSyncPeriod,
		EnableGangScheduling:     enableGangScheduling,
	}

	jc := JobController{
		Controller:         controllerImpl,
		Config:             jobControllerConfig,
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
	labelGroupName := util.LabelGroupName
	labelJobName := util.LabelJobName
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
	labelJobName := util.LabelJobName

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
