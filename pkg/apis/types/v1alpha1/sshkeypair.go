package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SSHKeyPairSpec struct {
	Length          string            `json:"length,omitempty"`
	PrivateKey      string            `json:"privateKey,omitempty"`
	Type            string            `json:"type,omitempty"`
	Data            map[string]string `json:"data,omitempty"`
	ForceRegenerate bool              `json:"forceRecreate,omitempty"`
}

type SSHKeyPair struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SSHKeyPairSpec `json:"spec"`
	Status SecretStatus   `json:"status"`
}

func (in *SSHKeyPair) GetStatus() *SecretStatus {
	return &in.Status
}

type SSHKeyPairList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []SSHKeyPair `json:"items"`
}
