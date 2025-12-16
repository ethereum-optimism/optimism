# Pipeline Metrics Benchmarks

## System Information
- **OS:** Linux
- **Architecture:** amd64
- **CPU:** Intel(R) Core(TM) i9-9900K CPU @ 3.60GHz

## Benchmark Results

| Benchmark | Iterations | ns/op |
|-----------|------------|-------|
| RecordStageProcessing-16 | 13,425,480 | 92.32 |
| RecordStageQueueDepth-16 | 25,175,820 | 46.95 |
| RecordAllMetrics-16 | 4,543,458 | 268.6 |

## Summary

- **RecordStageProcessing:** ~92 ns per operation (10.8M ops/sec)
- **RecordStageQueueDepth:** ~47 ns per operation (21.3M ops/sec)
- **RecordAllMetrics:** ~269 ns per operation (3.7M ops/sec)

All benchmarks passed successfully in **4.094s**.

---
*Package: `github.com/ethereum-optimism/optimism/op-node/rollup/driver`*

