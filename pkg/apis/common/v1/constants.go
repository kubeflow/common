package v1

const (
	// TODO(#149): Remove deprecated labels.

	// ReplicaIndexLabel represents the label key for the replica-index, e.g. 0, 1, 2.. etc
	ReplicaIndexLabel = "training.kubeflow.org/replica-index"

	// ReplicaIndexLabelDeprecated represents the label key for the replica-index, e.g. the value is 0, 1, 2.. etc
	// DEPRECATED: Use ReplicaIndexLabel
	ReplicaIndexLabelDeprecated = "replica-index"

	// ReplicaTypeLabel represents the label key for the replica-type, e.g. ps, worker etc.
	ReplicaTypeLabel = "training.kubeflow.org/replica-type"

	// ReplicaTypeLabelDeprecated represents the label key for the replica-type, e.g. the value is ps , worker etc.
	// DEPRECATED: Use ReplicaTypeLabel
	ReplicaTypeLabelDeprecated = "replica-type"

	// OperatorNameLabel represents the label key for the operator name, e.g. tf-operator, mpi-operator, etc.
	OperatorNameLabel = "training.kubeflow.org/operator-name"

	// GroupNameLabelDeprecated represents the label key for group name, e.g. the value is kubeflow.org
	// DEPRECATED: Use OperatorNameLabel
	GroupNameLabelDeprecated = "group-name"

	// JobNameLabel represents the label key for the job name, the value is the job name.
	JobNameLabel = "training.kubeflow.org/job-name"

	// JobNameLabelDeprecated represents the label key for the job name, the value is job name
	// DEPRECATED: Use JobNameLabel
	JobNameLabelDeprecated = "job-name"

	// JobRoleLabel represents the label key for the job role, e.g. master.
	JobRoleLabel = "training.kubeflow.org/job-role"

	// JobRoleLabelDeprecated represents the label key for the job role, e.g. the value is master
	// DEPRECATED: Use JobRoleLabel
	JobRoleLabelDeprecated = "job-role"
)
