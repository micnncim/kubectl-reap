package resource

import (
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	KindPod                     = "Pod"
	KindConfigMap               = "ConfigMap"
	KindSecret                  = "Secret"
	KindServiceAccount          = "ServiceAccount"
	KindPersistentVolume        = "PersistentVolume"
	KindPersistentVolumeClaim   = "PersistentVolumeClaim"
	KindPodDisruptionBudget     = "PodDisruptionBudget"
	KindHorizontalPodAutoscaler = "HorizontalPodAutoscaler"
)

var unstructuredConverter = runtime.DefaultUnstructuredConverter

func ObjectToPod(obj runtime.Object) (*corev1.Pod, error) {
	u, err := unstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}

	var pod corev1.Pod
	if err := unstructuredConverter.FromUnstructured(u, &pod); err != nil {
		return nil, err
	}

	return &pod, nil
}

func ObjectToPersistentVolume(obj runtime.Object) (*corev1.PersistentVolume, error) {
	u, err := unstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}

	var volume corev1.PersistentVolume
	if err := unstructuredConverter.FromUnstructured(u, &volume); err != nil {
		return nil, err
	}

	return &volume, nil
}

func ObjectToPodDisruptionBudget(obj runtime.Object) (*policyv1beta1.PodDisruptionBudget, error) {
	u, err := unstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}

	var pdb policyv1beta1.PodDisruptionBudget
	if err := unstructuredConverter.FromUnstructured(u, &pdb); err != nil {
		return nil, err
	}

	return &pdb, nil
}

func ObjectToHorizontalPodAutoscaler(obj runtime.Object) (*autoscalingv1.HorizontalPodAutoscaler, error) {
	u, err := unstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}

	var hpa autoscalingv1.HorizontalPodAutoscaler
	if err := unstructuredConverter.FromUnstructured(u, &hpa); err != nil {
		return nil, err
	}

	return &hpa, nil
}
