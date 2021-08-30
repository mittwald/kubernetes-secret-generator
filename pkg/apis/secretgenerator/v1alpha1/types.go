package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type SecretStatus interface {
	GetSecret() *v1.ObjectReference
	SetSecret(secret *v1.ObjectReference)
}

type ReconcilerState string

type APIObject interface {
	GetStatus() SecretStatus
	GetType() string
	runtime.Object
	metav1.Object
}
