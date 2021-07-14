package core

import (
	"fmt"
	"strconv"

	apiv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// FilterServicesForReplicaType returns service belong to a replicaType.
func FilterServicesForReplicaType(services []*v1.Service, replicaType apiv1.ReplicaType) ([]*v1.Service, error) {
	var result []*v1.Service

	replicaSelector := &metav1.LabelSelector{
		MatchLabels: make(map[string]string),
	}

	replicaSelector.MatchLabels[apiv1.ReplicaTypeLabel] = string(replicaType)

	for _, service := range services {
		selector, err := metav1.LabelSelectorAsSelector(replicaSelector)
		if err != nil {
			return nil, err
		}
		if !selector.Matches(labels.Set(service.Labels)) {
			continue
		}
		result = append(result, service)
	}
	return result, nil
}

// GetServiceSlices returns a slice, which element is the slice of service.
// Assume the return object is serviceSlices, then serviceSlices[i] is an
// array of pointers to services corresponding to Services for replica i.
func GetServiceSlices(services []*v1.Service, replicas int, logger *log.Entry) [][]*v1.Service {
	serviceSlices := make([][]*v1.Service, CalculateServiceSliceSize(services, replicas))
	for _, service := range services {
		if _, ok := service.Labels[apiv1.ReplicaIndexLabel]; !ok {
			logger.Warning("The service do not have the index label.")
			continue
		}
		index, err := strconv.Atoi(service.Labels[apiv1.ReplicaIndexLabel])
		if err != nil {
			logger.Warningf("Error when strconv.Atoi: %v", err)
			continue
		}
		if index < 0 || index >= replicas {
			logger.Warningf("The label index is not expected: %d, service: %s/%s", index, service.Namespace, service.Name)
		}

		serviceSlices[index] = append(serviceSlices[index], service)
	}
	return serviceSlices
}

// CalculateServiceSliceSize compare max pod index with desired replicas and return larger size
func CalculateServiceSliceSize(services []*v1.Service, replicas int) int {
	size := 0
	for _, svc := range services {
		if _, ok := svc.Labels[apiv1.ReplicaIndexLabel]; !ok {
			continue
		}
		index, err := strconv.Atoi(svc.Labels[apiv1.ReplicaIndexLabel])
		if err != nil {
			continue
		}
		size = MaxInt(size, index)
	}

	// size comes from index, need to +1 to indicate real size
	return MaxInt(size+1, replicas)
}

// GetPortsFromJob gets the ports of job container. Port could be nil, if distributed communication strategy doesn't need and no other ports that need to be exposed.
func GetPortsFromJob(spec *apiv1.ReplicaSpec, defaultContainerName string) (map[string]int32, error) {
	ports := make(map[string]int32)

	containers := spec.Template.Spec.Containers
	for _, container := range containers {
		if container.Name == defaultContainerName {
			containerPorts := container.Ports
			if len(containerPorts) == 0 {
				return nil, nil
			}
			for _, port := range containerPorts {
				ports[port.Name] = port.ContainerPort
			}
			return ports, nil
		}
	}

	return nil, fmt.Errorf("failed to find the port")
}
