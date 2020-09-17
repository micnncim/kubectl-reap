package cmd

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes"
)

func listPods(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]*corev1.Pod, error) {
	podList, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return podListToPods(podList), nil
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

func podListToPods(podList *corev1.PodList) []*corev1.Pod {
	pods := make([]*corev1.Pod, 0, len(podList.Items))
	for i := range podList.Items {
		pods = append(pods, &podList.Items[i])
	}
	return pods
}

func infoToPod(info *resource.Info) (*corev1.Pod, error) {
	var pod corev1.Pod
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(
		info.Object.(runtime.Unstructured).UnstructuredContent(),
		&pod,
	); err != nil {
		return nil, err
	}

	return &pod, nil
}
