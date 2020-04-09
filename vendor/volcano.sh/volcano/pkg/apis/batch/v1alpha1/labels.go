/*
Copyright 2017 The Volcano Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

const (
	// TaskSpecKey task spec key used in pod annotation
	TaskSpecKey = "volcano.sh/task-spec"
	// JobNameKey job name key used in pod annotation / labels
	JobNameKey = "volcano.sh/job-name"
	// JobNamespaceKey job namespace key
	JobNamespaceKey = "volcano.sh/job-namespace"
	// DefaultTaskSpec default task spec value
	DefaultTaskSpec = "default"
	// JobVersion job version key used in pod annotation
	JobVersion = "volcano.sh/job-version"
	// JobTypeKey job type key used in labels
	JobTypeKey = "volcano.sh/job-type"
	// PodgroupNamePrefix podgroup name prefix
	PodgroupNamePrefix = "podgroup-"
)
