package basicauth

import (
	"context"
	"time"

	"golang.org/x/crypto/bcrypt"
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

var log = logf.Log.WithName("controller_basicauth_secret")

const Kind = "BasicAuth"

// Add creates a new BasicAuth Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
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
	err = c.Watch(&source.Kind{Type: &v1alpha1.BasicAuth{}}, &handler.EnqueueRequestForObject{})
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
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling BasicAuth")
	ctx := context.Background()
	// Fetch the BasicAuth instance
	instance := &v1alpha1.BasicAuth{}
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

	// get configuration values from instance
	username := instance.Spec.Username

	// fallback in case username is empty
	if username == "" {
		username = "admin"
	}

	length := instance.Spec.Length
	encoding := instance.Spec.Encoding
	secretType := instance.Spec.Type
	regenerate := instance.Spec.ForceRecreate

	values := make(map[string][]byte)

	secretLength, isByteLength, err := crd.ParseByteLength(secret.SecretLength(), length)
	if err != nil {
		reqLogger.Error(err, "could not parse length for new random string")
		return reconcile.Result{RequeueAfter: time.Second * 30}, err
	}

	// attempt to fetch secret object described by this BasicAuth
	existing := &v1.Secret{}
	err = r.client.Get(ctx, request.NamespacedName, existing)
	if errors.IsNotFound(err) {
		// secret not found, create new one
		password, innerErr := secret.GenerateRandomString(secretLength, encoding, isByteLength)
		if err != nil {
			reqLogger.Error(innerErr, "could not generate random string")
			return reconcile.Result{RequeueAfter: time.Second * 30}, innerErr
		}

		var passwordHash []byte
		passwordHash, innerErr = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			reqLogger.Error(innerErr, "could not hash random string")

			return reconcile.Result{RequeueAfter: time.Second * 30}, innerErr
		}

		values[secret.SecretFieldBasicAuthIngress] = append([]byte(username+":"), passwordHash...)
		values[secret.SecretFieldBasicAuthUsername] = []byte(username)
		values[secret.SecretFieldBasicAuthPassword] = password

		desiredSecret := crd.NewSecret(instance, values, secretType)

		innerErr = r.client.Create(context.Background(), desiredSecret)
		if innerErr != nil {
			// secret has been created at some point during this reconcile, retry
			return reconcile.Result{Requeue: true}, innerErr
		}

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
	// secret already exists
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

	existingAuth := existing.Data[secret.SecretFieldBasicAuthIngress]

	if len(existingAuth) > 0 && !regenerate {
		// auth is set anf regeneration is not forced, do nothing
		return reconcile.Result{}, nil
	}

	targetSecret := existing.DeepCopy()

	var password []byte
	password, err = secret.GenerateRandomString(secretLength, encoding, isByteLength)
	if err != nil {
		reqLogger.Error(err, "could not hash random string")
		return reconcile.Result{RequeueAfter: time.Second * 30}, err
	}

	var passwordHash []byte
	passwordHash, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		reqLogger.Error(err, "could not hash random string")
		return reconcile.Result{RequeueAfter: time.Second * 30}, err
	}
	targetSecret.Data[secret.SecretFieldBasicAuthIngress] = append([]byte(username+":"), passwordHash...)
	targetSecret.Data[secret.SecretFieldBasicAuthUsername] = []byte(username)
	targetSecret.Data[secret.SecretFieldBasicAuthPassword] = password

	err = r.client.Update(ctx, targetSecret)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	err = r.GetSecretRefAndSetStatus(ctx, targetSecret, instance)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	return reconcile.Result{}, nil
}

// GetSecretRefAndSetStatus fetches the object reference for desiredSecret and writes it into the status of instance
func (r *ReconcileBasicAuth) GetSecretRefAndSetStatus(ctx context.Context, desiredSecret *v1.Secret, instance *v1alpha1.BasicAuth) error {
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
