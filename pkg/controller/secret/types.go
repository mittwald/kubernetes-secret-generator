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
)

type SecretType string

const (
	SecretTypePassword   SecretType = "password"
	SecretTypeSSHKeypair SecretType = "ssh-keypair"
)

func (st SecretType) IsValid() error {
	switch st {
	case SecretTypePassword,
		SecretTypeSSHKeypair:
		return nil
	}
	return fmt.Errorf("%s is not a valid secret type", st)
}

type SecretGenerator interface {
	generateData(*corev1.Secret) (reconcile.Result, error)
}
