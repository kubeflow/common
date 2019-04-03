// Copyright 2018 The Kubeflow Authors
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

package test_util

import (
	"time"

	common "github.com/kubeflow/common/operator/v1"
)

const (
	TestImageName = "test-image-for-kubeflow-common:latest"
	TestJobName = "test-job"
	LabelWorker   = "worker"

	SleepInterval = 500 * time.Millisecond
	ThreadCount   = 1

	// DefaultPortName is name of the port used to communicate between PS and
	// workers.
	DefaultPortName = "tfjob-port"
	// DefaultContainerName is the name of the TFJob container.
	DefaultContainerName = "tensorflow"
	// DefaultPort is default value of the port.
	DefaultPort = 2222
	// DefaultRestartPolicy is default RestartPolicy for TFReplicaSpec.
	DefaultRestartPolicy = common.RestartPolicyNever
)

var (
	AlwaysReady = func() bool { return true }
)
