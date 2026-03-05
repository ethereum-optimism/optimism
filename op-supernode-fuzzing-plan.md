# Fuzzing Campaign Plan for OP-Supernode

## Context

Runtime Verification previously conducted a fuzzing campaign for **op-supervisor** (deprecated) as part of a security audit. The campaign:
- `op-supervisor/supervisor/backend/cross_update_fuzz_test.go` (1,256 lines, 9 fuzz functions)
- `op-supervisor/supervisor/backend/chain_randomizer_test.go` (564 lines, random chain generation)

**op-supernode** replaces op-supervisor with a fundamentally different architecture: instead of event-driven safety-level promotions (cross-unsafe → cross-safe), it uses a **sequential timestamp-based verification loop** that processes blocks timestamp-by-timestamp and records verified results in a bbolt database.

This campaign targets invariant violations, edge cases in timestamp arithmetic, state corruption during rewind/reset, chain continuity violations, and DoS vectors.

---

## Fuzzing Targets

### Target 1: Interop Message Verification Algorithm (CRITICAL)
**File:** `op-supernode/supernode/activity/interop/algo.go`
**Functions:** `l1Inclusion`, `verifyInteropMessages`, `verifyExecutingMessage`

#### Code-Level Edge Cases

**1a. `ErrSkipped` fallback path**
When `OpenBlock` returns `types.ErrSkipped`, the code falls back to `FirstSealedBlock()`. Three sub-paths:
- `FirstSealedBlock()` fails → wraps original error
- `firstBlock.Number == expectedBlock.Number` + hash mismatch → marks `InvalidHeads[chain]` AND `L2Heads[chain]`
- `firstBlock.Number != expectedBlock.Number` → returns hard error
- **Critical**: First block is assumed to have NO executing messages. If a real first block has executing messages, they are silently skipped.

**1b. Block hash mismatch behavior**
A hash mismatch marks both `InvalidHeads[chainID]` AND `L2Heads[chainID]`. Fuzz verifies this dual-marking is consistent.

**1c. Map iteration non-determinism**
`execMsgs` is `map[uint32]*types.ExecutingMessage` — iteration order is non-deterministic. The algorithm sets `blockValid = false` and breaks on first invalid message. With multiple invalid messages in one block, different executions may flag different messages. Fuzz tests blocks with multiple invalid messages.

**1d. Missing chain silently skipped**
If `blocksAtTimestamp` includes a chain not in `i.logsDBs`, it's silently skipped. The resulting `Result` may not include all chains from input.

**1e. Expiry boundary exact values**
- `execMsg.Timestamp > executingTimestamp` → strictly greater triggers `ErrTimestampViolation` (equal timestamps are valid)
- `execMsg.Timestamp + ExpiryTime < executingTimestamp` → at boundary `==` is VALID
- **uint64 overflow**: `execMsg.Timestamp + ExpiryTime` could overflow if `execMsg.Timestamp` is near `math.MaxUint64`

**1f. Self-chain references not checked**
`verifyExecutingMessage` does NOT check for `execMsg.ChainID == executingChain`. A message on chain A can reference an initiating message also on chain A, passing if timestamps and checksum are valid.

#### Properties
- P1: Valid cross-chain messages never produce `InvalidHeads`
- P2: Every invalidation type is correctly detected
- P3: `Result.IsValid()` ↔ `len(InvalidHeads) == 0`
- P4: `execMsg.Timestamp + ExpiryTime` overflow doesn't cause false positive/negative
- P5: First block (ErrSkipped path) correctly handles hash mismatch
- P6: Block with multiple invalid messages still gets marked invalid (regardless of iteration order)
- P7: Missing chains in logsDBs are consistently excluded from Result

#### Fuzz Functions (7 total)
- `FuzzVerifyInteropMessagesValid` — valid states always pass (P1, P3)
- `FuzzVerifyInteropMessagesFails` — each invalidation type detected (P2)
- `FuzzVerifyExpiryBoundary` — timestamps at exact expiry boundary `ExpiryTime ± 1` (P4)
- `FuzzVerifyFirstBlockSkipped` — ErrSkipped path with valid/invalid first blocks (P5)
- `FuzzVerifyMultipleInvalidMessages` — blocks with multiple invalid messages (P6)
- `FuzzVerifyMissingChains` — chains not in logsDBs are excluded (P7)
- `FuzzResultProperties` — Result type methods: IsValid, IsEmpty, ToVerifiedResult (P34-P36)

---

### Target 2: Log Database Continuity & Loading (HIGH)
**File:** `op-supernode/supernode/activity/interop/logdb.go`
**Functions:** `loadLogs`, `verifyCanAddTimestamp`, `processBlockLogs`

#### Code-Level Edge Cases

**2a. Block skip silently passes**
When `latestBlock.Number >= block.Number`, loading is skipped without verifying hash matching. If the logsDB has block 5 with hash A but chain provides block 5 with hash B, it silently accepts.

**2b. Gap calculation edge**
`gap := ts - seal.Timestamp` — safe because the code returns early if `seal.Timestamp > ts`. But `gap < blockTime` only warns, doesn't error. Non-block-time-aligned timestamps can be processed.

**2c. First block parent handling**
Two separate branches:
- `isFirstBlock && blockNum > 0`: Seals a "virtual parent" block first, then processes logs with the real parent hash
- `blockNum == 0`: Actual genesis block — uses empty parent block and hash

**2d. Silent error in DecodeExecutingMessageLog**
`execMsg, _ := processors.DecodeExecutingMessageLog(l)` — errors are silently ignored. A malformed log could result in `nil` execMsg (valid — means not an executing message) but could also mask encoding bugs.

**2e. Activation timestamp special case**
If DB is empty but timestamp != activationTimestamp → `ErrPreviousTimestampNotSealed`. This enforces that the first timestamp processed must be exactly the activation timestamp.

#### Properties
- P9: Gap violations are always detected (gap > blockTime)
- P11: First block uses empty parent hash (genesis) or virtual parent seal (non-genesis)
- P12: After any error, the DB remains consistent (no partial writes)
- P13: Non-block-time-aligned gaps only warn, don't error

#### Fuzz Functions (2 total)
- `FuzzVerifyCanAddTimestamp` — boundary conditions in gap calculation (P9, P13)
- `FuzzProcessBlockLogs` — arbitrary receipts with varying log counts, first-block handling (P11, P12)

---

### Target 3: VerifiedDB Sequential Enforcement & Rewind (HIGH)
**File:** `op-supernode/supernode/activity/interop/verified_db.go`
**Functions:** `Commit`, `Rewind`, `RewindAfter`, `Get`, `Has`, `LastTimestamp`

#### Code-Level Edge Cases
- Big-endian uint64 key encoding — must be lexicographically correct
- First commit can be at any timestamp; subsequent commits must be sequential
- Rewind to timestamp 0 (delete everything)
- Rewind to timestamp beyond last committed (no-op)
- JSON serialization/deserialization round-trip of `VerifiedResult`
- Interleaved commit/rewind patterns
- `RewindAfter(ts)` calls `Rewind(ts + 1)` — keeps entries at `ts`, deletes strictly after

#### Properties
- P15: `Commit(result)` succeeds iff `result.Timestamp == lastTimestamp + 1` (or first commit at any timestamp)
- P16: After `Rewind(ts)`, `LastTimestamp()` returns `ts - 1` (or uninitialized if all deleted)
- P17: After `Rewind(ts)`, `Get(t)` errors for all `t >= ts`
- P18: After `Rewind(ts)`, `Commit(ts)` succeeds (re-commit from rewind point)
- P19: `ErrAlreadyCommitted` and `ErrNonSequential` are correctly distinguished
- P20: JSON round-trip preserves all VerifiedResult fields (including through close/reopen)

#### Fuzz Functions (3 total)
- `FuzzVerifiedDBCommitRewind` — random sequences of commit/rewind operations (P15-P20)
- `FuzzVerifiedDBFirstCommit` — first commit at any timestamp, sequential rule after (P15, P18)
- `FuzzVerifiedDBPersistence` — data survives close/reopen of bbolt DB (P20)

---

### Target 4: DenyList / Block Invalidation (MEDIUM)
**File:** `op-supernode/supernode/chain_container/invalidation.go`

#### Properties
- P21: `Contains(h, hash)` returns true iff `Add(h, hash)` was previously called
- P22: `Add` is idempotent
- P23: Hashes at different heights are isolated
- P24: Concatenated 32-byte hash storage handles boundary alignment correctly

#### Fuzz Functions (2 total)
- `FuzzDenyListAddContains` — random add/contains sequences with in-memory oracle (P21-P24)
- `FuzzDenyListConcurrent` — parallel operations from multiple goroutines for thread safety

---

### Target 5: Engine Rewind Algorithm (MEDIUM)
**File:** `op-supernode/supernode/chain_container/engine_controller/rewind.go`

Note: 13+ existing test cases cover error taxonomy. Fuzzing adds coverage for random state combinations.

#### Properties
- P25: Rewind never succeeds when target is before finalized head
- P26: After successful rewind, unsafe head == target block
- P27: After successful rewind, finalized head is unchanged (or clamped to target)

#### Fuzz Functions (2 total)
- `FuzzRewindToTimestamp` — full rewind flow with random engine states (P25-P27)
- `FuzzComputeRewindTargets` — clamping logic in isolation (P25, P27)

---

### Target 6: End-to-End Interop Progress Loop (HIGH)
**File:** `op-supernode/supernode/activity/interop/interop.go`
**Functions:** `progressInterop`, `handleResult`, `checkChainsReady`, `Reset`

#### Code-Level Edge Cases

**6a. Reset race window**
`Reset` acquires `mu.Lock()` but `progressAndRecord()` doesn't hold the lock during `progressInterop()`. A Reset could occur between `loadLogs` and `verifyFn`, corrupting the logsDB state mid-verification.

**6b. resetLogsDB clear-vs-rewind boundary**
`resetLogsDB` takes `invalidatedBlock` and computes `targetBlockID` as the parent of the invalidated block:
- `firstBlock.Number > targetBlockID.Number` → clear
- `firstBlock.Number <= targetBlockID.Number` → rewind
Edge case: when `firstBlock.Number == targetBlockID.Number`, it rewinds to the first block.

**6c. handleResult empty-vs-valid-vs-invalid**
Empty results → no-op. Invalid results → invalidate blocks, return without L1 update. Valid results → commit and update `currentL1` to `result.L1Inclusion`.

**6d. cycleVerifyFn (step 4 of progressInterop)**
After `verifyFn`, `progressInterop` runs `cycleVerifyFn` and merges any invalid heads from cycle verification into the result. This is set to `verifyCycleMessages` by default.

**6e. resetVerifiedDB uses RewindAfter**
`resetVerifiedDB(timestamp)` calls `verifiedDB.RewindAfter(timestamp)`, which internally calls `Rewind(timestamp + 1)`. This **keeps** entries at `timestamp` and deletes entries strictly after.

**6f. checkChainsReady goroutine handling**
Queries all chains in parallel via goroutines. If one chain errors, the function returns immediately. The channel is sized `len(i.chains)` so no goroutine blocks, but results are discarded.

#### Properties
- P28: Timestamps are processed strictly sequentially (no gaps, no repeats)
- P29: Valid results are committed; invalid results trigger block invalidation
- P30: Empty results are no-ops (do not modify state)
- P31: After invalidation, the interop loop can resume from the same timestamp
- P32: `Reset` correctly rewinds both logsDB and verifiedDB

#### Fuzz Functions (4 total)
- `FuzzProgressInteropValid` — valid multi-chain states always commit (P28, P29)
- `FuzzProgressInteropInvalid` — invalid messages trigger correct invalidation (P29, P31)
- `FuzzProgressInteropReset` — reset at various points doesn't corrupt state (P32)
- `FuzzHandleResultEmpty` — empty results are true no-ops (P30)

---

### Target 7: Interop Type Properties (LOW)
**File:** `op-supernode/supernode/activity/interop/types.go`

#### Properties
- P34: `Result.IsValid()` == `(len(InvalidHeads) == 0)`
- P35: `ToVerifiedResult()` strips invalid heads, preserves `Timestamp`, `L1Inclusion`, `L2Heads`
- P36: Empty results correctly detected (`L1Inclusion` is zero AND both maps empty)

Tested by `FuzzResultProperties` in `fuzz_algo_test.go`.

---

## Implemented Files

| File | Fuzz Functions | Lines |
|------|---------------|-------|
| `interop/fuzz_algo_test.go` | 7 (Valid, Fails, ExpiryBoundary, FirstBlockSkipped, MultipleInvalid, MissingChains, ResultProperties) | ~810 |
| `interop/fuzz_verified_db_test.go` | 3 (CommitRewind, FirstCommit, Persistence) | ~330 |
| `interop/fuzz_logdb_test.go` | 2 (VerifyCanAddTimestamp, ProcessBlockLogs) | ~240 |
| `interop/fuzz_interop_test.go` | 4 (Valid, Invalid, Reset, HandleResultEmpty) | ~400 |
| `chain_container/fuzz_invalidation_test.go` | 2 (AddContains, Concurrent) | ~225 |
| `chain_container/engine_controller/fuzz_rewind_test.go` | 2 (RewindToTimestamp, ComputeRewindTargets) | ~215 |

### Mock Strategy
- **`fuzzMockLogsDB`** (in `fuzz_algo_test.go`): Per-block configurable behavior via maps, no-op mutating methods. Designed for high-speed fuzzing.
- **`trackingMockLogsDB`** (in `fuzz_logdb_test.go`): Counts AddLog/SealBlock calls, records parameters for verification.
- **`mockChainContainer`** (from `interop_test.go`): Reused by `fuzz_interop_test.go` for E2E tests with `invalidateBlockCalls` tracking.
- **`mockL2`** (from `engine_controller_test.go`): Full L2 state simulation with FCU tracking for rewind tests.

### Running the Tests

```bash
# Quick smoke test (10 seconds per target)
go test -run '^$' -fuzz=FuzzVerifyInteropMessagesValid -fuzztime=10s ./op-supernode/supernode/activity/interop/

# Extended campaign (5 minutes per target)
go test -run '^$' -fuzz=FuzzVerifyInteropMessagesValid -fuzztime=5m ./op-supernode/supernode/activity/interop/

# Run all unit tests to ensure no regressions
cd op-supernode && go test ./...
```

---

## Summary

| # | Target | File | Fuzz Functions | Properties | Priority |
|---|--------|------|---------------|------------|----------|
| 1 | Interop Algo | `fuzz_algo_test.go` | 7 | P1-P7, P34-P36 | CRITICAL |
| 2 | LogsDB | `fuzz_logdb_test.go` | 2 | P9, P11-P13 | HIGH |
| 3 | VerifiedDB | `fuzz_verified_db_test.go` | 3 | P15-P20 | HIGH |
| 4 | DenyList | `fuzz_invalidation_test.go` | 2 | P21-P24 | MEDIUM |
| 5 | Engine Rewind | `fuzz_rewind_test.go` | 2 | P25-P27 | MEDIUM |
| 6 | E2E Interop | `fuzz_interop_test.go` | 4 | P28-P32 | HIGH |

**Total: 6 targets, 20 fuzz functions, 32 properties**

### Potential Findings (from code analysis)
1. **Self-chain references not checked** — `verifyExecutingMessage` doesn't reject messages referencing the executing chain itself
2. **Block skip doesn't verify hash** — `loadLogs` skips loading when `latestBlock.Number >= block.Number` without checking hash match
3. **Silent error in DecodeExecutingMessageLog** — malformed logs result in nil execMsg, silently treated as non-executing
4. **uint64 overflow in expiry check** — `execMsg.Timestamp + ExpiryTime` could overflow near `math.MaxUint64`
