package handlers

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const prometheusNamespace = "ionscale"

var (
	connectedDevices = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: prometheusNamespace,
		Name:      "connected_machines_total",
		Help:      "Total amount of connected machines",
	}, []string{"tailnet"})
)
