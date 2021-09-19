// Copyright 2021 The Kubeflow Authors
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

package labels

import (
	"errors"
	v1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"strconv"
)

// TODO(#149): Remove deprecated labels.

func ReplicaIndex(labels map[string]string) (int, error) {
	v, ok := labels[v1.ReplicaIndexLabel]
	if !ok {
		v, ok = labels[v1.ReplicaIndexLabelDeprecated]
		if !ok {
			return 0, errors.New("replica index label not found")
		}
	}
	return strconv.Atoi(v)
}

func SetReplicaIndex(labels map[string]string, idx int) {
	SetReplicaIndexStr(labels, strconv.Itoa(idx))
}

func SetReplicaIndexStr(labels map[string]string, idx string) {
	labels[v1.ReplicaIndexLabel] = idx
	labels[v1.ReplicaIndexLabelDeprecated] = idx
}

func ReplicaType(labels map[string]string) (string, error) {
	v, ok := labels[v1.ReplicaTypeLabel]
	if !ok {
		v, ok = labels[v1.ReplicaTypeLabelDeprecated]
		if !ok {
			return "", errors.New("replica type label not found")
		}
	}
	return v, nil
}

func SetReplicaType(labels map[string]string, rt string) {
	labels[v1.ReplicaTypeLabel] = rt
	labels[v1.ReplicaTypeLabelDeprecated] = rt
}

func HasKnownLabels(labels map[string]string, groupName string) bool {
	_, has := labels[v1.OperatorNameLabel]
	return has || labels[v1.GroupNameLabelDeprecated] == groupName
}

func SetJobRole(labels map[string]string, role string) {
	labels[v1.JobRoleLabel] = role
	labels[v1.JobRoleLabelDeprecated] = role
}
