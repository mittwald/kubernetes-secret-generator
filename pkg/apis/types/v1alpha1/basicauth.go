package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type BasicAuthSpec struct {
	Length   string `json:"length"`
	Username string `json:"username"`
	Encoding string `json:"encoding"`
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
