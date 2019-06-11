# common

[![Build Status](https://travis-ci.com/kubeflow/common.svg?branch=master)](https://travis-ci.com/kubeflow/common/)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubeflow/common)](https://goreportcard.com/report/github.com/kubeflow/common)

This repo contains the libraries for writing a custom job operators such as tf-operator and pytorch-operator.
To write a custom operator, user need to do following three steps

- Write a custom controller that implements [controller interface](./job_controller/api/v1/controller.go), such as the [TestJobController](test_job/v1/test_job_controller.go) and instantiate a testJobController object
```
 testJobController := TestJobController {
    ...
 }
```
- Instantiate a [JobController](https://github.com/kubeflow/common/blob/master/job_controller/api/v1/controller.go#L44) struct object and pass in the custom controller written in step 1 as a parameter
```go
jobController := JobController {
    Controller: testJobController,
    Config:     v1.JobControllerConfiguration{EnableGangScheduling: false},
    Recorder:   recorder,
}
```
- Within you main reconcile loop, call the [JobController.ReconcileJobs](https://github.com/kubeflow/common/blob/master/job_controller/job.go#L72) method.
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
The API fies are located under `job_controller/api/v1`:

- [constants.go](./job_controller/api/v1/constants.go): the constants such as label keys.
- [interface.go](./job_controller/api/v1/controller.go): the interfaces to be implemented by custom controllers.
- [controller.go](./job_controller/api/v1/controller.go): the main `JobController` that contains the `ReconcileJobs` API method to be invoked by user. This is the entrypoint of
the JobController logic. The rest of code under `job_controller/` folder contains the core logic for the `JobController` to work, such as creating and managing worker pods, services etc.





