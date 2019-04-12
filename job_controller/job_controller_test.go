package job_controller

import (
	"testing"
	"time"

	commonv1 "github.com/kubeflow/common/operator/v1"
	testutilv1 "github.com/kubeflow/common/test_util/v1"
	kubebatchclient "github.com/kubernetes-sigs/kube-batch/pkg/client/clientset/versioned"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func TestReconcileJobs(t *testing.T) {
	kubeClientSet := kubeclientset.NewForConfigOrDie(&rest.Config{
		Host: "",
		ContentConfig: rest.ContentConfig{
			GroupVersion: &v1.SchemeGroupVersion,
		},
	},)
	kubeBatchClientSet := kubebatchclient.NewForConfigOrDie(&rest.Config{
		Host: "",
		ContentConfig: rest.ContentConfig{
			GroupVersion: &v1.SchemeGroupVersion,
		},
	}, )

	testJob := testutilv1.NewTestJob(2)

	tc := &TestJobController{
		job: testJob,
	}

	jc := NewJobController(tc, metav1.Duration{Duration: 15 * time.Second},
		false, kubeClientSet, kubeBatchClientSet, nil, "job")


	replicas := make(map[commonv1.ReplicaType]*commonv1.ReplicaSpec)
	for key, val := range testJob.Spec.TestReplicaSpecs {
		replicas[commonv1.ReplicaType(key)] = val
	}
	err := jc.reconcileJobs(testJob, replicas, testJob.Status, testJob.Spec.RunPolicy)
	if err != nil{
		t.Errorf("error in reconcileJobs %v", err)
	}
}