package v1alpha1

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto copies all properties of this object into another object of the
// same type that is provided as a pointer.
func (in *String) DeepCopyInto(out *String) {
	fmt.Println(in.Spec.Encoding)
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = in.ObjectMeta
	out.Spec = StringSpec{
		Length:     in.Spec.Length,
		Encoding:   in.Spec.Encoding,
		FieldNames: in.Spec.FieldNames,
		Data:       in.Spec.Data,
	}
}

// DeepCopyObject returns a generically typed copy of an object
func (in *String) DeepCopyObject() runtime.Object {
	out := String{}
	in.DeepCopyInto(&out)

	return &out
}

// DeepCopyObject returns a generically typed copy of an object
func (in *StringList) DeepCopyObject() runtime.Object {
	out := StringList{}
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta

	if in.Items != nil {
		out.Items = make([]String, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}

	return &out
}

// DeepCopyInto copies all properties of this object into another object of the
// same type that is provided as a pointer.
func (in *BasicAuth) DeepCopyInto(out *BasicAuth) {
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = in.ObjectMeta
	out.Spec = BasicAuthSpec{
		Length:   in.Spec.Length,
		Username: in.Spec.Username,
		Encoding: in.Spec.Encoding,
	}
}

// DeepCopyObject returns a generically typed copy of an object
func (in *BasicAuth) DeepCopyObject() runtime.Object {
	out := BasicAuth{}
	in.DeepCopyInto(&out)

	return &out
}

// DeepCopyObject returns a generically typed copy of an object
func (in *BasicAuthList) DeepCopyObject() runtime.Object {
	out := BasicAuthList{}
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta

	if in.Items != nil {
		out.Items = make([]BasicAuth, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}

	return &out
}

// DeepCopyInto copies all properties of this object into another object of the
// same type that is provided as a pointer.
func (in *SSHKeyPair) DeepCopyInto(out *SSHKeyPair) {
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = in.ObjectMeta
	out.Spec = SSHKeyPairSpec{
		Length: in.Spec.Length,
	}
}

// DeepCopyObject returns a generically typed copy of an object
func (in *SSHKeyPair) DeepCopyObject() runtime.Object {
	out := SSHKeyPair{}
	in.DeepCopyInto(&out)

	return &out
}

// DeepCopyObject returns a generically typed copy of an object
func (in *SSHKeyPairList) DeepCopyObject() runtime.Object {
	out := SSHKeyPairList{}
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta

	if in.Items != nil {
		out.Items = make([]SSHKeyPair, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}

	return &out
}
