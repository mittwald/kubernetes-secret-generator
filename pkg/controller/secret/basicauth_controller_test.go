package secret_test

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/mittwald/kubernetes-secret-generator/pkg/apis/types/v1alpha1"
	"github.com/mittwald/kubernetes-secret-generator/pkg/controller/crd"
	"github.com/mittwald/kubernetes-secret-generator/pkg/controller/crd/basicauth"
	"github.com/mittwald/kubernetes-secret-generator/pkg/controller/secret"
)

const testUsername = "testuser123"

func newBasicAuthTestCR(authSpec v1alpha1.BasicAuthSpec, name string) *v1alpha1.BasicAuth {
	if name == "" {
		name = uuid.New().String()
	}
	cr := &v1alpha1.BasicAuth{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "mittwald.systems/v1alpha1",
			Kind:       "BasicAuth",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			Labels: map[string]string{
				labelSecretGeneratorTest: "yes",
			},
		},
		Spec: authSpec,
	}

	return cr
}

func verifyBasicAuthSecretFromCR(t *testing.T, in *v1alpha1.BasicAuth, out *corev1.Secret) {
	// Check for correct ownership
	for index := range out.OwnerReferences {
		if out.OwnerReferences[index].Kind == basicauth.Kind {
			break
		}
		if index == len(out.OwnerReferences)-1 {
			t.Errorf("generated secret not owned by kind %s", basicauth.Kind)
		}
	}

	// check if cr status was updated properly with secret reference
	if in.Status.Secret != nil && in.Status.Secret.Name != out.Name {
		t.Error("generated secret not referenced in CR status")
	}

	auth := out.Data[secret.SecretFieldBasicAuthIngress]
	password := out.Data[secret.SecretFieldBasicAuthPassword]
	desiredLength, _, err := crd.ParseByteLength(secret.SecretLength(), in.Spec.Length)
	if err != nil {
		t.Error("Failed to determine secret length")
	}

	// check if password has been saved in clear text
	// and has correct length (if the secret has actually been generated)
	if len(password) == 0 || len(password) != desiredLength {
		t.Errorf("generated field has wrong length of %d", len(password))
	}

	// check if auth field has been generated (with separator)
	if len(auth) == 0 || !strings.Contains(string(auth), ":") {
		t.Errorf("auth field has wrong or no values %s", string(auth))
	}
}

func TestControllerGenerateBasicAuthWithoutUsername(t *testing.T) {
	testSpec := v1alpha1.BasicAuthSpec{
		Encoding: "base64",
		Length:   "40",
		Type:     string(corev1.SecretTypeOpaque),
		Data:     map[string]string{},
	}

	in := newBasicAuthTestCR(testSpec, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcileBasicAuthController(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	// reacquire object for updated status
	require.NoError(t, mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, in))

	verifyBasicAuthSecretFromCR(t, in, out)

	require.Equal(t, "admin", string(out.Data[secret.SecretFieldBasicAuthUsername]))
	// check correct deletion of generated secret
	require.NoError(t, mgr.GetClient().Delete(context.TODO(), in))
	// give deletion time to be processed
	time.Sleep(1 * time.Second)

	out = &corev1.Secret{}
	err := mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, out)
	require.True(t, errors.IsNotFound(err), "Secret was not deleted upon cr deletion")
}

func TestControllerGenerateBasicAuthWithUsername(t *testing.T) {
	testSpec := v1alpha1.BasicAuthSpec{
		Encoding: "base64",
		Length:   "40",
		Username: testUsername,
		Type:     string(corev1.SecretTypeOpaque),
		Data:     map[string]string{},
	}

	in := newBasicAuthTestCR(testSpec, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcileBasicAuthController(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	// reacquire object for updated status
	require.NoError(t, mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, in))

	verifyBasicAuthSecretFromCR(t, in, out)

	require.Equal(t, testUsername, string(out.Data[secret.SecretFieldBasicAuthUsername]))
	// check correct deletion of generated secret

}

func doReconcileBasicAuthController(t *testing.T, basicAuth *v1alpha1.BasicAuth, isErr bool) {
	rec := basicauth.NewReconciler(mgr)
	req := reconcile.Request{NamespacedName: types.NamespacedName{Name: basicAuth.Name, Namespace: basicAuth.Namespace}}

	res, err := rec.Reconcile(req)

	if isErr {
		require.Error(t, err)
	} else {
		require.NoError(t, err)
	}
	require.False(t, res.Requeue)
}

func TestControllerGenerateBasicAuthNoRegenerate(t *testing.T) {
	testSpec := v1alpha1.BasicAuthSpec{
		Encoding:        "base64",
		Length:          "40",
		Username:        "Hans",
		Type:            string(corev1.SecretTypeOpaque),
		Data:            map[string]string{},
		ForceRegenerate: false,
	}

	in := newBasicAuthTestCR(testSpec, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcileBasicAuthController(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, out))
	oldPassword := string(out.Data[secret.SecretFieldBasicAuthPassword])
	oldAuth := string(out.Data[secret.SecretFieldBasicAuthIngress])

	verifyBasicAuthSecretFromCR(t, in, out)

	in.Spec.Username = "Hugo"
	in.Spec.Length = "35"
	doReconcileBasicAuthController(t, in, false)

	outNew := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, outNew))

	newPassword := string(outNew.Data[secret.SecretFieldBasicAuthPassword])
	newAuth := string(outNew.Data[secret.SecretFieldBasicAuthIngress])

	if oldPassword != newPassword {
		t.Errorf("secret has been updated")
	}

	if oldAuth != newAuth {
		t.Errorf("secret has been updated")
	}
}

func TestControllerGenerateBasicAuthRegenerate(t *testing.T) {
	testSpec := v1alpha1.BasicAuthSpec{
		Encoding:        "base64",
		Length:          "40",
		Username:        "Hans",
		Type:            string(corev1.SecretTypeOpaque),
		Data:            map[string]string{},
		ForceRegenerate: true,
	}

	in := newBasicAuthTestCR(testSpec, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcileBasicAuthController(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, out))
	oldPassword := string(out.Data[secret.SecretFieldBasicAuthPassword])
	oldAuth := string(out.Data[secret.SecretFieldBasicAuthIngress])

	verifyBasicAuthSecretFromCR(t, in, out)

	in.Spec.Username = "Hugo"
	in.Spec.Length = "35"
	doReconcileBasicAuthController(t, in, false)

	outNew := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, outNew))

	newPassword := string(outNew.Data[secret.SecretFieldBasicAuthPassword])
	newAuth := string(outNew.Data[secret.SecretFieldBasicAuthIngress])

	if oldPassword == newPassword {
		t.Errorf("secret has not been updated")
	}

	if oldAuth == newAuth {
		t.Errorf("secret has not been updated")
	}
}

func TestControllerDoNotTouchOtherSecrets(t *testing.T) {
	secret := &corev1.Secret{
		Type: corev1.SecretTypeOpaque,
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecretName,
			Namespace: "default",
			Labels: map[string]string{
				labelSecretGeneratorTest: "yes",
			},
		},
		Data: map[string][]byte{
			"username": []byte("test"),
			"password": []byte("test2"),
			"auth":     []byte("test:test2"),
		},
	}

	require.NoError(t, mgr.GetClient().Create(context.TODO(), secret))

	testSpec := v1alpha1.BasicAuthSpec{
		Encoding:        "base64",
		Length:          "40",
		Username:        "Hans",
		Type:            string(corev1.SecretTypeOpaque),
		Data:            map[string]string{},
		ForceRegenerate: false,
	}

	in := newBasicAuthTestCR(testSpec, testSecretName)
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcileBasicAuthController(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      secret.Name,
		Namespace: secret.Namespace}, out))

	if !reflect.DeepEqual(secret, out) {
		t.Errorf("secret not owned by BasicAuth cr has been reconciled")
	}
	require.NoError(t, mgr.GetClient().Delete(context.TODO(), secret))
}
