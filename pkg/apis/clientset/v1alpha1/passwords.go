package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"github.com/mittwald/kubernetes-secret-generator/pkg/apis/types/v1alpha1"
)

type StringInterface interface {
	List(opts metav1.ListOptions) (*v1alpha1.StringList, error)
	Get(name string, options metav1.GetOptions) (*v1alpha1.String, error)
	Create(*v1alpha1.String) (*v1alpha1.String, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
}

type stringClient struct {
	restClient rest.Interface
	ns         string
}

func (c *stringClient) List(opts metav1.ListOptions) (*v1alpha1.StringList, error) {
	result := v1alpha1.StringList{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource("strings").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(&result)

	return &result, err
}

func (c *stringClient) Get(name string, opts metav1.GetOptions) (*v1alpha1.String, error) {
	result := v1alpha1.String{}
	err := c.restClient.Get().Namespace(c.ns).Resource("strings").Name(name).VersionedParams(&opts, scheme.ParameterCodec).Do().Into(&result)
	return &result, err
}

func (c *stringClient) Create(project *v1alpha1.String) (*v1alpha1.String, error) {
	result := v1alpha1.String{}
	err := c.restClient.Post().Namespace(c.ns).Resource("strings").Body(project).Do().Into(&result)
	return &result, err
}

func (c *stringClient) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.restClient.Get().Namespace(c.ns).Resource("string").VersionedParams(&opts, scheme.ParameterCodec).Watch()
}
