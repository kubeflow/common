package job_controller

import (
	common "github.com/kubeflow/common/operator/v1"
	"github.com/prometheus/common/log"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"time"
)

func (jc *JobController) deletePodsAndServices(runPolicy *common.RunPolicy, job *runtime.Object, pods []*v1.Pod) error {
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
		if err := jc.PodControl.DeletePod(pod.Namespace, pod.Name, *job); err != nil {
			return err
		}
		// Pod and service have the same name, thus the service could be deleted using pod's name.
		if err := jc.ServiceControl.DeleteService(pod.Namespace, pod.Name, *job); err != nil {
			return err
		}
	}
	return nil
}

func (jc *JobController) cleanupJob(runPolicy *common.RunPolicy, jobStatus common.JobStatus, job *runtime.Object) error {
	currentTime := time.Now()
	ttl := runPolicy.TTLSecondsAfterFinished
	if ttl == nil {
		// do nothing if the cleanup delay is not set
		return nil
	}
	duration := time.Second * time.Duration(*ttl)
	if currentTime.After(jobStatus.CompletionTime.Add(duration)) {
		err := jc.Controller.DeleteJobHandler(job)
		if err != nil {
			log.Warnf("Cleanup Job error: %v.", err)
			return err
		}
		return nil
	}
	key, err := KeyFunc(job)
	if err != nil {
		log.Warnf("Couldn't get key for job object: %v", err)
		return err
	}
	jc.WorkQueue.AddRateLimited(key)
	return nil
}
