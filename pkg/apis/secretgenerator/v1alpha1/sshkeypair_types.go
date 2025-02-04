package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SSHKeyPairSpec defines the desired state of SSHKeyPair
type SSHKeyPairSpec struct {
	// +optional
	Length string `json:"length,omitempty"`
	// +optional
	PrivateKey string `json:"privateKey,omitempty"`
	// +optional
	PrivateKeyField string `json:"privateKeyField,omitempty"`
	// +optional
	PublicKeyField string `json:"publicKeyField,omitempty"`
	// +optional
	Type string `json:"type,omitempty"`
	// +optional
	Data map[string]string `json:"data,omitempty"`
	// +optional
	ForceRegenerate bool `json:"forceRegenerate,omitempty"`
}

// SSHKeyPairStatus defines the observed state of SSHKeyPair
type SSHKeyPairStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Secret *v1.ObjectReference `json:"secret,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SSHKeyPair is the Schema for the sshkeypairs API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=sshkeypairs,scope=Namespaced
type SSHKeyPair struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SSHKeyPairSpec   `json:"spec,omitempty"`
	Status SSHKeyPairStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SSHKeyPairList contains a list of SSHKeyPair
type SSHKeyPairList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SSHKeyPair `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SSHKeyPair{}, &SSHKeyPairList{})
}

func (in *SSHKeyPairList) GetTypeMeta() metav1.TypeMeta {
	return in.TypeMeta
}

func (in *SSHKeyPairList) SetTypeMeta(meta metav1.TypeMeta) {
	in.TypeMeta = meta
}

func (in *SSHKeyPairList) GetListMeta() metav1.ListMeta {
	return in.ListMeta
}

func (in *SSHKeyPairList) SetListMeta(meta metav1.ListMeta) {
	in.ListMeta = meta
}

func (in *SSHKeyPair) GetPrivateKeyField() string {
	if in.Spec.PrivateKeyField != "" {
		return in.Spec.PrivateKeyField
	}
	return "ssh-privatekey"
}

func (in *SSHKeyPair) GetPublicKeyField() string {
	if in.Spec.PublicKeyField != "" {
		return in.Spec.PublicKeyField
	}
	return "ssh-publickey"
}

func (in *SSHKeyPair) GetStatus() SecretStatus {
	return &in.Status
}

func (in *SSHKeyPair) GetType() string {
	return in.Spec.Type
}

func (in *SSHKeyPairStatus) GetSecret() *v1.ObjectReference {
	return in.Secret
}

func (in *SSHKeyPairStatus) SetSecret(secret *v1.ObjectReference) {
	in.Secret = secret
}
