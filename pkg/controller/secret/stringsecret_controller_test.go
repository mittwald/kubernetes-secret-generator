package secret_test

import (
	"context"
	"reflect"
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
	"github.com/mittwald/kubernetes-secret-generator/pkg/controller/crd/stringsecret"
	"github.com/mittwald/kubernetes-secret-generator/pkg/controller/secret"
)

const testSecretName = "testsec123"

func newStringSecretTestCR(stringSpec v1alpha1.StringSecretSpec, name string) *v1alpha1.StringSecret {
	if name == "" {
		name = uuid.New().String()
	}
	cr := &v1alpha1.StringSecret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "mittwald.systems/v1alpha1",
			Kind:       "StringSecret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			Labels: map[string]string{
				labelSecretGeneratorTest: "yes",
			},
		},
		Spec: stringSpec,
	}

	return cr
}

func verifyStringSecretFromCR(t *testing.T, in *v1alpha1.StringSecret, out *corev1.Secret) {
	// Check for correct ownership
	for index := range out.OwnerReferences {
		if out.OwnerReferences[index].Kind == stringsecret.Kind {
			break
		}
		if index == len(out.OwnerReferences)-1 {
			t.Errorf("generated secret not owned by kind %s", stringsecret.Kind)
		}
	}

	// check if cr status was updated properly with secret reference
	if in.Status.Secret != nil && in.Status.Secret.Name != out.Name {
		t.Error("generated secret not referenced in CR status")
	}

	for _, field := range in.Spec.Fields {
		fieldName := field.FieldName

		val, ok := out.Data[fieldName]
		if !ok {
			t.Error("secret value has not been generated")
		}

		secLength, _, err := crd.ParseByteLength(secret.SecretLength(), field.Length)
		if err != nil {
			t.Error("Failed to determine secret length")
		}

		if len(val) != secLength {
			t.Errorf("generated field has wrong length of %d", len(val))
		}

		t.Logf("generated secret value: %s", val)
	}

	for _, key := range in.Spec.FieldNames {
		val, ok := out.Data[key]
		if !ok {
			t.Error("secret value has not been generated")
		}

		secLength, _, err := crd.ParseByteLength(secret.SecretLength(), in.Spec.Length)
		if err != nil {
			t.Error("Failed to determine secret length")
		}

		if len(val) != secLength {
			t.Errorf("generated field has wrong length of %d", len(val))
		}

		t.Logf("generated secret value: %s", val)
	}
}

func doReconcileStringSecretController(t *testing.T, stringSecret *v1alpha1.StringSecret, isErr bool) {
	rec := stringsecret.NewReconciler(mgr)
	req := reconcile.Request{NamespacedName: types.NamespacedName{Name: stringSecret.Name, Namespace: stringSecret.Namespace}}

	res, err := rec.Reconcile(req)

	if isErr {
		require.Error(t, err)
	} else {
		require.NoError(t, err)
	}
	require.False(t, res.Requeue)
}

func TestControllerGenerateSecretSingleFieldLegacy(t *testing.T) {
	testSpec := v1alpha1.StringSecretSpec{
		Encoding:   "base64",
		Length:     "40",
		Type:       string(corev1.SecretTypeOpaque),
		Data:       map[string]string{},
		FieldNames: []string{"test"},
	}
	in := newStringSecretTestCR(testSpec, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcileStringSecretController(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifyStringSecretFromCR(t, in, out)

	require.NoError(t, mgr.GetClient().Delete(context.TODO(), in))
	// give deletion time to be processed
	time.Sleep(1 * time.Second)
	out = &corev1.Secret{}
	err := mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, out)
	require.True(t, errors.IsNotFound(err), "Secret was not deleted upon cr deletion")
}

func TestControllerGenerateSecretSingleField(t *testing.T) {
	testSpec := v1alpha1.StringSecretSpec{
		Type: string(corev1.SecretTypeOpaque),
		Data: map[string]string{},
		Fields: []v1alpha1.Field{v1alpha1.Field{
			FieldName: "test",
			Encoding:  "base64",
			Length:    "40",
		}},
	}
	in := newStringSecretTestCR(testSpec, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcileStringSecretController(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifyStringSecretFromCR(t, in, out)

	require.NoError(t, mgr.GetClient().Delete(context.TODO(), in))
	// give deletion time to be processed
	time.Sleep(1 * time.Second)
	out = &corev1.Secret{}
	err := mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, out)
	require.True(t, errors.IsNotFound(err), "Secret was not deleted upon cr deletion")
}

func TestControllerGenerateSecretSingleFieldMixed(t *testing.T) {
	testSpec := v1alpha1.StringSecretSpec{
		Encoding:   "base64",
		Length:     "40",
		Type:       string(corev1.SecretTypeOpaque),
		Data:       map[string]string{},
		FieldNames: []string{"test"},
		Fields: []v1alpha1.Field{v1alpha1.Field{
			FieldName: "test2",
			Encoding:  "base64",
			Length:    "40",
		}},
	}
	in := newStringSecretTestCR(testSpec, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcileStringSecretController(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifyStringSecretFromCR(t, in, out)

	require.NoError(t, mgr.GetClient().Delete(context.TODO(), in))
	// give deletion time to be processed
	time.Sleep(1 * time.Second)
	out = &corev1.Secret{}
	err := mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, out)
	require.True(t, errors.IsNotFound(err), "Secret was not deleted upon cr deletion")
}

func TestControllerGenerateSecretMultipleFields(t *testing.T) {
	testSpec := v1alpha1.StringSecretSpec{
		Type: string(corev1.SecretTypeOpaque),
		Data: map[string]string{},
		Fields: []v1alpha1.Field{{
			FieldName: "test",
			Encoding:  "base64",
			Length:    "40",
		},
			{
				FieldName: "test2",
				Encoding:  "base32",
				Length:    "35",
			},
		},
	}
	in := newStringSecretTestCR(testSpec, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcileStringSecretController(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifyStringSecretFromCR(t, in, out)
}

func TestRegenerateSecretsSingleField(t *testing.T) {
	testSpec := v1alpha1.StringSecretSpec{
		Type: string(corev1.SecretTypeOpaque),
		Data: map[string]string{},
		Fields: []v1alpha1.Field{v1alpha1.Field{
			FieldName: "test",
			Encoding:  "base64",
			Length:    "40",
		}},
		ForceRegenerate: true,
	}
	in := newStringSecretTestCR(testSpec, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcileStringSecretController(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))
	verifyStringSecretFromCR(t, in, out)

	oldSecretValue := string(out.Data["test"])

	in.Spec.Length = "35"

	doReconcileStringSecretController(t, in, false)

	outNew := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, outNew))

	newSecretValue := string(outNew.Data["test"])

	if oldSecretValue == newSecretValue {
		t.Errorf("secret has not been updated")
	}

}

func TestRegenerateSecretsmultipleFields(t *testing.T) {
	testSpec := v1alpha1.StringSecretSpec{
		Type: string(corev1.SecretTypeOpaque),
		Data: map[string]string{},
		Fields: []v1alpha1.Field{{
			FieldName: "test",
			Encoding:  "base64",
			Length:    "40",
		},
			{
				FieldName: "test2",
				Encoding:  "base32",
				Length:    "35",
			},
		},
		ForceRegenerate: true,
	}
	in := newStringSecretTestCR(testSpec, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcileStringSecretController(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))
	verifyStringSecretFromCR(t, in, out)

	oldSecretValue := string(out.Data["test"])
	oldSecretValue2 := string(out.Data["test2"])

	in.Spec.Length = "35"

	doReconcileStringSecretController(t, in, false)

	outNew := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, outNew))

	newSecretValue := string(outNew.Data["test"])
	newSecretValue2 := string(outNew.Data["test2"])

	if oldSecretValue == newSecretValue {
		t.Errorf("secret has not been updated")
	}

	if oldSecretValue2 == newSecretValue2 {
		t.Errorf("secret has not been updated")
	}
}

func TestDoNotTouchOtherSecrets(t *testing.T) {
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
		},
	}

	require.NoError(t, mgr.GetClient().Create(context.TODO(), secret))

	testSpec := v1alpha1.StringSecretSpec{
		Encoding:        "base64",
		Length:          "40",
		Type:            string(corev1.SecretTypeOpaque),
		Data:            map[string]string{},
		FieldNames:      []string{"test"},
		ForceRegenerate: false,
	}
	in := newStringSecretTestCR(testSpec, testSecretName)
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcileStringSecretController(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      secret.Name,
		Namespace: secret.Namespace}, out))

	if !reflect.DeepEqual(secret, out) {
		t.Errorf("secret not owned by BasicAuth cr has been reconciled")
	}
	require.NoError(t, mgr.GetClient().Delete(context.TODO(), secret))
}
