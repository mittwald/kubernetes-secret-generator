package crd

import (
	"context"

	"github.com/go-logr/logr"
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

	"github.com/mittwald/kubernetes-secret-generator/pkg/apis/secretgenerator/v1alpha1"
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
	secret, err := NewSecret(instance, values, instance.GetType())
	if err != nil {
		// unable to set ownership of secret
		return reconcile.Result{}, err
	}
	return c.getSecretRefAndSetStatus(ctx, true, secret, instance, scheme)
}

// ClientCreateSecret stores a newly created Secret resource, uses the client to save it to the cluster and gets its resource
// ref to set the status of instance
func (c *Client) ClientStoreSecret(ctx context.Context, secret *corev1.Secret,
	instance v1alpha1.APIObject, scheme *runtime.Scheme) (reconcile.Result, error) {

	return c.getSecretRefAndSetStatus(ctx, true, secret, instance, scheme)
}

// ClientUpdateSecret updates a Secret resource, uses the client to save it to the cluster and gets its resource
// ref to set the status of instance.
func (c *Client) ClientUpdateSecret(ctx context.Context, secret *corev1.Secret, instance v1alpha1.APIObject, scheme *runtime.Scheme) (reconcile.Result, error) {
	return c.getSecretRefAndSetStatus(ctx, false, secret, instance, scheme)
}

// getSecretRefAndSetStatus fetches the object reference for desiredSecret and writes it into the status of instance.
func (c *Client) getSecretRefAndSetStatus(ctx context.Context, create bool, secret *corev1.Secret, instance v1alpha1.APIObject, scheme *runtime.Scheme) (reconcile.Result, error) {
	// get Secret reference for status
	var err error

	if create {
		err = c.Create(ctx, secret)
	} else {
		err = c.Update(ctx, secret)
	}

	if err != nil {
		return reconcile.Result{}, err
	}

	stringRef, err := reference.GetReference(scheme, secret)
	if err != nil {
		return reconcile.Result{}, err
	}

	status := instance.GetStatus()
	status.SetSecret(stringRef)
	if err = c.Status().Update(ctx, instance); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// updateData updates the given Secret's data property. If regenerate is false,
// only new keys will be added, existing keys will not be modified.
func UpdateData(data map[string]string, targetSecret *corev1.Secret, regenerate bool) {
	for key := range data {
		if string(targetSecret.Data[key]) == "" || regenerate {
			targetSecret.Data[key] = []byte(data[key])
		}
	}
}

func IsOwnedByCorrectCR(logger logr.Logger, ownerRefs []metav1.OwnerReference, kind string) bool {
	for _, ref := range ownerRefs {
		if ref.Kind == kind {
			return true
		}
	}

	// secret is not owned by correct cr, do nothing
	logger.Info("secret not generated by correct cr kind, skipping", "correctKind", kind)

	return false
}
