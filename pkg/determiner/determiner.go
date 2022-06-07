package determiner

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	cliresource "k8s.io/cli-runtime/pkg/resource"

	"github.com/micnncim/kubectl-reap/pkg/resource"
)

var checkVolumeSatisfyClaimFunc = resource.CheckVolumeSatisfyClaim

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
	replicaSets            []*appsv1.ReplicaSet
	persistentVolumeClaims []*corev1.PersistentVolumeClaim
}

// Guarantee *determiner implements Determiner.
var _ Determiner = (*determiner)(nil)

func New(resourceClient resource.Client, r *cliresource.Result, namespace string) (Determiner, error) {
	d := &determiner{
		resourceClient: resourceClient,
	}

	var (
		reapConfigMaps             bool
		reapSecrets                bool
		reapPersistentVolumes      bool
		reapPersistentVolumeClaims bool
		reapPodDisruptionBudgets   bool
	)

	if err := r.Visit(func(info *cliresource.Info, err error) error {
		switch info.Object.GetObjectKind().GroupVersionKind().Kind {
		case resource.KindConfigMap:
			reapConfigMaps = true
		case resource.KindSecret:
			reapSecrets = true
		case resource.KindPersistentVolume:
			reapPersistentVolumes = true
		case resource.KindPersistentVolumeClaim:
			reapPersistentVolumeClaims = true
		case resource.KindPodDisruptionBudget:
			reapPodDisruptionBudgets = true
		}
		return nil
	}); err != nil {
		return nil, err
	}

	ctx := context.Background()

	if reapConfigMaps || reapSecrets || reapPersistentVolumeClaims || reapPodDisruptionBudgets {
		var err error
		d.pods, err = d.resourceClient.ListPods(ctx, namespace)
		if err != nil {
			return nil, err
		}
	}

	if reapConfigMaps || reapSecrets {
		var err error
		d.replicaSets, err = d.resourceClient.ListReplicaSets(ctx, namespace)
		if err != nil {
			return nil, err
		}
	}

	if reapPersistentVolumes {
		var err error
		d.persistentVolumeClaims, err = d.resourceClient.ListPersistentVolumeClaims(ctx, namespace)
		if err != nil {
			return nil, err
		}
	}

	if reapConfigMaps {
		d.usedConfigMaps = d.detectUsedConfigMaps()
	}

	if reapSecrets {
		sas, err := d.resourceClient.ListServiceAccounts(ctx, namespace)
		if err != nil {
			return nil, err
		}
		d.usedSecrets = d.detectUsedSecrets(sas)
	}

	if reapPersistentVolumeClaims {
		d.usedPersistentVolumeClaims = d.detectUsedPersistentVolumeClaims()
	}

	return d, nil
}

// DetermineDeletion determines whether a resource should be deleted.
func (d *determiner) DetermineDeletion(ctx context.Context, info *cliresource.Info) (bool, error) {
	switch kind := info.Object.GetObjectKind().GroupVersionKind().Kind; kind {
	case resource.KindPod:
		return d.determineDeletionPod(info)

	case resource.KindConfigMap:
		return d.determineDeletionConfigMap(info)

	case resource.KindSecret:
		return d.determineDeletionSecret(info)

	case resource.KindPersistentVolume:
		return d.determineDeletionPersistentVolume(info)

	case resource.KindPersistentVolumeClaim:
		return d.determineDeletionPersistentVolumeClaim(info)

	case resource.KindJob:
		return d.determineDeletionJob(info)

	case resource.KindPodDisruptionBudget:
		return d.determineDeletionPodDisruptionBudget(info)

	case resource.KindHorizontalPodAutoscaler:
		return d.determineDeletionHorizontalPodAutoscaler(ctx, info)

	default:
		return false, fmt.Errorf("unsupported kind: %s/%s", kind, info.Name)
	}
}

func (d *determiner) determineDeletionPod(info *cliresource.Info) (bool, error) {
	pod, err := resource.ObjectToPod(info.Object)
	if err != nil {
		return false, err
	}

	return pod.Status.Phase != corev1.PodRunning, nil
}

func (d *determiner) determineDeletionConfigMap(info *cliresource.Info) (bool, error) {
	_, ok := d.usedConfigMaps[info.Name]
	return !ok, nil
}

func (d *determiner) determineDeletionSecret(info *cliresource.Info) (bool, error) {
	_, ok := d.usedSecrets[info.Name]
	return !ok, nil
}

func (d *determiner) determineDeletionPersistentVolume(info *cliresource.Info) (bool, error) {
	volume, err := resource.ObjectToPersistentVolume(info.Object)
	if err != nil {
		return false, err
	}

	for _, claim := range d.persistentVolumeClaims {
		if ok := checkVolumeSatisfyClaimFunc(volume, claim); ok {
			return false, nil
		}
	}
	return true, nil // should delete PV if it doesn't satisfy any PVCs
}

func (d *determiner) determineDeletionPersistentVolumeClaim(info *cliresource.Info) (bool, error) {
	_, ok := d.usedPersistentVolumeClaims[info.Name]
	return !ok, nil
}

func (d *determiner) determineDeletionJob(info *cliresource.Info) (bool, error) {
	job, err := resource.ObjectToJob(info.Object)
	if err != nil {
		return false, err
	}

	return job.Status.CompletionTime != nil, nil
}

func (d *determiner) determineDeletionPodDisruptionBudget(info *cliresource.Info) (bool, error) {
	pdb, err := resource.ObjectToPodDisruptionBudget(info.Object)
	if err != nil {
		return false, err
	}

	used, err := d.determineUsedPodDisruptionBudget(pdb)
	if err != nil {
		return false, err
	}
	return !used, nil
}

func (d *determiner) determineDeletionHorizontalPodAutoscaler(ctx context.Context, info *cliresource.Info) (bool, error) {
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
}

func (d *determiner) detectUsedConfigMaps() map[string]struct{} {
	usedConfigMaps := make(map[string]struct{})

	// Add Secrets used by Pods
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

	// Add Secrets used by ReplicaSets
	for _, rs := range d.replicaSets {
		for _, container := range rs.Spec.Template.Spec.Containers {
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

		for _, volume := range rs.Spec.Template.Spec.Volumes {
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

	// Add Secrets used by Pods
	for _, pod := range d.pods {
		for _, imagePullSecret := range pod.Spec.ImagePullSecrets {
			usedSecrets[imagePullSecret.Name] = struct{}{}
		}

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

	// Add Secrets used by ReplicaSets
	for _, rs := range d.replicaSets {
		for _, container := range rs.Spec.Template.Spec.Containers {
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

		for _, volume := range rs.Spec.Template.Spec.Volumes {
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

	// Add Secrets used by ServiceAccounts
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
