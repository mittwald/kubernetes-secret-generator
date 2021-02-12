package v1alpha1

type ReconcilerState string

const (
	ReconcilerStatePending   ReconcilerState = "Pending"
	ReconcilerStateCompleted                 = "Success"
	ReconcilerStateError                     = "Error"
)
