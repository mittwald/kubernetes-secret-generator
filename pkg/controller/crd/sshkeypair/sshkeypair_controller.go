package sshkeypair

import (
	"context"
	"crypto/rsa"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/reference"
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

const Kind = "SSHKeyPair"

// Add creates a new SSHKeyPair Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileSSHKeyPair{client: mgr.GetClient(), scheme: mgr.GetScheme()}
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

	// Watch for changes to primary resource SSHKeyPair
	err = c.Watch(&source.Kind{Type: &v1alpha1.SSHKeyPair{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// Reconcile reads that state of the cluster for a SSHKeyPair object and makes changes based on the state read
// and what is in the Secret.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileSSHKeyPair) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling SSSHKeyPair")
	ctx := context.Background()

	// Fetch the SSHKeyPair instance
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

	// get config values from instance
	length := instance.Spec.Length
	secretType := instance.Spec.Type
	regenerate := instance.Spec.ForceRecreate

	values := make(map[string][]byte)

	secretLength, _, err := crd.ParseByteLength(secret.SecretLength(), length)
	if err != nil {
		reqLogger.Error(err, "could not parse length for new random string")
		return reconcile.Result{RequeueAfter: time.Second * 30}, err
	}

	existing := &v1.Secret{}
	err = r.client.Get(ctx, request.NamespacedName, existing)
	// secret not found, create new one
	if errors.IsNotFound(err) {
		keyPair, innerErr := secret.GenerateSSHKeypair(secretLength)
		if innerErr != nil {
			reqLogger.Error(innerErr, "could not generate ssh key pair")
			return reconcile.Result{RequeueAfter: time.Second * 30}, innerErr
		}

		values[secret.SecretFieldPublicKey] = keyPair.PublicKey
		values[secret.SecretFieldPrivateKey] = keyPair.PrivateKey

		var desiredSecret *v1.Secret
		desiredSecret, innerErr = crd.NewSecret(instance, values, secretType)
		if innerErr != nil {
			// unable to set ownership of secret
			return reconcile.Result{Requeue: true}, innerErr
		}

		innerErr = r.client.Create(context.Background(), desiredSecret)
		if innerErr != nil {
			return reconcile.Result{Requeue: true}, innerErr
		}

		// Get reference to created secret and store it in status
		innerErr = r.GetSecretRefAndSetStatus(ctx, desiredSecret, instance)
		if innerErr != nil {
			return reconcile.Result{Requeue: true}, innerErr
		}

		return reconcile.Result{}, nil
	} else {
		// other error occurred
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// check if secret was created by this cr
	existingOwnerRefs := existing.OwnerReferences
	ownedByCR := false
	for _, ref := range existingOwnerRefs {
		if ref.Kind != Kind {
			continue
		} else {
			ownedByCR = true
			break
		}
	}
	if !ownedByCR {
		// secret is not owned by cr, do nothing
		reqLogger.Info("secret not generated by this cr, skipping")
		return reconcile.Result{}, nil
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
		// Keys exist and regeneration is not forced, nothing to do
		return reconcile.Result{}, nil
	}

	var keyPair secret.SSHKeypair
	keyPair, err = secret.GenerateSSHKeypair(secretLength)
	if err != nil {
		reqLogger.Error(err, "could not generate ssh key pair")
		return reconcile.Result{RequeueAfter: time.Second * 30}, err
	}

	targetSecret.Data[secret.SecretFieldPublicKey] = keyPair.PublicKey
	targetSecret.Data[secret.SecretFieldPrivateKey] = keyPair.PrivateKey

	err = r.client.Update(ctx, targetSecret)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	// retrieve secret reference and set it as status in instance
	err = r.GetSecretRefAndSetStatus(ctx, targetSecret, instance)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	return reconcile.Result{}, nil
}

// GetSecretRefAndSetStatus fetches the object reference for desiredSecret and writes it into the status of instance
func (r *ReconcileSSHKeyPair) GetSecretRefAndSetStatus(ctx context.Context, desiredSecret *v1.Secret, instance *v1alpha1.SSHKeyPair) error {
	// get Secret reference for status
	stringRef, err := reference.GetReference(r.scheme, desiredSecret)
	if err != nil {
		return err
	}
	status := instance.GetStatus()
	status.SetSecret(stringRef)

	if err = r.client.Status().Update(ctx, instance); err != nil {
		return err
	}

	return nil
}
