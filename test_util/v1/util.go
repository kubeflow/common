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
	"encoding/json"
	"strings"
	"testing"

	common "github.com/kubeflow/common/operator/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"

	testjobv1 "github.com/kubeflow/common/test_job/v1"
)

const (
	LabelGroupName = "group-name"
	LabelTestJobName = "test-job-name"
)

var (
	// KeyFunc is the short name to DeletionHandlingMetaNamespaceKeyFunc.
	// IndexerInformer uses a delta queue, therefore for deletes we have to use this
	// key function but it should be just fine for non delete events.
	KeyFunc   = cache.DeletionHandlingMetaNamespaceKeyFunc
	TestGroupName = testjobv1.GroupName
)

func GenLabels(jobName string) map[string]string {
	return map[string]string{
		LabelGroupName: TestGroupName,
		LabelTestJobName: strings.Replace(jobName, "/", "-", -1),
	}
}

func GenOwnerReference(testjob *testjobv1.TestJob) *metav1.OwnerReference {
	boolPtr := func(b bool) *bool { return &b }
	controllerRef := &metav1.OwnerReference{
		APIVersion:         testjobv1.SchemeGroupVersion.String(),
		Kind:               testjobv1.Kind,
		Name:               testjob.Name,
		UID:                testjob.UID,
		BlockOwnerDeletion: boolPtr(true),
		Controller:         boolPtr(true),
	}

	return controllerRef
}

// ConvertTFJobToUnstructured uses JSON to convert TFJob to Unstructured.
func ConvertTFJobToUnstructured(testJob *testjobv1.TestJob) (*unstructured.Unstructured, error) {
	var unstructured unstructured.Unstructured
	b, err := json.Marshal(testJob)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(b, &unstructured); err != nil {
		return nil, err
	}
	return &unstructured, nil
}

func GetKey(testJob *testjobv1.TestJob, t *testing.T) string {
	key, err := KeyFunc(testJob)
	if err != nil {
		t.Errorf("Unexpected error getting key for job %v: %v", testJob.Name, err)
		return ""
	}
	return key
}

func CheckCondition(testJob *testjobv1.TestJob, condition common.JobConditionType, reason string) bool {
	for _, v := range testJob.Status.Conditions {
		if v.Type == condition && v.Status == v1.ConditionTrue && v.Reason == reason {
			return true
		}
	}
	return false
}
