package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BasicAuthSpec defines the desired state of BasicAuth
type BasicAuthSpec struct {
	// +optional
	Length   string `json:"length,omitempty"`
	Username string `json:"username"`
	// +optional
	Encoding string `json:"encoding,omitempty"`
	// +optional
	Data map[string]string `json:"data,omitempty"`
	// +optional
	ForceRegenerate bool `json:"forceRegenerate,omitempty"`
}

// BasicAuthStatus defines the observed state of BasicAuth
type BasicAuthStatus struct {
	Secret *v1.ObjectReference `json:"secret,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BasicAuth is the Schema for the basicauths API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=basicauths,scope=Namespaced
type BasicAuth struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BasicAuthSpec   `json:"spec,omitempty"`
	Status BasicAuthStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BasicAuthList contains a list of BasicAuth
type BasicAuthList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BasicAuth `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BasicAuth{}, &BasicAuthList{})
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

func (in *BasicAuth) GetStatus() SecretStatus {
	return &in.Status
}

func (in *BasicAuth) GetType() string {
	return "Opaque"
}

func (in *BasicAuthStatus) GetSecret() *v1.ObjectReference {
	return in.Secret
}

func (in *BasicAuthStatus) SetSecret(secret *v1.ObjectReference) {
	in.Secret = secret
}
