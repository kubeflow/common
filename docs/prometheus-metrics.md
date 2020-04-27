# Prometheus Metrics Coverage

We plan to collect a rich set of metrics in kubeflow/common's `JobController` using [Prometheus](https://prometheus.io/).
The goal is to report generic metrics (e.g. metrics related to pods/jobs/services) during the lifecycle of `JobController` so that:

* Other operators built on top of it will automatically report Prometheus metrics without additional efforts;
* It is easier for users of Kubeflow distributed training operators to monitor operator performance and behaviors using consistent set of metrics for different distributed training operators.

This document outlines the list of Prometheus metrics we plan to cover in `JobController`. We follow the metric naming convention
outlined [here](https://prometheus.io/docs/practices/naming/).

## Pod Metrics

The following metrics related to the lifecycle of pods will be reported:

| Metric Name | Description |
| ------------ | ------- |
| created_pods_total | The total number of created pods |
| restarted_pods_total | The total number of restarted pods |
| deleted_pods_total | The total number of deleted pods |
| failed_pods_total | The total number of failed pods |

The following metrics will be reported on each pod:

| Metric Name | Description |
| ------------ | ------- |
| container_cpu_usage_seconds_total | CPU usage |
| container_accelerator_memory_used_bytes | GPU usage |
| container_memory_usage_bytes | Memory usage |
| container_network_transmit_bytes_total | Network usage |
| container_fs_write_seconds_total | I/O usage |
| up | Keep-Alive check (maintained by Prometheus on its own with its `up` metric detailed in the documentation [here](https://prometheus.io/docs/concepts/jobs_instances/#automatically-generated-labels-and-time-series))) |
| common_operator_is_leader | Whether this client is the leader of this common operator client set |

Note that some of the above metrics are derived from [cAdvisor](https://github.com/google/cadvisor) kubelet
integration which reports to Prometheus through our prometheus-operator installation.

## Job Metrics

The following metrics related to the lifecycle of jobs will be reported:

| Metric Name | Description |
| ------------ | ------- |
| created_jobs_total | The total number of created jobs |
| deleted_jobs_total | The total number of deleted jobs |
| completed_jobs_total | The total number of completed jobs |
| restarted_jobs_total | The total number of restarted jobs |
| pending_jobs_total | The total number of pending jobs |
| failed_jobs_total | The total number of failed jobs |

## Service Metrics

The following metrics related to the lifecycle of services will be reported:

| Metric Name | Description |
| ------------ | ------- |
| succeeded_service_creations_total | The total number of succeeded service creations |
| failed_service_creations_total | The total number of failed service creations |
| restarted_service_creations_total | The total number of restarted service creations |
| service_patches_total | The total number of service patches |
| deleted_services_total | The total number of deleted services |

## Scheduling Metrics

The following metrics related to scheduling will be reported:

| Metric Name | Description |
| ------------ | ------- |
| created_pod_disruption_policies_total | The total number of created pod disruption policies |
| deleted_pod_disruption_policies_total | The total number of deleted pod disruption policies |
| created_pod_groups_total | The total number of created pod groups |
| deleted_pod_groups_total | The total number of deleted pod groups |
