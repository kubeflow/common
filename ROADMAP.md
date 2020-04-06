# Kubeflow Distributed Training Operators 2020 Roadmap

This document outlines the main directions on the Kubeflow Distributed Training Operators in 2020.

## Maintenance and reliability

We will continue developing capabilities for better reliability, scaling, and maintenance of production distributed training experiences provided by operators.

* Enhance maintainability of operator common module https://github.com/kubeflow/common/issues/54
* Migrate operators to use kubeflow/common apis
* Graduate MPI Operator, Mxnet Operator and XGBoost Operator to v1

## Features

To take advatanges of other capabilities of job scheduler components, operators will expose more APIs for advanced scheduling. More features will be added to simplify usage like dynamic volume supports and git ops experiences. In order to make it easily used in the Kubeflow ecosystem, we can add more launcher KFP components for adoption.

* Support dynamic volume provisioning for distributed training jobs https://github.com/kubeflow/common/issues/19
* MLOPS - Allow user to submit jobs using Git repo without building container images.
* Add Job priority and Queue in SchedulingPolicy for advanced scheduling in common operator https://github.com/kubeflow/common/issues/46
* Add pipeline launcher components for different training jobs. https://github.com/kubeflow/pipelines/issues/3445


## Monitoring

* Provides a better common logger https://github.com/kubeflow/common/issues/60
* Expose generic prometheus metrics in common operators https://github.com/kubeflow/common/issues/22
* Centralized Job Dashboard for training jobs (Add metadata graph, model artifacts later)

## Performance

Continue to optimize reconciler performance and reduce latency to take actions on CR events.

* Performance optimization for 500 concurrent jobs and large scale completed jobs. https://github.com/kubeflow/tf-operator/issues/965  https://github.com/kubeflow/tf-operator/issues/1079
