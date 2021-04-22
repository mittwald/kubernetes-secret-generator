package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SSHKeyPairSpec struct {
	Length          string            `json:"length,omitempty"`
	PrivateKey      string            `json:"privateKey,omitempty"`
	Type            string            `json:"type,omitempty"`
	Data            map[string]string `json:"data,omitempty"`
	ForceRegenerate bool              `json:"forceRegenerate,omitempty"`
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

func (in *SSHKeyPair) GetType() string {
	return in.Spec.Type
}

type SSHKeyPairList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []SSHKeyPair `json:"items"`
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
