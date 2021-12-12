package test_job

import (
	"context"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	common_reconciler "github.com/kubeflow/common/pkg/reconciler.v1/common"
	v1 "github.com/kubeflow/common/test_job/apis/test_job/v1"
	"github.com/kubeflow/common/test_job/client/clientset/versioned/scheme"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type TestReconciler struct {
	common_reconciler.ReconcilerUtil
	common_reconciler.ServiceReconciler
	common_reconciler.PodReconciler
	common_reconciler.VolcanoReconciler
	common_reconciler.JobReconciler

	DC       *DummyClient
	Job      *v1.TestJob
	Pods     []*corev1.Pod
	Services []*corev1.Service
	PodGroup client.Object
}

func NewTestReconciler() *TestReconciler {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1.AddToScheme(scheme))

	dummy_client := &DummyClient{}

	r := &TestReconciler{
		DC: dummy_client,
	}

	// Generate Bare Components
	jobR := common_reconciler.BareJobReconciler(dummy_client)
	jobR.OverrideForJobInterface(r, r, r, r)

	podR := common_reconciler.BarePodReconciler(dummy_client)
	podR.OverrideForPodInterface(r, r, r)

	svcR := common_reconciler.BareServiceReconciler(dummy_client)
	svcR.OverrideForServiceInterface(r, r, r)

	gangR := common_reconciler.BareVolcanoReconciler(dummy_client, nil, false)
	gangR.OverrideForGangSchedulingInterface(r)

	Log := log.Log
	utilR := common_reconciler.BareUtilReconciler(nil, Log, scheme)
	//kubeflowReconciler := common_reconciler.BareKubeflowReconciler()

	r.JobReconciler = *jobR
	r.PodReconciler = *podR
	r.ServiceReconciler = *svcR
	r.VolcanoReconciler = *gangR
	r.ReconcilerUtil = *utilR

	return r
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *TestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	job, err := r.GetJob(ctx, req)
	if err != nil {
		return ctrl.Result{}, err
	}

	logger := r.GetLogger(job)

	if job.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, nil
	}

	scheme.Scheme.Default(job)

	// Get rid of SatisfiedExpectation
	replicasSpec, err := r.ExtractReplicasSpec(job)
	if err != nil {
		return ctrl.Result{}, err
	}

	runPolicy, err := r.ExtractRunPolicy(job)
	if err != nil {
		return ctrl.Result{}, err
	}

	status, err := r.ExtractJobStatus(job)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.ReconcileJob(ctx, job, replicasSpec, status, runPolicy)
	if err != nil {
		logger.Info("Reconcile Test Job error %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *TestReconciler) GetReconcilerName() string {
	return "Test Reconciler"
}

func (r *TestReconciler) GetJob(ctx context.Context, req ctrl.Request) (client.Object, error) {
	return r.Job, nil
}

func (r *TestReconciler) GetDefaultContainerName() string {
	return v1.DefaultContainerName
}

func (r *TestReconciler) GetPodGroupForJob(ctx context.Context, job client.Object) (client.Object, error) {
	return r.PodGroup, nil
}

func (r *TestReconciler) GetPodsForJob(ctx context.Context, job client.Object) ([]*corev1.Pod, error) {
	return r.Pods, nil
}

func (r *TestReconciler) GetServicesForJob(ctx context.Context, job client.Object) ([]*corev1.Service, error) {
	return r.Services, nil
}

func (r *TestReconciler) ExtractReplicasSpec(job client.Object) (map[commonv1.ReplicaType]*commonv1.ReplicaSpec, error) {
	tj := job.(*v1.TestJob)

	rs := map[commonv1.ReplicaType]*commonv1.ReplicaSpec{}
	for k, v := range tj.Spec.TestReplicaSpecs {
		rs[commonv1.ReplicaType(k)] = v
	}

	return rs, nil
}

func (r *TestReconciler) ExtractRunPolicy(job client.Object) (*commonv1.RunPolicy, error) {
	tj := job.(*v1.TestJob)

	return tj.Spec.RunPolicy, nil
}

func (r *TestReconciler) ExtractJobStatus(job client.Object) (*commonv1.JobStatus, error) {
	tj := job.(*v1.TestJob)

	return &tj.Status, nil
}

func (r *TestReconciler) IsMasterRole(replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, rtype commonv1.ReplicaType, index int) bool {
	return string(rtype) == string(v1.TestReplicaTypeMaster)
}
