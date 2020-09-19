package resource

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	"k8s.io/kubectl/pkg/scheme"
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

func Test_client_GetUnstructured(t *testing.T) {
	const (
		fakeAPIVersion = "apps/v1"
		fakeKind       = "Deployment"
		fakeName       = "fake-deploy"
		fakeNamespace  = "fake-ns"
	)

	type args struct {
		apiVersion string
		kind       string
		name       string
		namespace  string
	}

	tests := []struct {
		name        string
		args        args
		fakeObjects []runtime.Object
		want        *unstructured.Unstructured
		wantErr     bool
	}{
		{
			name: "expected unstructured object",
			args: args{
				apiVersion: fakeAPIVersion,
				kind:       fakeKind,
				name:       fakeName,
				namespace:  fakeNamespace,
			},
			fakeObjects: []runtime.Object{
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fakeName,
						Namespace: fakeNamespace,
					},
				},
			},
			want: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": fakeAPIVersion,
					"kind":       fakeKind,
					"metadata": map[string]interface{}{
						"creationTimestamp": nil,
						"name":              fakeName,
						"namespace":         fakeNamespace,
					},
					"spec": map[string]interface{}{
						"selector": nil,
						"strategy": map[string]interface{}{},
						"template": map[string]interface{}{
							"metadata": map[string]interface{}{
								"creationTimestamp": nil,
							},
							"spec": map[string]interface{}{
								"containers": nil,
							},
						},
					},
					"status": map[string]interface{}{},
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
				dynamicClient: fakedynamic.NewSimpleDynamicClient(scheme.Scheme, tt.fakeObjects...),
			}

			got, err := c.GetUnstructured(context.Background(), tt.args.apiVersion, tt.args.kind, tt.args.name, tt.args.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("client.GetUnstructured() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
