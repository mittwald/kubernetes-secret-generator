package util

import (
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/api"
)

func CopyObjToSecret(secret *v1.Secret) (*v1.Secret, error) {
	objCopy, err := api.Scheme.Copy(secret)
	if err != nil {
		return nil, err
	}

	secretCopy := objCopy.(*v1.Secret)
	if secretCopy.Annotations == nil {
		secretCopy.Annotations = make(map[string]string)
	}

	return secretCopy, nil
}