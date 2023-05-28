# Contributing Guide

This doc is the contributing guideline for Kubeflow/common developers.

## Git development workflow

We use the [GitHub flow](https://guides.github.com/introduction/flow/) for development. Please check it out to get familiar with the process.

## Before PR submission

Before submitting a pull request, please make sure the code passes all the tests and is free of lint errors. The following sections outlines the instructions.

### Build

```bash
# Build the package.
go build ./...
```

### Code formatting

```bash
# Format your code.
go fmt ./...
```

### Code generation

```bash
# Make sure to update the generated code if there are any API-level changes.
./hack/update-codegen.sh
```

```bash
# Make sure your API and client are update-to-date.
./hack/verify-codegen.sh
```

### Code from upstream Kubernetes

Some of the code is borrowed from upstream Kubernetes, such as [controller_utils.go](https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/controller_utils.go), which helps us remove the direct dependency on Kubernetes. For more background on this, please check out the discussions in [issue #48](https://github.com/jazzsir/common/issues/48). In addition, the following folders also contain some auxiliary codes to help us easily build the operators:

- [control](./pkg/controller.v1/control)
- [expectation](./pkg/controller.v1/expectation)

*Note: Please don't edit these files. If you encounter any issues, please file an issue [here](https://github.com/jazzsir/common/issues).*

We have a long-term plan to move them to [kubernetes/client-go](https://github.com/kubernetes/client-go). See issue [kubernetes/client-go/issues/332](https://github.com/kubernetes/client-go/issues/332) for more details.

## Additional Guideline for Maintainers

If you are one of the maintainers of the repo, please check out the following additional guidelines.

### Commit style guide

1. Please always use bot to merge PRs and never manually merge PRs.

2. Release notes are generated from commits so writing good commit messages are important. Below is an example:

```md
Enhance maintainability of operator common module (#55, @Jeffwan)
Add proposal for Prometheus metrics coverage (#77, @terrytangyuan)
```
