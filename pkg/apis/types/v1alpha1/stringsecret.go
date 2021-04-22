package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type StringSecretSpec struct {
	Length          string            `json:"length,omitempty"`
	Encoding        string            `json:"encoding,omitempty"`
	Type            string            `json:"type,omitempty"`
	FieldNames      []string          `json:"fieldNames,omitempty"`
	Data            map[string]string `json:"data,omitempty"`
	ForceRegenerate bool              `json:"forceRegenerate,omitempty"`
	Fields          []Field           `json:"fields,omitempty"`
}

type Field struct {
	FieldName string `json:"fieldName,omitempty"`
	Encoding  string `json:"encoding,omitempty"`
	Length    string `json:"length,omitempty"`
}

type StringSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StringSecretSpec `json:"spec"`
	Status SecretStatus     `json:"status"`
}

func (in *StringSecret) GetStatus() *SecretStatus {
	return &in.Status
}

func (in *StringSecret) GetType() string {
	return in.Spec.Type
}

type StringSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []StringSecret `json:"items"`
}

func (in *StringSecretList) GetTypeMeta() metav1.TypeMeta {
	return in.TypeMeta
}

func (in *StringSecretList) SetTypeMeta(meta metav1.TypeMeta) {
	in.TypeMeta = meta
}

func (in *StringSecretList) GetListMeta() metav1.ListMeta {
	return in.ListMeta
}

func (in *StringSecretList) SetListMeta(meta metav1.ListMeta) {
	in.ListMeta = meta
}
