package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

//Define the metrics we wish to expose
var (
	addressTotalElements = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "l2geth_total_elements",
			Help: "GetTotalElements value."},
		[]string{"state", "address"},
	)
	addressTotalElementsCallStatus = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "l2geth_total_elements_call_status",
			Help: "GetTotalElements call status."},
		[]string{"status", "address"},
	)
)

func init() {
	//Register metrics with prometheus
	prometheus.MustRegister(addressTotalElements)
	prometheus.MustRegister(addressTotalElementsCallStatus)
}
