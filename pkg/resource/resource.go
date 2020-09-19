package resource

import (
	"context"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliresource "k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

var (
	unstructuredConverter = runtime.DefaultUnstructuredConverter
)

type Client interface {
	ListPods(ctx context.Context, namespace string) ([]*corev1.Pod, error)
	ListServiceAccounts(ctx context.Context, namespace string) ([]*corev1.ServiceAccount, error)
	ListPersistentVolumeClaims(ctx context.Context, namespace string) ([]*corev1.PersistentVolumeClaim, error)
	FindScaleTargetRefObject(ctx context.Context, objectRef *autoscalingv1.CrossVersionObjectReference, namespace string) (bool, error)
}

type client struct {
	clientset     *kubernetes.Clientset
	dynamicClient dynamic.Interface
}

// Guarantee *client implements Client.
var _ Client = (*client)(nil)

func NewClient(clientset *kubernetes.Clientset, dynamicClient dynamic.Interface) Client {
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

func (c *client) FindScaleTargetRefObject(ctx context.Context, objectRef *autoscalingv1.CrossVersionObjectReference, namespace string) (bool, error) {
	gv, err := schema.ParseGroupVersion(objectRef.APIVersion)
	if err != nil {
		return false, err
	}

	gvk := gv.WithKind(objectRef.Kind)
	gvr, _ := apimeta.UnsafeGuessKindToResource(gvk)

	_, err = c.dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, objectRef.Name, metav1.GetOptions{})
	switch {
	case err == nil:
		return true, nil
	case apierrors.IsNotFound(err):
		return false, nil
	default:
		return false, err
	}
}

func InfoToPod(info *cliresource.Info) (*corev1.Pod, error) {
	u, err := unstructuredConverter.ToUnstructured(info.Object)
	if err != nil {
		return nil, err
	}

	var pod corev1.Pod
	if err := unstructuredConverter.FromUnstructured(u, &pod); err != nil {
		return nil, err
	}

	return &pod, nil
}

func InfoToPersistentVolume(info *cliresource.Info) (*corev1.PersistentVolume, error) {
	u, err := unstructuredConverter.ToUnstructured(info.Object)
	if err != nil {
		return nil, err
	}

	var volume corev1.PersistentVolume
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u, &volume); err != nil {
		return nil, err
	}

	return &volume, nil
}

func InfoToPodDisruptionBudget(info *cliresource.Info) (*policyv1beta1.PodDisruptionBudget, error) {
	u, err := unstructuredConverter.ToUnstructured(info.Object)
	if err != nil {
		return nil, err
	}

	var pdb policyv1beta1.PodDisruptionBudget
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u, &pdb); err != nil {
		return nil, err
	}

	return &pdb, nil
}

func InfoToHorizontalPodAutoscaler(info *cliresource.Info) (*autoscalingv1.HorizontalPodAutoscaler, error) {
	u, err := unstructuredConverter.ToUnstructured(info.Object)
	if err != nil {
		return nil, err
	}

	var hpa autoscalingv1.HorizontalPodAutoscaler
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u, &hpa); err != nil {
		return nil, err
	}

	return &hpa, nil
}

func PodListToPods(podList *corev1.PodList) []*corev1.Pod {
	pods := make([]*corev1.Pod, 0, len(podList.Items))
	for i := range podList.Items {
		pods = append(pods, &podList.Items[i])
	}
	return pods
}
