/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resource

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestCheckVolumeSatisfyClaim(t *testing.T) {
	var (
		fakeResourceQtyLow  = "1Gi"
		fakeResourceQtyHigh = "2Gi"
		fakeStorageClass1   = "standard"
		fakeStorageClass2   = "slow"
		fakeVolumeMode      = corev1.PersistentVolumeFilesystem
	)

	type args struct {
		volume *corev1.PersistentVolume
		claim  *corev1.PersistentVolumeClaim
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "PersistentVolume should satisfy PersistentVolumeClaim",
			args: args{
				volume: &corev1.PersistentVolume{
					Spec: corev1.PersistentVolumeSpec{
						Capacity: corev1.ResourceList{
							corev1.ResourceName(corev1.ResourceStorage): resource.MustParse(fakeResourceQtyHigh),
						},
						StorageClassName: fakeStorageClass1,
						VolumeMode:       &fakeVolumeMode,
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
							corev1.ReadWriteMany,
						},
					},
				},
				claim: &corev1.PersistentVolumeClaim{
					Spec: corev1.PersistentVolumeClaimSpec{
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceName(corev1.ResourceStorage): resource.MustParse(fakeResourceQtyLow),
							},
						},
						StorageClassName: &fakeStorageClass1,
						VolumeMode:       &fakeVolumeMode,
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
					},
				},
			},
			want: true,
		},
		{
			name: "PersistentVolume with different StorageClass should not satisfy PersistentVolumeClaim",
			args: args{
				volume: &corev1.PersistentVolume{
					Spec: corev1.PersistentVolumeSpec{
						Capacity: corev1.ResourceList{
							corev1.ResourceName(corev1.ResourceStorage): resource.MustParse(fakeResourceQtyHigh),
						},
						StorageClassName: fakeStorageClass1,
						VolumeMode:       &fakeVolumeMode,
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
							corev1.ReadWriteMany,
						},
					},
				},
				claim: &corev1.PersistentVolumeClaim{
					Spec: corev1.PersistentVolumeClaimSpec{
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceName(corev1.ResourceStorage): resource.MustParse(fakeResourceQtyLow),
							},
						},
						StorageClassName: &fakeStorageClass2,
						VolumeMode:       &fakeVolumeMode,
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := CheckVolumeSatisfyClaim(tt.args.volume, tt.args.claim); got != tt.want {
				t.Errorf("CheckVolumeSatisfyClaim() = %v, want %v", got, tt.want)
			}
		})
	}
}
