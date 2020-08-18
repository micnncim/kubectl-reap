# kubectl-prune

[![actions-workflow-test][actions-workflow-test-badge]][actions-workflow-test]

`kubectl-prune` prunes unused ConfigMap, Secret, Pod, and ReplicaSet resources.

## Installation

```
$ GO111MODULE=on go get github.com/micnncim/kubectl-prune/cmd/kubectl-prune
```

## Examples

In this example, `kubectl-prune` deletes the unused ConfigMap `config-2`.

```
$ kubectl get cm
NAME       DATA   AGE
config-1   1      0m15s
config-2   1      0m10s

$ cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  containers:
  - name: nginx
    image: nginx
    volumeMounts:
    - name: config-1-volume
      mountPath: /var/config
  volumes:
  - name: config-1-volume
    configMap:
      name: config-1
EOF

$ kubectl get po
NAME                     READY   STATUS              RESTARTS   AGE
nginx                    1/1     Running             0          0m40s

$ kubectl prune cm --dry-run=client
configmap/config-2 deleted (dry run)

$ kubectl get cm
NAME       DATA   AGE
config-1   1      1m15s
config-2   1      1m10s

$ kubectl prune cm
configmap/config-2 deleted

$ kubectl get cm
NAME       DATA   AGE
config-1   1      1m30s
```

## Usage

```console
$ kubectl prune --help
Usage:
  kubectl prune TYPE [flags]

Examples:

  # Delete ConfigMaps not mounted on any Pods and in the current namespace and context
  $ kubectl prune configmaps

  # Delete Secrets not mounted on any Pods and in the namespace/my-namespace and context/my-context
  $ kubectl prune secret -n my-namespace --context my-context

  # Delete ConfigMaps not mounted on any Pods and across all namespace
  $ kubectl prune cm --all-namespaces

  # Delete Pods not managed by any ReplicaSets and ReplicaSets not managed by any Deployments
  $ kubectl prune po,rs

Flags:
  -A, --all-namespaces                 If true, prune the targeted resources accross all namespace
      --allow-missing-template-keys    If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats. (default true)
      --as string                      Username to impersonate for the operation
      --as-group stringArray           Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --cache-dir string               Default HTTP cache directory (default "/Users/micnncim/.kube/http-cache")
      --certificate-authority string   Path to a cert file for the certificate authority
      --client-certificate string      Path to a client certificate file for TLS
      --client-key string              Path to a client key file for TLS
      --cluster string                 The name of the kubeconfig cluster to use
      --context string                 The name of the kubeconfig context to use
      --dry-run string[="unchanged"]   Must be "none", "server", or "client". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource. (default "none")
  -h, --help                           help for kubectl
      --insecure-skip-tls-verify       If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure
      --kubeconfig string              Path to the kubeconfig file to use for CLI requests.
  -n, --namespace string               If present, the namespace scope for this CLI request
  -o, --output string                  Output format. One of: json|yaml|name|go-template|go-template-file|template|templatefile|jsonpath|jsonpath-file.
  -q, --quiet                          If true, no output is produced
      --request-timeout string         The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0")
  -s, --server string                  The address and port of the Kubernetes API server
      --template string                Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].
      --tls-server-name string         Server name to use for server certificate validation. If it is not provided, the hostname used to contact the server is used
      --token string                   Bearer token for authentication to the API server
      --user string                    The name of the kubeconfig user to use

```

## Background

`kubectl apply --prune` allows us prune unused resources.
However, it's not very flexible when we want to choose what kind resource to be deleted.
`kubectl-prune` provides more flexible, easy way to prune resources.

## Related Projects

- [dtan4/k8s-unused-secret-detector](https://github.com/dtan4/k8s-unused-secret-detector)

<!-- badge links -->

[actions-workflow-test]: https://github.com/micnncim/kubectl-prune/actions?query=workflow%3ATest
[actions-workflow-test-badge]: https://img.shields.io/github/workflow/status/micnncim/kubectl-prune/Test?label=Test&style=for-the-badge&logo=github
