# Optimistic OP execution

See the [PoC write-up](POC.md) for the design decisions, summarized results, and production gates.

Execution remains sequential by default. The node accepts `--execution.mode shadow` or
`--execution.mode parallel`. Both modes cover Engine payload validation, historical execution,
forced ordinary transactions, and pool transactions. Tracing stays sequential. Shadow mode commits
the reference result, logs differences, counts mismatches, and disables speculation for the rest of
that block after a mismatch. `--execution.scheduler window` remains the default. Select
`--execution.scheduler rolling` to overlap ordered commit with independent-reader execution;
broker reads continue using drained windows.

## Coordinator and workers

`ParallelRuntime` owns the worker budget. Node construction supplies reth's existing prewarming
pool through `with_spawner`; validation and payload builds share that budget instead of adding
another pool alongside prewarming and state-root work. Standalone integrations can use `new`.

A window freezes the coordinator's committed state while isolated `op-revm` instances execute.
With `--execution.state-reads auto` (the default), compatible integrations open one independent
base reader on each active worker and reuse it for that window or rolling worker task. The owned `StateReadFactory` is
`Send + Sync`; its readers need neither trait. Readers are created, used and destroyed on their
worker. Successful direct execution sends completion messages and no broker read requests.
`--execution.state-reads broker` forces the original bounded request/reply channel; integrations
without an explicit compatible source also use it.

`SnapshotSession` belongs to one execution operation. It combines the exact pinned base with a
`CommittedSnapshot` of committed accounts, storage, bytecode and block hashes. Persistent `imbl`
maps share unchanged nodes, including per-account storage. A collector composed with public state
hooks records dirty addresses, slots and lifecycle resets. Before dispatching another committed version, a typed adapter
refreshes just those entries from canonical revm `State`, including fees, refunds, deposits and
system changes. Initial canonical cache/preloaded-bundle state takes precedence over the base.
Unknown accounts/slots remain distinct from absent accounts, explicit zero slots and cleared
storage. Creation, deletion and recreation follow revm's committed lifecycle status.

Workers have private journals, read caches, dependencies and observations for every transaction.
Dependency recording sits above every backend and cache; ordered validation still loads canonical
caches before commit. Physical cache sharing does not change EVM warmth or transient storage.
No worker sees another worker's writes, and speculative state never reaches public hooks.

A source failure invalidates the affected speculation and invokes the canonical retry; subsequent
windows use the broker for that session. Initialization, read and reader-destruction panics are
contained. Cancellation checks guard reads (including cache hits) and result acceptance. Workers
and readers drain before the window returns. Exceeding the snapshot estimate releases the overlay
and keeps that session on broker reads.

Engine execution carries the existing parent-provider closure, its cache generation/instrumentation
and retained in-memory ancestry into an execution-local config clone. Historical stage execution
opts in only when a durable checkpoint matches its starting writer state; one session follows the
batch's unpersisted changes. Custom write transactions retain the broker unless their caller supplies
an explicit source contract. Sequencing pins the selected parent, shares immutable physical reads,
and carries committed state across forced/pool transactions and subblock continuation. Competing
builds create separate sessions. Reusing a session with a newly constructed canonical State reseeds
its actual preloaded state; reusing the same State updates only dirty entries.

Window scheduling drains before ordered commit. Rolling scheduling owns an `ExecutionPipeline`
with bounded queued, running, completed and prepared attempts. The authoritative transaction can
commit while other attempts execute. Commit refreshes dirty snapshot entries and refills available
slots; workers never consume uncommitted results. Each task yields between transactions when another
operation is waiting, and after at most `max_in_flight` attempts. Idle tasks destroy their readers and
return the shared worker permit instead of parking a prewarming thread. Readers may be reopened by a
later task; this setup cost is included in benchmarks.

At the authoritative transaction's turn, its environment, identity and read set must still match. A failed or stale attempt executes once through the canonical
path. Account dependencies protect metadata; revm applies only touched accounts and changed slots,
with its existing creation/deletion semantics. State, fee settlement, receipt accounting, and policy
publication happen only after the commit decision. Prepared transactions must be finalized and
committed in order, as required by the underlying `BlockExecutor` API.

Each window dispatches at most one candidate per sender. Rolling scheduling retains the same
restriction across outstanding attempts and releases the sender reservation on commit or rejection.
A running cancelled attempt retains its slot until it stops. An authoritative candidate that was
not admitted while other speculation remains executes canonically; it does not wait for unrelated
hints. A prepared result retains its slot until commit or rejection.

In window scheduling, descendants consume no speculative gas or worker slots and execute canonically while cached results for other senders remain available.
A window with only one candidate executes canonically in parallel mode; shadow mode still runs
the worker for comparison. Pure nonce chains avoid repeated lookahead scans until another sender
is selected or fresh candidate hints arrive.

Each executor owns a build-generation identifier. Boundaries clear cached outcomes and rotate the
generation; dropping or cancelling the executor invalidates it. Rolling workers may continue between
selector calls. Deposits, structural post-exec transactions,
environment changes, block/subblock completion, cancellation and executor destruction drain work
before the execution context is released. Cancellation is checked again before a prepared pool
transaction commits. Deposits,
system changes, activation rules, and the trailing structural post-exec transaction retain the
canonical path.

The integration supplies bounded `ParallelCandidate` hints through `OpBlockExecutionCtx`.
The reth configuration hooks supply independent state sources and preserve composed state hooks. This lets reth's existing Engine and historical loops retain
authority over transaction order and commit hooks without requiring a block access list. Pool
lookahead uses an independent iterator. `PayloadTransactionsWithCommitHook::preview` lets custom
selectors provide hints without advancing `next()`. Existing selectors can return no previews.
`on_commit` still runs before the selector's next authoritative `next()`.
`reject_parallel_candidate` retires skipped candidates and invalid sender descendants;
`drain_parallel_work` closes a selection phase while preserving the execution configuration. These
extension methods default to no-ops for custom executors. Replacing previews retires obsolete hints.
Existing bounded preview coverage is unchanged; transactions beyond it remain executable canonically.

For downstream Rust integrations, `OpBlockExecutionCtx` has a new `parallel_candidates` field;
sequential callers can supply an empty vector or use `..Default::default()`. Executor factories
are now `Clone` rather than `Copy` because they can share an `Arc<ParallelRuntime>`.
`OpExecutorBuilder` also requires explicit cloning, custom `OpTxEnv` implementations must implement
`Clone`, and generic users of `OpBlockExecutor` must carry its `Evm` bound. Construct `OpHandler`
through `OpHandler::new()` rather than external struct literals; its deferred-fee storage is private.

The upstream source hook currently preserves its no-op default on borrowed and `Arc` configuration
wrappers. Those wrappers retain broker reads even when the inner configuration advertises source
support. Stock node paths use concrete `OpEvmConfig`; wrapper support is an upstream follow-up.

## Fees and SDM policies

`op-revm` returns ordered protocol fee credits for recipients not otherwise loaded by the EVM.
The coordinator applies them to current state, preserving account touches, aliasing, ordering and
checked-add overflow behavior. Credits to recipients already loaded by the transaction stay in its
journal and create real dependencies. Refund settlement follows these credits and precedes state
publication, so a later transaction can spend a preceding refund after a canonical retry.

`TransactionObserver` records facts from one transaction, including reverted calls and relevant
outcomes. Extra database reads are dependencies. Observers must reset per transaction and bound
their own collections; reported retained bytes count against the output limit. They receive a
provisional index of zero. `ParallelRefundPolicy::evaluate` receives the final index and committed
policy state and returns the decision and prepared next state. `ObservedRefundPolicy` uses that
same evaluator in sequential and parallel modes and publishes the next state only at commit.
Snapshots contain committed policy state for subblock carry-forward. Rejected and failed attempts
discard prepared updates. Adapted policy state must implement `PartialEq`: shadow mode compares
the entire prepared state as well as this transaction's refund and diagnostics, so a discrepancy
that only changes future refunds is still detected.

The public policy still refunds nothing. Production formulas belong downstream. Opaque inspector
policies remain supported in sequential mode; selecting parallel production with an opaque policy
returns an error. Custom factory adapters need an explicit parallel implementation and currently
return a configuration error. Validators consume declared refund entries and never evaluate a
producer policy.

Workers reuse stock precompile results when the canonical EVM still has the untouched OP map for
the same hardfork. The factory records this provenance at construction. Any mutable access through
`precompiles_mut()` or `components_mut()` invalidates it; database and inspector access alone do
not. Address sets, precompile IDs and cacheability flags are insufficient to prove equivalence.

Customized maps retain canonical fallback for precompile calls. This includes reth's opaque Engine
precompile-cache wrappers: cached Engine validation still takes that fallback, while stock maps
used in sequencing, historical execution and Engine execution without the cache can reuse results.
A warmth-set difference also forces fallback. The call recorder includes reverted and nested calls
and dynamic lookups. Custom instruction sets are not supported by the stock parallel factory.

## Bounds and measurements

| Option | Default | Scope |
| --- | --- | --- |
| `--execution.scheduler` | `window` | `rolling` overlaps direct execution with ordered commit |
| `--execution.state-reads` | `auto` | Independent readers when explicitly supported; `broker` forces coordinator reads |
| `--execution.max-snapshot-bytes` | `67108864` | All retained snapshot versions plus dirty tracking per session |
| `--execution.workers` | `0` | Auto: half available CPUs, capped by the shared pool |
| `--execution.max-in-flight` | `16` | Retained transactions per executor |
| `--execution.max-speculative-gas` | `30000000` | Additional declared gas per block/build |
| `--execution.max-read-bytes` | `4194304` | Estimated retained dependencies per attempt |
| `--execution.max-output-bytes` | `4194304` | Estimated retained output and observations per attempt |

Failed speculative attempts consume the gas budget too. A transaction consumes at least 21,000
units of this budget, preventing zero-gas invalid candidates from growing the attempted set without
bound. Preview lists are capped at `min(64 * max_in_flight, 4096)`. Both schedulers share runtime-wide
worker permits. FIFO admission between finite tasks prevents a rolling build from monopolizing the
pool. Each execution operation retains at most its configured transaction budget, including queued
hints and completed output.

Snapshot leases charge each distinct live immutable version once. Clones share a charge; different
versions conservatively pay their full logical size even where persistent nodes share storage. The
canonical version and pending dirty keys also count. Completed results release their snapshot lease
and retain only output and dependencies. Budget exhaustion disables direct reads for the session,
drains workers and releases retained versions before switching to broker windows. This conservative
estimate can cause earlier fallback on large historical overlays.

A rolling reader failure discards uncommitted speculation and disables direct reads. An output
already transferred to the coordinator still requires canonical dependency validation; a later
reader teardown failure cannot invalidate a dependency-validated committed prefix. A dropped
scheduled task also releases its completion bookkeeping, including during pool shutdown.

Retained-byte estimates are admission limits, not an operating-system RSS cap; EVM working memory is additionally
bounded by transaction and speculative gas limits.

Metrics use the `optimism_parallel` prefix: `attempts`, `worker_failures`, `conflicts`, `reused`,
`shadow_comparisons{result}`, `worker_seconds` (including database waits), `evm_seconds`
(excluding broker waits, including observation), `read_wait_seconds`, `window_seconds`,
`finalization_seconds`, `policy_evaluation_seconds`, `canonical_retry_seconds`,
`canonical_retry_gas`, `read_bytes`, and `output_bytes`. Policy observation is included in worker
EVM time; evaluation is measured separately. Read-backend measurements add `direct_reads`,
`broker_reads`, `snapshot_hits`, `provider_opens`, `source_fallbacks{reason}`, `snapshot_bytes`,
`snapshot_update_seconds`, `provider_open_seconds`, `provider_read_seconds` and
`dependency_validation_seconds`. Rolling adds `head_wait_seconds`, `queue_wait_seconds`,
`pipeline_drain_seconds`, `worker_lease_seconds`, `active_workers`, `in_flight` and `snapshot_versions`.
Worker lease time includes reader setup and teardown; divide summed lease time by wall time and the
worker count to estimate utilization. Histogram timings from concurrent workers overlap and must
not be added to wall latency. Provider reads exclude snapshot hits; dependency validation and
ordered finalization are timed separately. The existing `read_wait_seconds` includes backend wait
and lookup time in either mode. `statistics()` also exposes counters without
requiring a metrics recorder. Use existing reth block validation, payload build, and state-root timing
metrics together with process RSS when evaluating end-to-end performance.

Run the correctness suites from `rust/`:

```sh
mise exec -- cargo nextest run -p reth-optimism-parallel -p alloy-op-evm -p op-revm --features alloy-op-evm/parallel
mise exec -- cargo nextest run -p reth-optimism-evm -p reth-optimism-payload-builder -p kona-executor
mise exec -- cargo build -p alloy-op-evm -p op-revm -p kona-executor --no-default-features --target riscv32imac-unknown-none-elf
```

The ignored `benchmark_optimistic_execution` fixture compares sequential, broker and direct
execution at 1/2/4/8 workers, plus both shadow backends. It covers compute, nonce chains, stock
BN254 pairings, warm/cold shared and independent storage reads, and conflicting storage writes.
Every sample checks state, receipt and refund parity. Supported direct samples assert zero broker
reads. Conflicting samples verify canonical retry; nonce-chain production dispatches no workers.

```sh
mise exec -- cargo nextest run -p alloy-op-evm --features parallel,metrics --run-ignored only --nocapture -E 'test(benchmark_optimistic_execution)'
```

See the [session-wide performance comparison](benchmarks/session-summary.md) for the effects of
sender filtering, precompile reuse, independent readers and rolling scheduling.

See [benchmark results](benchmarks/independent-reads.md) for the measured setup, complete
1/2/4/8-worker results, snapshot estimates, base-reader amplification and limitations. Synthetic
storage measurements include session creation, initial capture, provider opening, dependency
validation, canonical retries, policy/fee settlement and bundle merging. They do not model MDBX
I/O or include trie-root construction, pool selection or Engine API overhead.

The ignored `benchmark_rolling_execution` fixture uses 64-transaction blocks, four times the default
window size. It compares direct windows and rolling at 1/2/4/8 workers, shadow at four workers, and
sequential execution. It also covers a slow transaction at each window's head/tail and an explicitly
synthetic expensive ordered policy. It checks state, receipt roots, gas and policy/refund parity;
root calculation is measured separately from execution. See
[rolling benchmark results](benchmarks/rolling.md) for full results, costs and limitations.

```sh
mise exec -- cargo nextest run -p alloy-op-evm --features parallel,metrics --run-ignored only --nocapture -E 'test(benchmark_rolling_execution)'
```

End-to-end replay remains a separate acceptance check: replay a fixed representative chain range
from the same parent-state database in every mode and worker configuration. Compare state/receipt
roots and refund payloads; record validation/build latency, retry work, policy/finalization time,
RSS, database calls and state-root time. Include high-contention blocks, unpersisted ancestors and
cancellation/reorg scenarios. No target-chain database was supplied for this run, so no end-to-end
speedup is claimed. Parallel execution remains disabled by default.

## Upstream dependency

The reth hooks are in [ethereum-optimism/reth#10](https://github.com/ethereum-optimism/reth/pull/10),
pinned for integration to `a643e0989ffc0c225dc5925031e5870bcfbb6e0d`, a direct descendant of the
previous `0fbe428518594611bdd3fdd822eb5968823c8e66` pin. This development pin must be replaced by
the reviewed merge revision before landing, following `rust/UPDATING-RETH.md`.

The 11-file upstream delta changes only source/configuration plumbing, immutable cache access,
parent retention and checkpoint opt-in. It preserves the fork's existing patches and shared crate
versions. Slot-preimage APIs/layout and all 22 non-frozen mirrored symbols are unchanged; their
source files were compared across these revisions before advancing mirror tags. Lockfile changes
are limited to the reth revision and the explicitly added workspace/dev dependencies; both SP1
workspace lockfiles still resolve without updates.
