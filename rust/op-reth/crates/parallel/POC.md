# Optimistic parallel execution PoC

This PoC adds optimistic parallel execution to op-reth across block validation, historical
execution, and sequencing. It keeps op-revm as the EVM and preserves transaction order, state,
receipts, fees, and SDM settlement. Sequential execution remains the default, correctness reference,
and fallback. The implementation includes independent state readers and an optional rolling
scheduler. It is intended to demonstrate the architecture and its performance tradeoffs; production
enablement requires the additional work described below.

**High-level design.** One coordinator owns canonical block state, transaction order, cumulative
limits, receipts, and policy state. Workers execute isolated transactions against immutable snapshots
of a committed prefix. They never consume another worker's uncommitted writes. This choice avoids
multi-version speculative state and cascading rollback; a full Block-STM scheduler is outside the
PoC's scope.

Each transaction follows the same ordered lifecycle:

1. Dispatch a transaction with its environment and a snapshot of committed state.
2. At its authoritative turn, validate the recorded dependencies against current canonical state.
3. If the attempt failed or a dependency changed, execute once canonically to guarantee progress.
4. Enforce limits, evaluate the refund, settle fees, build the receipt, and publish state and policy
   changes together.

Only committed transactions reach state hooks, inclusion callbacks, and persistent output. System
changes, deposits, and structural post-exec transactions retain their existing sequential boundaries.
Validators consume the block's declared refunds; producers evaluate the configured policy.

Two schedulers share this execution core. **Window** scheduling drains a bounded batch before
committing. **Rolling** scheduling commits the authoritative transaction as soon as its result is
ready and refills from the new committed prefix while unrelated workers continue. Rolling requires
independent readers; broker fallback retains window scheduling. A selected transaction outside the
preview can execute canonically without waiting for unrelated speculation. Neither parallel
execution nor rolling scheduling becomes the default.

**Low-level decisions.** The main implementation constraints are state fidelity, ordered effects,
and resource ownership:

- **Independent readers and committed snapshots.** A `Send + Sync` factory opens readers on the
  worker that uses and destroys them; the readers themselves need not be `Send` or `Sync`.
  Persistent maps share unchanged state across versions. Initial snapshots capture the actual
  canonical cache and preloaded bundle, and a collector composed with public state hooks tracks
  committed dirty entries. A typed adapter reads those entries back from canonical revm `State`,
  preserving its account lifecycle rules. Unknown entries, absent accounts, zero slots, and cleared
  storage remain distinct. Engine ancestry, historical starting checkpoints, and subblock sessions
  retain their exact execution base. Generic integrations opt into this source contract explicitly.
- **Dependency recording above caches.** Account existence and metadata, code identity, individual
  slots, and lifecycle dependencies are recorded regardless of which backend or cache satisfies a
  read. Validation checks current canonical state and loads canonical caches before commit. Workers
  retain private journals, warmth, and transient storage; physical cache sharing has no gas effect.
- **Protocol fees without artificial conflicts.** Fee credits for recipients otherwise untouched by
  the EVM are returned as ordered operations and applied during canonical settlement. Explicit EVM
  accesses remain dependencies. Settlement preserves touches, address aliasing, and the order of
  checked and saturating arithmetic rather than aggregating unchecked balance deltas.
- **Parallel SDM observation, ordered evaluation.** A transaction-local observer records policy
  inputs, including relevant reverted calls and outcomes. An evaluator consumes the observation,
  final transaction index, and committed policy state, preparing the refund and next policy state.
  The update becomes visible only on commit. Rejected or invalidated attempts discard observations
  and prepared updates. Sequential and parallel modes use the same evaluator, and shadow mode also
  compares prepared policy state. Production formulas remain downstream; opaque inspector policies
  remain sequential, and explicit parallel production with an unsupported policy reports an error.
- **Bounded ownership.** Queued, running, completed, and prepared attempts all occupy the in-flight
  budget. One outstanding candidate per sender avoids useless nonce-chain speculation, and discarded
  attempts still consume speculative gas. Version leases charge each distinct retained snapshot
  once against the session budget; clones share the charge. Shared worker permits and finite tasks
  let competing operations make progress without parking pool threads while commits run.
- **Authoritative selection and reusable computation.** Preview is a bounded scheduling hint. The
  existing selector still decides inclusion, rechecks eligibility and budgets, and receives
  `on_commit` before its next authoritative `next()`. Rejected and obsolete hints retire their
  attempts. Stock precompile results are reusable only when the canonical EVM's provenance proves
  compatibility; customized maps and opaque cache wrappers use canonical fallback.

Cancellation, sequential boundaries, environment changes, completion, and destruction drain workers
and their readers before releasing execution context. Source failures discard uncommitted
speculation and disable direct reads for that session. Snapshot-budget exhaustion also drains and
falls back to broker windows. Scheduling lives in the standard-library execution component; shared
execution and fee primitives retain `no_std` support. RPC tracing remains sequential.

**Measured results.** The headline comparisons are summarized below. Each row compares the stated
pair of configurations; the improvements come from separate experiments and are not cumulative.

| Comparison | Measured result |
| --- | --- |
| Final rolling vs sequential, independent compute | 3.1x at 4 workers; 4.4x at 8 |
| Final rolling vs sequential, stock BN254 pairings | 3.7x at 4 workers; 6.3x at 8 |
| Independent readers vs broker windows, storage workloads | 5-6x at 4 workers |
| Rolling vs direct windows, storage workloads | 14-35% lower median latency at 4 workers |
| Rolling vs direct windows, expensive ordered policy | 13% lower median latency at 4 workers |

These are local, in-memory synthetic measurements using an optimized test profile, one warmup and
20 measured samples. Independent-reader comparisons use eight transactions; rolling comparisons
use 64. Execution timings include snapshot work, reader setup and teardown, validation, retries,
and settlement. State-root construction is measured separately. The
[performance report](benchmarks/session-summary.md) includes raw data links, methodology, scaling,
tail latency, memory estimates, and database-interface call counts.

Cheap storage workloads often remain faster sequentially. True conflicts cause retries; rolling
can improve overlap while increasing wasted work. Cold direct reads amplify base-reader calls,
and retaining versions increases the conservative memory charge. Serial policy evaluation limits
scaling, and improved medians do not guarantee better tails. These results establish no production
MDBX, full-node, sustained-throughput, or end-to-end replay speedup. Stock-precompile results also
do not establish the same gain through opaque Engine precompile-cache wrappers.

The latest affected-suite run contains 603 passing execution, integration, txpool, and CLI tests,
with three skipped tests. The saved 121-configuration rolling benchmark includes parity assertions.
Formatting, scoped Clippy, rustdoc,
and affected `no_std` checks passed. Coverage includes storage and account lifecycle conflicts,
nonce chains, deposits, fee/refund dependencies, stateful policies, selector callback order,
historical and subblock state, cancellation, reader failures, and snapshot limits. These checks
support the PoC; sequential and parallel paths share refactored code, so agreement alone cannot
exclude a common bug.

**Before production.** Four gates remain:

1. Independently review dependency completeness, state capture, fee arithmetic, account lifecycle,
   and commit atomicity. Add sustained randomized scheduling, differential fuzzing, and replay
   against known historical roots or an independent implementation.
2. Measure release builds against representative MDBX databases and complete validation/build
   pipelines, including roots, pool selection, cache behavior, and competing operations. Stress
   persistence, pruning, reorgs, long batches, memory, and payload deadlines. Logical session bounds
   are not a process RSS cap; cooperative cancellation cannot preempt a long EVM or provider call.
3. Integrate the real SDM policy with bounded observations and verify its reverted-call and subblock
   semantics. Define the supported custom-EVM/precompile matrix and land reviewed upstream source
   hooks, including source installation through borrowed and `Arc` configuration wrappers (which
   currently retain broker reads). The development dependency on
   [ethereum-optimism/reth#10](https://github.com/ethereum-optimism/reth/pull/10) must be replaced by
   the reviewed merge revision before landing.
4. Run a sustained shadow deployment with useful mismatch captures and alerts, then limited canaries
   with explicit correctness, latency, resource, and rollback criteria. Default enablement is a
   separate decision based on representative workload gains and contention regressions.

For configuration, API migration notes, and reproduction commands, see the [README](README.md).
