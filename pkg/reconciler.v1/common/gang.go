package common

import (
	"context"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *KubeflowReconciler) GangSchedulingEnabled() bool {

}

func (r *KubeflowReconciler) GetGangResourcesForJob(ctx context.Context, job client.Object) ([]interface{}, error) {

}

func (r *KubeflowReconciler) DeleteGangResources(ctx context.Context, gangs []interface{}) error {

}

func (r *KubeflowReconciler) GenerateExpectedGangResources(job client.Object, runPolicy *commonv1.RunPolicy,
	replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec) ([]interface{}, error) {

}

func (r *KubeflowReconciler) ReconcileGangResources(job client.Object, runPolicy *commonv1.RunPolicy,
	replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, status *commonv1.JobStatus) error {

}
