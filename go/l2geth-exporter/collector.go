package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

//Define the metrics we wish to expose
var (
	ovmctcTotalElements = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ovmctc_total_elements",
			Help: "OVM CTC GetTotalElements value."},
		[]string{"state"},
	)
)

func init() {
	//Register metrics with prometheus
	prometheus.MustRegister(ovmctcTotalElements)
}
