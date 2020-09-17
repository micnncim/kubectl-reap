package determiner

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	cliresource "k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes"

	"github.com/micnncim/kubectl-prune/pkg/resource"
)

const (
	kindConfigMap             = "ConfigMap"
	kindSecret                = "Secret"
	kindPod                   = "Pod"
	kindPersistentVolumeClaim = "PersistentVolumeClaim"
	kindPodDisruptionBudget   = "PodDisruptionBudget"
)

// Determiner determines whether a resource should be pruned.
type Determiner struct {
	UsedConfigMaps             map[string]struct{} // key=ConfigMap.Name
	UsedSecrets                map[string]struct{} // key=Secret.Name
	UsedPersistentVolumeClaims map[string]struct{} // key=PersistentVolumeClaim.Name

	Pods []*corev1.Pod
}

func New(clientset *kubernetes.Clientset, r *cliresource.Result, namespace string) (*Determiner, error) {
	var (
		pruneConfigMaps             bool
		pruneSecrets                bool
		prunePersistentVolumeClaims bool
		prunePodDisruptionBudgets   bool
	)

	if err := r.Visit(func(info *cliresource.Info, err error) error {
		switch info.Object.GetObjectKind().GroupVersionKind().Kind {
		case kindConfigMap:
			pruneConfigMaps = true
		case kindSecret:
			pruneSecrets = true
		case kindPersistentVolumeClaim:
			prunePersistentVolumeClaims = true
		case kindPodDisruptionBudget:
			prunePodDisruptionBudgets = true
		}
		return nil
	}); err != nil {
		return nil, err
	}

	d := &Determiner{}
	client := resource.NewClient(clientset)

	ctx := context.Background()

	if pruneConfigMaps || pruneSecrets || prunePersistentVolumeClaims || prunePodDisruptionBudgets {
		var err error
		d.Pods, err = client.ListPods(ctx, namespace)
		if err != nil {
			return nil, err
		}
	}

	if pruneConfigMaps {
		d.UsedConfigMaps = d.detectUsedConfigMaps()
	}

	if pruneSecrets {
		sas, err := client.ListServiceAccounts(ctx, namespace)
		if err != nil {
			return nil, err
		}
		d.UsedSecrets = d.detectUsedSecrets(sas)
	}

	if prunePersistentVolumeClaims {
		d.UsedPersistentVolumeClaims = d.detectUsedPersistentVolumeClaims()
	}

	return d, nil
}

// DeterminePrune determines whether a resource should be pruned.
func (d *Determiner) DeterminePrune(info *cliresource.Info) (bool, error) {
	switch kind := info.Object.GetObjectKind().GroupVersionKind().Kind; kind {
	case kindConfigMap:
		if _, ok := d.UsedConfigMaps[info.Name]; !ok {
			return true, nil
		}

	case kindSecret:
		if _, ok := d.UsedSecrets[info.Name]; !ok {
			return true, nil
		}

	case kindPersistentVolumeClaim:
		if _, ok := d.UsedPersistentVolumeClaims[info.Name]; !ok {
			return true, nil
		}

	case kindPod:
		pod, err := resource.InfoToPod(info)
		if err != nil {
			return false, err
		}

		if pod.Status.Phase != corev1.PodRunning {
			return true, nil
		}

	case kindPodDisruptionBudget:
		pdb, err := resource.InfoToPodDisruptionBudget(info)
		if err != nil {
			return false, err
		}

		used, err := d.determineUsedPodDisruptionBudget(pdb)
		if err != nil {
			return false, err
		}
		return !used, nil

	default:
		return false, fmt.Errorf("unsupported kind: %s/%s", kind, info.Name)
	}

	return false, nil
}

func (d *Determiner) detectUsedConfigMaps() map[string]struct{} {
	usedConfigMaps := make(map[string]struct{})

	for _, pod := range d.Pods {
		for _, container := range pod.Spec.Containers {
			for _, envFrom := range container.EnvFrom {
				if envFrom.ConfigMapRef != nil {
					usedConfigMaps[envFrom.ConfigMapRef.Name] = struct{}{}
				}
			}

			for _, env := range container.Env {
				if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil {
					usedConfigMaps[env.ValueFrom.ConfigMapKeyRef.Name] = struct{}{}
				}
			}
		}

		for _, volume := range pod.Spec.Volumes {
			if volume.ConfigMap != nil {
				usedConfigMaps[volume.ConfigMap.Name] = struct{}{}
			}

			if volume.Projected != nil {
				for _, source := range volume.Projected.Sources {
					if source.ConfigMap != nil {
						usedConfigMaps[source.ConfigMap.Name] = struct{}{}
					}
				}
			}
		}
	}

	return usedConfigMaps
}

func (d *Determiner) detectUsedSecrets(sas []*corev1.ServiceAccount) map[string]struct{} {
	usedSecrets := make(map[string]struct{})

	// Add Secrets used in Pods
	for _, pod := range d.Pods {
		for _, container := range pod.Spec.Containers {
			for _, envFrom := range container.EnvFrom {
				if envFrom.SecretRef != nil {
					usedSecrets[envFrom.SecretRef.Name] = struct{}{}
				}
			}

			for _, env := range container.Env {
				if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
					usedSecrets[env.ValueFrom.SecretKeyRef.Name] = struct{}{}
				}
			}
		}

		for _, volume := range pod.Spec.Volumes {
			if volume.Secret != nil {
				usedSecrets[volume.Secret.SecretName] = struct{}{}
			}

			if volume.Projected != nil {
				for _, source := range volume.Projected.Sources {
					if source.Secret != nil {
						usedSecrets[source.Secret.Name] = struct{}{}
					}
				}
			}
		}
	}

	// Add Secrets used in ServiceAccounts
	for _, sa := range sas {
		for _, secret := range sa.Secrets {
			usedSecrets[secret.Name] = struct{}{}
		}
	}

	return usedSecrets
}

func (d *Determiner) detectUsedPersistentVolumeClaims() map[string]struct{} {
	usedPersistentVolumeClaims := make(map[string]struct{})

	for _, pod := range d.Pods {
		for _, volume := range pod.Spec.Volumes {
			if volume.PersistentVolumeClaim == nil {
				continue
			}
			usedPersistentVolumeClaims[volume.PersistentVolumeClaim.ClaimName] = struct{}{}
		}
	}

	return usedPersistentVolumeClaims
}

func (d *Determiner) determineUsedPodDisruptionBudget(pdb *policyv1beta1.PodDisruptionBudget) (bool, error) {
	selector, err := metav1.LabelSelectorAsSelector(pdb.Spec.Selector)
	if err != nil {
		return false, fmt.Errorf("invalid label selector (%s): %w", pdb.Name, err)
	}

	for _, pod := range d.Pods {
		if selector.Matches(labels.Set(pod.Labels)) {
			return true, nil
		}
	}

	return false, nil
}
