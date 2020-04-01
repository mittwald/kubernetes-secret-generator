package secret

import (
	"bytes"
	"context"
	"github.com/google/uuid"
	"github.com/imdario/mergo"
	"github.com/mittwald/kubernetes-secret-generator/pkg/apis"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/flowcontrol"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
	"testing"
	"time"
)

var mgr manager.Manager

const labelSecretGeneratorTest = "kubernetes-secret-generator-test"

func getSecretName() string {
	return uuid.New().String()
}

func TestMain(m *testing.M) {
	cfgPath := os.Getenv("KUBECONFIG")
	cfg, err := clientcmd.BuildConfigFromFlags("", cfgPath)

	if err != nil {
		panic(err)
	}

	restMapper := func(cfg *rest.Config) (meta.RESTMapper, error) {
		return apiutil.NewDynamicRESTMapper(cfg)
	}

	mgrOpts := manager.Options{
		MapperProvider: restMapper,
		NewClient: func(_ cache.Cache, config *rest.Config, options client.Options) (client.Client, error) {
			config.RateLimiter = flowcontrol.NewFakeAlwaysRateLimiter()
			return client.New(config, options)
		},
	}

	mgr, err = manager.New(cfg, mgrOpts)
	if err != nil {
		panic(err)
	}

	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		panic(err)
	}

	setupViper()
	reset()

	code := m.Run()

	os.Exit(code)
}

func setupViper() {
	viper.Set("secret-length", 40)
	viper.Set("regenerate-insecure", false)
}

func reset() {
	list := &corev1.SecretList{}
	err := mgr.GetClient().List(context.TODO(),
		list,
		client.MatchingLabels(map[string]string{
			labelSecretGeneratorTest: "yes",
		}),
	)
	if err != nil {
		panic(err)
	}

	for _, s := range list.Items {
		err := mgr.GetClient().Delete(context.TODO(), &s)
		if err != nil {
			panic(err)
		}
	}
}

func doReconcile(t *testing.T, secret *corev1.Secret, isErr bool) {
	rec := ReconcileSecret{mgr.GetClient(), mgr.GetScheme()}
	req := reconcile.Request{NamespacedName: types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}}

	res, err := rec.Reconcile(req)

	if isErr {
		require.Error(t, err)
	} else {
		require.NoError(t, err)
	}
	require.False(t, res.Requeue)
}

func newTestSecret(fields string, extraAnnotations map[string]string, initValues string) *corev1.Secret {
	annotations := map[string]string{
		SecretGenerateAnnotation: fields,
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
func verifySecret(t *testing.T, in, out *corev1.Secret, secure bool) {
	if out.Annotations[SecretSecureAnnotation] != "yes" && secure {
		t.Errorf("generated secret has no %s annotation", SecretSecureAnnotation)
	} else if out.Annotations[SecretSecureAnnotation] == "yes" && !secure {
		t.Errorf("generated secret has %s annotation", SecretSecureAnnotation)
	}

	_, wasGenerated := in.Annotations[SecretGeneratedAtAnnotation]

	for _, key := range strings.Split(in.Annotations[SecretGenerateAnnotation], ",") {
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

	if _, ok := out.Annotations[SecretGeneratedAtAnnotation]; !ok {
		t.Errorf("secret has no %s annotation", SecretGeneratedAtAnnotation)
	}
}

// verify requested keys have been regenerated
func verifyRegen(t *testing.T, in, out *corev1.Secret) {
	if _, ok := out.Annotations[SecretRegenerateAnnotation]; ok {
		t.Errorf("%s annotation is still present", SecretRegenerateAnnotation)
	}

	if _, ok := in.Annotations[SecretRegenerateAnnotation]; !ok && !regenerateInsecure() { // test the tester
		t.Errorf("%s annotation is not present on input", SecretRegenerateAnnotation)
	}

	if _, ok := in.Annotations[SecretGeneratedAtAnnotation]; !ok { // test the tester
		t.Errorf("%s annotation is not present on input", SecretGeneratedAtAnnotation)
	}

	var regenKeys []string
	if in.Annotations[SecretRegenerateAnnotation] == "yes" ||
		regenerateInsecure() && in.Annotations[SecretSecureAnnotation] == "" {
		regenKeys = strings.Split(in.Annotations[SecretGenerateAnnotation], ",")
	} else if in.Annotations[SecretRegenerateAnnotation] != "" {
		regenKeys = strings.Split(in.Annotations[SecretRegenerateAnnotation], ",")
	}

	t.Logf("checking regenerated keys are regenerated and have correct length")
	t.Logf("keys expected to be regenerated: %d", len(regenKeys))
	if len(regenKeys) != 0 {
		for _, key := range regenKeys {
			val := out.Data[key]
			if len(val) == 0 || len(val) != secretLength() {
				// check length here again, verifySecret skips this for secrets that already had the generatedAt Annotation
				t.Errorf("regenerated field has wrong length of %d", len(val))
			}

			if bytes.Compare(in.Data[key], val) == 0 {
				t.Errorf("key %s is equal for in(%s) and out (%s)", key, in.Data[key], out.Data[key])
				continue
			}
			t.Logf("key %s is NOT equal for in(%s) and out (%s)", key, in.Data[key], out.Data[key])
		}
	}

	t.Logf("checking generated keys are not regenerated")
	genKeys := strings.Split(in.Annotations[SecretGenerateAnnotation], ",")
	for _, key := range genKeys {
		if stringInSlice(key, regenKeys) {
			continue
		}
		if bytes.Compare(in.Data[key], out.Data[key]) == 0 {
			t.Logf("key %s is equal for in(%s) and out (%s)", key, in.Data[key], out.Data[key])
			continue
		}
		t.Errorf("key %s is NOT equal for in(%s) and out (%s)", key, in.Data[key], out.Data[key])
	}
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func TestGenerateSecretSingleField(t *testing.T) {
	in := newTestSecret("testfield", nil, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifySecret(t, in, out, true)
}

func TestGenerateSecretMultipleFields(t *testing.T) {
	in := newTestSecret("testfield,test1,test2,test3,abc,12345,6789", nil, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifySecret(t, in, out, true)
}

func TestRegenerateSingleField(t *testing.T) {
	in := newTestSecret("testfield", map[string]string{
		SecretRegenerateAnnotation:  "testfield",
		SecretGeneratedAtAnnotation: time.Now().String(),
	}, "test")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifySecret(t, in, out, true)
	verifyRegen(t, in, out)
}

func TestRegenerateAllSingleField(t *testing.T) {
	in := newTestSecret("testfield", map[string]string{
		SecretRegenerateAnnotation:  "yes",
		SecretGeneratedAtAnnotation: time.Now().String(),
	}, "test")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifySecret(t, in, out, true)
	verifyRegen(t, in, out)
}

func TestRegenerateMultipleFieldsSecure(t *testing.T) {
	in := newTestSecret("testfield,test1,test2", map[string]string{
		SecretRegenerateAnnotation:  "testfield",
		SecretGeneratedAtAnnotation: time.Now().String(),
		SecretSecureAnnotation:      "yes",
	}, "test,abc,def")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifySecret(t, in, out, true)
	verifyRegen(t, in, out)
}

func TestRegenerateMultipleFieldsNotSecure(t *testing.T) {
	in := newTestSecret("testfield,test1,test2", map[string]string{
		SecretRegenerateAnnotation:  "testfield",
		SecretGeneratedAtAnnotation: time.Now().String(),
	}, "test,abc,def")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifySecret(t, in, out, false)
	verifyRegen(t, in, out)
}

func TestRegenerateAllMultipleFields(t *testing.T) {
	in := newTestSecret("testfield,test1,test2", map[string]string{
		SecretRegenerateAnnotation:  "yes",
		SecretGeneratedAtAnnotation: time.Now().String(),
	}, "test,abc,def")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifySecret(t, in, out, true)
	verifyRegen(t, in, out)
}

func TestRegenerateInsecureSingleField(t *testing.T) {
	viper.Set("regenerate-insecure", true)
	in := newTestSecret("testfield", map[string]string{
		SecretGeneratedAtAnnotation: time.Now().String(),
	}, "test")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifySecret(t, in, out, true)
	verifyRegen(t, in, out)
	viper.Set("regenerate-insecure", false)
}

func TestRegenerateInsecureEmpty(t *testing.T) {
	viper.Set("regenerate-insecure", true)
	in := newTestSecret("testfield", nil, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifySecret(t, in, out, true)
	viper.Set("regenerate-insecure", false)
}

func TestRegenerateInsecureSingleFieldSecureBefore(t *testing.T) {
	viper.Set("regenerate-insecure", true)
	in := newTestSecret("testfield", map[string]string{
		SecretGeneratedAtAnnotation: time.Now().String(),
		SecretSecureAnnotation:      "yes",
	}, "test")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifySecret(t, in, out, true)
	verifyRegen(t, in, out)
	viper.Set("regenerate-insecure", false)
}

func TestRegenerateInsecureMultipleField(t *testing.T) {
	viper.Set("regenerate-insecure", true)
	in := newTestSecret("testfield,test1,test2,test3", map[string]string{
		SecretGeneratedAtAnnotation: time.Now().String(),
	}, "abc,def,ghi,jkl")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifySecret(t, in, out, true)
	verifyRegen(t, in, out)
	viper.Set("regenerate-insecure", false)
}

func TestRegenerateInsecureMultipleFieldSecureBefore(t *testing.T) {
	viper.Set("regenerate-insecure", true)
	in := newTestSecret("testfield,test1,test2,test3", map[string]string{
		SecretGeneratedAtAnnotation: time.Now().String(),
		SecretSecureAnnotation:      "yes",
	}, "abc,def,ghi,jkl")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifySecret(t, in, out, true)
	verifyRegen(t, in, out)
	viper.Set("regenerate-insecure", false)
}

func TestUniqueness(t *testing.T) {
	in := newTestSecret("testfield,abc,test,abc,oops,oops", nil, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, true)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))
}

func TestGeneratedSecretsHaveCorrectLength(t *testing.T) {
	pwd, err := generateSecret(20)

	t.Log("generated", pwd)

	if err != nil {
		t.Error(err)
	}

	if len(pwd) != 20 {
		t.Error("password length", "expected", 20, "got", len(pwd))
	}
}

func TestGeneratedSecretsAreRandom(t *testing.T) {
	one, errOne := generateSecret(32)
	two, errTwo := generateSecret(32)

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
		_, err := generateSecret(32)
		if err != nil {
			b.Error(err)
		}
	}
}
