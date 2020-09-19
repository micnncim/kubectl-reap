package resource

import (
	"context"
	"sync"

	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

type FakeClient struct {
	fakeObjects                map[fakeObjectKey]runtime.Object
	fakePods                   []*corev1.Pod
	fakeServiceAccounts        []*corev1.ServiceAccount
	fakePersistentVolumeClaims []*corev1.PersistentVolumeClaim

	mu sync.RWMutex
}

type fakeObjectKey struct {
	apiVersion string
	kind       string
	name       string
	namespace  string
}

func NewFakeClient(objects ...runtime.Object) (*FakeClient, error) {
	fakeObjects := make(map[fakeObjectKey]runtime.Object)

	var (
		fakePods                   []*corev1.Pod
		fakeServiceAccounts        []*corev1.ServiceAccount
		fakePersistentVolumeClaims []*corev1.PersistentVolumeClaim
	)

	accessor := apimeta.NewAccessor()

	for _, obj := range objects {
		kind, err := accessor.Kind(obj)
		if err != nil {
			return nil, err
		}

		switch kind {
		case KindPod:
			fakePods = append(fakePods, obj.(*corev1.Pod))
			continue
		case KindServiceAccount:
			fakeServiceAccounts = append(fakeServiceAccounts, obj.(*corev1.ServiceAccount))
			continue
		case KindPersistentVolumeClaim:
			fakePersistentVolumeClaims = append(fakePersistentVolumeClaims, obj.(*corev1.PersistentVolumeClaim))
			continue
		}

		apiVersion, err := accessor.APIVersion(obj)
		if err != nil {
			return nil, err
		}
		name, err := accessor.Name(obj)
		if err != nil {
			return nil, err
		}
		namespace, err := accessor.Namespace(obj)
		if err != nil {
			return nil, err
		}

		key := fakeObjectKey{
			apiVersion: apiVersion,
			kind:       kind,
			name:       name,
			namespace:  namespace,
		}
		fakeObjects[key] = obj
	}

	return &FakeClient{
		fakeObjects:                fakeObjects,
		fakePods:                   fakePods,
		fakeServiceAccounts:        fakeServiceAccounts,
		fakePersistentVolumeClaims: fakePersistentVolumeClaims,
	}, nil
}

// Guarantee *FakeClient implements Client.
var _ Client = (*FakeClient)(nil)

func (c *FakeClient) ListPods(ctx context.Context, namespace string) ([]*corev1.Pod, error) {
	return c.fakePods, nil
}

func (c *FakeClient) ListServiceAccounts(ctx context.Context, namespace string) ([]*corev1.ServiceAccount, error) {
	return c.fakeServiceAccounts, nil
}

func (c *FakeClient) ListPersistentVolumeClaims(ctx context.Context, namespace string) ([]*corev1.PersistentVolumeClaim, error) {
	return c.fakePersistentVolumeClaims, nil
}

func (c *FakeClient) GetUnstructured(ctx context.Context, apiVersion, kind, name, namespace string) (*unstructured.Unstructured, error) {
	key := fakeObjectKey{
		apiVersion: apiVersion,
		kind:       kind,
		name:       name,
		namespace:  namespace,
	}

	c.mu.RLock()
	obj, ok := c.fakeObjects[key]
	c.mu.RUnlock()
	if !ok {
		return nil, nil
	}

	u, err := unstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{
		Object: u,
	}, nil
}
