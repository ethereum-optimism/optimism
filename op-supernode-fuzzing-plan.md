# Fuzzing Campaign Plan for OP-Supernode (Refined)

## Context

Runtime Verification previously conducted a fuzzing campaign for **op-supervisor** (deprecated) as part of a security audit. The campaign:
- `op-supervisor/supervisor/backend/cross_update_fuzz_test.go` (1,256 lines, 9 fuzz functions)
- `op-supervisor/supervisor/backend/chain_randomizer_test.go` (564 lines, random chain generation)

**op-supernode** replaces op-supervisor with a fundamentally different architecture: instead of event-driven safety-level promotions (cross-unsafe → cross-safe), it uses a **sequential timestamp-based verification loop** that processes blocks timestamp-by-timestamp and records verified results in a bbolt database.

This campaign targets invariant violations, edge cases in timestamp arithmetic, state corruption during rewind/reset, chain continuity violations, and DoS vectors.

---

## Reusable Components Analysis

### A. Directly Reusable (import as-is)

| Component | Location | Usage |
|-----------|----------|-------|
| `AddFuzzerFunctions()` | `op-service/testutils/fuzzerutils/fuzzer_functions.go` | Custom fuzz handlers for `*big.Int`, `*common.Hash`, `*common.Address` |
| `RandomHash()` | `op-service/testutils/random.go` | Generate random `common.Hash` |
| `RandomBlockRef()` | `op-service/testutils/random.go` | Generate random `eth.L1BlockRef` |
| `RandomL2BlockRef()` | `op-service/testutils/random.go` | Generate random `eth.L2BlockRef` |
| `NextRandomL2Ref()` | `op-service/testutils/random.go` | Generate sequential L2 block refs with proper parent linkage |
| `RandomLog()` | `op-service/testutils/random.go` | Generate random geth `*types.Log` |
| `RandomData()` | `op-service/testutils/random.go` | Generate random byte slices |

### B. Already in op-supernode Tests (reuse directly)

| Component | Location | Description |
|-----------|----------|-------------|
| `algoMockLogsDB` | `interop/algo_test.go` | Minimal mock with `OpenBlock`, `FirstSealedBlock`, `Contains` stubs |
| `mockLogsDB` | `interop/logdb_test.go` | Full mock with call tracking (`addLogCalls`, `sealBlockCall`) |
| `mockChainContainer` | `interop/interop_test.go` | Full ChainContainer mock with configurable responses |
| `statefulMockChainContainer` | `interop/interop_test.go` | Dynamic mock with function pointers |
| `interopTestHarness` | `interop/interop_test.go` | Builder pattern: `WithActivation()`, `WithChain()`, `Build()` |
| `testBlockInfo` | `interop/interop_test.go` | Implements `eth.BlockInfo` interface |
| `noopInvalidator` | `interop/logdb.go` (production code) | No-op `reads.Invalidator` for logsDB Rewind/Clear |
| `mockEngineController` | `chain_container/chain_container_test.go` | Engine mock with rewind tracking |
| `mockL2` | `chain_container/engine_controller/engine_controller_test.go` | Full L2 state simulation with FCU tracking |

### C. Patterns to Adapt from op-supervisor

| Pattern | Source | Adaptation Strategy |
|---------|--------|-------------------|
| Seed-based determinism | `chain_randomizer_test.go:94` `MakeRandomChain(seed)` | Same `int64` seed → `rand.New(rand.NewSource(seed))` pattern, new struct `SupernodeRandomState` |
| Multi-chain block generation | `chain_randomizer_test.go:126-200` | Generate blocks at **specific timestamps** (not just sequential block numbers) |
| Cross-chain dependency creation | `chain_randomizer_test.go:227+` | Generate `types.ExecutingMessage` structs with proper `ContainsQuery` data |
| 5 invalidation strategies | `chain_randomizer_test.go:410` `InvalidateBlock` | Adapt all 5 for op-supernode's `LogsDB.Contains()`-based verification (see below) |
| Fuzz test template | `cross_update_fuzz_test.go:1176+` | Same `f.Add(seed); f.Fuzz(func(t, seed) { ... })` pattern |
| State assertion after operations | `cross_update_fuzz_test.go:1162` `AssertInvariants` | New invariant set for op-supernode's sequential model |

### D. New Components to Create

#### 1. `SupernodeRandomState` (in `fuzz_helpers_test.go`)

```go
type SupernodeStateParams struct {
    ChainCount      int    // Number of chains (default: 3)
    TimestampCount  int    // Timestamps to process (default: 10-30)
    BlockTime       uint64 // Block time per chain (default: 2)
    MsgFrequency    int    // Percentage [0-100] of blocks with cross-chain msgs
    ActivationTS    uint64 // First timestamp to process
}

type SupernodeRandomState struct {
    rng              *rand.Rand
    chainIDs         []eth.ChainID
    activationTS     uint64
    blockTime        uint64
    timestamps       []uint64 // ordered list of timestamps to process

    // Per-chain block data: chainID → timestamp → block info
    blocks           map[eth.ChainID]map[uint64]*FuzzBlock
    // Cross-chain messages embedded in blocks
    execMessages     map[eth.ChainID]map[uint64][]*types.ExecutingMessage
}

type FuzzBlock struct {
    Ref        eth.L2BlockRef
    ParentHash common.Hash
    Receipts   gethTypes.Receipts
    Logs       []*gethTypes.Log
}
```

**Key difference from op-supervisor's `RandomChain`**: Blocks are organized by **timestamp** (op-supernode's primary key) rather than by block number with safety-level cutoffs (op-supervisor's model).

#### 2. `MakeRandomSupernodeState(seed int64, params SupernodeStateParams)`

Generation algorithm:
1. Create `chainCount` chain IDs
2. Generate `timestampCount` sequential timestamps starting from `activationTS` with step `blockTime`
3. For each chain at each timestamp, generate a block with proper parent hash linkage
4. With probability `msgFrequency/100`, add cross-chain `ExecutingMessage`s referencing other chains
5. Ensure all referenced initiating messages actually exist in source chain's blocks
6. Generate receipts with encoded executing message logs

#### 3. Invalidation Injectors (adapted from op-supervisor's 5 strategies)

| # | op-supervisor Strategy | op-supernode Adaptation |
|---|----------------------|------------------------|
| 1 | `InsertMessageWithInvalidIdentifier` | `InjectInvalidChecksum(state, chain, ts)` — corrupt `ExecutingMessage.Checksum` so `LogsDB.Contains()` returns `ErrConflict` |
| 2 | `InsertSelfDependency` | `InjectSelfReference(state, chain, ts)` — set `ExecutingMessage.ChainID` to the executing chain itself (valid per algo.go but may cause `Contains` to find the message in wrong context) |
| 3 | `InsertFutureDependency` | `InjectFutureTimestamp(state, chain, ts)` — set `ExecutingMessage.Timestamp >= executingTimestamp` to trigger `ErrTimestampViolation` |
| 4 | `InsertDependencyToExpiredMessage` | `InjectExpiredMessage(state, chain, ts)` — set `ExecutingMessage.Timestamp + ExpiryTime < executingTimestamp` to trigger `ErrMessageExpired` |
| 5 | `InsertCycle` | `InjectMissingMessage(state, chain, ts)` — reference a message that doesn't exist in source chain's logsDB (triggers `ErrConflict` from `Contains`) |

**Note on self-dependency**: Unlike op-supervisor which had explicit cycle detection, op-supernode's `verifyExecutingMessage` does NOT check for self-chain references. If a message on chain A references an initiating message also on chain A, it will pass if the timestamps and checksum are valid. This is a potential finding to verify.

#### 4. `FuzzLogsDB` Helper

Wraps real `logs.DB` in temp directory, pre-populates with block data from `SupernodeRandomState`:
```go
func NewFuzzLogsDB(t *testing.T, chainID eth.ChainID, state *SupernodeRandomState) LogsDB
```
Uses `logs.NewFromFile()` with temp directory, populates via `AddLog` + `SealBlock` calls.

#### 5. `FuzzVerifiedDB` Helper

Creates temp-dir bbolt DB with automatic cleanup:
```go
func NewFuzzVerifiedDB(t *testing.T) *VerifiedDB
```

#### 6. Receipt/Log Encoding Helpers

```go
func EncodeExecutingMessageLog(execMsg *types.ExecutingMessage) *gethTypes.Log
func GenerateReceiptsFromExecMsgs(execMsgs []*types.ExecutingMessage) gethTypes.Receipts
```
Uses `processors.DecodeExecutingMessageLog` in reverse to create valid encoded logs.

---

## Fuzzing Targets (Deep Analysis)

### Target 1: Interop Message Verification Algorithm (CRITICAL)
**File:** `op-supernode/supernode/activity/interop/algo.go`
**Functions:** `verifyInteropMessages`, `verifyExecutingMessage`

#### Code-Level Edge Cases Identified

**1a. `ErrSkipped` fallback path** (algo.go:56-79)
When `OpenBlock` returns `types.ErrSkipped`, the code falls back to `FirstSealedBlock()`. Three sub-paths:
- `FirstSealedBlock()` fails → wraps original error
- `firstBlock.Number == expectedBlock.Number` + hash mismatch → marks `InvalidHeads[chain]` AND `L2Heads[chain]`
- `firstBlock.Number != expectedBlock.Number` → returns hard error
- **Critical**: First block is assumed to have NO executing messages. If a real first block has executing messages, they are silently skipped.

**1b. Block hash mismatch behavior** (algo.go:83-92)
A hash mismatch marks both `InvalidHeads[chainID]` AND `L2Heads[chainID]`. Fuzz should verify this dual-marking is consistent.

**1c. Map iteration non-determinism** (algo.go:96)
`execMsgs` is `map[uint32]*types.ExecutingMessage` — iteration order is non-deterministic. The algorithm breaks on first invalid message (`break` at line 106). With multiple invalid messages in one block, different executions may flag different messages. Fuzz should test blocks with multiple invalid messages.

**1d. Missing chain silently skipped** (algo.go:47-52)
If `blocksAtTimestamp` includes a chain not in `i.logsDBs`, it's silently skipped. The resulting `Result` may not include all chains from input.

**1e. Expiry boundary exact values** (algo.go:131, 137)
- `execMsg.Timestamp >= executingTimestamp` → exactly equal is INVALID (`ErrTimestampViolation`)
- `execMsg.Timestamp + ExpiryTime < executingTimestamp` → at boundary `==` is VALID
- **uint64 overflow**: `execMsg.Timestamp + ExpiryTime` could overflow if `execMsg.Timestamp` is near `math.MaxUint64`

**1f. L1Head never set** (algo.go:40-44)
The `Result` returned by `verifyInteropMessages` never sets `L1Head`. When `progressAndRecord` at interop.go:219 uses `result.L1Head`, it gets zero `BlockID`. This propagates to `VerifiedDB.Commit()`. Potential finding to verify.

#### Properties to Verify
- P1: Valid cross-chain messages never produce `InvalidHeads`
- P2: Every invalidation type is correctly detected
- P3: `Result.IsValid()` ↔ `len(InvalidHeads) == 0`
- P4: `execMsg.Timestamp + ExpiryTime` overflow doesn't cause false positive/negative
- P5: First block (ErrSkipped path) correctly handles hash mismatch
- P6: Block with multiple invalid messages still gets marked invalid (regardless of iteration order)
- P7: Missing chains in logsDBs are consistently excluded from Result

#### Fuzz Functions
- `FuzzVerifyInteropMessagesValid` — valid states always pass (P1, P3)
- `FuzzVerifyInteropMessagesFails` — each invalidation type detected (P2)
- `FuzzVerifyExpiryBoundary` — timestamps at exact expiry boundary `ExpiryTime ± 1` (P4)
- `FuzzVerifyFirstBlockSkipped` — ErrSkipped path with valid/invalid first blocks (P5)
- `FuzzVerifyMultipleInvalidMessages` — blocks with multiple invalid messages (P6)

---

### Target 2: Log Database Continuity & Loading (HIGH)
**File:** `op-supernode/supernode/activity/interop/logdb.go`
**Functions:** `loadLogs`, `verifyCanAddTimestamp`, `processBlockLogs`

#### Code-Level Edge Cases Identified

**2a. Block skip silently passes** (logdb.go:121-125)
When `latestBlock.Number >= block.Number`, loading is skipped. But this doesn't verify hash matching! If the logsDB has block 5 with hash A but chain provides block 5 with hash B, it silently accepts.

**2b. Gap calculation edge** (logdb.go:179-185)
`gap := ts - seal.Timestamp` — safe because line 175 returns early if `seal.Timestamp > ts`. But `gap < blockTime` only warns, doesn't error. Non-block-time-aligned timestamps can be processed.

**2c. First block parent handling** (logdb.go:213)
`blockNum == 0 || isFirstBlock` — the `||` means block 0 ALWAYS gets empty parent, even if it's not the first block in the DB. Edge case: what if block 0 is loaded again after the DB already has data?

**2d. Silent error in DecodeExecutingMessageLog** (logdb.go:224)
`execMsg, _ := processors.DecodeExecutingMessageLog(l)` — errors are silently ignored. A malformed log could result in `nil` execMsg (which is valid — means not an executing message) but could also mask encoding bugs.

**2e. Activation timestamp special case** (logdb.go:157-163)
If DB is empty but timestamp != activationTimestamp → `ErrPreviousTimestampNotSealed`. This enforces that the first timestamp processed must be exactly the activation timestamp.

#### Properties to Verify
- P8: Sequential timestamps always succeed when chain data is available
- P9: Gap violations are always detected (gap > blockTime)
- P10: Parent hash mismatches are detected for non-first blocks
- P11: First block with empty parent hash is accepted exactly once
- P12: After any error, the DB remains consistent (no partial writes)
- P13: Non-block-time-aligned gaps only warn, don't error
- P14: Block skip when `latestBlock.Number >= block.Number` doesn't corrupt state

#### Fuzz Functions
- `FuzzLoadLogsSequential` — valid sequential loading always succeeds (P8)
- `FuzzLoadLogsWithGaps` — missing timestamps are detected (P9)
- `FuzzVerifyCanAddTimestamp` — boundary conditions in gap calculation (P9, P13)
- `FuzzProcessBlockLogs` — arbitrary receipts with varying log counts and exec messages (P12)

---

### Target 3: VerifiedDB Sequential Enforcement & Rewind (HIGH)
**File:** `op-supernode/supernode/activity/interop/verified_db.go`
**Functions:** `Commit`, `Rewind`, `Get`, `Has`, `LastTimestamp`

#### Code-Level Edge Cases
- Big-endian uint64 key encoding — must be lexicographically correct
- First commit vs subsequent commits
- Rewind to timestamp 0 (delete everything)
- Rewind to timestamp beyond last committed (no-op?)
- JSON serialization/deserialization round-trip of `VerifiedResult`
- Interleaved commit/rewind patterns

#### Properties to Verify
- P15: `Commit(result)` succeeds iff `result.Timestamp == lastTimestamp + 1` (or first commit at activationTS)
- P16: After `Rewind(ts)`, `LastTimestamp()` returns `ts - 1`
- P17: After `Rewind(ts)`, `Get(t)` errors for all `t >= ts`
- P18: After `Rewind(ts)`, `Commit(ts)` succeeds (re-commit from rewind point)
- P19: `ErrAlreadyCommitted` and `ErrNonSequential` are correctly distinguished
- P20: JSON round-trip preserves all VerifiedResult fields

#### Fuzz Functions
- `FuzzVerifiedDBCommitRewind` — random sequences of commit/rewind operations (P15-P20)

---

### Target 4: DenyList / Block Invalidation (MEDIUM)
**File:** `op-supernode/supernode/chain_container/invalidation.go`

#### Properties to Verify
- P21: `Contains(h, hash)` returns true iff `Add(h, hash)` was previously called
- P22: `Add` is idempotent
- P23: Hashes at different heights are isolated
- P24: Concatenated 32-byte hash storage handles boundary alignment correctly

#### Fuzz Functions
- `FuzzDenyListAddContains` — random add/contains sequences (P21-P24)
- `FuzzDenyListConcurrent` — parallel operations for thread safety

---

### Target 5: Engine Rewind Algorithm (MEDIUM)
**File:** `op-supernode/supernode/chain_container/engine_controller/rewind.go`

Note: 13+ existing test cases cover error taxonomy. Fuzzing adds coverage for random state combinations.

#### Properties to Verify
- P25: Rewind never succeeds when target is before finalized head
- P26: After successful rewind, unsafe head == target block
- P27: After successful rewind, finalized head is unchanged

#### Fuzz Functions
- `FuzzRewindToTimestamp` — random engine states and rewind targets (P25-P27)

---

### Target 6: End-to-End Interop Progress Loop (HIGH)
**File:** `op-supernode/supernode/activity/interop/interop.go`
**Functions:** `progressInterop`, `handleResult`, `checkChainsReady`, `Reset`

#### Code-Level Edge Cases Identified

**6a. Reset race window** (interop.go:401-426)
`Reset` acquires `mu.Lock()` but `progressAndRecord()` doesn't hold the lock during `progressInterop()`. A Reset could occur between `loadLogs` and `verifyFn`, corrupting the logsDB state mid-verification.

**6b. resetLogsDB clear-vs-rewind boundary** (interop.go:449)
`firstBlock.Number > targetBlock.Number` → clear. `firstBlock.Number <= targetBlock.Number` → rewind. Edge case: when `firstBlock.Number == targetBlock.Number`, it rewinds to the first block (which may clear most data anyway).

**6c. handleResult empty-vs-valid-vs-invalid** (interop.go:299-325)
Empty results → no-op. Invalid results → invalidate blocks, return without L1 update (line 204-207). Valid results → commit. Fuzz should test all three paths and transitions.

**6d. L1Head tracking** (interop.go:215-224)
`verifiedAdvanced` is `!result.IsEmpty()`. When true, `currentL1 = result.L1Head`. But `verifyInteropMessages` never sets L1Head (it's zero). This means successful verification sets `currentL1` to zero `BlockID`. **Potential finding**.

**6e. checkChainsReady goroutine leaks** (interop.go:361-368)
If one chain errors, the function returns immediately. Other goroutines may still be writing to the buffered channel. The channel is sized `len(i.chains)` so no goroutine blocks, but results are discarded.

#### Properties to Verify
- P28: Timestamps are processed strictly sequentially (no gaps, no repeats)
- P29: Valid results are committed; invalid results trigger block invalidation
- P30: Chain not-ready (ethereum.NotFound) causes retry without advancing
- P31: After invalidation, the interop loop can resume from the same timestamp
- P32: `Reset` correctly rewinds both logsDB and verifiedDB
- P33: currentL1 is correctly maintained through valid/invalid/empty result flows

#### Fuzz Functions
- `FuzzProgressInteropValid` — valid multi-chain states always commit (P28, P29)
- `FuzzProgressInteropInvalid` — invalid messages trigger correct invalidation (P29, P31)
- `FuzzProgressInteropReset` — reset at various points doesn't corrupt state (P32)

---

### Target 7: Interop Type Properties (LOW)
**File:** `op-supernode/supernode/activity/interop/types.go`

#### Properties
- P34: `Result.IsValid()` == `(len(InvalidHeads) == 0)`
- P35: `ToVerifiedResult()` strips invalid heads, preserves other fields
- P36: Empty results correctly detected

#### Fuzz Functions
- `FuzzResultProperties` — random Result construction (P34-P36)

---

## Implementation Plan

### Files to Create

1. **`op-supernode/supernode/activity/interop/fuzz_helpers_test.go`** (~400-500 lines)
   - `SupernodeStateParams` struct and `SupernodeRandomState` struct
   - `MakeRandomSupernodeState(seed, params)` — full state generation
   - `NewFuzzLogsDB(t, chainID, state)` — creates real `logs.DB` in temp dir, pre-populated
   - `NewFuzzVerifiedDB(t)` — creates temp bbolt DB with cleanup
   - 5 invalidation injector functions
   - Receipt/log encoding helpers

2. **`op-supernode/supernode/activity/interop/fuzz_algo_test.go`** (~300-400 lines)
   - `FuzzVerifyInteropMessagesValid`
   - `FuzzVerifyInteropMessagesFails`
   - `FuzzVerifyExpiryBoundary`
   - `FuzzVerifyFirstBlockSkipped`
   - `FuzzVerifyMultipleInvalidMessages`

3. **`op-supernode/supernode/activity/interop/fuzz_verified_db_test.go`** (~150-200 lines)
   - `FuzzVerifiedDBCommitRewind`

4. **`op-supernode/supernode/activity/interop/fuzz_logdb_test.go`** (~200-300 lines)
   - `FuzzLoadLogsSequential`
   - `FuzzLoadLogsWithGaps`
   - `FuzzVerifyCanAddTimestamp`
   - `FuzzProcessBlockLogs`

5. **`op-supernode/supernode/chain_container/fuzz_invalidation_test.go`** (~150-200 lines)
   - `FuzzDenyListAddContains`
   - `FuzzDenyListConcurrent`

6. **`op-supernode/supernode/chain_container/engine_controller/fuzz_rewind_test.go`** (~150-200 lines)
   - `FuzzRewindToTimestamp`

7. **`op-supernode/supernode/activity/interop/fuzz_interop_test.go`** (~300-400 lines)
   - `FuzzProgressInteropValid`
   - `FuzzProgressInteropInvalid`
   - `FuzzProgressInteropReset`

### Implementation Order

1. **Phase 1: Infrastructure** — `fuzz_helpers_test.go` (SupernodeRandomState, mock helpers, invalidation injectors)
2. **Phase 2: Core Algorithm** — `fuzz_algo_test.go` (Target 1, highest value, most isolated)
3. **Phase 3: VerifiedDB** — `fuzz_verified_db_test.go` (Target 3, simple stateful testing)
4. **Phase 4: LogsDB** — `fuzz_logdb_test.go` (Target 2, builds on DB patterns)
5. **Phase 5: DenyList** — `fuzz_invalidation_test.go` (Target 4, medium complexity)
6. **Phase 6: Engine Rewind** — `fuzz_rewind_test.go` (Target 5, requires mock engine)
7. **Phase 7: E2E Interop** — `fuzz_interop_test.go` (Target 6, integration)
8. **Phase 8: Types** — Add `FuzzResultProperties` to `fuzz_algo_test.go` (Target 7, quick)

### Verification

```bash
# Quick smoke test (10 seconds per target)
cd op-supernode && go test -fuzz=FuzzVerifyInteropMessagesValid -fuzztime=10s ./supernode/activity/interop/

# Extended campaign (5 minutes per target)
cd op-supernode && go test -fuzz=FuzzVerifyInteropMessagesValid -fuzztime=5m ./supernode/activity/interop/

# Run all unit tests to ensure no regressions
cd op-supernode && go test ./...
```

---

## Summary

| # | Target | File | Fuzz Functions | Properties | Priority |
|---|--------|------|---------------|------------|----------|
| 1 | Interop Algo | `fuzz_algo_test.go` | 5 functions | P1-P7 | CRITICAL |
| 2 | LogsDB | `fuzz_logdb_test.go` | 4 functions | P8-P14 | HIGH |
| 3 | VerifiedDB | `fuzz_verified_db_test.go` | 1 function | P15-P20 | HIGH |
| 4 | DenyList | `fuzz_invalidation_test.go` | 2 functions | P21-P24 | MEDIUM |
| 5 | Engine Rewind | `fuzz_rewind_test.go` | 1 function | P25-P27 | MEDIUM |
| 6 | E2E Interop | `fuzz_interop_test.go` | 3 functions | P28-P33 | HIGH |
| 7 | Types | `fuzz_algo_test.go` | 1 function | P34-P36 | LOW |

**Total: 7 targets, 17 fuzz functions, 36 properties**

### Potential Findings (from code analysis)
1. **L1Head never set in verifyInteropMessages** — `Result.L1Head` is zero, propagates to `VerifiedDB.Commit()` and `currentL1` tracking
2. **Self-chain references not checked** — `verifyExecutingMessage` doesn't reject messages referencing the executing chain itself
3. **Block skip doesn't verify hash** — `loadLogs` skips loading when `latestBlock.Number >= block.Number` without checking hash match
4. **Silent error in DecodeExecutingMessageLog** — malformed logs result in nil execMsg, silently treated as non-executing
5. **uint64 overflow in expiry check** — `execMsg.Timestamp + ExpiryTime` could overflow near `math.MaxUint64`
