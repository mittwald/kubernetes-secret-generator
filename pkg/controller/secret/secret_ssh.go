package secret

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"time"

	"github.com/go-logr/logr"
	"golang.org/x/crypto/ssh"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
       DefaultSecretFieldPublicKey  = "ssh-publickey"
       DefaultSecretFieldPrivateKey = "ssh-privatekey"
)

type SSHKeypairGenerator struct {
	log logr.Logger
}

func (sg SSHKeypairGenerator) generateData(instance *corev1.Secret) (reconcile.Result, error) {
	regenerate := instance.Annotations[AnnotationSecretRegenerate] != ""

	if regenerate {
		delete(instance.Annotations, AnnotationSecretRegenerate)
	}

	length, err := GetLengthFromAnnotation(SSHKeyLength(), instance.Annotations)
	if err != nil {
		return reconcile.Result{}, err
	}

	privateKeyField, err := GetPrivateKeyFieldFromAnnotation(DefaultSecretFieldPrivateKey, instance.Annotations)
	if err != nil {
		return reconcile.Result{}, err
	}

	publicKeyField, err := GetPublicKeyFieldFromAnnotation(DefaultSecretFieldPublicKey, instance.Annotations)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = GenerateSSHKeypairData(sg.log, length, privateKeyField, publicKeyField, regenerate, instance.Data)
	if err != nil {
		return reconcile.Result{RequeueAfter: time.Second * 30}, err
	}

	return reconcile.Result{}, nil
}

// generates ssh private and public key of given length
// and writes the result to data. The public key is in authorized-keys format,
// the private key is PEM encoded
func GenerateSSHKeypairData(logger logr.Logger, length string, privateKeyField string, publicKeyField string, regenerate bool, data map[string][]byte) error {
	privateKey := data[privateKeyField]
	publicKey := data[publicKeyField]

	if len(privateKey) > 0 && !regenerate {
		return CheckAndRegenPublicKey(data, publicKey, privateKey, publicKeyField)
	}

	key, err := generateNewPrivateKey(length, logger)
	if err != nil {
		return err
	}

	return generateKeysHelper(key, privateKeyField, publicKeyField, data)
}

// generateNewPrivateKey parses the given length and generates a matching private key
func generateNewPrivateKey(length string, logger logr.Logger) (*rsa.PrivateKey, error) {
	// check for existing values, if regeneration isn't forced

	parsedLen, _, err := ParseByteLength(DefaultLength(), length)
	if err != nil {
		logger.Error(err, "could not parse length for new random string")

		return nil, err
	}
	return rsa.GenerateKey(rand.Reader, parsedLen)
}

// generateKeysHelper generates the public key from the given private key and stores the result in data
func generateKeysHelper(key *rsa.PrivateKey, privateKeyField string, publicKeyField string, data map[string][]byte) error {
	privateKeyBytes := &bytes.Buffer{}
	err := pem.Encode(
		privateKeyBytes,
		&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	if err != nil {
		return err
	}

	var publicKeyBytes []byte
	publicKeyBytes, err = SSHPublicKeyForPrivateKey(key)
	if err != nil {
		return err
	}

	data[publicKeyField] = publicKeyBytes
	data[privateKeyField] = privateKeyBytes.Bytes()

	return nil
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

func SSHPublicKeyForPrivateKey(privateKey *rsa.PrivateKey) ([]byte, error) {
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}

	return ssh.MarshalAuthorizedKey(publicKey), nil
}

// CheckAndRegenPublicKey checks if the specified public key has length > 0 and regenerates it from the given private key
// otherwise. The result is written into data
func CheckAndRegenPublicKey(data map[string][]byte, publicKey, privateKey []byte, publicKeyField string) error {
	if len(publicKey) > 0 {
		return nil
	}

	// restore public key if private key exists
	rsaKey, err := PrivateKeyFromPEM(privateKey)
	if err != nil {
		return err
	}
	publicKey, err = SSHPublicKeyForPrivateKey(rsaKey)
	if err != nil {
		return err
	}
	data[publicKeyField] = publicKey

	return nil
}
