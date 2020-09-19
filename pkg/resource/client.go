package resource

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type Client interface {
	ListPods(ctx context.Context, namespace string) ([]*corev1.Pod, error)
	ListServiceAccounts(ctx context.Context, namespace string) ([]*corev1.ServiceAccount, error)
	ListPersistentVolumeClaims(ctx context.Context, namespace string) ([]*corev1.PersistentVolumeClaim, error)
	GetUnstructured(ctx context.Context, apiVersion, kind, name, namespace string) (*unstructured.Unstructured, error)
}

type client struct {
	clientset     kubernetes.Interface
	dynamicClient dynamic.Interface
}

// Guarantee *client implements Client.
var _ Client = (*client)(nil)

func NewClient(clientset kubernetes.Interface, dynamicClient dynamic.Interface) Client {
	return &client{
		clientset:     clientset,
		dynamicClient: dynamicClient,
	}
}

func (c *client) ListPods(ctx context.Context, namespace string) ([]*corev1.Pod, error) {
	podList, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	pods := make([]*corev1.Pod, 0, len(podList.Items))
	for i := range podList.Items {
		pods = append(pods, &podList.Items[i])
	}

	return pods, nil
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

func (c *client) ListPersistentVolumeClaims(ctx context.Context, namespace string) ([]*corev1.PersistentVolumeClaim, error) {
	pvcList, err := c.clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	pvcs := make([]*corev1.PersistentVolumeClaim, 0, len(pvcList.Items))
	for i := range pvcList.Items {
		pvcs = append(pvcs, &pvcList.Items[i])
	}

	return pvcs, nil
}

func (c *client) GetUnstructured(ctx context.Context, apiVersion, kind, name, namespace string) (*unstructured.Unstructured, error) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return nil, err
	}

	gvk := gv.WithKind(kind)
	gvr, _ := apimeta.UnsafeGuessKindToResource(gvk)

	u, err := c.dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	switch {
	case err == nil:
		return u, nil
	case apierrors.IsNotFound(err):
		return nil, nil
	default:
		return nil, err
	}
}
