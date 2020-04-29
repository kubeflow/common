# Prometheus Metrics Coverage

We plan to collect a rich set of metrics in kubeflow/common's `JobController` using [Prometheus](https://prometheus.io/).
The goal is to report generic metrics (e.g. metrics related to pods/jobs/services) during the lifecycle of `JobController` so that:

* Other operators built on top of it will automatically report Prometheus metrics without additional efforts;
* It is easier for users of Kubeflow distributed training operators to monitor operator performance and behaviors using consistent set of metrics for different distributed training operators.

This document outlines the list of Prometheus metrics we plan to cover in `JobController`. We follow the metric naming convention
outlined [here](https://prometheus.io/docs/practices/naming/) and the metric types supported by Prometheus [here](https://prometheus.io/docs/concepts/metric_types/).

## Pod Metrics

The following metrics related to the lifecycle of pods will be reported:

| Metric Name | Metric Type | Description |
| ----------- | ------------| ----------- |
| created_pods_total | Counter | The total number of created pods |
| restarted_pods_total | Counter | The total number of restarted pods |
| deleted_pods_total | Counter | The total number of deleted pods |
| failed_pods_total | Counter | The total number of failed pods |

The following metrics will be reported on each pod:

| Metric Name | Metric Type | Description |
| ----------- | ------------| ----------- |
| container_cpu_usage_seconds_total | Counter | Cumulative cpu time consumed in seconds |
| container_accelerator_memory_used_bytes | Gauge | Total accelerator memory allocated |
| container_memory_usage_bytes | Gauge | Current memory usage in bytes, including all memory regardless of when it was accessed |
| container_network_transmit_bytes_total | Counter | Cumulative count of bytes transmitted |
| container_fs_usage_bytes | Gauge | Number of bytes that are consumed by the container on this filesystem |
| up | Gauge | Keep-Alive check (maintained by Prometheus on its own with its `up` metric detailed in the documentation [here](https://prometheus.io/docs/concepts/jobs_instances/#automatically-generated-labels-and-time-series))) |

Note that some of the above metrics are derived from [cAdvisor](https://github.com/google/cadvisor) kubelet
integration which reports to Prometheus through our prometheus-operator installation.

## Job Metrics

The following metrics related to the lifecycle of jobs will be reported:

| Metric Name | Metric Type | Description |
| ----------- | ------------| ----------- |
| created_jobs_total | Counter | The total number of created jobs |
| deleted_jobs_total | Counter | The total number of deleted jobs |
| completed_jobs_total | Counter | The total number of completed jobs |
| restarted_jobs_total | Counter | The total number of restarted jobs |
| pending_jobs_total | Counter | The total number of pending jobs |
| failed_jobs_total | Counter | The total number of failed jobs |
| running_jobs_total | Counter | The total number of running jobs |

The following metrics related to the duration among various job phases will be reported:

| Metric Name | Metric Type | Description |
| ----------- | ------------| ----------- |
| from_created_to_completed_job_duration_seconds_total | Counter | The duration between job created and job completed in seconds |
| from_completed_to_deleted_job_duration_seconds_total | Counter | The duration between job completed and job deleted in seconds |
| from_failed_to_restarted_job_duration_seconds_total | Counter | The duration between job failed and job restarted in seconds |
| from_pending_to_running_job_duration_seconds_total | Counter | The duration between job pending and job running in seconds |

## Service Metrics

The following metrics related to the lifecycle of services will be reported:

| Metric Name | Metric Type | Description |
| ----------- | ------------| ----------- |
| succeeded_service_creations_total | Counter | The total number of succeeded service creations |
| failed_service_creations_total | Counter | The total number of failed service creations |
| restarted_service_creations_total | Counter | The total number of restarted service creations |
| service_patches_total | Counter | The total number of service patches |
| deleted_services_total | Counter | The total number of deleted services |

## Scheduling Metrics

The following metrics related to scheduling will be reported:

| Metric Name | Metric Type | Description |
| ----------- | ------------| ----------- |
| created_pod_disruption_policies_total | Counter | The total number of created pod disruption policies |
| deleted_pod_disruption_policies_total | Counter | The total number of deleted pod disruption policies |
| created_pod_groups_total | Counter | The total number of created pod groups |
| deleted_pod_groups_total | Counter | The total number of deleted pod groups |
