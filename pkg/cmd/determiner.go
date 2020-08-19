package cmd

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes"
)

const (
	kindConfigMap  = "ConfigMap"
	kindSecret     = "Secret"
	kindPod        = "Pod"
	kindReplicaSet = "ReplicaSet"
)

// determiner determines whether a resource should be pruned.
type determiner struct {
	// key=ConfigMap.Name
	usedConfigMaps map[string]struct{}
	// key=Secret.Name
	usedSecrets map[string]struct{}
	// key=ReplicaSet.Name
	replicaSets map[string]struct{}
	// key=Deployment.Name
	deployments map[string]struct{}
}

func newDeterminer(clientset *kubernetes.Clientset, r *resource.Result, namespace string) (*determiner, error) {
	var (
		pruneConfigMaps  bool
		pruneSecrets     bool
		prunePods        bool
		pruneReplicaSets bool
	)

	if err := r.Visit(func(info *resource.Info, err error) error {
		switch info.Object.GetObjectKind().GroupVersionKind().Kind {
		case kindConfigMap:
			pruneConfigMaps = true
		case kindSecret:
			pruneSecrets = true
		case kindPod:
			prunePods = true
		case kindReplicaSet:
			pruneReplicaSets = true
		}
		return nil
	}); err != nil {
		return nil, err
	}

	var (
		usedConfigMaps = make(map[string]struct{})
		usedSecrets    = make(map[string]struct{})
		replicaSets    = make(map[string]struct{})
		deployments    = make(map[string]struct{})
	)

	ctx := context.Background()

	if pruneConfigMaps || pruneSecrets {
		pods, err := listPods(ctx, clientset, namespace)
		if err != nil {
			return nil, err
		}
		if pruneConfigMaps {
			usedConfigMaps = detectUsedConfigMaps(pods)
		}
		if pruneSecrets {
			sas, err := listServiceAccounts(ctx, clientset, namespace)
			if err != nil {
				return nil, err
			}
			usedSecrets = detectUsedSecrets(pods, sas)
		}
	}

	if prunePods {
		resp, err := listReplicaSets(ctx, clientset, namespace)
		if err != nil {
			return nil, err
		}
		for _, v := range resp {
			replicaSets[v.Name] = struct{}{}
		}
	}

	if pruneReplicaSets {
		resp, err := listDeployments(ctx, clientset, namespace)
		if err != nil {
			return nil, err
		}
		for _, v := range resp {
			deployments[v.Name] = struct{}{}
		}
	}

	return &determiner{
		usedConfigMaps: usedConfigMaps,
		usedSecrets:    usedSecrets,
		replicaSets:    replicaSets,
		deployments:    deployments,
	}, nil
}

// determinePrune determines whether a resource should be pruned.
func (d *determiner) determinePrune(info *resource.Info) (bool, error) {
	switch kind := info.Object.GetObjectKind().GroupVersionKind().Kind; kind {
	case kindConfigMap:
		if _, ok := d.usedConfigMaps[info.Name]; ok {
			return false, nil
		}

	case kindSecret:
		if _, ok := d.usedSecrets[info.Name]; ok {
			return false, nil
		}

	case kindPod:
		unstructured := info.Object.(runtime.Unstructured).UnstructuredContent()
		var pod corev1.Pod
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured, &pod); err != nil {
			return false, err
		}
		for _, ownerRef := range pod.OwnerReferences {
			if _, ok := d.replicaSets[ownerRef.Name]; ok {
				return false, nil
			}
		}

	case kindReplicaSet:
		unstructured := info.Object.(runtime.Unstructured).UnstructuredContent()
		var rs appsv1.ReplicaSet
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured, &rs); err != nil {
			return false, err
		}
		for _, ownerRef := range rs.OwnerReferences {
			if _, ok := d.deployments[ownerRef.Name]; ok {
				return false, nil
			}
		}

	default:
		return false, fmt.Errorf("unsupported kind: %s/%s", kind, info.Name)
	}

	return true, nil
}
