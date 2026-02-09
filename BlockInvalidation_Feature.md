# BlockInvalidation Feature Diary

## Feature Overview
**Feature Name:** Block Invalidation & Replacement
**Branch:** `supernode/BlockInvalidation`
**Developer:** Axel Kingsley
**Started:** 2026-02-06

### Purpose
Implement a block invalidation mechanism in the op-supernode that:
1. Persists invalid payload hashes in a DenyList (keyed by block height)
2. Triggers chain rewinds when the current chain uses an invalidated block
3. Notifies activities (especially Interop) to clean up cached state on reset
4. Integrates with op-node to deny payloads before insertion and trigger deposits-only replacement

---

## Diary of Interactions

### Session 1 (Retroactive) — Prior to Skill Adoption

**Context:** Development began before the op-feature skill was adopted. The following commits were created through iterative prompts.

### Session 2 — Skill Adopted

**Prompt:** Developer provided op-feature skill prompt and asked me to adopt it.

**Action:** Created this diary file. Developer granted standing permission to update diary without approval.

### Session 2.1 — Sub-Feature Review for PR

**Prompt:** Developer requested detailed breakdown of Sub-Feature 1 for peer review.

**Action:** Generated comprehensive report (see Sub-Feature 1 section below).

---

## Current State

### Commits on Branch (9 total)
```
31ea9484c6 op-acceptance-tests: add replacement block assertions
90db9cf396 op-supernode: implement ResetOn in Interop activity
dbbbc6568c op-supernode: add SetResetCallback to ChainContainer
c920528378 op-supernode: add ResetOn method to Activity interface
539e46aef0 op-node: add SuperAuthority interface for payload denial
227537f99a Fix block invalidation: use eth.Unsafe label and improve test resilience
9a63b131e9 Wire up block invalidation from interop activity to chain container
ae4358d202 op-supernode: Add unit tests for block invalidation
425bbb2d9a op-supernode: Add block invalidation and deny list to chain container
```

### Test Coverage Summary

| Component | Test File | Status |
|-----------|-----------|--------|
| DenyList | `invalidation_test.go` | ✅ Implemented |
| InvalidateBlock | `invalidation_test.go` | ✅ Implemented |
| IsDenied | `invalidation_test.go` | ✅ Implemented |
| VerifiedDB.RewindTo | `verified_db_test.go` | ❌ Not yet tested |
| Interop.ResetOn | `interop_test.go` | ❌ Not yet tested |
| SuperAuthority denial | - | ❌ Not yet tested |
| Acceptance (Halt) | `invalid_message_halt_test.go` | ✅ Exists |
| Acceptance (Replace) | `invalid_message_replacement_test.go` | ✅ Exists |

---

# Sub-Feature Breakdowns

## Sub-Feature 1: DenyList & InvalidateBlock

### Commits

| SHA | Message | Files Changed |
|-----|---------|---------------|
| `425bbb2d9a` | op-supernode: Add block invalidation and deny list to chain container | 3 files, +471 |
| `ae4358d202` | op-supernode: Add unit tests for block invalidation | 6 files, +538 |

### Purpose

Provide a persistent mechanism to track invalid block payload hashes and trigger chain rewinds when the current chain uses an invalidated block.

**Why this exists:** When the Interop activity detects an invalid cross-chain executing message, it needs to:
1. Remember that block is invalid (so it's never re-applied)
2. Trigger a rewind if the chain is currently using that block

### Specification

#### DenyList
A bbolt-backed key-value store that persists invalid payload hashes keyed by block height.

| Method | Signature | Behavior |
|--------|-----------|----------|
| `OpenDenyList` | `(dataDir string) (*DenyList, error)` | Opens/creates DB, ensures bucket exists, creates parent dirs |
| `Add` | `(height uint64, payloadHash Hash) error` | Appends hash to height's entry. Idempotent (no duplicates) |
| `Contains` | `(height uint64, payloadHash Hash) (bool, error)` | Returns true if hash exists at height |
| `GetDeniedHashes` | `(height uint64) ([]Hash, error)` | Returns all hashes at height |
| `Close` | `() error` | Closes bbolt DB |

**Storage format:**
- Key: `uint64` height as 8-byte big-endian
- Value: Concatenated 32-byte hashes

#### InvalidateBlock
Added to `ChainContainer` interface.

```go
InvalidateBlock(ctx context.Context, height uint64, payloadHash common.Hash) (bool, error)
```

**Behavior:**
1. Add hash to DenyList
2. If engine available, check if current block at `height` matches `payloadHash`
3. If match → call `RewindEngine(ctx, priorTimestamp)` → return `true`
4. If no match → return `false` (no rewind needed)

#### IsDenied
Helper method on `ChainContainer`:
```go
IsDenied(height uint64, payloadHash common.Hash) (bool, error)
```
Delegates to `denyList.Contains`.

### Test Coverage

#### `TestDenyList_AddAndContains` — 4 subcases

| Subcase | Setup | Assertion |
|---------|-------|-----------|
| `single hash at height` | Add 1 hash at height 100 | `Contains(100, hash)` → `true` |
| `multiple hashes same height` | Add 3 hashes at height 50 | All 3 return `true` from `Contains` |
| `hash at wrong height returns false` | Add hash at height 10 | `Contains(11, hash)` → `false`, `Contains(10, hash)` → `true` |
| `duplicate add is idempotent` | Add same hash 3 times | `GetDeniedHashes` returns exactly 1 entry |

#### `TestDenyList_Persistence` — 2 subcases

| Subcase | Setup | Assertion |
|---------|-------|-----------|
| `survives close and reopen` | Add 4 hashes, close DB | Reopen → all 4 hashes present, correct counts |
| `empty DB on fresh open` | (none) | `Contains` → `false`, `GetDeniedHashes` → empty |

#### `TestDenyList_GetDeniedHashes` — 3 subcases

| Subcase | Setup | Assertion |
|---------|-------|-----------|
| `returns all hashes at height` | Add 5 hashes at height 100 | `GetDeniedHashes(100)` returns 5 |
| `empty for clean height` | Add at heights 10, 30 | `GetDeniedHashes(20)` → empty |
| `isolated by height` | Add 2 at h10, 3 at h20, 1 at h30 | Correct counts at each height |

#### `TestInvalidateBlock` — 4 subcases

| Subcase | Config | Assertion |
|---------|--------|-----------|
| `current block matches triggers rewind` | currentHash == payloadHash | `rewound=true`, `RewindToTimestamp` called with correct ts |
| `current block differs no rewind` | currentHash ≠ payloadHash | `rewound=false`, no rewind call |
| `engine unavailable adds to denylist only` | engine=nil | `rewound=false`, hash still in denylist |
| `rewind to height-1 timestamp calculated correctly` | height=10 | Rewind ts = `genesis + (9 * blockTime)` |

#### `TestIsDenied` — 3 subcases

| Subcase | Setup | Assertion |
|---------|-------|-----------|
| `denied block returns true` | Add hash at height 100 | `IsDenied(100, hash)` → `true` |
| `non-denied returns false` | Add hash at height 100 | `IsDenied(100, differentHash)` → `false` |
| `wrong height returns false` | Add hash at height 10 | `IsDenied(11, sameHash)` → `false` |

#### `TestDenyList_ConcurrentAccess` — 1 case

| Subcase | Setup | Assertion |
|---------|-------|-----------|
| 10 concurrent accessors | 10 goroutines, 100 ops each | All writes succeed, all reads find own hashes, no errors |

### Untested Behavior

| Behavior | Why Untested |
|----------|--------------|
| Corrupt bbolt file recovery | Out of scope |
| Very large number of hashes at single height | Performance not tested |

### Specified Behavior (Clarified)

| Behavior | Status |
|----------|--------|
| `height=0` | ✅ Works fine, acceptable |
| `RewindEngine` fails | ✅ Hash remains in denylist — **intended behavior** |
| Concurrent access | ✅ Now tested (see `TestDenyList_ConcurrentAccess`)

### Code Locations

| Component | File | Lines |
|-----------|------|-------|
| DenyList struct | `chain_container/invalidation.go` | 23-28 |
| OpenDenyList | `chain_container/invalidation.go` | 30-52 |
| Add | `chain_container/invalidation.go` | 62-91 |
| Contains | `chain_container/invalidation.go` | 94-119 |
| GetDeniedHashes | `chain_container/invalidation.go` | 122-143 |
| InvalidateBlock | `chain_container/invalidation.go` | 149-207 |
| IsDenied | `chain_container/invalidation.go` | 218-223 |
| Tests | `chain_container/invalidation_test.go` | 1-514 |

---

## Sub-Feature 2: Wire Interop → ChainContainer

### Commits

| SHA | Message | Files Changed |
|-----|---------|---------------|
| `9a63b131e9` | Wire up block invalidation from interop activity | 2 files |
| `227537f99a` | Fix block invalidation: use eth.Unsafe label | 2 files |

### Purpose

Connect the Interop activity's invalid message detection to the ChainContainer's invalidation mechanism.

**Flow:**
1. Interop detects invalid executing message in `verifyInteropMessages`
2. `handleResult` sees `InvalidHeads` in the result
3. Calls `invalidateBlock(chainID, blockID)` for each invalid head
4. `invalidateBlock` calls `chain.InvalidateBlock(ctx, blockNum, hash)`

### Specification

```go
func (i *Interop) invalidateBlock(chainID eth.ChainID, blockID eth.BlockID) error
```

| Scenario | Behavior |
|----------|----------|
| Chain not found | Return error `"chain %s not found"` |
| Chain.InvalidateBlock errors | Log error, return error |
| Chain.InvalidateBlock returns `rewound=true` | Log warn "chain rewound" |
| Chain.InvalidateBlock returns `rewound=false` | Log info "block added to denylist" |

### Test Coverage

#### `TestInvalidateBlock` (interop_test.go) — 4 subcases

| Subcase | Setup | Assertion |
|---------|-------|-----------|
| `calls chain.InvalidateBlock with correct args` | Call invalidateBlock(chainID, blockID) | mock tracks height=500, hash=0xBAD |
| `returns error when chain not found` | Call with unknown chainID | Error contains "not found", no mock calls |
| `returns error when chain.InvalidateBlock fails` | mock returns error | Error propagated |
| `handleResult calls invalidateBlock for each invalid head` | Result with 2 InvalidHeads | Both mocks have 1 call each with correct args |

### Code Locations

| Component | File | Lines |
|-----------|------|-------|
| invalidateBlock | `activity/interop/interop.go` | 293-322 |
| Tests | `activity/interop/interop_test.go` | TestInvalidateBlock |

## Sub-Feature 3: SuperAuthority Injection

### Commits

| SHA | Message | Files Changed |
|-----|---------|---------------|
| `539e46aef0` | op-node: add SuperAuthority interface for payload denial | 8 files |

### Purpose

Allow the `op-supernode` to inject a "SuperAuthority" into the `op-node` engine controller. This authority can deny payloads before they are inserted, triggering deposits-only replacement during Holocene derivation.

**Flow:**
1. `ChainContainer` implements `SuperAuthority.IsDenied(blockNumber, hash)`
2. Passed via `InitializationOverrides` when creating `VirtualNode`
3. `EngineController` calls `IsDenied` before `NewPayload`
4. If denied + Holocene + derived → request deposits-only replacement
5. If denied otherwise → emit `PayloadInvalidEvent`

### Specification

```go
type SuperAuthority interface {
    IsDenied(blockNumber uint64, payloadHash common.Hash) (bool, error)
}
```

| Scenario | Behavior |
|----------|----------|
| IsDenied returns `(true, nil)` | Payload rejected, replacement requested (Holocene) or invalid event |
| IsDenied returns `(false, nil)` | Payload proceeds to engine |
| IsDenied returns `(_, error)` | Log warning, proceed with payload (graceful degradation) |
| SuperAuthority is nil | No check, proceed with payload |

### Test Coverage

#### `TestSuperAuthority_*` (engine_controller_test.go) — 4 tests

| Test | Setup | Assertion |
|------|-------|-----------|
| `DeniedPayload_EmitsInvalidEvent` | sa.DenyBlock for payload | PayloadInvalidEvent emitted, NewPayload NOT called |
| `AllowedPayload_Proceeds` | sa empty (no deny) | NewPayload called, PayloadSuccessEvent emitted |
| `Error_ProceedsWithPayload` | sa.shouldError = true | NewPayload called despite error, PayloadSuccessEvent emitted |
| `NilAuthority_Proceeds` | sa = nil | NewPayload called, PayloadSuccessEvent emitted |

### Code Locations

| Component | File | Lines |
|-----------|------|-------|
| SuperAuthority interface | `op-node/rollup/engine/engine_controller.go` | 96-104 |
| IsDenied check | `op-node/rollup/engine/payload_process.go` | 31-58 |
| InitializationOverrides | `op-node/node/node.go` | InitializationOverrides struct |
| Tests | `op-node/rollup/engine/engine_controller_test.go` | TestSuperAuthority_* |

## Sub-Feature 4: Activity Reset Notification Chain

### Commits

| SHA | Message | Files Changed |
|-----|---------|---------------|
| `c920528378` | op-supernode: add ResetOn method to Activity interface | 6 files |
| `dbbbc6568c` | op-supernode: add SetResetCallback to ChainContainer | 3 files |
| `90db9cf396` | op-supernode: implement ResetOn in Interop activity | 6 files |

### Purpose

When a chain container rewinds due to block invalidation, all activities must be notified so they can clean up cached state. The Interop activity specifically must rewind its `logsDB` and `verifiedDB`.

**Flow:**
1. `ChainContainer.InvalidateBlock` triggers a rewind
2. After successful rewind, calls `onReset(chainID, timestamp)` callback
3. `Supernode.onChainReset` receives the notification
4. Iterates through all activities, calling `ResetOn(chainID, timestamp)`
5. Interop: rewinds logsDB and verifiedDB
6. Heartbeat/Superroot: no-op (no cached state)

### Specification

```go
// Activity interface
ResetOn(chainID eth.ChainID, timestamp uint64)

// ChainContainer interface
SetResetCallback(cb ResetCallback)

// VerifiedDB
RewindTo(timestamp uint64) (deleted bool, err error)
```

| Scenario | Behavior |
|----------|----------|
| Previous block available | logsDB.Rewind(prevBlockID) |
| Previous block not found | logsDB.Clear() |
| timestamp ≤ blockTime | logsDB.Clear() |
| Verified results deleted | Log ERROR (unexpected) |

### Test Coverage

#### `TestVerifiedDB_RewindTo` (verified_db_test.go) — 4 subcases

| Subcase | Setup | Assertion |
|---------|-------|-----------|
| `removes entries at and after timestamp` | Commit 100-105, RewindTo(103) | 100-102 exist, 103-105 gone, lastTs=102 |
| `returns false when no entries deleted` | Commit 98-100, RewindTo(200) | All exist, deleted=false |
| `rewind all entries` | Commit 100-102, RewindTo(0) | All gone, lastTs uninitialized |
| `allows sequential commits after rewind` | Commit 100-105, RewindTo(103), Commit 103 | New 103 data readable |

#### `TestResetOn` (interop_test.go) — 6 subcases

| Subcase | Setup | Assertion |
|---------|-------|-----------|
| `rewinds logsDB when previous block available` | mock returns valid block | logsDB.Rewind called with prevBlockID |
| `clears logsDB when previous block not available` | mock returns error | logsDB.Clear called |
| `clears logsDB when timestamp at or before blockTime` | timestamp=1, blockTime=1 | logsDB.Clear called |
| `rewinds verifiedDB` | Commit 98-102, ResetOn(100) | 98-99 exist, 100-102 gone |
| `resets currentL1` | currentL1={500, 0xL1} | currentL1 = {} after reset |
| `handles unknown chain gracefully` | ResetOn(unknownChain, 100) | No panic |

### Code Locations

| Component | File | Lines |
|-----------|------|-------|
| ResetOn (Activity interface) | `activity/activity.go` | 4-8 |
| SetResetCallback | `chain_container/chain_container.go` | SetResetCallback method |
| onReset callback | `chain_container/invalidation.go` | In InvalidateBlock |
| Supernode.onChainReset | `supernode/supernode.go` | onChainReset method |
| Interop.ResetOn | `activity/interop/interop.go` | 388-480 |
| VerifiedDB.RewindTo | `activity/interop/verified_db.go` | 184-220 |
| Tests | verified_db_test.go, interop_test.go | TestVerifiedDB_RewindTo, TestResetOn |

## Sub-Feature 5: Acceptance Tests

### Commits

| SHA | Message | Files Changed |
|-----|---------|---------------|
| `31ea9484c6` | op-acceptance-tests: add replacement block assertions | 1 file |

### Purpose

End-to-end test verifying the complete block invalidation and replacement flow works in a full supernode environment.

**Test Flow:**
1. Start supernode with interop chains
2. Send cross-chain message that will be invalid
3. Wait for message to be included in a block
4. Interop activity detects invalid message
5. ChainContainer invalidates block, triggers rewind
6. Activities are notified via ResetOn
7. op-node derives replacement block (deposits-only)
8. New block at same height has different hash
9. Invalid transaction is NOT in replacement block
10. Timestamp eventually becomes verified

### Test: `TestSupernodeInteropInvalidMessageReplacement`

**Location:** `op-acceptance-tests/tests/supernode/interop/invalid_message_replacement_test.go`

### Phases

| Phase | What is verified |
|-------|------------------|
| 1. Setup | Supernode running, chains synced |
| 2. Send invalid message | Cross-chain exec message sent, receipt obtained |
| 3. Observe reset | Block at invalid height changes or disappears |
| 4. Detect replacement | New block at same height with different hash |
| 5. Verify replacement | Replacement hash ≠ invalid hash, invalid tx not in replacement |
| 6. Verify timestamp | `SuperRootAtTimestamp` returns verified data |

### Assertions

| Assertion | Purpose |
|-----------|---------|
| `resetDetected = true` | Rewind was triggered |
| `replacementDetected = true` | New block created at same height |
| `replacementHash ≠ invalidHash` | Block was actually replaced |
| `invalidTx NOT in replacement` | Invalid transaction removed |
| `verified = true` | Replacement passes verification |

### Code Location

| Component | File |
|-----------|------|
| Test | `op-acceptance-tests/tests/supernode/interop/invalid_message_replacement_test.go` |

---

## Test Summary

All unit tests added for missing coverage:

| Test File | Tests Added |
|-----------|-------------|
| `chain_container/invalidation_test.go` | `TestDenyList_ConcurrentAccess` |
| `activity/interop/interop_test.go` | `TestInvalidateBlock` (4 cases), `TestResetOn` (6 cases) |
| `activity/interop/verified_db_test.go` | `TestVerifiedDB_RewindTo` (4 cases) |
| `op-node/rollup/engine/engine_controller_test.go` | `TestSuperAuthority_*` (4 tests) |

## Next Steps

Ready to commit all tests as "fill in missing unit tests" commit.

---

## Sub-Feature 6: Test Control for Interop Activity

### Purpose
Provide integration test control for pausing and resuming the interop activity at specific timestamps. This allows acceptance tests to precisely control when interop validation occurs.

### Specification
- `PauseInterop(ts uint64)`: When called, the interop activity pauses at the given timestamp - if it would process that timestamp in its progress loop, it returns early without making progress.
- `ResumeInterop()`: Clears the pause, allowing normal processing to continue.
- Zero value for `ts` indicates "not paused" (always process all values).
- Values are stored atomically for concurrent read/write safety.
- This is test-only functionality, not wired at production level.

### Implementation

| Component | Location | Changes |
|-----------|----------|---------|
| Interop Activity | `op-supernode/supernode/activity/interop/interop.go` | Added `pauseAtTimestamp atomic.Uint64` field, `PauseAt(ts)` and `Resume()` methods, check in `progressInterop()` |
| Supernode | `op-supernode/supernode/supernode.go` | Added `PauseInterop(ts)` and `ResumeInterop()` methods that delegate to interop activity |
| sysgo.SuperNode | `op-devstack/sysgo/l2_cl_supernode.go` | Added `PauseInterop(ts)` and `ResumeInterop()` methods |
| Stack Interface | `op-devstack/stack/supernode.go` | Added `InteropTestControl` interface |
| Orchestrator | `op-devstack/sysgo/orchestrator.go` | Added `InteropTestControl(id)` method to get test control for a supernode |
| DSL Supernode | `op-devstack/dsl/supernode.go` | Added `testControl` field, `NewSupernodeWithTestControl()` constructor, `PauseInterop(ts)` and `ResumeInterop()` methods |
| Preset | `op-devstack/presets/twol2.go` | Wire up `InteropTestControl` in `NewTwoL2SupernodeInterop()` |

### Usage in Tests

```go
// Pause interop at a specific timestamp
sys.Supernode.PauseInterop(targetTimestamp + 1)

// ... perform test actions ...

// Resume interop processing
sys.Supernode.ResumeInterop()
```

### Test Coverage
Test-only functionality - exercised through acceptance test usage.
