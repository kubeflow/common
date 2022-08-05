// Copyright 2022 The Kubeflow Authors
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
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/scheduler-plugins/apis/scheduling/v1alpha1"
	"volcano.sh/apis/pkg/apis/scheduling/v1beta1"
	volcanoclient "volcano.sh/apis/pkg/client/clientset/versioned"
)

// volcanoTaskSpecKey task spec key used in pod annotation when EnableGangScheduling is true
const volcanoTaskSpecKey = "volcano.sh/task-spec"

// PodGroupControlInterface is an interface that knows how to add or delete PodGroups
// created as an interface to allow testing.
type PodGroupControlInterface interface {
	// NewEmptyPodGroup returns an empty PodGroup
	NewEmptyPodGroup() client.Object
	// GetPodGroup gets the PodGroup identified by namespace and name
	GetPodGroup(namespace string, name string) (metav1.Object, error)
	// DeletePodGroup deletes the PodGroup identified by namespace and name.
	DeletePodGroup(namespace string, name string) error
	// CreatePodGroup creates a new PodGroup with PodGroup spec fill function.
	CreatePodGroup(podGroup client.Object) error
	// DelayPodCreationDueToPodGroup determines whether it should delay Pod Creation.
	DelayPodCreationDueToPodGroup(pg metav1.Object) bool
	// DecoratePodTemplateSpec decorates PodTemplateSpec.
	// If the PodTemplateSpec has SchedulerName set, this method will Not override
	DecoratePodTemplateSpec(pts *corev1.PodTemplateSpec, job metav1.Object, rtype string)
	// GetSchedulerName returns the name of the gang scheduler
	GetSchedulerName() string
}

// VolcanoControl is the implementation of PodGroupControlInterface with volcano.
type VolcanoControl struct {
	Client volcanoclient.Interface
}

func (v *VolcanoControl) GetSchedulerName() string {
	return "volcano"
}

func (v *VolcanoControl) DecoratePodTemplateSpec(pts *corev1.PodTemplateSpec, job metav1.Object, rtype string) {
	if pts.Spec.SchedulerName == "" {
		pts.Spec.SchedulerName = v.GetSchedulerName()
	}
	if pts.Annotations == nil {
		pts.Annotations = map[string]string{}
	}
	pts.Annotations[v1beta1.KubeGroupNameAnnotationKey] = job.GetName()
	pts.Annotations[volcanoTaskSpecKey] = rtype
}

// NewVolcanoControl returns a VolcanoControl
func NewVolcanoControl(vci volcanoclient.Interface) PodGroupControlInterface {
	return &VolcanoControl{Client: vci}
}

func (v *VolcanoControl) DelayPodCreationDueToPodGroup(pg metav1.Object) bool {
	if pg == nil {
		return true
	}

	volcanoPodGroup := pg.(*v1beta1.PodGroup)

	if volcanoPodGroup.Status.Phase == "" {
		return true
	}

	if volcanoPodGroup.Status.Phase == v1beta1.PodGroupPending {
		return true
	}

	return false
}

func (v *VolcanoControl) NewEmptyPodGroup() client.Object {
	return &v1beta1.PodGroup{}
}

func (v *VolcanoControl) GetPodGroup(namespace string, name string) (metav1.Object, error) {
	pg, err := v.Client.SchedulingV1beta1().PodGroups(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return pg, nil
}

func (v *VolcanoControl) DeletePodGroup(namespace string, name string) error {
	return v.Client.SchedulingV1beta1().PodGroups(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
}

func (v *VolcanoControl) CreatePodGroup(podGroup client.Object) error {
	pg := podGroup.(*v1beta1.PodGroup)
	createPodGroup, err := v.Client.SchedulingV1beta1().PodGroups(podGroup.GetNamespace()).Create(context.TODO(), pg, metav1.CreateOptions{})
	if err != nil {
		podGroup = createPodGroup
		return fmt.Errorf("unable to create PodGroup: %v", err)
	}
	return err
}

var _ PodGroupControlInterface = &VolcanoControl{}

// SchedulerPluginsControl is the  implementation of PodGroupControlInterface with scheduler-plugins.
type SchedulerPluginsControl struct {
	Client client.Client
}

func (s *SchedulerPluginsControl) DecoratePodTemplateSpec(pts *corev1.PodTemplateSpec, job metav1.Object, rtype string) {
	if pts.Spec.SchedulerName == "" {
		pts.Spec.SchedulerName = s.GetSchedulerName()
	}
	if pts.Labels == nil {
		pts.Labels = map[string]string{}
	}
	pts.Labels["pod-group.scheduling.sigs.k8s.io"] = job.GetName()
}

func (s *SchedulerPluginsControl) GetSchedulerName() string {
	return "scheduler-plugins-scheduler"
}

// NewSchedulerPluginsControl returns a SchedulerPluginsControl
func NewSchedulerPluginsControl(c client.Client) PodGroupControlInterface {
	return &SchedulerPluginsControl{Client: c}
}

func (s *SchedulerPluginsControl) DelayPodCreationDueToPodGroup(pg metav1.Object) bool {
	return false
}

func (s *SchedulerPluginsControl) NewEmptyPodGroup() client.Object {
	return &v1alpha1.PodGroup{}
}

func (s *SchedulerPluginsControl) GetPodGroup(namespace string, name string) (metav1.Object, error) {
	pg := &v1beta1.PodGroup{}
	ctx := context.TODO()
	key := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	err := s.Client.Get(ctx, key, pg)
	if err == nil {
		return pg, nil
	}
	return nil, err
}

func (s *SchedulerPluginsControl) DeletePodGroup(namespace string, name string) error {
	ctx := context.TODO()
	pg := s.NewEmptyPodGroup()
	pg.SetNamespace(namespace)
	pg.SetName(name)

	return s.Client.Delete(ctx, pg)
}

func (s *SchedulerPluginsControl) CreatePodGroup(podGroup client.Object) error {
	ctx := context.TODO()
	return s.Client.Create(ctx, podGroup)
}

var _ PodGroupControlInterface = &SchedulerPluginsControl{}
