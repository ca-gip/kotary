package utils

import (
	v1 "k8s.io/api/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"
	"k8s.io/kubernetes/cmd/kubeadm/app/constants"
)

// Predicate to filter out un-schedulable, role master labeled or not ready nodes
func FilterWorkerNode() corelisters.NodeConditionPredicate {
	return func(node *v1.Node) bool {
		// Unschedulable Node
		if node.Spec.Unschedulable {
			return false
		}
		// Role Master Label
		if _, hasMasterRoleLabel := node.Labels[constants.LabelNodeRoleMaster]; hasMasterRoleLabel {
			return false
		}
		// No conditions available
		if len(node.Status.Conditions) == 0 {
			return false
		}
		for _, cond := range node.Status.Conditions {
			// We consider the node only when its NodeReady condition status is ConditionTrue
			if cond.Type == v1.NodeReady && cond.Status != v1.ConditionTrue {
				klog.V(4).Infof("Ignoring node %v with %v condition status %v", node.Name, cond.Type, cond.Status)
				return false
			}
		}
		return true
	}
}
