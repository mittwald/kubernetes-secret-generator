package secret

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"github.com/go-logr/logr"
	"golang.org/x/crypto/ssh"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

const (
	SecretFieldPublicKey  = "ssh-publickey"
	SecretFieldPrivateKey = "ssh-privatekey"
)

type SSHKeypairGenerator struct {
	log logr.Logger
}

type SSHKeypair struct {
	PrivateKey []byte
	PublicKey  []byte
}

func (sg SSHKeypairGenerator) generateData(instance *corev1.Secret) (reconcile.Result, error) {
	privateKey := instance.Data[SecretFieldPrivateKey]
	publicKey := instance.Data[SecretFieldPublicKey]

	regenerate := instance.Annotations[AnnotationSecretRegenerate] != ""

	// check for existing values, if regeneration isn't forced
	if len(privateKey) > 0 && !regenerate {
		if len(publicKey) == 0 {
			// restore public key if private key exists
			rsaKey, err := PrivateKeyFromPEM(privateKey)
			if err != nil {
				return reconcile.Result{}, err
			}

			publicKey, err = SshPublicKeyForPrivateKey(rsaKey)
			if err != nil {
				return reconcile.Result{}, err
			}

			instance.Data[SecretFieldPublicKey] = publicKey
		}

		// do nothing, both keys are present
		return reconcile.Result{}, nil
	}

	if regenerate {
		delete(instance.Annotations, AnnotationSecretRegenerate)
	}

	length, _, err := SecretLengthFromAnnotation(SSHKeyLength(), instance.Annotations)
	if err != nil {
		return reconcile.Result{}, err
	}

	keyPair, err := GenerateSSHKeypair(length)
	if err != nil {
		return reconcile.Result{RequeueAfter: time.Second * 30}, err
	}

	instance.Data[SecretFieldPublicKey] = keyPair.PublicKey
	instance.Data[SecretFieldPrivateKey] = keyPair.PrivateKey

	return reconcile.Result{}, nil
}

// generates ssh private and public key of given length
// the returned public key is in authorized-keys format
// the private key is PEM encoded
func GenerateSSHKeypair(length int) (SSHKeypair, error) {
	key, err := rsa.GenerateKey(rand.Reader, length)
	if err != nil {
		return SSHKeypair{}, err
	}

	privateKeyBytes := &bytes.Buffer{}
	err = pem.Encode(
		privateKeyBytes,
		&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	if err != nil {
		return SSHKeypair{}, err
	}

	publicKey, err := SshPublicKeyForPrivateKey(key)
	if err != nil {
		return SSHKeypair{}, err
	}

	return SSHKeypair{
		PublicKey:  publicKey,
		PrivateKey: privateKeyBytes.Bytes(),
	}, nil
}

func PrivateKeyFromPEM(pemKey []byte) (*rsa.PrivateKey, error) {
	b, _ := pem.Decode(pemKey)
	if b == nil {
		return nil, errors.New("failed to parse private Key PEM block")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(b.Bytes)
	if err != nil {
		return nil, err
	}
	return privateKey, nil
}

func SshPublicKeyForPrivateKey(privateKey *rsa.PrivateKey) ([]byte, error) {
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}

	return ssh.MarshalAuthorizedKey(publicKey), nil
}
