package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// StringSecretSpec defines the desired state of TemplateSecret
type TemplateSecretSpec struct {
	// +optional
	ForceRegenerate bool              `json:"forceRegenerate,omitempty"`
	Data            map[string]string `json:"data,omitempty"`
	Fields          []TemplateField   `json:"fields"`
}

type TemplateField struct {
	FieldName string `json:"fieldName,omitempty"`
	Encoding  string `json:"encoding,omitempty"`
	Length    string `json:"length,omitempty"`
}

type TemplateSecretStatus struct {
	Secret *v1.ObjectReference `json:"secret,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TemplateSecret is the Schema for the templatesecret API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=templatesecrets,scope=Namespaced
type TemplateSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              TemplateSecretSpec   `json:"spec,omitempty"`
	Status            TemplateSecretStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TemplateSecretList contains a list of TemplateSecret
type TemplateSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TemplateSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TemplateSecret{}, &TemplateSecretList{})
}

func (in *TemplateSecretList) GetTypeMeta() metav1.TypeMeta {
	return in.TypeMeta
}

func (in *TemplateSecretList) SetTypeMeta(meta metav1.TypeMeta) {
	in.TypeMeta = meta
}

func (in *TemplateSecretList) GetListMeta() metav1.ListMeta {
	return in.ListMeta
}

func (in *TemplateSecretList) SetListMeta(meta metav1.ListMeta) {
	in.ListMeta = meta
}

func (in *TemplateSecret) GetStatus() SecretStatus {
	return &in.Status
}

func (in *TemplateSecret) GetType() string {
	return "Opaque"
}

func (in *TemplateSecretStatus) GetSecret() *v1.ObjectReference {
	return in.Secret
}

func (in *TemplateSecretStatus) SetSecret(secret *v1.ObjectReference) {
	in.Secret = secret
}
