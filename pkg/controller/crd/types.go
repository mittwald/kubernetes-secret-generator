package crd

import (
	"context"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/reference"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/mittwald/kubernetes-secret-generator/pkg/apis/types/v1alpha1"
	"github.com/mittwald/kubernetes-secret-generator/pkg/controller/secret"
)

// NewSecret creates an new Secret with given owner-info, type and data values
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

// ParseByteLength parses the given length string into an integer length and determines whether the byte-length-suffix was set.
// In case paring fails, or the string is empty, the fallback will be returned, along with false.
func ParseByteLength(fallback int, length string) (int, bool, error) {
	isByteLength := false

	lengthString := strings.ToLower(length)
	if len(lengthString) == 0 {
		return fallback, isByteLength, nil
	}

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

// CheckError returns nil and no requeue if err is  'NotFound', else returns err
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

// IgnoreStatusUpdatePredicate is a reconciler predicate that will allow the reconciler to ignore updates that only change a cr's status
func IgnoreStatusUpdatePredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Evaluates to false if the object has been confirmed deleted.
			return !e.DeleteStateUnknown
		},
	}
}

type Client struct {
	client.Client
}

// ClientCreateSecret creates a new Secret resource, uses the client to save it to the cluster and gets its resource
// ref to set the status of instance
func (c *Client) ClientCreateSecret(ctx context.Context, values map[string][]byte,
	instance v1alpha1.APIObject, scheme *runtime.Scheme) (reconcile.Result, error) {
	desiredSecret, err := NewSecret(instance, values, instance.GetType())
	if err != nil {
		// unable to set ownership of secret
		return reconcile.Result{Requeue: true}, err
	}

	err = c.Create(context.Background(), desiredSecret)
	if err != nil {
		// secret has been created at some point during this reconcile, retry
		return reconcile.Result{Requeue: true}, err
	}

	err = c.getSecretRefAndSetStatus(ctx, desiredSecret, instance, scheme)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}
	return reconcile.Result{}, nil
}

// ClientUpdateSecret updates a Secret resource, uses the client to save it to the cluster and gets its resource
// ref to set the status of instance.
func (c *Client) ClientUpdateSecret(ctx context.Context, targetSecret *corev1.Secret, instance v1alpha1.APIObject, scheme *runtime.Scheme) (reconcile.Result, error) {
	err := c.Update(ctx, targetSecret)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	err = c.getSecretRefAndSetStatus(ctx, targetSecret, instance, scheme)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}
	return reconcile.Result{}, nil
}

// getSecretRefAndSetStatus fetches the object reference for desiredSecret and writes it into the status of instance.
func (c *Client) getSecretRefAndSetStatus(ctx context.Context, desiredSecret *corev1.Secret, instance v1alpha1.APIObject, scheme *runtime.Scheme) error {
	// get Secret reference for status
	stringRef, err := reference.GetReference(scheme, desiredSecret)
	if err != nil {
		return err
	}

	status := instance.GetStatus()
	status.SetSecret(stringRef)
	if err = c.Status().Update(ctx, instance); err != nil {
		return err
	}

	return nil
}
