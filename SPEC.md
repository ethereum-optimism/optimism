# SPEC: Dafny-model invariant assertion helpers for op-supernode tests

Owner: architect.
Implementers and reviewers treat this file as read-only; questions go through DECISIONS.md.

## Overview

`op-supernode/dafny-models/` contains a Dafny model of the interop activity (`Interop.dfy`), its persistence layers (`VerifiedDB.dfy`, `LogsDB.dfy`), the chain-container interface (`ChainContainer.dfy`), shared datatypes (`Types.dfy`), and utility predicates (`Utils.dfy`).
The model states invariants (`Valid()` predicates), structural validity predicates, and method pre/postconditions that the real Go code in `op-supernode/supernode/activity/interop/` is intended to satisfy.

This feature adds Go test assertion helpers that check those invariants on the real Go types at test time.
Every helper carries a doc comment naming the exact Dafny file and predicate it mirrors, conjunct by conjunct.
The `.dfy` files are never modified.

## Code placement

All new code lives in package `interop` at `op-supernode/supernode/activity/interop/`, in new files prefixed `dafny_check_`.
Rationale: the predicates quantify over unexported state (`Interop.verifiedDB`, `Interop.logsDBs`, `Interop.chains`, the bbolt buckets inside `VerifiedDB`), so the checkers must be in-package.
Non-test (`.go`, not `_test.go`) files are used for the checkers themselves, following the established `interop_test_access.go` pattern ("test/debug only, keep the boundary auditable"), so that integration tests outside the package can call the exported entry points on a `*Interop`.
Checker self-tests are ordinary `_test.go` files in the same package.

## API shape

Two layers:

1. `Check*` functions: pure checkers that return `error` (`nil` = predicate holds).
   Each violated conjunct contributes one wrapped error via `errors.Join`, with a message citing the Dafny source (e.g. `"VerifiedDB.dfy Valid() conjunct (1): committed timestamps not sequential: gap at 1005"`).
   Checkers never panic on nil/empty inputs; structural impossibility is reported as a violation.
2. `Assert*` wrappers: take a `t` (minimal local interface satisfied by `*testing.T`: `Helper()`, `Errorf(string, ...any)`, `FailNow()`), call the corresponding `Check*`, and fail the test with the full violation list.

Top-level entry point, mirroring the requires/ensures of `Interop.ProgressAndRecord` in `Interop.dfy`:

```go
// AssertInvariants checks Interop.Valid() && PendingTransitionIsConsistent()
// from Interop.dfy on the live Interop instance.
func AssertInvariants(t dafnyT, i *Interop)
```

Tests call it before and after exercising `progressAndRecord` / `applyPendingTransition` / `applyRewindPlan`.

Model ghost constants are carried in an explicit params struct so datatype-level checkers stay pure:

```go
// ModelParams instantiates the ghost constants of Types.dfy.
type ModelParams struct {
    ActivationTimestamp uint64                    // Types.dfy ACTIVATION_TIMESTAMP
    ChainIDs            map[eth.ChainID]struct{}  // Types.dfy CHAIN_IDS
}
```

`modelParamsFromInterop(i *Interop) ModelParams` derives them from a live instance.

## Model-to-Go mapping (binding decisions)

These mappings are normative for all helpers; deviations require a DECISIONS.md ruling.

- `Types.ACTIVATION_TIMESTAMP` maps to the Go verifier's first verifiable timestamp, i.e. `i.firstVerifiableTimestamp()` (the verifiedDB's `FirstTimestamp` when commits exist, else `verificationStartTimestamp`), NOT necessarily `i.activationTimestamp`.
  Rationale: the model's `Valid()` requires `activationTimestamp in verifiedDB.db` whenever the DB is non-empty and `NextTimestamp()` falls back to `activationTimestamp` when empty; in Go the first committed timestamp is `verificationStartTimestamp`, which may exceed the protocol activation timestamp.
  `ModelParams.ActivationTimestamp` therefore carries the first-verifiable timestamp.
- `Types.CHAIN_IDS` maps to the key set of `i.chains` (equivalently `i.logsDBs`; their equality is itself a checked conjunct).
- `Option<T>` maps to Go pointers (`*uint64`, `*Result`, `*RewindPlan`) or `(value, bool)` pairs (`LastTimestamp() (uint64, bool)`, `LatestSealedBlock() (eth.BlockID, bool)`).
- `Types.Decision` maps to `interop.Decision` (`Wait/Advance/Invalidate/Rewind` ↔ `DecisionWait/DecisionAdvance/DecisionInvalidate/DecisionRewind`).
- `Types.StepOutput` (algebraic) maps to Go `StepOutput{Decision, Result}`: `WaitOutput` ↔ `DecisionWait`, `AdvanceOutput(r)` ↔ `{DecisionAdvance, r}`, `InvalidateOutput(r)` ↔ `{DecisionInvalidate, r}`, `RewindOutput` ↔ `{DecisionRewind, _}` (Result ignored).
- `Types.RoundObservation` field mapping: model `l1Consistent` ↔ Go `!obs.L1NeedsRewind` (accepted L1 inclusion still canonical → rewind when false); model `l2sConsistent` ↔ Go `obs.L1Consistent` (frontier L1 heads mutually consistent → wait when false).
  This cross-naming follows from comparing `ProgressInterop` in Interop.dfy (`!l2sConsistent => Wait`, `!l1Consistent => Rewind`) with `checkPreconditions` in interop.go (`L1NeedsRewind => Rewind`, `!L1Consistent => Wait`).
  Go `Paused` is omitted from the model; checkers skip model checks that are conditional on a non-paused round when `obs.Paused` is true and document this.
- `Types.BlockSeal` maps to `suptypes.BlockSeal` (`op-supervisor/supervisor/types`); `seal.id` ↔ `eth.BlockID{Hash: seal.Hash, Number: seal.Number}`.
- `nat` maps to `uint64`; model subtraction is always guarded (e.g. `ACTIVATION_TIMESTAMP < plan.rewindAtOrAfter` before `rewindAtOrAfter - 1`), and checkers must preserve those guards to avoid uint64 underflow.
- The model's `VerifiedDB.db: map<nat, VerifiedResult>` maps to the bbolt `verified` bucket; checkers enumerate it via a new in-package snapshot helper (see T2) rather than trusting the cached `firstTimestamp`/`lastTimestamp` fields, because the model invariant explicitly ties the cache to the map contents.
- Where the Go code is more lenient than the model (e.g. `applyPendingTransition` tolerates `Decision: DecisionInvalidate` with nil `Result`, while `ValidPendingTransition` requires it present), helpers enforce the model.
  A test that feeds such a state to a checker must expect a violation; whether the production leniency is a bug is an audit finding, not something the helper hides.

## Predicate inventory and helper names

| Dafny source | Predicate / contract | Go helper |
|---|---|---|
| Types.dfy | `ValidRewindPlan(plan)` | `CheckValidRewindPlan(p ModelParams, plan RewindPlan) error` |
| Types.dfy | `ValidPendingTransition(pending)` | `CheckValidPendingTransition(p ModelParams, pending PendingTransition) error` |
| Types.dfy | `ValidStepOutput(output, obs)` | `CheckValidStepOutput(p ModelParams, output StepOutput, obs RoundObservation) error` |
| Types.dfy | `ValidRoundObservation(obs)` | `CheckValidRoundObservation(p ModelParams, obs RoundObservation) error` |
| Utils.dfy | `Sequential(m)` | `checkSequential(keys []uint64) error` (building block) |
| VerifiedDB.dfy | `VerifiedDB.Valid()` (4 conjuncts: Sequential, key==Timestamp field, lastTimestamp cache consistency, per-chain monotone l2Heads numbers) | `CheckVerifiedDBValid(v *VerifiedDB) error` |
| LogsDB.dfy | `FirstSealedBlock`/`LatestSealedBlock`/`FindSealedBlock` axioms (number matches key, strictly increasing timestamps, first/latest bracket the sealed range) | `CheckLogsDBSealsWellFormed(db LogsDB) error` |
| ChainContainer.dfy | `FetchReceipts` postcondition `info.id == blockID` | `CheckFetchReceiptsPost(blockID eth.BlockID, info eth.BlockInfo) error` |
| Interop.dfy | `Interop.Valid()` | `CheckInteropValid(i *Interop) error` |
| Interop.dfy | `DBsInSyncUpTo(chainID, upperTS)` | `CheckDBsInSyncUpTo(i *Interop, chainID eth.ChainID, upper uint64) error` |
| Interop.dfy | `DBsInSync(chainID)` | `CheckDBsInSync(i *Interop, chainID eth.ChainID) error` |
| Interop.dfy | `AllDBsInSyncUpTo(upper)` / `AllDBsInSync()` | `CheckAllDBsInSyncUpTo(i *Interop, upper uint64) error` / `CheckAllDBsInSync(i *Interop) error` |
| Interop.dfy | `AdvancesVerifiedDB(ts, blocksAtTS)` | `CheckAdvancesVerifiedDB(i *Interop, ts uint64, blocks map[eth.ChainID]eth.BlockID) error` |
| Interop.dfy | `AdvancesLogsDB(ts, chainID, newBlock)` / `AdvancesAllLogsDBs` | `CheckAdvancesLogsDB(...)` / `CheckAdvancesAllLogsDBs(...)` |
| Interop.dfy | `PlanConsistentWithVerified(plan)` | `CheckPlanConsistentWithVerified(i *Interop, plan RewindPlan) error` |
| Interop.dfy | `PlanConsistentWithLogs(plan, chainID)` | `CheckPlanConsistentWithLogs(i *Interop, plan RewindPlan, chainID eth.ChainID) error` |
| Interop.dfy | `RewoundVerifiedDB(plan)` | `CheckRewoundVerifiedDB(i *Interop, plan RewindPlan) error` |
| Interop.dfy | `RewoundLogsDB(plan, chainID)` | `CheckRewoundLogsDB(i *Interop, plan RewindPlan, chainID eth.ChainID) error` |
| Interop.dfy | `TransitionConsistentWithVerified(pending)` | `CheckTransitionConsistentWithVerified(i *Interop, pending PendingTransition) error` |
| Interop.dfy | `TransitionConsistentWithLogs(pending)` | `CheckTransitionConsistentWithLogs(i *Interop, pending PendingTransition) error` |
| Interop.dfy | `PendingTransitionIsConsistent()` | `CheckPendingTransitionIsConsistent(i *Interop) error` |
| Interop.dfy | `OutputConsistentWithVerified(output, obs)` | `CheckOutputConsistentWithVerified(i *Interop, output StepOutput, obs RoundObservation) error` |
| Interop.dfy | `OutputConsistentWithLogs(output, obs)` | `CheckOutputConsistentWithLogs(i *Interop, output StepOutput, obs RoundObservation) error` |
| Interop.dfy | `ObservationConsistentWithVerified(obs)` | `CheckObservationConsistentWithVerified(i *Interop, obs RoundObservation) error` |
| Interop.dfy | `ObservationConsistentWithLogs(obs)` | `CheckObservationConsistentWithLogs(i *Interop, obs RoundObservation) error` |
| Interop.dfy | `ProgressAndRecord` requires/ensures (`Valid() && PendingTransitionIsConsistent()`) | `CheckInvariants(i *Interop) error` + `AssertInvariants(t, i)` |

Composite checkers respect the model's requires-chain: e.g. `CheckInteropValid` runs `CheckVerifiedDBValid` first and short-circuits dependent conjuncts when prerequisites already failed, so violation reports stay readable.
`CheckDBsInSyncUpTo` iterates `t` from `ModelParams.ActivationTimestamp` to `upper` using `verifiedDB.Get(t)` and `logsDBs[chainID].FindSealedBlock`; this is O(range) and intended for test workloads only.

## Requirements

R1. Every ghost predicate and every checkable `ensures`/class invariant listed in the inventory above has a Go checker; nothing in the inventory is silently dropped.
R2. Each checker's doc comment names the Dafny file and predicate (e.g. `// CheckVerifiedDBValid mirrors VerifiedDB.Valid() in op-supernode/dafny-models/VerifiedDB.dfy.`) and each violation message identifies the violated conjunct.
R3. Checkers are read-only: they must not mutate `Interop`, `VerifiedDB`, or `LogsDB` state (bbolt `View` transactions only, no writes, no `Commit`/`Rewind`/`Clear` calls).
R4. For every checker there are unit tests covering at least one passing state and one violating state per conjunct group (violations injected via in-package access, e.g. direct bbolt bucket writes or mock `LogsDB` implementations).
R5. `AssertInvariants` is exercised from at least one existing-style unit test that drives a real transition (`applyPendingTransition` advance, invalidate, and rewind paths), asserting invariants before and after.
R6. `go build ./...` and `go test ./op-supernode/supernode/activity/interop/...` (run from the repo root) pass at every task boundary.
R7. No `.dfy` file is modified; no production code path (anything reachable from `New`/`Start`) calls the checkers.

## Non-goals

- No runtime/production enforcement of invariants; this is test-time tooling only.
- No checking of Dafny lemmas (`Utils.dfy` lemmas, framing lemmas in `Interop.dfy`): they are proof artifacts, not runtime properties.
- No coverage of model methods that are pure I/O stubs with no `ensures` (e.g. `PruneDeniedAtOrAfterTimestamp`, `RewindEngine`, `InvalidateBlock`).
- No fuzz harnesses or property-based campaigns (natural follow-up, out of scope here).
- No wiring into op-e2e / op-acceptance-tests; only op-supernode package tests.
- No Go-side invariants beyond the model (e.g. `firstTimestamp` cache checks not present in the Dafny model).

## Design decisions

D1. Checkers live in package `interop` as non-test files (test-access pattern), so unexported state is reachable and external integration tests can still call exported entry points. (See "Code placement".)
D2. `Check*`-returns-error / `Assert*`-wraps split keeps the predicates composable and the failure output complete (`errors.Join`, one error per conjunct).
D3. `ModelParams` carries the model's ghost constants explicitly; `ActivationTimestamp` is bound to the Go first-verifiable timestamp (see mapping section) — this is the single most error-prone mapping and is fixed here, not per-helper.
D4. VerifiedDB enumeration is done with a new unexported `allVerified() (map[uint64]VerifiedResult, error)` method on `*VerifiedDB` (bbolt `View` over the `verified` bucket), defined in the `dafny_check_verifieddb.go` file, so the checker validates the cache fields against the actual store as the model demands.
D5. Helpers enforce the model even where Go is more lenient; discrepancies surface as test failures at the call site, which is the point of the exercise.
D6. The `t` parameter is a minimal local interface (`dafnyT`), not `*testing.T`, so helpers work under `testing.T`, `testing.F`-derived `*testing.T`, and testify-style wrappers.

## Tasks

### T1 — ModelParams and Types.dfy structural predicate checkers
Files: `op-supernode/supernode/activity/interop/dafny_check_types.go`, `dafny_check_types_test.go`.
Build: `ModelParams`, `modelParamsFromInterop`, `dafnyT` interface, the `errors.Join`-based violation reporting convention, `CheckValidRewindPlan`, `CheckValidPendingTransition`, `CheckValidStepOutput`, `CheckValidRoundObservation`, plus their `Assert*` wrappers.
These four predicates read only their arguments and `ModelParams` (no DB access), so this slice has no test-fixture burden.
Acceptance: doc comments reference `Types.dfy` predicates conjunct by conjunct; unit tests cover pass + at least one violation per conjunct group of each predicate; `go build ./...` and the interop package tests pass.

### T2 — VerifiedDB.Valid checker
Files: `op-supernode/supernode/activity/interop/dafny_check_verifieddb.go`, `dafny_check_verifieddb_test.go`.
Build: unexported `allVerified()` snapshot method on `*VerifiedDB` (read-only bbolt `View`), `checkSequential` (mirrors `Utils.dfy Sequential`), `CheckVerifiedDBValid` covering all four `VerifiedDB.Valid()` conjuncts (sequential keys, `Timestamp` field == key, `lastTimestamp`/`initialized` cache consistency with store contents, per-chain monotone `L2Heads` block numbers), plus `AssertVerifiedDBValid`.
Tests: positive path via `OpenVerifiedDB` in a temp dir + sequential `Commit`s; negative paths by writing malformed entries directly into the bbolt bucket (in-package access to `v.db`).
Acceptance: R2–R4 satisfied for this checker; builds and package tests pass.

### T3 — LogsDB and ChainContainer contract checkers
Files: `op-supernode/supernode/activity/interop/dafny_check_logsdb.go`, `dafny_check_logsdb_test.go`.
Build: `CheckLogsDBSealsWellFormed(db LogsDB) error` mirroring the `LogsDB.dfy` function axioms — for the sealed range `[FirstSealedBlock().Number, LatestSealedBlock().Number]`: `FindSealedBlock(n)` (when found) has `Number == n`, timestamps strictly increase with block number, and first/latest agree with `FindSealedBlock` at their own numbers; `CheckFetchReceiptsPost(blockID, info)` mirroring the `FetchReceipts` `ensures info.id == blockID` axiom in `ChainContainer.dfy`; `Assert*` wrappers.
Tests: positive path against a temp `raftwallogdb` (or sequential mock) instance; negative paths via a stateful mock `LogsDB` (pattern exists in `logdb_test.go` / `interop_test.go`).
Acceptance: R2–R4 satisfied; builds and package tests pass.

### T4 — Interop.Valid and DB-sync checkers
Files: `op-supernode/supernode/activity/interop/dafny_check_interop.go`, `dafny_check_interop_test.go`.
Build: `CheckInteropValid` (all `Interop.Valid()` conjuncts from `Interop.dfy`: chains/logsDBs key-set equality with `ModelParams.ChainIDs`, distinct logsDBs values, embedded `CheckVerifiedDBValid`, first-verifiable timestamp present when DB non-empty, all committed timestamps ≥ it, every committed result's `L2Heads` key set == chain set, stored pending transition passes `CheckValidPendingTransition`), `CheckDBsInSyncUpTo`, `CheckDBsInSync`, `CheckAllDBsInSyncUpTo`, `CheckAllDBsInSync`, with `Assert*` wrappers.
Tests: construct `*Interop` directly in-package (temp-dir `VerifiedDB`, mock `LogsDB`s) for pass and violation cases, including a logsDB/verifiedDB divergence caught by `CheckDBsInSync`.
Acceptance: R2–R4 satisfied; builds and package tests pass.

### T5 — Transition consistency checkers and top-level AssertInvariants
Files: `op-supernode/supernode/activity/interop/dafny_check_transition.go`, `dafny_check_transition_test.go`.
Build: `CheckAdvancesVerifiedDB`, `CheckAdvancesLogsDB`, `CheckAdvancesAllLogsDBs`, `CheckPlanConsistentWithVerified`, `CheckPlanConsistentWithLogs`, `CheckRewoundVerifiedDB`, `CheckRewoundLogsDB`, `CheckTransitionConsistentWithVerified`, `CheckTransitionConsistentWithLogs`, `CheckPendingTransitionIsConsistent` (full match over stored pending decision per `Interop.dfy`), then `CheckInvariants(i) = CheckInteropValid + CheckPendingTransitionIsConsistent` and `AssertInvariants(t, i)`.
Tests: pass/violation cases per checker; one test drives `AssertInvariants` on a healthy instance and on an instance with a stored inconsistent pending transition.
Acceptance: R1 fully discharged for `Interop.dfy` state predicates; R2–R4 satisfied; builds and package tests pass.

### T6 — Round output and observation consistency checkers
Files: `op-supernode/supernode/activity/interop/dafny_check_round.go`, `dafny_check_round_test.go`.
Build: `CheckOutputConsistentWithVerified`, `CheckOutputConsistentWithLogs`, `CheckObservationConsistentWithVerified`, `CheckObservationConsistentWithLogs`, with `Assert*` wrappers, honoring the `l1Consistent`/`l2sConsistent` ↔ `L1NeedsRewind`/`L1Consistent` mapping and the Paused skip rule from the mapping section.
Tests: pass/violation cases per checker, including the rewind case (`!l1Consistent` with `lastVerifiedTS - 1` present/absent in the DB).
Acceptance: R1 fully discharged (whole inventory now covered); R2–R4 satisfied; builds and package tests pass.

### T7 — Wire AssertInvariants into existing transition tests
Files: existing tests only, e.g. `op-supernode/supernode/activity/interop/interop_test.go`, `startup_test.go`, `decide_test.go` (smallest reasonable touch set; no checker changes except bug fixes surfaced by wiring, each as its own commit).
Build: add `AssertInvariants` (and where natural, `AssertOutputConsistentWith*` / `AssertObservationConsistentWith*`) calls before/after real transitions: advance, invalidate, and rewind paths through `applyPendingTransition`, plus at least one `observeRound`/`progressInterop` call site.
Acceptance: R5 satisfied; full `go test ./op-supernode/supernode/activity/interop/...` green from the repo root; if a wiring-discovered model/code discrepancy cannot be resolved by fixing the checker, it is recorded as a question in DECISIONS.md instead of weakening the checker silently.
