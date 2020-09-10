package utils

import (
	"k8s.io/apimachinery/pkg/api/resource"

	underscore "github.com/ahl5esoft/golang-underscore"
	v1 "k8s.io/api/core/v1"
)

func PodsCpuRequest(pods []*v1.Pod) (result int64) {
	underscore.
		Chain(pods).
		Map(func(pod v1.Pod, _ int) []v1.Container { return pod.Spec.Containers }).
		Map(func(containers []v1.Container, _ int) (cpuReq int64) {
			underscore.
				Chain(containers).
				Aggregate(int64(0), func(acc int64, cur v1.Container, _ int) int64 { return acc + cur.Resources.Requests.Cpu().MilliValue() }).
				Value(&cpuReq)
			return
		}).
		Aggregate(int64(0), func(acc int64, cur int64, _ int) int64 { return acc + cur }).
		Value(&result)

	return
}

func PodsMemRequest(pods []*v1.Pod) (result int64) {
	underscore.
		Chain(pods).
		Map(func(pod v1.Pod, _ int) []v1.Container { return pod.Spec.Containers }).
		Map(func(containers []v1.Container, _ int) (memReq int64) {
			underscore.
				Chain(containers).
				Aggregate(int64(0), func(acc int64, cur v1.Container, _ int) int64 { return acc + cur.Resources.Requests.Memory().Value() }).
				Value(&memReq)
			return
		}).
		Aggregate(int64(0), func(acc int64, cur int64, _ int) int64 { return acc + cur }).
		Value(&result)

	return
}

func TotalRequestNS(pods []*v1.Pod) *v1.ResourceList {
	return &v1.ResourceList{
		v1.ResourceCPU:    *resource.NewMilliQuantity(PodsCpuRequest(pods), resource.DecimalSI),
		v1.ResourceMemory: *resource.NewQuantity(PodsMemRequest(pods), resource.BinarySI),
	}
}

func NodesCpuAllocatable(workerNodes []*v1.Node) (result int64) {
	underscore.
		Chain(workerNodes).
		Map(func(node v1.Node, _ int) int64 { return node.Status.Allocatable.Cpu().MilliValue() }).
		Aggregate(int64(0), func(acc int64, cur int64, _ int) int64 { return acc + cur }).
		Value(&result)
	return
}

func NodesMemAllocatable(workerNodes []*v1.Node) (result int64) {
	underscore.
		Chain(workerNodes).
		Map(func(node v1.Node, _ int) int64 { return node.Status.Allocatable.Memory().Value() }).
		Aggregate(int64(0), func(acc int64, cur int64, _ int) int64 { return acc + cur }).
		Value(&result)
	return
}
