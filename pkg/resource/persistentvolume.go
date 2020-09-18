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
	corev1 "k8s.io/api/core/v1"
)

const (
	storageObjectInUseProtection = "StorageObjectInUseProtection"
)

// CheckVolumeSatisfyClaim checks if the volume requested by the claim satisfies the requirements of the claim.
func CheckVolumeSatisfyClaim(volume *corev1.PersistentVolume, claim *corev1.PersistentVolumeClaim) bool {
	if !checkCapacitySatisfyRequest(volume, claim) {
		return false
	}

	if !checkStorageClassMatch(volume, claim) {
		return false
	}

	if !checkVolumeModeMatch(volume, claim) {
		return false
	}

	if !checkAccessModesMatch(volume, claim) {
		return false
	}

	return true
}

// checkCapacitySatisfyRequest returns true if PV's capacity satisfies the PVC's requested resource.
func checkCapacitySatisfyRequest(volume *corev1.PersistentVolume, claim *corev1.PersistentVolumeClaim) bool {
	volumeQty := volume.Spec.Capacity[corev1.ResourceStorage]
	volumeSize := volumeQty.Value()

	requestedQty := claim.Spec.Resources.Requests[corev1.ResourceName(corev1.ResourceStorage)]
	requestedSize := requestedQty.Value()

	return volumeSize > requestedSize
}

// checkStorageClassMatch returns true if PV satisfies the PVC's requested StrageClass.
func checkStorageClassMatch(volume *corev1.PersistentVolume, claim *corev1.PersistentVolumeClaim) bool {
	var (
		requestedClass string
		pvVolumeClass  string
		ok             bool
	)

	// Use beta annotation first
	requestedClass, ok = claim.Annotations[corev1.BetaStorageClassAnnotation]
	if !ok && claim.Spec.StorageClassName != nil {
		requestedClass = *claim.Spec.StorageClassName
	}

	// Use beta annotation first
	pvVolumeClass, ok = volume.Annotations[corev1.BetaStorageClassAnnotation]
	if !ok {
		pvVolumeClass = volume.Spec.StorageClassName
	}

	return requestedClass == pvVolumeClass
}

// checkVolumeModeMatch returns true if PV satisfies the PVC's requested VolumeMode.
func checkVolumeModeMatch(volume *corev1.PersistentVolume, claim *corev1.PersistentVolumeClaim) bool {
	// In HA upgrades, we cannot guarantee that the apiserver is on a version >= controller-manager.
	// So we default a nil volumeMode to filesystem
	requestedVolumeMode := corev1.PersistentVolumeFilesystem
	if claim.Spec.VolumeMode != nil {
		requestedVolumeMode = *claim.Spec.VolumeMode
	}

	pvVolumeMode := corev1.PersistentVolumeFilesystem
	if volume.Spec.VolumeMode != nil {
		pvVolumeMode = *volume.Spec.VolumeMode
	}

	return requestedVolumeMode == pvVolumeMode
}

// checkAccessModesMatch returns true if PV satisfies all the PVC's requested AccessModes.
func checkAccessModesMatch(volume *corev1.PersistentVolume, claim *corev1.PersistentVolumeClaim) bool {
	pvAccessModes := make(map[corev1.PersistentVolumeAccessMode]struct{})
	for _, mode := range volume.Spec.AccessModes {
		pvAccessModes[mode] = struct{}{}
	}

	for _, mode := range claim.Spec.AccessModes {
		if _, ok := pvAccessModes[mode]; !ok {
			return false
		}
	}

	return true
}
