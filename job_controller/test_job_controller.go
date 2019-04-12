package job_controller

import (
	commonv1 "github.com/kubeflow/common/operator/v1"
	testv1 "github.com/kubeflow/common/test_job/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type TestJobController struct {
	JobController
	job            *testv1.TestJob
	createdService *corev1.Service
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
	return "group-name"
}

func (TestJobController) GetJobNameLabelKey() string {
	return "test-replica-type"
}

func (TestJobController) GetGroupNameLabelValue() string {
	return testv1.GroupName
}

func (TestJobController) GetReplicaTypeLabelKey() string {
	return "test-replica-type"
}

func (TestJobController) GetReplicaIndexLabelKey() string {
	return "test-replica-index"
}

func (TestJobController) GetJobRoleKey() string {
	return "test-job-role"
}

func (t *TestJobController) GetJobFromInformerCache(namespace, name string) (v1.Object, error) {
	return t.job, nil
}

func (t *TestJobController) GetJobFromAPIClient(namespace, name string) (v1.Object, error) {
	return t.job, nil
}

func (t TestJobController) GetDefaultContainerName() string {
	panic("implement me")
}

func (t TestJobController) GetDefaultContainerPortName() string {
	panic("implement me")
}

func (t TestJobController) UpdateJobStatus(job interface{}, replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec,
	jobStatus *commonv1.JobStatus, restart bool) error {
	panic("implement me")
}

func (t TestJobController) UpdateJobStatusInApiServer(job interface{}) error {
	panic("implement me")
}

func (t TestJobController) CreateService(job interface{}, service *corev1.Service) error {
	panic("implement me")
}

func (t TestJobController) DeleteService(job interface{}, service *corev1.Service) error {
	panic("implement me")
}

func (t TestJobController) CreatePod(job interface{}, podTemplate *corev1.PodTemplateSpec) error {
	panic("implement me")
}

func (t TestJobController) DeletePod(job interface{}, pod *corev1.Pod) error {
	panic("implement me")
}

func (t TestJobController) SetClusterSpec(job interface{}, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	panic("implement me")
}

func (t TestJobController) IsMasterRole(replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, rtype commonv1.ReplicaType, index int) bool {
	panic("implement me")
}



