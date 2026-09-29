# Optimistic OP execution

Execution remains sequential by default. The node accepts `--execution.mode shadow` or
`--execution.mode parallel`. Both modes cover Engine payload validation, historical execution,
forced ordinary transactions, and pool transactions. Tracing stays sequential. Shadow mode commits
the reference result, logs differences, counts mismatches, and disables speculation for the rest of
that block after a mismatch.

## Coordinator and workers

`ParallelRuntime` owns the worker budget. Node construction supplies reth's existing prewarming
pool through `with_spawner`; validation and payload builds share that budget instead of adding
another pool alongside prewarming and state-root work. Standalone integrations can use `new`.

A window freezes the coordinator's committed state while isolated `op-revm` instances execute.
Workers request reads through a bounded channel. The coordinator services the requests on its own
thread, so reth providers and their read transactions need not be `Send` or `Sync`. Each worker has
its own journal and read cache. Reads from canonical overlays and physical caches are recorded
above those caches, including missing accounts, account metadata, code, slots, and block hashes.
No worker can see another worker's writes. No speculative state reaches a state hook or receipt
stream.

Windows drain before ordered commit. At the authoritative transaction's turn, its entire environment,
identity and read set must still match. A failed or stale attempt executes once through the canonical
path. Account dependencies protect metadata; revm applies only touched accounts and changed slots,
with its existing creation/deletion semantics. State, fee settlement, receipt accounting, and policy
publication happen only after the commit decision. Prepared transactions must be finalized and
committed in order, as required by the underlying `BlockExecutor` API.

Each window dispatches at most one candidate per sender. Descendants consume no speculative gas
or worker slots and execute canonically while cached results for other senders remain available.
A window with only one candidate executes canonically in parallel mode; shadow mode still runs
the worker for comparison. Pure nonce chains avoid repeated lookahead scans until another sender
is selected or fresh candidate hints arrive.

Each executor owns a build-generation identifier. Boundaries clear cached outcomes and rotate the
generation; dropping or cancelling the executor invalidates it. Workers finish before control returns
to the selector. Cancellation is checked again before a prepared pool transaction commits. Deposits,
system changes, activation rules, and the trailing structural post-exec transaction retain the
canonical path.

The integration supplies bounded `ParallelCandidate` hints through `OpBlockExecutionCtx` instead
of changing the pinned reth traits. This lets reth's existing Engine and historical loops retain
authority over transaction order and commit hooks without requiring a block access list. Pool
lookahead uses an independent iterator. `PayloadTransactionsWithCommitHook::preview` lets custom
selectors provide hints without advancing `next()`. Existing selectors can return no previews.
`on_commit` still runs before the selector's next authoritative `next()`.

For downstream Rust integrations, `OpBlockExecutionCtx` has a new `parallel_candidates` field;
sequential callers can supply an empty vector or use `..Default::default()`. Executor factories
are now `Clone` rather than `Copy` because they can share an `Arc<ParallelRuntime>`.

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
| `--execution.workers` | `0` | Auto: half available CPUs, capped by the shared pool |
| `--execution.max-in-flight` | `16` | Retained transactions per executor |
| `--execution.max-speculative-gas` | `30000000` | Additional declared gas per block/build |
| `--execution.max-read-bytes` | `4194304` | Estimated retained dependencies per attempt |
| `--execution.max-output-bytes` | `4194304` | Estimated retained output and observations per attempt |

Failed speculative attempts consume the gas budget too. A transaction consumes at least 21,000
units of this budget, preventing zero-gas invalid candidates from growing the attempted set without
bound. Preview lists are capped at `min(64 * max_in_flight, 4096)`. One window per runtime is admitted
at a time, bounding queued jobs and active worker memory across competing callers. Completed
windows retained by separate executors each have their own per-executor bound. Retained-byte
estimates are admission limits, not an operating-system RSS cap; EVM working memory is additionally
bounded by transaction and speculative gas limits.

Metrics use the `optimism_parallel` prefix: `attempts`, `worker_failures`, `conflicts`, `reused`,
`shadow_comparisons{result}`, `worker_seconds` (including database waits), `evm_seconds`
(excluding broker waits, including observation), `read_wait_seconds`, `window_seconds`,
`finalization_seconds`, `policy_evaluation_seconds`, `canonical_retry_seconds`,
`canonical_retry_gas`, `read_bytes`, and `output_bytes`. Policy observation is included in worker
EVM time; evaluation is measured separately. `statistics()` also exposes counters without
requiring a metrics recorder. Use existing reth block validation, payload build, and state-root timing
metrics together with process RSS when evaluating end-to-end performance.

Run the correctness suites from `rust/`:

```sh
mise exec -- cargo nextest run -p reth-optimism-parallel -p alloy-op-evm -p op-revm --features alloy-op-evm/parallel
mise exec -- cargo nextest run -p reth-optimism-evm -p reth-optimism-payload-builder -p kona-executor
mise exec -- cargo build -p alloy-op-evm -p op-revm -p kona-executor --no-default-features --target riscv32imac-unknown-none-elf
```

The ignored `benchmark_optimistic_execution` fixture measures warmed execution windows for
independent senders, a nonce chain and stock BN254 pairings, including stateful policy evaluation
and settlement. It checks successful receipts, state and refund parity on every sample, and asserts
that parallel independent/pairing transactions actually reuse worker results:

```sh
mise exec -- cargo nextest run -p alloy-op-evm --features parallel --run-ignored only --nocapture -E 'test(benchmark_optimistic_execution)'
```

One local run on an Intel Core i9-13900 (32 logical CPUs), Rust 1.95, using the workspace's
optimized development/test profile (`opt-level = 1`) produced these timings. Each sample contains
eight transactions; the runtime is reused, one warmup sample is discarded, and 20 samples are
measured. These include EVM execution, observation, ordered settlement and bundle merging, but
exclude provider I/O, trie roots, pool selection and Engine API overhead.

| Workload | Mode / workers | Median ms | p95 ms |
| --- | --- | ---: | ---: |
| Independent senders | Sequential | 0.755 | 0.791 |
| Independent senders | Shadow / 4 | 1.173 | 1.791 |
| Independent senders | Parallel / 1 | 1.076 | 1.471 |
| Independent senders | Parallel / 2 | 0.525 | 0.641 |
| Independent senders | Parallel / 4 | 0.435 | 0.493 |
| Same-sender nonce chain | Sequential | 0.753 | 0.765 |
| Same-sender nonce chain | Shadow / 4 | 1.669 | 1.996 |
| Same-sender nonce chain | Parallel / 1 | 0.744 | 0.764 |
| Same-sender nonce chain | Parallel / 2 | 0.742 | 0.757 |
| Same-sender nonce chain | Parallel / 4 | 0.746 | 0.773 |
| Stock BN254 pairings | Sequential | 12.675 | 12.936 |
| Stock BN254 pairings | Shadow / 4 | 19.673 | 20.520 |
| Stock BN254 pairings | Parallel / 1 | 13.768 | 14.087 |
| Stock BN254 pairings | Parallel / 2 | 7.268 | 9.069 |
| Stock BN254 pairings | Parallel / 4 | 5.203 | 5.834 |

All independent and pairing transactions reused worker output. Parallel nonce chains dispatched no
workers. Shadow mode deliberately still dispatches singleton windows: it now compares every nonce
chain transaction, increasing coverage and runtime. No shadow mismatch occurred.

A baseline run of the same fixture before sender filtering and stock-precompile compatibility had
these median timings. The runs are separate local samples and subject to scheduling noise:

| Workload / mode | Before ms | After ms |
| --- | ---: | ---: |
| Nonce chain / sequential | 0.747 | 0.753 |
| Nonce chain / parallel 1 | 1.091 | 0.744 |
| Nonce chain / parallel 2 | 0.887 | 0.742 |
| Nonce chain / parallel 4 | 0.790 | 0.746 |
| Pairings / sequential | 11.981 | 12.675 |
| Pairings / parallel 1 | 25.653 | 13.768 |
| Pairings / parallel 2 | 20.480 | 7.268 |
| Pairings / parallel 4 | 17.265 | 5.203 |

Previously, nonce descendants consumed the speculative budget and then ran canonically, while
precompile results were always discarded. The new paths avoid these duplicate executions. The
pairing measurements use stock maps, without reth's opaque precompile-cache wrappers.

Synthetic results do not justify enabling parallel mode by default. Before deployment, replay a
fixed representative chain range from the same parent-state snapshot in each mode and worker
configuration. Compare state/receipt roots and refund payloads, and record validation/build latency,
retry work, policy/finalization time, RSS and state-root time. Include high-contention blocks and
cancellation/reorg scenarios. Default enablement remains a separate decision.
