package secret

import (
	"bytes"
	"context"
	"github.com/imdario/mergo"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"strings"
	"testing"
	"time"
)

func newPasswordTestSecret(fields string, extraAnnotations map[string]string, initValues string) *corev1.Secret {
	annotations := map[string]string{
		AnnotationSecretGenerate: fields,
	}
	if extraAnnotations != nil {
		if err := mergo.Merge(&annotations, extraAnnotations, mergo.WithOverride); err != nil {
			panic(err)
		}
	}

	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getSecretName(),
			Namespace: "default",
			Labels: map[string]string{
				labelSecretGeneratorTest: "yes",
			},
			Annotations: annotations,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{},
	}

	keys := strings.Split(fields, ",")
	for i, init := range strings.Split(initValues, ",") {
		s.Data[keys[i]] = []byte(init)
	}

	return s
}

// verify basic fields of the secret are present
func verifyPasswordSecret(t *testing.T, in, out *corev1.Secret, secure bool) {
	if out.Annotations[AnnotationSecretType] != string(SecretTypePassword) {
		t.Errorf("generated secret has wrong type %s on  %s annotation", out.Annotations[AnnotationSecretType], AnnotationSecretType)
	}

	if out.Annotations[AnnotationSecretSecure] != "yes" && secure {
		t.Errorf("generated secret has no %s annotation", AnnotationSecretSecure)
	} else if out.Annotations[AnnotationSecretSecure] == "yes" && !secure {
		t.Errorf("generated secret has %s annotation", AnnotationSecretSecure)
	}

	_, wasGenerated := in.Annotations[AnnotationSecretGeneratedAt]

	for _, key := range strings.Split(in.Annotations[AnnotationSecretGenerate], ",") {
		val, ok := out.Data[key]
		if !ok {
			t.Error("secret value has not been generated")
		}

		// check if secret has correct length (if the secret has actually been generated)
		if !wasGenerated && (len(val) == 0 || len(val) != secretLength()) {
			t.Errorf("generated field has wrong length of %d", len(val))
		}

		t.Logf("generated secret value: %s", val)
	}

	if _, ok := out.Annotations[AnnotationSecretGeneratedAt]; !ok {
		t.Errorf("secret has no %s annotation", AnnotationSecretGeneratedAt)
	}
}

// verify requested keys have been regenerated
func verifyPasswordRegen(t *testing.T, in, out *corev1.Secret) {
	if _, ok := out.Annotations[AnnotationSecretRegenerate]; ok {
		t.Errorf("%s annotation is still present", AnnotationSecretRegenerate)
	}

	if _, ok := in.Annotations[AnnotationSecretRegenerate]; !ok && !regenerateInsecure() { // test the tester
		t.Errorf("%s annotation is not present on input", AnnotationSecretRegenerate)
	}

	if _, ok := in.Annotations[AnnotationSecretGeneratedAt]; !ok { // test the tester
		t.Errorf("%s annotation is not present on input", AnnotationSecretGeneratedAt)
	}

	var regenKeys []string
	if in.Annotations[AnnotationSecretRegenerate] == "yes" ||
		regenerateInsecure() && in.Annotations[AnnotationSecretSecure] == "" {
		regenKeys = strings.Split(in.Annotations[AnnotationSecretGenerate], ",")
	} else if in.Annotations[AnnotationSecretRegenerate] != "" {
		regenKeys = strings.Split(in.Annotations[AnnotationSecretRegenerate], ",")
	}

	t.Logf("checking regenerated keys are regenerated and have correct length")
	t.Logf("keys expected to be regenerated: %d", len(regenKeys))
	if len(regenKeys) != 0 {
		for _, key := range regenKeys {
			val := out.Data[key]
			if len(val) == 0 || len(val) != secretLength() {
				// check length here again, verifyPasswordSecret skips this for secrets that already had the generatedAt Annotation
				t.Errorf("regenerated field has wrong length of %d", len(val))
			}

			if bytes.Equal(in.Data[key], val) {
				t.Errorf("key %s is equal for in(%s) and out (%s)", key, in.Data[key], out.Data[key])
				continue
			}
			t.Logf("key %s is NOT equal for in(%s) and out (%s)", key, in.Data[key], out.Data[key])
		}
	}

	t.Logf("checking generated keys are not regenerated")
	genKeys := strings.Split(in.Annotations[AnnotationSecretGenerate], ",")
	for _, key := range genKeys {
		if stringInSlice(key, regenKeys) {
			continue
		}
		if bytes.Equal(in.Data[key], out.Data[key]) {
			t.Logf("key %s is equal for in(%s) and out (%s)", key, in.Data[key], out.Data[key])
			continue
		}
		t.Errorf("key %s is NOT equal for in(%s) and out (%s)", key, in.Data[key], out.Data[key])
	}
}

func TestGenerateSecretSingleField(t *testing.T) {
	in := newPasswordTestSecret("testfield", nil, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifyPasswordSecret(t, in, out, true)
}

func TestGenerateSecretMultipleFields(t *testing.T) {
	in := newPasswordTestSecret("testfield,test1,test2,test3,abc,12345,6789", nil, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifyPasswordSecret(t, in, out, true)
}

func TestRegenerateSingleField(t *testing.T) {
	in := newPasswordTestSecret("testfield", map[string]string{
		AnnotationSecretRegenerate:  "testfield",
		AnnotationSecretGeneratedAt: time.Now().Format(time.RFC3339),
	}, "test")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifyPasswordSecret(t, in, out, true)
	verifyPasswordRegen(t, in, out)
}

func TestRegenerateAllSingleField(t *testing.T) {
	in := newPasswordTestSecret("testfield", map[string]string{
		AnnotationSecretRegenerate:  "yes",
		AnnotationSecretGeneratedAt: time.Now().Format(time.RFC3339),
	}, "test")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifyPasswordSecret(t, in, out, true)
	verifyPasswordRegen(t, in, out)
}

func TestRegenerateMultipleFieldsSecure(t *testing.T) {
	in := newPasswordTestSecret("testfield,test1,test2", map[string]string{
		AnnotationSecretRegenerate:  "testfield",
		AnnotationSecretGeneratedAt: time.Now().Format(time.RFC3339),
		AnnotationSecretSecure:      "yes",
	}, "test,abc,def")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifyPasswordSecret(t, in, out, true)
	verifyPasswordRegen(t, in, out)
}

func TestRegenerateMultipleFieldsNotSecure(t *testing.T) {
	in := newPasswordTestSecret("testfield,test1,test2", map[string]string{
		AnnotationSecretRegenerate:  "testfield",
		AnnotationSecretGeneratedAt: time.Now().Format(time.RFC3339),
	}, "test,abc,def")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifyPasswordSecret(t, in, out, false)
	verifyPasswordRegen(t, in, out)
}

func TestRegenerateAllMultipleFields(t *testing.T) {
	in := newPasswordTestSecret("testfield,test1,test2", map[string]string{
		AnnotationSecretRegenerate:  "yes",
		AnnotationSecretGeneratedAt: time.Now().Format(time.RFC3339),
	}, "test,abc,def")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifyPasswordSecret(t, in, out, true)
	verifyPasswordRegen(t, in, out)
}

func TestRegenerateInsecureSingleField(t *testing.T) {
	viper.Set("regenerate-insecure", true)
	in := newPasswordTestSecret("testfield", map[string]string{
		AnnotationSecretGeneratedAt: time.Now().Format(time.RFC3339),
	}, "test")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifyPasswordSecret(t, in, out, true)
	verifyPasswordRegen(t, in, out)
	viper.Set("regenerate-insecure", false)
}

func TestRegenerateInsecureEmpty(t *testing.T) {
	viper.Set("regenerate-insecure", true)
	in := newPasswordTestSecret("testfield", nil, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifyPasswordSecret(t, in, out, true)
	viper.Set("regenerate-insecure", false)
}

func TestRegenerateInsecureSingleFieldSecureBefore(t *testing.T) {
	viper.Set("regenerate-insecure", true)
	in := newPasswordTestSecret("testfield", map[string]string{
		AnnotationSecretGeneratedAt: time.Now().Format(time.RFC3339),
		AnnotationSecretSecure:      "yes",
	}, "test")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifyPasswordSecret(t, in, out, true)
	verifyPasswordRegen(t, in, out)
	viper.Set("regenerate-insecure", false)
}

func TestRegenerateInsecureMultipleField(t *testing.T) {
	viper.Set("regenerate-insecure", true)
	in := newPasswordTestSecret("testfield,test1,test2,test3", map[string]string{
		AnnotationSecretGeneratedAt: time.Now().Format(time.RFC3339),
	}, "abc,def,ghi,jkl")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifyPasswordSecret(t, in, out, true)
	verifyPasswordRegen(t, in, out)
	viper.Set("regenerate-insecure", false)
}

func TestRegenerateInsecureMultipleFieldSecureBefore(t *testing.T) {
	viper.Set("regenerate-insecure", true)
	in := newPasswordTestSecret("testfield,test1,test2,test3", map[string]string{
		AnnotationSecretGeneratedAt: time.Now().Format(time.RFC3339),
		AnnotationSecretSecure:      "yes",
	}, "abc,def,ghi,jkl")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifyPasswordSecret(t, in, out, true)
	verifyPasswordRegen(t, in, out)
	viper.Set("regenerate-insecure", false)
}

func TestUniqueness(t *testing.T) {
	in := newPasswordTestSecret("testfield,abc,test,abc,oops,oops", nil, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, true)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))
}

func TestDefaultToPasswordGeneration(t *testing.T) {
	in := newPasswordTestSecret("testfield", nil, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))
	verifyPasswordSecret(t, in, out, true)
}

func TestPasswordTypeAnnotationDetected(t *testing.T) {
	in := newPasswordTestSecret("testfield", map[string]string{
		AnnotationSecretType: string(SecretTypePassword),
	}, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))
	verifyPasswordSecret(t, in, out, true)
}

func TestGeneratedSecretsHaveCorrectLength(t *testing.T) {
	pwd, err := generatePassword(20)

	t.Log("generated", pwd)

	if err != nil {
		t.Error(err)
	}

	if len(pwd) != 20 {
		t.Error("password length", "expected", 20, "got", len(pwd))
	}
}

func TestGeneratedSecretsAreRandom(t *testing.T) {
	one, errOne := generatePassword(32)
	two, errTwo := generatePassword(32)

	if errOne != nil {
		t.Error(errOne)
	}
	if errTwo != nil {
		t.Error(errTwo)
	}

	if one == two {
		t.Error("password equality", "got", one)
	}
}

func BenchmarkGenerateSecret(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := generatePassword(32)
		if err != nil {
			b.Error(err)
		}
	}
}
