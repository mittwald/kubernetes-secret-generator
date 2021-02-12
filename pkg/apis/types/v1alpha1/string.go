package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type StringSpec struct {
	Length        string            `json:"length,omitempty"`
	Encoding      string            `json:"encoding,omitempty"`
	Type          string            `json:"type,omitempty"`
	FieldNames    []string          `json:"fieldNames,omitempty"`
	Data          map[string]string `json:"data,omitempty"`
	ForceRecreate bool              `json:"forceRecreate,omitempty"`
}

type StringStatus struct {
	State  ReconcilerState     `json:"state"`
	Secret *v1.ObjectReference `json:"secret,omitempty"`
}

func (in *StringStatus) GetState() ReconcilerState {
	return in.State
}

func (in *StringStatus) SetState(state ReconcilerState) {
	in.State = state
}

func (in *StringStatus) GetSecret() *v1.ObjectReference {
	return in.Secret
}

func (in *StringStatus) SetSecret(secret *v1.ObjectReference) {
	in.Secret = secret
}

type String struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StringSpec   `json:"spec"`
	Status StringStatus `json:"status"`
}

func (in *String) GetStatus() *StringStatus {
	return &in.Status
}

type StringList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []String `json:"items"`
}
