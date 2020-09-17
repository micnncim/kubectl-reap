package determiner

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/resource"
)

func Test_determiner_determinePrune(t *testing.T) {
	const (
		fakeConfigMap             = "fake-cm"
		fakeSecret                = "fake-secret"
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
		info *resource.Info
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "configmap should be pruned when it is used",
			fields: fields{
				usedConfigMaps: map[string]struct{}{
					fakeConfigMap: {},
				},
				pods: []*corev1.Pod{
					{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									EnvFrom: []corev1.EnvFromSource{
										{
											ConfigMapRef: &corev1.ConfigMapEnvSource{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: fakeConfigMap,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				info: &resource.Info{
					Object: &corev1.ConfigMap{
						TypeMeta: metav1.TypeMeta{
							Kind: kindConfigMap,
						},
					},
					Name: fakeConfigMap,
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "configmap should not be pruned when it is not used",
			args: args{
				info: &resource.Info{
					Object: &corev1.ConfigMap{
						TypeMeta: metav1.TypeMeta{
							Kind: kindConfigMap,
						},
					},
					Name: fakeConfigMap,
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "secret should be pruned when it is used",
			fields: fields{
				usedSecrets: map[string]struct{}{
					fakeSecret: {},
				},
				pods: []*corev1.Pod{
					{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									EnvFrom: []corev1.EnvFromSource{
										{
											SecretRef: &corev1.SecretEnvSource{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: fakeSecret,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				info: &resource.Info{
					Object: &corev1.Secret{
						TypeMeta: metav1.TypeMeta{
							Kind: kindSecret,
						},
					},
					Name: fakeSecret,
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "secret should not be pruned when it is not used",
			args: args{
				info: &resource.Info{
					Object: &corev1.Secret{
						TypeMeta: metav1.TypeMeta{
							Kind: kindSecret,
						},
					},
					Name: fakeSecret,
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "pvc should be pruned when it is used",
			fields: fields{
				usedPersistentVolumes: map[string]struct{}{
					fakePersistentVolumeClaim: {},
				},
				pods: []*corev1.Pod{
					{
						Spec: corev1.PodSpec{
							Volumes: []corev1.Volume{
								{
									VolumeSource: corev1.VolumeSource{
										PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
											ClaimName: fakePersistentVolumeClaim,
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				info: &resource.Info{
					Object: &corev1.PersistentVolume{
						TypeMeta: metav1.TypeMeta{
							Kind: kindPersistentVolumeClaim,
						},
					},
					Name: fakePersistentVolumeClaim,
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "pvc should not be pruned when it is not used",
			args: args{
				info: &resource.Info{
					Object: &corev1.PersistentVolume{
						TypeMeta: metav1.TypeMeta{
							Kind: kindPersistentVolumeClaim,
						},
					},
					Name: fakePersistentVolumeClaim,
				},
			},
			want:    true,
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

			got, err := d.DeterminePrune(tt.args.info)
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

func Test_determineUsedPodDisruptionBudget(t *testing.T) {
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
