# [metrics/metrics.go](./metrics.go)

A custom wrapper over the Prometheus Go client library for monitoring and metric collection. Provides an API to record various metrics related to the operation of the indexer.

- **Overview**

- Metrics struct: It holds various Prometheus metrics objects like GaugeVec, CounterVec, Counter, Gauge, and SummaryVec. These objects are used to record different types of metrics.

- NewMetrics: This function creates a new Metrics struct and initializes all the Prometheus metric objects with appropriate names, namespaces, and help strings.

- RecordDeposit, RecordWithdrawal, RecordStateBatches, SetL1CatchingUp, SetL2CatchingUp, SetL1SyncPercent, SetL2SyncPercent, IncL1CachedTokensCount, IncL2CachedTokensCount, RecordHTTPRequest, RecordHTTPResponse are methods that record/update the corresponding metric.

- Serve: This method starts an HTTP server that serves the current state of all metrics in the Prometheus exposition format at the "/metrics" endpoint. The metrics can then be scraped by a Prometheus server.

The metrics tracked here are related to:

Sync status of L1 and L2 chains (height, catching up state, sync percentage).
Transaction metrics (deposits, withdrawals, state batches).
Cached tokens count.
HTTP request/response count and durations.

- **What is Prometheus?**

Prometheus provides a powerful platform for understanding how the indexer is performing. It is a popular standard for instrumenting code with metrics, allowing you to visualize and alert on problems. It's a powerful tool for understanding the behavior of your system.

