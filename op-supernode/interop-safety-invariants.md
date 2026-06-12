# Interop Verifier Safety Invariants & Hardening

This document records the safety invariants the interop verification loop relies
on, how each is enforced, and the hardening changes that make assumption
violations fail loud (terminal halt) instead of silent (infinite retry).

It is the prose companion to the Dafny model under `dafny-models/`, which
formally specifies the verifier's observe → decide → WAL → apply state machine.
Where the model uses an `assume`, this document records whether the assumed
property is enforced by the implementation and how.

All changes and the regression tests below live on branch
`interop-safety-hardening` (pushed to `ethereum-optimism/optimism`).

## Repros / regression tests

Each hardening change and bug fix has a regression test, runnable with
`go test ./op-supernode/... ./op-interop-mon/...`:

| What it guards | Test(s) | File |
|---|---|---|
| `VerifiedDB.Rewind` bounds corruption on multi-page DBs | `TestVerifiedDB_RewindMultiPage` | `op-supernode/supernode/activity/interop/verified_db_test.go` |
| Halt on stale-logsDB / parent-hash / rewind-rejected conflicts | `TestProgress_HaltsOnStaleLogsDBConflict`, `TestProgress_HaltsOnParentHashMismatch`, `TestProgress_HaltsOnRewindRejected`, `TestProgress_HaltsOnRewindOverFinalized` | `.../interop/divergence_test.go` |
| Commit head-regression guard | `TestCommit_HeadRegression` | `.../interop/divergence_test.go` |
| Rewind chain-set mismatch halt | `TestBuildRewindPlan_ChainSetMismatch` | `.../interop/divergence_test.go` |
| Startup-handoff gap metric | `TestHandoffGap_MetricReflectsWindow` | `.../interop/divergence_test.go` |
| logsDB retention pruning | `TestPrune_*`, `TestPruneLogsDBs_*` | `.../interop/raftwallogdb/prune_test.go`, `.../interop/divergence_test.go` |
| Resume-time consistency check | `TestCheckResumeConsistency_*`, `TestRunLoop_Resume*` | `.../interop/resume_check_test.go` |
| Rewind error terminal-vs-transient classification | `TestIsCriticalRewindError` | `.../chain_container/chain_container_test.go` |
| Cross-replica divergence detection + edge cases | `TestCompareSuperRoots`, `TestCollectOnce_*` | `op-interop-mon/monitor/divergence_collector_test.go` |
| `op-interop-mon` Stop() nil-deref with metrics disabled | `TestStop_MetricsDisabledDoesNotPanic` | `op-interop-mon/monitor/service_test.go` |
| VN start gate (restart-loop freeze race) | `TestVirtualNode_Lifecycle/Start_gated_by_beforeStart_*` | `op-supernode/supernode/chain_container/virtual_node/virtual_node_test.go` |

## Background: the one load-bearing assumption

The entire verifier rests on one assumption: **the L2 chain is a deterministic
function of L1.** Every reorg scenario is handled correctly *under* that
assumption — the recorded `L1Inclusion` of each verified entry lets the loop
detect a stale L2 head and unwind it (see `observeRound`). The hardening below
covers the cases where the assumption *itself* breaks: an environment reset
under a kept data dir, a non-deterministic derivation bug, or finality advanced
over unverified state. None of these can occur on a correctly operated,
bug-free system — they are the boundaries of the assumption, not bugs inside it.

The design principle for all of them: **a state that can neither make progress
nor be unwound must halt, not retry.** A silent retry loop looks identical to a
transient RPC hiccup in logs and metrics; a halt sets
`supernode_interop_activity_state = 3` (halted), emits a categorized error
metric, logs remediation, and preserves the WAL entry for inspection.

## Hardening changes

| Area | Failure mode (old) | Behavior (new) | Code |
|------|--------------------|----------------|------|
| WAL'd Advance conflicts with sealed logsDB | Infinite 2s retry; `observeRound` (where reorg detection lives) starved by the pending transition | Halt `state_divergence` | `interop.go` `isDivergenceError`, `progress` |
| Resume on inconsistent local state | No check on the resume path (only cold start reconciled via `reconcileLogsDBTail`) | One-shot resume check halts on divergence-under-canonical-L1, proceeds on offline L1 reorg, retries while chains warm up | `resume_check.go` |
| Verified head regresses or changes hash at a fixed height | Silently committed | Halt `state_divergence` (`ErrHeadRegression`) — defense-in-depth mirroring the model's `Commit` precondition | `verified_db.go` `Commit` |
| Engine-reset rewind targets an entry whose chain set ≠ configured set (chain added/removed) | WAL'd plan wedges forever in `resetChainEnginesIfNeeded` | Halt at build time `state_divergence` (`ErrRewindChainSetMismatch`) | `interop.go` `buildRewindPlan` |
| Invalidation must rewind over the finalized head | Permanent error (`ErrRewindOverFinalizedHead`) flattened to a string and retried every backoff forever | Halt `rewind_over_finalized` — reachable via transitive invalidation | `interop.go` invalidation apply + `progress` |

Each halt category has a distinct remediation message because the operator
action differs: `state_divergence` usually means "wipe the data dir if the
environment was reset"; `rewind_over_finalized` means "a finalized block was
found invalid — investigate why finality advanced over unverified state."

### Resume consistency check (detail)

On restart with existing verified history (`resume_check.go`), before the first
verification round:

1. **Local invariant (no RPC):** with no pending WAL entry, every chain in the
   last verified entry must have its logsDB tip exactly equal to its verified
   head. Every apply path re-establishes this before clearing the WAL, so a
   mismatch is data loss or a foreign data dir — never a reorg.
2. **Chain check:** each verified head must still be canonical at its height.
   If it diverged **and** the recorded `L1Inclusion` is still canonical, that is
   the determinism-violation signature (no protocol reorg can change a verified
   block while its L1 source stays canonical) → halt. If `L1Inclusion` is *not*
   canonical, it is an offline L1 reorg → proceed; the round loop's existing
   rewind machinery handles it.

A pending WAL entry defers the check entirely (replay owns convergence;
divergence during replay is halt-classified in `progress`). The cold-start path
sets the check done because `reconcileLogsDBTail` already reconciles there.

## Confirmed invariants (verified, not just assumed)

### Deposit-only replacement blocks cannot contain valid executing messages

When the verifier invalidates a block, the chain is rebuilt with a **deposit-only**
replacement at the same height. Kona's consolidation treats replacement blocks
as cross-safe *without re-validating them*, on the assumption they carry no
executing messages. This assumption is **structurally enforced**, not merely
trusted:

- An executing message is only valid if `CrossL2Inbox.validateMessage` finds the
  message checksum's storage slot **warm**
  (`packages/contracts-bedrock/src/L2/CrossL2Inbox.sol`), which requires the slot
  to be pre-declared in the transaction's **EIP-2930 access list**.
- Deposit transactions (type `0x7E`) have **no access-list field** — confirmed in
  both `types.DepositTx` (Go, op-geth) and `TxDeposit` (Rust,
  `rust/op-alloy/.../transaction/deposit.rs`), and the deposit→EVM mapping
  (`rust/alloy-op-evm/src/tx.rs` `deposit_tx_env`) sets no access list.
- There is no in-transaction path to warm the slot either: only code running in
  `CrossL2Inbox`'s own context can warm its storage slots, its only
  storage-touching path is `validateMessage`→`_isWarm`, and that path reverts
  when the slot is cold (and a reverted frame's warm-set is rolled back per
  EIP-2929).

Therefore `validateMessage` always reverts inside a deposit, and deposit-only
blocks cannot contain valid executing messages. The kona assumption is sound.

### Multiple denials at one height are unreachable

`DenyList` storage is undefined for multiple denied blocks at the same height
(see comment in `chain_container/invalidation.go`). This case is unreachable in
normal operation: it would require invalidating a replacement block at a height
already denied, but replacement blocks are deposit-only and (by the invariant
above) cannot be message- or cycle-invalid, so the verifier never denies a
second block at that height.

## Audited surfaces beyond the verifier core

The verifier state machine was audited first; these are the adjacent surfaces,
with verdicts. Findings that became code changes are folded into the table
above and the metrics list below.

### Cross-implementation validity parity (vs kona) — clean

The supernode's message-validity rules were compared rule-by-rule against
kona's fault-proof validity logic (`rust/kona/crates/protocol/interop/`):
expiry window and boundary (`<` vs `<=`), timestamp ordering, the
activation + blockTime rule on both executing and initiating chains,
same-timestamp dependency resolution, Kahn-cycle detection scope, checksum /
identifier binding, and the uint32/uint64 identifier caps. **All match.** The
only non-equivalence is at `activationTimestamp == 0` with a sub-blocktime
timestamp — unreachable (Lagoon never activates at 0), and in the safe
direction (the supernode rejects what the proof would accept). A consensus
split where the supernode *accepts what the proof rejects* was not found on any
reachable input.

### Consumer safety semantics — no overstatement, one observability gap

Every consumer of `CurrentL1` honors the strict `> X` contract (no off-by-one
treating `>= X` as verified); `SafeL2Head` clamps the authority's verified head
to local-safe and never reports ahead of it. The one gap was the
**startup-handoff window** `[activationTimestamp, firstVerifiableTimestamp)`,
reported "verified" without being verified and previously invisible — now
exposed as `supernode_interop_handoff_gap_seconds` plus a `Warn` log when
non-zero (`recordHandoffGap`). The window is covered by the pre-activation /
startup-handoff trust assumption; making its width observable lets operators
alert when a reseeded node pushes it wide.

### Replica divergence — not reachable under correct operation

The verification decision is a pure function of `(L2 block data, canonical L1)`:
expiry/ordering/activation all compare block timestamps, never wall-clock; the
only `clock.Now`/sleep uses are backoff and latency metrics; the deny list is
*derived* from deterministic decisions. So two correct replicas of the same
chains converge, and divergence implies a determinism violation — which the
local halt guards catch. There is no leader among replicas and none is needed;
they are independent verifiers of a deterministic function. The residual gap is
*detection of a subtle inter-replica divergence that does not trip a local
guard*, addressed by the monitoring proposal (below), not by adding consensus.

### External mutation surface — single-writer holds by convention

The engine API (the only true mutation surface) is JWT-guarded and reached only
by the supernode's own VN derivation and its engine controller; rewinds pause
and stop the VN first and serialize via a CAS, so the two in-process writers
never race. Mutating op-node RPCs (`admin_*`, sequencer) are off by default and
absent from every supernode k8s overlay. **No externally-reachable, ungated,
mutation-capable surface exists under supported deployments.** The assumption is
enforced by *convention*, not code, and rests on two operator-controlled
invariants worth stating in runbooks:

1. `--rpc.enable-admin` stays off on the supernode (else `admin_startSequencer`
   / `admin_postUnsafePayload` become an ungated second writer).
2. No second engine-connected component (manual FCU, extra op-node, conductor,
   sequencer) is pointed at a chain's EL while the supernode owns it.

Operational note: the engine JWT is committed in plaintext in the k8s values for
both dev and prod deployments. The mitigating control is that the engine
authrpc port is not exposed as a cross-pod Service, so reaching it needs
in-cluster network access — but the committed secret is a defense-in-depth
weakness worth addressing in secret management.

## Trust boundary: synthetic-block visibility during rewind

The engine rewind (`engine_controller/rewind.go`) cannot truncate the EL
directly, so it fabricates a **synthetic block** (the target's parent with
mangled `ExtraData`, hence a different hash), forkchoice-updates the engine to
it — evicting the bad block from the canonical chain — then re-inserts the
genuine target and FCUs back. For a brief window the synthetic block is the
EL's unsafe head.

The supernode bounds exposure: during a rewind the chain container pauses and
stops the virtual node, and the RPC router reports not-ready, so no consumer
routed through the supernode can observe the synthetic block. The residual
exposure is a service holding a **direct** EL/engine connection that bypasses
the router and issues its own forkchoice updates concurrently with a rewind.

This sits within a reasonable operator trust model: **do not manually drive
your EL's forkchoice while the supernode owns it.** The supernode is the sole
forkchoice controller for the chains it runs (see `safety-labels.md` —
"CL-Centric Control"); a second stateful controller is a misconfiguration. The
external mutation-surface enumeration catalogs which services hold direct EL
access and confirms none does so under supported deployments.

## logsDB pruning (built)

The per-chain logsDB (raft-wal) was append-only with no steady-state pruning:
one fsynced entry per sealed block, ~86,400/chain/day at 1s blocks, growing
forever. It now prunes (`raftwallogdb.DB.Prune`, called from the verifier's
periodic loop via `pruneLogsDBs`).

Horizon: the verifier frontier minus a retention window of
`logsDBRetentionFactor × messageExpiryWindow` (2 × 7 days). One expiry window is
the correctness floor — an initiating message older than that can never be
referenced, and `verifyExecutingMessage` rejects it with `ErrMessageExpired`
*before* any logsDB read, so a pruned entry can never change a validation
outcome — and the extra factor is conservative margin. Mechanics: raft-wal
`DeleteRange(FirstIndex, indexFor(target))` head-truncates (segment-granular,
lazy); `Prune` never removes the tip and only removes a contiguous prefix;
`refreshCache` derives `firstBlock` from `FirstIndex()` so it persists across
restarts. Metrics: `supernode_logsdb_entries{chain_id}`,
`supernode_logsdb_pruned_total{chain_id}`, `supernode_logsdb_prune_horizon_timestamp`.

## Cross-replica divergence monitor (built)

Replicas can't diverge under correct operation, but a subtle divergence that
doesn't trip a local halt guard had no detector. `op-interop-mon` now has a
`ReplicaDivergenceCollector` (enabled by `--supernode-replica-endpoints` with ≥2
endpoints): it polls each replica's `supernode_syncStatus`, picks the minimum
finalized timestamp `T` (so a merely-lagging replica isn't flagged), fetches
`superroot_atTimestamp(T)` from each, and compares `Data.SuperRoot`. A mismatch
sets `op_interop_mon_replica_superroot_divergence`, logs the diverging groups,
and (with `--trigger-failsafe`) enables the supervisor failsafe. Requires no
supernode changes — the RPC already folds verified heads + deny-list state into
one hash.

## Re-review findings (second pass over previously-unexamined surfaces)

A second pass bug-hunted the superroot API, chain-container concurrency, and the
engine rewind state machine. Outcomes:

### Fixed

- **Deterministic rewind rejections were retried forever.** When the engine
  declared a rewind payload or forkchoice invalid (`ErrRewindSyntheticPayloadRejected`,
  `ErrRewindCanonicalPayloadRejected`, `ErrRewindFCURejected`), `RewindEngine`'s
  retry loop treated it as transient and looped every second forever — the same
  silent-wedge class as the verifier findings. These are now classified terminal
  in `isCriticalRewindError` (so `RewindEngine` returns promptly) and halt-classified
  in `progress` (`rewind_rejected`). This also subsumes the rare case where a
  synthetic rewind block fails Holocene/Jovian extraData validation (it surfaces
  as a visible halt instead of an infinite loop). Transient errors (transport,
  FCU non-convergence) still retry.

### Corrected (claim did not hold)

- **Synthetic-block extraData is NOT broken on current chains.** The rewind
  mutates the *last* extraData byte; version, length, and denominator are
  preserved and elasticity stays non-zero except when its low byte is exactly
  `0xff`, so Holocene/Jovian synthetic blocks pass header validation (consistent
  with the real-rewind acceptance test). Only pre-Holocene chains would
  deterministically reject, and none are in operation. The residual `0xff` edge
  is covered by the `rewind_rejected` halt above.

### Fixed: restart-loop freeze race + two production data races it surfaced

- **Restart-loop freeze race (chain_container).** `PauseAndStopVN` set the
  `pause` flag and stopped the current VN, but the restart loop checked `pause`
  *before* `vn.Start` (a check-then-act window), and `vn.Stop` is a no-op on a
  not-yet-started VN — so a VN created in that window started anyway, leaving a
  peer VN running during a multi-chain rewind. Fixed with a **start gate**:
  `vn.Start` consults `beforeStart` (`= !paused && !stopping`) under its own
  lock, atomically with the NotStarted→Running transition (virtual_node.go;
  wired in the restart loop via `SetBeforeStart`). Because that transition and
  `vn.Stop`'s state check both serialize on the VN lock, and `PauseAndStopVN`
  sets `pause` *before* calling `Stop`, the VN either observes the pause and
  aborts (`ErrVirtualNodeStartGated`) or completes its transition and is then
  reliably stopped — the window is closed. Deterministic repros:
  `TestVirtualNode_Lifecycle/Start_gated_by_beforeStart_*` (virtual_node_test.go).
- **Production data race in `virtualNode.Start` (pre-existing, surfaced by
  `-race`).** `innerErr`/`cancelErr` were written by the inner-node goroutine
  but read by the parent after `<-runCtx.Done()` with no happens-before. Fixed
  by delivering the inner result over a channel and receiving it after
  `n.Stop()`, which synchronizes both reads.
- **Production bounds-corruption in `VerifiedDB.Rewind` (pre-existing).** See the
  code-review section above — fixed and regression-tested
  (`TestVerifiedDB_RewindMultiPage`).

Still open: `pause` is a non-nestable bool, so a `RewindEngine` deferred
`Resume` can lift a concurrent multi-chain freeze (last-writer-wins). Fixing it
cleanly needs converting `pause` to a balanced counter with a Pause/Resume
pairing audit — tracked as a follow-up.

### `-race` cleanup surfaced two more production races

Making `chain_container` and `virtual_node` run `-race`-clean (mostly guarding
test mocks' shared fields) surfaced two more real production races:

- **Logging a VirtualNode value dumped the whole struct.** `c.log.Warn(...,
  "vn_id", vn, ...)` had no `String()` on `*simpleVirtualNode`, so the logger
  reflectively read every field (mutex, atomic state, inner) while the run
  goroutine mutated them. Added a stable `String()` returning the immutable
  `vnID` (also stops giant struct dumps in logs).
- **`c.rollupClient` close/replace was unguarded.** The restart loop's
  `attachInProcRollupClient` (close-old + assign-new) raced `Stop()`'s
  close. Guarded with a dedicated `rollupClientMu`.

Both `op-supernode` (all sub-packages) and `op-interop-mon` now pass
`go test -race`.

### Remaining superroot / rewind items scoped for owner review

- **Superroot optimistic (output, requiredL1) consistency.** Fixed the nil-output
  panic (`TestSuperroot_AtTimestamp_NilOptimisticOutputFailsNotPanic`). The
  remaining TOCTOU between the two reads is entangled with deny-list semantics
  (a denied height legitimately pairs the *original* optimistic output with the
  *replacement* L1; the original block's derivation L1 may be gone from SafeDB),
  so making it strictly atomic is a product decision — left for the owner.
- **Rewind no-op guard ancestry check.** The equal-height-different-hash
  finalized guard is fixed (`computeRewindTargets`,
  `TestEngineController_Rewind/target_at_finalized_height_with_different_hash`).
  The `unsafe.Number < target` no-op guard (which can skip a rewind when unsafe
  is on a wrong fork below target) needs real-EL ancestry validation — scoped
  for acceptance-test-backed work.
- **Superroot optimistic branch reads (output, requiredL1) from two
  un-snapshotted calls** (`OptimisticOutputAtTimestamp` then `OptimisticAt`),
  which can pin to different L2 blocks if the safe head moves between them →
  a mismatched (output root, required L1) pair the challenger consumes. The fix
  must preserve `OptimisticOutputAtTimestamp`'s deny-list-aware semantics while
  making the pair consistent.
- **Rewind no-op / finalized guards are number-only, not ancestry/hash-aware**
  (`rewind.go`): `unsafe.Number < target` skips a rewind even if `unsafe` is on a
  wrong fork; the finalized guard compares numbers, not hashes, so an
  equal-height-different-hash finalized block isn't rejected. Low likelihood,
  consensus-critical to get right.

### Minor / latent

- `DenyList.Close()` omits the `d.mu` lock every other method takes (benign with
  bbolt — yields a clean `ErrDatabaseNotOpen` — latent if the backend changes).
- `onReset` is a bare unsynchronized field (safe by init-ordering today).

## Code-review findings (review of this session's own diff + the surrounding logic it integrates with)

A `/code-review` pass over the diff, then a deeper pass over the *existing* code
the new code depends on, found and fixed:

- **`VerifiedDB.Rewind` zeroed its bounds on multi-page databases (pre-existing,
  high severity).** Rewind re-derived `firstTimestamp`/`lastTimestamp` with
  `Cursor.Last()` *inside* the delete transaction. After a tail delete large
  enough to leave empty trailing b-tree pages (which only rebalance at commit),
  `Cursor.Last()` returns nil, so the bounds were set to "empty / not
  initialized" while entries remained on disk — silently corrupting
  `FirstTimestamp()` and disabling the new Commit head-regression guard on the
  next re-verification. Existing tests used ≤6 entries (single page) and never
  hit it. Fixed: collect-then-delete (avoids the bbolt delete-during-iteration
  skip), then re-derive bounds in a *separate read transaction* after commit.
  Regression test: `TestVerifiedDB_RewindMultiPage` (800 entries).
- **`op-interop-mon` `Stop()` nil-deref panic (pre-existing, newly exposed).**
  `ms.collector.Stop()` had no nil guard, but the collector is only built when
  metrics are enabled (off by default). Running the new divergence collector
  with metrics off and shutting down cleanly panicked here — before the
  divergence collector / replica clients were cleaned up. Fixed with a nil guard
  mirroring `Start()`; regression test `TestStop_MetricsDisabledDoesNotPanic`.
- **Divergence collector hardening (this session's own new code):** a replica
  reporting `finalized=0` (booting/desynced) no longer drags the comparison
  point to 0 and blinds detection; replicas with mismatched dependency sets are
  no longer compared (they'd produce a *false* divergence → false failsafe
  trip); per-tick `context.WithTimeout` so a hung replica can't block the
  collector forever; per-replica RPCs fan out concurrently instead of 2N serial
  round-trips.
- **Reuse / structure:** the hand-rolled replica RPC client now delegates to the
  existing `op-service/sources.SuperNodeClient` (avoids wire-format drift);
  `Prune` was promoted from an optional type-assertion to a method on the
  `LogsDB` interface, so any future logsDB implementation is obligated to
  support retention pruning at compile time rather than silently regrowing.

Confirmed-correct by the review (no change needed): the raft-wal `Prune`
binary search and head-truncation (validated against raft-wal v0.4.2 internals —
`FirstIndex` advances and persists immediately, and the `number < firstBlock`
read guards prevent any pruned-block leak); the Commit guard vs idempotent-replay
ordering; the `resumeChecked` lifecycle; and the prune-vs-rewind single-goroutine
exclusion.

## Deferred / out of scope here

- **Interop `Reset` is a no-op.** `onChainReset` broadcasts to activities, but
  `Interop.Reset` is intentionally empty (`interop.go`): invalidation and rewind
  run synchronously through pending-transition apply, not callbacks. The
  onReset→Reset wiring is vestigial for interop; do not wire safety-critical
  cleanup expecting it to fire.

## Metrics added by this hardening

- `supernode_interop_activity_state` (pre-existing) — `3` = halted; alert on it.
- `supernode_interop_handoff_gap_seconds` — startup-handoff window width.
- `supernode_logsdb_entries{chain_id}` — logsDB retained-entry count (growth).
- Halt error categories on `supernode_activity_errors_total{activity="interop"}`:
  `state_divergence`, `rewind_over_finalized`, `history_unavailable`.
