package job_controller

import (
	commonv1 "github.com/kubeflow/common/operator/v1"
	testv1 "github.com/kubeflow/common/test_job/v1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type TestJobController struct {
	job            *testv1.TestJob
	pods           []*corev1.Pod
	services       []*corev1.Service
}

func (TestJobController) ControllerName() string {
	return "test-operator"
}

func (TestJobController) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return testv1.SchemeGroupVersionKind
}

func (TestJobController) GetAPIGroupVersion() schema.GroupVersion {
	return testv1.SchemeGroupVersion
}

func (TestJobController) GetGroupNameLabelKey() string {
	return LabelGroupName
}

func (TestJobController) GetJobNameLabelKey() string {
	return LabelJobName
}

func (TestJobController) GetGroupNameLabelValue() string {
	return testv1.GroupName
}

func (TestJobController) GetReplicaTypeLabelKey() string {
	return ReplicaTypeLabel
}

func (TestJobController) GetReplicaIndexLabelKey() string {
	return ReplicaIndexLabel
}

func (TestJobController) GetJobRoleKey() string {
	return LabelJobRole
}

func (t *TestJobController) GetJobFromInformerCache(namespace, name string) (v1.Object, error) {
	return t.job, nil
}

func (t *TestJobController) GetJobFromAPIClient(namespace, name string) (v1.Object, error) {
	return t.job, nil
}

func (t *TestJobController) DeleteJob(job interface{}) error {
	log.Info("Delete job")
	t.job = nil
	return nil
}

func (t *TestJobController) UpdateJobStatus(job interface{}, replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec,
	jobStatus commonv1.JobStatus) error {
	return nil
}

func (t *TestJobController) CreateService(job interface{}, service *corev1.Service) error {
	return nil
}

func (t *TestJobController) DeleteService(job interface{}, name string, namespace string) error {
	log.Info("Deleting service " + name)
	var remainingServices []*corev1.Service
	for _, tservice:= range t.services{
		if tservice.Name != name {
			remainingServices = append(remainingServices, tservice)
		}
	}
	t.services = remainingServices
	return nil
}

func (t *TestJobController) CreatePod(job interface{}, podTemplate *corev1.PodTemplate) error {
	return nil
}

func (t *TestJobController) DeletePod(job interface{}, pod *corev1.Pod) error {
	log.Info("Deleting pod " + pod.Name)
	var remainingPods []*corev1.Pod
	for _, tpod := range t.pods {
		if tpod.Name != pod.Name {
			remainingPods = append(remainingPods, tpod)
		}
	}
	t.pods = remainingPods
	return nil
}

func (t *TestJobController) SetClusterSpec(job interface{}, podTemplate *corev1.PodTemplate, rtype, index string) error {
	return nil
}
