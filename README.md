# Kubeflow common for operators

[![Build Status](https://github.com/jazzsir/common/actions/workflows/build.yml/badge.svg?branch=master)](https://github.com/jazzsir/common/actions/?query=workflow%3ABuild)
[![Go Report Card](https://goreportcard.com/badge/github.com/jazzsir/common)](https://goreportcard.com/report/github.com/jazzsir/common)

This repo contains the libraries for writing a custom job operators such as tf-operator and pytorch-operator.
To write a custom operator, user need to do following steps

- Generate operator skeleton using [kube-builder](https://github.com/kubernetes-sigs/kubebuilder) or [operator-sdk](https://github.com/operator-framework/operator-sdk)

- Define job crd and reuse common API. Check [test_job](test_job) for full example.

```go
import (
    commonv1 "github.com/jazzsir/common/pkg/apis/common/v1"
)

// reuse commonv1 api in your type.go
RunPolicy *commonv1.RunPolicy                              `json:"runPolicy,omitempty"`
TestReplicaSpecs map[TestReplicaType]*commonv1.ReplicaSpec `json:"testReplicaSpecs"`
```

- Write a custom controller that implements [controller interface](pkg/apis/common/v1/interface.go), such as the [TestJobController](test_job/controller.v1/test_job/test_job_controller.go) and instantiate a testJobController object
```go
 testJobController := TestJobController {
    ...
 }
```
- Instantiate a [JobController](pkg/controller.v1/common/job_controller.go) struct object and pass in the custom controller written in step 1 as a parameter
```go
import "github.com/jazzsir/common/pkg/controller.v1/common"

jobController := common.JobController {
    Controller: testJobController,
    Config:     v1.JobControllerConfiguration{EnableGangScheduling: false},
    Recorder:   recorder,
}
```
- Within you main reconcile loop, call the [JobController.ReconcileJobs](pkg/controller.v1/common/job.go) method.
```go
    reconcile(...) {
    	// Your main reconcile loop.
    	...
    	jobController.ReconcileJobs(...)
    	...
    }

```
Note that this repo is still under construction, API compatibility is not guaranteed at this point.

## API Reference

Please refer to the [API documentation](docs/api/generated.asciidoc).

The API files are located under `pkg/apis/common/v1`:

- [constants.go](pkg/apis/common/v1/constants.go): the constants such as label keys.
- [interface.go](pkg/apis/common/v1/interface.go): the interfaces to be implemented by custom controllers.
- [controller.go](pkg/controller.v1/common/job_controller.go): the main `JobController` that contains the `ReconcileJobs` API method to be invoked by user. This is the entrypoint of
the `JobController` logic. The rest of the code under `job_controller/` folder contains the core logic for the `JobController` to work, such as creating and managing worker pods, services, etc.
