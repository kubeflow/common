package core

import (
	utillabels "github.com/kubeflow/common/pkg/util/labels"

	apiv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// FilterPodsForReplicaType returns pods belong to a replicaType.
func FilterPodsForReplicaType(pods []*v1.Pod, replicaType string) ([]*v1.Pod, error) {
	var result []*v1.Pod

	selector := labels.SelectorFromValidatedSet(labels.Set{
		apiv1.ReplicaTypeLabel: replicaType,
	})

	// TODO(#149): Remove deprecated selector.
	deprecatedSelector := labels.SelectorFromValidatedSet(labels.Set{
		apiv1.ReplicaTypeLabelDeprecated: replicaType,
	})

	for _, pod := range pods {
		set := labels.Set(pod.Labels)
		if !selector.Matches(set) && !deprecatedSelector.Matches(set) {
			continue
		}
		result = append(result, pod)
	}
	return result, nil
}

// GetPodSlices returns a slice, which element is the slice of pod.
// It gives enough information to caller to make decision to up/down scale resources.
func GetPodSlices(pods []*v1.Pod, replicas int, logger *log.Entry) [][]*v1.Pod {
	podSlices := make([][]*v1.Pod, CalculatePodSliceSize(pods, replicas))
	for _, pod := range pods {
		index, err := utillabels.ReplicaIndex(pod.Labels)
		if err != nil {
			logger.Warningf("Error obtaining replica index from Pod %s/%s: %v", pod.Namespace, pod.Name, err)
			continue
		}
		if index < 0 || index >= replicas {
			logger.Warningf("The label index is not expected: %d, pod: %s/%s", index, pod.Namespace, pod.Name)
		}

		podSlices[index] = append(podSlices[index], pod)
	}
	return podSlices
}

// CalculatePodSliceSize compare max pod index with desired replicas and return larger size
func CalculatePodSliceSize(pods []*v1.Pod, replicas int) int {
	size := 0
	for _, pod := range pods {
		index, err := utillabels.ReplicaIndex(pod.Labels)
		if err != nil {
			continue
		}
		size = MaxInt(size, index)
	}

	// size comes from index, need to +1 to indicate real size
	return MaxInt(size+1, replicas)
}

// SetRestartPolicy check the RestartPolicy defined in job spec and overwrite RestartPolicy in podTemplate if necessary
func SetRestartPolicy(podTemplateSpec *v1.PodTemplateSpec, spec *apiv1.ReplicaSpec) {
	// This is necessary since restartPolicyExitCode is not supported in v1.PodTemplateSpec
	if spec.RestartPolicy == apiv1.RestartPolicyExitCode {
		podTemplateSpec.Spec.RestartPolicy = v1.RestartPolicyNever
	} else {
		podTemplateSpec.Spec.RestartPolicy = v1.RestartPolicy(spec.RestartPolicy)
	}
}
