package string

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

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

var log = logf.Log.WithName("controller_secret")

// Add creates a new Secret Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileString{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

type MyCRStatus struct {
	// +kubebuilder:validation:Enum=Success,Failure
	Status     string      `json:"status,omitempty"`
	LastUpdate metav1.Time `json:"lastUpdate,omitempty"`
	Reason     string      `json:"reason,omitempty"`
}

type ReconcileString struct {
	// This Client, initialized using mgr.Client() above, is a split Client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("password-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Secret
	err = c.Watch(&source.Kind{Type: &v1alpha1.String{}}, &handler.EnqueueRequestForObject{})
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
func (r *ReconcileString) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling String")
	// Fetch the Secret instance
	instance := &v1alpha1.String{}
	err := r.client.Get(context.Background(), request.NamespacedName, instance)
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
	// fmt.Println("<<<<<")
	// fmt.Println(instance.Spec.Initialized)
	// if ok := r.IsInitialized(instance); !ok {
	// 	fmt.Println("Jojjjjjj")
	// 	fmt.Println(instance.Spec)
	// 	fmt.Println(">>>>>")
	// 	err := r.client.Update(context.TODO(), instance)
	// 	if err != nil {
	//
	// 		log.Error(err, "unable to update instance", "instance", instance)
	//
	// 		return reconcile.Result{}, err
	//
	// 	}
	//
	// 	return reconcile.Result{}, nil
	//
	// }

	fieldNames := instance.Spec.FieldNames
	length := instance.Spec.Length
	encoding := instance.Spec.Encoding
	data := instance.Spec.Data

	values := make(map[string][]byte)

	for key := range data {
		values[key] = []byte(data[key])
	}

	fmt.Println(instance.Spec)
	secretLength, isByteLength, err := crd.ParseByteLength(secret.SecretLength(), length)
	if err != nil {

	}
	for _, field := range fieldNames {
		randomString, randErr := secret.GenerateRandomString(secretLength, encoding, isByteLength)
		if randErr != nil {
			reqLogger.Error(err, "could not generate new random string")
			return reconcile.Result{RequeueAfter: time.Second * 30}, err
		}
		values[field] = randomString
	}
	desiredSecret := crd.NewSecret(instance, values)

	err = r.client.Create(context.Background(), desiredSecret)
	if err != nil {
		if errors.IsAlreadyExists(err) {

			existing := &v1.Secret{}
			err = r.client.Get(context.Background(), types.NamespacedName{Name: desiredSecret.Name, Namespace: desiredSecret.Namespace}, existing)
			if err != nil {
				return reconcile.Result{Requeue: true}, err
			}
			for key := range existing.Data {
				if _, ok := desiredSecret.Data[key]; !ok {
					desiredSecret.Data[key] = existing.Data[key]
				}
			}

			err = r.client.Update(context.Background(), desiredSecret)
			if err != nil {
				return reconcile.Result{Requeue: true}, err
			}
		} else {
			return reconcile.Result{Requeue: true}, err
		}
	}
	return reconcile.Result{}, nil
}

// func (r *ReconcileString) IsInitialized(obj metav1.Object) bool {
// 	mycrd, ok := obj.(*v1alpha1.String)
// 	if !ok {
// 		fmt.Println("mr stark")
// 		return false
// 	}
// 	if mycrd.Spec.Initialized {
// 		fmt.Println("i don't feel so good")
// 		return true
// 	}
// 	mycrd.Spec.Initialized = true
// 	return false
//
// }
