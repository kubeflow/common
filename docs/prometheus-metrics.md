# Prometheus Metrics Coverage

We plan to collect a rich set of metrics in kubeflow/common's `JobController` using [Prometheus](https://prometheus.io/).
The goal is to report generic metrics (e.g. metrics related to pods/jobs/services) during the lifecycle of `JobController` so that:

* Other operators built on top of it will automatically report Prometheus metrics without additional efforts;
* It is easier for users of Kubeflow distributed training operators to monitor operator performance and behaviors using consistent set of metrics for different distributed training operators.

This document outlines the list of Prometheus metrics we plan to cover in `JobController`.

## Pod Metrics

The following metrics related to the lifecycle of pods will be reported:

* The total number of created pods
* The total number of restarted pods
* The total number of deleted pods
* The total number of failed pods

The following metrics will be reported on each pod:

* CPU usage
* GPU usage
* Memory usage
* Network usage
* I/O usage
* Keep-Alive check
* Is-leader check

## Job Metrics

The following metrics related to the lifecycle of jobs will be reported:

* The total number of created jobs
* The total number of deleted jobs
* The total number of completed jobs
* The total number of restarted jobs
* The total number of pending jobs
* The total number of failed jobs

## Service Metrics

The following metrics related to the lifecycle of services will be reported:

* The total number of succeeded service creations
* The total number of failed service creations
* The total number of restarted service creations
* The total number of service patches
* The total number of deleted services

## Scheduling Metrics

The following metrics related to scheduling will be reported:

* The total number of created pod disruption policies
* The total number of deleted pod disruption policies
* The total number of created pod groups
* The total number of deleted pod groups
