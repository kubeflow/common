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

package control

import (
	testutilv1 "github.com/kubeflow/common/test_job/test_util/v1"
	"reflect"
	"testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	testjobv1 "github.com/kubeflow/common/test_job/apis/test_job/v1"
)

func TestClaimPods(t *testing.T) {
	controllerUID := "123"

	type test struct {
		name    string
		manager *PodControllerRefManager
		pods    []*v1.Pod
		claimed []*v1.Pod
	}
	var tests = []test{
		func() test {
			testJob := testutilv1.NewTestJob(1)
			testJobLabelSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
				MatchLabels: testutilv1.GenLabels(testJob.Name),
			})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			testPod := testutilv1.NewBasePod("pod2", testJob, nil)
			testPod.Labels[testutilv1.LabelGroupName] = "testing"

			return test{
				name: "Claim pods with correct label",
				manager: NewPodControllerRefManager(&FakePodControl{},
					testJob,
					testJobLabelSelector,
					testjobv1.SchemeGroupVersionKind,
					func() error { return nil }),
				pods:    []*v1.Pod{testutilv1.NewBasePod("pod1", testJob, t), testPod},
				claimed: []*v1.Pod{testutilv1.NewBasePod("pod1", testJob, t)},
			}
		}(),
		func() test {
			controller := testutilv1.NewTestJob(1)
			controllerLabelSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
				MatchLabels: testutilv1.GenLabels(controller.Name),
			})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			controller.UID = types.UID(controllerUID)
			now := metav1.Now()
			controller.DeletionTimestamp = &now
			testPod1 := testutilv1.NewBasePod("pod1", controller, t)
			testPod1.SetOwnerReferences([]metav1.OwnerReference{})
			testPod2 := testutilv1.NewBasePod("pod2", controller, t)
			testPod2.SetOwnerReferences([]metav1.OwnerReference{})
			return test{
				name: "Controller marked for deletion can not claim pods",
				manager: NewPodControllerRefManager(&FakePodControl{},
					controller,
					controllerLabelSelector,
					testjobv1.SchemeGroupVersionKind,
					func() error { return nil }),
				pods:    []*v1.Pod{testPod1, testPod2},
				claimed: nil,
			}
		}(),
		func() test {
			controller := testutilv1.NewTestJob(1)
			controllerLabelSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
				MatchLabels: testutilv1.GenLabels(controller.Name),
			})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			controller.UID = types.UID(controllerUID)
			now := metav1.Now()
			controller.DeletionTimestamp = &now
			testPod2 := testutilv1.NewBasePod("pod2", controller, t)
			testPod2.SetOwnerReferences([]metav1.OwnerReference{})
			return test{
				name: "Controller marked for deletion can not claim new pods",
				manager: NewPodControllerRefManager(&FakePodControl{},
					controller,
					controllerLabelSelector,
					testjobv1.SchemeGroupVersionKind,
					func() error { return nil }),
				pods:    []*v1.Pod{testutilv1.NewBasePod("pod1", controller, t), testPod2},
				claimed: []*v1.Pod{testutilv1.NewBasePod("pod1", controller, t)},
			}
		}(),
		func() test {
			controller := testutilv1.NewTestJob(1)
			controllerLabelSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
				MatchLabels: testutilv1.GenLabels(controller.Name),
			})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			controller2 := testutilv1.NewTestJob(1)
			controller.UID = types.UID(controllerUID)
			controller2.UID = types.UID("AAAAA")
			return test{
				name: "Controller can not claim pods owned by another controller",
				manager: NewPodControllerRefManager(&FakePodControl{},
					controller,
					controllerLabelSelector,
					testjobv1.SchemeGroupVersionKind,
					func() error { return nil }),
				pods:    []*v1.Pod{testutilv1.NewBasePod("pod1", controller, t), testutilv1.NewBasePod("pod2", controller2, t)},
				claimed: []*v1.Pod{testutilv1.NewBasePod("pod1", controller, t)},
			}
		}(),
		func() test {
			controller := testutilv1.NewTestJob(1)
			controllerLabelSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
				MatchLabels: testutilv1.GenLabels(controller.Name),
			})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			controller.UID = types.UID(controllerUID)
			testPod2 := testutilv1.NewBasePod("pod2", controller, t)
			testPod2.Labels[testutilv1.LabelGroupName] = "testing"
			return test{
				name: "Controller releases claimed pods when selector doesn't match",
				manager: NewPodControllerRefManager(&FakePodControl{},
					controller,
					controllerLabelSelector,
					testjobv1.SchemeGroupVersionKind,
					func() error { return nil }),
				pods:    []*v1.Pod{testutilv1.NewBasePod("pod1", controller, t), testPod2},
				claimed: []*v1.Pod{testutilv1.NewBasePod("pod1", controller, t)},
			}
		}(),
		func() test {
			controller := testutilv1.NewTestJob(1)
			controllerLabelSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
				MatchLabels: testutilv1.GenLabels(controller.Name),
			})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			controller.UID = types.UID(controllerUID)
			testPod1 := testutilv1.NewBasePod("pod1", controller, t)
			testPod2 := testutilv1.NewBasePod("pod2", controller, t)
			testPod2.Labels[testutilv1.LabelGroupName] = "testing"
			now := metav1.Now()
			testPod1.DeletionTimestamp = &now
			testPod2.DeletionTimestamp = &now

			return test{
				name: "Controller does not claim orphaned pods marked for deletion",
				manager: NewPodControllerRefManager(&FakePodControl{},
					controller,
					controllerLabelSelector,
					testjobv1.SchemeGroupVersionKind,
					func() error { return nil }),
				pods:    []*v1.Pod{testPod1, testPod2},
				claimed: []*v1.Pod{testPod1},
			}
		}(),
	}
	for _, test := range tests {
		claimed, err := test.manager.ClaimPods(test.pods)
		if err != nil {
			t.Errorf("Test case `%s`, unexpected error: %v", test.name, err)
		} else if !reflect.DeepEqual(test.claimed, claimed) {
			t.Errorf("Test case `%s`, claimed wrong pods. Expected %v, got %v", test.name, podToStringSlice(test.claimed), podToStringSlice(claimed))
		}

	}
}

func podToStringSlice(pods []*v1.Pod) []string {
	var names []string
	for _, pod := range pods {
		names = append(names, pod.Name)
	}
	return names
}

func TestClaimServices(t *testing.T) {
	controllerUID := "123"

	type test struct {
		name     string
		manager  *ServiceControllerRefManager
		services []*v1.Service
		claimed  []*v1.Service
	}
	var tests = []test{
		func() test {
			testJob := testutilv1.NewTestJob(1)
			testJobLabelSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
				MatchLabels: testutilv1.GenLabels(testJob.Name),
			})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			testService := testutilv1.NewBaseService("service2", testJob, nil)
			testService.Labels[testutilv1.LabelGroupName] = "testing"

			return test{
				name: "Claim services with correct label",
				manager: NewServiceControllerRefManager(&FakeServiceControl{},
					testJob,
					testJobLabelSelector,
					testjobv1.SchemeGroupVersionKind,
					func() error { return nil }),
				services: []*v1.Service{testutilv1.NewBaseService("service1", testJob, t), testService},
				claimed:  []*v1.Service{testutilv1.NewBaseService("service1", testJob, t)},
			}
		}(),
		func() test {
			controller := testutilv1.NewTestJob(1)
			controllerLabelSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
				MatchLabels: testutilv1.GenLabels(controller.Name),
			})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			controller.UID = types.UID(controllerUID)
			now := metav1.Now()
			controller.DeletionTimestamp = &now
			testService1 := testutilv1.NewBaseService("service1", controller, t)
			testService1.SetOwnerReferences([]metav1.OwnerReference{})
			testService2 := testutilv1.NewBaseService("service2", controller, t)
			testService2.SetOwnerReferences([]metav1.OwnerReference{})
			return test{
				name: "Controller marked for deletion can not claim services",
				manager: NewServiceControllerRefManager(&FakeServiceControl{},
					controller,
					controllerLabelSelector,
					testjobv1.SchemeGroupVersionKind,
					func() error { return nil }),
				services: []*v1.Service{testService1, testService2},
				claimed:  nil,
			}
		}(),
		func() test {
			controller := testutilv1.NewTestJob(1)
			controllerLabelSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
				MatchLabels: testutilv1.GenLabels(controller.Name),
			})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			controller.UID = types.UID(controllerUID)
			now := metav1.Now()
			controller.DeletionTimestamp = &now
			testService2 := testutilv1.NewBaseService("service2", controller, t)
			testService2.SetOwnerReferences([]metav1.OwnerReference{})
			return test{
				name: "Controller marked for deletion can not claim new services",
				manager: NewServiceControllerRefManager(&FakeServiceControl{},
					controller,
					controllerLabelSelector,
					testjobv1.SchemeGroupVersionKind,
					func() error { return nil }),
				services: []*v1.Service{testutilv1.NewBaseService("service1", controller, t), testService2},
				claimed:  []*v1.Service{testutilv1.NewBaseService("service1", controller, t)},
			}
		}(),
		func() test {
			controller := testutilv1.NewTestJob(1)
			controllerLabelSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
				MatchLabels: testutilv1.GenLabels(controller.Name),
			})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			controller2 := testutilv1.NewTestJob(1)
			controller.UID = types.UID(controllerUID)
			controller2.UID = types.UID("AAAAA")
			return test{
				name: "Controller can not claim services owned by another controller",
				manager: NewServiceControllerRefManager(&FakeServiceControl{},
					controller,
					controllerLabelSelector,
					testjobv1.SchemeGroupVersionKind,
					func() error { return nil }),
				services: []*v1.Service{testutilv1.NewBaseService("service1", controller, t), testutilv1.NewBaseService("service2", controller2, t)},
				claimed:  []*v1.Service{testutilv1.NewBaseService("service1", controller, t)},
			}
		}(),
		func() test {
			controller := testutilv1.NewTestJob(1)
			controllerLabelSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
				MatchLabels: testutilv1.GenLabels(controller.Name),
			})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			controller.UID = types.UID(controllerUID)
			testService2 := testutilv1.NewBaseService("service2", controller, t)
			testService2.Labels[testutilv1.LabelGroupName] = "testing"
			return test{
				name: "Controller releases claimed services when selector doesn't match",
				manager: NewServiceControllerRefManager(&FakeServiceControl{},
					controller,
					controllerLabelSelector,
					testjobv1.SchemeGroupVersionKind,
					func() error { return nil }),
				services: []*v1.Service{testutilv1.NewBaseService("service1", controller, t), testService2},
				claimed:  []*v1.Service{testutilv1.NewBaseService("service1", controller, t)},
			}
		}(),
		func() test {
			controller := testutilv1.NewTestJob(1)
			controllerLabelSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
				MatchLabels: testutilv1.GenLabels(controller.Name),
			})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			controller.UID = types.UID(controllerUID)
			testService1 := testutilv1.NewBaseService("service1", controller, t)
			testService2 := testutilv1.NewBaseService("service2", controller, t)
			testService2.Labels[testutilv1.LabelGroupName] = "testing"
			now := metav1.Now()
			testService1.DeletionTimestamp = &now
			testService2.DeletionTimestamp = &now

			return test{
				name: "Controller does not claim orphaned services marked for deletion",
				manager: NewServiceControllerRefManager(&FakeServiceControl{},
					controller,
					controllerLabelSelector,
					testjobv1.SchemeGroupVersionKind,
					func() error { return nil }),
				services: []*v1.Service{testService1, testService2},
				claimed:  []*v1.Service{testService1},
			}
		}(),
	}
	for _, test := range tests {
		claimed, err := test.manager.ClaimServices(test.services)
		if err != nil {
			t.Errorf("Test case `%s`, unexpected error: %v", test.name, err)
		} else if !reflect.DeepEqual(test.claimed, claimed) {
			t.Errorf("Test case `%s`, claimed wrong services. Expected %v, got %v", test.name, serviceToStringSlice(test.claimed), serviceToStringSlice(claimed))
		}

	}
}

func serviceToStringSlice(services []*v1.Service) []string {
	var names []string
	for _, service := range services {
		names = append(names, service.Name)
	}
	return names
}
