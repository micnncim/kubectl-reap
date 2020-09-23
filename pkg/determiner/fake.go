package determiner

import (
	"context"
	"sync"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	cliresource "k8s.io/cli-runtime/pkg/resource"
)

type FakeDeterminer struct {
	fakeObjectsToBeDeleted map[fakeObjectKey]struct{}

	mu sync.RWMutex
}

type fakeObjectKey struct {
	kind      string
	name      string
	namespace string
}

// Guarantee *FakeDeterminer implements Determiner.
var _ Determiner = (*FakeDeterminer)(nil)

func NewFakeDeterminer(objects ...runtime.Object) (*FakeDeterminer, error) {
	fakeObjectsToBeDeleted := make(map[fakeObjectKey]struct{})

	accessor := apimeta.NewAccessor()

	for _, obj := range objects {
		kind, err := accessor.Kind(obj)
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
			kind:      kind,
			name:      name,
			namespace: namespace,
		}
		fakeObjectsToBeDeleted[key] = struct{}{}
	}

	return &FakeDeterminer{
		fakeObjectsToBeDeleted: fakeObjectsToBeDeleted,
		mu:                     sync.RWMutex{},
	}, nil
}

func (d *FakeDeterminer) DetermineDeletion(_ context.Context, info *cliresource.Info) (bool, error) {
	key := fakeObjectKey{
		kind:      info.Object.GetObjectKind().GroupVersionKind().Kind,
		name:      info.Name,
		namespace: info.Namespace,
	}

	d.mu.RLock()
	_, ok := d.fakeObjectsToBeDeleted[key]
	d.mu.RUnlock()

	return ok, nil
}
