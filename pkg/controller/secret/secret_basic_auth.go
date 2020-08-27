package secret

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Ingress basic auth secret field
const SecretFieldBasicAuthIngress = "auth"
const SecretFieldBasicAuthUsername = "username"
const SecretFieldBasicAuthPassword = "password"

type BasicAuthGenerator struct {
	log logr.Logger
}

func (bg BasicAuthGenerator) generateData(instance *corev1.Secret) (reconcile.Result, error) {
	existingAuth := string(instance.Data[SecretFieldBasicAuthIngress])

	regenerate := instance.Annotations[AnnotationSecretRegenerate] != ""

	if len(existingAuth) > 0 && !regenerate {
		return reconcile.Result{}, nil
	}
	delete(instance.Annotations, AnnotationSecretRegenerate)

	// if no username is given, fall back to "admin"
	username := instance.Annotations[AnnotationBasicAuthUsername]
	if username == "" {
		username = "admin"
	}

	length, err := secretLengthFromAnnotation(secretLength(), instance.Annotations)
	if err != nil {
		return reconcile.Result{}, err
	}

	password, err := generateRandomString(length)
	if err != nil {
		bg.log.Error(err, "could not generate new random string")
		return reconcile.Result{RequeueAfter: time.Second * 30}, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		bg.log.Error(err, "could not hash random string")
		return reconcile.Result{RequeueAfter: time.Second * 30}, err
	}

	instance.Data[SecretFieldBasicAuthIngress] = append([]byte(username+":"), passwordHash...)
	instance.Data[SecretFieldBasicAuthUsername] = []byte(username)
	instance.Data[SecretFieldBasicAuthPassword] = []byte(password)
	return reconcile.Result{}, nil
}
