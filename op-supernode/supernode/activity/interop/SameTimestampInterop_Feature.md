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
**Current Subfeature**: 5 (COMPLETE) - `verifyCycleMessages` fully implemented

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

---

### Entry 5: checkCycle Algorithm Implemented (2026-02-17)

**Prompt received**:
> yes, it is time to explain to you the cycle detection algo so we can document the spec, make tests, and then make the implementation.
> [Algorithm description: Kahn's topological sort with dependencyNode structure]
> Start with a test plan.

**Algorithm specification documented** (see `circular.go` spec comment):
- Build dependency graph from same-timestamp executing messages
- Each node tracks `dependsOn` (outgoing) and `dependedOnBy` (incoming) edges
- Kahn's algorithm: iteratively remove nodes with no `dependedOnBy`, mark resolved
- Termination: all resolved = acyclic (valid), unresolved with empty removeSet = cycle (invalid)

**Actions taken**:

1. **Created `circular_test.go`**:
   - Test plan documented in header comment
   - `TestDependencyGraph_AddNode` - verifies node addition
   - `TestDependencyGraph_AddEdge` - verifies edge bidirectional linkage
   - `TestCheckCycle` - table-driven tests covering:
     - Empty graph (no cycle)
     - Single node, no deps (resolves)
     - Linear chain A→B→C (acyclic)
     - Simple cycle A↔B (cycle detected)
     - Triangle cycle A→B→C→A (cycle detected)
     - Diamond pattern A→B,C→D (acyclic)
     - Intra-chain sequential logs (resolves)
     - Cross-chain valid exec message (resolves)
     - Cross-chain mutual exec (cycle detected)

2. **Implemented `checkCycle` in `circular.go`**:
   - Added `ErrCycle` error variable
   - Implemented Kahn's topological sort algorithm:
     - Part 1: Find nodes with no `dependedOnBy`, mark resolved, add to removeSet
     - Part 2: Remove resolved nodes from `dependedOnBy` of all nodes they depend on
     - Terminates when all resolved (valid) or no progress (cycle)
   - Added `removeFromSlice` helper function

3. **Verification**:
   - All unit tests pass: `go test -v -run "TestCheckCycle|TestDependencyGraph" ./supernode/activity/interop/`
   - Full test suite passes: `go test ./supernode/activity/interop/...`

**Files Modified**:
- `op-supernode/supernode/activity/interop/circular.go`
- `op-supernode/supernode/activity/interop/circular_test.go` (NEW)

**Next**: Implement `verifyCycleMessages` to build the dependency graph from actual blockchain data and call `checkCycle`.

---

### Entry 6: verifyCycleMessages Fully Implemented (2026-02-17)

**Prompt received**:
> Yes, next we will write `verifyCycleMessages`. Here's something to consider:
> - for each Executing Message we add, we'll be attaching two outgoing dependencies:
>   - to the previous Executing Message for this chain if it exists
>   - to the location where the Executing Message points
> BUT HERE IS AN IMPORTANT POINT: rather than use the actual location of the Executing Message, we'll use a helper function called "ExecutingMessageBefore", which grabs the first Executing Message to have occured on that chain equal to or prior to the referenced log index.

**Key insight**: Dependencies are between **Executing Messages only**, not all logs. When an EM references a log index X, we find the latest EM with **logIndex <= X** on the target chain. This represents: "everything that happened up to and including this point must resolve first."

**Semantic clarification**: `<=` (not `<`) - so an EM referencing log 0 on a chain with an EM at log 0 WILL create a dependency.

**Actions taken**:

1. **Added `executingMessageBefore` helper** (`circular.go`):
   - Finds the latest EM with `logIndex <= targetLogIdx` in a sorted slice
   - Returns nil if no such EM exists (no dependency to add)

2. **Added `buildCycleGraph` function** (`circular.go`):
   - Takes `ts` and `chainEMs map[chainID]map[logIdx]*ExecutingMessage`
   - Creates nodes for same-timestamp EMs only
   - Adds intra-chain edges: EM depends on previous EM on same chain
   - Adds cross-chain edges: EM depends on `executingMessageBefore(targetChain, refLogIdx)`

3. **Updated `verifyCycleMessages`** (`circular.go`):
   - Collects same-timestamp EMs from all chains via `db.OpenBlock`
   - Calls `buildCycleGraph` to construct dependency graph
   - Calls `checkCycle` to detect cycles
   - Returns InvalidHeads for all chains with same-ts EMs if cycle detected

4. **Test cases** (`circular_test.go`):
   - `TestExecutingMessageBefore`: 6 test cases covering empty, no-match, exact match, latest before, edge cases
   - `TestBuildCycleGraph`: 10 test cases covering:
     - Empty graph, past-timestamp refs, single EM
     - Mutual refs at same log index → CYCLE
     - One-way dependencies → no cycle
     - Triangle patterns (both cycle and no-cycle variants)
     - Prior EMs creating mutual dependency → CYCLE

**Files Modified**:
- `op-supernode/supernode/activity/interop/circular.go`
- `op-supernode/supernode/activity/interop/circular_test.go`

**Status**: All unit tests pass. Feature implementation complete.

---

### Entry 7: Acceptance Test for Cycle Detection (2026-02-17)

**Prompt received**:
> Looking good. Let's package this up as another commit. Once done, create a test coverage table which describes all the testing so far.
> after, we will build a new Acceptance test... which demonstrates that cycle checks cause reorgs
> also let's rename the file `same_timestamp_test.go` since it's no longer strictly invalid.

**Actions taken**:

1. **Renamed acceptance test file**:
   - `same_timestamp_invalid_test.go` → `same_timestamp_test.go`
   - Updated package spec comment to document all three test scenarios

2. **Added `TestSupernodeSameTimestampCycle`**:
   - Creates mutual same-timestamp exec messages: A executes B, B executes A
   - Both chains emit init messages at log 0, exec messages at log 1
   - This creates a circular dependency that cycle detection identifies
   - Verifies both blocks are replaced after validation

3. **Commit**: `c9dcdf87c3` - "acceptance: add cycle detection test and rename same_timestamp_test.go"

**Files Modified**:
- `op-acceptance-tests/tests/supernode/interop/same_timestamp_invalid/same_timestamp_test.go` (renamed + new test)

---

## Test Coverage Summary

### Unit Tests (`op-supernode/supernode/activity/interop/`)

#### algo_test.go - Timestamp Validation
| Test | Description |
|------|-------------|
| `ValidBlocks/SameTimestampMessage` | Same-timestamp exec message is VALID |
| `InvalidBlocks/FutureTimestamp` | Future timestamp exec message is INVALID |

#### interop_test.go - cycleVerifyFn Integration
| Test | Description |
|------|-------------|
| `TestNew` | Verifies `cycleVerifyFn` is initialized (not nil) |
| `TestProgressInteropWithCycleVerify/default_cycleVerifyFn_returns_valid_result` | Default stub returns valid |
| `TestProgressInteropWithCycleVerify/cycleVerifyFn_called_after_verifyFn_and_results_merged` | Results merged correctly |
| `TestProgressInteropWithCycleVerify/cycleVerifyFn_error_propagated` | Errors bubble up |
| `TestProgressInteropWithCycleVerify/both_verifyFn_and_cycleVerifyFn_invalid_heads_are_merged` | InvalidHeads combined |

#### circular_test.go - Graph Construction & Cycle Detection
| Test | Description |
|------|-------------|
| `TestDependencyGraph_AddNode` | Node addition works |
| `TestDependencyGraph_AddEdge` | Edge creates bidirectional links |
| `TestExecutingMessageBefore/empty_chain_returns_nil` | Empty chain → nil |
| `TestExecutingMessageBefore/no_EM_at_or_before_target_returns_nil` | No match → nil |
| `TestExecutingMessageBefore/exact_match_returns_that_EM` | Exact match (<=) works |
| `TestExecutingMessageBefore/returns_latest_EM_at_or_before_target` | Latest EM ≤ target |
| `TestExecutingMessageBefore/target_at_index_0_with_EM_at_0_returns_that_EM` | Edge case: log 0 with EM |
| `TestExecutingMessageBefore/target_at_index_0_with_no_EM_at_0_returns_nil` | No EM at 0 → nil |
| `TestCheckCycle/empty_graph_has_no_cycle` | Empty → valid |
| `TestCheckCycle/single_node_no_deps_resolves` | Single node → valid |
| `TestCheckCycle/linear_chain_A->B->C_resolves_(acyclic)` | Linear → valid |
| `TestCheckCycle/simple_cycle_A<->B_detected` | Mutual dep → CYCLE |
| `TestCheckCycle/triangle_cycle_A->B->C->A_detected` | Triangle → CYCLE |
| `TestCheckCycle/diamond_pattern_resolves_(acyclic)` | Diamond → valid |
| `TestCheckCycle/intra-chain_sequential_logs_resolve` | Sequential → valid |
| `TestCheckCycle/cross-chain_valid_exec_message_resolves` | One-way cross-chain → valid |
| `TestCheckCycle/cross-chain_mutual_exec_creates_cycle` | Mutual cross-chain → CYCLE |
| `TestBuildCycleGraph/no_same-timestamp_EMs_returns_valid` | Empty → valid |
| `TestBuildCycleGraph/single_chain_single_EM_referencing_past_timestamp` | Past ts filtered |
| `TestBuildCycleGraph/single_chain_single_same-ts_EM_referencing_chain_with_no_EMs` | No target EMs → valid |
| `TestBuildCycleGraph/two_chains_mutual_same-ts_EMs_at_same_log_index` | A:0↔B:0 → CYCLE |
| `TestBuildCycleGraph/chain_A_refs_B_log_5_B_has_EM_at_3` | One-way → valid |
| `TestBuildCycleGraph/chain_A_refs_B_log_2_B_has_EM_only_at_3` | No dep (3>2) → valid |
| `TestBuildCycleGraph/sequential_EMs_on_same_chain` | Intra-chain only → valid |
| `TestBuildCycleGraph/triangle_at_same_log_index` | A:0→B:0→C:0→A:0 → CYCLE |
| `TestBuildCycleGraph/triangle_where_one_leg_has_no_matching_EM` | Missing leg → valid |
| `TestBuildCycleGraph/two_chains_with_prior_EMs_creating_mutual_dependency` | Prior EMs → CYCLE |

### Acceptance Tests (`op-acceptance-tests/tests/supernode/interop/same_timestamp_invalid/`)

| Test | Scenario | Expected Outcome |
|------|----------|------------------|
| `TestSupernodeSameTimestampExecMessage` | Chain B executes Chain A's init at same timestamp | VALID - no block replacement |
| `TestSupernodeSameTimestampInvalidTransitive` | Chain B exec has invalid log index (9999), causes transitive invalidation | INVALID - both blocks replaced |
| `TestSupernodeSameTimestampCycle` | Mutual same-ts exec messages (A↔B) create cycle | INVALID - both blocks replaced (cycle detected) |
