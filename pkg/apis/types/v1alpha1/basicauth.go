package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type BasicAuthSpec struct {
	Length        string `json:"length,omitempty"`
	Username      string `json:"username,omitempty"`
	Encoding      string `json:"encoding,omitempty"`
	Type          string `json:"type,omitempty"`
	ForceRecreate bool   `json:"forceRecreate,omitempty"`
}

type BasicAuth struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec BasicAuthSpec `json:"spec"`
}

type BasicAuthList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []BasicAuth `json:"items"`
}
