package basicauth

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/mittwald/kubernetes-secret-generator/pkg/apis"
	"github.com/mittwald/kubernetes-secret-generator/pkg/apis/types/v1alpha1"
	"github.com/mittwald/kubernetes-secret-generator/pkg/controller/crd"
	"github.com/mittwald/kubernetes-secret-generator/pkg/controller/secret"
)

var mgr manager.Manager

const labelSecretGeneratorTest = "kubernetes-secret-generator-test"
const testUsername = "testuser123"
const testSecretName = "testsec123"

func TestMain(m *testing.M) {
	cfgPath := os.Getenv("KUBECONFIG")
	cfg, err := clientcmd.BuildConfigFromFlags("", cfgPath)

	if err != nil {
		panic(err)
	}

	restMapper := func(cfg *rest.Config) (meta.RESTMapper, error) {
		return apiutil.NewDynamicRESTMapper(cfg)
	}

	// Set default manager options
	options := manager.Options{
		Namespace:      "default",
		MapperProvider: restMapper,
		NewCache:       cache.MultiNamespacedCacheBuilder(strings.Split("default", ",")),
		NewClient: func(_ cache.Cache, config *rest.Config, options client.Options) (client.Client, error) {
			return client.New(config, options)
		},
	}

	err = v1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		panic(err)
	}

	mgr, err = manager.New(cfg, options)
	if err != nil {
		panic(err)
	}

	if err = apis.AddToScheme(mgr.GetScheme()); err != nil {
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
	viper.Set("ssh-key-length", 2048)
}

func reset() {
	list := &v1alpha1.BasicAuthList{}
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
	secList := &corev1.SecretList{}
	err = mgr.GetClient().List(context.TODO(),
		secList,
		client.MatchingLabels(map[string]string{
			labelSecretGeneratorTest: "yes",
		}),
	)
	if err != nil {
		panic(err)
	}

	for _, s := range secList.Items {
		err := mgr.GetClient().Delete(context.TODO(), &s)
		if err != nil {
			panic(err)
		}
	}
}

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
		if out.OwnerReferences[index].Kind == Kind {
			break
		}
		if index == len(out.OwnerReferences)-1 {
			t.Errorf("generated secret not owned by kind %s", Kind)
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

func TestGenerateBasicAuthWithoutUsername(t *testing.T) {
	testSpec := v1alpha1.BasicAuthSpec{
		Encoding: "base64",
		Length:   "40",
		Type:     string(corev1.SecretTypeOpaque),
		Data:     map[string]string{},
	}

	in := newBasicAuthTestCR(testSpec, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

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
	// give deletion time to be processed, more time because kind cluster takes forever
	time.Sleep(1 * time.Second)

	out = &corev1.Secret{}
	err := mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, out)
	require.True(t, errors.IsNotFound(err), "Secret was not deleted upon cr deletion")
}

func TestGenerateBasicAuthWithUsername(t *testing.T) {
	testSpec := v1alpha1.BasicAuthSpec{
		Encoding: "base64",
		Length:   "40",
		Username: testUsername,
		Type:     string(corev1.SecretTypeOpaque),
		Data:     map[string]string{},
	}

	in := newBasicAuthTestCR(testSpec, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

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
	require.NoError(t, mgr.GetClient().Delete(context.TODO(), in))
	// give deletion time to be processed, more time since kind cluster takes forever
	time.Sleep(1 * time.Second)
	out = &corev1.Secret{}
	err := mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, out)
	require.True(t, errors.IsNotFound(err), "Secret was not deleted upon cr deletion")
}

func doReconcile(t *testing.T, basicAuth *v1alpha1.BasicAuth, isErr bool) {
	rec := ReconcileBasicAuth{mgr.GetClient(), mgr.GetScheme()}
	req := reconcile.Request{NamespacedName: types.NamespacedName{Name: basicAuth.Name, Namespace: basicAuth.Namespace}}

	res, err := rec.Reconcile(req)

	if isErr {
		require.Error(t, err)
	} else {
		require.NoError(t, err)
	}
	require.False(t, res.Requeue)
}

func TestGenerateBasicAuthNoRegenerate(t *testing.T) {
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

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, out))
	oldPassword := string(out.Data[secret.SecretFieldBasicAuthPassword])
	oldAuth := string(out.Data[secret.SecretFieldBasicAuthIngress])

	verifyBasicAuthSecretFromCR(t, in, out)

	in.Spec.Username = "Hugo"
	in.Spec.Length = "35"
	doReconcile(t, in, false)

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

func TestGenerateBasicAuthRegenerate(t *testing.T) {
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

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, out))
	oldPassword := string(out.Data[secret.SecretFieldBasicAuthPassword])
	oldAuth := string(out.Data[secret.SecretFieldBasicAuthIngress])

	verifyBasicAuthSecretFromCR(t, in, out)

	in.Spec.Username = "Hugo"
	in.Spec.Length = "35"
	doReconcile(t, in, false)

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

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      secret.Name,
		Namespace: secret.Namespace}, out))

	if !reflect.DeepEqual(secret, out) {
		t.Errorf("secret not owned by BasicAuth cr has been reconciled")
	}
}
