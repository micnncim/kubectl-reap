package cmd

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes"
)

const (
	kindConfigMap = "ConfigMap"
	kindSecret    = "Secret"
	kindPod       = "Pod"
)

// determiner determines whether a resource should be pruned.
type determiner struct {
	usedConfigMaps map[string]struct{} // key=ConfigMap.Name
	usedSecrets    map[string]struct{} // key=Secret.Name

	pods []*corev1.Pod
}

func newDeterminer(clientset *kubernetes.Clientset, r *resource.Result, namespace string) (*determiner, error) {
	var (
		pruneConfigMaps bool
		pruneSecrets    bool
		prunePods       bool
	)

	if err := r.Visit(func(info *resource.Info, err error) error {
		switch info.Object.GetObjectKind().GroupVersionKind().Kind {
		case kindConfigMap:
			pruneConfigMaps = true
		case kindSecret:
			pruneSecrets = true
		case kindPod:
			prunePods = true
		}
		return nil
	}); err != nil {
		return nil, err
	}

	d := &determiner{}

	ctx := context.Background()

	if pruneConfigMaps || pruneSecrets || prunePods {
		var err error
		d.pods, err = listPods(ctx, clientset, namespace)
		if err != nil {
			return nil, err
		}
	}

	if pruneConfigMaps {
		d.usedConfigMaps = detectUsedConfigMaps(d.pods)
	}

	if pruneSecrets {
		sas, err := listServiceAccounts(ctx, clientset, namespace)
		if err != nil {
			return nil, err
		}
		d.usedSecrets = detectUsedSecrets(d.pods, sas)
	}

	return d, nil
}

// determinePrune determines whether a resource should be pruned.
func (d *determiner) determinePrune(info *resource.Info) (bool, error) {
	switch kind := info.Object.GetObjectKind().GroupVersionKind().Kind; kind {
	case kindConfigMap:
		if _, ok := d.usedConfigMaps[info.Name]; !ok {
			return true, nil
		}

	case kindSecret:
		if _, ok := d.usedSecrets[info.Name]; !ok {
			return true, nil
		}

	case kindPod:
		var pod corev1.Pod
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(
			info.Object.(runtime.Unstructured).UnstructuredContent(),
			&pod,
		); err != nil {
			return false, err
		}

		if pod.Status.Phase != corev1.PodRunning {
			return true, nil
		}

	default:
		return false, fmt.Errorf("unsupported kind: %s/%s", kind, info.Name)
	}

	return false, nil
}
