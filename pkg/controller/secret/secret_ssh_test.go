package secret

import (
	"bytes"
	"context"
	"github.com/imdario/mergo"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"testing"
	"time"
)

func newSSHKeypairTestSecret(t *testing.T, extraAnnotations map[string]string, initialized bool) *corev1.Secret {
	annotations := map[string]string{
		AnnotationSecretType: string(SecretTypeSSHKeypair),
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

	if initialized {
		keypair, err := generateSSHKeypair()
		if err != nil {
			t.Error(err, "could not generate new ssh keypair")
		}
		s.Data[SecretFieldPublicKey] = keypair.PublicKey
		s.Data[SecretFieldPrivateKey] = keypair.PrivateKey

		s.Annotations[AnnotationSecretGeneratedAt] = time.Now().Format(time.RFC3339)
	}

	return s
}

func verifySSHKeypairSecret(t *testing.T, in, out *corev1.Secret) {
	if out.Annotations[AnnotationSecretType] != string(SecretTypeSSHKeypair) {
		t.Errorf("generated secret has wrong type %s on  %s annotation", out.Annotations[AnnotationSecretType], AnnotationSecretType)
	}

	if _, ok := out.Annotations[AnnotationSecretGeneratedAt]; !ok {
		t.Errorf("secret has no %s annotation", AnnotationSecretGeneratedAt)
	}

	publicKey := out.Data[SecretFieldPublicKey]
	privateKey := out.Data[SecretFieldPrivateKey]

	if len(privateKey) == 0 || len(publicKey) == 0 {
		t.Errorf("publicKey(%d) or privateKey(%d) have invalid length", len(publicKey), len(privateKey))
	}

	key, err := privateKeyFromPEM(privateKey)
	if err != nil {
		t.Error(err, "generated private key could not be parsed")
	}

	pub, err := sshPublicKeyForPrivateKey(key)
	if err != nil {
		t.Error(err, "generated public key could not be parsed")
	}

	if !bytes.Equal(publicKey, pub) {
		t.Error("publicKey doesn't match private key")
	}
}

func verifySSHKeypairRegen(t *testing.T, in, out *corev1.Secret, regenDesired bool) {
	if _, ok := out.Annotations[AnnotationSecretRegenerate]; ok {
		t.Errorf("%s annotation is still present", AnnotationSecretRegenerate)
	}

	if _, ok := in.Annotations[AnnotationSecretRegenerate]; !ok && regenDesired { // test the tester
		t.Errorf("%s annotation is not present on input", AnnotationSecretRegenerate)
	}

	if _, ok := in.Annotations[AnnotationSecretGeneratedAt]; !ok { // test the tester
		t.Errorf("%s annotation is not present on input", AnnotationSecretGeneratedAt)
	}

	t.Logf("checking if keys have been regenerated")
	oldPublicKey := in.Data[SecretFieldPublicKey]
	oldPrivateKey := in.Data[SecretFieldPrivateKey]

	newPublicKey := out.Data[SecretFieldPublicKey]
	newPrivateKey := out.Data[SecretFieldPrivateKey]

	equal := bytes.Equal(oldPublicKey, newPublicKey)
	if equal && regenDesired {
		t.Error("publicKey has not been regenerated")
	} else if !equal && !regenDesired {
		t.Error("publicKey has been regenerated")
	}

	equal = bytes.Equal(oldPrivateKey, newPrivateKey)
	if equal && regenDesired {
		t.Error("privateKey has not been regenerated")
	} else if !equal && !regenDesired {
		t.Error("privateKey has been regenerated")
	}
}

func TestSSHKeypairIsGenerated(t *testing.T) {
	in := newSSHKeypairTestSecret(t, nil, false)
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))
	verifySSHKeypairSecret(t, in, out)
}

func TestSSHKeypairIsNotRegenerated(t *testing.T) {
	in := newSSHKeypairTestSecret(t, nil, true)
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))
	verifySSHKeypairSecret(t, in, out)
	verifySSHKeypairRegen(t, in, out, false)
}

func TestSSHKeypairIsRegenerated(t *testing.T) {
	in := newSSHKeypairTestSecret(t, map[string]string{
		AnnotationSecretRegenerate: "true",
	}, true)
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace}, out))
	verifySSHKeypairSecret(t, in, out)
	verifySSHKeypairRegen(t, in, out, true)
}
