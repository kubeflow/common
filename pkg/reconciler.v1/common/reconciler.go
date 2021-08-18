// Copyright 2021 The Kubeflow Authors
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

package common

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// KubeflowReconciler reconciles a KubeflowJob object
type KubeflowReconciler struct {
	JobInterface
	PodInterface
	ServiceInterface
	GangSchedulingInterface
	ReconcilerUtilInterface
}

// BareKubeflowReconciler returns a pointer of KubeflowReconciler with minimal implementation
func BareKubeflowReconciler() *KubeflowReconciler {
	return &KubeflowReconciler{}
}

// DefaultKubeflowReconciler generates the default KubeflowReconciler with default sub-reconcilers fully setup
func DefaultKubeflowReconciler(mgr manager.Manager, gangEnable bool) *KubeflowReconciler {
	kubeflowReconciler := BareKubeflowReconciler()

	// Generate Bare Components
	jobInter := BareKubeflowJobReconciler(mgr.GetClient())
	podInter := BareKubeflowPodReconciler(mgr.GetClient())
	svcInter := BareKubeflowServiceReconciler(mgr.GetClient())
	gangInter := BareVolcanoReconciler(mgr.GetClient(), nil, gangEnable)
	utilInter := BareUtilReconciler(mgr.GetEventRecorderFor(kubeflowReconciler.GetReconcilerName()), mgr.GetLogger(), mgr.GetScheme())

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

	return kubeflowReconciler
}

// OverrideForKubeflowReconcilerInterface resets JobInterface, PodInterface, ServiceInterface, GangSchedulingInterface,
// ReconcilerUtilInterface for KubeflowReconciler and its sub-reconcilers
func (r *KubeflowReconciler) OverrideForKubeflowReconcilerInterface(
	ji JobInterface,
	pi PodInterface,
	si ServiceInterface,
	gi GangSchedulingInterface,
	ui ReconcilerUtilInterface) {
	r.JobInterface.OverrideForJobInterface(ui, pi, si, gi)
	r.PodInterface.OverrideForPodInterface(ui, gi, ji)
	r.ServiceInterface.OverrideForServiceInterface(ui, pi, ji)
	r.GangSchedulingInterface.OverrideForGangSchedulingInterface(ui)
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *KubeflowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	job, err := r.GetJob(ctx, req)
	if err != nil {
		return ctrl.Result{}, err
	}

	logger := r.GetLogger(job)

	if job.GetDeletionTimestamp() != nil {
		logger.Info(MsgReconcileCancelled, ReasonKey, ReasonJobDeleted)
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
		logger.Info("Reconcile PyTorch Job error %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KubeflowReconciler) SetupWithManager(mgr ctrl.Manager, obj client.Object) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(obj).
		Owns(&corev1.Pod{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
