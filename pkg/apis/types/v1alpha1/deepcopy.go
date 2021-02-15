package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto copies all properties of this object into another object of the
// same type that is provided as a pointer.
func (in *StringSecret) DeepCopyInto(out *StringSecret) {
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = in.ObjectMeta
	out.Spec = StringSecretSpec{
		Length:          in.Spec.Length,
		Encoding:        in.Spec.Encoding,
		FieldNames:      in.Spec.FieldNames,
		Data:            in.Spec.Data,
		Type:            in.Spec.Type,
		ForceRegenerate: in.Spec.ForceRegenerate,
	}
	out.Status = SecretStatus{
		Secret: in.Status.Secret,
	}
}

// DeepCopyObject returns a generically typed copy of an object
func (in *StringSecret) DeepCopyObject() runtime.Object {
	out := StringSecret{}
	in.DeepCopyInto(&out)

	return &out
}

// DeepCopyObject returns a generically typed copy of an object
func (in *StringSecretList) DeepCopyObject() runtime.Object {
	out := StringSecretList{}
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta

	if in.Items != nil {
		out.Items = make([]StringSecret, len(in.Items))
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
		Length:          in.Spec.Length,
		Username:        in.Spec.Username,
		Encoding:        in.Spec.Encoding,
		Type:            in.Spec.Type,
		ForceRegenerate: in.Spec.ForceRegenerate,
		Data:            in.Spec.Data,
	}
	out.Status = SecretStatus{
		Secret: in.Status.Secret,
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
		Length:          in.Spec.Length,
		Type:            in.Spec.Type,
		ForceRegenerate: in.Spec.ForceRegenerate,
		PrivateKey:      in.Spec.PrivateKey,
		Data:            in.Spec.Data,
	}
	out.Status = SecretStatus{
		Secret: in.Status.Secret,
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
