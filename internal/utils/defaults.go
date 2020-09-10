package utils

import (
	"k8s.io/apimachinery/pkg/labels"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func DefaultLabelSelector() (selector labels.Selector) {
	selector, _ = metav1.LabelSelectorAsSelector(&metav1.LabelSelector{})
	return
}

func DefaultDeleteOptions() *metav1.DeleteOptions {
	return &metav1.DeleteOptions{
		TypeMeta:           metav1.TypeMeta{},
		GracePeriodSeconds: nil,
		Preconditions:      nil,
		PropagationPolicy:  nil,
		DryRun:             nil,
	}
}
