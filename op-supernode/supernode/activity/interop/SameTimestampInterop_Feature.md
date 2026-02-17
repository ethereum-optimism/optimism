# Same-Timestamp Interop Feature

**Author**: AI Tool
**Branch**: `supernode/SameTimestampinterop` (based on `tests/SameTimestampInterop`)
**Started**: 2026-02-17

---

## Description

This feature enables **same-timestamp interop verification** for the supernode's interop activity. Currently, the interop verification logic (`algo.go`) rejects any executing message whose initiating message has a timestamp `>=` the executing message's timestamp. This feature relaxes that constraint to allow **same-timestamp** messages (`==`), while future timestamps (`>`) remain invalid.

Same-timestamp messages introduce **circular dependency** scenarios: Chain A block at T=1000 may contain an executing message referencing an initiating message on Chain B also at T=1000, and vice versa. These require special verification logic that can resolve cycles.

### Goals
1. Allow same-timestamp messages (change `>=` to `>` in timestamp check)
2. Introduce `cycleVerifyFn` as a pluggable function alongside `verifyFn` for same-timestamp verification
3. Route same-timestamp messages through `cycleVerifyFn`, propagating invalid heads from cycle verification into the overall result
4. Implement cycle verification algorithm in `circular.go`

---

## Breakdown: Commit-Based Sub-Features

| # | Subfeature | Key Change | Unit Tests | Acceptance Tests |
|---|------------|------------|------------|------------------|
| 1 | Allow same-timestamp messages | `>= → >` in timestamp check | ✅ New tests for same-ts allowed | ✅ **AMENDED**: Update to expect same-ts passes (no block replacement) |
| 2 | Add `cycleVerifyFn` field | New field on `Interop` struct | ✅ Mock tests | ✅ No change needed yet |
| 3 | Route same-timestamp through `cycleVerifyFn` | Detection + delegation logic | ✅ Routing tests | ✅ Tests show cycle path being taken |
| 4 | Implement `circular.go` | Algorithm implementation | ✅ Algorithm tests | ✅ Tests verify cycle resolution behavior |

### Acceptance Test Evolution Strategy

We amend `op-acceptance-tests/tests/supernode/interop/same_timestamp_invalid/` alongside each subfeature to show behavior changes:

| Subfeature | `TestSupernodeSameTimestampExecMessage` | `TestSupernodeSameTimestampInvalidTransitive` |
|------------|----------------------------------------|-----------------------------------------------|
| Before | Expects block replacement (same-ts = invalid) | Expects transitive invalidation |
| SF1 | **Update**: Expects NO replacement (same-ts = valid) | **Update**: Only invalid log index causes replacement |
| SF2 | No change | No change |
| SF3 | No change | No change |
| SF4 | Verify cycle resolution works | Verify cycle-aware transitive behavior |

---

### Subfeature 1: Allow Same-Timestamp Messages
**Change**: Modify `verifyExecutingMessage` in `algo.go` to use `>` instead of `>=` for timestamp comparison.

**Unit Tests** (in `algo_test.go`):
- Test that a message with `initTimestamp == execTimestamp` is **allowed** (no longer returns `ErrTimestampViolation`)
- Test that a message with `initTimestamp > execTimestamp` is still **rejected** (future timestamps invalid)

**Acceptance Test Amendments**:
- `TestSupernodeSameTimestampExecMessage`: Change assertions to expect:
  - Chain A's block: NOT replaced (init message valid) ✓ (unchanged)
  - Chain B's block: **NOT replaced** (same-ts exec now valid)
  - Exec transaction: **STILL EXISTS** in block (not removed)
- `TestSupernodeSameTimestampInvalidTransitive`: Change assertions to expect:
  - Chain B replaced: **YES** (but only because of invalid log index 9999, NOT same-timestamp)
  - Chain A replaced: **YES** (transitive - references B's init which is gone)
  - Test comment updated to clarify: same-timestamp is NOT the cause of invalidity

---

### Subfeature 2: Add `cycleVerifyFn` to Interop Struct
**Change**: Add `cycleVerifyFn` field to `Interop` struct with signature:
```go
cycleVerifyFn func(ts uint64, blocksAtTimestamp map[eth.ChainID]eth.BlockID) (Result, error)
```

**Unit Tests**:
- Test that `cycleVerifyFn` can be mocked/set on the Interop struct
- Test that when mocked to succeed, same-timestamp verification passes
- Test that when mocked to fail with invalid heads, those invalid heads are propagated to the overall result

**Acceptance Test Amendments**: None needed (behavior unchanged from SF1)

---

### Subfeature 3: Route Same-Timestamp Messages Through `cycleVerifyFn`
**Change**: In the verification flow, detect same-timestamp executing messages and route them through `cycleVerifyFn` rather than immediate validation.

**Unit Tests**:
- Test that same-timestamp messages invoke `cycleVerifyFn` (can track with mock)
- Test that non-same-timestamp messages do NOT invoke `cycleVerifyFn`
- Test that `cycleVerifyFn` failure marks blocks as invalid
- Test that `cycleVerifyFn` success allows the block to be valid

**Acceptance Test Amendments**: None needed (default cycleVerifyFn should pass-through)

---

### Subfeature 4: Implement Cycle Verification in `circular.go`
**Change**: Create `circular.go` with the actual same-timestamp cycle verification algorithm.

**Unit Tests**: (To be defined when we reach this subfeature - algorithm details pending from developer)

**Acceptance Test Amendments**:
- Add new test cases for cycle-specific scenarios if needed
- Verify existing tests still pass with real cycle verification

---

## Current State

**Phase**: Development
**Current Subfeature**: 4 (COMPLETE) - Stub cycleVerifyFn implementation

---

## Diary

### Entry 1: Initial Planning (2026-02-17)

**Prompt received**:
> Let's begin. We are going to implement same-timestamp interop. the way we will do it is thusly:
> 1. to the interop function, next to the verifyFn, we will also add cycleVerifyFn...
> 2. because it is a pluggable function, let's begin with tests...
> 3. We will create a file called `circular.go` inside interop...

**Actions taken**:
1. Read `interop.go` to understand the Interop struct and `verifyFn` field
2. Read `interop_test.go` to understand the test harness pattern
3. Read `algo.go` to find the timestamp check at line 131: `if execMsg.Timestamp >= executingTimestamp`
4. Read `algo_test.go` to understand existing test patterns for verification logic
5. Created this Feature diary file with breakdown of subfeatures

**Key findings**:
- `verifyFn` signature: `func(ts uint64, blocksAtTimestamp map[eth.ChainID]eth.BlockID) (Result, error)`
- Current timestamp check: `execMsg.Timestamp >= executingTimestamp` (line 131 of `algo.go`)
- `Result` type has `InvalidHeads map[eth.ChainID]eth.BlockID` for tracking invalid blocks
- Test harness uses builder pattern (`interopTestHarness`) with mock chain containers
- Algorithm tests use `algoMockLogsDB` for mocking the logsDB layer

**Awaiting**: Developer approval of this plan before proceeding to test development.

---

### Entry 2: Subfeature 1 Complete (2026-02-17)

**Prompt received**:
> yes, looks good. let's proceed through subfeature 1

**Actions taken**:

1. **Unit Tests** (`algo_test.go`):
   - Added `ValidBlocks/SameTimestampMessage` test - verifies same-timestamp messages are now VALID
   - Renamed `InvalidBlocks/TimestampViolation` → `InvalidBlocks/FutureTimestamp` for clarity
   - Updated comments to clarify: only future timestamps (>) are invalid, same-timestamp (==) allowed

2. **Implementation** (`algo.go`):
   - Changed timestamp check from `>=` to `>` in `verifyExecutingMessage` (line 131)
   - Updated error constant `ErrTimestampViolation` message
   - Updated function documentation for `verifyInteropMessages` and `verifyExecutingMessage`

3. **Acceptance Tests** (`same_timestamp_invalid_test.go`):
   - Updated package spec comment to reflect new behavior (same-timestamp = VALID)
   - `TestSupernodeSameTimestampExecMessage`:
     - Changed assertions: Chain B's block should NOT be replaced
     - Exec transaction stays in block (block hash unchanged = tx preserved)
   - `TestSupernodeSameTimestampInvalidTransitive`:
     - Updated doc comment to clarify: invalidation is due to bad log index (9999), NOT same-timestamp
     - Assertions unchanged (both chains still replaced due to invalid log index + transitive)

4. **Verification**:
   - All unit tests pass: `go test ./supernode/activity/interop/`
   - No linter errors

**Files Modified**:
- `op-supernode/supernode/activity/interop/algo.go`
- `op-supernode/supernode/activity/interop/algo_test.go`
- `op-acceptance-tests/tests/supernode/interop/same_timestamp_invalid/same_timestamp_invalid_test.go`

**Ready for**: Commit and proceed to Subfeature 2 (Add `cycleVerifyFn` field)

---

### Entry 3: Subfeature 2 Complete (2026-02-17)

**Prompt received**:
> I made some slight modifications to comments, but yes continue to make the commit, and then Subfeature-2.
> And before we wire it in, here is a clarification.
> The way we should use this is that in interop:301, we call verifyFn. we should continue on to call cycleVerifyFn just after, and then join the results.

**Clarification incorporated**:
- `cycleVerifyFn` is called AFTER `verifyFn` in `progressInterop` (not during verification loop)
- Results are merged: invalid heads from both functions are combined

**Actions taken**:

1. **Committed Subfeature 1**: `3a9e3b1799` - "interop: allow same-timestamp executing messages"

2. **Implementation** (`interop.go`):
   - Added `cycleVerifyFn` field to `Interop` struct (line 58-60)
   - Modified `progressInterop` to call `cycleVerifyFn` after `verifyFn` (lines 297-316)
   - Merge logic: if `cycleVerifyFn` returns invalid heads, they are added to `result.InvalidHeads`
   - Handles nil map case when merging

3. **Unit Tests** (`interop_test.go`):
   - Added `TestProgressInteropWithCycleVerify` test section with 4 test cases:
     - `cycleVerifyFn nil uses only verifyFn result` - backwards compatible
     - `cycleVerifyFn called after verifyFn and results merged` - verifies call order
     - `cycleVerifyFn error propagated` - errors bubble up
     - `both verifyFn and cycleVerifyFn invalid heads are merged` - both contribute to result

4. **Verification**:
   - All unit tests pass: `go test ./supernode/activity/interop/`
   - No linter errors

**Files Modified**:
- `op-supernode/supernode/activity/interop/interop.go`
- `op-supernode/supernode/activity/interop/interop_test.go`

**Ready for**: Commit and proceed to Subfeature 3 (Route same-timestamp through cycleVerifyFn)

---

### Entry 4: Subfeature 4 Complete (2026-02-17)

**Prompt received**:
> yes, subfeature 3 "routing" was meant to be this routing to progressInterop. Now that's complete so we can do subfeature 4. For now just make it return a trivial "Valid" structure. We can also remove the "if i.cycleVerifyFn != nil" check in progressInterop, we will always have this function set. In New, it will use this stub function you're about to make.

**Actions taken**:

1. **Created `circular.go`**:
   - Added `verifyCycleMessages` method as stub implementation
   - Returns valid `Result` with all blocks in `L2Heads`
   - TODO comment for actual algorithm implementation

2. **Updated `New()` in `interop.go`**:
   - Set `i.cycleVerifyFn = i.verifyCycleMessages`
   - cycleVerifyFn is now always set (not optional)

3. **Removed nil check** in `progressInterop`:
   - Changed from `if i.cycleVerifyFn != nil` to unconditional call
   - cycleVerifyFn is always called after verifyFn

4. **Updated tests**:
   - Renamed test to `default cycleVerifyFn returns valid result`
   - Added `require.NotNil(t, interop.cycleVerifyFn)` to TestNew

5. **Verification**:
   - All unit tests pass: `go test ./supernode/activity/interop/`
   - No linter errors

**Files Modified**:
- `op-supernode/supernode/activity/interop/circular.go` (NEW)
- `op-supernode/supernode/activity/interop/interop.go`
- `op-supernode/supernode/activity/interop/interop_test.go`

**Ready for**: Commit. Actual cycle verification algorithm can be implemented in `verifyCycleMessages` when ready.
