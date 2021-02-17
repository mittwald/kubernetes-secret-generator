# Automatically generated secrets for Kubernetes

This repository contains a custom Kubernetes controller that can automatically create
random secret values. This may be used for auto-generating random credentials for
applications run on Kubernetes.

## Security note

Older versions (>= 1.0.0) of this controller used the `math/rand` package for generating secrets, which is deterministic and not cryptographically secure (see #1 for more information). If you're already running this controller and want to regenerate all potentially compromised secrets, start the controller with the `-regenerate-insecure` flag (note that you will need to manually re-create any Pods using these secrets, though). When using the `kubectl apply` command from below, the new flag will be added to your Deployment automatically.

## Deployment

### Helm
The controller can be deployed using [Helm](https://helm.sh).

You might want to take a look a the [values.yaml](deploy/helm-chart/kubernetes-secret-generator/values.yaml) to adjust the operator to your needs:

- `secretLength` defines the length of the generated secret values.

- `watchNamespace` defines, which namespaces should be watched for secret objects.

  To watch a single namespace, set it to the desired namespace name.
Multiple namespaces are supported and can be set as a comma-separated list: `ns1,ns2`.

  If `watchNamespace` is set to the empty string value `""`, all namespaces will be watched.

Afterwards, deploy the operator using:

1. Add the [Mittwald Charts Repo](https://github.com/mittwald/helm-charts/blob/master/README.md#usage):
    ```shellsession
    $ helm repo add mittwald https://helm.mittwald.de
    "mittwald" has been added to your repositories

    $ helm repo update
    Hang tight while we grab the latest from your chart repositories...
    ...Successfully got an update from the "mittwald" chart repository
    Update Complete. ⎈ Happy Helming!⎈
    ```

2. Upgrade or install `kubernetes-secret-generator`:

    ```shellsession
    $ helm upgrade --install kubernetes-secret-generator mittwald/kubernetes-secret-generator
    ```
 
### Manually

If you don't want to use Helm (why wouldn't you?), the required .yaml files can also be applied manually using `kubectl apply`:

```shellsession
$ make install
```

To uninstall, use:

```shellsession
$ make uninstall
```

## Usage

This operator is capable of generating secure random strings and ssh keypair secrets. 

It supports two ways of secret generation, annotation-based and cr-based.

### Annotation-based generation

For annotation based generation, the type of secret to be generated can be specified by the `secret-generator.v1.mittwald.de/type` annotation.
This annotation can be added to any Kubernetes secret object in the operators `watchNamespace`.

The encoding of the secret can be specified by the `secret-generator.v1.mittwald.de/encoding` annotation.
Available encodings are `base64`, `base64url`, `base32`, `hex` and `raw`, with `raw` returning the unencoded byte sequence
that was generated. `base64` will be used, if the annotation was not used.

The length of the generated secret can be specified by the `secret-generator.v1.mittwald.de/length` annotation.
By default, this length refers to the length of the generated string, and not the length of the byte sequence encoded by it. 
The suffix `B` or `b` can be used to indicate that the provided value should refer to the encoded byte sequence instead.

### Secure Random Strings

By default, the operator will generate secure random strings. If the type annotation is not present, it will be added after the first
reconciliation loop and its value will be set to `string`.

To actually generate random string secrets, the `secret-generator.v1.mittwald.de/autogenerate` annotation is required as well.
The value of the annotation can be a field name (or comma separated list of field names) within the secret;
the SecretGeneratorController will pick up this annotation and add a field [or fields] 
(`password` in the example below) to the secret with a randomly generated string value.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: string-secret
  annotations:
    secret-generator.v1.mittwald.de/autogenerate: password
data:
  username: c29tZXVzZXI=
```

after reconciliation:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: string-secret
  annotations:
    secret-generator.v1.mittwald.de/type: stringsecret
    secret-generator.v1.mittwald.de/secure: "yes"
    secret-generator.v1.mittwald.de/autogenerate: password
    secret-generator.v1.mittwald.de/autogenerate-generated-at: "2020-04-03T14:07:47+02:00"
type: Opaque
data:
  username: c29tZXVzZXI=
  password: TWVwSU83L2huNXBralNTMHFwU3VKSkkwNmN4NmRpNTBBcVpuVDlLOQ==
```

### SSH Key Pairs

To generate SSH Key Pairs, the `secret-generator.v1.mittwald.de/type` annotation **has** to be present on the kubernetes secret object.

The operator will then add two keys to the secret object, `ssh-publickey` and `ssh-privatekey`, each containing the respective key.

The Private Key will be PEM encoded, the Public Key will have the authorized-keys format.

```yaml
apiVersion: v1
kind: Secret
metadata:
  annotations:
    secret-generator.v1.mittwald.de/type: ssh-keypair
data: {}
```

after reconciliation:

```yaml
apiVersion: v1
kind: Secret
metadata:
  annotations:
    secret-generator.v1.mittwald.de/type: ssh-keypair
    secret-generator.v1.mittwald.de/autogenerate-generated-at: "2020-04-03T14:07:47+02:00"
type: Opaque
data:
  ssh-publickey: c3NoLXJzYSBBQUFBQ...
  ssh-privatekey: LS0tLS1CRUdJTi...
```

### Ingress Basic Auth

To generate Ingress Basic Auth credentials, the `secret-generator.v1.mittwald.de/type` annotation **has** to be present on the kubernetes secret object.

The operator will then add three keys to the secret object.
The ingress will interpret the `auth` key as a htpasswd entry. This entry contains the username, and the hashed generated password for the user.
The operator also stores the username and cleartext password in the `username` and `password` keys.

If a username other than `admin` is desired, it can be specified using the `secret-generator.v1.mittwald.de/basic-auth-username` annotation.

```yaml
apiVersion: v1
kind: Secret
metadata:
  annotations:
    secret-generator.v1.mittwald.de/type: basic-auth
data: {}
```

after reconciliation:

```yaml
apiVersion: v1
kind: Secret
metadata:
  annotations:
    secret-generator.v1.mittwald.de/type: basic-auth
    secret-generator.v1.mittwald.de/autogenerate-generated-at: "2020-04-03T14:07:47+02:00"
type: Opaque
data:
  username: admin
  password: test123
  auth: "admin:PASSWORD_HASH"
```

### CR-based generation

The operator supports three kinds of custom resources: `StringSecret`, `SSHKeyPair` and `BasicAuth`. These crs can be used to trigger creation, update and deletion of desired secrets.
All crs support the field `spec.type` which can be used to define the kubernetes type of the generated `Secret`, e.g. "Opaque"

### Secure random strings

A `StringSecret` resource can be used to generate secure random strings similar to the ones offered by the annotation approach.
Desired Fields to be randomly generated can be supplied via the `spec.fieldNames` property, which supports a list of strings.
Secret length and encoding can be specified using `spec.length` and `spec.encoding` properties and have the same options as annotation based generation.
The `spec.data` property can be used to specify arbitrary data entries the generated secret's `data` property should be populated with.
Finally, the `spec.forceRegenerate` property can be used to control regeneration of secret fields.

Example:

```yaml
apiVersion: "mittwald.systems/v1alpha1"
kind: "StringSecret"
metadata:
  name: "example-pw"
  namespace: "default"
spec:
  forceRegenerate: false
  length: "40"
  encoding: "base64"
  fieldNames: 
  - "password"
  data:
    username: "testuser"
```

Upon creation of the cr, the controller will attempt to create a `Secret` resource matching the specifications. If successful, the new resource will have its owner set as `StringSecret` used to create it, providing automated deletion/updating of the secret if the creating cr is deleted/updated. The `StringSecret` will store an object reference to the created `Secret` in its status field.
During updating, any new fields in `spec.data` or `spec.fieldnames` will be added, while existing fields will only be overwritten/regenerated, if `spec.forceRegenerate` is set to true. 
If the target `Secret` already exists and is not owned by a `StringSecret` resource, no changes will be made to ìt.

### SSH Key Pair

A `SSHKeyPair` resource can be used to generate an ssh key pair. It supports `spec.length`, `spec.data` and `spec.forceRegenerate` similar to `StringSecret` resources.
The field `spec.privateKey` can be used to specify a private key, which will be used during runtime to regenerate a matching public key.
Updating is handled similar to `StringSecret` resources, unowned `Secrets` are not modified, and existing fields are only updated if regeneration is forced. However, should the public key be missing, the operator will attempt to regenerate it.

```yaml
apiVersion: "mittwald.systems/v1alpha1"
kind: "SSHKeyPair"
metadata:
  name: "example-ssh"
  namespace: "default"
spec:
  length: "40"
  forceRegenerate: false
  data:
    example: "data"
```

### Ingress Basic Auth

A `BasicAuth` resource can be used to generate Ingress Basic Auth credentials. Supported properties are `spec.length`, `spec.encoding`, `spec.data` and `spec.forceRegenerate`.
To specify a username, use `spec.username`. If no username is provided, the operator will use `admin`.
Updates follow the same rules as for the other crs, existing `secrets` will only be updated if owned by a `BasicAuth` resource and if `spec.forceRegenerate` is set to true. The exception to this are new `spec.data` entries, which are added even if `forceRegenerate` is false, and cases where the `auth` field in the `Secret` is empty.

```yaml
apiVersion: "mittwald.systems/v1alpha1"
kind: "BasicAuth"
metadata:
  name: "example-auth"
  namespace: "default"
spec:
  length: "40"
  username: "testuser"
  encoding: "base64"
  foreRegenerate: false
  data:
    example: "data"
```

## Operational tasks

-   Regenerate all automatically generated secrets:
    ```
    $ kubectl annotate secrets --all secret-generator.v1.mittwald.de/regenerate=true
    ```
    
-   Regenerate only certain fields, in case the secret is of the `password` type:
    ```
    $ kubectl annotate secrets --all secret-generator.v1.mittwald.de/regenerate=password1,password2
    ```
