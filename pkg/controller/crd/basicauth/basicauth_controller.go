package basicauth

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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

var log = logf.Log.WithName("controller_basicauth_secret")
var reqLogger logr.Logger

const Kind = "BasicAuth"

// Add creates a new BasicAuth Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, NewReconciler(mgr))
}

// NewReconciler returns a new reconcile.Reconciler
func NewReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileBasicAuth{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

type ReconcileBasicAuth struct {
	// This Client, initialized using mgr.Client() above, is a split Client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("basicauth-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource BasicAuth
	err = c.Watch(&source.Kind{Type: &v1alpha1.BasicAuth{}}, &handler.EnqueueRequestForObject{}, crd.IgnoreStatusUpdatePredicate())
	if err != nil {
		return err
	}

	return nil
}

// Reconcile reads that state of the cluster for a BasicAuth object and makes changes based on the state read
// and what is in the Secret.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileBasicAuth) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger = log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling BasicAuth")
	ctx := context.Background()

	// fetch the BasicAuth instance
	instance := &v1alpha1.BasicAuth{}
	err := r.client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		// if instance is not found don't requeue and don't return error, else requeue and return error
		return crd.CheckError(err)
	}

	// attempt to fetch secret object described by this BasicAuth
	existing := &v1.Secret{}
	err = r.client.Get(ctx, request.NamespacedName, existing)
	if errors.IsNotFound(err) {
		// secret not found, create new one
		return r.createNewSecret(ctx, instance, reqLogger)
	}
	// check for other errors
	if err != nil {
		return reconcile.Result{}, err
	}
	// secret already exists, update if necessary
	return r.updateSecret(ctx, instance, existing, reqLogger)
}

// updateSecret attempts to update an existing Secret object with new values. Secret will only be updated,
// if it is owned by a BasicAuth cr.
func (r *ReconcileBasicAuth) updateSecret(ctx context.Context, instance *v1alpha1.BasicAuth, existing *v1.Secret, reqLogger logr.Logger) (reconcile.Result, error) {
	existingOwnerRefs := existing.OwnerReferences

	result := crd.IsOwnedByCorrectCR(reqLogger, existingOwnerRefs, Kind)
	if result != nil {
		return *result, nil
	}

	username := instance.Spec.Username
	length := instance.Spec.Length
	encoding := instance.Spec.Encoding
	regenerate := instance.Spec.ForceRegenerate
	data := instance.Spec.Data

	existingAuth := existing.Data[secret.FieldBasicAuthIngress]

	targetSecret := existing.DeepCopy()

	c := crd.Client{Client: r.client}

	if len(existingAuth) > 0 && !regenerate {
		// auth is set and regeneration is not forced, only update new data fields
		crd.UpdateData(data, targetSecret, regenerate)

		return c.ClientUpdateSecret(ctx, targetSecret, instance, r.scheme)
	}

	// either auth is not set or regeneration is forced, create new values

	// generate auth fields and populate targetSecret.Data with them
	err := secret.GenerateBasicAuthData(reqLogger, &secret.BasicAuthConstraints{Length: length, Encoding: encoding, Username: username}, targetSecret.Data)
	if err != nil {
		return reconcile.Result{RequeueAfter: time.Second * 30}, err
	}

	// add new/updated fields from crd spec
	crd.UpdateData(data, targetSecret, regenerate)

	return c.ClientUpdateSecret(ctx, targetSecret, instance, r.scheme)
}

// createNewSecret creates a new basic auth secret from the provided values. The Secret's owner will be set
// as the BasicAuth that is being reconciled and a reference to the Secret will be stored in the cr's status.
func (r *ReconcileBasicAuth) createNewSecret(ctx context.Context, instance *v1alpha1.BasicAuth, reqLogger logr.Logger) (reconcile.Result, error) {
	username := instance.Spec.Username
	length := instance.Spec.Length
	encoding := instance.Spec.Encoding
	data := instance.Spec.Data

	values := make(map[string][]byte)

	for key := range data {
		values[key] = []byte(data[key])
	}

	// generate auth fields and populate values with them
	err := secret.GenerateBasicAuthData(reqLogger, &secret.BasicAuthConstraints{Length: length, Encoding: encoding, Username: username}, values)
	if err != nil {
		return reconcile.Result{RequeueAfter: time.Second * 30}, err
	}

	c := crd.Client{Client: r.client}

	return c.ClientCreateSecret(ctx, values, instance, r.scheme)
}
