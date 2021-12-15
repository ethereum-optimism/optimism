package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

//Define the metrics we wish to expose
var (
	gasPrice = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "op_gasPrice",
			Help: "Gas price."},
		[]string{"network", "layer"},
	)
	blockNumber = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "op_blocknumber",
			Help: "Current block number."},
		[]string{"network", "layer"},
	)
	healthySequencer = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "op_healthy_sequencer",
			Help: "Is the sequencer healthy?"},
		[]string{"network"},
	)
)

func init() {
	//Register metrics with prometheus
	prometheus.MustRegister(gasPrice)
	prometheus.MustRegister(blockNumber)
	prometheus.MustRegister(healthySequencer)
}
