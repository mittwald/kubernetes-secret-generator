# Automatically generated secrets for Kubernetes

This repository contains a custom Kubernetes controller that can automatically create
random secret values. This may be used for auto-generating random credentials for
applications run on Kubernetes.

## Security note

Older versions (>= 1.0.0) of this controller used the `math/rand` package for generating secrets, which is deterministic and not cryptographically secure (see #1 for more information). If you're already running this controller and want to regenerate all potentially compromised secrets, start the controller with the `-regenerate-insecure` flag (note that you will need to manually re-create any Pods using these secrets, though). When using the `kubectl apply` command from below, the new flag will be added to your Deployment automatically.

## Deployment

### Helm
The controller can be deployed using [Helm](https://helm.sh).

You might want to take a look a the [values.yaml](deploy/helm-chart/kubernetes-secret-generator/values.yaml) to adjust the operator to your needs.

`secretLength` defines the length of the generated secret values.

`watchNamespace` defines, which namespaces should be watched for secret objects.

To watch a single namespace, set it to the desired namespace name.
Multiple namespaces are supported and can be set as a comma-separated list: `ns1,ns2`.

If `watchNamespace` is set to the empty string value `""`, all namespaces will be watched.

Afterwards, deploy the operator using:

1. [Add the Mittwald-Charts Repo](https://github.com/mittwald/helm-charts/blob/master/README.md#usage):
    ```shellsession
    $ helm repo add mittwald https://helm.mittwald.de
    "mittwald" has been added to your repositories

    $ helm repo update
    Hang tight while we grab the latest from your chart repositories...
    ...Successfully got an update from the "mittwald" chart repository
    Update Complete. ⎈ Happy Helming!⎈
    ```

2. Upgrade or install `kubernetes-secret-generator`:  
  `helm upgrade --install kubernetes-secret-generator mittwald/kubernetes-secret-generator`
 
### Manually
If you don't want to use helm, the required .yaml files can also be applied manually using kubectl apply:
 ```shellsession
$ make install
```

To uninstall, use:
```shellsession
$ make uninstall
```

## Usage

Add the annotation `secret-generator.v1.mittwald.de/autogenerate` to any Kubernetes
secret object. The value of the annotation can be a field name 
(or comma separated list of field names) within the secret; the
SecretGeneratorController will pick up this annotation and add a field [or fields] 
(`password` in the example below) to the secret with a randomly generated string value.

```yaml
apiVersion: v1
kind: Secret
metadata:
  annotations:
    secret-generator.v1.mittwald.de/autogenerate: password
data:
  username: c29tZXVzZXI=
```

## Operational tasks

-   Regenerate all automatically generated passwords:

    ```
    $ kubectl annotate secrets --all secret-generator.v1.mittwald.de/regenerate=true
    ```
    
-   Regenerate only certain fields
    ```
    $ kubectl annotate secrets --all secret-generator.v1.mittwald.de/regenerate=password1,password2
    ```
