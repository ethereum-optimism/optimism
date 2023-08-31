package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

func main() {
	// Assuming you have some metric like this:
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "go_counter",
		Help: "Description of my counter",
	})

	// Register it locally (even if you're not exposing/serving it)
	prometheus.MustRegister(counter)

	// Increment the counter or perform some operations
	counter.Inc()

	// Push it to the Push Gateway
	if err := push.New("http://pushgateway:9091", "job_name").
		Collector(counter).
		Push(); err != nil {
		// Handle error
	}
}
