package secret_test

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/mittwald/kubernetes-secret-generator/pkg/apis/secretgenerator/v1alpha1"
	"github.com/mittwald/kubernetes-secret-generator/pkg/controller/crd/sshkeypair"
	"github.com/mittwald/kubernetes-secret-generator/pkg/controller/secret"
)

const (
	TestSecretFieldPrivateKey = "sshPrivateKey"
	TestSecretFieldPublicKey  = "sshPublicKey"
)

// newSSHKeyPairTestCR returns a SSHKeyPair custom resource. If name is set to "", a uuid will be generated
func newSSHKeyPairTestCR(sshSpec v1alpha1.SSHKeyPairSpec, name string) *v1alpha1.SSHKeyPair {
	if name == "" {
		name = uuid.New().String()
	}
	cr := &v1alpha1.SSHKeyPair{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiVersion,
			Kind:       sshkeypair.Kind,
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

// verifySSHSecretFromCR checks if the given Secret was correctly created from the cr
func verifySSHSecretFromCR(t *testing.T, in *v1alpha1.SSHKeyPair, out *corev1.Secret) {
	// Check for correct ownership
	for index := range out.OwnerReferences {
		if out.OwnerReferences[index].Kind == sshkeypair.Kind {
			break
		}
		if index == len(out.OwnerReferences)-1 {
			t.Errorf("generated secret not owned by kind %s", sshkeypair.Kind)
		}
	}

	// check if cr status was updated properly with secret reference
	if in.Status.Secret != nil && in.Status.Secret.Name != out.Name {
		t.Error("generated secret not referenced in CR status")
	}

	publicKey := out.Data[in.GetPublicKeyField()]
	privateKey := out.Data[in.GetPrivateKeyField()]

	// check if keys have valid length
	if len(privateKey) == 0 || len(publicKey) == 0 {
		t.Errorf("publicKey(%d) or privateKey(%d) have invalid length", len(publicKey), len(privateKey))
	}

	// verify validity of private key
	key, err := secret.PrivateKeyFromPEM(privateKey)
	if err != nil {
		t.Error(err, "generated private key could not be parsed")
	}

	err = key.Validate()
	if err != nil {
		t.Error(err, "key validation failed")
	}

	pub, err := secret.SSHPublicKeyForPrivateKey(key)
	if err != nil {
		t.Error(err, "generated public key could not be parsed")
	}

	if !bytes.Equal(publicKey, pub) {
		t.Error("publicKey doesn't match private key")
	}

	// check if custom data entries were set correctly
	for _, key := range in.Spec.Data {
		if _, ok := out.Data[key]; !ok {
			t.Errorf("missing data entry %s", key)
		}
	}
}

func doReconcileSSHKeyPairController(t *testing.T, sshKeyPair *v1alpha1.SSHKeyPair, isErr bool) {
	rec := sshkeypair.NewReconciler(mgr)
	req := reconcile.Request{NamespacedName: types.NamespacedName{Name: sshKeyPair.Name, Namespace: sshKeyPair.Namespace}}

	res, err := rec.Reconcile(req)

	if isErr {
		require.Error(t, err)
	} else {
		require.NoError(t, err)
	}
	require.False(t, res.Requeue)
}

func TestControllerGenerateSSHSecret(t *testing.T) {
	testSpec := v1alpha1.SSHKeyPairSpec{
		Length: "40",
		Type:   string(corev1.SecretTypeOpaque),
		Data:   map[string]string{},
	}
	in := newSSHKeyPairTestCR(testSpec, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcileSSHKeyPairController(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifySSHSecretFromCR(t, in, out)

	require.NoError(t, mgr.GetClient().Delete(context.TODO(), in))
}

func TestControllerRegenerateSSHSecret(t *testing.T) {
	testSpec := v1alpha1.SSHKeyPairSpec{
		Length:          "40",
		Type:            string(corev1.SecretTypeOpaque),
		Data:            map[string]string{},
		ForceRegenerate: true,
	}
	in := newSSHKeyPairTestCR(testSpec, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcileSSHKeyPairController(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))
	verifySSHSecretFromCR(t, in, out)

	oldPrivateKey := string(out.Data[secret.DefaultSecretFieldPrivateKey])
	oldPublicKey := string(out.Data[secret.DefaultSecretFieldPublicKey])

	in.Spec.Length = "35"

	doReconcileSSHKeyPairController(t, in, false)

	outNew := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, outNew))

	newPrivateKey := string(outNew.Data[secret.DefaultSecretFieldPrivateKey])
	newPublicKey := string(outNew.Data[secret.DefaultSecretFieldPublicKey])

	if oldPrivateKey == newPrivateKey {
		t.Errorf("secret has not been updated")
	}

	if oldPublicKey == newPublicKey {
		t.Errorf("secret has not been updated")
	}
}

func TestControllerDoNotRegenerateSecret(t *testing.T) {
	testSpec := v1alpha1.SSHKeyPairSpec{
		Length:          "40",
		Type:            string(corev1.SecretTypeOpaque),
		Data:            map[string]string{},
		ForceRegenerate: false,
	}
	in := newSSHKeyPairTestCR(testSpec, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcileSSHKeyPairController(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))
	verifySSHSecretFromCR(t, in, out)

	oldPrivateKey := string(out.Data[secret.DefaultSecretFieldPrivateKey])
	oldPublicKey := string(out.Data[secret.DefaultSecretFieldPublicKey])

	in.Spec.Length = "35"

	doReconcileSSHKeyPairController(t, in, false)

	outNew := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, outNew))

	newPrivateKey := string(outNew.Data[secret.DefaultSecretFieldPrivateKey])
	newPublicKey := string(outNew.Data[secret.DefaultSecretFieldPublicKey])

	if oldPrivateKey != newPrivateKey {
		t.Errorf("secret has been updated")
	}

	if oldPublicKey != newPublicKey {
		t.Errorf("secret has been updated")
	}
}

func TestControllerDoNotRegenerateSSHSecretFixMissingPublicKey(t *testing.T) {
	testSpec := v1alpha1.SSHKeyPairSpec{
		Length:          "40",
		Type:            string(corev1.SecretTypeOpaque),
		Data:            map[string]string{},
		ForceRegenerate: false,
	}
	in := newSSHKeyPairTestCR(testSpec, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcileSSHKeyPairController(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))
	verifySSHSecretFromCR(t, in, out)

	oldPrivateKey := string(out.Data[secret.DefaultSecretFieldPrivateKey])
	oldPublicKey := string(out.Data[secret.DefaultSecretFieldPublicKey])

	out.Data[secret.DefaultSecretFieldPublicKey] = []byte{}

	require.NoError(t, mgr.GetClient().Update(context.TODO(), out))

	in.Spec.Length = "35"

	doReconcileSSHKeyPairController(t, in, false)

	outNew := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, outNew))

	newPrivateKey := string(outNew.Data[secret.DefaultSecretFieldPrivateKey])
	newPublicKey := string(outNew.Data[secret.DefaultSecretFieldPublicKey])

	if oldPrivateKey != newPrivateKey {
		t.Errorf("secret has been updated")
	}

	if oldPublicKey != newPublicKey {
		t.Errorf("secret has been updated")
	}
}

func TestControllerRegeneratePublicKey(t *testing.T) {
	data := make(map[string][]byte)
	var log logr.Logger
	err := secret.GenerateSSHKeypairData(log, "40", TestSecretFieldPrivateKey, TestSecretFieldPublicKey, true, data)
	require.NoError(t, err)
	testSpec := v1alpha1.SSHKeyPairSpec{
		Length:          "40",
		PrivateKey:      string(data[TestSecretFieldPrivateKey]),
		PrivateKeyField: TestSecretFieldPrivateKey,
		PublicKeyField:  TestSecretFieldPublicKey,
		Type:            string(corev1.SecretTypeOpaque),
		Data:            map[string]string{},
		ForceRegenerate: true,
	}
	in := newSSHKeyPairTestCR(testSpec, "")
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcileSSHKeyPairController(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))
	verifySSHSecretFromCR(t, in, out)

	privateKey := string(out.Data[TestSecretFieldPrivateKey])

	if privateKey != string(data[TestSecretFieldPrivateKey]) {
		t.Errorf("Private key was regenerated")
	}
}

func TestControllerDoNotTouchOtherSSHSecrets(t *testing.T) {
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

	doReconcileSSHKeyPairController(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      secret.Name,
		Namespace: secret.Namespace}, out))

	if !reflect.DeepEqual(secret, out) {
		t.Errorf("secret not owned by BasicAuth cr has been reconciled")
	}

	require.NoError(t, mgr.GetClient().Delete(context.TODO(), secret))
}
