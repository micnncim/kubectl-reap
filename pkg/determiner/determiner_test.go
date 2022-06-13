package determiner

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	cliresource "k8s.io/cli-runtime/pkg/resource"

	"github.com/micnncim/kubectl-reap/pkg/resource"
)

func Test_determiner_DetermineDeletion(t *testing.T) {
	const (
		fakePod                   = "fake-pod"
		fakeConfigMap             = "fake-cm"
		fakeSecret                = "fake-secret"
		fakePersistentVolumeClaim = "fake-pvc"
		fakeJob                   = "fake-job"
		fakePodDisruptionBudget   = "fake-pdb"
		fakeLabelKey1             = "fake-label1-key"
		fakeLabelValue1           = "fake-label1-value"
		fakeLabelKey2             = "fake-label2-key"
		fakeLabelValue2           = "fake-label2-value"
	)

	fakeTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	type fields struct {
		usedConfigMaps        map[string]struct{}
		usedSecrets           map[string]struct{}
		usedPersistentVolumes map[string]struct{}
		pods                  []*corev1.Pod
	}
	type args struct {
		info *cliresource.Info
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "Pod should be deleted when it is not running",
			args: args{
				info: &cliresource.Info{
					Name: fakePod,
					Object: &corev1.Pod{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindPod,
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodFailed,
						},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Pod should not be deleted when it is running",
			args: args{
				info: &cliresource.Info{
					Name: fakePod,
					Object: &corev1.Pod{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindPod,
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodRunning,
						},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "ConfigMap should be deleted when it is not used",
			args: args{
				info: &cliresource.Info{
					Name: fakeConfigMap,
					Object: &corev1.ConfigMap{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindConfigMap,
						},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "ConfigMap should not be deleted when it is used",
			fields: fields{
				usedConfigMaps: map[string]struct{}{
					fakeConfigMap: {},
				},
			},
			args: args{
				info: &cliresource.Info{
					Name: fakeConfigMap,
					Object: &corev1.ConfigMap{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindConfigMap,
						},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "Secret should be deleted when it is not used",
			args: args{
				info: &cliresource.Info{
					Name: fakeSecret,
					Object: &corev1.Secret{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindSecret,
						},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Secret should not be deleted when it is used",
			fields: fields{
				usedSecrets: map[string]struct{}{
					fakeSecret: {},
				},
			},
			args: args{
				info: &cliresource.Info{
					Name: fakeSecret,
					Object: &corev1.Secret{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindSecret,
						},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "PersistentVolumeClaim should be deleted when it is not used",
			args: args{
				info: &cliresource.Info{
					Name: fakePersistentVolumeClaim,
					Object: &corev1.PersistentVolumeClaim{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindPersistentVolumeClaim,
						},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "PersistentVolumeClaim should not be deleted when it is used",
			fields: fields{
				usedPersistentVolumes: map[string]struct{}{
					fakePersistentVolumeClaim: {},
				},
			},
			args: args{
				info: &cliresource.Info{
					Name: fakePersistentVolumeClaim,
					Object: &corev1.PersistentVolumeClaim{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindPersistentVolumeClaim,
						},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "Job should be deleted when it is completed",
			args: args{
				info: &cliresource.Info{
					Name: fakeJob,
					Object: &batchv1.Job{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindJob,
						},
						Status: batchv1.JobStatus{
							CompletionTime: &metav1.Time{
								Time: fakeTime,
							},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Job should not be deleted when it is not completed",
			args: args{
				info: &cliresource.Info{
					Name: fakeJob,
					Object: &batchv1.Job{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindJob,
						},
						Status: batchv1.JobStatus{},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "PodDisruptionBudget should be deleted when it is not used",
			args: args{
				info: &cliresource.Info{
					Name: fakePodDisruptionBudget,
					Object: &policyv1beta1.PodDisruptionBudget{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindPodDisruptionBudget,
						},
						Spec: policyv1beta1.PodDisruptionBudgetSpec{
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									fakeLabelKey1: fakeLabelValue1,
								},
							},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "PodDisruptionBudget should not be deleted when it is used",
			fields: fields{
				pods: []*corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								fakeLabelKey1: fakeLabelValue1,
							},
						},
					},
				},
			},
			args: args{
				info: &cliresource.Info{
					Name: fakePodDisruptionBudget,
					Object: &policyv1beta1.PodDisruptionBudget{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindPodDisruptionBudget,
						},
						Spec: policyv1beta1.PodDisruptionBudgetSpec{
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									fakeLabelKey1: fakeLabelValue1,
								},
							},
						},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := &determiner{
				usedConfigMaps:             tt.fields.usedConfigMaps,
				usedSecrets:                tt.fields.usedSecrets,
				usedPersistentVolumeClaims: tt.fields.usedPersistentVolumes,
				pods:                       tt.fields.pods,
			}

			got, err := d.DetermineDeletion(context.Background(), tt.args.info)
			if (err != nil) != tt.wantErr {
				t.Errorf("determiner.DetermineDeletion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("determiner.DetermineDeletion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_determiner_DetermineDeletion_PersistentVolume(t *testing.T) {
	const (
		fakePersistentVolume       = "fake-pv"
		fakePersistentVolumeClaim1 = "fake-pvc1"
		fakePersistentVolumeClaim2 = "fake-pvc2"
		fakeLabelKey               = "fake-label-key"
		fakeLabelValue             = "fake-label-value"
	)

	var orgCheckVolumeSatisfyClaimFunc func(volume *corev1.PersistentVolume, claim *corev1.PersistentVolumeClaim) bool
	orgCheckVolumeSatisfyClaimFunc, checkVolumeSatisfyClaimFunc =
		checkVolumeSatisfyClaimFunc,
		func(volume *corev1.PersistentVolume, claim *corev1.PersistentVolumeClaim) bool {
			return volume.Labels[fakeLabelKey] == claim.Labels[fakeLabelKey]
		}
	t.Cleanup(func() {
		checkVolumeSatisfyClaimFunc = orgCheckVolumeSatisfyClaimFunc
	})

	type fields struct {
		persistentVolumeClaims []*corev1.PersistentVolumeClaim
	}
	type args struct {
		info *cliresource.Info
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name:   "PersistentVolume should be deleted when it is not used",
			fields: fields{},
			args: args{
				info: &cliresource.Info{
					Name: fakePersistentVolume,
					Object: &corev1.PersistentVolume{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindPersistentVolume,
						},
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								fakeLabelKey: fakeLabelValue,
							},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "PersistentVolume should not be deleted when it is used",
			fields: fields{
				persistentVolumeClaims: []*corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: fakePersistentVolumeClaim1,
							Labels: map[string]string{
								fakeLabelKey: fakeLabelValue,
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: fakePersistentVolumeClaim2,
						},
					},
				},
			},
			args: args{
				info: &cliresource.Info{
					Name: fakePersistentVolume,
					Object: &corev1.PersistentVolume{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindPersistentVolume,
						},
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								fakeLabelKey: fakeLabelValue,
							},
						},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := &determiner{
				persistentVolumeClaims: tt.fields.persistentVolumeClaims,
			}

			got, err := d.DetermineDeletion(context.Background(), tt.args.info)
			if (err != nil) != tt.wantErr {
				t.Errorf("determiner.DetermineDeletion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("determiner.DetermineDeletion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_determiner_DetermineDeletion_HorizontalPodAutoscaler(t *testing.T) {
	const (
		fakeNamespace                = "fake-ns"
		fakeHorizontalPodAutoscaler  = "fake-hpa"
		fakeScaleTargetRefAPIVersion = "apps/v1"
		fakeScaleTargetRefKind       = "Deployment"
		fakeScaleTargetRefName       = "fake-deploy"
	)

	type args struct {
		info *cliresource.Info
	}

	tests := []struct {
		name        string
		args        args
		fakeObjects []runtime.Object
		want        bool
		wantErr     bool
	}{
		{
			name: "HorizontalPodAutoscaler should be deleted when it is not used",
			args: args{
				info: &cliresource.Info{
					Name: fakeHorizontalPodAutoscaler,
					Object: &autoscalingv1.HorizontalPodAutoscaler{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindHorizontalPodAutoscaler,
						},
						Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
							ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
								APIVersion: fakeScaleTargetRefAPIVersion,
								Kind:       fakeScaleTargetRefKind,
								Name:       fakeScaleTargetRefName,
							},
						},
					},
				},
			},
			fakeObjects: []runtime.Object{},
			want:        true,
			wantErr:     false,
		},
		{
			name: "HorizontalPodAutoscaler should not be deleted when it is used",
			args: args{
				info: &cliresource.Info{
					Name:      fakeHorizontalPodAutoscaler,
					Namespace: fakeNamespace,
					Object: &autoscalingv1.HorizontalPodAutoscaler{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindHorizontalPodAutoscaler,
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      fakeHorizontalPodAutoscaler,
							Namespace: fakeNamespace,
						},
						Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
							ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
								APIVersion: fakeScaleTargetRefAPIVersion,
								Kind:       fakeScaleTargetRefKind,
								Name:       fakeScaleTargetRefName,
							},
						},
					},
				},
			},
			fakeObjects: []runtime.Object{
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						APIVersion: fakeScaleTargetRefAPIVersion,
						Kind:       fakeScaleTargetRefKind,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      fakeScaleTargetRefName,
						Namespace: fakeNamespace,
					},
				},
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, err := resource.NewFakeClient(tt.fakeObjects...)
			if err != nil {
				t.Errorf("failed to construct fake resource client")
				return
			}

			d := &determiner{
				resourceClient: c,
			}

			got, err := d.DetermineDeletion(context.Background(), tt.args.info)
			if (err != nil) != tt.wantErr {
				t.Errorf("determiner.DetermineDeletion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("determiner.DetermineDeletion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_determiner_determineUsedPodDisruptionBudget(t *testing.T) {
	const (
		fakePodDisruptionBudget = "fake-pdb"
		fakePod1                = "fake-pod1"
		fakePod2                = "fake-pod2"
		fakeLabelKey1           = "fake-label1-key"
		fakeLabelValue1         = "fake-label1-value"
		fakeLabelKey2           = "fake-label2-key"
		fakeLabelValue2         = "fake-label2-value"
	)

	type fields struct {
		pods []*corev1.Pod
	}
	type args struct {
		pdb *policyv1beta1.PodDisruptionBudget
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "used PodDisruptionBudget should be determined with MatchLabels",
			fields: fields{
				pods: []*corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								fakeLabelKey1: fakeLabelValue1,
								fakeLabelKey2: fakeLabelValue2,
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								fakeLabelKey2: fakeLabelValue2,
							},
						},
					},
				},
			},
			args: args{
				pdb: &policyv1beta1.PodDisruptionBudget{
					ObjectMeta: metav1.ObjectMeta{
						Name: fakePodDisruptionBudget,
					},
					Spec: policyv1beta1.PodDisruptionBudgetSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								fakeLabelKey1: fakeLabelValue1,
							},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "used PodDisruptionBudget should be determined with MatchExpressions",
			fields: fields{
				pods: []*corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								fakeLabelKey1: fakeLabelValue1,
							},
						},
					},
				},
			},
			args: args{
				pdb: &policyv1beta1.PodDisruptionBudget{
					ObjectMeta: metav1.ObjectMeta{
						Name: fakePodDisruptionBudget,
					},
					Spec: policyv1beta1.PodDisruptionBudgetSpec{
						Selector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      fakeLabelKey1,
									Operator: metav1.LabelSelectorOpIn,
									Values:   []string{fakeLabelValue1, fakeLabelValue2},
								},
							},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "used PodDisruptionBudget should not be determined when no Pods with corresponding label exist",
			fields: fields{
				pods: []*corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								fakeLabelKey2: fakeLabelValue2,
							},
						},
					},
				},
			},
			args: args{
				pdb: &policyv1beta1.PodDisruptionBudget{
					ObjectMeta: metav1.ObjectMeta{
						Name: fakePodDisruptionBudget,
					},
					Spec: policyv1beta1.PodDisruptionBudgetSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								fakeLabelKey1: fakeLabelValue1,
							},
						},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := &determiner{
				pods: tt.fields.pods,
			}

			got, err := d.determineUsedPodDisruptionBudget(tt.args.pdb)
			if (err != nil) != tt.wantErr {
				t.Errorf("determineUsedPodDisruptionBudget() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
func Test_determiner_determineUsedSecret(t *testing.T) {
	const (
		fakeSecret = "fake-secret"
	)
	type fields struct {
		pods []*corev1.Pod
	}
	type args struct {
		secret string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[string]struct{}
	}{
		{
			name: "secrets used in ImagePullSecret should be determined as used",
			fields: fields{
				pods: []*corev1.Pod{
					{
						Spec: corev1.PodSpec{
							ImagePullSecrets: []corev1.LocalObjectReference{{fakeSecret}}},
					},
				},
			},
			args: args{
				secret: fakeSecret,
			},
			want: map[string]struct{}{fakeSecret: {}},
		},
		{
			name: "secrets used in EnvFrom should be determined as used",
			fields: fields{
				pods: []*corev1.Pod{{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							EnvFrom: []corev1.EnvFromSource{
								{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: fakeSecret}}},
							},
						}},
					},
				}},
			},
			args: args{
				secret: fakeSecret,
			},
			want: map[string]struct{}{fakeSecret: {}},
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := &determiner{
				pods: tt.fields.pods,
			}
			got := d.detectUsedSecrets(nil)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
