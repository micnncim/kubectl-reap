# kubectl-prune

[![actions-workflow-test][actions-workflow-test-badge]][actions-workflow-test]

`kubectl-prune` prunes unused ConfigMap, Secret, Pod, and ReplicaSet resources.

## Usage

```
# Delete ConfigMaps not mounted on any Pods and in the namespace/app
$ kubectl prune configmaps -n app

# Delete Secrets not mounted on any Pods and across all namespace
$ kubectl prune secret --all-namespaces

# Run the command as client-side dry-run
$ kubectl prune cm --dry-run=client

# Delete Pods not managed by any ReplicaSets and ReplicaSets not managed by any Deployments
$ kubectl prune po,rs`
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
