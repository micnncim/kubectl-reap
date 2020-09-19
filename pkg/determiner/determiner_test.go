package determiner

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliresource "k8s.io/cli-runtime/pkg/resource"
)

func TestDeterminer_DetermineDeletion(t *testing.T) {
	const (
		fakeConfigMap             = "fake-cm"
		fakeSecret                = "fake-secret"
		fakePod                   = "fake-pod"
		fakePersistentVolumeClaim = "fake-pvc"
		fakePodDisruptionBudget   = "fake-pdb"
		fakeLabelKey1             = "fake-label1-key"
		fakeLabelValue1           = "fake-label1-value"
		fakeLabelKey2             = "fake-label2-key"
		fakeLabelValue2           = "fake-label2-value"
	)

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
			name: "ConfigMap should be deleted when it is not used",
			args: args{
				info: &cliresource.Info{
					Name: fakeConfigMap,
					Object: &corev1.ConfigMap{
						TypeMeta: metav1.TypeMeta{
							Kind: kindConfigMap,
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
							Kind: kindConfigMap,
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
							Kind: kindSecret,
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
							Kind: kindSecret,
						},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "Pod should be deleted when it is not running",
			args: args{
				info: &cliresource.Info{
					Name: fakePod,
					Object: &corev1.Pod{
						TypeMeta: metav1.TypeMeta{
							Kind: kindPod,
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
							Kind: kindPod,
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
			name: "PersistentVolumeClaim should be deleted when it is not used",
			args: args{
				info: &cliresource.Info{
					Name: fakePersistentVolumeClaim,
					Object: &corev1.PersistentVolume{
						TypeMeta: metav1.TypeMeta{
							Kind: kindPersistentVolumeClaim,
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
					Object: &corev1.PersistentVolume{
						TypeMeta: metav1.TypeMeta{
							Kind: kindPersistentVolumeClaim,
						},
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
							Kind: kindPodDisruptionBudget,
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
							Kind: kindPodDisruptionBudget,
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

			d := &Determiner{
				UsedConfigMaps:             tt.fields.usedConfigMaps,
				UsedSecrets:                tt.fields.usedSecrets,
				UsedPersistentVolumeClaims: tt.fields.usedPersistentVolumes,
				Pods:                       tt.fields.pods,
			}

			got, err := d.DetermineDeletion(context.Background(), tt.args.info)
			if (err != nil) != tt.wantErr {
				t.Errorf("determiner.determinePrune() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("determiner.determinePrune() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeterminer_determineUsedPodDisruptionBudget(t *testing.T) {
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

			d := &Determiner{
				Pods: tt.fields.pods,
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
