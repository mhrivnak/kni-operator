package objectreferences

import (
	corev1 "k8s.io/api/core/v1"
)

// SetObjectReference - updates list of object references based on newObject
func SetObjectReference(objects *[]corev1.ObjectReference, newObject corev1.ObjectReference) {
	if objects == nil {
		objects = &[]corev1.ObjectReference{}
	}
	existingObject := FindObjectReference(*objects, newObject)
	if existingObject == nil {
		*objects = append(*objects, newObject)
		return
	}
}

// FindObjectReference - finds an ObjectReference in a slice of objects
func FindObjectReference(objects []corev1.ObjectReference, object corev1.ObjectReference) *corev1.ObjectReference {
	for i := range objects {
		if objects[i].APIVersion == object.APIVersion && objects[i].Kind == object.Kind {
			return &objects[i]
		}
	}

	return nil
}
