package secret_test

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"text/template"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/mittwald/kubernetes-secret-generator/pkg/apis/secretgenerator/v1alpha1"
	"github.com/mittwald/kubernetes-secret-generator/pkg/controller/crd/templatesecret"
	"github.com/mittwald/kubernetes-secret-generator/pkg/controller/secret"
)

func newTemplateSecretTestCR(templates map[string]string, fields []v1alpha1.TemplateField, regenerate bool) *v1alpha1.TemplateSecret {
	return &v1alpha1.TemplateSecret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiVersion,
			Kind:       templatesecret.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      uuid.NewString(),
			Namespace: "default",
			Labels: map[string]string{
				labelSecretGeneratorTest: "yes",
			},
		},
		Spec: v1alpha1.TemplateSecretSpec{
			ForceRegenerate: regenerate,
			Data:            templates,
			Fields:          fields,
		},
	}
}

func verifyTemplateSecretFromCR(t *testing.T, in *v1alpha1.TemplateSecret, out *corev1.Secret) {
	// Check for correct ownership
	for index := range out.OwnerReferences {
		if out.OwnerReferences[index].Kind == templatesecret.Kind {
			break
		}
		if index == len(out.OwnerReferences)-1 {
			t.Errorf("generated secret not owned by kind %s", templatesecret.Kind)
		}
	}

	// check if cr status was updated properly with secret reference
	if in.Status.Secret != nil && in.Status.Secret.Name != out.Name {
		t.Error("generated secret not referenced in CR status")
	}

	valmp := map[string]string{}

	for _, f := range in.Spec.Fields {
		name := f.FieldName
		length := f.Length
		if len(length) == 0 {
			length = fmt.Sprintf("%d", secret.DefaultLength())
		}

		lenint, err := strconv.ParseInt(length, 10, 32)
		if err != nil {
			t.Errorf("error parsing length of field: %v", err)
		}

		str := strings.Repeat("-", int(lenint))

		valmp[name] = str
	}

	for k, v := range in.Spec.Data {
		val, ok := out.Data[k]
		if !ok {
			t.Errorf("field %q does not exist in secret", k)
			continue
		}

		tmpl, err := template.New(k).Parse(v)
		if err != nil {
			t.Errorf("error parsing template %s: %v", k, err)
		}
		buf := bytes.Buffer{}
		err = tmpl.Execute(&buf, valmp)
		if err != nil {
			t.Errorf("error parsing template %s: %v", k, err)
		}

		if len(val) != buf.Len() {
			t.Errorf("field %q: secret length %d != expected length %d\nSecret: %q", k, len(val), buf.Len(), string(val))
		}
	}
}

func doReconcileTemplateSecretController(t *testing.T, templateSecret *v1alpha1.TemplateSecret, isErr bool) {
	rec := templatesecret.NewReconciler(mgr)
	req := reconcile.Request{NamespacedName: types.NamespacedName{Name: templateSecret.Name, Namespace: templateSecret.Namespace}}

	res, err := rec.Reconcile(req)

	if isErr {
		require.Error(t, err)
	} else {
		require.NoError(t, err)
	}
	require.False(t, res.Requeue)
}

func TestControllerGenerateTemplateSecret(t *testing.T) {
	tests := []struct {
		name   string
		fields []v1alpha1.TemplateField
		data   map[string]string
	}{
		{
			"simple single field",
			[]v1alpha1.TemplateField{
				{FieldName: "test", Length: "10", Encoding: "ascii"},
			},
			map[string]string{
				"test1": "hello {{ .test }}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTemplateSecretTestCR(tt.data, tt.fields, false)

			require.NoError(t, mgr.GetClient().Create(context.TODO(), ts))

			doReconcileTemplateSecretController(t, ts, false)

			out := &corev1.Secret{}
			require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
				Name:      ts.Name,
				Namespace: ts.Namespace,
			}, out))

			verifyTemplateSecretFromCR(t, ts, out)

			require.NoError(t, mgr.GetClient().Delete(context.TODO(), ts))
		})
	}
}

func TestControllerRegenerateTemplateSecret(t *testing.T) {
	tests := []struct {
		name   string
		fields []v1alpha1.TemplateField
		data   map[string]string
	}{
		{
			"simple single field",
			[]v1alpha1.TemplateField{
				{FieldName: "test", Length: "10", Encoding: "ascii"},
			},
			map[string]string{
				"test1": "hello {{ .test }}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTemplateSecretTestCR(tt.data, tt.fields, true)

			require.NoError(t, mgr.GetClient().Create(context.TODO(), ts))

			doReconcileTemplateSecretController(t, ts, false)

			out := &corev1.Secret{}
			require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
				Name:      ts.Name,
				Namespace: ts.Namespace,
			}, out))

			verifyTemplateSecretFromCR(t, ts, out)

			oldData := out.Data

			doReconcileTemplateSecretController(t, ts, false)

			outNew := &corev1.Secret{}
			require.NoError(t, mgr.GetClient().Get(context.TODO(), client.ObjectKey{
				Name:      ts.Name,
				Namespace: ts.Namespace}, outNew))

			newData := outNew.Data

			for k, vold := range oldData {
				vnew, ok := newData[k]
				if !ok {
					t.Errorf("new secret does not contain expected key %q", k)
					continue
				}

				if string(vnew) == string(vold) {
					t.Errorf("value for %q was not updated", k)
				}
			}

			require.NoError(t, mgr.GetClient().Delete(context.TODO(), ts))
		})
	}
}
