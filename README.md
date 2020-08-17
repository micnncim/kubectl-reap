# kubectl-prune

[![actions-workflow-test][actions-workflow-test-badge]][actions-workflow-test]

`kubectl-prune` prunes unused ConfigMap, Secret, Pod, and ReplicaSet resources.

## Usage

```
# Delete ConfigMaps not mounted on any Pods and in the current namespace and context
$ kubectl prune configmaps

# Delete ConfigMaps not mounted on any Pods and in the namespace/my-namespace and context/my-context
$ kubectl prune cm -n my-namespace --context my-context

# Delete Secrets not mounted on any Pods and across all namespace
$ kubectl prune secret --all-namespaces

# Delete Pods not managed by any ReplicaSets and ReplicaSets not managed by any Deployments
$ kubectl prune po,rs
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
