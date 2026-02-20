# SuperRoot Refactor Feature

**Author:** AI Tool + Developer  
**Created:** 2026-02-18  
**Status:** Spec Complete - Ready for Test Development

---

## Description

Refactor the `superroot.AtTimestamp` RPC response to flatten the structure and change the semantics of what data is returned under what conditions. The goal is to make the response more useful by:

1. Always providing current chain state information (or zero values on failure)
2. Always providing optimistic data when derivable
3. Always providing SuperRoot when output roots are available for all chains
4. Only providing `VerifiedRequiredL1` when verification is complete (zero value signals optimistic data)

Additionally, update the Denylist to store output roots (`eth.Bytes32`) alongside block hashes so that when blocks are invalidated, we can return the "optimistic" output root that was replaced.

---

## Specification

### Response Structure (Flattened)

**Before:**
```go
type SuperRootAtTimestampResponse struct {
    CurrentL1                 BlockID
    CurrentSafeTimestamp      uint64
    CurrentFinalizedTimestamp uint64
    OptimisticAtTimestamp     map[ChainID]OutputWithRequiredL1
    ChainIDs                  []ChainID
    Data                      *SuperRootResponseData  // nil if any chain not verified
}

type SuperRootResponseData struct {
    VerifiedRequiredL1 BlockID
    Super              Super
    SuperRoot          Bytes32
}
```

**After:**
```go
type SuperRootAtTimestampResponse struct {
    // Current state - always populated (zero value on failure, no errors)
    CurrentL1                 BlockID
    CurrentSafeTimestamp      uint64
    CurrentFinalizedTimestamp uint64
    
    // Chain info - always populated
    ChainIDs                  []ChainID
    
    // Per-chain optimistic outputs - always populated if output root derivable
    OptimisticAtTimestamp     map[ChainID]OutputWithRequiredL1
    
    // Super root - always populated if ALL chains have output roots
    Super                     Super     // zero if any chain missing output
    SuperRoot                 Bytes32   // zero if any chain missing output
    
    // Verification status - zero value when not ALL chains verified
    // Zero value indicates Super/SuperRoot may be based on optimistic data
    VerifiedRequiredL1        BlockID   // zero = unverified/optimistic
}
```

**Interpretation:**
- `VerifiedRequiredL1 == BlockID{}` → SuperRoot is based on optimistic (unverified) data
- `SuperRoot == Bytes32{}` → Could not compute super root (missing output for one or more chains)

### Behavioral Changes

| Field | Current Behavior | New Behavior |
|-------|------------------|--------------|
| `CurrentL1` | Return error if SyncStatus fails | Return `BlockID{}`, continue processing |
| `CurrentSafeTimestamp` | Return error if SyncStatus fails | Return `0`, continue processing |
| `CurrentFinalizedTimestamp` | Return error if SyncStatus fails | Return `0`, continue processing |
| `OptimisticAtTimestamp[chain]` | Skip if `VerifiedAt` returns NotFound | Always populate if `OutputRootAtBlockNumber` succeeds |
| `Super` / `SuperRoot` | Only set if all chains verified | Always set if all chains have output roots |
| `VerifiedRequiredL1` | Inside `Data`, only if all verified | Top-level, zero value if any chain unverified |
| `Data` | Pointer, nil if unverified | **REMOVED** - fields moved to top level |

### Denylist Changes

**Current storage format:** `height → [payloadHash1, payloadHash2, ...]`

**New storage format:** `height → [(payloadHash1, outputRoot1), (payloadHash2, outputRoot2), ...]`

Each entry is 64 bytes: 32 bytes payload hash + 32 bytes output root.

**API Changes:**

```go
// Before
func (d *DenyList) Add(height uint64, payloadHash common.Hash) error
func (d *DenyList) Contains(height uint64, payloadHash common.Hash) (bool, error)
func (d *DenyList) GetDeniedHashes(height uint64) ([]common.Hash, error)

// After
func (d *DenyList) Add(height uint64, payloadHash common.Hash, outputRoot eth.Bytes32) error
func (d *DenyList) Contains(height uint64, payloadHash common.Hash) (bool, error)  // unchanged
func (d *DenyList) GetDeniedHashes(height uint64) ([]common.Hash, error)           // unchanged  
func (d *DenyList) GetOutputRoot(height uint64, payloadHash common.Hash) (eth.Bytes32, bool, error)  // NEW
```

**ChainContainer interface change:**
```go
// Before
InvalidateBlock(ctx context.Context, height uint64, payloadHash common.Hash) (bool, error)

// After
InvalidateBlock(ctx context.Context, height uint64, payloadHash common.Hash, outputRoot eth.Bytes32) (bool, error)
```

---

## Work Breakdown (Sub-features as Commits)

### Commit 1: Update eth types - flatten response structure
**Files:** `op-service/eth/superroot_at_timestamp.go`

- Add `Super`, `SuperRoot`, `VerifiedRequiredL1` fields to `SuperRootAtTimestampResponse`
- Remove (or deprecate) `SuperRootResponseData` type
- Update JSON marshalling if needed

**Tests to write:**
- JSON round-trip for new response structure
- Zero value semantics for `VerifiedRequiredL1`

---

### Commit 2: Update Denylist to store output roots
**Files:** `op-supernode/supernode/chain_container/invalidation.go`, `invalidation_test.go`

- Change storage format: each entry is `payloadHash (32) + outputRoot (32)`
- Update `Add()` to accept `outputRoot eth.Bytes32`
- Add `GetOutputRoot(height, payloadHash)` method
- Keep `Contains()` and `GetDeniedHashes()` backwards compatible

**Tests to write:**
- `Add` stores both hash and output root
- `GetOutputRoot` retrieves correct output root
- `GetOutputRoot` returns false for non-existent entries
- Persistence: output roots survive close/reopen
- Multiple entries at same height with different output roots

---

### Commit 3: Update ChainContainer interface and callers
**Files:** 
- `op-supernode/supernode/chain_container/chain_container.go` (interface)
- `op-supernode/supernode/chain_container/invalidation.go` (implementation)
- `op-supernode/supernode/activity/interop/interop.go` (caller)
- All mock implementations in test files

- Add `outputRoot` parameter to `InvalidateBlock`
- Update interop activity to pass output root when invalidating

**Tests to write:**
- `InvalidateBlock` stores output root in denylist
- Verify output root can be retrieved after invalidation

---

### Commit 4: Refactor superroot.atTimestamp logic
**Files:** `op-supernode/supernode/activity/superroot/superroot.go`, `superroot_test.go`

Restructure the function into phases:
1. **Current state collection** - Query SyncStatus for all chains, aggregate minimums, never error
2. **Output collection** - For each chain, get output root (optimistic path), populate `OptimisticAtTimestamp`
3. **Super root computation** - If all chains have outputs, compute `Super` and `SuperRoot`
4. **Verification check** - If all chains pass `VerifiedAt`, set `VerifiedRequiredL1`

**Tests to write:**
- All chains verified → `VerifiedRequiredL1` populated
- Some chains unverified → `VerifiedRequiredL1` is zero, but `SuperRoot` still populated
- Some chains missing output → `SuperRoot` is zero
- SyncStatus failure → Current values are zero, but processing continues
- OptimisticAtTimestamp populated for all chains with available outputs

---

### Commit 5: Update consumers and integration
**Files:** Various test files and any RPC consumers

- Update all callers that check `resp.Data != nil` to check `resp.SuperRoot != Bytes32{}`
- Update all callers that access `resp.Data.VerifiedRequiredL1` to use `resp.VerifiedRequiredL1`
- Fix all mock implementations

**Tests to verify:**
- Existing acceptance tests pass
- `op-challenger` provider tests updated

---

---

## Test Plan

Most changes involve modifying existing tests to use the new flattened response structure. Here's the breakdown:

### Commit 1: eth types - Response Structure Tests

**File:** `op-service/eth/superroot_at_timestamp_test.go` (NEW)

| Test | Purpose |
|------|---------|
| `TestSuperRootAtTimestampResponse_JSONRoundTrip` | Verify JSON marshal/unmarshal works with new structure |
| `TestSuperRootAtTimestampResponse_ZeroValues` | Verify zero values behave correctly for optional fields |

### Commit 2: Denylist - Output Root Storage Tests

**File:** `op-supernode/supernode/chain_container/invalidation_test.go` (MODIFY)

| Test | Modification |
|------|--------------|
| `TestDenyList_AddAndContains` | Update `Add()` calls to include output root parameter |
| `TestDenyList_Persistence` | Verify output roots persist across close/reopen |
| `TestDenyList_GetDeniedHashes` | No change (backwards compatible) |
| NEW: `TestDenyList_GetOutputRoot` | Test new method returns correct output root |
| NEW: `TestDenyList_GetOutputRoot_NotFound` | Returns false for non-existent entries |
| NEW: `TestDenyList_MultipleEntriesDifferentOutputRoots` | Multiple hashes at same height with different output roots |

### Commit 3: ChainContainer Interface Update Tests

**Files:** Multiple test files with mock implementations

| File | Change |
|------|--------|
| `invalidation_test.go` | Update `InvalidateBlock` test calls with output root |
| `interop_test.go` | Update mock `InvalidateBlock` signature |
| `superroot_test.go` | Update mock `InvalidateBlock` signature |
| `logdb_test.go` | Update mock `InvalidateBlock` signature |

### Commit 4: superroot.atTimestamp Logic Tests

**File:** `op-supernode/supernode/activity/superroot/superroot_test.go` (MODIFY)

| Test | Modification |
|------|--------------|
| `TestSuperroot_AtTimestamp_Succeeds` | Check `SuperRoot` and `VerifiedRequiredL1` directly (not via `.Data`) |
| `TestSuperroot_AtTimestamp_ComputesSuperRoot` | Check `resp.SuperRoot` directly |
| `TestSuperroot_AtTimestamp_ErrorOnCurrentL1` | Change: should NOT error, return zero values instead |
| `TestSuperroot_AtTimestamp_ErrorOnVerifiedAt` | Change: should still populate `SuperRoot` if output available |
| `TestSuperroot_AtTimestamp_NotFoundOnVerifiedAt` | Change: `SuperRoot` populated, `VerifiedRequiredL1` is zero |
| `TestSuperroot_AtTimestamp_ErrorOnOutputRoot` | Still errors (can't compute super root) |
| `TestSuperroot_AtTimestamp_ErrorOnOptimisticAt` | Still errors |
| `TestSuperroot_AtTimestamp_EmptyChains` | Check zero values for empty response |
| NEW: `TestSuperroot_AtTimestamp_SyncStatusFails_ContinuesProcessing` | Verify zero values but still computes super root |
| NEW: `TestSuperroot_AtTimestamp_PartialVerification` | Some chains verified, some not - SuperRoot populated, VerifiedRequiredL1 zero |
| NEW: `TestSuperroot_AtTimestamp_AllVerified` | All chains verified - both SuperRoot and VerifiedRequiredL1 populated |

### Commit 5: Consumer Tests Updates

**File:** `op-challenger/game/fault/trace/super/provider_supernode_test.go` (MODIFY)

| Test | Change |
|------|--------|
| All tests using `resp.Data.Super` | Change to `resp.Super` |
| All tests using `resp.Data.SuperRoot` | Change to `resp.SuperRoot` |
| All tests using `resp.Data.VerifiedRequiredL1` | Change to `resp.VerifiedRequiredL1` |
| All tests checking `resp.Data == nil` | Change to `resp.SuperRoot == eth.Bytes32{}` |
| `createValidSuperNodeSuperRoots()` helper | Update to use flattened structure |

**File:** `op-proposer/proposer/source/source_supernode_test.go` (MODIFY)

| Test | Change |
|------|--------|
| `TestSuperNodeSource_ProposalAtSequenceNum` | Update response construction, check `SuperRoot` directly |
| `Single-NoData` test | Change check from `Data: nil` to `SuperRoot == Bytes32{}` |

**File:** `op-service/sources/supernode_client_test.go` (MODIFY)

| Test | Change |
|------|--------|
| `Success` | Update expected response structure |
| `NotFound` | Change from `Data: nil` to zero `SuperRoot` |

**File:** `op-dispute-mon/mon/extract/super_agreement_enricher_test.go` (MODIFY)

| Test | Change |
|------|--------|
| `stubSuperNodeClient` | Update to return flattened response |
| All tests | Response construction uses new structure |

---

## Diary

### Entry 1: Initial Planning (2026-02-18)
**Prompt:** Developer asked to change super-root structure - flatten Data into main response, always return current/optimistic/super when possible, RequiredL1 only when verified. Also need to update denylist to store full output roots.

**Action:** Created feature file with initial spec.

---

### Entry 2: Clarifications (2026-02-18)
**Prompt:** Developer clarified: no pointers in RPC responses (use zero values), store only `eth.Bytes32` for output root (not full response), no backwards compatibility needed.

**Action:** Updated spec to use zero values instead of pointers, simplified denylist to store `eth.Bytes32`.

---

### Entry 3: Spec Finalized (2026-02-18)
**Prompt:** Developer approved questions, requested full plan and spec.

**Action:** Completed full specification with:
- Detailed before/after response structure
- Behavioral changes table
- Denylist API changes
- 5-commit work breakdown with test plans for each

**Status:** ✅ COMPLETE - All 5 commits implemented.

---

## Resolved Questions

1. ~~Should `Super` and `SuperRoot` be pointers?~~ **No** - use zero values (RPC response)
2. ~~Store `eth.Bytes32` or full `*eth.OutputResponse`?~~ **`eth.Bytes32`** only
3. ~~Backwards compatibility?~~ **None required**
