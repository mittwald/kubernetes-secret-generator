package sshkeypair

import (
	"context"
	"crypto/rsa"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/mittwald/kubernetes-secret-generator/pkg/apis/types/v1alpha1"
	"github.com/mittwald/kubernetes-secret-generator/pkg/controller/crd"
	"github.com/mittwald/kubernetes-secret-generator/pkg/controller/secret"
)

var log = logf.Log.WithName("controller_ssh_secret")

// Add creates a new Secret Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileSSHKeyPair{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

type MyCRStatus struct {
	// +kubebuilder:validation:Enum=Success,Failure
	Status     string      `json:"status,omitempty"`
	LastUpdate metav1.Time `json:"lastUpdate,omitempty"`
	Reason     string      `json:"reason,omitempty"`
}

type ReconcileSSHKeyPair struct {
	// This Client, initialized using mgr.Client() above, is a split Client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("ssh-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Secret
	err = c.Watch(&source.Kind{Type: &v1alpha1.SSHKeyPair{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// Reconcile reads that state of the cluster for a Secret object and makes changes based on the state read
// and what is in the Secret.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileSSHKeyPair) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling SSSHKeyPair")
	ctx := context.TODO()
	// Fetch the Secret instance
	instance := &v1alpha1.SSHKeyPair{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	length := instance.Spec.Length
	secretType := instance.Spec.Type
	regenerate := instance.Spec.ForceRecreate

	values := make(map[string][]byte)

	secretLength, _, err := crd.ParseByteLength(secret.SecretLength(), length)
	if err != nil {
		// TODO errorstuff
	}

	existing := &v1.Secret{}
	err = r.client.Get(ctx, request.NamespacedName, existing)
	// secret not found, create new one
	if errors.IsNotFound(err) {
		var keyPair secret.SSHKeypair
		keyPair, err = secret.GenerateSSHKeypair(secretLength)

		values[secret.SecretFieldPublicKey] = keyPair.PublicKey
		values[secret.SecretFieldPrivateKey] = keyPair.PrivateKey

		desiredSecret := crd.NewSecret(instance, values, secretType)

		err = r.client.Create(context.Background(), desiredSecret)
		if err != nil {
			if errors.IsAlreadyExists(err) {
				// TODO do error stuff
			} else {
				return reconcile.Result{Requeue: true}, err
			}
		}
		return reconcile.Result{}, nil
	} else {
		// other error occurred
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	existingPublicKey := existing.Data[secret.SecretFieldPublicKey]
	existingPrivateKey := existing.Data[secret.SecretFieldPrivateKey]

	targetSecret := existing.DeepCopy()

	if len(existingPrivateKey) > 0 && !regenerate {
		if len(existingPublicKey) == 0 {
			// restore public key if private key exists
			var rsaKey *rsa.PrivateKey
			rsaKey, err = secret.PrivateKeyFromPEM(existingPrivateKey)
			if err != nil {
				return reconcile.Result{}, err
			}

			existingPublicKey, err = secret.SshPublicKeyForPrivateKey(rsaKey)
			if err != nil {
				return reconcile.Result{}, err
			}

			targetSecret.Data[secret.SecretFieldPublicKey] = existingPublicKey
		}
		return reconcile.Result{}, nil
	}

	keyPair, err := secret.GenerateSSHKeypair(secretLength)

	targetSecret.Data[secret.SecretFieldPublicKey] = keyPair.PublicKey
	targetSecret.Data[secret.SecretFieldPrivateKey] = keyPair.PrivateKey

	err = r.client.Update(ctx, targetSecret)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	// var stringRef *v1.ObjectReference
	// stringRef, err = reference.GetReference(r.scheme, targetSecret)
	// if err != nil {
	// 	return reconcile.Result{}, err
	// }

	// // set status information TODO do something useful with this
	// status := instance.GetStatus()
	// status.SetState(v1alpha1.ReconcilerStateCompleted)
	// status.SetSecret(stringRef)
	//
	// if err := r.client.Status().Update(ctx, instance); err != nil {
	// 	return reconcile.Result{}, err
	// }

	return reconcile.Result{}, nil
}
