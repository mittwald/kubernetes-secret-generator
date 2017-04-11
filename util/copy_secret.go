/*
 * Copyright 2017 Martin Helmich <m.helmich@mittwald.de>
 *                Mittwald CM Service GmbH & Co. KG
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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