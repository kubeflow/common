// Copyright 2019 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1

import (
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
)

const (
	// EnvKubeflowNamespace is ENV for kubeflow namespace specified by user.
	EnvKubeflowNamespace = "KUBEFLOW_NAMESPACE"

	// DefaultPortName is name of the port used to communicate between workers.
	DefaultPortName = "job-port"
	// DefaultContainerName is the name of the TestJob container.
	DefaultContainerName = "test-container"
	// DefaultPort is default value of the port.
	DefaultPort = 2222
	// DefaultRestartPolicy is default RestartPolicy for TFReplicaSpec.
	DefaultRestartPolicy = commonv1.RestartPolicyNever
)
