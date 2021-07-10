package common

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
)

type KubeflowReconcilerInterface interface {
	Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)
}
