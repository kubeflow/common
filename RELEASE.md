# Kubeflow Common Releases

## Release v0.3.1

* Skip some logics for terminated job and add PodGroup reconcile loop ([#93](https://github.com/jazzsir/common/pull/93), @Jeffwan)
* Provides more flexibility for customizations ([#91](https://github.com/jazzsir/common/pull/91), @Jeffwan)
* Change expectation package log level to debug ([#90](https://github.com/jazzsir/common/pull/90), @Jeffwan)
* Bump Kubernetes dependency to 1.16.9 ([#87](https://github.com/jazzsir/common/pull/87), @Jeffwan)
* Add release notes document ([#88](https://github.com/jazzsir/common/pull/88), @terrytangyuan)

## Release v0.3.0

* Simplify interfaces and remove useless interfaces ([#85](https://github.com/jazzsir/common/pull/85), @Jeffwan)
* Add scheduling related Prometheus metrics ([#84](https://github.com/jazzsir/common/pull/84), @terrytangyuan)
* Reformat code with go fmt and check this on Travis CI ([#83](https://github.com/jazzsir/common/pull/83), @terrytangyuan)
* Add Prometheus metrics for pod created/deleted/failed ([#82](https://github.com/jazzsir/common/pull/82), @terrytangyuan)
* Fix crash when ReplicaSpec.Replicas is nil ([#81](https://github.com/jazzsir/common/pull/81), @hustcat)
* Add @Jeffwan to OWNERS ([#80](https://github.com/jazzsir/common/pull/80), @Jeffwan)
* Edits on dev guide and rename it to CONTRIBUTING.md ([#79](https://github.com/jazzsir/common/pull/79), @terrytangyuan)
* Add proposal for Prometheus metrics coverage ([#77](https://github.com/jazzsir/common/pull/77), @terrytangyuan)
* Add DEVELOPMENT.md doc ([#78](https://github.com/jazzsir/common/pull/78), @Jeffwan)
* Create expectation package ([#74](https://github.com/jazzsir/common/pull/74), @Jeffwan)
* Add PodControlInterface and PodControllerRefManager ([#73](https://github.com/jazzsir/common/pull/73), @Jeffwan)
* Add Prometheus metrics for service creation ([#75](https://github.com/jazzsir/common/pull/75), @terrytangyuan)
* Update comments on the use of -ignore flag for goveralls ([#76](https://github.com/jazzsir/common/pull/76), @terrytangyuan)
* Move Pods/Services control interface to separate folder ([#72](https://github.com/jazzsir/common/pull/72), @Jeffwan)
* Import volcano api ([#62](https://github.com/jazzsir/common/pull/62), @hzxuzhonghu)

## Release 0.2.0

* Add links to issues on ROADMAP.md ([#69](https://github.com/jazzsir/common/pull/69), @terrytangyuan)
* Add distributed training operator roadmap ([#61](https://github.com/jazzsir/common/pull/61), @Jeffwan)
* Remove vendor dependencies ([#63](https://github.com/jazzsir/common/pull/63), @Jeffwan)
* Make container port optional ([#57](https://github.com/jazzsir/common/pull/57), @Jeffwan)
* Support the case user down scale replicas ([#58](https://github.com/jazzsir/common/pull/58), @Jeffwan)
* Enhance maintainability of operator common module ([#55](https://github.com/jazzsir/common/pull/55), @Jeffwan)
* Deprecate glog and move to commonutil logger ([#53](https://github.com/jazzsir/common/pull/53), @Jeffwan)

## Release 0.1.1

* Remove unnecessary comment on gometalinter ([#50](https://github.com/jazzsir/common/pull/50), @terrytangyuan)
* Update openapi tag for SDK generating ([#49](https://github.com/jazzsir/common/pull/49), @jinchihe)

## Release 0.1.0

* [hot-fix-43] change the job status api  ([#44](https://github.com/jazzsir/common/pull/44), @merlintang)
* Add readme ([#42](https://github.com/jazzsir/common/pull/42), @jian-he)
* GetDefaultContainerPortNumber to GetDefaultContainerPortName ([#41](https://github.com/jazzsir/common/pull/41), @merlintang)
* re-org api package ([#39](https://github.com/jazzsir/common/pull/39), @jian-he)
* Remove the podControl and serviceControl interfaces ([#36](https://github.com/jazzsir/common/pull/36), @jian-he)
* refine arguments of ControllerInterface.UpdateJobStatus ([#35](https://github.com/jazzsir/common/pull/35), @sperlingxx)
* Generate vendor directories ([#28](https://github.com/jazzsir/common/pull/28), @terrytangyuan)
* Add skeleton code for reconcile service ([#25](https://github.com/jazzsir/common/pull/25), @merlintang)
* Correct function names in the comment ([#32](https://github.com/jazzsir/common/pull/32), @terrytangyuan)
* Make UpdateJobConditions public to be used by custom operators ([#30](https://github.com/jazzsir/common/pull/30), @jian-he)
* Unify the label keys ([#29](https://github.com/jazzsir/common/pull/29), @jian-he)
* Add utility methods for job reconciliation ([#24](https://github.com/jazzsir/common/pull/24), @terrytangyuan)
* test: Add util/train/train_util_test.go ([#26](https://github.com/jazzsir/common/pull/26), @gaocegege)
* chore: Fix package name ([#27](https://github.com/jazzsir/common/pull/27), @gaocegege)
* Add utility methods for pod creation ([#17](https://github.com/jazzsir/common/pull/17), @terrytangyuan)
* Remove mentions of tensorflow in test job ([#21](https://github.com/jazzsir/common/pull/21), @terrytangyuan)
* Fix incorrect name for restart policy exit code ([#20](https://github.com/jazzsir/common/pull/20), @terrytangyuan)
* Update go.sum changed by golangci-lint ([#18](https://github.com/jazzsir/common/pull/18), @terrytangyuan)
* Added .gitignore file ([#16](https://github.com/jazzsir/common/pull/16), @terrytangyuan)
* Add more utility functions for job and status ([#14](https://github.com/jazzsir/common/pull/14), @jian-he)
* Added missing generated lines in go.sum ([#15](https://github.com/jazzsir/common/pull/15), @terrytangyuan)
* Added schedulingPolicy api. ([#13](https://github.com/jazzsir/common/pull/13), @k82cn)
* Define interfaces for common job operators ([#12](https://github.com/jazzsir/common/pull/12), @jian-he)
* Add Travis badge and Go report card ([#9](https://github.com/jazzsir/common/pull/9), @terrytangyuan)
* Enable Travis CI on kubeflow/common ([#6](https://github.com/jazzsir/common/pull/6), @richardsliu)
* Common job controller library ([#5](https://github.com/jazzsir/common/pull/5), @richardsliu)
* Add terrytangyuan to OWNERS ([#4](https://github.com/jazzsir/common/pull/4), @richardsliu)
* Add common types to kubeflow/common ([#2](https://github.com/jazzsir/common/pull/2), @richardsliu)
* Create LICENSE ([#1](https://github.com/jazzsir/common/pull/1), @richardsliu)
