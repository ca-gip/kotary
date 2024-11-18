package utils

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var ClaimCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "rq_operator_claims",
	Help: "Number of claims",
}, []string{"status"})

var DefaultClaimCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "rq_operator_default_claims",
	Help: "Number of default claims created",
}, []string{"status"})

var RatioMaxAllocationCPUGauge = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "kotary_ratio_max_allocation_cpu",
	Help: "Maximum CPU allocation ratio allowed per namespace compared to total cluster CPU",
})

var RatioMaxAllocationMemoryGauge = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "kotary_ratio_max_allocation_memory",
	Help: "Maximum memory allocation ratio allowed per namespace compared to total cluster memory",
})

var RatioOverCommitCPUGauge = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "kotary_ratio_over_commit_cpu",
	Help: "CPU over-commit ratio applied to node available resources (percentage)",
})

var RatioOverCommitMemoryGauge = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "kotary_ratio_over_commit_memory",
	Help: "Memory over-commit ratio applied to node available resources (percentage)",
})
