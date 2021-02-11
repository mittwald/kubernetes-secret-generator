package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type StringSpec struct {
	Initialized bool              `json:"initialized"`
	Length      string            `json:"length"`
	Encoding    string            `json:"encoding"`
	FieldNames  []string          `json:"fieldNames"`
	Data        map[string]string `json:"data"`
}

type String struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec StringSpec `json:"spec"`
}

type StringList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []String `json:"items"`
}
