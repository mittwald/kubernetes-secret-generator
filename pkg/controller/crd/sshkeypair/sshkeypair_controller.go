package sshkeypair

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
var reqLogger logr.Logger

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
	err := r.client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		// if instance is not found don#t requeue and don't return error, else requeue and return error
		return crd.CheckError(err)
	}

	existing := &v1.Secret{}
	err = r.client.Get(ctx, request.NamespacedName, existing)
	// secret not found, create new one
	if apierrors.IsNotFound(err) {
		return r.createNewSecret(ctx, instance, reqLogger)
	} else {
		// other error occurred
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	return r.updateSecret(ctx, existing, instance)

}

// updateSecret attempts to update an existing Secret object with new values. Secret will only be updated,
// if it is owned by a SSHKeyPair CR.
func (r *ReconcileSSHKeyPair) updateSecret(ctx context.Context, existing *v1.Secret, instance *v1alpha1.SSHKeyPair) (reconcile.Result, error) {
	// check if secret was created by this cr
	var err error

	// Check if Secret is owned by SSHKeyPair, otherwise do nothing
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

	// get config values from instance
	length := instance.Spec.Length
	regenerate := instance.Spec.ForceRegenerate
	data := instance.Spec.Data
	instancePrivateKey := instance.Spec.PrivateKey

	existingPublicKey := existing.Data[secret.SecretFieldPublicKey]
	existingPrivateKey := existing.Data[secret.SecretFieldPrivateKey]

	if len(existingPrivateKey) == 0 && len(instancePrivateKey) > 0 {
		existingPrivateKey = []byte(instancePrivateKey)
	}
	// generate key pair or restore public key if private key exists
	var keyPair secret.SSHKeypair
	keyPair, err = restoreOrGenerateKeyPair(existingPrivateKey, existingPublicKey, length, regenerate)
	if err != nil {
		return reconcile.Result{RequeueAfter: time.Second * 30}, err
	}

	targetSecret := existing.DeepCopy()

	// update data properties
	for key := range data {
		if string(targetSecret.Data[key]) == "" || regenerate {
			targetSecret.Data[key] = []byte(data[key])
		}
	}

	targetSecret.Data[secret.SecretFieldPublicKey] = keyPair.PublicKey
	targetSecret.Data[secret.SecretFieldPrivateKey] = keyPair.PrivateKey

	return r.clientUpdateSecret(ctx, targetSecret, instance)
}

// createNewSecret creates a new ssh key pair from the provided values. The Secret's owner will be set
// as the SSHKeyPair that is being reconciled and a reference to the Secret will be stored in
// the CR's status
func (r *ReconcileSSHKeyPair) createNewSecret(ctx context.Context, instance *v1alpha1.SSHKeyPair, reqLogger logr.Logger) (reconcile.Result, error) {
	values := make(map[string][]byte)

	// get config values from instance
	length := instance.Spec.Length
	secretType := instance.Spec.Type
	data := instance.Spec.Data
	instancePrivateKey := []byte(instance.Spec.PrivateKey)

	keyPair, err := generateKeyPair(instancePrivateKey, length)
	if err != nil {
		return reconcile.Result{RequeueAfter: time.Second * 30}, err
	}

	for key := range data {
		values[key] = []byte(data[key])
	}

	values[secret.SecretFieldPublicKey] = keyPair.PublicKey
	values[secret.SecretFieldPrivateKey] = keyPair.PrivateKey

	return r.clientCreateNewSecret(ctx, values, secretType, instance)
}

// generateKeyPair generates a new ssh key pair with keys of the given length
func generateKeyPair(privateKey []byte, length string) (secret.SSHKeypair, error) {
	secretLength, _, err := crd.ParseByteLength(secret.SecretLength(), length)
	if err != nil {
		reqLogger.Error(err, "could not parse length for new random string")
		return secret.SSHKeypair{}, errors.WithStack(err)
	}
	var keyPair secret.SSHKeypair

	// no private key set, generate new key pair
	if len(privateKey) == 0 {
		keyPair, err = secret.GenerateSSHKeypair(secretLength)
		if err != nil {
			reqLogger.Error(err, "could not generate ssh key pair")
			return secret.SSHKeypair{}, errors.WithStack(err)
		}
	} else {
		// use private key to regenerate public key
		var publicKey []byte
		publicKey, err = regeneratePublicKey(privateKey)
		keyPair.PrivateKey = privateKey
		keyPair.PublicKey = publicKey
	}

	return keyPair, nil
}

// getSecretRefAndSetStatus fetches the object reference for desiredSecret and writes it into the status of instance
func (r *ReconcileSSHKeyPair) getSecretRefAndSetStatus(ctx context.Context, desiredSecret *v1.Secret, instance *v1alpha1.SSHKeyPair) error {
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

// clientCreateNewSecret creates a new Secret resource, uses the client to save it to the cluster and gets its resource
// ref to set the status of instance
func (r *ReconcileSSHKeyPair) clientCreateNewSecret(ctx context.Context, values map[string][]byte, secretType string,
	instance *v1alpha1.SSHKeyPair) (reconcile.Result, error) {
	desiredSecret, err := crd.NewSecret(instance, values, secretType)
	if err != nil {
		// unable to set ownership of secret
		return reconcile.Result{Requeue: true}, err
	}

	err = r.client.Create(context.Background(), desiredSecret)
	if err != nil {
		// secret has been created at some point during this reconcile, retry
		return reconcile.Result{Requeue: true}, err
	}

	err = r.getSecretRefAndSetStatus(ctx, desiredSecret, instance)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}
	return reconcile.Result{}, nil
}

// clientUpdateSecret updates a Secret resource, uses the client to save it to the cluster and gets its resource
// ref to set the status of instance
func (r *ReconcileSSHKeyPair) clientUpdateSecret(ctx context.Context, targetSecret *v1.Secret, instance *v1alpha1.SSHKeyPair) (reconcile.Result, error) {
	err := r.client.Update(ctx, targetSecret)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	err = r.getSecretRefAndSetStatus(ctx, targetSecret, instance)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}
	return reconcile.Result{}, nil
}

func restoreOrGenerateKeyPair(existingPrivateKey []byte, existingPublicKey []byte, length string, regenerate bool) (secret.SSHKeypair, error) {
	var keyPair secret.SSHKeypair
	if len(existingPrivateKey) > 0 && !regenerate {
		if len(existingPublicKey) == 0 {
			var err error
			existingPublicKey, err = regeneratePublicKey(existingPrivateKey)
			if err != nil {
				return secret.SSHKeypair{}, err
			}
		}
		// public key was either regenerated or no regeneration necessary, in that case return existing values
		keyPair.PrivateKey = existingPrivateKey
		keyPair.PublicKey = existingPublicKey
		return keyPair, nil
	}
	var err error
	// At this point regenerate is either true, or the private key has length 0. To
	// give priority to regenerate = true, the key is set the key to an empty slice
	keyPair, err = generateKeyPair([]byte{}, length)
	if err != nil {
		return secret.SSHKeypair{}, err
	}
	return keyPair, nil
}

// regeneratePublicKey regenerates the public key from a given private key
func regeneratePublicKey(existingPrivateKey []byte) ([]byte, error) {
	var existingPublicKey []byte

	// restore public key if private key exists
	rsaKey, err := secret.PrivateKeyFromPEM(existingPrivateKey)
	if err != nil {
		reqLogger.Error(err, "could not get private key from pem")
		return []byte{}, errors.WithStack(err)
	}

	existingPublicKey, err = secret.SshPublicKeyForPrivateKey(rsaKey)
	if err != nil {
		reqLogger.Error(err, "could not get private key from pem")
		return []byte{}, errors.WithStack(err)
	}
	return existingPublicKey, nil
}
