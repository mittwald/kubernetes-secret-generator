package secret

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	AnnotationSecretGenerate    = "secret-generator.v1.mittwald.de/autogenerate"
	AnnotationSecretGeneratedAt = "secret-generator.v1.mittwald.de/autogenerate-generated-at"
	AnnotationSecretRegenerate  = "secret-generator.v1.mittwald.de/regenerate"
	AnnotationSecretSecure      = "secret-generator.v1.mittwald.de/secure"
	AnnotationSecretType        = "secret-generator.v1.mittwald.de/type"
	AnnotationSecretLength      = "secret-generator.v1.mittwald.de/length"
)

type SecretType string

const (
	SecretTypeString     SecretType = "string"
	SecretTypeSSHKeypair SecretType = "ssh-keypair"
)

func (st SecretType) Validate() error {
	switch st {
	case SecretTypeString,
		SecretTypeSSHKeypair:
		return nil
	}
	return fmt.Errorf("%s is not a valid secret type", st)
}

type SecretGenerator interface {
	generateData(*corev1.Secret) (reconcile.Result, error)
}
