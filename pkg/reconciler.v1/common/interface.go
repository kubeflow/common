package common

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	ctrl "sigs.k8s.io/controller-runtime"
)

type GangSchedulingInterface interface {
	GangResourceName(job client.Object) string
	GetGangResourceForJob(ctx context.Context, job client.Object) (client.Object, error)
	DeleteGangResource(ctx context.Context, job client.Object) error
	ReconcileGangResource(ctx context.Context, job client.Object, runPolicy *commonv1.RunPolicy,
		replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, ownRefs *metav1.OwnerReference) error
}

type PodInterface interface {
}

type ServiceInterface interface {
}

type KubeflowReconcilerInterface interface {
	PodInterface
	ServiceInterface
	GangSchedulingInterface

	Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)
}
