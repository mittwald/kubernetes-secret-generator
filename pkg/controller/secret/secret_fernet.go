package secret

import (
	"strings"
	"time"

	"github.com/fernet/fernet-go"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type FernetGenerator struct {
	log logr.Logger
}

type secretFernetConfig struct {
	instance *corev1.Secret
	key      string
}

func (pg FernetGenerator) generateData(instance *corev1.Secret) (reconcile.Result, error) {
	toGenerate := instance.Annotations[AnnotationSecretAutoGenerate]

	genKeys := strings.Split(toGenerate, ",")

	if err := ensureUniqueness(genKeys); err != nil {
		return reconcile.Result{}, err
	}

	return pg.regenerateKeysWhereRequired(instance, genKeys)
}

func (pg FernetGenerator) generateFernetKey(conf secretFernetConfig) error {
	key := conf.key
	instance := conf.instance

	value, err := GenerateFernetKey()
	if err != nil {
		return err
	}
	instance.Data[key] = value

	pg.log.Info("set field of instance to new randomly generated instance", "field", key)

	return nil
}

// GenerateFernetKey generates a new fernet key with the specified encoding.
func GenerateFernetKey() ([]byte, error) {
	var fernetKey fernet.Key

	err := fernetKey.Generate()
	if err != nil {
		return nil, err
	}

	return []byte(fernetKey.Encode()), nil
}

func (pg FernetGenerator) regenerateKeysWhereRequired(instance *corev1.Secret, genKeys []string) (reconcile.Result, error) {
	var regenKeys []string

	if regenerate, ok := instance.Annotations[AnnotationSecretRegenerate]; ok {
		pg.log.Info("removing regenerate annotation from instance")
		delete(instance.Annotations, AnnotationSecretRegenerate)

		if regenerate == "yes" {
			regenKeys = genKeys
		} else {
			regenKeys = strings.Split(regenerate, ",") // regenerate requested keys
		}
	}

	generatedCount := 0
	for _, key := range genKeys {
		if len(instance.Data[key]) != 0 && !contains(regenKeys, key) {
			// dont generate key if it already has a value
			// and is not queued for regeneration
			continue
		}
		generatedCount++

		err := pg.generateFernetKey(secretFernetConfig{instance, key})
		if err != nil {
			pg.log.Error(err, "could not generate new fernet key")
			return reconcile.Result{RequeueAfter: time.Second * 30}, err
		}
	}

	pg.log.Info("generated secrets", "count", generatedCount)

	return reconcile.Result{}, nil
}
