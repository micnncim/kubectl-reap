package cmd

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/resource"
)

func Test_determiner_determinePrune(t *testing.T) {
	const (
		fakeConfigMap             = "fake-cm"
		fakeSecret                = "fake-secret"
		fakePersistentVolumeClaim = "fake-pvc"
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

			d := &determiner{
				usedConfigMaps:             tt.fields.usedConfigMaps,
				usedSecrets:                tt.fields.usedSecrets,
				usedPersistentVolumeClaims: tt.fields.usedPersistentVolumes,
				pods:                       tt.fields.pods,
			}

			got, err := d.determinePrune(tt.args.info)
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
