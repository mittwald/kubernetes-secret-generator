package secret_test

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/flowcontrol"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/mittwald/kubernetes-secret-generator/pkg/apis"
	"github.com/mittwald/kubernetes-secret-generator/pkg/apis/secretgenerator/v1alpha1"
	"github.com/mittwald/kubernetes-secret-generator/pkg/controller/secret"
)

var mgr manager.Manager

const labelSecretGeneratorTest = "kubernetes-secret-generator-test"

func getSecretName() string {
	return uuid.New().String()
}

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

	// add custom resources to scheme
	err = apis.AddToScheme(scheme.Scheme)
	if err != nil {
		panic(err)
	}

	mgr, err = manager.New(cfg, mgrOpts)
	if err != nil {
		panic(err)
	}

	if err = apis.AddToScheme(mgr.GetScheme()); err != nil {
		panic(err)
	}

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
		err = mgr.GetClient().Delete(context.TODO(), &s)
		if err != nil {
			panic(err)
		}
	}

	sshList := &v1alpha1.SSHKeyPairList{}
	err = mgr.GetClient().List(context.TODO(),
		sshList,
		client.MatchingLabels(map[string]string{
			labelSecretGeneratorTest: "yes",
		}),
	)
	if err != nil {
		panic(err)
	}

	for _, s := range sshList.Items {
		err = mgr.GetClient().Delete(context.TODO(), &s)
		if err != nil {
			panic(err)
		}
	}

	stringSecretList := &v1alpha1.StringSecretList{}
	err = mgr.GetClient().List(context.TODO(),
		stringSecretList,
		client.MatchingLabels(map[string]string{
			labelSecretGeneratorTest: "yes",
		}),
	)
	if err != nil {
		panic(err)
	}

	for _, s := range stringSecretList.Items {
		err = mgr.GetClient().Delete(context.TODO(), &s)
		if err != nil {
			panic(err)
		}
	}

	basicAuthList := &v1alpha1.BasicAuthList{}
	err = mgr.GetClient().List(context.TODO(),
		basicAuthList,
		client.MatchingLabels(map[string]string{
			labelSecretGeneratorTest: "yes",
		}),
	)
	if err != nil {
		panic(err)
	}

	for _, s := range basicAuthList.Items {
		err = mgr.GetClient().Delete(context.TODO(), &s)
		if err != nil {
			panic(err)
		}
	}
}

func doReconcile(t *testing.T, targetSecret *corev1.Secret, isErr bool) {
	rec := secret.NewReconciler(mgr)
	req := reconcile.Request{NamespacedName: types.NamespacedName{Name: targetSecret.Name, Namespace: targetSecret.Namespace}}

	res, err := rec.Reconcile(req)

	if isErr {
		require.Error(t, err)
	} else {
		require.NoError(t, err)
	}
	require.False(t, res.Requeue)
}

func TestDoesNotTouchOtherSecrets(t *testing.T) {
	secret := &corev1.Secret{
		Type: corev1.SecretTypeOpaque,
		ObjectMeta: metav1.ObjectMeta{
			Name:      getSecretName(),
			Namespace: "default",
			Labels: map[string]string{
				labelSecretGeneratorTest: "yes",
			},
		},
		Data: map[string][]byte{
			"testkey":  []byte("test"),
			"testkey2": []byte("test2"),
		},
	}

	require.NoError(t, mgr.GetClient().Create(context.TODO(), secret))

	doReconcile(t, secret, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      secret.Name,
		Namespace: secret.Namespace}, out))

	if !reflect.DeepEqual(secret, out) {
		t.Errorf("secret without operator annotations has been reconciled")
	}
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
