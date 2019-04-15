package job_controller

import (
	"time"

	common "github.com/kubeflow/common/operator/v1"
	commonutil "github.com/kubeflow/common/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (jc *JobController) deletePodsAndServices(runPolicy *common.RunPolicy, job interface{}, pods []*v1.Pod) error {
	if len(pods) == 0 {
		return nil
	}

	// Delete nothing when the cleanPodPolicy is None.
	if *runPolicy.CleanPodPolicy == common.CleanPodPolicyNone {
		return nil
	}

	for _, pod := range pods {
		if *runPolicy.CleanPodPolicy == common.CleanPodPolicyRunning && pod.Status.Phase != v1.PodRunning {
			continue
		}
		if err := jc.Controller.DeletePod(job, pod); err != nil {
			return err
		}
		// Pod and service have the same name, thus the service could be deleted using pod's name.
		if err := jc.Controller.DeleteService(job, pod.Name, pod.Namespace); err != nil {
			return err
		}
	}
	return nil
}

func (jc *JobController) cleanupJob(runPolicy *common.RunPolicy, jobStatus common.JobStatus, job interface{}) error {
	currentTime := time.Now()
	metaObject, _ := job.(metav1.Object)
	ttl := runPolicy.TTLSecondsAfterFinished
	if ttl == nil {
		// do nothing if the cleanup delay is not set
		return nil
	}
	duration := time.Second * time.Duration(*ttl)
	if currentTime.After(jobStatus.CompletionTime.Add(duration)) {
		err := jc.Controller.DeleteJob(job)
		if err != nil {
			commonutil.LoggerForJob(metaObject).Warnf("Cleanup Job error: %v.", err)
			return err
		}
		return nil
	}
	key, err := KeyFunc(job)
	if err != nil {
		commonutil.LoggerForJob(metaObject).Warnf("Couldn't get key for job object: %v", err)
		return err
	}
	jc.WorkQueue.AddRateLimited(key)
	return nil
}
