package common

import (
	"context"
	"fmt"
	"strconv"

	commonutil "github.com/kubeflow/common/pkg/util"

	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	succeededServiceCreationCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "succeeded_service_creation_total",
		Help: "The total number of succeeded service creation",
	})
	failedServiceCreationCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "failed_service_creation_total",
		Help: "The total number of failed service creation",
	})
)

func (r *KubeflowReconciler) GetServicesForJob(ctx context.Context, job client.Object) ([]*corev1.Service, error) {
	svcList := &corev1.ServiceList{}
	err := r.List(ctx, svcList, client.MatchingLabels(r.GenLabels(job.GetName())))
	if err != nil {
		return nil, err
	}

	var svcs []*corev1.Service = nil
	for _, svc := range svcList.Items {
		svcs = append(svcs, &svc)
	}

	return svcs, nil
}

// FilterServicesForReplicaType returns service belong to a replicaType.
func (r *KubeflowReconciler) FilterServicesForReplicaType(services []*corev1.Service, replicaType commonv1.ReplicaType) ([]*corev1.Service, error) {
	var result []*corev1.Service

	replicaSelector := &metav1.LabelSelector{
		MatchLabels: make(map[string]string),
	}

	replicaSelector.MatchLabels[commonv1.ReplicaTypeLabel] = string(replicaType)

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

func (r *KubeflowReconciler) ReconcileServices(job client.Object,
	services []*corev1.Service,
	rtype commonv1.ReplicaType,
	spec *commonv1.ReplicaSpec) error {

	replicas := int(*spec.Replicas)
	// Get all services for the type rt.
	services, err := r.FilterServicesForReplicaType(services, rtype)
	if err != nil {
		return err
	}

	// GetServiceSlices will return enough information here to make decision to add/remove/update resources.
	//
	// For example, let's assume we have services with replica-index 0, 1, 2
	// If replica is 4, return a slice with size 4. [[0],[1],[2],[]], a svc with replica-index 3 will be created.
	//
	// If replica is 1, return a slice with size 3. [[0],[1],[2]], svc with replica-index 1 and 2 are out of range and will be deleted.
	serviceSlices := r.GetServiceSlices(services, replicas, commonutil.LoggerForReplica(job, rtype))

	for index, serviceSlice := range serviceSlices {
		if len(serviceSlice) > 1 {
			commonutil.LoggerForReplica(job, rtype).Warningf("We have too many services for %s %d", rtype, index)
		} else if len(serviceSlice) == 0 {
			commonutil.LoggerForReplica(job, rtype).Infof("need to create new service: %s-%d", rtype, index)
			err = r.CreateNewService(job, rtype, spec, strconv.Itoa(index))
			if err != nil {
				return err
			}
		} else {
			// Check the status of the current svc.
			svc := serviceSlice[0]

			// check if the index is in the valid range, if not, we should kill the svc
			if index < 0 || index >= replicas {
				err = r.DeleteService(svc.Namespace, svc.Name, job)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil

}

func (r *KubeflowReconciler) DeleteService(ns string, name string, job client.Object) error {
	svc := &corev1.Service{}
	svc.Name = name
	svc.Namespace = ns
	err := r.Delete(context.Background(), svc)
	if err == nil {
		deletedPodsCount.Inc()
	}
	return err
}

func (r *KubeflowReconciler) GetServiceSlices(services []*corev1.Service, replicas int, logger *log.Entry) [][]*corev1.Service {
	serviceSlices := make([][]*corev1.Service, calculateServiceSliceSize(services, replicas))
	for _, service := range services {
		if _, ok := service.Labels[commonv1.ReplicaIndexLabel]; !ok {
			logger.Warning("The service do not have the index label.")
			continue
		}
		index, err := strconv.Atoi(service.Labels[commonv1.ReplicaIndexLabel])
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

// calculateServiceSliceSize compare max pod index with desired replicas and return larger size
func calculateServiceSliceSize(services []*corev1.Service, replicas int) int {
	size := 0
	for _, svc := range services {
		if _, ok := svc.Labels[commonv1.ReplicaIndexLabel]; !ok {
			continue
		}
		index, err := strconv.Atoi(svc.Labels[commonv1.ReplicaIndexLabel])
		if err != nil {
			continue
		}
		size = MaxInt(size, index)
	}

	// size comes from index, need to +1 to indicate real size
	return MaxInt(size+1, replicas)
}

// GetPortsFromJob gets the ports of job container. Port could be nil, if distributed communication strategy doesn't need and no other ports that need to be exposed.
func (r *KubeflowReconciler) GetPortsFromJob(spec *commonv1.ReplicaSpec) (map[string]int32, error) {
	ports := make(map[string]int32)

	containers := spec.Template.Spec.Containers
	for _, container := range containers {
		if container.Name == r.GetDefaultContainerName() {
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

func (r *KubeflowReconciler) CreateNewService(job metav1.Object, rtype commonv1.ReplicaType,
	spec *commonv1.ReplicaSpec, index string) error {

	// Append ReplicaTypeLabel and ReplicaIndexLabel labels.
	labels := r.GenLabels(job.GetName())
	labels[commonv1.ReplicaTypeLabel] = string(rtype)
	labels[commonv1.ReplicaIndexLabel] = index

	ports, err := r.GetPortsFromJob(spec)
	if err != nil {
		return err
	}

	service := &corev1.Service{
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Selector:  labels,
			Ports:     []corev1.ServicePort{},
		},
	}

	// Add service ports to headless service
	for name, port := range ports {
		svcPort := corev1.ServicePort{Name: name, Port: port}
		service.Spec.Ports = append(service.Spec.Ports, svcPort)
	}

	service.Name = GenGeneralName(job.GetName(), rtype, index)
	service.Labels = labels
	// Create OwnerReference.
	err = controllerutil.SetControllerReference(job, service, r.Scheme)
	if err != nil {
		return err
	}

	err = r.Create(context.Background(), service)
	if err != nil && errors.IsTimeout(err) {
		succeededServiceCreationCount.Inc()
		return nil
	} else if err != nil {
		failedServiceCreationCount.Inc()
		return err
	}
	succeededServiceCreationCount.Inc()
	return nil
}
