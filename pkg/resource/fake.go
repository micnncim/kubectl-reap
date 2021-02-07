package resource

import (
	"context"
	"sync"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

type FakeClient struct {
	fakeObjects                map[fakeObjectKey]runtime.Object
	fakePods                   []*corev1.Pod
	fakeReplicaSets            []*appsv1.ReplicaSet
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
		fakeReplicaSets            []*appsv1.ReplicaSet
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
		case KindReplicaSet:
			fakeReplicaSets = append(fakeReplicaSets, obj.(*appsv1.ReplicaSet))
		case KindServiceAccount:
			fakeServiceAccounts = append(fakeServiceAccounts, obj.(*corev1.ServiceAccount))
		case KindPersistentVolumeClaim:
			fakePersistentVolumeClaims = append(fakePersistentVolumeClaims, obj.(*corev1.PersistentVolumeClaim))
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
	c.mu.RLock()
	pods := c.fakePods
	c.mu.RUnlock()
	return pods, nil
}

func (c *FakeClient) ListReplicaSets(ctx context.Context, namespace string) ([]*appsv1.ReplicaSet, error) {
	c.mu.RLock()
	rss := c.fakeReplicaSets
	c.mu.RUnlock()
	return rss, nil
}

func (c *FakeClient) ListServiceAccounts(ctx context.Context, namespace string) ([]*corev1.ServiceAccount, error) {
	c.mu.RLock()
	sas := c.fakeServiceAccounts
	c.mu.RUnlock()
	return sas, nil
}

func (c *FakeClient) ListPersistentVolumeClaims(ctx context.Context, namespace string) ([]*corev1.PersistentVolumeClaim, error) {
	c.mu.RLock()
	pvcs := c.fakePersistentVolumeClaims
	c.mu.RUnlock()
	return pvcs, nil
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
