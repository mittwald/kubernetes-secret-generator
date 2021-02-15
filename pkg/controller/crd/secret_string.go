package crd

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/mittwald/kubernetes-secret-generator/pkg/controller/secret"
)

func NewSecret(ownerCR metav1.Object, values map[string][]byte, secretType string) (*corev1.Secret, error) {
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
	err := controllerutil.SetControllerReference(ownerCR, secret, scheme.Scheme)
	if err != nil {
		return nil, err
	}

	return secret, nil
}

func ParseByteLength(fallback int, length string) (int, bool, error) {
	isByteLength := false
	lengthString := strings.ToLower(length)
	if strings.HasSuffix(lengthString, secret.ByteSuffix) {
		isByteLength = true
	}
	intVal, err := strconv.Atoi(strings.TrimSuffix(lengthString, secret.ByteSuffix))
	if err != nil {
		return fallback, isByteLength, errors.WithStack(err)
	}
	secretLength := intVal

	return secretLength, isByteLength, nil
}

// CheckError if err is  'NotFound', returns nil and no requeue, else returns err
func CheckError(err error) (reconcile.Result, error) {
	if apierrors.IsNotFound(err) {
		// Request object not found, could have been deleted after reconcile request.
		// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
		// Return and don't requeue
		return reconcile.Result{}, nil
	}
	// Error reading the object - requeue the request.
	return reconcile.Result{Requeue: true}, err
}
