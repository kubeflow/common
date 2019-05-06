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

package job_controller

import (
	jobclientset "github.com/kubeflow/common/client/clientset/versioned"
	commonv1 "github.com/kubeflow/common/operator/v1"
	testv1 "github.com/kubeflow/common/test_job/v1"
	"github.com/kubeflow/common/util"
	kubebatchclient "github.com/kubernetes-sigs/kube-batch/pkg/client/clientset/versioned"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kubeinformers "k8s.io/client-go/informers"
	kubeclientset "k8s.io/client-go/kubernetes"
	"time"
)

type TestJobController struct {
	jobClientSet jobclientset.Interface
	JobController
	job      *testv1.TestJob
	pods     []*corev1.Pod
	services []*corev1.Service
}

func NewTestJobController(
	kubeClientSet kubeclientset.Interface,
	kubeBatchClientSet kubebatchclient.Interface,
	jobClientSet jobclientset.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	enableGangScheduling bool,
) *TestJobController {
	log.Info("Creating TestJob controller")
	// Create TestController.
	tc := &TestJobController{
		jobClientSet: jobClientSet,
	}

	// Create base controller
	log.Info("Creating Job controller")
	jc := NewJobController(tc, v1.Duration{Duration: 15 * time.Second},
		enableGangScheduling,
		kubeClientSet, kubeBatchClientSet, kubeInformerFactory,
		testv1.Plural)
	tc.JobController = jc

	return tc
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

func (TestJobController) GetGroupNameLabelValue() string {
	return testv1.GroupName
}

func (TestJobController) GetJobRoleKey() string {
	return util.LabelJobRole
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
	jobStatus commonv1.JobStatus, restart bool) error {
	return nil
}

func (t *TestJobController) UpdateJobStatusInApiServer(job interface{}, jobStatus *commonv1.JobStatus) error {
	return nil
}

func (t *TestJobController) CreateService(job interface{}, service *corev1.Service) error {
	return nil
}

func (t *TestJobController) DeleteService(job interface{}, name string, namespace string) error {
	log.Info("Deleting service " + name)
	var remainingServices []*corev1.Service
	for _, tservice := range t.services {
		if tservice.Name != name {
			remainingServices = append(remainingServices, tservice)
		}
	}
	t.services = remainingServices
	return nil
}

func (t *TestJobController) CreatePod(job interface{}, podTemplate *corev1.PodTemplateSpec) error {
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

func (t *TestJobController) SetClusterSpec(job interface{}, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	return nil
}

func (t *TestJobController) GetDefaultContainerName() string {
	return "default-container"
}

func (t *TestJobController) IsMasterRole(replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, rtype commonv1.ReplicaType, index int) bool {
	return true
}
