package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"github.com/mittwald/kubernetes-secret-generator/pkg/apis/types/v1alpha1"
)

type BasicAuthInterface interface {
	List(opts metav1.ListOptions) (*v1alpha1.BasicAuthList, error)
	Get(name string, options metav1.GetOptions) (*v1alpha1.BasicAuth, error)
	Create(auth *v1alpha1.BasicAuth) (*v1alpha1.BasicAuth, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
}

type basicAuthClient struct {
	restClient rest.Interface
	ns         string
}

func (c *basicAuthClient) List(opts metav1.ListOptions) (*v1alpha1.BasicAuthList, error) {
	result := v1alpha1.BasicAuthList{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource("basicauth").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(&result)

	return &result, err
}

func (c *basicAuthClient) Get(name string, opts metav1.GetOptions) (*v1alpha1.BasicAuth, error) {
	result := v1alpha1.BasicAuth{}
	err := c.restClient.Get().Namespace(c.ns).Resource("basicauth").Name(name).VersionedParams(&opts, scheme.ParameterCodec).Do().Into(&result)
	return &result, err
}

func (c *basicAuthClient) Create(project *v1alpha1.String) (*v1alpha1.String, error) {
	result := v1alpha1.String{}
	err := c.restClient.Post().Namespace(c.ns).Resource("basicauth").Body(project).Do().Into(&result)
	return &result, err
}

func (c *basicAuthClient) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.restClient.Get().Namespace(c.ns).Resource("basicauth").VersionedParams(&opts, scheme.ParameterCodec).Watch()
}
