package test_job

import (
	"context"

	"github.com/go-logr/logr"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	common_reconciler "github.com/kubeflow/common/pkg/reconciler.v1/common"
	v1 "github.com/kubeflow/common/test_job/apis/test_job/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ common_reconciler.KubeflowReconcilerInterface = &TestReconciler{}

type TestReconciler struct {
	common_reconciler.KubeflowReconciler
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

	kubeflowReconciler := common_reconciler.BareKubeflowReconciler()

	dummy_client := &DummyClient{}

	// Generate Bare Components
	jobInter := common_reconciler.BareKubeflowJobReconciler(dummy_client)
	podInter := common_reconciler.BareKubeflowPodReconciler(dummy_client)
	svcInter := common_reconciler.BareKubeflowServiceReconciler(dummy_client)
	gangInter := common_reconciler.BareVolcanoReconciler(dummy_client, nil, true)
	utilInter := common_reconciler.BareUtilReconciler(nil, logr.FromContext(context.Background()), scheme)

	// Assign interfaces for jobInterface
	jobInter.PodInterface = podInter
	jobInter.ServiceInterface = svcInter
	jobInter.GangSchedulingInterface = gangInter
	jobInter.ReconcilerUtilInterface = utilInter

	// Assign interfaces for podInterface
	podInter.JobInterface = jobInter
	podInter.GangSchedulingInterface = gangInter
	podInter.ReconcilerUtilInterface = utilInter

	// Assign interfaces for svcInterface
	svcInter.PodInterface = podInter
	svcInter.JobInterface = jobInter
	svcInter.ReconcilerUtilInterface = utilInter

	// Assign interfaces for gangInterface
	gangInter.ReconcilerUtilInterface = utilInter

	// Prepare KubeflowReconciler
	kubeflowReconciler.JobInterface = jobInter
	kubeflowReconciler.PodInterface = podInter
	kubeflowReconciler.ServiceInterface = svcInter
	kubeflowReconciler.GangSchedulingInterface = gangInter
	kubeflowReconciler.ReconcilerUtilInterface = utilInter

	testReconciler := &TestReconciler{
		KubeflowReconciler: *kubeflowReconciler,
		DC:                 dummy_client,
	}
	testReconciler.OverrideForKubeflowReconcilerInterface(testReconciler, testReconciler, testReconciler, testReconciler, testReconciler)

	return testReconciler
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
