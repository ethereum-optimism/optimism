# PR3: Enhanced Metrics with Prometheus

## Summary
This PR adds enhanced metrics capabilities to op-node for better monitoring and observability using Prometheus.

## Changes
- Added `EnhancedMetrics` struct extending base Metrics with:
  - L1/L2 request latency tracking
  - Config reload duration tracking
  - Resource utilization metrics (memory, CPU, goroutines)
  - Health check metrics
  - Performance metrics (blocks/sec, transactions/sec)
  - Error rate metrics (current, 1min, 5min)
- Added helper methods for recording metrics
- Improved monitoring capabilities for production environments

## Benefits
- Better observability of node performance
- Enhanced debugging capabilities
- Production-ready monitoring
- Better resource utilization tracking

## Files Changed
- `op-node/metrics/enhanced_metrics.go` (new file)

