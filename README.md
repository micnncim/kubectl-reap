# kubectl-reap

[![actions-workflow-test][actions-workflow-test-badge]][actions-workflow-test]
[![release][release-badge]][release]
[![pkg.go.dev][pkg.go.dev-badge]][pkg.go.dev]
[![license][license-badge]][license]

`kubectl-reap` is a kubectl plugin that deletes unused Kubernetes resources.

![screencast](/docs/assets/screencast.gif)

Supported resources:

- [x] Pods (whose status is not `Running`)
- [x] ConfigMaps (not used in any Pods)
- [x] Secrets (not used in any Pods or ServiceAccounts)
- [x] PersistentVolumes (not satisfying any PersistentVolumeClaims)
- [x] PersistentVolumeClaims (not used in any Pods)
- [x] Jobs (completed)
- [x] PodDisruptionBudgets (not targeting any Pods)
- [x] HorizontalPodAutoscalers (not targeting any resources)

Since this plugin supports dry-run as described below, it also helps you to find resources you misconfigured or forgot to delete.

## Installation

Download precompiled binaries from [GitHub Releases](https://github.com/micnncim/kubectl-reap/releases).

### Via [Krew](https://github.com/kubernetes-sigs/krew)

```
$ kubectl krew install reap
```

### Via Go

```
$ go get github.com/micnncim/kubectl-reap/cmd/kubectl-reap
```

## Examples

### Pods

In this example, this plugin deletes all Pods whose status is not `Running`.

```console
$ kubectl get po
NAME          READY   STATUS      RESTARTS   AGE
pod-running   1/1     Running     0          10s
pod-pending   0/1     Pending     0          20s
pod-failed    0/1     Failed      0          30s
pod-unknown   0/1     Unknown     0          40s
job-kqpxc     0/1     Completed   0          50s

$ kubectl reap po
pod/pod-pending deleted
pod/pod-failed deleted
pod/pod-unknown deleted
pod/job-kqpxc deleted
```

### ConfigMaps

In this example, this plugin deletes the unused ConfigMap `config-2`.

```console
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

$ kubectl reap cm
configmap/config-2 deleted
```

### Interactive Mode

You can choose which resource you will delete one by one by interactive mode.

```console
$ kubectl reap cm --interactive # or '-i'
? Are you sure to delete configmap/config-1? Yes
configmap/config-1 deleted
? Are you sure to delete configmap/config-2? No
? Are you sure to delete configmap/config-3? Yes
configmap/config-3 deleted
```

## Usage

```console
$ kubectl reap --help

Delete unused resources. Supported resources:

- Pods (whose status is not Running)
- ConfigMaps (not used in any Pods)
- Secrets (not used in any Pods or ServiceAccounts)
- PersistentVolumes (not satisfying any PersistentVolumeClaims)
- PersistentVolumeClaims (not used in any Pods)
- Jobs (completed)
- PodDisruptionBudgets (not targeting any Pods)
- HorizontalPodAutoscalers (not targeting any resources)

Usage:
  kubectl reap RESOURCE_TYPE [flags]

Examples:

  # Delete ConfigMaps not mounted on any Pods and in the current namespace and context
  $ kubectl reap configmaps

  # Delete unused ConfigMaps and Secrets in the namespace/my-namespace and context/my-context
  $ kubectl reap cm,secret -n my-namespace --context my-context

  # Delete ConfigMaps not mounted on any Pods and across all namespace
  $ kubectl reap cm --all-namespaces

  # Delete Pods whose status is not Running as client-side dry-run
  $ kubectl reap po --dry-run=client

Flags:
  -A, --all-namespaces                 If true, delete the targeted resources across all namespace except kube-system
      --allow-missing-template-keys    If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats. (default true)
      --as string                      Username to impersonate for the operation
      --as-group stringArray           Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --cache-dir string               Default cache directory (default "/Users/micnncim/.kube/cache")
      --certificate-authority string   Path to a cert file for the certificate authority
      --client-certificate string      Path to a client certificate file for TLS
      --client-key string              Path to a client key file for TLS
      --cluster string                 The name of the kubeconfig cluster to use
      --context string                 The name of the kubeconfig context to use
      --dry-run string[="unchanged"]   Must be "none", "server", or "client". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource. (default "none")
      --field-selector string          Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type.
      --force                          If true, immediately remove resources from API and bypass graceful deletion. Note that immediate deletion of some resources may result in inconsistency or data loss and requires confirmation.
      --grace-period int               Period of time in seconds given to the resource to terminate gracefully. Ignored if negative. Set to 1 for immediate shutdown. Can only be set to 0 when --force is true (force deletion). (default -1)
  -h, --help                           help for kubectl
      --insecure-skip-tls-verify       If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure
  -i, --interactive                    If true, a prompt asks whether resources can be deleted
      --kubeconfig string              Path to the kubeconfig file to use for CLI requests.
  -n, --namespace string               If present, the namespace scope for this CLI request
  -o, --output string                  Output format. One of: json|yaml|name|go-template|go-template-file|template|templatefile|jsonpath|jsonpath-as-json|jsonpath-file.
  -q, --quiet                          If true, no output is produced
      --request-timeout string         The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0")
  -l, --selector string                Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)
  -s, --server string                  The address and port of the Kubernetes API server
      --template string                Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].
      --timeout duration               The length of time to wait before giving up on a delete, zero means determine a timeout from the size of the object
      --tls-server-name string         Server name to use for server certificate validation. If it is not provided, the hostname used to contact the server is used
      --token string                   Bearer token for authentication to the API server
      --user string                    The name of the kubeconfig user to use
  -v, --version                        If true, show the version of this plugin
      --wait                           If true, wait for resources to be gone before returning. This waits for finalizers.

```

### Caveats

- It's recommended to run this plugin as dry-run (`--dry-run=client` or `--dry-run=server`) first or interactive mode (`--interactive`) in order to examine what resources will be deleted when running it, especially when you're trying to run it in a production environment.
- Even if you use `--namespace kube-system` or `--all-namespaces`, this plugin never deletes any resources in `kube-system` so that it prevents unexpected resource deletion.
- This plugin doesn't determine whether custom controllers or CRDs consume or depend on the supported resources. Make sure the resources you want to reap aren't used by them.
  - e.g.) A Secret, which isn't used in any Pods or ServiceAccounts but used in a custom controller, can be deleted

## Background

`kubectl apply --prune` allows us to delete unused resources.
However, it's not very flexible when we want to choose what kind resource to be deleted.
this plugin provides more flexible, easy way to delete resources.

## Similar Projects

- [dtan4/k8s-unused-secret-detector](https://github.com/dtan4/k8s-unused-secret-detector)
- [FikaWorks/kubectl-plugins/prune-unused](https://github.com/FikaWorks/kubectl-plugins/tree/master/prune-unused)
- [dirathea/kubectl-unused-volumes](https://github.com/dirathea/kubectl-unused-volumes)

<!-- badge links -->

[actions-workflow-test]: https://github.com/micnncim/kubectl-reap/actions?query=workflow%3ATest
[actions-workflow-test-badge]: https://img.shields.io/github/workflow/status/micnncim/kubectl-reap/Test?label=Test&style=for-the-badge&logo=github

[release]: https://github.com/micnncim/kubectl-reap/releases
[release-badge]: https://img.shields.io/github/v/release/micnncim/kubectl-reap?style=for-the-badge&logo=github

[pkg.go.dev]: https://pkg.go.dev/github.com/micnncim/kubectl-reap?tab=overview
[pkg.go.dev-badge]: http://bit.ly/pkg-go-dev-badge

[license]: LICENSE
[license-badge]: https://img.shields.io/github/license/micnncim/kubectl-reap?style=for-the-badge
