package v1

const (

	// ReplicaIndexLabel represents the label key for the replica-index, e.g. 0, 1, 2.. etc
	ReplicaIndexLabel = "training.kubeflow.org/replica-index"

	// ReplicaTypeLabel represents the label key for the replica-type, e.g. ps, worker etc.
	ReplicaTypeLabel = "training.kubeflow.org/replica-type"

	// OperatorNameLabel represents the label key for the operator name, e.g. tf-operator, mpi-operator, etc.
	OperatorNameLabel = "training.kubeflow.org/operator-name"

	// JobNameLabel represents the label key for the job name, the value is the job name.
	JobNameLabel = "training.kubeflow.org/job-name"

	// JobRoleLabel represents the label key for the job role, e.g. master.
	JobRoleLabel = "training.kubeflow.org/job-role"
)
