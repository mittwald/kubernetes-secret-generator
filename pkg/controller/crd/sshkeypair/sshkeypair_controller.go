package sshkeypair

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/mittwald/kubernetes-secret-generator/pkg/apis/secretgenerator/v1alpha1"
	"github.com/mittwald/kubernetes-secret-generator/pkg/controller/crd"
	"github.com/mittwald/kubernetes-secret-generator/pkg/controller/secret"
)

var log = logf.Log.WithName("controller_ssh_secret")
var reqLogger logr.Logger

const Kind = "SSHKeyPair"

// Add creates a new SSHKeyPair Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, NewReconciler(mgr))
}

// NewReconciler returns a new reconcile.Reconciler
func NewReconciler(mgr manager.Manager) reconcile.Reconciler {
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
	err = c.Watch(&source.Kind{Type: &v1alpha1.SSHKeyPair{}}, &handler.EnqueueRequestForObject{}, crd.IgnoreStatusUpdatePredicate())
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
	reqLogger = log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling SSSHKeyPair")
	ctx := context.Background()

	// fetch the SSHKeyPair instance
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
		return r.createNewSecret(ctx, instance)
	}
	// check for other errors
	if err != nil {
		return reconcile.Result{}, err
	}

	return r.updateSecret(ctx, existing, instance)
}

// updateSecret attempts to update an existing Secret object with new values. Secret will only be updated,
// if it is owned by a SSHKeyPair CR.
func (r *ReconcileSSHKeyPair) updateSecret(ctx context.Context, existing *v1.Secret, instance *v1alpha1.SSHKeyPair) (reconcile.Result, error) {
	// Check if Secret is owned by SSHKeyPair cr, otherwise do nothing
	existingOwnerRefs := existing.OwnerReferences

	if correct := crd.IsOwnedByCorrectCR(reqLogger, existingOwnerRefs, Kind); !correct {
		return reconcile.Result{}, nil
	}

	// get config values from instance
	length := instance.Spec.Length
	regenerate := instance.Spec.ForceRegenerate
	data := instance.Spec.Data
	instancePrivateKey := instance.Spec.PrivateKey
	privateKeyField := instance.GetPrivateKeyField()
	publicKeyField := instance.GetPublicKeyField()

	existingPrivateKey := existing.Data[privateKeyField]

	targetSecret := existing.DeepCopy()

	// if regeneration is forced or existing private key is empty use private key from spec
	if len(instancePrivateKey) > 0 && (len(existingPrivateKey) == 0 || regenerate) {
		targetSecret.Data[privateKeyField] = []byte(instancePrivateKey)
	}

	crd.UpdateData(data, targetSecret, regenerate)

	err := secret.GenerateSSHKeypairData(reqLogger, length, privateKeyField, publicKeyField, regenerate, targetSecret.Data)
	if err != nil {
		return reconcile.Result{RequeueAfter: time.Second * 30}, err
	}

	c := crd.Client{Client: r.client}

	return c.ClientUpdateSecret(ctx, targetSecret, instance, r.scheme)
}

// createNewSecret creates a new ssh key pair from the provided values. The Secret's owner will be set
// as the SSHKeyPair that is being reconciled and a reference to the Secret will be stored in
// the CR's status
func (r *ReconcileSSHKeyPair) createNewSecret(ctx context.Context, instance *v1alpha1.SSHKeyPair) (reconcile.Result, error) {
	values := make(map[string][]byte)

	// get config values from instance
	length := instance.Spec.Length
	data := instance.Spec.Data
	instancePrivateKey := []byte(instance.Spec.PrivateKey)
	privateKeyField := instance.GetPrivateKeyField()
	publicKeyField := instance.GetPublicKeyField()

	for key := range data {
		values[key] = []byte(data[key])
	}

	values[privateKeyField] = instancePrivateKey

	err := secret.GenerateSSHKeypairData(reqLogger, length, privateKeyField, publicKeyField, false, values)
	if err != nil {
		return reconcile.Result{RequeueAfter: time.Second * 30}, err
	}

	c := crd.Client{Client: r.client}

	return c.ClientCreateSecret(ctx, values, instance, r.scheme)
}
