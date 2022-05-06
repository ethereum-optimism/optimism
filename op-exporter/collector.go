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
	opExporterVersion = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "op_exporter_version",
			Help: "Verion of the op-exporter software"},
		[]string{"version", "commit", "goVersion", "buildDate"},
	)
)

func init() {
	//Register metrics with prometheus
	prometheus.MustRegister(gasPrice)
	prometheus.MustRegister(blockNumber)
	prometheus.MustRegister(healthySequencer)
	prometheus.MustRegister(opExporterVersion)

}
