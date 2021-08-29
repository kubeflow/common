package common

import (
	schedulingv1 "k8s.io/client-go/kubernetes/typed/scheduling/v1"
	"testing"

	apiv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/controller.v1/control"
	"github.com/kubeflow/common/pkg/controller.v1/expectation"
	testjobv1 "github.com/kubeflow/common/test_job/apis/test_job/v1"
	testjob "github.com/kubeflow/common/test_job/controller.v1/test_job"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	volcanoclient "volcano.sh/apis/pkg/client/clientset/versioned"
)

func TestCalculateServiceSliceSize(t *testing.T) {
	type testCase struct {
		services     []*corev1.Service
		replicas     int
		expectedSize int
	}

	services := []*corev1.Service{
		{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{apiv1.ReplicaIndexLabel: "0"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{apiv1.ReplicaIndexLabel: "1"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{apiv1.ReplicaIndexLabel: "2"},
			},
		},
	}

	var testCases = []testCase{
		{
			services:     services,
			replicas:     3,
			expectedSize: 3,
		},
		{
			services:     services,
			replicas:     4,
			expectedSize: 4,
		},
		{
			services:     services,
			replicas:     2,
			expectedSize: 3,
		},
		{
			services: append(services, &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{apiv1.ReplicaIndexLabel: "4"},
				},
			}),
			replicas:     3,
			expectedSize: 5,
		},
	}

	for _, tc := range testCases {
		result := calculateServiceSliceSize(tc.services, tc.replicas)
		assert.Equal(t, tc.expectedSize, result)
	}
}

func TestJobController_CreateNewService(t *testing.T) {
	type fields struct {
		Controller                  apiv1.ControllerInterface
		Config                      JobControllerConfiguration
		PodControl                  control.PodControlInterface
		ServiceControl              control.ServiceControlInterface
		KubeClientSet               kubeclientset.Interface
		VolcanoClientSet            volcanoclient.Interface
		PodLister                   corelisters.PodLister
		ServiceLister               corelisters.ServiceLister
		PriorityClassInterface      schedulingv1.PriorityClassInterface
		PodInformerSynced           cache.InformerSynced
		ServiceInformerSynced       cache.InformerSynced
		PriorityClassInformerSynced cache.InformerSynced
		Expectations                expectation.ControllerExpectationsInterface
		WorkQueue                   workqueue.RateLimitingInterface
		Recorder                    record.EventRecorder
	}
	type args struct {
		job   metav1.Object
		rtype apiv1.ReplicaType
		spec  *apiv1.ReplicaSpec
		index string
	}

	var replicas int32
	replicas = 2
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{name: "test0",
			fields: fields{
				Controller:     &testjob.TestJobController{},
				Expectations:   expectation.NewControllerExpectations(),
				ServiceControl: &control.FakeServiceControl{},
			},
			args: args{
				job:   &testjobv1.TestJob{},
				rtype: "Worker",
				spec: &apiv1.ReplicaSpec{
					Replicas: &replicas,
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								v1.Container{
									Name: "default-container",
									Ports: []v1.ContainerPort{
										v1.ContainerPort{
											Name:          "test",
											ContainerPort: 8080,
										},
										v1.ContainerPort{
											Name:          "default-port-name",
											ContainerPort: 2222,
										},
									},
								},
							},
						},
					},
				},
				index: "0",
			},
			wantErr: false,
		},
		{name: "test1",
			fields: fields{
				Controller:     &testjob.TestJobController{},
				Expectations:   expectation.NewControllerExpectations(),
				ServiceControl: &control.FakeServiceControl{},
			},
			args: args{
				job:   &testjobv1.TestJob{},
				rtype: "Master",
				spec: &apiv1.ReplicaSpec{
					Replicas: &replicas,
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								v1.Container{
									Name: "fake-container",
									Ports: []v1.ContainerPort{
										v1.ContainerPort{
											Name:          "test",
											ContainerPort: 8080,
										},
										v1.ContainerPort{
											Name:          "default-port-name",
											ContainerPort: 2222,
										},
									},
								},
							},
						},
					},
				},
				index: "0",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jc := &JobController{
				Controller:             tt.fields.Controller,
				Config:                 tt.fields.Config,
				PodControl:             tt.fields.PodControl,
				ServiceControl:         tt.fields.ServiceControl,
				KubeClientSet:          tt.fields.KubeClientSet,
				VolcanoClientSet:       tt.fields.VolcanoClientSet,
				PodLister:              tt.fields.PodLister,
				ServiceLister:          tt.fields.ServiceLister,
				PriorityClassInterface: tt.fields.PriorityClassInterface,
				PodInformerSynced:      tt.fields.PodInformerSynced,
				ServiceInformerSynced:  tt.fields.ServiceInformerSynced,
				Expectations:           tt.fields.Expectations,
				WorkQueue:              tt.fields.WorkQueue,
				Recorder:               tt.fields.Recorder,
			}
			if err := jc.CreateNewService(tt.args.job, tt.args.rtype, tt.args.spec, tt.args.index); (err != nil) != tt.wantErr {
				t.Errorf("JobController.CreateNewService() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
