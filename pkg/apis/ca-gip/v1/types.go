package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResourceQuotaClaim defines a request modify a ResourcesQuota
type ResourceQuotaClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status ResourceQuotaClaimStatus `json:"status,omitempty"`
	Spec   corev1.ResourceList      `json:"spec,omitempty"`
}

const (
	PhaseAccepted = "ACCEPTED"
	PhaseRejected = "REJECTED"
	PhasePending  = "PENDING"
)

// ResourceQuotaClaimStatus defines the observed state of ResourceQuotaClaim
type ResourceQuotaClaimStatus struct {
	Phase   string `json:"phase,omitempty"`
	Details string `json:"details,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResourceQuotaClaimList contains a list of ResourceQuotaClaim
type ResourceQuotaClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ResourceQuotaClaim `json:"items"`
}
