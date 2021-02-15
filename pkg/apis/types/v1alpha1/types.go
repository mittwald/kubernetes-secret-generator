package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
)

type ReconcilerState string

type SecretStatus struct {
	Secret *v1.ObjectReference `json:"secret,omitempty"`
}

func (in *SecretStatus) GetSecret() *v1.ObjectReference {
	return in.Secret
}

func (in *SecretStatus) SetSecret(secret *v1.ObjectReference) {
	in.Secret = secret
}
