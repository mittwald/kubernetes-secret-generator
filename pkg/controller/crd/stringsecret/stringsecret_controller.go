package stringsecret

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/reference"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

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

var log = logf.Log.WithName("controller_string_secret")

const Kind = "StringSecret"

// Add creates a new StringSecret Secret Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileStringSecret{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

type MyCRStatus struct {
	Status     string      `json:"status,omitempty"`
	LastUpdate metav1.Time `json:"lastUpdate,omitempty"`
	Reason     string      `json:"reason,omitempty"`
}

type ReconcileStringSecret struct {
	// This Client, initialized using mgr.Client() above, is a split Client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("string-secret-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}
	// Watch for changes to primary resource string
	err = c.Watch(&source.Kind{Type: &v1alpha1.StringSecret{}}, &handler.EnqueueRequestForObject{}, ignoreDeletionPredicate())
	if err != nil {
		return err
	}

	return nil
}

func ignoreDeletionPredicate() predicate.Predicate {
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

// Reconcile reads that state of the cluster for a StringSecret object and makes changes based on the state read
// and what is in the StringSecret.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileStringSecret) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	fmt.Println("reconciling")
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling StringSecret")
	ctx := context.Background()
	// Fetch the StringSecret instance
	instance := &v1alpha1.StringSecret{}
	err := r.client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{Requeue: true}, err
	}

	existing := &v1.Secret{}
	err = r.client.Get(ctx, request.NamespacedName, existing)
	// secret not found, create new one
	if errors.IsNotFound(err) {
		return r.createNewSecret(ctx, instance, reqLogger)
	} else {
		// other error occurred
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	// no errors, so secret exists
	return r.UpdateSecret(ctx, instance, existing, reqLogger)
}

// updateSecret attempts to update an existing Secret object with new values. Secret will only be updated,
// if it is owned by a StringSecret CR.
func (r *ReconcileStringSecret) UpdateSecret(ctx context.Context, instance *v1alpha1.StringSecret, existing *v1.Secret, reqLogger logr.Logger) (reconcile.Result, error) {
	fieldNames := instance.Spec.FieldNames
	length := instance.Spec.Length
	encoding := instance.Spec.Encoding
	regenerate := instance.Spec.ForceRecreate

	secretLength, isByteLength, err := crd.ParseByteLength(secret.SecretLength(), length)
	if err != nil {
		reqLogger.Error(err, "could not parse length for new random string")
		return reconcile.Result{RequeueAfter: time.Second * 30}, err
	}

	values := make(map[string][]byte)

	// generate only empty fields if regenerate wasn't set to true
	for _, field := range fieldNames {
		if string(existing.Data[field]) == "" || regenerate {
			randomString, randErr := secret.GenerateRandomString(secretLength, encoding, isByteLength)
			if randErr != nil {
				reqLogger.Error(err, "could not generate new random string")
				return reconcile.Result{RequeueAfter: time.Second * 30}, err
			}
			values[field] = randomString
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

	targetSecret := existing.DeepCopy()

	// Add new keys to Secret.Data
	for key := range values {
		targetSecret.Data[key] = values[key]
	}

	return r.clientUpdateSecret(ctx, targetSecret, instance)
}

// createNewSecret creates a new string secret from the provided values. The Secret's owner will be set
// as the StringSecret that is being reconciled and a reference to the Secret will be stored in
// the CR's status
func (r *ReconcileStringSecret) createNewSecret(ctx context.Context, instance *v1alpha1.StringSecret, reqLogger logr.Logger) (reconcile.Result, error) {
	fieldNames := instance.Spec.FieldNames
	length := instance.Spec.Length
	encoding := instance.Spec.Encoding
	data := instance.Spec.Data
	secretType := instance.Spec.Type

	secretLength, isByteLength, err := crd.ParseByteLength(secret.SecretLength(), length)
	if err != nil {
		reqLogger.Error(err, "could not parse length for new random string")
		return reconcile.Result{RequeueAfter: time.Second * 30}, err
	}

	values := make(map[string][]byte)

	for key := range data {
		values[key] = []byte(data[key])
	}
	for _, field := range fieldNames {
		randomString, randErr := secret.GenerateRandomString(secretLength, encoding, isByteLength)
		if randErr != nil {
			reqLogger.Error(err, "could not generate new random string")
			return reconcile.Result{RequeueAfter: time.Second * 30}, err
		}
		values[field] = randomString
	}

	return r.clientCreateNewSecret(ctx, values, secretType, instance)
}

// getSecretRefAndSetStatus fetches the object reference for desiredSecret and writes it into the status of instance
func (r *ReconcileStringSecret) getSecretRefAndSetStatus(ctx context.Context, desiredSecret *v1.Secret, instance *v1alpha1.StringSecret) error {
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
func (r *ReconcileStringSecret) clientCreateNewSecret(ctx context.Context, values map[string][]byte, secretType string,
	instance *v1alpha1.StringSecret) (reconcile.Result, error) {
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
func (r *ReconcileStringSecret) clientUpdateSecret(ctx context.Context, targetSecret *v1.Secret, instance *v1alpha1.StringSecret) (reconcile.Result, error) {
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