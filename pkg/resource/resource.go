package resource

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	cliresource "k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes"
)

type Client interface {
	ListPods(ctx context.Context, namespace string) ([]*corev1.Pod, error)
	ListServiceAccounts(ctx context.Context, namespace string) ([]*corev1.ServiceAccount, error)
}

type client struct {
	clientset *kubernetes.Clientset
}

// Guarantee *client implements Client.
var _ Client = (*client)(nil)

func NewClient(clientset *kubernetes.Clientset) Client {
	return &client{
		clientset: clientset,
	}
}

func (c *client) ListPods(ctx context.Context, namespace string) ([]*corev1.Pod, error) {
	podList, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return PodListToPods(podList), nil
}

func (c *client) ListServiceAccounts(ctx context.Context, namespace string) ([]*corev1.ServiceAccount, error) {
	saList, err := c.clientset.CoreV1().ServiceAccounts(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	sas := make([]*corev1.ServiceAccount, 0, len(saList.Items))
	for i := range saList.Items {
		sas = append(sas, &saList.Items[i])
	}

	return sas, nil
}

func InfoToPod(info *cliresource.Info) (*corev1.Pod, error) {
	var pod corev1.Pod
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(
		info.Object.(runtime.Unstructured).UnstructuredContent(),
		&pod,
	); err != nil {
		return nil, err
	}

	return &pod, nil
}

func InfoToPodDisruptionBudget(info *cliresource.Info) (*policyv1beta1.PodDisruptionBudget, error) {
	var pdb policyv1beta1.PodDisruptionBudget
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(
		info.Object.(runtime.Unstructured).UnstructuredContent(),
		&pdb,
	); err != nil {
		return nil, err
	}

	return &pdb, nil
}

func PodListToPods(podList *corev1.PodList) []*corev1.Pod {
	pods := make([]*corev1.Pod, 0, len(podList.Items))
	for i := range podList.Items {
		pods = append(pods, &podList.Items[i])
	}
	return pods
}
