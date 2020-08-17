package cmd

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func listPods(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]*corev1.Pod, error) {
	podList, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	pods := make([]*corev1.Pod, 0, len(podList.Items))
	for i := range podList.Items {
		pods = append(pods, &podList.Items[i])
	}

	return pods, nil
}

func listReplicaSets(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]*appsv1.ReplicaSet, error) {
	rsList, err := clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	rss := make([]*appsv1.ReplicaSet, 0, len(rsList.Items))
	for i := range rsList.Items {
		rss = append(rss, &rsList.Items[i])
	}

	return rss, nil
}

func listDeployments(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]*appsv1.Deployment, error) {
	deployList, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	deploys := make([]*appsv1.Deployment, 0, len(deployList.Items))
	for i := range deployList.Items {
		deploys = append(deploys, &deployList.Items[i])
	}

	return deploys, nil
}

func listServiceAccounts(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]*corev1.ServiceAccount, error) {
	saList, err := clientset.CoreV1().ServiceAccounts(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	sas := make([]*corev1.ServiceAccount, 0, len(saList.Items))
	for i := range saList.Items {
		sas = append(sas, &saList.Items[i])
	}

	return sas, nil
}

func detectUsedConfigMaps(pods []*corev1.Pod) map[string]struct{} {
	usedCms := make(map[string]struct{})

	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {
			for _, envFrom := range container.EnvFrom {
				if envFrom.ConfigMapRef != nil {
					usedCms[envFrom.ConfigMapRef.Name] = struct{}{}
				}
			}

			for _, env := range container.Env {
				if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil {
					usedCms[env.ValueFrom.ConfigMapKeyRef.Name] = struct{}{}
				}
			}
		}

		for _, volume := range pod.Spec.Volumes {
			if volume.ConfigMap != nil {
				usedCms[volume.ConfigMap.Name] = struct{}{}
			}

			if volume.Projected != nil {
				for _, source := range volume.Projected.Sources {
					if source.ConfigMap != nil {
						usedCms[source.ConfigMap.Name] = struct{}{}
					}
				}
			}
		}
	}

	return usedCms
}

func detectUsedSecrets(pods []*corev1.Pod, sas []*corev1.ServiceAccount) map[string]struct{} {
	usedSecrets := make(map[string]struct{})

	for name := range detectUsedSecretsInPods(pods) {
		usedSecrets[name] = struct{}{}
	}
	for name := range detectUsedSecretsInServiceAccounts(sas) {
		usedSecrets[name] = struct{}{}
	}

	return usedSecrets
}

func detectUsedSecretsInPods(pods []*corev1.Pod) map[string]struct{} {
	usedSecrets := make(map[string]struct{})

	for _, pod := range pods {
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

	return usedSecrets
}

func detectUsedSecretsInServiceAccounts(sas []*corev1.ServiceAccount) map[string]struct{} {
	usedSecrets := make(map[string]struct{})

	for _, sa := range sas {
		for _, secret := range sa.Secrets {
			usedSecrets[secret.Name] = struct{}{}
		}
	}

	return usedSecrets
}

func detectUsedPods(pods []*corev1.Pod, rss []*appsv1.ReplicaSet) map[string]struct{} {
	usedPods := make(map[string]struct{})

	for _, pod := range pods {
		for _, ownerRef := range pod.OwnerReferences {
			for _, rs := range rss {
				if ownerRef.UID == rs.GetUID() {
					usedPods[ownerRef.Name] = struct{}{}
				}
			}
		}
	}

	return usedPods
}

func detectUsedReplicaSets(deploys []*appsv1.Deployment) map[string]struct{} {
	usedRss := make(map[string]struct{})

	for _, deploy := range deploys {
		for _, ownerRef := range deploy.OwnerReferences {
			usedRss[ownerRef.Name] = struct{}{}
		}
	}

	return usedRss
}
