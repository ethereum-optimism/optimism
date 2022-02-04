package metrics

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// RunServer spins up a prometheus metrics server at the provided hostname and
// port.
//
// NOTE: This method MUST be run as a goroutine.
func RunServer(hostname string, port uint64) {
	metricsPortStr := strconv.FormatUint(port, 10)
	metricsAddr := fmt.Sprintf("%s:%s", hostname, metricsPortStr)

	http.Handle("/metrics", promhttp.Handler())
	_ = http.ListenAndServe(metricsAddr, nil)
}
