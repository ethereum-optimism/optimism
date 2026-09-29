# Performance across the optimistic-execution session

This report separates measured effects of each optimization from the final execution comparison.
The final executor scales compute and stock-precompile workloads well. Independent readers remove
the broker bottleneck in storage workloads. Rolling scheduling improves overlap and reduces window
barriers, with extra snapshot work, retained versions and sometimes reader opens. Cheap in-memory
storage, contention, serial policies and tail latency still limit the result.

## What “before” and “after” mean

There is no end-to-end measurement of the untouched node from before this session. The strongest
controlled overall comparison is the final sequential reference versus final parallel execution
using the same fixture and run. Earlier records isolate stages of the implementation:

1. The initial broker/window prototype, before sender filtering and stock-precompile reuse.
2. The optimized broker/window checkpoint, commit `daa143adca`.
3. Independent readers versus broker reads, measured on eight-transaction blocks.
4. Rolling versus direct windows, measured on 64-transaction blocks.

The original prototype and checkpoint numbers below come from the benchmark record in
`rust/op-reth/crates/parallel/README.md` at `daa143adca`. The later data is preserved in
[independent-reads.csv](independent-reads.csv) and [rolling.csv](rolling.csv). These are separate
experiments. Their ratios must not be multiplied to claim a cumulative speedup.

All measurements used an Intel Core i9-13900, 32 logical CPUs, Rust 1.95.0, and the optimized test
profile (`opt-level=1`). Each configuration has one discarded warmup and 20 measured samples.
They are short local samples with scheduling variability, not production release-profile results
or statistical confidence intervals. Workers are additional to the canonical coordinator.

## Final implementation against the sequential reference

64 transactions per sample, four speculative workers, direct readers. Cells contain median / p95
milliseconds. The final column is sequential median divided by rolling median; below 1× is slower.
Execution includes snapshot/session work, EVM, observation, validation, retries, fee/refund settlement,
reader teardown and bundle merging. Roots are measured separately. The long-block benchmark raises
the speculation budget from the default 30M to 128M gas, with a 100M block limit, so all 64
transactions can participate.

| Workload | Sequential | Direct window | Final rolling | Rolling speedup over sequential |
| --- | ---: | ---: | ---: | ---: |
| Independent compute | 6.050 / 6.263 | 1.953 / 2.670 | 1.957 / 2.158 | 3.09× |
| Same-sender nonce chain | 5.873 / 7.884 | 5.684 / 5.754 | 5.661 / 5.693 | 1.04× |
| BN254 pairing | 95.980 / 99.339 | 28.939 / 34.030 | 25.667 / 27.907 | 3.74× |
| Shared storage, warm | 2.095 / 2.126 | 3.372 / 3.971 | 2.185 / 3.173 | 0.96× |
| Independent storage, warm | 2.013 / 2.035 | 3.364 / 3.753 | 2.891 / 3.174 | 0.70× |
| Shared storage, cold | 2.033 / 2.116 | 2.870 / 3.567 | 2.146 / 2.498 | 0.95× |
| Independent storage, cold | 2.511 / 2.592 | 3.916 / 5.197 | 2.961 / 3.984 | 0.85× |
| Conflicting storage, cold | 2.115 / 2.130 | 4.624 / 6.361 | 3.526 / 6.022 | 0.60× |
| Slow window heads | 4.004 / 4.073 | 3.980 / 6.543 | 4.038 / 4.725 | 0.99× |
| Slow window tails | 4.059 / 4.121 | 10.774 / 20.798 | 3.819 / 14.120 | 1.06× |
| Expensive ordered policy | 15.938 / 16.597 | 13.574 / 16.659 | 11.852 / 12.768 | 1.34× |

Independent compute improves 3.09× and stock pairings 3.74× at four workers. At eight workers,
the corresponding improvements are 4.44× and 6.31×. Pure nonce chains remain sequential; their
small timing differences should be treated as noise. Four-worker storage medians remain 4–67%
slower than sequential in this in-memory fixture despite large gains over earlier parallel paths.
The expensive-policy control improves 1.34× against sequential and then plateaus as ordered policy
work dominates. For a fixed transaction mix, inverse block latency gives the same executor-only
throughput ratios; sustained node TPS, gas/second and CPU energy per block were not measured.

Slow-tail rolling improves median latency 2.82× relative to windows, but only 1.06× relative to
sequential. Its 14.120 ms p95 is still worse than sequential's 4.121 ms. Conflicting storage also
has worse tails: 6.022 ms versus sequential's 2.130 ms. A better median does not establish a better
tail or justify enabling parallel execution for every workload.

## Contributions of the individual changes

The initial execution split, deferred protocol fee credits, transaction-local policy observation,
ordered evaluation and shared validation/sequencing integration have no isolated timing A/B in the
saved data. They enable safe reuse and avoid artificial fee-recipient dependencies; assigning them
a separate percentage would be speculation.

Sender filtering and the sequential shortcut eliminate useless nonce-chain attempts. In the initial
prototype, each eight-transaction nonce chain speculated all eight and only one attempt succeeded.
The optimized operational path dispatches zero speculative jobs for a pure chain. Shadow mode now
checks every transaction, increasing coverage and work.

| Eight-transaction workload | Initial prototype median ms | Optimized checkpoint median ms | Reduction |
| --- | ---: | ---: | ---: |
| Nonce chain, 1 worker | 1.091 | 0.744 | 31.8% |
| Nonce chain, 2 workers | 0.887 | 0.742 | 16.3% |
| Nonce chain, 4 workers | 0.790 | 0.746 | 5.6% |
| Stock pairings, 1 worker | 25.653 | 13.768 | 46.3% |
| Stock pairings, 2 workers | 20.480 | 7.268 | 64.5% |
| Stock pairings, 4 workers | 17.265 | 5.203 | 69.9% |

Stock-precompile compatibility changed pairing-result reuse from 0/8 to 8/8. Before this change,
the speculative pairing work was discarded and repeated canonically. The checkpoint's four-worker
pairing result is 3.32× faster than that prototype. Separate runs varied materially (another
post-optimization run recorded 3.396 ms), so the saved checkpoint result is used consistently above.
Customized precompile maps, including the current opaque Engine precompile-cache wrappers, still
require canonical fallback. Stock pairing speedups must not be generalized to those paths.

Independent state readers remove synchronous coordinator request/reply messages. Eight-transaction
results at four workers isolate that change:

| Workload | Broker window median ms | Direct window median ms | Speedup |
| --- | ---: | ---: | ---: |
| Independent compute | 0.397 | 0.341 | 1.16× |
| Same-sender nonce chain | 0.708 | 0.707 | 1.00× |
| BN254 pairing | 3.950 | 3.799 | 1.04× |
| Shared storage, warm | 3.024 | 0.502 | 6.02× |
| Independent storage, warm | 2.853 | 0.488 | 5.85× |
| Shared storage, cold | 2.824 | 0.477 | 5.92× |
| Independent storage, cold | 2.742 | 0.479 | 5.72× |
| Conflicting storage, cold | 3.120 | 0.615 | 5.07× |

The storage improvement is 5.07–6.02× (80–83% less latency) versus the broker. Shared storage
previously required 2,104 broker read requests per eight transactions; successful direct execution
requires zero. Warm shared storage serves all 2,104 direct reads from the snapshot. Compute and
pairing gain much less because their execution time is less sensitive to database-message overhead.

Rolling scheduling adds 14–35% lower storage medians and about 13% lower expensive-policy median
latency versus direct windows in the 64-transaction experiment. Independent compute is effectively
flat, and slow heads regress about 1.5%. Rolling removes the drain-before-commit barrier; it does
not remove true dependencies or make ordered policy evaluation parallel.

## Scaling with worker count

Final rolling median latency in milliseconds. A worker count of one still allows the coordinator
to overlap canonical finalization with its separate worker.

| Workload | Sequential | 1 worker | 2 workers | 4 workers | 8 workers |
| --- | ---: | ---: | ---: | ---: | ---: |
| Independent compute | 6.050 | 6.410 | 3.350 | 1.957 | 1.364 |
| BN254 pairing | 95.980 | 101.618 | 52.282 | 25.667 | 15.213 |
| Same-sender nonce chain | 5.873 | 5.967 | 5.683 | 5.661 | 5.736 |
| Shared storage, cold | 2.033 | 5.251 | 3.106 | 2.146 | 2.104 |
| Independent storage, warm | 2.013 | 5.671 | 3.447 | 2.891 | 2.839 |
| Conflicting storage, cold | 2.115 | 5.431 | 3.434 | 3.526 | 3.422 |
| Expensive ordered policy | 15.938 | 11.962 | 11.974 | 11.852 | 11.854 |

Compute and pairings continue benefiting through eight workers. Storage largely flattens beyond
2–4 workers. Nonce chains dispatch no workers, and the expensive-policy control gets nearly all
its benefit with one worker. Additional workers consume capacity without materially reducing its
ordered bottleneck. Competing-operation fairness is covered by deterministic tests; concurrent-build
throughput and admission-latency distributions have not been benchmarked.

## Worker occupancy and coordinator costs

Representative four-worker means per 64-transaction block. Utilization is summed worker lease time
divided by wall time and worker count; it includes reader setup and teardown and is not a hardware
CPU-utilization counter. Concurrent worker timings overlap and cannot be added to wall latency.

| Metric | Direct window | Rolling |
| --- | ---: | ---: |
| Compute worker utilization | 78.1% | 92.6% |
| Pairing worker utilization | 91.6% | 98.2% |
| Shared warm storage worker utilization | 48.4% | 75.5% |
| Independent warm storage worker utilization | 39.7% | 55.2% |
| Compute snapshot update time | 0.029 ms | 0.284 ms |
| Independent warm storage snapshot update time | 0.669 ms | 0.871 ms |
| Shared cold storage dependency validation | 0.360 ms | 0.414 ms |
| Conflicting storage canonical retry time | 1.678 ms | 1.919 ms |
| Expensive-policy finalization | 11.679 ms | 11.310 ms |

Better occupancy has costs: snapshots advance more often and coordinator work can absorb the gain.
The expensive-policy evaluator alone still takes about 11.2 ms with rolling; overall mean execution
is 12.167 ms. This explains why that case does not scale with additional EVM workers. Its synthetic
policy performs 256 chained Keccak operations per ordered decision; no production refund formula
was benchmarked.

Rolling head-wait sums average 0.723 ms for compute, 0.170 ms for shared cold storage, 24.870 ms for
pairings, and 0.173 ms for the expensive-policy control. Window's corresponding wait is inside its
whole-window barrier, so its zero `head_wait` field does not mean zero waiting. Queue-wait sums
count overlapping waits from different transactions and may exceed block wall time.

## Retry work and database amplification

In the conflicting 64-transaction case, windows retry 60 transactions and rolling retries 63:
only 4/64 and 1/64 speculative results, respectively, are reusable. Rolling lowers wall latency by
overlapping work even while doing more wasted execution. Sender filtering avoids this waste for
pure nonce chains; generic storage dependency prediction has not been implemented.

Independent readers remove messaging but can increase base-reader calls:

| Eight-transaction workload | Broker base calls | Direct base calls | Amplification |
| --- | ---: | ---: | ---: |
| Shared storage, warm | 4 | 4 | 1.00× |
| Shared storage, cold | 261 | 2317 | 8.88× |
| Independent storage, cold | 2061 | 4125 | 2.00× |

These include canonical validation/retry reads and worker reads. They are database-interface calls,
not physical disk reads. The four-worker 64-transaction window and rolling runs have equal base-call
counts for each workload; rolling changes timing and reader reuse, rather than inherently reducing
read amplification. Reader opens vary: compute falls from 16 to about 6 per block and pairings from
16 to about 5, while expensive-policy work rises from 16 to about 51 and conflicts from 16 to about
40. The finite tasks yield fairly and close when idle. This fixture's Arc-based reader open is cheap;
actual MDBX transaction-opening costs remain unmeasured.

## Memory and resource bounds

Rolling retains multiple immutable committed versions. In the four-worker independent warm storage
case, the conservative estimate rises from about 2.0 MiB with windows to 24.4 MiB with rolling.
The maximum across the matrix is 32.6 MiB. Distinct versions are charged their full logical size,
even where persistent nodes share memory; clones of the same version share one charge. Successful
results release their version lease and keep their output/dependencies.

The session budget defaults to 64 MiB and triggers sticky broker fallback after cancellation and
drain. The default 16 in-flight attempts include queued, running, ready and prepared work. Retained
read and output limits remain 4 MiB each per attempt. These are logical admission bounds, not a
process RSS guarantee; caches, providers, EVM working memory and other sessions also use memory.
Large retained historical overlays can exhaust the conservative version budget and fall back to
broker windows, so direct/rolling performance is not guaranteed throughout a long batch.

Whole-process fixture RSS was 15,240 KiB for the eight-transaction independent-read suite and
23,136 KiB for the 64-transaction rolling suite. Different fixture sizes and mode mixtures make
these unsuitable as an isolated before/after memory comparison. Per-mode production RSS has not
been measured.

## Shadow-mode cost

Shadow uses sequential output and compares speculation. Four-worker medians:

| Workload | Sequential ms | Window shadow ms | Rolling shadow ms | Rolling overhead over sequential |
| --- | ---: | ---: | ---: | ---: |
| Independent compute | 6.050 | 8.552 | 6.923 | 14% |
| BN254 pairing | 95.980 | 127.801 | 102.244 | 7% |
| Same-sender nonce chain | 5.873 | 12.848 | 13.029 | 122% |
| Shared storage, cold | 2.033 | 4.987 | 3.951 | 94% |
| Independent storage, warm | 2.013 | 6.412 | 4.842 | 141% |
| Expensive ordered policy | 15.938 | 30.816 | 28.744 | 80% |

All measured shadow comparisons matched. Its duplicate work and comparison costs are operationally
material, particularly for chains and cheap storage; shadow is a correctness rollout mechanism.

## Roots and the end-to-end limit

The headline tables time execution only. Synthetic state/receipt roots are measured separately;
root construction itself was not parallelized by this work. For example, independent warm storage
has mean execution of 3.416 → 2.927 ms for direct window → rolling, but fixture root construction
costs 32.014 → 33.522 ms. Including both phases gives 35.430 → 36.449 ms in that run: the execution
improvement disappears. This full in-memory fixture root is not reth's incremental trie pipeline,
so it establishes neither a production root bottleneck nor a production end-to-end regression.

No target-chain database/range was supplied. Engine RPC, pool selection, real MDBX reads, production
trie work, sustained competing builds, release-profile behavior and production policy formulas lack
end-to-end before/after measurements. Reorg, historical, subblock, selector, lifecycle, fee/refund,
cancellation and source-failure behavior have correctness coverage, which does not quantify their
performance.

The saved validation record is 542 passing execution/integration/CLI tests plus the 121-configuration
rolling benchmark, with formatting, Clippy, rustdoc and affected no_std checks passing. Execution
remains sequential by default; window remains the selected scheduler if parallel mode is enabled
without an explicit scheduler flag. Real-chain replay with fixed parents, budgets, cache policy and
worker allocation is needed before choosing a production default.
