package v1alpha1

import (
	"github.com/djzager/operator-status/conditions"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KNIClusterSpec defines the desired state of KNICluster
// +k8s:openapi-gen=true
type KNIClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
}

// KNIClusterStatus defines the observed state of KNICluster
// +k8s:openapi-gen=true
type KNIClusterStatus struct {
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	// conditions describes the state of the operator's reconciliation functionality.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +optional

	// Conditions is a list of conditions related to operator reconciliation
	Conditions []conditions.Condition `json:"conditions,omitempty"  patchStrategy:"merge" patchMergeKey:"type"`
	// RelatedObjects is a list of objects that are "interesting" or related to this operator.
	RelatedObjects []corev1.ObjectReference `json:"relatedObjects,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KNICluster is the Schema for the kniclusters API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type KNICluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KNIClusterSpec   `json:"spec,omitempty"`
	Status KNIClusterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KNIClusterList contains a list of KNICluster
type KNIClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KNICluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KNICluster{}, &KNIClusterList{})
}
