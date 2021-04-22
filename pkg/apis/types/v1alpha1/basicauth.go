package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type BasicAuthSpec struct {
	Length          string            `json:"length,omitempty"`
	Username        string            `json:"username,omitempty"`
	Encoding        string            `json:"encoding,omitempty"`
	Type            string            `json:"type,omitempty"`
	Data            map[string]string `json:"data,omitempty"`
	ForceRegenerate bool              `json:"forceRegenerate,omitempty"`
}

type BasicAuth struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BasicAuthSpec `json:"spec"`
	Status SecretStatus  `json:"status"`
}

func (in *BasicAuth) GetStatus() *SecretStatus {
	return &in.Status
}

func (in *BasicAuth) GetType() string {
	return in.Spec.Type
}

type BasicAuthList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []BasicAuth `json:"items"`
}

func (in *BasicAuthList) GetTypeMeta() metav1.TypeMeta {
	return in.TypeMeta
}

func (in *BasicAuthList) SetTypeMeta(meta metav1.TypeMeta) {
	in.TypeMeta = meta
}

func (in *BasicAuthList) GetListMeta() metav1.ListMeta {
	return in.ListMeta
}

func (in *BasicAuthList) SetListMeta(meta metav1.ListMeta) {
	in.ListMeta = meta
}
