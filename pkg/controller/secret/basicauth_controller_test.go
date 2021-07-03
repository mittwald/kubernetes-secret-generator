package secret_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/mittwald/kubernetes-secret-generator/pkg/apis/secretgenerator/v1alpha1"
	"github.com/mittwald/kubernetes-secret-generator/pkg/controller/crd/basicauth"
	"github.com/mittwald/kubernetes-secret-generator/pkg/controller/secret"
)

const testUsername = "testuser123"

// newBasicAuthTestCR reurns a BasicAuth custom resource. If name is set to "", a uuid will be generated
func newBasicAuthTestCR(authSpec v1alpha1.BasicAuthSpec, name string) *v1alpha1.BasicAuth {
	if name == "" {
		name = uuid.New().String()
	}
	cr := &v1alpha1.BasicAuth{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiVersion,
			Kind:       basicauth.Kind,
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

// verifyBasicAuthSecretFromCR verifies that the given Secret was correctly created from the given cr
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

	auth := out.Data[secret.FieldBasicAuthIngress]
	password := out.Data[secret.FieldBasicAuthPassword]
	length, _, err := secret.ParseByteLength(secret.DefaultLength(), in.Spec.Length)
	if err != nil {
		t.Error("Failed to determine secret length")
	}

	// check if password has been saved in clear text
	// and has correct length (if the secret has actually been generated)
	if len(password) == 0 || len(password) != length {
		t.Errorf("generated field has wrong length of %d", len(password))
	}

	// check if auth field has been generated (with separator)
	if len(auth) == 0 || !strings.Contains(string(auth), ":") {
		t.Errorf("auth field has wrong or no values %s", string(auth))
	}

	// check if custom data entries were set correctly
	for _, key := range in.Spec.Data {
		if _, ok := out.Data[key]; !ok {
			t.Errorf("missing data entry %s", key)
		}
	}
}

func TestControllerGenerateBasicAuthWithoutUsername(t *testing.T) {
	testSpec := v1alpha1.BasicAuthSpec{
		Encoding: "base64",
		Length:   "40",
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

	require.Equal(t, "admin", string(out.Data[secret.FieldBasicAuthUsername]))
	// check correct deletion of generated secret
	require.NoError(t, mgr.GetClient().Delete(context.TODO(), in))
}

func TestControllerGenerateBasicAuthWithUsername(t *testing.T) {
	testSpec := v1alpha1.BasicAuthSpec{
		Encoding: "base64",
		Length:   "40",
		Username: testUsername,
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

	require.Equal(t, testUsername, string(out.Data[secret.FieldBasicAuthUsername]))
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
		Username:        testUsername,
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
	oldPassword := string(out.Data[secret.FieldBasicAuthPassword])
	oldAuth := string(out.Data[secret.FieldBasicAuthIngress])

	verifyBasicAuthSecretFromCR(t, in, out)

	// modify cr to trigger update
	in.Spec.Username = "AnotherTestUser"
	doReconcileBasicAuthController(t, in, false)

	outNew := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, outNew))

	newPassword := string(outNew.Data[secret.FieldBasicAuthPassword])
	newAuth := string(outNew.Data[secret.FieldBasicAuthIngress])

	// ensure values before and after update are the same
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
		Username:        testUsername,
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
	oldPassword := string(out.Data[secret.FieldBasicAuthPassword])
	oldAuth := string(out.Data[secret.FieldBasicAuthIngress])

	verifyBasicAuthSecretFromCR(t, in, out)

	// modify cr to trigger update
	in.Spec.Username = "AnotherTestUser"
	doReconcileBasicAuthController(t, in, false)

	outNew := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, outNew))

	newPassword := string(outNew.Data[secret.FieldBasicAuthPassword])
	newAuth := string(outNew.Data[secret.FieldBasicAuthIngress])

	// ensure secret has been updated
	if oldPassword == newPassword {
		t.Errorf("secret has not been updated")
	}

	if oldAuth == newAuth {
		t.Errorf("secret has not been updated")
	}
}

func TestControllerDoNotTouchOtherSecrets(t *testing.T) {
	testSecret := &corev1.Secret{
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

	// create secret not owned by cr
	require.NoError(t, mgr.GetClient().Create(context.TODO(), testSecret))

	testSpec := v1alpha1.BasicAuthSpec{
		Encoding:        "base64",
		Length:          "40",
		Username:        "Hans",
		Data:            map[string]string{},
		ForceRegenerate: false,
	}

	in := newBasicAuthTestCR(testSpec, testSecretName)
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcileBasicAuthController(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      testSecret.Name,
		Namespace: testSecret.Namespace}, out))

	// ensure secret has not been modified
	if !reflect.DeepEqual(testSecret, out) {
		t.Errorf("testSecret not owned by BasicAuth cr has been reconciled")
	}
	require.NoError(t, mgr.GetClient().Delete(context.TODO(), testSecret))
}
