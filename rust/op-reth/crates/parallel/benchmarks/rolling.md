# Rolling execution synthetic benchmark

Measured on 2026-09-29, Intel Core i9-13900 (32 logical CPUs), Linux, Rust 1.95.0,
workspace optimized test profile (`opt-level = 1`). [rolling.csv](rolling.csv) contains all
121 configurations: 11 workloads, sequential, direct window/rolling at 1/2/4/8 workers,
and window/rolling shadow at four workers. Parallel/shadow execution remains opt-in; window remains the scheduler default and sequential
remains the execution default.

## Method

Each sample executes 64 transactions, four times the 16-attempt limit. One warmup precedes
20 measured samples per configuration; the worker runtime is reused. The speculative gas budget
is 128 million and the block gas limit is 100 million so the benchmark can measure the complete
long block rather than exhausting the default speculation budget halfway through it.

Storage transactions read 256 slots each; shared cases use one contract and independent cases
use different contracts. The conflict case also increments a shared slot. Warm means the values
start in canonical revm State's cache; cold means reads reach the base. The base is an immutable
in-memory database and opening a reader clones an Arc. These cases do not model disk I/O or the
operating system page cache. Slow-head/tail cases put a 30,000-iteration EVM loop first/last in each
group of 16, with 150-iteration loops elsewhere. The expensive-policy control adds 256 chained
Keccak hashes to each ordered policy evaluation; this is synthetic work, not a production refund
formula.

Execution latency includes session creation, snapshot construction/updates, reader setup, EVM,
observations, dependency validation, retries, fee/refund settlement, reader teardown, and bundle
merging. It excludes fixture/runtime construction, pool selection, Engine RPC overhead and roots.
State/receipt root construction is timed separately in `synthetic_roots_mean_ms`; this is an
in-memory fixture root, not reth's persisted trie pipeline. Every sample asserts identical final
state, roots, receipts, gas, refunds and policy state. All direct samples made zero broker read
requests and all shadow comparisons matched. No compiler ran during the measured process.

## Four-worker latency

Milliseconds; window and rolling entries are median / p95. Ratio is window median divided by
rolling median, so values below one are regressions. These are short synthetic samples on a shared
host, not confidence intervals or a default-enablement recommendation.

| Workload | Sequential median | Direct window | Direct rolling | Window/rolling |
| --- | ---: | ---: | ---: | ---: |
| Independent compute | 6.050 | 1.953 / 2.670 | 1.957 / 2.158 | 1.00× |
| Same-sender nonce chain | 5.873 | 5.684 / 5.754 | 5.661 / 5.693 | 1.00× |
| BN254 pairing | 95.980 | 28.939 / 34.030 | 25.667 / 27.907 | 1.13× |
| Shared storage, warm | 2.095 | 3.372 / 3.971 | 2.185 / 3.173 | 1.54× |
| Independent storage, warm | 2.013 | 3.364 / 3.753 | 2.891 / 3.174 | 1.16× |
| Shared storage, cold | 2.033 | 2.870 / 3.567 | 2.146 / 2.498 | 1.34× |
| Independent storage, cold | 2.511 | 3.916 / 5.197 | 2.961 / 3.984 | 1.32× |
| Conflicting storage, cold | 2.115 | 4.624 / 6.361 | 3.526 / 6.022 | 1.31× |
| Slow window heads | 4.004 | 3.980 / 6.543 | 4.038 / 4.725 | 0.99× |
| Slow window tails | 4.059 | 10.774 / 20.798 | 3.819 / 14.120 | 2.82× |
| Expensive ordered policy | 15.938 | 13.574 / 16.659 | 11.852 / 12.768 | 1.15× |

Four-worker rolling medians improve by about 14–35% on storage cases and 13% on the
expensive-policy control compared with direct windows. Independent compute and nonce chains
are effectively unchanged; slow heads regress about 1.5%. The slow-tail median improves strongly,
but its wide p95 spread shows considerable scheduling variability. Storage cases remain generally
faster sequentially with this cheap in-memory base. More workers are not uniformly beneficial;
use the complete matrix rather than extrapolating from the four-worker comparison.

## Retained state and provider work

Base calls include worker reads plus canonical validation/retry reads; these are database-interface
calls, not physical disk reads. The table reports maximum calls per block, mean provider opens
per block, and maximum estimated retained snapshot KiB (window / rolling). Each distinct live
version pays its full conservative logical charge even where persistent nodes share memory.
Clones of the same version pay once. The version column is rolling's peak observed count.

| Workload | Base calls W / R | Opens W / R | Snapshot KiB W / R | Rolling versions |
| --- | ---: | ---: | ---: | ---: |
| Independent compute | 69 / 69 | 16.0 / 6.1 | 23.8 / 296.2 | 16 |
| Same-sender nonce chain | 5 / 5 | 0.0 / 0.0 | 19.0 / 19.0 | 1 |
| BN254 pairing | 86 / 86 | 16.0 / 4.6 | 24.0 / 300.2 | 16 |
| Shared storage, warm | 4 / 4 | 16.0 / 14.5 | 57.0 / 828.0 | 16 |
| Independent storage, warm | 69 / 69 | 16.0 / 18.3 | 2092.8 / 25005.0 | 12 |
| Shared storage, cold | 16661 / 16661 | 16.0 / 19.0 | 25.0 / 313.4 | 16 |
| Independent storage, cold | 32965 / 32965 | 16.0 / 29.8 | 40.8 / 211.3 | 10 |
| Conflicting storage, cold | 16613 / 16613 | 16.0 / 40.1 | 25.3 / 176.2 | 9 |
| Slow window heads | 69 / 69 | 16.0 / 34.6 | 24.4 / 245.7 | 13 |
| Slow window tails | 69 / 69 | 16.0 / 34.1 | 24.4 / 112.4 | 6 |
| Expensive ordered policy | 69 / 69 | 16.0 / 50.8 | 23.8 / 54.5 | 3 |

Peak RSS for the complete fixture was 23,136 KiB, measured with `/usr/bin/time -v` against
the compiled test binary, excluding Cargo/compiler processes. It is not a per-mode production RSS
measurement. Snapshot estimates count live versions and dirty tracking; they are admission limits,
not an RSS cap. Rolling can reopen readers more often because finite tasks exit when work runs out;
this benchmark's cheap Arc reader cannot establish the cost of that behavior on MDBX.

The maximum conservative snapshot charge across the matrix is 32.6 MiB. The independent warm
case is particularly sensitive to retained versions: at four workers, its estimate rises from about
2.0 MiB to 24.4 MiB. The expensive-policy case opens about 51 readers per block with rolling,
compared with 16 for windows. Shared-storage conflicts trigger 63 canonical retries per rolling
block versus 60 with windows because rolling retains attempts based on older committed prefixes.

## Waiting, retries and utilization

All timings below are mean milliseconds per block. Head wait measures waiting for the selected
rolling result; window has no separate head-wait metric because its barrier is in `window_mean_ms`.
Utilization is summed worker lease time / (block wall time × worker count); it includes reader setup
and teardown. A zero nonce-chain value reflects the intentional sequential shortcut. Timings from
concurrent workers overlap and must not be added to wall latency.

| Workload | Scheduler | Head wait | Snapshot updates | Canonical retries | Worker utilization |
| --- | --- | ---: | ---: | ---: | ---: |
| Independent compute | window | 0.000 | 0.029 | 0.000 | 78.13% |
| Independent compute | rolling | 0.723 | 0.284 | 0.000 | 92.63% |
| Same-sender nonce chain | window | 0.000 | 0.009 | 0.000 | 0.00% |
| Same-sender nonce chain | rolling | 0.000 | 0.009 | 0.000 | 0.00% |
| BN254 pairing | window | 0.000 | 0.039 | 0.000 | 91.61% |
| BN254 pairing | rolling | 24.870 | 0.191 | 0.000 | 98.22% |
| Shared storage, warm | window | 0.000 | 0.048 | 0.000 | 48.41% |
| Shared storage, warm | rolling | 0.210 | 0.229 | 0.000 | 75.53% |
| Independent storage, warm | window | 0.000 | 0.669 | 0.000 | 39.69% |
| Independent storage, warm | rolling | 0.241 | 0.871 | 0.000 | 55.17% |
| Shared storage, cold | window | 0.000 | 0.030 | 0.000 | 54.08% |
| Shared storage, cold | rolling | 0.170 | 0.219 | 0.000 | 75.59% |
| Independent storage, cold | window | 0.000 | 0.046 | 0.000 | 39.46% |
| Independent storage, cold | rolling | 0.268 | 0.209 | 0.000 | 53.39% |
| Conflicting storage, cold | window | 0.000 | 0.032 | 1.678 | 39.84% |
| Conflicting storage, cold | rolling | 0.137 | 0.227 | 1.919 | 43.64% |
| Slow window heads | window | 0.000 | 0.032 | 0.000 | 30.12% |
| Slow window heads | rolling | 3.261 | 0.197 | 0.000 | 31.12% |
| Slow window tails | window | 0.000 | 0.079 | 0.000 | 26.93% |
| Slow window tails | rolling | 4.360 | 0.291 | 0.000 | 30.64% |
| Expensive ordered policy | window | 0.000 | 0.034 | 0.000 | 12.57% |
| Expensive ordered policy | rolling | 0.173 | 0.108 | 0.000 | 13.30% |

The CSV also includes EVM, queue waiting, provider open/read, dependency-validation,
finalization, policy evaluation, drain and root timings. Timing columns ending in `_mean_ms`
average the 20 measured samples; runtime counters are totals over all 21 samples including warmup.
`provider_reads` and snapshot/version counts are maxima. Counter totals expose admitted work,
completed work, conflicts, reused outcomes, provider opens and read amplification.

## Correctness and deployment gate

Validation completed with 542 passing execution/integration/CLI tests, plus this benchmark.
Formatting, Clippy with warnings denied, rustdoc with warnings denied, and affected `no_std`
builds for `riscv32imac-unknown-none-elf` passed.

The correctness suite exercises both schedulers, broker fallback, direct and shadow execution at
1/2/4/8 workers. Deterministic channel tests cover early commit/refill with a slow tail, refreshed
snapshots, canonical progress outside the preview, fair competing builds, retained prepared/cancelled
attempts, source failures, reader teardown, cancellation and snapshot-budget fallback. Integration
coverage includes historical batches, Engine parent overlays, subblocks, stateful policy/refunds,
shrinking previews, skipped/invalid descendants and selector callback ordering.

These measurements are synthetic. Real-chain replay was not run: no target database and block
range were supplied. Before recommending rolling operationally, replay representative ranges with
both schedulers from the same parent state. Include physical database I/O and read amplification,
state-root work, pool selection, Engine validation/build latency, contention, persistence/reorgs,
latency distributions and process memory. Keep the sequential and window defaults unchanged.

Reproduce from `rust/`:

```sh
mise exec -- cargo nextest run -p alloy-op-evm --features parallel,metrics --locked \
  --run-ignored only --nocapture -E 'test(benchmark_rolling_execution)'
```
