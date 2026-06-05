# op-interop-mon — Align with supernode / lightnode / interop-filter design

**Date:** 2026-06-05
**Status:** Draft for review
**Branch:** `interop-consistent-failsafe-supernodes`

## Problem

`op-interop-mon` was built for the **op-supervisor** era of interop. op-supervisor has since
been deleted as a standalone service. The current topology is:

- **op-supernode** — CL host running every chain in the dependency set in-process; makes the
  authoritative cross-chain safety decision. Exposes `supernode_syncStatus`,
  `superroot_atTimestamp`, `heartbeat_check`. Holds **no** failsafe.
- **Light CL** — `op-node --l2.follow.source` mode following a supernode (the "lightnode").
- **op-interop-filter** — EL-side service that holds the **failsafe** and answers
  `interop_checkAccessList` for op-reth/op-geth tx admission, backed by a LogsDB cross-validator.

The monitor is inconsistent with this design in two ways:

1. **Validation is stale.** `RPCUpdater.UpdateJobStatus` validates an executing message (EM) by
   raw payload-hash equality at the claimed initiating log index. The current protocol defines
   validity via the **`MessageChecksum`** (which binds origin, block number, timestamp, log index,
   chainID) **plus message expiry** (`init.Timestamp + MessageExpiryWindow() >= exec.Timestamp`).
   The monitor never checks origin, timestamp binding, or expiry.
2. **No awareness of the new components.** It cannot cross-check the interop-filter or supernode,
   which are now the live enforcement/decision points.

The end goal is to run the updated monitor against the **interop-reorg-0** devnet (2 chains:
`420120132`, `420120133`; op-reth + light op-nodes; 3 supernodes via `proxyd-cl`; 1 interop-filter
fronting `proxyd-public`) and gather data.

## Goals

1. Make EM validation faithful to current interop validity invariants (checksum binding + expiry).
2. Classify failures with **distinct statuses**: `invalid`, `expired`, `timestamp_mismatch`.
3. Source the chain set and expiry window from a **depset JSON file** (`--dependency-set`),
   consistent with op-supernode / op-node.
4. Add an **optional, read-only interop-filter observer**: `interop_checkAccessList` divergence
   metric + `admin_getFailsafeEnabled` gauge.
5. Add an **optional, read-only supernode observer**: liveness/sync gauges + a cross-safety
   violation metric (bad EM found in a block the supernode already treats as cross-safe/finalized).
6. Promote the existing initiating-block reorg detection from a log line to a metric.
7. Deploy against interop-reorg-0 and gather data.

## Non-goals (explicit)

- **Do not touch the failsafe-triggering path.** `monitor/supervisor_client.go`,
  `--supervisor-endpoints`, `--trigger-failsafe`, and `MetricCollector.TriggerFailsafe` /
  `CheckFailsafeStatus` stay exactly as they are. The failsafe now lives in op-interop-filter; a
  future change may repoint it, but not here.
- **Do not couple core correctness to the new services.** The filter and supernode are *observers*.
  If their endpoints are not configured, the monitor behaves exactly as in Phase 1. The monitor
  remains an independent watchdog with its own source of truth (L2 receipts).

## Approach

Chosen: **independent watchdog + passive observers**, delivered in phases so meaningful data
flows after Phase 1, with observers added incrementally. Rejected: making the filter/supernode the
source of truth (destroys the independent-watchdog value), and a minimal validation-only fix
(does not honor the cross-check requirement).

---

## Phase 1 — Validity correctness (core)

### Data model

- `Job` gains `executingTimestamp uint64` — the executing block's timestamp, needed for the expiry
  check. Captured in the finder's `processBlock` (it already receives `eth.BlockInfo`, which has
  `Time()`), and threaded through `BlockReceiptsToJobs` / `JobFromExecutingMessageLog`.
- `Job` retains `executingPayload` (the EM's asserted payload hash); optionally also stores the
  EM's asserted `MessageChecksum` for a single-comparison binding check.
- `jobStatus` enum gains `jobStatusExpired` and `jobStatusTimestampMismatch`. Both are terminal,
  like `valid`/`invalid`. `unknown` stays non-terminal.

### Dependency set

- New flag `--dependency-set <path>` (env `OP_INTEROP_MON_DEPENDENCY_SET`), loaded via
  `depset.JSONDependencySetLoader`. Provides `Chains()` and `MessageExpiryWindow()`.
- At init: dial each `--l2-rpcs` endpoint, query chain ID, and reconcile against `depset.Chains()`.
  Warn on an RPC whose chain is not in the depset; error if a depset chain has no RPC (the monitor
  cannot validate EMs initiating on a chain it can't read).
- The depset's `MessageExpiryWindow()` (default `604800`s; override honored) feeds the expiry check.
  `--dependency-set` is **required** going forward (the expiry window and authoritative chain set
  come from it).

### New validation logic (`RPCUpdater.UpdateJobStatus`)

1. Fetch initiating block info + receipts at `job.initiating.BlockNumber`. Error -> `unknown`.
2. `AddInitiatingHash(blockInfo.Hash())` (existing reorg tracking).
3. Find the log at `initiating.LogIndex`. Not found -> `invalid` (missing initiating message).
4. `log.Address != initiating.Origin` -> `invalid` (wrong origin).
5. `initiatingBlock.Time() != initiating.Timestamp` -> `timestamp_mismatch`.
6. Payload/checksum binding: compute the checksum from the *actual* initiating log
   (`messages.Identifier{Origin: log.Address, BlockNumber, LogIndex, Timestamp, ChainID}` +
   `LogToLogHash(log)` -> `ChecksumArgs.Checksum()`) and compare to the EM's asserted checksum.
   Mismatch -> `invalid`. (Equivalent payload-hash compare is retained as the inner check.)
7. Expiry (`depset.MessageExpiryWindow()`):
   - `executingTimestamp < initiating.Timestamp` -> `invalid` (EM precedes its initiating message).
   - `executingTimestamp > initiating.Timestamp + window` -> `expired`.
8. Otherwise -> `valid`.

### Metrics (`MetricCollector`)

- Initialize and emit `expired` and `timestamp_mismatch` alongside `valid`/`invalid`/`unknown` in
  the per-`(executingChain, initiatingChain)` status map.
- **Failsafe trigger unchanged:** `shouldFailsafe` stays driven by `jobStatusInvalid` and the
  valid<->invalid terminal transition only. `expired` / `timestamp_mismatch` are new metric
  categories that do **not** feed the (untouched) failsafe path.
- New reorg metric: when a job has more than one initiating hash, emit a counter
  `RecordInitiatingReorg(executingChain, initiatingChain)` (currently only a `log.Warn`).

### Tests (TDD, write first)

Table-driven `UpdateJobStatus` tests: valid; missing log; wrong origin; timestamp mismatch;
payload/checksum mismatch; expired; EM-before-initiating. Depset loading + RPC/depset reconciliation
test. Keep existing tests green.

---

## Phase 2 — interop-filter observer (optional, read-only)

- New optional flag `--interop-filter-endpoint <url>` (env `OP_INTEROP_MON_INTEROP_FILTER_ENDPOINT`).
  Absent -> observer disabled, no behavior change.
- Optional `--interop-filter-min-safety` (default `cross-unsafe`) for the cross-check call.
- New `FilterClient` (read-only):
  - `CheckAccessList(ctx, inboxEntries []common.Hash, minSafety safety.Level, execDesc messages.ExecutingDescriptor) error`
  - `GetFailsafeEnabled(ctx) (bool, error)` (public read-only method)
- A `FilterObserver`, invoked by the collector with a per-call timeout (does not block core metrics):
  - For each current job, rebuild the access list: `access := msg.ToCheckSumArgs().Access()`,
    `inboxEntries := messages.EncodeAccessList([]messages.Access{access})`,
    `execDesc := messages.ExecutingDescriptor{ChainID: executingChain, Timestamp: executingTimestamp, Timeout: 0}`.
    Call `CheckAccessList`. Emit `RecordFilterDivergence(executingChain, initiatingChain, monitorVerdict, filterVerdict)`
    when the filter's verdict (ok/err) disagrees with the monitor's status.
  - Poll `GetFailsafeEnabled` once per interval -> `RecordFilterFailsafe(enabled)` gauge.
- For reorg-0 the job count is small; cross-checking all current jobs per collection interval is fine.
  (If scale matters later, sample.)

### Tests

`FilterObserver` with a mock `FilterClient`: agreement (no divergence), filter-says-invalid /
monitor-says-valid and vice versa (divergence emitted), failsafe gauge reflects polled value,
disabled when endpoint unset.

---

## Phase 3 — supernode observer (optional, read-only)

- New optional flag `--supernode-endpoints <url...>` (env `OP_INTEROP_MON_SUPERNODE_ENDPOINTS`).
  Absent -> disabled. (reorg-0 exposes supernodes via a single `proxyd-cl` endpoint.)
- New `SupernodeClient` (read-only): `SyncStatus(ctx)`, `Heartbeat(ctx)`. Exact method
  signatures confirmed against `op-supernode` at implementation time.
- Emit liveness gauge (`heartbeat_check` ok) and per-chain cross-safe / finalized head gauges
  from `supernode_syncStatus`.
- **Cross-safety violation metric (highest-signal):** when a job is `invalid` / `expired` /
  `timestamp_mismatch` **and** its executing block number <= the supernode's cross-safe (or
  finalized) head for the executing chain, emit `RecordCrossSafetyViolation(executingChain,
  initiatingChain, level)`. This flags a bad EM that the CL already promoted.

### Tests

`SupernodeObserver` with a mock client: liveness gauge; head gauges; violation emitted when a bad
job sits at/below the cross-safe head; no violation otherwise; disabled when unset.

---

## Phase 4 — Deploy & gather data

1. Build: `just op-interop-mon` (binary) / `docker buildx bake op-interop-mon` (image) — targets
   already exist.
2. Produce the reorg-0 depset file:
   `{"dependencies":{"420120132":{},"420120133":{}}}` (add `overrideMessageExpiryWindow` only if
   reorg-0 overrides it — verify from the devnet config; default `604800`s otherwise).
3. Run against interop-reorg-0:
   - `--dependency-set <reorg0-depset.json>`
   - `--l2-rpcs` -> the two chains' RPC endpoints (port-forward `proxyd-public` / rpc nodes per chain)
   - `--interop-filter-endpoint` -> the filter (via `proxyd-public`)
   - `--supernode-endpoints` -> `proxyd-cl`
   - failsafe flags left **off** (`--supervisor-endpoints` unset; `--trigger-failsafe` irrelevant)
   - metrics enabled
   First run locally against port-forwarded endpoints to gather data quickly; optionally add to the
   devnet inventory via neti afterward.
4. Gather: scrape `/metrics`; summarize valid/invalid/expired/timestamp_mismatch counts, filter
   divergences, filter failsafe engagements, initiating reorgs, and cross-safety violations.

## Risks / open questions

- **reorg-0 expiry override:** confirm whether the devnet overrides `MessageExpiryWindow`; if
  unknown, default `604800`s.
- **Supernode RPC signatures:** confirm `supernode_syncStatus` / `heartbeat_check` shapes against
  `op-supernode` before wiring Phase 3.
- **min-safety for the filter cross-check:** default `cross-unsafe`; revisit if it produces noise.
- **Spec-doc placement:** this file lives under `docs/superpowers/specs/`; drop from the final PR if
  the monorepo shouldn't carry it.
