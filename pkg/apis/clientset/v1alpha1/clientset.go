package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"github.com/mittwald/kubernetes-secret-generator/pkg/apis/types/v1alpha1"
)

type StringV1Alpha1Interface interface {
	Strings(namesapce string) StringInterface
}

type StringV1Alpha1Client struct {
	restClient rest.Interface
}

func NewForConfig(c *rest.Config) (*StringV1Alpha1Client, error) {
	config := *c
	config.ContentConfig.GroupVersion = &schema.GroupVersion{Group: v1alpha1.GroupName, Version: v1alpha1.GroupVersion}
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	return &StringV1Alpha1Client{restClient: client}, nil
}

func (c *StringV1Alpha1Client) Projects(namespace string) StringInterface {
	return &stringClient{
		restClient: c.restClient,
		ns:         namespace,
	}
}

func (c *StringV1Alpha1Client) Strings(namespace string) StringInterface {
	return &stringClient{
		restClient: c.restClient,
		ns:         namespace,
	}
}
