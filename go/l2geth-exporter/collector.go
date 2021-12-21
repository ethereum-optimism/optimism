package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

//Define the metrics we wish to expose
var (
	ctcTotalElements = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "l2geth_ctc_total_elements",
			Help: "CTC GetTotalElements value."},
		[]string{"state"},
	)
	ctcTotalElementsCallSuccess = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "l2geth_ctc_total_elements_call_success",
			Help: "CTC GetTotalElements call success."},
	)
)

func init() {
	//Register metrics with prometheus
	prometheus.MustRegister(ctcTotalElements)
	prometheus.MustRegister(ctcTotalElementsCallSuccess)
}
