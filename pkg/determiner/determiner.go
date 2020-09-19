package determiner

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	cliresource "k8s.io/cli-runtime/pkg/resource"

	"github.com/micnncim/kubectl-prune/pkg/resource"
)

const (
	kindConfigMap               = "ConfigMap"
	kindSecret                  = "Secret"
	kindPod                     = "Pod"
	kindPersistentVolume        = "PersistentVolume"
	kindPersistentVolumeClaim   = "PersistentVolumeClaim"
	kindPodDisruptionBudget     = "PodDisruptionBudget"
	kindHorizontalPodAutoscaler = "HorizontalPodAutoscaler"
)

type Determiner interface {
	DetermineDeletion(ctx context.Context, info *cliresource.Info) (bool, error)
}

// determiner determines whether a resource should be deleted.
type determiner struct {
	resourceClient resource.Client

	usedConfigMaps             map[string]struct{} // key=ConfigMap.Name
	usedSecrets                map[string]struct{} // key=Secret.Name
	usedPersistentVolumeClaims map[string]struct{} // key=PersistentVolumeClaim.Name

	pods                   []*corev1.Pod
	persistentVolumeClaims []*corev1.PersistentVolumeClaim
}

// Guarantee *determiner implements Determiner.
var _ Determiner = (*determiner)(nil)

func New(resourceClient resource.Client, r *cliresource.Result, namespace string) (Determiner, error) {
	d := &determiner{
		resourceClient: resourceClient,
	}

	var (
		pruneConfigMaps             bool
		pruneSecrets                bool
		prunePersistentVolumes      bool
		prunePersistentVolumeClaims bool
		prunePodDisruptionBudgets   bool
	)

	if err := r.Visit(func(info *cliresource.Info, err error) error {
		switch info.Object.GetObjectKind().GroupVersionKind().Kind {
		case kindConfigMap:
			pruneConfigMaps = true
		case kindSecret:
			pruneSecrets = true
		case kindPersistentVolume:
			prunePersistentVolumes = true
		case kindPersistentVolumeClaim:
			prunePersistentVolumeClaims = true
		case kindPodDisruptionBudget:
			prunePodDisruptionBudgets = true
		}
		return nil
	}); err != nil {
		return nil, err
	}

	ctx := context.Background()

	if pruneConfigMaps || pruneSecrets || prunePersistentVolumeClaims || prunePodDisruptionBudgets {
		var err error
		d.pods, err = d.resourceClient.ListPods(ctx, namespace)
		if err != nil {
			return nil, err
		}
	}

	if prunePersistentVolumes {
		var err error
		d.persistentVolumeClaims, err = d.resourceClient.ListPersistentVolumeClaims(ctx, namespace)
		if err != nil {
			return nil, err
		}
	}

	if pruneConfigMaps {
		d.usedConfigMaps = d.detectUsedConfigMaps()
	}

	if pruneSecrets {
		sas, err := d.resourceClient.ListServiceAccounts(ctx, namespace)
		if err != nil {
			return nil, err
		}
		d.usedSecrets = d.detectUsedSecrets(sas)
	}

	if prunePersistentVolumeClaims {
		d.usedPersistentVolumeClaims = d.detectUsedPersistentVolumeClaims()
	}

	return d, nil
}

// DetermineDeletion determines whether a resource should be deleted.
func (d *determiner) DetermineDeletion(ctx context.Context, info *cliresource.Info) (bool, error) {
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
		pod, err := resource.ObjectToPod(info.Object)
		if err != nil {
			return false, err
		}

		if pod.Status.Phase != corev1.PodRunning {
			return true, nil
		}

	case kindPersistentVolume:
		volume, err := resource.ObjectToPersistentVolume(info.Object)
		if err != nil {
			return false, err
		}

		for _, claim := range d.persistentVolumeClaims {
			if ok := resource.CheckVolumeSatisfyClaim(volume, claim); ok {
				return false, nil
			}
		}
		return true, nil // should delete PV if it doesn't satisfy any PVCs

	case kindPersistentVolumeClaim:
		if _, ok := d.usedPersistentVolumeClaims[info.Name]; !ok {
			return true, nil
		}

	case kindPodDisruptionBudget:
		pdb, err := resource.ObjectToPodDisruptionBudget(info.Object)
		if err != nil {
			return false, err
		}

		used, err := d.determineUsedPodDisruptionBudget(pdb)
		if err != nil {
			return false, err
		}
		return !used, nil

	case kindHorizontalPodAutoscaler:
		hpa, err := resource.ObjectToHorizontalPodAutoscaler(info.Object)
		if err != nil {
			return false, err
		}

		ref := hpa.Spec.ScaleTargetRef
		u, err := d.resourceClient.GetUnstructured(ctx, ref.APIVersion, ref.Kind, ref.Name, info.Namespace)
		if err != nil {
			return false, err
		}
		return u == nil, nil // should delete HPA if ScaleTargetRef's target object is not found

	default:
		return false, fmt.Errorf("unsupported kind: %s/%s", kind, info.Name)
	}

	return false, nil
}

func (d *determiner) detectUsedConfigMaps() map[string]struct{} {
	usedConfigMaps := make(map[string]struct{})

	for _, pod := range d.pods {
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

func (d *determiner) detectUsedSecrets(sas []*corev1.ServiceAccount) map[string]struct{} {
	usedSecrets := make(map[string]struct{})

	// Add Secrets used in Pods
	for _, pod := range d.pods {
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

func (d *determiner) detectUsedPersistentVolumeClaims() map[string]struct{} {
	usedPersistentVolumeClaims := make(map[string]struct{})

	for _, pod := range d.pods {
		for _, volume := range pod.Spec.Volumes {
			if volume.PersistentVolumeClaim == nil {
				continue
			}
			usedPersistentVolumeClaims[volume.PersistentVolumeClaim.ClaimName] = struct{}{}
		}
	}

	return usedPersistentVolumeClaims
}

func (d *determiner) determineUsedPodDisruptionBudget(pdb *policyv1beta1.PodDisruptionBudget) (bool, error) {
	selector, err := metav1.LabelSelectorAsSelector(pdb.Spec.Selector)
	if err != nil {
		return false, fmt.Errorf("invalid label selector (%s): %w", pdb.Name, err)
	}

	for _, pod := range d.pods {
		if selector.Matches(labels.Set(pod.Labels)) {
			return true, nil
		}
	}

	return false, nil
}
