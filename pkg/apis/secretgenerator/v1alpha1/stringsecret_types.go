package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// StringSecretSpec defines the desired state of StringSecret
type StringSecretSpec struct {
	// +optional
	Type string `json:"type,omitempty"`
	// +optional
	Data map[string]string `json:"data,omitempty"`
	// +optional
	ForceRegenerate bool    `json:"forceRegenerate,omitempty"`
	Fields          []Field `json:"fields"`
}

type Field struct {
	FieldName string `json:"fieldName,omitempty"`
	Type      string `json:"type,omitempty"`
	Encoding  string `json:"encoding,omitempty"`
	Length    string `json:"length,omitempty"`
}

// StringSecretStatus defines the observed state of StringSecret
type StringSecretStatus struct {
	Secret *v1.ObjectReference `json:"secret,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StringSecret is the Schema for the stringsecrets API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=stringsecrets,scope=Namespaced
type StringSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              StringSecretSpec   `json:"spec,omitempty"`
	Status            StringSecretStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StringSecretList contains a list of StringSecret
type StringSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StringSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StringSecret{}, &StringSecretList{})
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

func (in *StringSecret) GetStatus() SecretStatus {
	return &in.Status
}

func (in *StringSecret) GetType() string {
	return in.Spec.Type
}

func (in *StringSecretStatus) GetSecret() *v1.ObjectReference {
	return in.Secret
}

func (in *StringSecretStatus) SetSecret(secret *v1.ObjectReference) {
	in.Secret = secret
}
