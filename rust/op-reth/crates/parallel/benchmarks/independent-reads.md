# Independent-read synthetic benchmark

Measured on 2026-09-29, Intel Core i9-13900 (32 logical CPUs), Rust 1.95.0, Linux,
workspace optimized test profile (`opt-level = 1`). The complete matrix and counters are in
[independent-reads.csv](independent-reads.csv): sequential, broker/direct at 1/2/4/8 workers,
and both shadow backends at four workers.

Each sample executes eight transactions with the stateful fixture policy. One warmup is discarded;
20 samples are measured per configuration. The worker runtime is reused. Storage cases read 256
slots per transaction. Shared cases use one contract; independent cases use separate contracts.
The conflicting case also increments a shared slot, forcing seven canonical retries per sample.
Warm means slots/code are initially loaded in canonical revm State; cold means the values must be
read from the base. These labels do not describe the operating system's page cache.

The base is an immutable in-memory database, and opening a reader clones an Arc. Timings start
before session construction and snapshot capture, and include reader setup, EVM execution,
observations, dependency validation, ordered fees/refunds, retries and bundle merging. Fixture
construction, pool selection, root construction, Engine RPC overhead and disk/MDBX I/O are excluded.
The fixture asserts identical state, receipts, gas and refunds for every sample.

## Four-worker latency

Times are milliseconds. Each broker/direct entry is median / p95. Sequential entries are medians.

| Workload | Sequential | Broker | Direct | Broker/direct median ratio |
| --- | ---: | ---: | ---: | ---: |
| Independent compute | 0.895 | 0.397 / 0.475 | 0.341 / 0.390 | 1.16× |
| Same-sender nonce chain | 0.738 | 0.708 / 0.716 | 0.707 / 0.718 | 1.00× |
| BN254 pairing | 11.721 | 3.950 / 5.005 | 3.799 / 4.947 | 1.04× |
| Shared storage, warm | 0.307 | 3.024 / 3.568 | 0.502 / 0.518 | 6.02× |
| Independent storage, warm | 0.242 | 2.853 / 3.081 | 0.488 / 0.602 | 5.85× |
| Shared storage, cold | 0.262 | 2.824 / 2.889 | 0.477 / 0.560 | 5.92× |
| Independent storage, cold | 0.285 | 2.742 / 2.833 | 0.479 / 0.559 | 5.72× |
| Conflicting storage, cold | 0.257 | 3.120 / 3.505 | 0.615 / 0.643 | 5.07× |

Direct execution removes the broker message cost. These short in-memory storage transactions are
still faster sequentially than either parallel mode. Nonce-chain production correctly avoids
speculation. Shadow nonce chains still pay for a singleton reader and comparison on every
transaction; direct reads can increase that diagnostic overhead. More workers are not uniformly
better, especially with contention. All shadow comparisons matched; all successful direct paths
made zero broker read requests.

## Memory and read amplification

`provider_reads` counts calls to the immutable base reader by both speculative workers and the
canonical path, including validation and retries. It excludes snapshot/physical cache hits and is
the maximum per eight-transaction sample. These are database-interface calls, not physical disk
reads. Counter columns in the CSV are totals over all 21 iterations including warmup.
`snapshot_bytes` is the maximum estimated retained session data, including dirty tracking, across
the samples; it is an admission estimate, not heap/RSS measurement.

| Workload | Broker base calls | Direct base calls | Amplification | Direct snapshot KiB |
| --- | ---: | ---: | ---: | ---: |
| Independent compute | 5 | 13 | 2.60× | 6.76 |
| Same-sender nonce chain | 5 | 5 | 1.00× | 5.01 |
| BN254 pairing | 6 | 22 | 3.67× | 6.76 |
| Shared storage, warm | 4 | 4 | 1.00× | 40.00 |
| Independent storage, warm | 5 | 13 | 2.60× | 267.75 |
| Shared storage, cold | 261 | 2317 | 8.88× | 6.38 |
| Independent storage, cold | 2061 | 4125 | 2.00× | 9.75 |
| Conflicting storage, cold | 261 | 2317 | 8.88× | 6.50 |

Peak process RSS for the complete benchmark fixture, measured with `/usr/bin/time -v` against the
compiled test binary (excluding Cargo/compiler processes), was 15,240 KiB. This is not a per-mode
production memory estimate. The configured 64 MiB session limit is separately tested for initial
capture and dirty-entry exhaustion; fallback releases the snapshot and preserves public hooks.

Cold shared reads show substantial amplification: isolated workers each access the base, and
ordered validation also reads it to populate canonical state. Engine's shared physical cache can
reduce this, but an in-memory benchmark cannot establish the cost on a real database.

## Replay status

No target-chain database and replay range were supplied. End-to-end replay results are therefore
unmeasured. Before deployment, record chain ID, parent hash, database checkpoint, block interval,
cache policy and worker allocation, then replay that identical range sequentially, in shadow mode,
and with broker/direct parallel reads. Compare roots, receipts and refund payloads, and measure
latency, RSS, database-read amplification, validation/finalization and state-root time. Include
high-contention blocks, reorgs and persistence of in-memory ancestors. Parallel mode remains opt-in.
