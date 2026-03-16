package sysgo

import "fmt"

const dockerToLocalHost = "host.docker.internal"

type L2MetricsRegistrar interface {
	RegisterL2MetricsTargets(serviceName string, endpoints ...PrometheusMetricsTarget)
}

type PrometheusMetricsTarget string

func NewPrometheusMetricsTarget(host string, port string, isRunningInDocker bool) PrometheusMetricsTarget {
	if !isRunningInDocker {
		host = dockerToLocalHost
	}
	return PrometheusMetricsTarget(fmt.Sprintf("%s:%s", host, port))
}
