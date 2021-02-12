package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type SSHKeyPairSpec struct {
	Length        string `json:"length,omitempty"`
	Type          string `json:"type,omitempty"`
	ForceRecreate bool   `json:"forceRecreate,omitempty"`
}

type SSHKeyPair struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec SSHKeyPairSpec `json:"spec"`
}

type SSHKeyPairList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []SSHKeyPair `json:"items"`
}
