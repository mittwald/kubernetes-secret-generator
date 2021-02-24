package crd

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mittwald/kubernetes-secret-generator/pkg/apis/types/v1alpha1"
)

func TestNewSecret(t *testing.T) {
	owner := v1alpha1.StringSecret{}
	owner.Name = "testSecret"
	owner.Namespace = "testns"
	owner.Labels = map[string]string{"test": "test"}

	data := map[string][]byte{"test": []byte("blah")}

	target, err := NewSecret(&owner, data, "opaque")
	require.NoError(t, err)
	require.Equal(t, "testSecret", target.Name)
	require.Equal(t, "testns", target.Namespace)
	require.Equal(t, map[string]string{"test": "test"}, target.Labels)

}
