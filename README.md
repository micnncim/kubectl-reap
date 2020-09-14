# kubectl-prune

[![actions-workflow-test][actions-workflow-test-badge]][actions-workflow-test]
[![release][release-badge]][release]
[![pkg.go.dev][pkg.go.dev-badge]][pkg.go.dev]
[![license][license-badge]][license]

`kubectl-prune` is a kubectl plugin that prunes unused Kubernetes resources.

Supported resources:

- [x] ConfigMaps (not used in any Pods)
- [x] Secrets (not used in any Pods or ServiceAccounts)
- [x] Pods (whose status is not `Running`)
- [ ] PodDisruptionBudgets
- [ ] HorizontalPodAutoscalers

## Installation

```
$ GO111MODULE=on go get github.com/micnncim/kubectl-prune/cmd/kubectl-prune
```

## Examples

### Pods

In this example, `kubectl-prune` deletes all Pods whose status is not `Running`.

```console
$ kubectl get po
NAME                     READY   STATUS      RESTARTS   AGE
nginx-54565674c6-fmw7g   1/1     Running     0          10s
nginx-54565674c6-t8hnm   0/1     Pending     0          20s
nginx-54565674c6-v7xw9   0/1     Failed      0          30s
nginx-54565674c6-wwb6m   0/1     Unknown     0          40s
job-kqpxc                0/1     Completed   0          50s

$ kubectl prune po
pod/nginx-54565674c6-t8hnm deleted
pod/nginx-54565674c6-v7xw9 deleted
pod/nginx-54565674c6-wwb6m deleted
pod/job-kqpxc deleted

$ kubectl get po
NAME                     READY   STATUS      RESTARTS   AGE
nginx-54565674c6-fmw7g   1/1     Running     0          20s
```

### ConfigMaps

In this example, `kubectl-prune` deletes the unused ConfigMap `config-2`.

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

**It's recommended to run `kubectl-prune` as dry-run (client or server) first before actually running it to examine what resources will be deleted, especially if you want to run it in a production environment.**

## Usage

```console
$ kubectl prune --help
Usage:
  kubectl prune TYPE [flags]

Examples:

  # Delete ConfigMaps not mounted on any Pods and in the current namespace and context
  $ kubectl prune configmaps

  # Delete unused ConfigMaps and Secrets in the namespace/my-namespace and context/my-context
  $ kubectl prune cm,secret -n my-namespace --context my-context

  # Delete ConfigMaps not mounted on any Pods and across all namespace
  $ kubectl prune cm --all-namespaces

  # Delete Pods whose status is not Running as client-side dry-run
  $ kubectl prune po --dry-run=client

Flags:
  -A, --all-namespaces                 If true, prune the targeted resources across all namespace except kube-system
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

Note: When you use `--all-namespaces`, `kubectl-prune` prunes resources across all namespace except `kube-system` so that it prevents unexpected resource deletion.

## Background

`kubectl apply --prune` allows us to prune unused resources.
However, it's not very flexible when we want to choose what kind resource to be deleted.
`kubectl-prune` provides more flexible, easy way to prune resources.

## Similar Projects

This project is inspired by `dtan4/k8s-unused-secret-detector` :cherry_blossom:

- [dtan4/k8s-unused-secret-detector](https://github.com/dtan4/k8s-unused-secret-detector)
- [FikaWorks/kubectl-plugins/prune-unused](https://github.com/FikaWorks/kubectl-plugins/tree/master/prune-unused)
- [dirathea/kubectl-unused-volumes](https://github.com/dirathea/kubectl-unused-volumes)

<!-- badge links -->

[actions-workflow-test]: https://github.com/micnncim/kubectl-prune/actions?query=workflow%3ATest
[actions-workflow-test-badge]: https://img.shields.io/github/workflow/status/micnncim/kubectl-prune/Test?label=Test&style=for-the-badge&logo=github

[release]: https://github.com/micnncim/kubectl-prune/releases
[release-badge]: https://img.shields.io/github/v/release/micnncim/kubectl-prune?style=for-the-badge&logo=github

[pkg.go.dev]: https://pkg.go.dev/github.com/micnncim/kubectl-prune?tab=overview
[pkg.go.dev-badge]: https://img.shields.io/badge/pkg.go.dev-reference-02ABD7?style=for-the-badge&logoWidth=25&logo=data%3Aimage%2Fsvg%2Bxml%3Bbase64%2CPHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9Ijg1IDU1IDEyMCAxMjAiPjxwYXRoIGZpbGw9IiMwMEFERDgiIGQ9Ik00MC4yIDEwMS4xYy0uNCAwLS41LS4yLS4zLS41bDIuMS0yLjdjLjItLjMuNy0uNSAxLjEtLjVoMzUuN2MuNCAwIC41LjMuMy42bC0xLjcgMi42Yy0uMi4zLS43LjYtMSAuNmwtMzYuMi0uMXptLTE1LjEgOS4yYy0uNCAwLS41LS4yLS4zLS41bDIuMS0yLjdjLjItLjMuNy0uNSAxLjEtLjVoNDUuNmMuNCAwIC42LjMuNS42bC0uOCAyLjRjLS4xLjQtLjUuNi0uOS42bC00Ny4zLjF6bTI0LjIgOS4yYy0uNCAwLS41LS4zLS4zLS42bDEuNC0yLjVjLjItLjMuNi0uNiAxLS42aDIwYy40IDAgLjYuMy42LjdsLS4yIDIuNGMwIC40LS40LjctLjcuN2wtMjEuOC0uMXptMTAzLjgtMjAuMmMtNi4zIDEuNi0xMC42IDIuOC0xNi44IDQuNC0xLjUuNC0xLjYuNS0yLjktMS0xLjUtMS43LTIuNi0yLjgtNC43LTMuOC02LjMtMy4xLTEyLjQtMi4yLTE4LjEgMS41LTYuOCA0LjQtMTAuMyAxMC45LTEwLjIgMTkgLjEgOCA1LjYgMTQuNiAxMy41IDE1LjcgNi44LjkgMTIuNS0xLjUgMTctNi42LjktMS4xIDEuNy0yLjMgMi43LTMuN2gtMTkuM2MtMi4xIDAtMi42LTEuMy0xLjktMyAxLjMtMy4xIDMuNy04LjMgNS4xLTEwLjkuMy0uNiAxLTEuNiAyLjUtMS42aDM2LjRjLS4yIDIuNy0uMiA1LjQtLjYgOC4xLTEuMSA3LjItMy44IDEzLjgtOC4yIDE5LjYtNy4yIDkuNS0xNi42IDE1LjQtMjguNSAxNy05LjggMS4zLTE4LjktLjYtMjYuOS02LjYtNy40LTUuNi0xMS42LTEzLTEyLjctMjIuMi0xLjMtMTAuOSAxLjktMjAuNyA4LjUtMjkuMyA3LjEtOS4zIDE2LjUtMTUuMiAyOC0xNy4zIDkuNC0xLjcgMTguNC0uNiAyNi41IDQuOSA1LjMgMy41IDkuMSA4LjMgMTEuNiAxNC4xLjYuOS4yIDEuNC0xIDEuN3oiLz48cGF0aCBmaWxsPSIjMDBBREQ4IiBkPSJNMTg2LjIgMTU0LjZjLTkuMS0uMi0xNy40LTIuOC0yNC40LTguOC01LjktNS4xLTkuNi0xMS42LTEwLjgtMTkuMy0xLjgtMTEuMyAxLjMtMjEuMyA4LjEtMzAuMiA3LjMtOS42IDE2LjEtMTQuNiAyOC0xNi43IDEwLjItMS44IDE5LjgtLjggMjguNSA1LjEgNy45IDUuNCAxMi44IDEyLjcgMTQuMSAyMi4zIDEuNyAxMy41LTIuMiAyNC41LTExLjUgMzMuOS02LjYgNi43LTE0LjcgMTAuOS0yNCAxMi44LTIuNy41LTUuNC42LTggLjl6bTIzLjgtNDAuNGMtLjEtMS4zLS4xLTIuMy0uMy0zLjMtMS44LTkuOS0xMC45LTE1LjUtMjAuNC0xMy4zLTkuMyAyLjEtMTUuMyA4LTE3LjUgMTcuNC0xLjggNy44IDIgMTUuNyA5LjIgMTguOSA1LjUgMi40IDExIDIuMSAxNi4zLS42IDcuOS00LjEgMTIuMi0xMC41IDEyLjctMTkuMXoiLz48L3N2Zz4=

[license]: LICENSE
[license-badge]: https://img.shields.io/github/license/micnncim/kubectl-prune?style=for-the-badge
