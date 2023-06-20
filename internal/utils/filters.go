package utils

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/cmd/kubeadm/app/constants"
)

// Predicate to filter out un-schedulable, role master labeled or not ready nodes
func FilterWorkerNode() func(*v1.Node) bool {
	return func(node *v1.Node) bool {
		// Unschedulable Node
		if node.Spec.Unschedulable {
			return false
		}
		// Role Master Label
		if _, hasMasterRoleLabel := node.Labels[constants.LabelNodeRoleControlPlane]; hasMasterRoleLabel {
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

func FilterRunningPods(pods []*v1.Pod) []*v1.Pod {
	var filteredPods []*v1.Pod
	for _, pod := range pods {
		if pod.Status.Phase == "Running" {
			filteredPods = append(filteredPods, pod)
		}
	}
	return filteredPods
}

func FilterNodesWithPredicate(ss []*v1.Node, test func(*v1.Node) bool) (ret []*v1.Node) {
	for _, s := range ss {
		if test(s) {
			ret = append(ret, s)
		}
	}
	return
}
