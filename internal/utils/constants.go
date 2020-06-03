package utils

const (
	ControllerName = "kotary-controller"

	MessageRejectedMemory = "Not enough Memory claiming %s but %s currently available"
	MessageRejectedCPU    = "Not enough CPU claiming %s but %s currently available"

	MessageMemoryAllocationLimit = "Exceeded Memory allocation limit claiming %s but limited to %s"
	MessageCpuAllocationLimit    = "Exceeded CPU allocation limit claiming %s but limited to %s"

	MessagePendingMemoryDownscale = "Awaiting lower Memory consumption claiming %s but current total of request is %s"
	MessagePendingCpuDownscale    = "Awaiting lower CPU consumption claiming %s but current total of CPU request is %s"

	ResourceQuotaName = "managed-quota"

	EmptyMsg = ""
)
