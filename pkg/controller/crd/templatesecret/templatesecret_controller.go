package templatesecret

import (
	"bytes"
	"context"
	"text/template"
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

var log = logf.Log.WithName("controller_template_secret")
var reqLogger logr.Logger

const Kind = "TemplateSecret"

// Add creates a new TemplateSecret Secret Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, NewReconciler(mgr))
}

// NewReconciler returns a new reconcile.Reconciler
func NewReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileTemplateSecret{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

type ReconcileTemplateSecret struct {
	// This Client, initialized using mgr.Client() above, is a split Client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("templatesecret-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}
	// Watch for changes to primary resource string
	err = c.Watch(&source.Kind{Type: &v1alpha1.TemplateSecret{}}, &handler.EnqueueRequestForObject{}, crd.IgnoreStatusUpdatePredicate())
	if err != nil {
		return err
	}

	return nil
}

// Reconcile reads that state of the cluster for a TemplateSecret object and makes changes based on the state read
// and what is in the TemplateSecret.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileTemplateSecret) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger = log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling TemplateSecret")
	ctx := context.Background()
	// fetch the TemplateSecret instance
	instance := &v1alpha1.TemplateSecret{}
	err := r.client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		// if instance is not found don't requeue and don't return error, else requeue and return error
		return crd.CheckError(err)
	}

	existing := &v1.Secret{}
	err = r.client.Get(ctx, request.NamespacedName, existing)
	// secret not found, create new one
	if errors.IsNotFound(err) {
		return r.createNewSecret(ctx, instance)
	}
	// check for other errors
	if err != nil {
		return reconcile.Result{}, err
	}

	// no errors, so secret exists, attempt to update
	return r.updateSecret(ctx, instance, existing)
}

// updateSecret attempts to update an existing Secret object with new values. Secret will only be updated,
// if it is owned by a TemplateSecret cr.
func (r *ReconcileTemplateSecret) updateSecret(ctx context.Context, instance *v1alpha1.TemplateSecret, existing *v1.Secret) (reconcile.Result, error) {
	// check if secret was created by a cr of the TemplateSecret kind
	existingOwnerRefs := existing.OwnerReferences

	if correct := crd.IsOwnedByCorrectCR(reqLogger, existingOwnerRefs, Kind); !correct {
		return reconcile.Result{}, nil
	}

	regenerate := instance.Spec.ForceRegenerate
	data := instance.Spec.Data

	targetSecret := existing.DeepCopy()

	// update data values from spec
	crd.UpdateData(data, targetSecret, regenerate)

	values, err := processTemplates(instance)
	if err != nil {
		return reconcile.Result{RequeueAfter: time.Second * 30}, err
	}
	targetSecret.Data = values

	c := crd.Client{Client: r.client}

	return c.ClientUpdateSecret(ctx, targetSecret, instance, r.scheme)
}

// createNewSecret creates a new template secret from the provided values. The Secret's owner will be set
// as the TemplateSecret resource that is being reconciled and a reference to the Secret will be stored in
// the cr's status
func (r *ReconcileTemplateSecret) createNewSecret(ctx context.Context, instance *v1alpha1.TemplateSecret) (reconcile.Result, error) {
	values, err := processTemplates(instance)
	if err != nil {
		return reconcile.Result{RequeueAfter: time.Second * 30}, err
	}

	c := crd.Client{Client: r.client}

	return c.ClientCreateSecret(ctx, values, instance, r.scheme)
}

func processTemplates(instance *v1alpha1.TemplateSecret) (map[string][]byte, error) {
	fields := instance.Spec.Fields
	data := instance.Spec.Data

	genValues := make(map[string]string)
	// generate values from fields property
	err := setValuesForFields(fields, true, genValues)
	if err != nil {
		return nil, err
	}

	values := make(map[string][]byte)

	for key := range data {
		tmpl, err := template.New(key).Parse(data[key])
		if err != nil {
			return nil, err
		}

		buf := bytes.Buffer{}
		err = tmpl.Execute(&buf, genValues)
		if err != nil {
			return nil, err
		}

		values[key] = buf.Bytes()
	}

	return values, nil
}

// setValuesForFields iterates over the given list of Fields and generates new random strings if the corresponding entry is empty or
// regeneration is forced
func setValuesForFields(fields []v1alpha1.TemplateField, regenerate bool, values map[string]string) error {
	// generate only empty fields if regenerate wasn't set to true
	for _, field := range fields {
		if string(values[field.FieldName]) == "" || regenerate {
			fieldLength, isByteLength, err := secret.ParseByteLength(secret.DefaultLength(), field.Length)
			if err != nil {
				reqLogger.Error(err, "could not parse length from map for new random string")
				return err
			}
			encoding := field.Encoding
			if encoding == "" {
				encoding = "ascii"
			}
			randomString, randErr := secret.GenerateRandomString(fieldLength, encoding, isByteLength)
			if randErr != nil {
				reqLogger.Error(randErr, "could not generate new random string")
				return randErr
			}
			values[field.FieldName] = string(randomString)
		}
	}

	return nil
}
