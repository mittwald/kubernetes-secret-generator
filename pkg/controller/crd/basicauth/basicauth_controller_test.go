package basicauth

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/flowcontrol"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/mittwald/kubernetes-secret-generator/pkg/apis"
	"github.com/mittwald/kubernetes-secret-generator/pkg/apis/types/v1alpha1"
	"github.com/mittwald/kubernetes-secret-generator/pkg/controller/crd"
	"github.com/mittwald/kubernetes-secret-generator/pkg/controller/secret"
)

var mgr manager.Manager

const labelSecretGeneratorTest = "kubernetes-secret-generator-test"

func TestMain(m *testing.M) {
	cfgPath := os.Getenv("KUBECONFIG")
	cfg, err := clientcmd.BuildConfigFromFlags("", cfgPath)

	if err != nil {
		panic(err)
	}

	restMapper := func(cfg *rest.Config) (meta.RESTMapper, error) {
		return apiutil.NewDynamicRESTMapper(cfg)
	}

	mgrOpts := manager.Options{
		MapperProvider: restMapper,
		NewClient: func(_ cache.Cache, config *rest.Config, options client.Options) (client.Client, error) {
			config.RateLimiter = flowcontrol.NewFakeAlwaysRateLimiter()
			return client.New(config, options)
		},
	}

	mgr, err = manager.New(cfg, mgrOpts)
	if err != nil {
		panic(err)
	}

	if err = apis.AddToScheme(mgr.GetScheme()); err != nil {
		panic(err)
	}

	if err = v1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		panic(err)
	}
	// Setup Scheme for all resources

	setupViper()
	reset()

	code := m.Run()

	os.Exit(code)
}

func setupViper() {
	viper.Set("secret-length", 40)
	viper.Set("regenerate-insecure", false)
	viper.Set("ssh-key-length", 2048)
}

func reset() {
	list := &corev1.SecretList{}
	err := mgr.GetClient().List(context.TODO(),
		list,
		client.MatchingLabels(map[string]string{
			labelSecretGeneratorTest: "yes",
		}),
	)
	if err != nil {
		panic(err)
	}

	for _, s := range list.Items {
		err := mgr.GetClient().Delete(context.TODO(), &s)
		if err != nil {
			panic(err)
		}
	}
}

func newBasicAuthTestCR(authSpec v1alpha1.BasicAuthSpec) *v1alpha1.BasicAuth {
	cr := &v1alpha1.BasicAuth{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "mittwald.systems/v1alpha1",
			Kind:       "BasicAuth",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      uuid.New().String(),
			Namespace: "default",
			Labels: map[string]string{
				labelSecretGeneratorTest: "yes",
			},
		},
		Spec: authSpec,
	}

	return cr
}

func verifyBasicAuthSecretFromCR(t *testing.T, in *v1alpha1.BasicAuth, out *corev1.Secret) {
	// Check for correct ownership
	for index := range out.OwnerReferences {
		if out.OwnerReferences[index].Kind == Kind {
			break
		}
		if index == len(out.OwnerReferences)-1 {
			t.Errorf("generated secret not owned by kind %s", Kind)
		}
	}
	fmt.Println(in)
	fmt.Println(out)
	if in.Status.Secret.Name != out.Name {
		t.Error("generated secret not referenced in CR status")
	}

	auth := out.Data[secret.SecretFieldBasicAuthIngress]
	password := out.Data[secret.SecretFieldBasicAuthPassword]
	desiredLength, _, err := crd.ParseByteLength(secret.SecretLength(), in.Spec.Length)
	if err != nil {
		t.Error("Failed to determine secret length")
	}
	// check if password has been saved in clear text
	// and has correct length (if the secret has actually been generated)
	if len(password) == 0 || len(password) != desiredLength {
		t.Errorf("generated field has wrong length of %d", len(password))
	}

	// check if auth field has been generated (with separator)
	if len(auth) == 0 || !strings.Contains(string(auth), ":") {
		t.Errorf("auth field has wrong or no values %s", string(auth))
	}
}

func TestGenerateBasicAuthWithoutUsername(t *testing.T) {
	testSpec := v1alpha1.BasicAuthSpec{
		Encoding: "base64",
		Length:   "40",
		Type:     string(corev1.SecretTypeOpaque),
		Data:     map[string]string{},
	}

	in := newBasicAuthTestCR(testSpec)
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifyBasicAuthSecretFromCR(t, in, out)
}

func doReconcile(t *testing.T, basicAuth *v1alpha1.BasicAuth, isErr bool) {
	rec := ReconcileBasicAuth{mgr.GetClient(), mgr.GetScheme()}
	req := reconcile.Request{NamespacedName: types.NamespacedName{Name: basicAuth.Name, Namespace: basicAuth.Namespace}}

	res, err := rec.Reconcile(req)

	if isErr {
		require.Error(t, err)
	} else {
		require.NoError(t, err)
	}
	require.False(t, res.Requeue)
}
