package sshkeypair

import (
	"bytes"
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
	"github.com/mittwald/kubernetes-secret-generator/pkg/controller/secret"
)

var mgr manager.Manager

const labelSecretGeneratorTest = "kubernetes-secret-generator-test"
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
		if err.Error() == "error listening on :8080: listen tcp :8080: bind: address already in use" {

		} else {
			panic(err)
		}
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
	list := &v1alpha1.SSHKeyPairList{}
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

func newSSHKeyPairTestCR(sshSpec v1alpha1.SSHKeyPairSpec, name string) *v1alpha1.SSHKeyPair {
	if name == "" {
		name = uuid.New().String()
	}
	cr := &v1alpha1.SSHKeyPair{
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
		Spec: sshSpec,
	}

	return cr
}

func verifySSHSecretFromCR(t *testing.T, in *v1alpha1.SSHKeyPair, out *corev1.Secret) {
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

	publicKey := out.Data[secret.SecretFieldPublicKey]
	privateKey := out.Data[secret.SecretFieldPrivateKey]

	if len(privateKey) == 0 || len(publicKey) == 0 {
		t.Errorf("publicKey(%d) or privateKey(%d) have invalid length", len(publicKey), len(privateKey))
	}

	key, err := secret.PrivateKeyFromPEM(privateKey)
	if err != nil {
		t.Error(err, "generated private key could not be parsed")
	}

	err = key.Validate()
	if err != nil {
		t.Error(err, "key validation failed")
	}

	pub, err := secret.SshPublicKeyForPrivateKey(key)
	if err != nil {
		t.Error(err, "generated public key could not be parsed")
	}

	if !bytes.Equal(publicKey, pub) {
		t.Error("publicKey doesn't match private key")
	}
}

func doReconcile(t *testing.T, sshKeyPair *v1alpha1.SSHKeyPair, isErr bool) {
	rec := ReconcileSSHKeyPair{mgr.GetClient(), mgr.GetScheme()}
	req := reconcile.Request{NamespacedName: types.NamespacedName{Name: sshKeyPair.Name, Namespace: sshKeyPair.Namespace}}

	res, err := rec.Reconcile(req)

	if isErr {
		require.Error(t, err)
	} else {
		require.NoError(t, err)
	}
	require.False(t, res.Requeue)
}

func TestGenerateSecrets(t *testing.T) {
	testSpec := v1alpha1.SSHKeyPairSpec{
		Length: "40",
		Type:   string(corev1.SecretTypeOpaque),
		Data:   map[string]string{},
	}
	in := newSSHKeyPairTestCR(testSpec, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifySSHSecretFromCR(t, in, out)

	require.NoError(t, mgr.GetClient().Delete(context.TODO(), in))
	// give deletion time to be processed
	time.Sleep(1 * time.Second)
	out = &corev1.Secret{}
	err := mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, out)
	require.True(t, errors.IsNotFound(err), "Secret was not deleted upon cr deletion")
}

func TestRegenerateSecrets(t *testing.T) {
	testSpec := v1alpha1.SSHKeyPairSpec{
		Length:          "40",
		Type:            string(corev1.SecretTypeOpaque),
		Data:            map[string]string{},
		ForceRegenerate: true,
	}
	in := newSSHKeyPairTestCR(testSpec, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))
	verifySSHSecretFromCR(t, in, out)

	oldPrivateKey := string(out.Data[secret.SecretFieldPrivateKey])
	oldPublicKey := string(out.Data[secret.SecretFieldPublicKey])

	in.Spec.Length = "35"

	doReconcile(t, in, false)

	outNew := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, outNew))

	newPrivateKey := string(outNew.Data[secret.SecretFieldPrivateKey])
	newPublicKey := string(outNew.Data[secret.SecretFieldPublicKey])

	if oldPrivateKey == newPrivateKey {
		t.Errorf("secret has not been updated")
	}

	if oldPublicKey == newPublicKey {
		t.Errorf("secret has not been updated")
	}
}

func TestDoNotRegenerateSecrets(t *testing.T) {
	testSpec := v1alpha1.SSHKeyPairSpec{
		Length:          "40",
		Type:            string(corev1.SecretTypeOpaque),
		Data:            map[string]string{},
		ForceRegenerate: false,
	}
	in := newSSHKeyPairTestCR(testSpec, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))
	verifySSHSecretFromCR(t, in, out)

	oldPrivateKey := string(out.Data[secret.SecretFieldPrivateKey])
	oldPublicKey := string(out.Data[secret.SecretFieldPublicKey])

	in.Spec.Length = "35"

	doReconcile(t, in, false)

	outNew := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, outNew))

	newPrivateKey := string(outNew.Data[secret.SecretFieldPrivateKey])
	newPublicKey := string(outNew.Data[secret.SecretFieldPublicKey])

	if oldPrivateKey != newPrivateKey {
		t.Errorf("secret has been updated")
	}

	if oldPublicKey != newPublicKey {
		t.Errorf("secret has been updated")
	}
}

func TestDoNotRegenerateSecretsFixMissingPublicKey(t *testing.T) {
	testSpec := v1alpha1.SSHKeyPairSpec{
		Length:          "40",
		Type:            string(corev1.SecretTypeOpaque),
		Data:            map[string]string{},
		ForceRegenerate: false,
	}
	in := newSSHKeyPairTestCR(testSpec, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))
	verifySSHSecretFromCR(t, in, out)

	oldPrivateKey := string(out.Data[secret.SecretFieldPrivateKey])
	oldPublicKey := string(out.Data[secret.SecretFieldPublicKey])

	out.Data[secret.SecretFieldPublicKey] = []byte{}

	require.NoError(t, mgr.GetClient().Update(context.TODO(), out))

	in.Spec.Length = "35"

	doReconcile(t, in, false)

	outNew := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, outNew))

	newPrivateKey := string(outNew.Data[secret.SecretFieldPrivateKey])
	newPublicKey := string(outNew.Data[secret.SecretFieldPublicKey])

	if oldPrivateKey != newPrivateKey {
		t.Errorf("secret has been updated")
	}

	if oldPublicKey != newPublicKey {
		t.Errorf("secret has been updated")
	}
}

func TestRegeneratePublicKey(t *testing.T) {
	keyPair, err := secret.GenerateSSHKeypair(40)
	require.NoError(t, err)
	testSpec := v1alpha1.SSHKeyPairSpec{
		Length:          "40",
		PrivateKey:      string(keyPair.PrivateKey),
		Type:            string(corev1.SecretTypeOpaque),
		Data:            map[string]string{},
		ForceRegenerate: true,
	}
	in := newSSHKeyPairTestCR(testSpec, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))
	verifySSHSecretFromCR(t, in, out)

	privateKey := string(out.Data[secret.SecretFieldPrivateKey])

	if privateKey != string(keyPair.PrivateKey) {
		t.Errorf("Private key was regenerated")
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

	testSpec := v1alpha1.SSHKeyPairSpec{
		Length: "40",
		Type:   string(corev1.SecretTypeOpaque),
		Data:   map[string]string{},
	}
	in := newSSHKeyPairTestCR(testSpec, "")
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
