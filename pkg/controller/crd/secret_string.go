package crd

import (
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/mittwald/kubernetes-secret-generator/pkg/controller/secret"
)

func NewSecret(ownerCR metav1.Object, values map[string][]byte, secretType string) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ownerCR.GetName(),
			Namespace: ownerCR.GetNamespace(),
			Labels:    ownerCR.GetLabels(),
		},
		Data: values,
	}

	if secretType != "" {
		secret.Type = corev1.SecretType(secretType)
	}
	controllerutil.SetControllerReference(ownerCR, secret, scheme.Scheme)

	return secret
}

func ParseByteLength(fallback int, length string) (int, bool, error) {
	isByteLength := false
	secretLength := fallback
	lengthString := strings.ToLower(length)
	if strings.HasSuffix(lengthString, secret.ByteSuffix) {
		isByteLength = true
	}
	intVal, err := strconv.Atoi(strings.TrimSuffix(lengthString, secret.ByteSuffix))
	if err != nil {
		return 0, false, err
	}
	secretLength = intVal

	return secretLength, isByteLength, nil
}
