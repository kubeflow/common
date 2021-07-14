package common

import (
	"context"
	"sort"

	"k8s.io/api/scheduling/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeflow/common/pkg/util/k8sutil"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/api/errors"

	volcano "volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *KubeflowReconciler) GangSchedulingEnabled() bool {
	return r.Config.EnableGangScheduling
}

type BaseGangReconciler struct {
	GangSchedulingInterface
}

func (r *BaseGangReconciler) GangResourceName(job client.Object) string {
	return job.GetName()
}

type VolcanoReconciler struct {
	BaseGangReconciler
	client.Client
}

func NewVolcanoReconciler(client client.Client) GangSchedulingInterface {
	return &VolcanoReconciler{
		Client: client,
	}
}

func (r *VolcanoReconciler) DeleteGangResources(ctx context.Context, job client.Object) error {
	pg := &volcano.PodGroup{}
	pg.SetNamespace(job.GetNamespace())
	pg.SetName(r.GangResourceName(job))

	err := r.Delete(ctx, pg)
	if errors.IsNotFound(err) {
		return nil
	}
	return err
}

func (r *VolcanoReconciler) GetGangResourcesForJob(ctx context.Context, job client.Object) (client.Object, error) {
	var pg *volcano.PodGroup = nil
	err := r.Get(ctx, types.NamespacedName{
		Namespace: job.GetNamespace(),
		Name:      r.GangResourceName(job),
	}, pg)
	return pg, err
}

func (r *VolcanoReconciler) ReconcileGangResource(
	ctx context.Context,
	job client.Object,
	runPolicy *commonv1.RunPolicy,
	replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec,
	ownRefs *metav1.OwnerReference) error {

	minMember := k8sutil.GetTotalReplicas(replicas)
	queue := ""
	priorityClass := ""
	var minResources *corev1.ResourceList

	if runPolicy.SchedulingPolicy != nil {
		if runPolicy.SchedulingPolicy.MinAvailable != nil {
			minMember = *runPolicy.SchedulingPolicy.MinAvailable
		}

		if runPolicy.SchedulingPolicy.Queue != "" {
			queue = runPolicy.SchedulingPolicy.Queue
		}

		if runPolicy.SchedulingPolicy.PriorityClass != "" {
			priorityClass = runPolicy.SchedulingPolicy.PriorityClass
		}

		if runPolicy.SchedulingPolicy.MinResources != nil {
			minResources = runPolicy.SchedulingPolicy.MinResources
		}
	}

	if minResources == nil {
		minResources = r.calcPGMinResources(minMember, replicas)
	}

	pgSpec := volcano.PodGroupSpec{
		MinMember:         minMember,
		Queue:             queue,
		PriorityClassName: priorityClass,
		MinResources:      minResources,
	}

	// Check if exist
	pg := &volcano.PodGroup{}
	err := r.Get(ctx, types.NamespacedName{Namespace: job.GetNamespace(), Name: r.GangResourceName(job)}, pg)
	// If Created, check updates, otherwise create it
	if err == nil {
		pg.ObjectMeta = metav1.ObjectMeta{}
		pg.Spec = pgSpec
		err = r.Update(ctx, pg)
	}

	if errors.IsNotFound(err) {
		pg.ObjectMeta = metav1.ObjectMeta{}
		pg.Spec = pgSpec
		err = r.Create(ctx, pg)
	}

	if err != nil {
		log.Warnf("Sync PodGroup %v: %v",
			types.NamespacedName{Namespace: job.GetNamespace(), Name: r.GangResourceName(job)}, err)
		return err
	}

	return nil
}

func (r *VolcanoReconciler) calcPGMinResources(minMember int32, replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec) *corev1.ResourceList {
	var replicasPriority ReplicasPriority
	for t, replica := range replicas {
		rp := ReplicaPriority{0, *replica}
		pc := replica.Template.Spec.PriorityClassName

		var priorityClass *v1beta1.PriorityClass = nil
		err := r.Get(context.Background(), types.NamespacedName{Name: pc}, priorityClass)
		if err != nil || priorityClass == nil {
			log.Warnf("Ignore task %s priority class %s: %v", t, pc, err)
		} else {
			rp.priority = priorityClass.Value
		}

		replicasPriority = append(replicasPriority, rp)
	}

	sort.Sort(replicasPriority)

	minAvailableTasksRes := corev1.ResourceList{}
	podCnt := int32(0)
	for _, task := range replicasPriority {
		if task.Replicas == nil {
			continue
		}

		for i := int32(0); i < *task.Replicas; i++ {
			if podCnt >= minMember {
				break
			}
			podCnt++
			for _, c := range task.Template.Spec.Containers {
				AddResourceList(minAvailableTasksRes, c.Resources.Requests, c.Resources.Limits)
			}
		}
	}

	return &minAvailableTasksRes
}

type SchedulerFrameworkReconciler struct {
	BaseGangReconciler
}
