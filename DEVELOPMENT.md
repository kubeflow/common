## Development Guidelines

This doc is a development guidance for Kubeflow/common developers.

## Git development cycle

We use [git flow](https://guides.github.com/introduction/flow/) workflow. The steps are

1. Fork `kubeflow/common` repo. 

1. Clone your fork under GOPATH,  add the `kubeflow/common` repository as a remote.

    ```bash
    mkdir -p ${go env GOPATH}/src/github.com/kubeflow
    cd ${go env GOPATH}/src/github.com/kubeflow
    
    git clone https://github.com/YOUR_USERNAME/common
    cd common
    git remote add upstream https://github.com/kubeflow/common
    
    git checkout -b your_feature_branch
    git fetch upstream master
    git rebase upstream/your_feature_branch
    
    ```

1. Make you changes and commit your work.

1. Send a pull request (PR) on GitHub. As we use squash merge, you can create multiple commit. You can also update a pull request using the the `git push -f` option as well. Please help to make your commits clean and meaningful.

## Before PR submission 

Before submitting a pull request, make sure the code passes all the tests and is clean of lint errors:

### Build

```bash
# cd into the root directory
go build ./...
```

### Format

```bash
# format your code
go fmt ./...
```

### Code generation

```bash
# make sure you generate code if you make api level changes
./hack/update-codegen.sh
```

```bash
# check your api and client are update to date.
./hack/verify-codegen.sh
```

### Code from upstream Kubernetes

Following folder includes some auxiliary codes to help us easily build operator. 
Codes are from Kubernetes internal package [controller_utils.go](https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/controller_utils.go),
This help us remove directly dependency on Kubernetes. If we want to see more details, please check issue [#48](https://github.com/kubeflow/common/issues/48).

- [control](./pkg/controller.v1/control)
- [expectation](./pkg/controller.v1/expectation) 

Note: Please don't edit these files. If you have any issues on them, feel free to file an issue.

We have a long term plan to move codes to [kubernetes/client-go](https://github.com/kubernetes/client-go). 
See issue [kubernetes/client-go/issues/332](https://github.com/kubernetes/client-go/issues/332) for more details.


## For repository owners

Here's some guidelines for repository owners


### commit style guide

1. Always use bot to merge PRs, never manually merge PRs please.

2. Release notes are generated from commits. A typical commit may looks something like:

```
Enhance maintainability of operator common module (#55, @Jeffwan)
Add proposal for Prometheus metrics coverage (#77, @terrytangyuan)
``` 
