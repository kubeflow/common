package common

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *KubeflowReconciler) GetJob(ctx context.Context, req ctrl.Request) (client.Object, error) {
	panic("implement KubeflowReconciler.GetJob!")
}

func (r *KubeflowReconciler) EmptyJob() client.Object {
	panic("implement KubeflowReconciler.EmptyJob!")
}

func (r *KubeflowReconciler) GetAPIGroupVersionKind() schema.GroupVersionKind {
	panic("implement KubeflowReconciler.GetAPIGroupVersionKind!")
}

func (r *KubeflowReconciler) GetAPIGroupVersion() schema.GroupVersion {
	panic("implement KubeflowReconciler.GetAPIGroupVersion!")
}

func (r *KubeflowReconciler) ExtractReplicasSpec(job client.Object) (map[commonv1.ReplicaType]*commonv1.ReplicaSpec, error) {
	panic("implement KubeflowReconciler.ExtractReplicasSpec!")
}

func (r *KubeflowReconciler) ExtractRunPolicy(job client.Object) (*commonv1.RunPolicy, error) {
	panic("implement KubeflowReconciler.ExtractRunPolicy")
}

func (r *KubeflowReconciler) ExtractJobStatus(job client.Object) (*commonv1.JobStatus, error) {
	panic("implement KubeflowReconciler.ExtractJobStatus")
}

func (r *KubeflowReconciler) IsMasterRole(replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, rtype commonv1.ReplicaType, index int) bool {
	panic("implement KubeflowReconciler.IsMasterRole")
}
