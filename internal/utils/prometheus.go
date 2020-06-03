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
