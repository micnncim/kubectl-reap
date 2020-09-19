package resource

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
)

func Test_client_ListPods(t *testing.T) {
	const (
		fakeNamespace = "fake-ns"
		fakePod       = "fake-pod"
	)

	tests := []struct {
		name    string
		objects []runtime.Object
		want    []*corev1.Pod
		wantErr bool
	}{
		{
			name: "expected Pods",
			objects: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fakePod,
						Namespace: fakeNamespace,
					},
				},
			},
			want: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fakePod,
						Namespace: fakeNamespace,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := &client{
				clientset: fakeclientset.NewSimpleClientset(tt.objects...),
			}

			got, err := c.ListPods(context.Background(), fakeNamespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("client.ListPods() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func Test_client_ListServiceAccounts(t *testing.T) {
	const (
		fakeNamespace      = "fake-ns"
		fakeServiceAccount = "fake-sa"
	)

	tests := []struct {
		name    string
		objects []runtime.Object
		want    []*corev1.ServiceAccount
		wantErr bool
	}{
		{
			name: "expected ServiceAccounts",
			objects: []runtime.Object{
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fakeServiceAccount,
						Namespace: fakeNamespace,
					},
				},
			},
			want: []*corev1.ServiceAccount{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fakeServiceAccount,
						Namespace: fakeNamespace,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := &client{
				clientset: fakeclientset.NewSimpleClientset(tt.objects...),
			}

			got, err := c.ListServiceAccounts(context.Background(), fakeNamespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("client.ListServiceAccounts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func Test_client_ListPersistentVolumeClaims(t *testing.T) {
	const (
		fakeNamespace             = "fake-ns"
		fakePersistentVolumeClaim = "fake-pvc"
	)

	tests := []struct {
		name    string
		objects []runtime.Object
		want    []*corev1.PersistentVolumeClaim
		wantErr bool
	}{
		{
			name: "expected PersistentVolumeClaims",
			objects: []runtime.Object{
				&corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fakePersistentVolumeClaim,
						Namespace: fakeNamespace,
					},
				},
			},
			want: []*corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fakePersistentVolumeClaim,
						Namespace: fakeNamespace,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := &client{
				clientset: fakeclientset.NewSimpleClientset(tt.objects...),
			}

			got, err := c.ListPersistentVolumeClaims(context.Background(), fakeNamespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("client.ListPersistentVolumeClaims() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
