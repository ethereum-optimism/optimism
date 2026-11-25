# Dafny Spec Check Report

> **Project**: `op-supernode`  
> **Dafny model**: `dafny-models`  
> **Generated**: 2026-07-01

---

## Summary

| # | Go Function | Dafny Method | Pre ✓ | Pre ? | Pre ✗ | Post ✓ | Post ? | Post ✗ |
|---|---|---|---|---|---|---|---|---|
| 1 | `persistFrontierLogs` | `Interop.PersistFrontierLogs` | 5 | 0 | 0 | 1 | 4 | 0 |
| 2 | `processBlockLogs` | `Interop.ProcessBlock` | 1 | 5 | 0 | 0 | 1 | 1 |
| 3 | `refreshCurrentL1OnWait` | `Interop.RefreshCurrentL1OnWait` | 1 | 0 | 0 | 1 | 0 | 0 |
| 4 | `verifyExecutingMessage` | `Interop.VerifyExecutingMessage` | 2 | 0 | 0 | 2 | 1 | 0 |
| 5 | `l1Inclusion` | `Interop.ComputeL1Inclusion` | 1 | 0 | 0 | 1 | 0 | 0 |
| 6 | `resetChainEnginesIfNeeded` | `Interop.RewindChainEngines` | 7 | 0 | 0 | 6 | 0 | 0 |
| 7 | `resolveFrontierVerificationView` | `Interop.ResolveFrontierVerificationView` | 3 | 0 | 0 | 2 | 1 | 0 |
| 8 | `checkChainsReady` | `Interop.CheckChainsReady` | 0 | 3 | 0 | 4 | 1 | 0 |
| 9 | `buildPendingTransition` | `Interop.BuildPendingTransition` | 6 | 0 | 0 | 3 | 0 | 3 |
| 10 | `verifyInteropMessages` | `Interop.VerifyMessages` | 0 | 4 | 0 | 2 | 1 | 1 |
| 11 | `applyRewindPlan` | `Interop.ApplyRewindPlan` | 10 | 0 | 0 | 8 | 2 | 0 |
| 12 | `verify` | `Interop.Verify` | 2 | 2 | 0 | 4 | 0 | 0 |
| 13 | `observeRound` | `Interop.ObserveRound` | 2 | 0 | 0 | 3 | 0 | 3 |
| 14 | `applyPendingTransition` | `Interop.ApplyPendingTransition` | 6 | 0 | 0 | 1 | 8 | 0 |
| 15 | `progressInterop` | `Interop.ProgressInterop` | 3 | 0 | 0 | 6 | 2 | 0 |
| 16 | `progressAndRecord` | `Interop.ProgressAndRecord` | 0 | 4 | 0 | 4 | 0 | 0 |

---

## 1. `persistFrontierLogs` → `Interop.PersistFrontierLogs`

**Go file**: `supernode/activity/interop/logdb.go:82`  
**Dafny**: `dafny-models/Interop.dfy` — class `Interop`, method `PersistFrontierLogs`  
**Notes**: Go validates timestamp monotonicity and enforces a maximum gap size (no more than one block time between consecutive timestamps) inside sealBlockDataIntoLogsDB; the Dafny model uses assume for these parent-hash and monotonicity conditions. Go also iterates over a map (nondeterministic order), while the Dafny model sequences log persistence deterministically.  

### Preconditions

- [x] `Valid()` — **Satisfied**
- [x] `blocksAtTS.Keys == chains.Keys` — **Satisfied**
- [x] `AdvancesAllLogsDBs(ts, blocksAtTS)` — **Satisfied**
- [x] `BlocksExistedOnChain(blocksAtTS)` — **Satisfied**
- [x] `AllLogsDBsConsistentWithChainData()` — **Satisfied**

### Postconditions

- [?] `Valid()` — **Uncertain** — Go's processBlockLogs can return an I/O error with no analog in the Dafny model — ProcessBlock is axiomatically assumed to always succeed (ensures {:axiom} Valid()). If processBlockLogs for chain C fails after partially writing to logsDBs[C] (e.g., block data written but seal not committed), the sub-predicates AllLogsDBsConsistentWithChainData() and VerifiedHeadsAreHighestBlocksUpToTimestamp() that compose Valid() may no longer hold for chain C. The ProcessBlock axiom only covers the success path and therefore does not bound this Go-specific failure mode.
- [?] `AdvancesAllLogsDBs(ts, blocksAtTS)` — **Uncertain** — If processBlockLogs fails in Go for chain C after partially updating logsDBs[C], logsDBs[C].LatestSealedBlock() may be left in an intermediate state — neither the old value (which satisfied AdvancesLogsDB by precondition 3) nor Some(blocksAtTS[C]) (which would satisfy it trivially). In that case AdvancesLogsDB(ts, C, blocksAtTS[C]) can fail the check latestBlock.number <= newBlock.number <= latestBlock.number + 1 evaluated in the post-state, violating AdvancesAllLogsDBs.
- [?] `AllLogsDBsConsistentWithChainData()` — **Uncertain** — Same root cause as post-conditions 1 and 2: if processBlockLogs fails in Go after writing a partial seal entry for chain C, FindSealedBlock may return a record whose on-chain block data (BlockInfo / BlockLogs) does not match what was committed to the logsDB, directly violating LogsDBConsistentWithChainData(C) and therefore AllLogsDBsConsistentWithChainData(). This failure path has no representation in the Dafny spec.
- [x] `success ==> UpdatedAllLogsDBs(blocksAtTS)` — **Satisfied**
- [?] `!success ==> forall chainID :: chainID in logsDBs.Keys ==> UpdatedLogsDB(chainID, blocksAtTS[chainID]) || unchanged(logsDBs[chainID])` — **Uncertain** — If processBlockLogs for chain C returns a Go I/O error after partially modifying logsDBs[C] (a failure mode absent from the Dafny spec), neither disjunct holds: UpdatedLogsDB(C, blocksAtTS[C]) requires LatestSealedBlock() == Some(blocksAtTS[C]) and a specific framing on all other block numbers, which a half-written logsDB does not satisfy; unchanged(logsDBs[C]) requires every FindSealedBlock entry to equal its old value, which the partial write violates. Go's nondeterministic map iteration order also means the identity of chain C is non-deterministic, so this scenario can affect any element of logsDBs.Keys.

### Callsite Analysis

| Caller | Precondition Verdicts |
|---|---|
| `applyPendingTransition` (supernode/activity/interop/interop.go:757) | `Valid()` ✓ — applyPendingTransition is the Go body of ApplyPendingTransition, whose Dafny spec carries Valid() as a precondition. The DecisionAdvance branch performs no state-mutating operations before line 757 (only a nil-guard on pending.Result); Valid() is therefore intact at the call site.; `blocksAtTS.Keys == chains.Keys` ✓ — ValidPendingTransition(pending) — implied by Valid() (a precondition of ApplyPendingTransition) — requires pending.result.value.l2Heads.Keys == CHAIN_IDS for an Advance decision. Valid() also requires chains.Keys == CHAIN_IDS. Together they give pending.Result.L2Heads.Keys == i.chains.Keys, which is exactly blocksAtTS.Keys == chains.Keys at the call site.; `AdvancesAllLogsDBs(ts, blocksAtTS)` ✓ — PendingTransitionIsConsistent() is a precondition of ApplyPendingTransition. For a stored Advance pending transition, TransitionConsistentWithLogs(pending) holds, which directly asserts AdvancesAllLogsDBs(pending.result.value.timestamp, pending.result.value.l2Heads). These are the exact ts and blocksAtTS values passed to persistFrontierLogs. No logsDB mutations intervene between ApplyPendingTransition entry and line 757.; `BlocksExistedOnChain(blocksAtTS)` ✓ — PendingTransitionIsConsistent() is a precondition of ApplyPendingTransition. For a stored Advance transition, TransitionConsistentWithChainState(pending) holds, which asserts BlocksExistedOnChain(pending.result.value.l2Heads). That map is identical to blocksAtTS at the call site. No chain-state mutations occur before line 757.; `AllLogsDBsConsistentWithChainData()` ✓ — AllLogsDBsConsistentWithChainData() is a direct precondition of ApplyPendingTransition. The DecisionAdvance branch writes to no logsDB before reaching line 757 (only a nil-guard and the pending.Result nil-check execute), so the invariant is preserved intact at the call site. |

### Violation Scenarios

**Postcondition `Valid()`** (uncertain):

Go's processBlockLogs can return an I/O error with no analog in the Dafny model — ProcessBlock is axiomatically assumed to always succeed (ensures {:axiom} Valid()). If processBlockLogs for chain C fails after partially writing to logsDBs[C] (e.g., block data written but seal not committed), the sub-predicates AllLogsDBsConsistentWithChainData() and VerifiedHeadsAreHighestBlocksUpToTimestamp() that compose Valid() may no longer hold for chain C. The ProcessBlock axiom only covers the success path and therefore does not bound this Go-specific failure mode.

**Postcondition `AdvancesAllLogsDBs(ts, blocksAtTS)`** (uncertain):

If processBlockLogs fails in Go for chain C after partially updating logsDBs[C], logsDBs[C].LatestSealedBlock() may be left in an intermediate state — neither the old value (which satisfied AdvancesLogsDB by precondition 3) nor Some(blocksAtTS[C]) (which would satisfy it trivially). In that case AdvancesLogsDB(ts, C, blocksAtTS[C]) can fail the check latestBlock.number <= newBlock.number <= latestBlock.number + 1 evaluated in the post-state, violating AdvancesAllLogsDBs.

**Postcondition `AllLogsDBsConsistentWithChainData()`** (uncertain):

Same root cause as post-conditions 1 and 2: if processBlockLogs fails in Go after writing a partial seal entry for chain C, FindSealedBlock may return a record whose on-chain block data (BlockInfo / BlockLogs) does not match what was committed to the logsDB, directly violating LogsDBConsistentWithChainData(C) and therefore AllLogsDBsConsistentWithChainData(). This failure path has no representation in the Dafny spec.

**Postcondition `!success ==> forall chainID :: chainID in logsDBs.Keys ==> UpdatedLogsDB(chainID, blocksAtTS[chainID]) || unchanged(logsDBs[chainID])`** (uncertain):

If processBlockLogs for chain C returns a Go I/O error after partially modifying logsDBs[C] (a failure mode absent from the Dafny spec), neither disjunct holds: UpdatedLogsDB(C, blocksAtTS[C]) requires LatestSealedBlock() == Some(blocksAtTS[C]) and a specific framing on all other block numbers, which a half-written logsDB does not satisfy; unchanged(logsDBs[C]) requires every FindSealedBlock entry to equal its old value, which the partial write violates. Go's nondeterministic map iteration order also means the identity of chain C is non-deterministic, so this scenario can affect any element of logsDBs.Keys.

---

## 2. `processBlockLogs` → `Interop.ProcessBlock`

**Go file**: `supernode/activity/interop/logdb.go:222`  
**Dafny**: `dafny-models/Interop.dfy` — class `Interop`, method `ProcessBlock`  
**Notes**: The Dafny method is an axiom stub (ensures {:axiom} Valid(), ensures {:axiom} UpdatedLogsDB). In Go, `processBlockLogs` handles a special first-block case by sealing a virtual parent block when the logsDB is empty; it then iterates over all receipt logs, decodes executing messages when present, calls AddLog for each, and seals the block at the end. The parent-hash and block-number preconditions from the Dafny spec are enforced by the outer caller (`sealBlockDataIntoLogsDB`) rather than by `processBlockLogs` itself.  

### Preconditions

- [x] `chainID in CHAIN_IDS` — **Satisfied**
- [?] `Valid()` — **Uncertain** — If sealBlockDataIntoLogsDB is invoked from a context where Valid() does not hold (e.g., mid-mutation by a concurrent or incompletely applied operation), Valid() would be absent at the processBlockLogs call site.
- [?] `BlockExistedOnChain(chainID, blockID)` — **Uncertain** — If sealBlockDataIntoLogsDB is called with a blockID that does not correspond to any block on chainID in the chain container (e.g., a speculative or incorrect block), BlockExistedOnChain(chainID, blockID) fails.
- [?] `info == chains[chainID].BlockInfo(blockID).value` — **Uncertain** — If the blockInfo supplied to sealBlockDataIntoLogsDB is stale, belongs to a different fork, or is otherwise non-canonical for blockID on chainID, the precondition fails.
- [?] `logs == chains[chainID].BlockLogs(blockID).value` — **Uncertain** — If the receipts parameter does not match the canonical BlockLogs for blockID on chainID (e.g., receipts from a reorged fork), the precondition fails.
- [?] `logsDBs[chainID].LatestSealedBlock().Some? ==> logsDBs[chainID].LatestSealedBlock().value.number + 1 == blockID.number && info.parentHash == logsDBs[chainID].LatestSealedBlock().value.hash` — **Uncertain** — Under non-canonical block data (preconditions 3/4 not guaranteed), blockID.Number could exceed latestBlock.Number by more than 1 while blockInfo.ParentHash() still equals latestBlock.Hash if blockInfo is malformed. The Go code lacks an explicit numeric check that blockID.Number == latestBlock.Number + 1.

### Postconditions

- [?] `{:axiom} Valid()` — **Uncertain** — processBlockLogs calls db.AddLog for each receipt log and db.SealBlock twice in the first-block case (virtual parent plus blockID). These operations modify one chain's logsDB. Whether all Valid() invariants are preserved — in particular AllLogsDBsConsistentWithChainData (requiring block seals to reflect canonical chain data), VerifiedHeadsAreHighestBlocksUpToTimestamp, and logsDBs distinctness — depends on the correctness of the LogsDB implementation and the canonicality of the supplied blockInfo and receipts. This cannot be verified from the shown code.
- [ ] `{:axiom} UpdatedLogsDB(chainID, blockID)` — **Violated** — UpdatedLogsDB requires `forall n :: n != blockID.number ==> db.FindSealedBlock(n) == old(db.FindSealedBlock(n))`. In the first-block case (isFirstBlock == true && blockNum > 0), processBlockLogs calls db.SealBlock(common.Hash{}, parentBlock, blockInfo.Time()) at approximately line 242 before iterating logs. This seals the virtual parent at blockNum-1. Because the logsDB is empty before the call, old(db.FindSealedBlock(blockNum-1)) == None, but after the call FindSealedBlock(blockNum-1) == Some(parentBlock seal). Since blockNum-1 != blockID.number (= blockNum), this change violates the forall constraint, falsifying UpdatedLogsDB.

### Callsite Analysis

| Caller | Precondition Verdicts |
|---|---|
| `sealBlockDataIntoLogsDB` (supernode/activity/interop/logdb.go:143) | `chainID in CHAIN_IDS` ✓ — sealBlockDataIntoLogsDB guards with `chain, ok := i.chains[chainID]; if !ok { return nil }` before reaching the processBlockLogs call site. processBlockLogs is therefore only called when chainID ∈ i.chains. Valid() guarantees chains.Keys == CHAIN_IDS, so chainID ∈ CHAIN_IDS holds at that point.; `Valid()` ? — sealBlockDataIntoLogsDB neither asserts nor re-establishes Valid() before calling processBlockLogs. The code between function entry and the call modifies no Interop state, so Valid() is preserved if it held at entry. However, the callers of sealBlockDataIntoLogsDB are not shown; whether Valid() holds at entry cannot be determined from the shown code.; `BlockExistedOnChain(chainID, blockID)` ? — sealBlockDataIntoLogsDB receives blockInfo and blockID as caller-supplied parameters. No call to chains[chainID].BlockInfo(blockID) is made to verify the block exists on the canonical chain. Satisfaction depends entirely on callers of sealBlockDataIntoLogsDB (not shown) guaranteeing canonical block data.; `info == chains[chainID].BlockInfo(blockID).value` ? — sealBlockDataIntoLogsDB passes the blockInfo parameter directly to processBlockLogs without verifying it equals i.chains[chainID].BlockInfo(blockID). The mapping between the Go blockInfo argument and the canonical chain block info is not checked inside the shown caller.; `logs == chains[chainID].BlockLogs(blockID).value` ? — sealBlockDataIntoLogsDB passes the receipts parameter directly to processBlockLogs without comparing it to i.chains[chainID].BlockLogs(blockID). Correctness depends on callers of sealBlockDataIntoLogsDB (not shown) supplying canonical receipt data.; `logsDBs[chainID].LatestSealedBlock().Some? ==> logsDBs[chainID].LatestSealedBlock().value.number + 1 == blockID.number && info.parentHash == logsDBs[chainID].LatestSealedBlock().value.hash` ? — When hasBlocks is true, sealBlockDataIntoLogsDB explicitly checks `blockInfo.ParentHash() != latestBlock.Hash` and returns an error if mismatched, enforcing the info.parentHash == hash part. The number+1 relationship is not checked directly: the code only establishes latestBlock.Number < blockID.Number (by eliminating the > and == cases). The exact +1 is implied only under canonical block data (blockInfo.ParentHash() being the hash of block blockID.Number-1), which depends on uncertain preconditions 3 and 4 holding. |

### Violation Scenarios

**Precondition `Valid()`** (uncertain):

If sealBlockDataIntoLogsDB is invoked from a context where Valid() does not hold (e.g., mid-mutation by a concurrent or incompletely applied operation), Valid() would be absent at the processBlockLogs call site.

**Precondition `BlockExistedOnChain(chainID, blockID)`** (uncertain):

If sealBlockDataIntoLogsDB is called with a blockID that does not correspond to any block on chainID in the chain container (e.g., a speculative or incorrect block), BlockExistedOnChain(chainID, blockID) fails.

**Precondition `info == chains[chainID].BlockInfo(blockID).value`** (uncertain):

If the blockInfo supplied to sealBlockDataIntoLogsDB is stale, belongs to a different fork, or is otherwise non-canonical for blockID on chainID, the precondition fails.

**Precondition `logs == chains[chainID].BlockLogs(blockID).value`** (uncertain):

If the receipts parameter does not match the canonical BlockLogs for blockID on chainID (e.g., receipts from a reorged fork), the precondition fails.

**Precondition `logsDBs[chainID].LatestSealedBlock().Some? ==> logsDBs[chainID].LatestSealedBlock().value.number + 1 == blockID.number && info.parentHash == logsDBs[chainID].LatestSealedBlock().value.hash`** (uncertain):

Under non-canonical block data (preconditions 3/4 not guaranteed), blockID.Number could exceed latestBlock.Number by more than 1 while blockInfo.ParentHash() still equals latestBlock.Hash if blockInfo is malformed. The Go code lacks an explicit numeric check that blockID.Number == latestBlock.Number + 1.

**Postcondition `{:axiom} Valid()`** (uncertain):

processBlockLogs calls db.AddLog for each receipt log and db.SealBlock twice in the first-block case (virtual parent plus blockID). These operations modify one chain's logsDB. Whether all Valid() invariants are preserved — in particular AllLogsDBsConsistentWithChainData (requiring block seals to reflect canonical chain data), VerifiedHeadsAreHighestBlocksUpToTimestamp, and logsDBs distinctness — depends on the correctness of the LogsDB implementation and the canonicality of the supplied blockInfo and receipts. This cannot be verified from the shown code.

**Postcondition `{:axiom} UpdatedLogsDB(chainID, blockID)`** (violated):

UpdatedLogsDB requires `forall n :: n != blockID.number ==> db.FindSealedBlock(n) == old(db.FindSealedBlock(n))`. In the first-block case (isFirstBlock == true && blockNum > 0), processBlockLogs calls db.SealBlock(common.Hash{}, parentBlock, blockInfo.Time()) at approximately line 242 before iterating logs. This seals the virtual parent at blockNum-1. Because the logsDB is empty before the call, old(db.FindSealedBlock(blockNum-1)) == None, but after the call FindSealedBlock(blockNum-1) == Some(parentBlock seal). Since blockNum-1 != blockID.number (= blockNum), this change violates the forall constraint, falsifying UpdatedLogsDB.

---

## 3. `refreshCurrentL1OnWait` → `Interop.RefreshCurrentL1OnWait`

**Go file**: `supernode/activity/interop/interop.go:496`  
**Dafny**: `dafny-models/Interop.dfy` — class `Interop`, method `RefreshCurrentL1OnWait`  
**Notes**: The Dafny method is an axiom stub (ensures {:axiom} Valid()). In Go, `refreshCurrentL1OnWait` calls `collectCurrentL1` and updates `currentL1` under a mutex lock. If collection fails, the error is logged and the function returns gracefully with (false, nil); this silent-failure path is absent from the Dafny model.  

### Preconditions

- [x] `Valid()` — **Satisfied**

### Postconditions

- [x] `{:axiom} Valid()` — **Satisfied**

### Callsite Analysis

| Caller | Precondition Verdicts |
|---|---|
| `progressAndRecord` (supernode/activity/interop/interop.go:477) | `Valid()` ✓ — progressAndRecord has Valid() as its own precondition. The only call that modifies state before reaching line 477 is progressInterop(), whose postcondition ensures Valid() (modifies only chains.Values). The metrics increment between progressInterop's return and the refreshCurrentL1OnWait call is not in Valid()'s reads clause (reads verifiedDB, logsDBs.Values), so Valid() remains unaffected. The pending != nil early-return path bypasses refreshCurrentL1OnWait entirely. The err != nil early-return after progressInterop also bypasses it. Therefore, at line 477, Valid() is guaranteed by progressInterop's postcondition. |

---

## 4. `verifyExecutingMessage` → `Interop.VerifyExecutingMessage`

**Go file**: `supernode/activity/interop/algo.go:184`  
**Dafny**: `dafny-models/Interop.dfy` — class `Interop`, method `VerifyExecutingMessage`  
**Notes**: This function is also a co_function of the VerifyMessages mapping entry; here it is checked directly against its own Dafny spec. In Go, the function returns error (nil = valid, non-nil = invalid) rather than bool. Go performs two separate ErrUnknownChain checks (for the source logsDB and for both chains map entries) before the activation checks; the Dafny model uses a single combined guard. Same-timestamp messages are checked against the frontier view first in both models.  

### Preconditions

- [x] `Valid()` — **Satisfied**
- [x] `executingChain in CHAIN_IDS` — **Satisfied**

### Postconditions

- [x] `Valid()` — **Satisfied**
- [x] `valid ==> ValidExecutingMessage(executingTimestamp, executingChain, execMsg)` — **Satisfied**
- [?] `valid ==> if execMsg.timestamp == executingTimestamp then InitMsgInFrontierView(execMsg, view) else assert execMsg.timestamp < executingTimestamp; InitMsgInLogsDB(execMsg)` — **Uncertain** — execMsg.Timestamp == executingTimestamp (line 233), view.contains(execMsg.ChainID, query) returns ok=false (e.g., the frontier block for execMsg.ChainID has a different block number), but i.logsDBs[execMsg.ChainID] contains a sealed block at execMsg.BlockNum with timestamp executingTimestamp and matching checksum. sourceDB.Contains(query) at line 240 returns nil, so the function returns nil (valid), yet InitMsgInFrontierView(execMsg, view) is false. Under Valid() alone, VerifiedHeadsAreHighestBlocksUpToTimestamp() only requires logsDB blocks above the verified head to have timestamps strictly greater than the verified timestamp, which permits a block with timestamp == executingTimestamp. The scenario is excluded in practice by the caller's AdvancesAllLogsDBs(ts, blocksAtTS) + AllDBsInSync() invariants (ensuring all logsDB blocks carry timestamps < executingTimestamp), but those are preconditions of VerifyMessages, not of VerifyExecutingMessage itself.

### Callsite Analysis

| Caller | Precondition Verdicts |
|---|---|
| `verifyInteropMessages` (supernode/activity/interop/algo.go:151) | `Valid()` ✓ — VerifyMessages requires Valid() and has no modifies clause (frame-safe), so Valid() holds throughout the caller's execution up to and including the call at line 151. No mutations of Interop state occur before this call site.; `executingChain in CHAIN_IDS` ✓ — The argument is chainID, drawn from iterating blocksAtTimestamp. VerifyMessages requires blocksAtTS.Keys == CHAIN_IDS, so every chainID in the iteration is guaranteed to be in CHAIN_IDS. |

### Violation Scenarios

**Postcondition `valid ==> if execMsg.timestamp == executingTimestamp then InitMsgInFrontierView(execMsg, view) else assert execMsg.timestamp < executingTimestamp; InitMsgInLogsDB(execMsg)`** (uncertain):

execMsg.Timestamp == executingTimestamp (line 233), view.contains(execMsg.ChainID, query) returns ok=false (e.g., the frontier block for execMsg.ChainID has a different block number), but i.logsDBs[execMsg.ChainID] contains a sealed block at execMsg.BlockNum with timestamp executingTimestamp and matching checksum. sourceDB.Contains(query) at line 240 returns nil, so the function returns nil (valid), yet InitMsgInFrontierView(execMsg, view) is false. Under Valid() alone, VerifiedHeadsAreHighestBlocksUpToTimestamp() only requires logsDB blocks above the verified head to have timestamps strictly greater than the verified timestamp, which permits a block with timestamp == executingTimestamp. The scenario is excluded in practice by the caller's AdvancesAllLogsDBs(ts, blocksAtTS) + AllDBsInSync() invariants (ensuring all logsDB blocks carry timestamps < executingTimestamp), but those are preconditions of VerifyMessages, not of VerifyExecutingMessage itself.

---

## 5. `l1Inclusion` → `Interop.ComputeL1Inclusion`

**Go file**: `supernode/activity/interop/algo.go:43`  
**Dafny**: `dafny-models/Interop.dfy` — class `Interop`, method `ComputeL1Inclusion`  
**Notes**: The Dafny method is an axiom stub (ensures {:axiom} Valid()). In Go, `l1Inclusion` iterates over `blocksAtTimestamp`, skips chains absent from `i.chains`, and selects the L1 block with the highest block number from `l1Heads`. If a chain is present in `blocksAtTimestamp` but absent from `l1Heads`, Go returns an error; the Dafny axiom stub does not model this failure path.  

### Preconditions

- [x] `Valid()` — **Satisfied**

### Postconditions

- [x] `{:axiom} Valid()` — **Satisfied**

### Callsite Analysis

| Caller | Precondition Verdicts |
|---|---|
| `verifyInteropMessages` (supernode/activity/interop/algo.go:76) | `Valid()` ✓ — verifyInteropMessages maps to Interop.VerifyMessages, which has Valid() as its own precondition. The call to i.l1Inclusion is the first substantive operation in the body (line 76), executed before any state-mutating calls. VerifyMessages has no modifies clause, so no receiver state can have changed before this call site. |

---

## 6. `resetChainEnginesIfNeeded` → `Interop.RewindChainEngines`

**Go file**: `supernode/activity/interop/interop.go:905`  
**Dafny**: `dafny-models/Interop.dfy` — class `Interop`, method `RewindChainEngines`  
**Notes**: This function is also a co_function of the ApplyRewindPlan mapping entry; here it is checked directly against its own Dafny spec. In Go, `resetChainEnginesIfNeeded` returns early if `plan.ResetAllChainsTo == nil`, which corresponds to the `plan.resetAllChainsTo.None?` guard inside the Dafny loop body. Errors are accumulated via a `recordErr` callback in Go rather than a boolean `failedAny` flag as in Dafny.  

### Preconditions

- [x] `Valid()` — **Satisfied**
- [x] `ValidRewindPlan(plan)` — **Satisfied**
- [x] `PlanConsistentWithVerified(plan)` — **Satisfied**
- [x] `RewoundVerifiedDB(plan)` — **Satisfied**
- [x] `AllVerifiedCrossValid()` — **Satisfied**
- [x] `forall k :: k in chainIDs <==> k in chains.Keys` — **Satisfied**
- [x] `forall k :: k in chainIDs ==> PlanConsistentWithLogs(plan, k)` — **Satisfied**

### Postconditions

- [x] `Valid()` — **Satisfied**
- [x] `ValidRewindPlan(plan)` — **Satisfied**
- [x] `RewoundVerifiedDB(plan)` — **Satisfied**
- [x] `AllVerifiedCrossValid()` — **Satisfied**
- [x] `forall k :: k in chainIDs ==> PlanConsistentWithLogs(plan, k)` — **Satisfied**
- [x] `unchanged(logsDBs.Values)` — **Satisfied**

### Callsite Analysis

| Caller | Precondition Verdicts |
|---|---|
| `applyRewindPlan` (supernode/activity/interop/interop.go:876) | `Valid()` ✓ — applyRewindPlan's Dafny spec requires Valid() on entry. Before line 876, three operations occur: (1) verifiedDB.Rewind — VerifiedDB.Rewind spec maintains VerifiedDB.Valid() and preserves pendingTransition; under ValidRewindPlan the None case means rewindAtOrAfter <= ACTIVATION_TIMESTAMP, so all entries (each ts >= activationTimestamp >= rewindAtOrAfter) are removed, leaving an empty DB. (2) PruneDeniedAtOrAfterTimestamp — modifies chain containers; its spec has no Interop-level postcondition. (3) db.Clear() on all logsDBs — LogsDB.Clear ensures LatestSealedBlock()==None, making AllLogsDBsConsistentWithChainData() and VerifiedHeadsAreHighestBlocksUpToTimestamp() vacuously true over the now-empty verifiedDB. Valid() is maintained.; `ValidRewindPlan(plan)` ✓ — applyRewindPlan's Dafny spec lists ValidRewindPlan(plan) as a precondition. The plan is a Go value type passed through unchanged; no intermediate operation modifies it or the constants (ACTIVATION_TIMESTAMP, CHAIN_IDS) on which ValidRewindPlan depends.; `PlanConsistentWithVerified(plan)` ✓ — applyRewindPlan requires PlanConsistentWithVerified(plan). For the None case (the only valid case when TargetHeads is nil under ValidRewindPlan), the predicate's antecedent plan.resetAllChainsTo.Some? is false, so the predicate holds trivially after the verifiedDB.Rewind. No logsDB modifications affect verifiedDB state.; `RewoundVerifiedDB(plan)` ✓ — applyRewindPlan's preconditions establish verifiedDB.pendingTransition.Some? with decision==Rewind and rewind==Some(plan). VerifiedDB.Rewind spec guarantees pendingTransition == old(pendingTransition), so the decision/rewind fields are preserved. For the None case, rewindAtOrAfter <= ACTIVATION_TIMESTAMP, so VerifiedDB.Rewind removes all entries (all ts >= activationTimestamp >= rewindAtOrAfter), giving |verifiedDB.db|==0. db.Clear() and PruneDenied do not modify verifiedDB.; `AllVerifiedCrossValid()` ✓ — applyRewindPlan requires AllVerifiedCrossValid(). Under the None case, verifiedDB.db is empty after the rewind, so AllVerifiedCrossValid() is vacuously true (the outer implication on LastTimestamp().Some? is false). db.Clear() on all logsDBs does not affect this vacuously empty verifiedDB condition.; `forall k :: k in chainIDs <==> k in chains.Keys` ✓ — sortedChainIDs is constructed by ranging over i.chains (lines immediately before the call in applyRewindPlan), so it contains exactly the keys of i.chains. The Dafny quantifier forall k :: k in chainIDs <==> k in chains.Keys is therefore satisfied by construction; no subsequent operation adds or removes chains from i.chains.; `forall k :: k in chainIDs ==> PlanConsistentWithLogs(plan, k)` ✓ — Under ValidRewindPlan, plan.TargetHeads==nil (empty map) implies resetAllChainsTo.None, making PlanConsistentWithLogs(plan, k) vacuously true for every k (the antecedent plan.resetAllChainsTo.Some? is false). The db.Clear() calls that precede this line are irrelevant because the predicate body is never evaluated in the None case. |
| `applyRewindPlan` (supernode/activity/interop/interop.go:900) | `Valid()` ✓ — applyRewindPlan's Dafny spec requires Valid() on entry. Before line 900, verifiedDB.Rewind retains entries k < rewindAtOrAfter and preserves VerifiedDB.Valid(). PruneDeniedAtOrAfterTimestamp has no Interop.Valid() side-effects per its spec. db.Rewind(expectedHead) for selective chains — LogsDB.Rewind spec preserves FindSealedBlock(n) for all n <= expectedHead.number = plan.targetHeads[chainID].number, so AllLogsDBsConsistentWithChainData() is maintained for retained blocks; removed blocks can no longer trigger the forall. VerifiedHeadsAreHighestBlocksUpToTimestamp() is preserved because retained logsDB blocks had timestamps satisfying the predicate before the rewind (part of Valid() pre-call) and no new blocks are added.; `ValidRewindPlan(plan)` ✓ — Same reasoning as line 876; sortedChainIDs is a derived slice and does not alter plan or the constants ValidRewindPlan reads.; `PlanConsistentWithVerified(plan)` ✓ — For the Some(ts) case with ts = rewindAtOrAfter-1, VerifiedDB.Rewind(rewindAtOrAfter) postcondition retains all k < rewindAtOrAfter, so ts is still in the DB. PlanConsistentWithVerified requires ts to be the max key below rewindAtOrAfter (established by the precondition), which holds after the rewind since no new entries are introduced. plan.targetHeads == verifiedDB.Get(ts).l2Heads is unchanged. The subsequent logsDB operations do not modify verifiedDB.; `RewoundVerifiedDB(plan)` ✓ — pendingTransition is preserved by VerifiedDB.Rewind. For the Some(ts) case with ts = rewindAtOrAfter-1, after verifiedDB.Rewind(rewindAtOrAfter) the DB contains exactly {k : k < rewindAtOrAfter}; its MaxKey is ts (because PlanConsistentWithVerified guaranteed ts was the max key below rewindAtOrAfter), so verifiedDB.LastTimestamp()==Some(ts). plan.targetHeads == verifiedDB.Get(ts).l2Heads is unchanged. logsDB operations do not touch verifiedDB.; `AllVerifiedCrossValid()` ✓ — AllVerifiedCrossValid() was established as a precondition of applyRewindPlan. After verifiedDB.Rewind, only entries with ts < rewindAtOrAfter remain; each was already cross-valid by the precondition. db.Rewind(expectedHead) preserves all blocks with number <= plan.targetHeads[chainID].number per LogsDB.Rewind spec. Since verified l2Heads at each remaining timestamp have block numbers <= plan.targetHeads[chainID].number (l2Heads are non-decreasing in the verifiedDB by Valid()), all InitMsgInLogsDB witnesses for those timestamps are preserved. Cross-validity therefore holds for all retained timestamps.; `forall k :: k in chainIDs <==> k in chains.Keys` ✓ — The same sortedChainIDs slice is reused; the construction guarantee is identical and no intermediate step modifies i.chains keys.; `forall k :: k in chainIDs ==> PlanConsistentWithLogs(plan, k)` ✓ — applyRewindPlan's precondition plan.resetAllChainsTo.Some? ==> PlanConsistentWithAllLogs(plan) establishes FindSealedBlock(plan.targetHeads[chainID].number).Some? for every chain at the start. For chains where db.Rewind(expectedHead) is called (expectedHead == plan.TargetHeads[chainID]), LogsDB.Rewind spec preserves FindSealedBlock(n) for all n <= expectedHead.number, so PlanConsistentWithLogs is maintained. For chains skipped because latestBlock==expectedHead, the logsDB is unchanged and the condition trivially holds. The case latestBlock.Number < expectedHead.Number cannot occur under valid preconditions: it would require FindSealedBlock(expectedHead.number)==None, contradicting PlanConsistentWithAllLogs. |

---

## 7. `resolveFrontierVerificationView` → `Interop.ResolveFrontierVerificationView`

**Go file**: `supernode/activity/interop/verification_view.go:29`  
**Dafny**: `dafny-models/Interop.dfy` — class `Interop`, method `ResolveFrontierVerificationView`  
**Notes**: The Dafny method is an axiom stub (ensures {:axiom} IsCorrectFrontierView). In Go, `resolveFrontierVerificationView` fetches block receipts for each chain in `blocksAtTS` via `chain.FetchReceipts`, extracts executing messages using `buildFrontierBlockView`, and builds a lookup map keyed by `frontierQueryKey`. Chains absent from `i.chains` are silently skipped in Go, while the Dafny precondition requires `blocksAtTS.Keys == CHAIN_IDS`.  

### Preconditions

- [x] `Valid()` — **Satisfied**
- [x] `blocksAtTS.Keys == CHAIN_IDS` — **Satisfied**
- [x] `BlocksExistedOnChain(blocksAtTS)` — **Satisfied**

### Postconditions

- [x] `{:axiom} Valid()` — **Satisfied**
- [x] `{:axiom} fresh(view)` — **Satisfied**
- [?] `{:axiom} IsCorrectFrontierView(view, blocksAtTS)` — **Uncertain** — Two gaps prevent confirmation. (1) FetchReceipts failure: the Dafny FetchReceipts spec guarantees only result.Some? ==> BlockInfo(blockID).Some?, not the converse; FetchReceipts can return None even when BlocksExistedOnChain(blocksAtTS) holds. In that case the Go function returns (nil, error) and produces no view, while the Dafny spec has no error return and unconditionally ensures a correctly-populated FrontierView — the postcondition cannot be discharged for that execution path. (2) buildFrontierBlockView opacity: in the success path, buildFrontierBlockView(chainID, blockInfo, receipts) at line ~42 must populate view.blocks[chainID] so that the abstract view.BlockInfo(chainID) and view.BlockLogs(chainID) ghost functions equal chains[chainID].BlockInfo(blocksAtTS[chainID]).value and chains[chainID].BlockLogs(blocksAtTS[chainID]).value respectively; the implementation of buildFrontierBlockView is not in scope, so this mapping cannot be confirmed. Note: the silent-skip branch (if !ok { continue }) is unreachable when preconditions hold because Valid() guarantees chains.Keys == CHAIN_IDS == blocksAtTS.Keys, so that branch is not a source of violation.

### Callsite Analysis

| Caller | Precondition Verdicts |
|---|---|
| `verify` (supernode/activity/interop/interop.go:601) | `Valid()` ✓ — Interop.Verify lists Valid() as its first requires clause. verify is the Go implementation of Verify; it passes blocksAtTS unchanged to resolveFrontierVerificationView without any intervening mutations, so Valid() holds at the call site.; `blocksAtTS.Keys == CHAIN_IDS` ✓ — Interop.Verify requires blocksAtTS.Keys == CHAIN_IDS. verify at line 601 forwards the same blocksAtTS map directly to resolveFrontierVerificationView without modification, so the key-set equality holds at the call site.; `BlocksExistedOnChain(blocksAtTS)` ✓ — Interop.Verify requires BlocksExistedOnChain(blocksAtTS). verify passes the same blocksAtTS to resolveFrontierVerificationView at line 601 without any modification, so BlocksExistedOnChain holds for every chainID in the map. |

### Violation Scenarios

**Postcondition `{:axiom} IsCorrectFrontierView(view, blocksAtTS)`** (uncertain):

Two gaps prevent confirmation. (1) FetchReceipts failure: the Dafny FetchReceipts spec guarantees only result.Some? ==> BlockInfo(blockID).Some?, not the converse; FetchReceipts can return None even when BlocksExistedOnChain(blocksAtTS) holds. In that case the Go function returns (nil, error) and produces no view, while the Dafny spec has no error return and unconditionally ensures a correctly-populated FrontierView — the postcondition cannot be discharged for that execution path. (2) buildFrontierBlockView opacity: in the success path, buildFrontierBlockView(chainID, blockInfo, receipts) at line ~42 must populate view.blocks[chainID] so that the abstract view.BlockInfo(chainID) and view.BlockLogs(chainID) ghost functions equal chains[chainID].BlockInfo(blocksAtTS[chainID]).value and chains[chainID].BlockLogs(blocksAtTS[chainID]).value respectively; the implementation of buildFrontierBlockView is not in scope, so this mapping cannot be confirmed. Note: the silent-skip branch (if !ok { continue }) is unreachable when preconditions hold because Valid() guarantees chains.Keys == CHAIN_IDS == blocksAtTS.Keys, so that branch is not a source of violation.

---

## 8. `checkChainsReady` → `Interop.CheckChainsReady`

**Go file**: `supernode/activity/interop/interop.go:941`  
**Dafny**: `dafny-models/Interop.dfy` — class `Interop`, method `CheckChainsReady`  
**Notes**: In Go, all chains are queried concurrently via goroutines, with results collected through a buffered channel before returning; the Dafny model uses a sequential for loop over chainIDs. Go returns an error (wrapping ethereum.NotFound) when any chain is not ready, while the Dafny model returns None for the optional result type.  

### Preconditions

- [?] `Valid()` — **Uncertain** — In the uninitialized branch of observeRound, firstVerifiableTimestamp() is called before checkChainsReady. Because firstVerifiableTimestamp is not in scope (no Dafny spec provided), any modification it makes to the Interop struct could break Valid() before line 558 is reached.
- [?] `AllDBsInSync()` — **Uncertain** — Same as Valid(): firstVerifiableTimestamp() is not covered by any in-scope Dafny spec, so its effect on AllDBsInSync() cannot be confirmed. A modification to verifiedDB or any logsDB inside that function would break AllDBsInSync() before line 558.
- [?] `forall k :: k in logsDBs && logsDBs[k].LatestSealedBlock().None? ==> ts == activationTimestamp` — **Uncertain** — When the verifiedDB is empty (initialized=false), all logsDBs have LatestSealedBlock()==None. obs.NextTimestamp is assigned from firstVerifiableTimestamp(), which is not mapped to any in-scope Dafny spec. If firstVerifiableTimestamp() returns a value other than activationTimestamp (e.g. due to some offset or configuration), the precondition forall k :: LatestSealedBlock(k)==None ==> ts==activationTimestamp is violated.

### Postconditions

- [x] `Valid()` — **Satisfied**
- [x] `result.Some? ==> result.value.blocks.Keys == chains.Keys` — **Satisfied**
- [?] `result.Some? ==> AdvancesAllLogsDBs(ts, result.value.blocks)` — **Uncertain** — AdvancesLogsDB requires logsDBs[chainID].LatestSealedBlock().value.number <= newBlock.number <= logsDBs[chainID].LatestSealedBlock().value.number + 1 for the Some case. The Dafny spec for OptimisticAt only guarantees BlockInfo(l2Block).Some? and BlockInfo(l2Block).value.timestamp <= ts; it provides no bound on l2Block.number relative to the logsDB's latest sealed block number. If OptimisticAt returns a block whose number is more than one step ahead of or behind the logsDB tip, AdvancesLogsDB is violated and the postcondition fails.
- [x] `result.Some? ==> BlocksExistedOnChain(result.value.blocks)` — **Satisfied**
- [x] `result.Some? ==> FrontierBlocksConsistentWithTimestamp(ts, result.value.blocks)` — **Satisfied**

### Callsite Analysis

| Caller | Precondition Verdicts |
|---|---|
| `observeRound` (supernode/activity/interop/interop.go:558) | `Valid()` ? — ObserveRound's spec requires Valid() at entry. Before line 558, the function calls verifiedDB.LastTimestamp() and verifiedDB.Get(lastTS), both frame-safe (no modifies clause), so they cannot break Valid(). However, in the uninitialized branch (verifiedDB empty), firstVerifiableTimestamp() is called and is not an in-scope mapped function; its modifies footprint and effect on Interop state are unknown. If it mutates verifiedDB, logsDBs, or Interop fields, Valid() may no longer hold at line 558.; `AllDBsInSync()` ? — ObserveRound's spec requires AllDBsInSync() at entry. Frame-safe verifiedDB calls before line 558 cannot break AllDBsInSync(). But firstVerifiableTimestamp() in the uninitialized branch is not in scope; if it advances or alters the verifiedDB or logsDBs independently, AllDBsInSync() may be violated before the call to checkChainsReady at line 558.; `forall k :: k in logsDBs && logsDBs[k].LatestSealedBlock().None? ==> ts == activationTimestamp` ? — When initialized=true: verifiedDB.LastTimestamp().Some? and AllDBsInSync() together imply logsDBs[k].LatestSealedBlock() == Some(...) for all k, so the antecedent is universally false and the condition holds vacuously. When initialized=false: AllDBsInSync() with verifiedDB.LastTimestamp()==None implies all logsDBs have LatestSealedBlock()==None, making the antecedent true for every k. ts is then set to the return value of firstVerifiableTimestamp(), which must equal activationTimestamp for the condition to hold. firstVerifiableTimestamp() is not in scope; it is not confirmed to return activationTimestamp. |

### Violation Scenarios

**Precondition `Valid()`** (uncertain):

In the uninitialized branch of observeRound, firstVerifiableTimestamp() is called before checkChainsReady. Because firstVerifiableTimestamp is not in scope (no Dafny spec provided), any modification it makes to the Interop struct could break Valid() before line 558 is reached.

**Precondition `AllDBsInSync()`** (uncertain):

Same as Valid(): firstVerifiableTimestamp() is not covered by any in-scope Dafny spec, so its effect on AllDBsInSync() cannot be confirmed. A modification to verifiedDB or any logsDB inside that function would break AllDBsInSync() before line 558.

**Precondition `forall k :: k in logsDBs && logsDBs[k].LatestSealedBlock().None? ==> ts == activationTimestamp`** (uncertain):

When the verifiedDB is empty (initialized=false), all logsDBs have LatestSealedBlock()==None. obs.NextTimestamp is assigned from firstVerifiableTimestamp(), which is not mapped to any in-scope Dafny spec. If firstVerifiableTimestamp() returns a value other than activationTimestamp (e.g. due to some offset or configuration), the precondition forall k :: LatestSealedBlock(k)==None ==> ts==activationTimestamp is violated.

**Postcondition `result.Some? ==> AdvancesAllLogsDBs(ts, result.value.blocks)`** (uncertain):

AdvancesLogsDB requires logsDBs[chainID].LatestSealedBlock().value.number <= newBlock.number <= logsDBs[chainID].LatestSealedBlock().value.number + 1 for the Some case. The Dafny spec for OptimisticAt only guarantees BlockInfo(l2Block).Some? and BlockInfo(l2Block).value.timestamp <= ts; it provides no bound on l2Block.number relative to the logsDB's latest sealed block number. If OptimisticAt returns a block whose number is more than one step ahead of or behind the logsDB tip, AdvancesLogsDB is violated and the postcondition fails.

---

## 9. `buildPendingTransition` → `Interop.BuildPendingTransition`

**Go file**: `supernode/activity/interop/interop.go:651`  
**Dafny**: `dafny-models/Interop.dfy` — class `Interop`, method `BuildPendingTransition`  
**Notes**: Go uses firstVerifiableTimestamp (which may exceed activationTimestamp after a cold start with late-joining chains) as the lower bound in buildRewindPlan, while the Dafny model always uses activationTimestamp. Go also sets resetAllChainsTo based on a per-chain deny-list check (HasDeniedAtOrAfterTimestamp); the Dafny model treats this field as an opaque decision with no link to a deny-list predicate.  

### Preconditions

- [x] `Valid()` — **Satisfied**
- [x] `!output.WaitOutput?` — **Satisfied**
- [x] `ValidStepOutput(output, obs)` — **Satisfied**
- [x] `OutputConsistentWithVerified(output, obs)` — **Satisfied**
- [x] `OutputConsistentWithLogs(output, obs)` — **Satisfied**
- [x] `OutputConsistentWithChainState(output, obs)` — **Satisfied**

### Postconditions

- [ ] `ValidPendingTransition(pendingTx)` — **Violated** — In the DecisionRewind branch, buildRewindPlan can produce plans that fail ValidRewindPlan, which is a conjunct of ValidPendingTransition. Two concrete paths: (1) shouldResetEnginesOnRewind returns false and *obs.LastVerifiedTS > activationTimestamp — plan.ResetAllChainsTo is nil (Dafny None), but ValidRewindPlan's None branch requires rewindAtOrAfter <= ACTIVATION_TIMESTAMP; since rewindAtOrAfter = lastTS > activationTimestamp, the condition fails. (2) shouldResetEnginesOnRewind returns true and lastTS <= firstVerifiableTimestamp — buildRewindPlan sets plan.ResetAllChainsTo = Some(lastTS−1) but returns before setting plan.TargetHeads (empty map); ValidRewindPlan's Some branch requires targetHeads.Keys == CHAIN_IDS, which an empty map cannot satisfy. The Advance and Invalidate branches are unaffected: ValidStepOutput (precondition 3) directly supplies |invalidHeads|==0 for Advance, 0<|invalidHeads| for Invalidate, and l2Heads.Keys==CHAIN_IDS for both.
- [ ] `TransitionConsistentWithVerified(pendingTx)` — **Violated** — Advance maps to AdvancesVerifiedDB, which is guaranteed by precondition OutputConsistentWithVerified (AdvanceOutput case). Invalidate is trivially true. For Rewind, the obligation is PlanConsistentWithVerified, which fires only when resetAllChainsTo.Some?. When shouldResetEnginesOnRewind=true and lastTS <= firstVerifiableTimestamp: plan.ResetAllChainsTo = Some(lastTS−1) but plan.TargetHeads is empty; PlanConsistentWithVerified requires plan.targetHeads == verifiedDB.Get(lastTS−1).l2Heads, but verifiedDB.Get returns a map with Keys==CHAIN_IDS (from Valid()) while plan.TargetHeads is empty. Additionally, if lastTS == activationTimestamp, verifiedDB.Has(lastTS−1) = verifiedDB.Has(activationTimestamp−1) is false because Valid() guarantees all db entries >= activationTimestamp, again violating PlanConsistentWithVerified.
- [ ] `TransitionConsistentWithLogs(pendingTx)` — **Violated** — Advance requires AdvancesAllLogsDBs, which is established by precondition OutputConsistentWithLogs (AdvanceOutput case), and newL2Heads.Keys==logsDBs.Keys follows from ValidStepOutput and Valid(). Invalidate is trivially true. For Rewind, TransitionConsistentWithLogs requires: for every k in logsDBs.Keys, if resetAllChainsTo.Some? then k in pending.rewind.value.targetHeads and PlanConsistentWithLogs holds. When shouldResetEnginesOnRewind=true and lastTS <= firstVerifiableTimestamp, buildRewindPlan returns with plan.TargetHeads empty; the antecedent resetAllChainsTo.Some? is true (plan.ResetAllChainsTo = Some(lastTS−1)), so k in targetHeads is required for every chain k in CHAIN_IDS, but an empty map satisfies this for no chain — violated for all CHAIN_IDS elements. When shouldResetEnginesOnRewind=true and lastTS > firstVerifiableTimestamp, plan.TargetHeads = prevResult.L2Heads whose keys are CHAIN_IDS (from Valid()), and PlanConsistentWithLogs follows from OutputConsistentWithLogs (RewindOutput with activationTimestamp < lastTS).
- [x] `output.AdvanceOutput? <==> pendingTx.decision.Advance?` — **Satisfied**
- [x] `output.InvalidateOutput? <==> pendingTx.decision.Invalidate?` — **Satisfied**
- [x] `(pendingTx.decision.Advance? || pendingTx.decision.Invalidate?) ==> pendingTx.result.value == output.result` — **Satisfied**

### Callsite Analysis

| Caller | Precondition Verdicts |
|---|---|
| `progressAndRecord` (supernode/activity/interop/interop.go:483) | `Valid()` ✓ — ProgressAndRecord's own Dafny spec requires Valid() as a precondition. ProgressInterop (called immediately before buildPendingTransition) ensures Valid() in its postcondition, so Valid() is re-established at the callsite.; `!output.WaitOutput?` ✓ — Before reaching the buildPendingTransition call, progressAndRecord returns early when output.Decision == DecisionWait (the Go guard 'if output.Decision == DecisionWait { return i.refreshCurrentL1OnWait() }'). DecisionWait is therefore excluded from any call to buildPendingTransition.; `ValidStepOutput(output, obs)` ✓ — ProgressInterop's Dafny spec ensures ValidStepOutput(output, obs). This postcondition is established by the call to progressInterop immediately before buildPendingTransition is invoked in progressAndRecord.; `OutputConsistentWithVerified(output, obs)` ✓ — ProgressInterop's Dafny spec ensures OutputConsistentWithVerified(output, obs). This postcondition is established by the call to progressInterop immediately before buildPendingTransition is invoked.; `OutputConsistentWithLogs(output, obs)` ✓ — ProgressInterop's Dafny spec ensures OutputConsistentWithLogs(output, obs). This postcondition is established by the call to progressInterop immediately before buildPendingTransition is invoked.; `OutputConsistentWithChainState(output, obs)` ✓ — ProgressInterop's Dafny spec ensures OutputConsistentWithChainState(output, obs). This postcondition is established by the call to progressInterop immediately before buildPendingTransition is invoked. |

### Violation Scenarios

**Postcondition `ValidPendingTransition(pendingTx)`** (violated):

In the DecisionRewind branch, buildRewindPlan can produce plans that fail ValidRewindPlan, which is a conjunct of ValidPendingTransition. Two concrete paths: (1) shouldResetEnginesOnRewind returns false and *obs.LastVerifiedTS > activationTimestamp — plan.ResetAllChainsTo is nil (Dafny None), but ValidRewindPlan's None branch requires rewindAtOrAfter <= ACTIVATION_TIMESTAMP; since rewindAtOrAfter = lastTS > activationTimestamp, the condition fails. (2) shouldResetEnginesOnRewind returns true and lastTS <= firstVerifiableTimestamp — buildRewindPlan sets plan.ResetAllChainsTo = Some(lastTS−1) but returns before setting plan.TargetHeads (empty map); ValidRewindPlan's Some branch requires targetHeads.Keys == CHAIN_IDS, which an empty map cannot satisfy. The Advance and Invalidate branches are unaffected: ValidStepOutput (precondition 3) directly supplies |invalidHeads|==0 for Advance, 0<|invalidHeads| for Invalidate, and l2Heads.Keys==CHAIN_IDS for both.

**Postcondition `TransitionConsistentWithVerified(pendingTx)`** (violated):

Advance maps to AdvancesVerifiedDB, which is guaranteed by precondition OutputConsistentWithVerified (AdvanceOutput case). Invalidate is trivially true. For Rewind, the obligation is PlanConsistentWithVerified, which fires only when resetAllChainsTo.Some?. When shouldResetEnginesOnRewind=true and lastTS <= firstVerifiableTimestamp: plan.ResetAllChainsTo = Some(lastTS−1) but plan.TargetHeads is empty; PlanConsistentWithVerified requires plan.targetHeads == verifiedDB.Get(lastTS−1).l2Heads, but verifiedDB.Get returns a map with Keys==CHAIN_IDS (from Valid()) while plan.TargetHeads is empty. Additionally, if lastTS == activationTimestamp, verifiedDB.Has(lastTS−1) = verifiedDB.Has(activationTimestamp−1) is false because Valid() guarantees all db entries >= activationTimestamp, again violating PlanConsistentWithVerified.

**Postcondition `TransitionConsistentWithLogs(pendingTx)`** (violated):

Advance requires AdvancesAllLogsDBs, which is established by precondition OutputConsistentWithLogs (AdvanceOutput case), and newL2Heads.Keys==logsDBs.Keys follows from ValidStepOutput and Valid(). Invalidate is trivially true. For Rewind, TransitionConsistentWithLogs requires: for every k in logsDBs.Keys, if resetAllChainsTo.Some? then k in pending.rewind.value.targetHeads and PlanConsistentWithLogs holds. When shouldResetEnginesOnRewind=true and lastTS <= firstVerifiableTimestamp, buildRewindPlan returns with plan.TargetHeads empty; the antecedent resetAllChainsTo.Some? is true (plan.ResetAllChainsTo = Some(lastTS−1)), so k in targetHeads is required for every chain k in CHAIN_IDS, but an empty map satisfies this for no chain — violated for all CHAIN_IDS elements. When shouldResetEnginesOnRewind=true and lastTS > firstVerifiableTimestamp, plan.TargetHeads = prevResult.L2Heads whose keys are CHAIN_IDS (from Valid()), and PlanConsistentWithLogs follows from OutputConsistentWithLogs (RewindOutput with activationTimestamp < lastTS).

---

## 10. `verifyInteropMessages` → `Interop.VerifyMessages`

**Go file**: `supernode/activity/interop/algo.go:69`  
**Dafny**: `dafny-models/Interop.dfy` — class `Interop`, method `VerifyMessages`  
**Notes**: In Go, executing messages in the first sealed block are silently skipped because no predecessor frontier entry exists for that block. A chain is also silently skipped if it is absent from i.logsDBs. Additionally, Go breaks on the first invalid executing message per block rather than checking all messages before returning a verdict.  

### Preconditions

- [?] `Valid()` — **Uncertain** — No in-scope callers are available; it cannot be determined from the code whether the Interop struct invariants (logsDBs.Keys == CHAIN_IDS, verifiedDB.Valid(), AllLogsDBsConsistentWithChainData, chains.Keys == CHAIN_IDS, etc.) always hold at every call site.
- [?] `blocksAtTS.Keys == CHAIN_IDS` — **Uncertain** — No in-scope callers are available; it cannot be determined whether the blocksAtTimestamp argument always covers exactly CHAIN_IDS at every call site.
- [?] `AdvancesAllLogsDBs(ts, blocksAtTS)` — **Uncertain** — No in-scope callers are available; it cannot be determined whether each chain's logsDB is at the correct tip (LatestSealedBlock().number <= blocksAtTS[chainID].number <= LatestSealedBlock().number + 1, or None and ts == activationTimestamp) before every call.
- [?] `IsCorrectFrontierView(view, blocksAtTS)` — **Uncertain** — No in-scope callers are available; it cannot be determined whether the frontier view always carries view.BlockInfo(chainID) == chains[chainID].BlockInfo(blocksAtTS[chainID]).value and matching BlockLogs for every chain in CHAIN_IDS at call time.

### Postconditions

- [?] `Valid()` — **Uncertain** — i.newInvalidHead is invoked in three branches (first-block hash-mismatch, block hash-mismatch, and invalid-exec-msgs) but has no Dafny spec; its frame safety is not established. All other called functions (l1Inclusion, db.OpenBlock, db.FirstSealedBlock, verifyExecutingMessage) carry 'modifies nothing' in their specs and cannot disturb Valid(). If newInvalidHead writes to verifiedDB, logsDBs, or chains, the invariants constituting Valid() could be broken.
- [x] `result.timestamp == ts` — **Satisfied**
- [x] `result.l2Heads == blocksAtTS` — **Satisfied**
- [ ] `|result.invalidHeads| == 0 ==> forall chainID in CHAIN_IDS :: BlockIsCrossValid(view.BlockInfo(chainID).timestamp, chainID, view.BlockInfo(chainID).id) && AllInitMsgsPresent(chainID, view.BlockInfo(chainID).id, blocksAtTS)` — **Violated** — When view.block(chainID) returns false for some chain C and db.OpenBlock(expectedBlock.Number) returns ErrSkipped because C's block is the first sealed entry in its logsDB, the code calls db.FirstSealedBlock(), confirms the block number matches expectedBlock.Number, and executes 'continue' without iterating over execMsgs at all. If the hash also matches, no entry is added to result.InvalidHeads, so |result.invalidHeads| == 0. But any executing messages that exist in that first block are never passed to verifyExecutingMessage, so neither ValidExecutingMessage nor InitMsgPresent is established for them. BlockIsCrossValid (which quantifies over all execMsgs in chains[C].BlockLogs(blocksAtTS[C]).value) and AllInitMsgsPresent may therefore be false, violating the postcondition's consequent. The Dafny LogsDB axiom |BlockLogs(FirstSealedBlock().value.number).execMsgs| == 0 is a model assumption not enforced on the concrete Go logsDB, so the violation is reachable.

### Violation Scenarios

**Precondition `Valid()`** (uncertain):

No in-scope callers are available; it cannot be determined from the code whether the Interop struct invariants (logsDBs.Keys == CHAIN_IDS, verifiedDB.Valid(), AllLogsDBsConsistentWithChainData, chains.Keys == CHAIN_IDS, etc.) always hold at every call site.

**Precondition `blocksAtTS.Keys == CHAIN_IDS`** (uncertain):

No in-scope callers are available; it cannot be determined whether the blocksAtTimestamp argument always covers exactly CHAIN_IDS at every call site.

**Precondition `AdvancesAllLogsDBs(ts, blocksAtTS)`** (uncertain):

No in-scope callers are available; it cannot be determined whether each chain's logsDB is at the correct tip (LatestSealedBlock().number <= blocksAtTS[chainID].number <= LatestSealedBlock().number + 1, or None and ts == activationTimestamp) before every call.

**Precondition `IsCorrectFrontierView(view, blocksAtTS)`** (uncertain):

No in-scope callers are available; it cannot be determined whether the frontier view always carries view.BlockInfo(chainID) == chains[chainID].BlockInfo(blocksAtTS[chainID]).value and matching BlockLogs for every chain in CHAIN_IDS at call time.

**Postcondition `Valid()`** (uncertain):

i.newInvalidHead is invoked in three branches (first-block hash-mismatch, block hash-mismatch, and invalid-exec-msgs) but has no Dafny spec; its frame safety is not established. All other called functions (l1Inclusion, db.OpenBlock, db.FirstSealedBlock, verifyExecutingMessage) carry 'modifies nothing' in their specs and cannot disturb Valid(). If newInvalidHead writes to verifiedDB, logsDBs, or chains, the invariants constituting Valid() could be broken.

**Postcondition `|result.invalidHeads| == 0 ==> forall chainID in CHAIN_IDS :: BlockIsCrossValid(view.BlockInfo(chainID).timestamp, chainID, view.BlockInfo(chainID).id) && AllInitMsgsPresent(chainID, view.BlockInfo(chainID).id, blocksAtTS)`** (violated):

When view.block(chainID) returns false for some chain C and db.OpenBlock(expectedBlock.Number) returns ErrSkipped because C's block is the first sealed entry in its logsDB, the code calls db.FirstSealedBlock(), confirms the block number matches expectedBlock.Number, and executes 'continue' without iterating over execMsgs at all. If the hash also matches, no entry is added to result.InvalidHeads, so |result.invalidHeads| == 0. But any executing messages that exist in that first block are never passed to verifyExecutingMessage, so neither ValidExecutingMessage nor InitMsgPresent is established for them. BlockIsCrossValid (which quantifies over all execMsgs in chains[C].BlockLogs(blocksAtTS[C]).value) and AllInitMsgsPresent may therefore be false, violating the postcondition's consequent. The Dafny LogsDB axiom |BlockLogs(FirstSealedBlock().value.number).execMsgs| == 0 is a model assumption not enforced on the concrete Go logsDB, so the violation is reachable.

---

## 11. `applyRewindPlan` → `Interop.ApplyRewindPlan`

**Go file**: `supernode/activity/interop/interop.go:838`  
**Dafny**: `dafny-models/Interop.dfy` — class `Interop`, method `ApplyRewindPlan`  
**Notes**: In Go, resetChainEnginesIfNeeded is called only after all logsDB rewinds succeed; the Dafny model calls RewindChainEngines before the logsDB steps and treats engine-rewind success as independent of logsDB outcomes. If any logsDB operation fails in Go, the engine reset is skipped entirely, making the rewind sequence non-idempotent under partial failure.  

### Preconditions

- [x] `Valid()` — **Satisfied**
- [x] `AllLogsDBsConsistentWithChainData()` — **Satisfied**
- [x] `verifiedDB.pendingTransition.Some?` — **Satisfied**
- [x] `verifiedDB.pendingTransition.value.decision == Rewind` — **Satisfied**
- [x] `verifiedDB.pendingTransition.value.rewind == Some(plan)` — **Satisfied**
- [x] `ValidRewindPlan(plan)` — **Satisfied**
- [x] `PlanConsistentWithVerified(plan)` — **Satisfied**
- [x] `plan.resetAllChainsTo.Some? ==> PlanConsistentWithAllLogs(plan)` — **Satisfied**
- [x] `plan.resetAllChainsTo.Some? ==> AllDBsInSyncUpTo(plan.resetAllChainsTo.value)` — **Satisfied**
- [x] `AllVerifiedCrossValid()` — **Satisfied**

### Postconditions

- [?] `Valid()` — **Uncertain** — If LogsDB.Clear() or LogsDB.Rewind() fails in Go and leaves the logsDB in a partially-modified state, AllLogsDBsConsistentWithChainData() — a component of Valid() — may not hold. The Dafny LogsDB model never fails; the Go implementation can. All other components of Valid() (verifiedDB.Valid() via VerifiedDB.Rewind's ensures, VerifiedHeadsAreHighestBlocksUpToTimestamp() because remaining logsDB blocks at n > old verifiedHead for any remaining ts have timestamps >= rewindAtOrAfter > ts, AllVerifiedHeadsBoundedByTimestamp() because l2Heads and BlockInfo are unchanged) are maintained even under partial failure.
- [?] `AllLogsDBsConsistentWithChainData()` — **Uncertain** — LogsDB.Clear() and LogsDB.Rewind() are modeled in Dafny as always-succeeding; their ensures clauses then guarantee consistency. In Go both can return errors. If a failed Clear() or Rewind() atomically leaves the logsDB unchanged, the precondition guarantees AllLogsDBsConsistentWithChainData() still holds. If a failed operation partially modifies the logsDB's internal index in a way that makes FindSealedBlock return data inconsistent with chain data, the predicate can be violated. This cannot be determined from the Go source alone.
- [x] `ValidRewindPlan(plan)` — **Satisfied**
- [x] `verifiedDB.pendingTransition == old(verifiedDB.pendingTransition)` — **Satisfied**
- [x] `PlanConsistentWithVerified(plan)` — **Satisfied**
- [x] `plan.resetAllChainsTo.Some? ==> PlanConsistentWithAllLogs(plan)` — **Satisfied**
- [x] `plan.resetAllChainsTo.Some? && success ==> RewoundAllLogsDB(plan)` — **Satisfied**
- [x] `plan.resetAllChainsTo.Some? ==> AllDBsInSyncUpTo(plan.resetAllChainsTo.value)` — **Satisfied**
- [x] `success ==> AllDBsInSync()` — **Satisfied**
- [x] `AllVerifiedCrossValid()` — **Satisfied**

### Callsite Analysis

| Caller | Precondition Verdicts |
|---|---|
| `applyPendingTransition` (supernode/activity/interop/interop.go:682) | `Valid()` ✓ — applyPendingTransition's Dafny spec lists Valid() as a requires clause, so Valid() is established before applyRewindPlan is called.; `AllLogsDBsConsistentWithChainData()` ✓ — applyPendingTransition's Dafny spec explicitly requires AllLogsDBsConsistentWithChainData(), so it holds at the call site.; `verifiedDB.pendingTransition.Some?` ✓ — applyPendingTransition's spec requires verifiedDB.GetPendingTransition() == Some(pending), guaranteeing pendingTransition.Some? before the call.; `verifiedDB.pendingTransition.value.decision == Rewind` ✓ — applyRewindPlan is only reached inside case DecisionRewind at line 682, so the decision is Rewind at every call site.; `verifiedDB.pendingTransition.value.rewind == Some(plan)` ✓ — ValidPendingTransition (implied by Valid()) guarantees rewind.Some? when decision == Rewind. The call passes *pending.Rewind, so plan equals pending.rewind.value. The nil guard at line ~671 ensures pending.Rewind != nil before the dereference.; `ValidRewindPlan(plan)` ✓ — ValidPendingTransition (implied by Valid() + verifiedDB.pendingTransition.Some?) requires ValidRewindPlan(pending.rewind.value). plan == pending.rewind.value, so this holds.; `PlanConsistentWithVerified(plan)` ✓ — applyPendingTransition's spec requires PendingTransitionIsConsistent(). For the Rewind decision, PendingTransitionIsConsistent() unfolds to TransitionConsistentWithVerified(pending), which for Rewind equals PlanConsistentWithVerified(pending.rewind.value) == PlanConsistentWithVerified(plan).; `plan.resetAllChainsTo.Some? ==> PlanConsistentWithAllLogs(plan)` ✓ — PendingTransitionIsConsistent() for the Rewind case also expands to TransitionConsistentWithLogs(pending), which for Rewind reads: forall k in logsDBs.Keys && resetAllChainsTo.Some? ==> k in targetHeads && PlanConsistentWithLogs(plan, k). This is exactly PlanConsistentWithAllLogs(plan) under the Some? antecedent.; `plan.resetAllChainsTo.Some? ==> AllDBsInSyncUpTo(plan.resetAllChainsTo.value)` ✓ — PendingTransitionIsConsistent() for the Rewind case explicitly includes: p.rewind.value.resetAllChainsTo.Some? ==> AllDBsInSyncUpTo(p.rewind.value.resetAllChainsTo.value). This is exactly the precondition.; `AllVerifiedCrossValid()` ✓ — applyPendingTransition's Dafny spec requires AllVerifiedCrossValid() before the call. |

### Violation Scenarios

**Postcondition `Valid()`** (uncertain):

If LogsDB.Clear() or LogsDB.Rewind() fails in Go and leaves the logsDB in a partially-modified state, AllLogsDBsConsistentWithChainData() — a component of Valid() — may not hold. The Dafny LogsDB model never fails; the Go implementation can. All other components of Valid() (verifiedDB.Valid() via VerifiedDB.Rewind's ensures, VerifiedHeadsAreHighestBlocksUpToTimestamp() because remaining logsDB blocks at n > old verifiedHead for any remaining ts have timestamps >= rewindAtOrAfter > ts, AllVerifiedHeadsBoundedByTimestamp() because l2Heads and BlockInfo are unchanged) are maintained even under partial failure.

**Postcondition `AllLogsDBsConsistentWithChainData()`** (uncertain):

LogsDB.Clear() and LogsDB.Rewind() are modeled in Dafny as always-succeeding; their ensures clauses then guarantee consistency. In Go both can return errors. If a failed Clear() or Rewind() atomically leaves the logsDB unchanged, the precondition guarantees AllLogsDBsConsistentWithChainData() still holds. If a failed operation partially modifies the logsDB's internal index in a way that makes FindSealedBlock return data inconsistent with chain data, the predicate can be violated. This cannot be determined from the Go source alone.

---

## 12. `verify` → `Interop.Verify`

**Go file**: `supernode/activity/interop/interop.go:600`  
**Dafny**: `dafny-models/Interop.dfy` — class `Interop`, method `Verify`  
**Notes**: In Go, `verify` delegates to injected closures (`verifyFn` defaulting to `verifyInteropMessages`, and `cycleVerifyFn`) rather than calling those functions directly; this indirection supports test overrides. The Dafny model inlines the three sub-operations (ResolveFrontierVerificationView, VerifyMessages, VerifyCycles) directly. Invalid heads from the cycle check are merged into the message-verification result in both models.  

### Preconditions

- [x] `Valid()` — **Satisfied**
- [?] `blocksAtTS.Keys == CHAIN_IDS` — **Uncertain** — If checkPreconditions returns nil while obs.chainsReady is false (e.g., a code path that bypasses the chainsReady check), obs.BlocksAtTS.Keys would not equal CHAIN_IDS, violating this precondition. Not provable safe from the listed Dafny specs alone.
- [x] `BlocksExistedOnChain(blocksAtTS)` — **Satisfied**
- [?] `AdvancesAllLogsDBs(ts, blocksAtTS)` — **Uncertain** — If checkPreconditions returns nil when obs.l2sConsistent or obs.l1Consistent is false (while obs.chainsReady is true), AdvancesAllLogsDBs(obs.nextTimestamp, obs.blocksAtTS) is not guaranteed by any listed spec, violating this precondition.

### Postconditions

- [x] `Valid()` — **Satisfied**
- [x] `result.timestamp == ts` — **Satisfied**
- [x] `result.l2Heads == blocksAtTS` — **Satisfied**
- [x] `|result.invalidHeads| == 0 ==> ResultIsCrossValid(result)` — **Satisfied**

### Callsite Analysis

| Caller | Precondition Verdicts |
|---|---|
| `progressInterop` (supernode/activity/interop/interop.go:522) | `Valid()` ✓ — ProgressInterop (Dafny) carries Valid() as a precondition. observeRound has `requires Valid()` and `ensures Valid()` (modifies only chains.Values). checkPreconditions is not in the Dafny spec list, but from the Go code it accepts obs by value and returns a *StepOutput without touching Interop fields. Therefore Valid() is unaffected between the observeRound return and the verify call at line 522.; `blocksAtTS.Keys == CHAIN_IDS` ? — obs.BlocksAtTS is passed as blocksAtTS. observeRound ensures ValidRoundObservation(obs) (`obs.chainsReady ==> obs.blocksAtTS.Keys == CHAIN_IDS`) and ObservationConsistentWithChainState(obs) (`0 < |obs.blocksAtTS| ==> obs.blocksAtTS.Keys == CHAIN_IDS`). Neither guarantee unconditionally holds: if obs.chainsReady is false and obs.blocksAtTS is empty, both conditionals are vacuously satisfied but blocksAtTS.Keys != CHAIN_IDS (when CHAIN_IDS is non-empty). checkPreconditions presumably guards on obs.chainsReady, but it has no Dafny spec in the listed functions, so its postcondition cannot be treated as an established fact.; `BlocksExistedOnChain(blocksAtTS)` ✓ — observeRound ensures ObservationConsistentWithChainState(obs): `0 < |obs.blocksAtTS| ==> obs.blocksAtTS.Keys == CHAIN_IDS && BlocksExistedOnChain(obs.blocksAtTS)`. When obs.blocksAtTS is empty the predicate `BlocksExistedOnChain(blocksAtTS)` is vacuously true (universal quantification over an empty domain). So in every case established by observeRound's postconditions, BlocksExistedOnChain(obs.BlocksAtTS) holds. checkPreconditions does not modify obs, so the property is preserved at the verify call site.; `AdvancesAllLogsDBs(ts, blocksAtTS)` ? — observeRound ensures ObservationConsistentWithLogs(obs): `obs.chainsReady && obs.l2sConsistent && obs.l1Consistent ==> AdvancesAllLogsDBs(obs.nextTimestamp, obs.blocksAtTS)`. All three flags must be true for the guarantee to apply. checkPreconditions presumably enforces these flags before allowing the verify call to proceed, but checkPreconditions has no Dafny spec in the listed functions, so this conditional cannot be discharged from specs alone. |

### Violation Scenarios

**Precondition `blocksAtTS.Keys == CHAIN_IDS`** (uncertain):

If checkPreconditions returns nil while obs.chainsReady is false (e.g., a code path that bypasses the chainsReady check), obs.BlocksAtTS.Keys would not equal CHAIN_IDS, violating this precondition. Not provable safe from the listed Dafny specs alone.

**Precondition `AdvancesAllLogsDBs(ts, blocksAtTS)`** (uncertain):

If checkPreconditions returns nil when obs.l2sConsistent or obs.l1Consistent is false (while obs.chainsReady is true), AdvancesAllLogsDBs(obs.nextTimestamp, obs.blocksAtTS) is not guaranteed by any listed spec, violating this precondition.

---

## 13. `observeRound` → `Interop.ObserveRound`

**Go file**: `supernode/activity/interop/interop.go:532`  
**Dafny**: `dafny-models/Interop.dfy` — class `Interop`, method `ObserveRound`  
**Notes**: Go uses a two-phase L1 consistency check: it tracks both L1NeedsRewind (which triggers a rewind from the last-verified L1) and l1Consistent (whether all chains agree on L1). The Dafny model uses a single l1Consistent flag and does not model L1NeedsRewind as a distinct concept.  

### Preconditions

- [x] `Valid()` — **Satisfied**
- [x] `AllDBsInSync()` — **Satisfied**

### Postconditions

- [x] `Valid()` — **Satisfied**
- [x] `AllDBsInSync()` — **Satisfied**
- [ ] `ValidRoundObservation(obs)` — **Violated** — ValidRoundObservation requires (!obs.l1Consistent ==> obs.lastVerifiedTS.Some?). When verifiedDB is empty (initialized=false on the LastTimestamp call), obs.LastVerifiedTS stays nil and obs.LastVerified stays nil. If checkChainsReady then returns successfully (obs.ChainsReady=true, obs.BlocksAtTS and obs.L1Heads set), the guard 'if obs.LastVerified != nil' is false, so the L1NeedsRewind branch is skipped entirely. Execution falls through to the frontier-heads SameL1Chain call and sets obs.L1Consistent = same. If same=false, obs.L1Consistent=false while obs.LastVerifiedTS=nil, violating the implication. The same violation occurs on the NotFound early-return path: obs.L1Consistent defaults to false (Go zero value) while obs.LastVerifiedTS may be nil when the verifiedDB is uninitialized.
- [ ] `ObservationConsistentWithVerified(obs)` — **Violated** — ObservationConsistentWithVerified has ValidRoundObservation(obs) as a predicate precondition. The same scenario that violates POST3 (empty verifiedDB + frontier L1 check returns false) makes ValidRoundObservation(obs) false, so calling ObservationConsistentWithVerified(obs) is ill-typed and the ensures clause is violated. Additionally, the sub-condition obs.nextTimestamp == NextTimestamp() in the uninitialized branch (initialized=false) depends on firstVerifiableTimestamp() returning exactly activationTimestamp; this function carries no Dafny spec, so conformance cannot be confirmed from specs alone. In the initialized branch and under a valid ValidRoundObservation, the remaining sub-conditions hold: obs.LastVerifiedTS matches verifiedDB.lastTimestamp by direct assignment; the rewind sub-condition holds because Valid()/Sequential and AllDBsInSync() guarantee lastTS-1 is in verifiedDB.db when activationTimestamp < lastTS; and the AdvancesVerifiedDB sub-condition is vacuously true because obs.L2sConsistent is never set to true in observeRound (Go zero value is false).
- [ ] `ObservationConsistentWithLogs(obs)` — **Violated** — ObservationConsistentWithLogs has both ValidRoundObservation(obs) and ObservationConsistentWithVerified(obs) as predicate preconditions. The POST3-violation scenario (empty verifiedDB + frontier L1 check returns false) makes ValidRoundObservation(obs) false, so the ensures clause is violated. In scenarios where ValidRoundObservation does hold: the rewind sub-condition (!obs.l1Consistent && lastVerifiedTS.value > activationTimestamp ==> sealed blocks for lastTS-1 exist) is satisfied because AllDBsInSync() guarantees DBsInSyncUpTo through lastTS, which covers lastTS-1; the advance sub-condition (chainsReady && l2sConsistent && l1Consistent ==> AdvancesAllLogsDBs) is vacuously true because observeRound never sets obs.L2sConsistent=true (Go zero value is false, a structural divergence from the Dafny model).
- [x] `ObservationConsistentWithChainState(obs)` — **Satisfied**

### Callsite Analysis

| Caller | Precondition Verdicts |
|---|---|
| `progressInterop` (supernode/activity/interop/interop.go:513) | `Valid()` ✓ — The Dafny spec for progressInterop lists Valid() as a requires clause. observeRound is the first call in progressInterop (line 513), so Valid() is established at the call site with no intervening mutations.; `AllDBsInSync()` ✓ — The Dafny spec for progressInterop lists AllDBsInSync() as a requires clause. observeRound is the first call in progressInterop (line 513), so AllDBsInSync() is established at the call site with no intervening mutations. |

### Violation Scenarios

**Postcondition `ValidRoundObservation(obs)`** (violated):

ValidRoundObservation requires (!obs.l1Consistent ==> obs.lastVerifiedTS.Some?). When verifiedDB is empty (initialized=false on the LastTimestamp call), obs.LastVerifiedTS stays nil and obs.LastVerified stays nil. If checkChainsReady then returns successfully (obs.ChainsReady=true, obs.BlocksAtTS and obs.L1Heads set), the guard 'if obs.LastVerified != nil' is false, so the L1NeedsRewind branch is skipped entirely. Execution falls through to the frontier-heads SameL1Chain call and sets obs.L1Consistent = same. If same=false, obs.L1Consistent=false while obs.LastVerifiedTS=nil, violating the implication. The same violation occurs on the NotFound early-return path: obs.L1Consistent defaults to false (Go zero value) while obs.LastVerifiedTS may be nil when the verifiedDB is uninitialized.

**Postcondition `ObservationConsistentWithVerified(obs)`** (violated):

ObservationConsistentWithVerified has ValidRoundObservation(obs) as a predicate precondition. The same scenario that violates POST3 (empty verifiedDB + frontier L1 check returns false) makes ValidRoundObservation(obs) false, so calling ObservationConsistentWithVerified(obs) is ill-typed and the ensures clause is violated. Additionally, the sub-condition obs.nextTimestamp == NextTimestamp() in the uninitialized branch (initialized=false) depends on firstVerifiableTimestamp() returning exactly activationTimestamp; this function carries no Dafny spec, so conformance cannot be confirmed from specs alone. In the initialized branch and under a valid ValidRoundObservation, the remaining sub-conditions hold: obs.LastVerifiedTS matches verifiedDB.lastTimestamp by direct assignment; the rewind sub-condition holds because Valid()/Sequential and AllDBsInSync() guarantee lastTS-1 is in verifiedDB.db when activationTimestamp < lastTS; and the AdvancesVerifiedDB sub-condition is vacuously true because obs.L2sConsistent is never set to true in observeRound (Go zero value is false).

**Postcondition `ObservationConsistentWithLogs(obs)`** (violated):

ObservationConsistentWithLogs has both ValidRoundObservation(obs) and ObservationConsistentWithVerified(obs) as predicate preconditions. The POST3-violation scenario (empty verifiedDB + frontier L1 check returns false) makes ValidRoundObservation(obs) false, so the ensures clause is violated. In scenarios where ValidRoundObservation does hold: the rewind sub-condition (!obs.l1Consistent && lastVerifiedTS.value > activationTimestamp ==> sealed blocks for lastTS-1 exist) is satisfied because AllDBsInSync() guarantees DBsInSyncUpTo through lastTS, which covers lastTS-1; the advance sub-condition (chainsReady && l2sConsistent && l1Consistent ==> AdvancesAllLogsDBs) is vacuously true because observeRound never sets obs.L2sConsistent=true (Go zero value is false, a structural divergence from the Dafny model).

---

## 14. `applyPendingTransition` → `Interop.ApplyPendingTransition`

**Go file**: `supernode/activity/interop/interop.go:673`  
**Dafny**: `dafny-models/Interop.dfy` — class `Interop`, method `ApplyPendingTransition`  
**Notes**: Before any invalidateBlock call, Go pauses all chains' validation nodes via PauseAndStopVN and resumes non-invalidated chains afterwards; this VN lifecycle management has no counterpart in the Dafny model. Go also has a nil-result guard that calls ClearPendingTransition and returns gracefully when pending.Result == nil; the Dafny model treats this scenario as a precondition violation rather than a recoverable runtime condition.  

### Preconditions

- [x] `Valid()` — **Satisfied**
- [x] `verifiedDB.GetPendingTransition() == Some(pending)` — **Satisfied**
- [x] `PendingTransitionIsConsistent()` — **Satisfied**
- [x] `TransitionIsCrossValid(pending)` — **Satisfied**
- [x] `AllVerifiedCrossValid()` — **Satisfied**
- [x] `AllLogsDBsConsistentWithChainData()` — **Satisfied**

### Postconditions

- [?] `Valid()` — **Uncertain** — DecisionInvalidate path: invalidateBlock has no Dafny spec, so it is unknown whether it preserves the Valid() invariants on verifiedDB and logsDBs. ClearPendingTransition only modifies verifiedDB.pendingTransition and cannot repair deeper invariant violations. DecisionAdvance path: commitVerifiedResult has no Dafny spec; if it leaves verifiedDB.db or logsDBs in a state that violates Valid() (e.g. non-sequential timestamps or mismatched l2Heads.Keys), Valid() would not hold despite the subsequent ClearPendingTransition call.
- [?] `PendingTransitionIsConsistent()` — **Uncertain** — DecisionInvalidate success path: after ClearPendingTransition sets pendingTransition == None, PendingTransitionIsConsistent() requires AllDBsInSync(). invalidateBlock has no Dafny spec guaranteeing AllDBsInSync() is preserved. DecisionAdvance success path: after commitVerifiedResult and ClearPendingTransition, AllDBsInSync() is required; commitVerifiedResult has no spec confirming it performs the VerifiedDB.Commit step that would bring verifiedDB.db in line with the already-updated logsDBs. DecisionInvalidate error path with failedAny: pendingTransition == Some(pending) remains but the Invalidate extra condition AllDBsInSync() may be broken by partial invalidateBlock calls.
- [?] `AllVerifiedCrossValid()` — **Uncertain** — DecisionInvalidate path: invalidateBlock has no Dafny spec; if it modifies verifiedDB.db in a way that adds or alters entries, previously cross-valid results might no longer satisfy ResultIsCrossValid under the new chain state. DecisionAdvance path: commitVerifiedResult has no Dafny spec; if it does not perform the VerifiedDB.Commit that establishes AllVerifiedCrossValid() for the new timestamp, the postcondition is unverified.
- [?] `AllLogsDBsConsistentWithChainData()` — **Uncertain** — AllLogsDBsConsistentWithChainData() is a conjunct of Valid(). Same uncertainty applies: invalidateBlock (DecisionInvalidate) and commitVerifiedResult (DecisionAdvance) have no Dafny specs confirming they preserve the LogsDB-to-chain-data consistency invariant. ClearPendingTransition does not repair logsDB state.
- [?] `madeProgress.None? ==> verifiedDB.pendingTransition == Some(pending)` — **Uncertain** — DecisionAdvance error from commitVerifiedResult: commitVerifiedResult has no Dafny spec. It is unknown whether it modifies verifiedDB.pendingTransition before returning an error. If it does (e.g. if it internally calls ClearPendingTransition or sets pendingTransition to None as part of a partial commit), the error return would have pendingTransition != Some(pending), violating this postcondition. All other error paths (applyRewindPlan failure, persistFrontierLogs failure, failedAny in Invalidate) do not call ClearPendingTransition first and the relevant modifies clauses (applyRewindPlan: verifiedDB.pendingTransition == old(verifiedDB.pendingTransition); ChainContainer.InvalidateBlock: modifies this only; persistFrontierLogs: modifies logsDBs.Values and chains.Values only) confirm pendingTransition is unchanged in those cases.
- [?] `madeProgress.Some? ==> verifiedDB.pendingTransition == None` — **Uncertain** — All successful Go return paths call ClearPendingTransition last, which ensures pendingTransition == None — but ClearPendingTransition requires Valid() as a precondition. For DecisionInvalidate: Valid() after invalidateBlock calls is not guaranteed (no spec for invalidateBlock), so the precondition of ClearPendingTransition may be violated and its postcondition (pendingTransition == None) cannot be relied upon. For DecisionAdvance: Valid() after commitVerifiedResult is not guaranteed (no spec), same issue.
- [x] `madeProgress.Some? ==> (madeProgress.value <==> pending.decision == Advance)` — **Satisfied**
- [?] `madeProgress.Some? ==> AllDBsInSync()` — **Uncertain** — DecisionInvalidate success: after invalidateBlock calls, AllDBsInSync() requires each chain's logsDB LatestSealedBlock to match the verifiedDB's latest verified l2Head. invalidateBlock has no Dafny spec guaranteeing this; it may rewind some logsDBs but leave verifiedDB.db unchanged, or vice versa. DecisionAdvance success: persistFrontierLogs postcondition ensures UpdatedAllLogsDBs (logsDBs advanced), but commitVerifiedResult has no Dafny spec to confirm it performs the VerifiedDB.Commit step that would make verifiedDB.db consistent with those logsDB updates, so AllDBsInSync() is unverified.
- [?] `madeProgress.None? ==> TransitionIsCrossValid(pending)` — **Uncertain** — TransitionIsCrossValid is trivially true for Rewind and Invalidate decisions, so error paths in those branches satisfy this postcondition trivially. For DecisionAdvance errors from commitVerifiedResult: commitVerifiedResult has no Dafny spec; if it modifies logsDBs (which AllInitMsgsPresent/InitMsgInLogsDB depends on) in a way that removes entries needed by ResultIsCrossValid(pending.result.value), the postcondition could be violated. persistFrontierLogs errors are safe because its partial-update postcondition only adds blocks to logsDBs (Contains is monotone), which can only maintain or strengthen InitMsgInLogsDB queries.

### Callsite Analysis

| Caller | Precondition Verdicts |
|---|---|
| `progressAndRecord` (supernode/activity/interop/interop.go:467) | `Valid()` ✓ — Valid() is a direct precondition of progressAndRecord. GetPendingTransition() is frame-safe (no modifies clause), so Valid() is unmodified between progressAndRecord entry and the call.; `verifiedDB.GetPendingTransition() == Some(pending)` ✓ — pending is the non-nil result of GetPendingTransition(); applyPendingTransition receives *pending. Nothing between the GetPendingTransition call and the applyPendingTransition call modifies verifiedDB.pendingTransition, so GetPendingTransition() == Some(*pending) at the call.; `PendingTransitionIsConsistent()` ✓ — PendingTransitionIsConsistent() is a direct precondition of progressAndRecord; GetPendingTransition() is frame-safe, so it holds unchanged at the call.; `TransitionIsCrossValid(pending)` ✓ — progressAndRecord precondition: GetPendingTransition().Some? ==> TransitionIsCrossValid(...). Since pending != nil the Some? branch fires, yielding TransitionIsCrossValid(*pending).; `AllVerifiedCrossValid()` ✓ — AllVerifiedCrossValid() is a direct precondition of progressAndRecord; frame-safe GetPendingTransition() does not change verifiedDB.db.; `AllLogsDBsConsistentWithChainData()` ✓ — AllLogsDBsConsistentWithChainData() is a conjunct of Valid(). Valid() is a direct precondition of progressAndRecord; frame-safe GetPendingTransition() does not disturb it. |
| `progressAndRecord` (supernode/activity/interop/interop.go:490) | `Valid()` ✓ — progressInterop postcondition ensures Valid(). buildPendingTransition is frame-safe. SetPendingTransition postcondition ensures Valid(). All three preserve Valid() up to the call.; `verifiedDB.GetPendingTransition() == Some(pending)` ✓ — SetPendingTransition postcondition ensures pendingTransition == Some(pendingTx). applyPendingTransition(pendingTx) is called immediately after with no intervening state changes, so GetPendingTransition() == Some(pendingTx) == Some(pending).; `PendingTransitionIsConsistent()` ✓ — progressInterop postcondition yields AllDBsInSync(). buildPendingTransition postcondition yields TransitionConsistentWithVerified, TransitionConsistentWithLogs, and TransitionConsistentWithChainState (via OutputConsistentWithChainState from progressInterop). SetPendingTransition does not change verifiedDB.db or logsDBs. For each decision: Invalidate/Advance require AllDBsInSync() (preserved through frame-safe operations); Rewind extra condition AllDBsInSyncUpTo(resetAllChainsTo.value) follows because that timestamp <= LastTimestamp() and AllDBsInSync() implies AllDBsInSyncUpTo for any timestamp at or below LastTimestamp().; `TransitionIsCrossValid(pending)` ✓ — progressInterop postcondition: output.AdvanceOutput? ==> ResultIsCrossValid(output.result). buildPendingTransition postcondition: output.AdvanceOutput? <==> pendingTx.decision.Advance? and pendingTx.result.value == output.result for Advance. TransitionIsCrossValid is trivially true for Rewind and Invalidate decisions.; `AllVerifiedCrossValid()` ✓ — progressInterop postcondition ensures AllVerifiedCrossValid(). buildPendingTransition is frame-safe. SetPendingTransition does not modify verifiedDB.db (postcondition: db == old(db)), so AllVerifiedCrossValid() is preserved.; `AllLogsDBsConsistentWithChainData()` ✓ — progressInterop postcondition ensures Valid() which includes AllLogsDBsConsistentWithChainData(). buildPendingTransition is frame-safe; SetPendingTransition only modifies verifiedDB.pendingTransition. Condition preserved at the call. |

### Violation Scenarios

**Postcondition `Valid()`** (uncertain):

DecisionInvalidate path: invalidateBlock has no Dafny spec, so it is unknown whether it preserves the Valid() invariants on verifiedDB and logsDBs. ClearPendingTransition only modifies verifiedDB.pendingTransition and cannot repair deeper invariant violations. DecisionAdvance path: commitVerifiedResult has no Dafny spec; if it leaves verifiedDB.db or logsDBs in a state that violates Valid() (e.g. non-sequential timestamps or mismatched l2Heads.Keys), Valid() would not hold despite the subsequent ClearPendingTransition call.

**Postcondition `PendingTransitionIsConsistent()`** (uncertain):

DecisionInvalidate success path: after ClearPendingTransition sets pendingTransition == None, PendingTransitionIsConsistent() requires AllDBsInSync(). invalidateBlock has no Dafny spec guaranteeing AllDBsInSync() is preserved. DecisionAdvance success path: after commitVerifiedResult and ClearPendingTransition, AllDBsInSync() is required; commitVerifiedResult has no spec confirming it performs the VerifiedDB.Commit step that would bring verifiedDB.db in line with the already-updated logsDBs. DecisionInvalidate error path with failedAny: pendingTransition == Some(pending) remains but the Invalidate extra condition AllDBsInSync() may be broken by partial invalidateBlock calls.

**Postcondition `AllVerifiedCrossValid()`** (uncertain):

DecisionInvalidate path: invalidateBlock has no Dafny spec; if it modifies verifiedDB.db in a way that adds or alters entries, previously cross-valid results might no longer satisfy ResultIsCrossValid under the new chain state. DecisionAdvance path: commitVerifiedResult has no Dafny spec; if it does not perform the VerifiedDB.Commit that establishes AllVerifiedCrossValid() for the new timestamp, the postcondition is unverified.

**Postcondition `AllLogsDBsConsistentWithChainData()`** (uncertain):

AllLogsDBsConsistentWithChainData() is a conjunct of Valid(). Same uncertainty applies: invalidateBlock (DecisionInvalidate) and commitVerifiedResult (DecisionAdvance) have no Dafny specs confirming they preserve the LogsDB-to-chain-data consistency invariant. ClearPendingTransition does not repair logsDB state.

**Postcondition `madeProgress.None? ==> verifiedDB.pendingTransition == Some(pending)`** (uncertain):

DecisionAdvance error from commitVerifiedResult: commitVerifiedResult has no Dafny spec. It is unknown whether it modifies verifiedDB.pendingTransition before returning an error. If it does (e.g. if it internally calls ClearPendingTransition or sets pendingTransition to None as part of a partial commit), the error return would have pendingTransition != Some(pending), violating this postcondition. All other error paths (applyRewindPlan failure, persistFrontierLogs failure, failedAny in Invalidate) do not call ClearPendingTransition first and the relevant modifies clauses (applyRewindPlan: verifiedDB.pendingTransition == old(verifiedDB.pendingTransition); ChainContainer.InvalidateBlock: modifies this only; persistFrontierLogs: modifies logsDBs.Values and chains.Values only) confirm pendingTransition is unchanged in those cases.

**Postcondition `madeProgress.Some? ==> verifiedDB.pendingTransition == None`** (uncertain):

All successful Go return paths call ClearPendingTransition last, which ensures pendingTransition == None — but ClearPendingTransition requires Valid() as a precondition. For DecisionInvalidate: Valid() after invalidateBlock calls is not guaranteed (no spec for invalidateBlock), so the precondition of ClearPendingTransition may be violated and its postcondition (pendingTransition == None) cannot be relied upon. For DecisionAdvance: Valid() after commitVerifiedResult is not guaranteed (no spec), same issue.

**Postcondition `madeProgress.Some? ==> AllDBsInSync()`** (uncertain):

DecisionInvalidate success: after invalidateBlock calls, AllDBsInSync() requires each chain's logsDB LatestSealedBlock to match the verifiedDB's latest verified l2Head. invalidateBlock has no Dafny spec guaranteeing this; it may rewind some logsDBs but leave verifiedDB.db unchanged, or vice versa. DecisionAdvance success: persistFrontierLogs postcondition ensures UpdatedAllLogsDBs (logsDBs advanced), but commitVerifiedResult has no Dafny spec to confirm it performs the VerifiedDB.Commit step that would make verifiedDB.db consistent with those logsDB updates, so AllDBsInSync() is unverified.

**Postcondition `madeProgress.None? ==> TransitionIsCrossValid(pending)`** (uncertain):

TransitionIsCrossValid is trivially true for Rewind and Invalidate decisions, so error paths in those branches satisfy this postcondition trivially. For DecisionAdvance errors from commitVerifiedResult: commitVerifiedResult has no Dafny spec; if it modifies logsDBs (which AllInitMsgsPresent/InitMsgInLogsDB depends on) in a way that removes entries needed by ResultIsCrossValid(pending.result.value), the postcondition could be violated. persistFrontierLogs errors are safe because its partial-update postcondition only adds blocks to logsDBs (Contains is monotone), which can only maintain or strengthen InitMsgInLogsDB queries.

---

## 15. `progressInterop` → `Interop.ProgressInterop`

**Go file**: `supernode/activity/interop/interop.go:512`  
**Dafny**: `dafny-models/Interop.dfy` — class `Interop`, method `ProgressInterop`  
**Notes**: The precondition-check logic is extracted into a separate standalone function checkPreconditions in Go; the Dafny model inlines this logic directly inside ProgressInterop. Go also supports a Paused state (controlled by pauseAtTimestamp) that silently halts verification progress; this concept is absent from the Dafny model.  

### Preconditions

- [x] `Valid()` — **Satisfied**
- [x] `AllDBsInSync()` — **Satisfied**
- [x] `AllVerifiedCrossValid()` — **Satisfied**

### Postconditions

- [x] `Valid()` — **Satisfied**
- [x] `AllDBsInSync()` — **Satisfied**
- [x] `AllVerifiedCrossValid()` — **Satisfied**
- [x] `ValidStepOutput(output, obs)` — **Satisfied**
- [?] `OutputConsistentWithVerified(output, obs)` — **Uncertain** — For the AdvanceOutput case, OutputConsistentWithVerified requires AdvancesVerifiedDB(result.timestamp, result.l2Heads). ObservationConsistentWithVerified (ensured by observeRound) only guarantees AdvancesVerifiedDB when obs.chainsReady && obs.l2sConsistent && obs.l1Consistent. However, observeRound in Go never sets obs.L2sConsistent — it retains its zero value (false) — so the antecedent of that implication is never satisfied and AdvancesVerifiedDB is not established through any listed Dafny postcondition. An informal argument (AllDBsInSync() precondition + checkChainsReady postcondition AdvancesAllLogsDBs implies AdvancesVerifiedDB) would establish it, but this equivalence is not captured by any Dafny lemma or spec in scope.
- [?] `OutputConsistentWithLogs(output, obs)` — **Uncertain** — OutputConsistentWithLogs carries a requires of OutputConsistentWithVerified(output, obs). For the AdvanceOutput case that requires clause is uncertain (see post 5). For WaitOutput and RewindOutput the requires is satisfied and ObservationConsistentWithLogs (ensured by observeRound) supplies the needed sealed-block property for the RewindOutput path; for AdvanceOutput, checkChainsReady ensures AdvancesAllLogsDBs independently, but since the requires (post 5) is uncertain the overall status of post 6 is uncertain.
- [x] `OutputConsistentWithChainState(output, obs)` — **Satisfied**
- [x] `output.AdvanceOutput? ==> ResultIsCrossValid(output.result)` — **Satisfied**

### Callsite Analysis

| Caller | Precondition Verdicts |
|---|---|
| `progressAndRecord` (supernode/activity/interop/interop.go:471) | `Valid()` ✓ — progressAndRecord's Dafny spec requires Valid(). Before the call to progressInterop at line 471, only verifiedDB.GetPendingTransition() (frame-safe, no modifies) executes. When pending != nil the function returns early via applyPendingTransition without reaching progressInterop. No state is modified between the entry of progressAndRecord and the call to progressInterop, so Valid() holds at the call site.; `AllDBsInSync()` ✓ — progressAndRecord requires PendingTransitionIsConsistent(). When verifiedDB.GetPendingTransition() returns None (pending == nil in Go), PendingTransitionIsConsistent() reduces to AllDBsInSync(). progressInterop is only reachable when pending == nil (the non-nil branch returns early at line 467). GetPendingTransition() is frame-safe, so AllDBsInSync() is unmodified at the call site.; `AllVerifiedCrossValid()` ✓ — progressAndRecord requires AllVerifiedCrossValid(). Only the frame-safe GetPendingTransition() executes before progressInterop is called, leaving AllVerifiedCrossValid() unmodified at the call site. |

### Violation Scenarios

**Postcondition `OutputConsistentWithVerified(output, obs)`** (uncertain):

For the AdvanceOutput case, OutputConsistentWithVerified requires AdvancesVerifiedDB(result.timestamp, result.l2Heads). ObservationConsistentWithVerified (ensured by observeRound) only guarantees AdvancesVerifiedDB when obs.chainsReady && obs.l2sConsistent && obs.l1Consistent. However, observeRound in Go never sets obs.L2sConsistent — it retains its zero value (false) — so the antecedent of that implication is never satisfied and AdvancesVerifiedDB is not established through any listed Dafny postcondition. An informal argument (AllDBsInSync() precondition + checkChainsReady postcondition AdvancesAllLogsDBs implies AdvancesVerifiedDB) would establish it, but this equivalence is not captured by any Dafny lemma or spec in scope.

**Postcondition `OutputConsistentWithLogs(output, obs)`** (uncertain):

OutputConsistentWithLogs carries a requires of OutputConsistentWithVerified(output, obs). For the AdvanceOutput case that requires clause is uncertain (see post 5). For WaitOutput and RewindOutput the requires is satisfied and ObservationConsistentWithLogs (ensured by observeRound) supplies the needed sealed-block property for the RewindOutput path; for AdvanceOutput, checkChainsReady ensures AdvancesAllLogsDBs independently, but since the requires (post 5) is uncertain the overall status of post 6 is uncertain.

---

## 16. `progressAndRecord` → `Interop.ProgressAndRecord`

**Go file**: `supernode/activity/interop/interop.go:460`  
**Dafny**: `dafny-models/Interop.dfy` — class `Interop`, method `ProgressAndRecord`  

### Preconditions

- [?] `Valid()` — **Uncertain** — If a prior invocation of `applyPendingTransition` returned a Go error after partially mutating `verifiedDB` or `logsDBs`, `Valid()` (which requires `verifiedDB.Valid()`, `AllLogsDBsConsistentWithChainData()`, sequential commits, etc.) could be broken before the next call to `progressAndRecord`. Dafny models errors as `None` returns and does not cover partial mutations on Go error paths, so this scenario is outside the formal model but observable in the implementation.
- [?] `PendingTransitionIsConsistent()` — **Uncertain** — Same error-path scenario as `Valid()`: if `applyPendingTransition` partially rewinds or advances `verifiedDB`/`logsDBs` before returning a Go error, `logsDBs` and `verifiedDB` could be out of sync, violating `AllDBsInSync()` and therefore `PendingTransitionIsConsistent()` at the next call site.
- [?] `AllVerifiedCrossValid()` — **Uncertain** — If `applyPendingTransition` commits a new timestamp to `verifiedDB` but then fails before updating the logsDBs to match, the new entry in `verifiedDB` would not satisfy `ResultIsCrossValid` (because `AllInitMsgsInLogsDB` could fail), violating `AllVerifiedCrossValid()` at the next entry to `progressAndRecord`.
- [?] `verifiedDB.GetPendingTransition().Some? ==> TransitionIsCrossValid(verifiedDB.GetPendingTransition().value)` — **Uncertain** — If a future version of the code stores a pending transition in `verifiedDB` before the cross-validity of its result is confirmed (e.g., calling `SetPendingTransition` before `progressInterop` validates the blocks), then a crash-and-restart picking up that pending transition would find `TransitionIsCrossValid` not holding. This is not the current code flow, but the condition cannot be proven satisfied purely from the Go code without formal verification.

### Postconditions

- [x] `Valid()` — **Satisfied**
- [x] `PendingTransitionIsConsistent()` — **Satisfied**
- [x] `AllVerifiedCrossValid()` — **Satisfied**
- [x] `verifiedDB.GetPendingTransition().Some? ==> TransitionIsCrossValid(verifiedDB.GetPendingTransition().value)` — **Satisfied**

### Callsite Analysis

| Caller | Precondition Verdicts |
|---|---|
| `progress` (supernode/activity/interop/interop.go:386) | `Valid()` ? — `progress` calls `progressAndRecord` as its first statement with no prior action that establishes or checks `Valid()`. `Valid()` must therefore hold on entry to `progress` itself, which requires tracing the full call chain back to the constructor. The constructor ensures `Valid()`, and every in-scope mapped function (including `applyPendingTransition`, `progressInterop`, `refreshCurrentL1OnWait`, `SetPendingTransition`) lists `Valid()` in its postconditions, so the invariant is designed to be maintained across calls. However, Go provides no static enforcement of this; an I/O-error early-exit from `applyPendingTransition` (modifies `this, verifiedDB, chains.Values, logsDBs.Values`) could leave the object in a state where `Valid()` no longer holds before the next invocation of `progress`.; `PendingTransitionIsConsistent()` ? — `progress` performs no action before calling `progressAndRecord`, so `PendingTransitionIsConsistent()` must hold on entry to `progress`. When `verifiedDB.GetPendingTransition() == None`, this collapses to `AllDBsInSync()`; when `Some(p)`, it additionally requires `TransitionConsistentWithVerified(p)`, `TransitionConsistentWithLogs(p)`, and `TransitionConsistentWithChainState(p)`. `applyPendingTransition` ensures `PendingTransitionIsConsistent()` post-call, and `progressInterop` ensures `AllDBsInSync()`, so the invariant is designed to be propagated. The same caveat applies as for `Valid()`: a Go-error path out of `applyPendingTransition` after partial DB mutation could leave `AllDBsInSync()` violated for the next round.; `AllVerifiedCrossValid()` ? — `progress` does not check or establish `AllVerifiedCrossValid()`. The constructor ensures it; `applyPendingTransition` and `progressInterop` both list it in their postconditions; and functions that do not modify `verifiedDB` or `logsDBs.Values` (e.g., `buildPendingTransition`, `refreshCurrentL1OnWait`) preserve it by framing. The same caveat applies: a Go error mid-way through `applyPendingTransition` (which commits to `verifiedDB` before potentially failing) could invalidate a just-committed result, breaking `AllVerifiedCrossValid()` before the next call.; `verifiedDB.GetPendingTransition().Some? ==> TransitionIsCrossValid(verifiedDB.GetPendingTransition().value)` ? — `progress` does not check this condition. The invariant is maintained by: (a) the constructor ensuring it, (b) `applyPendingTransition` postcondition ensuring it when `madeProgress.None?` (pending kept), and clearing the pending when `madeProgress.Some?` (vacuously satisfied). In path 2c of `progressAndRecord`, `TransitionIsCrossValid(pendingTx)` is established from `progressInterop`'s `output.AdvanceOutput? ==> ResultIsCrossValid(output.result)` combined with `buildPendingTransition`'s `pendingTx.decision.Advance? <==> output.AdvanceOutput?` and `pendingTx.decision.Advance? ==> pendingTx.result.value == output.result`. However, if a prior Go error left a pending transition in `verifiedDB` whose cross-validity was not confirmed (e.g., stored before `ResultIsCrossValid` was established), this precondition could be violated. |

### Violation Scenarios

**Precondition `Valid()`** (uncertain):

If a prior invocation of `applyPendingTransition` returned a Go error after partially mutating `verifiedDB` or `logsDBs`, `Valid()` (which requires `verifiedDB.Valid()`, `AllLogsDBsConsistentWithChainData()`, sequential commits, etc.) could be broken before the next call to `progressAndRecord`. Dafny models errors as `None` returns and does not cover partial mutations on Go error paths, so this scenario is outside the formal model but observable in the implementation.

**Precondition `PendingTransitionIsConsistent()`** (uncertain):

Same error-path scenario as `Valid()`: if `applyPendingTransition` partially rewinds or advances `verifiedDB`/`logsDBs` before returning a Go error, `logsDBs` and `verifiedDB` could be out of sync, violating `AllDBsInSync()` and therefore `PendingTransitionIsConsistent()` at the next call site.

**Precondition `AllVerifiedCrossValid()`** (uncertain):

If `applyPendingTransition` commits a new timestamp to `verifiedDB` but then fails before updating the logsDBs to match, the new entry in `verifiedDB` would not satisfy `ResultIsCrossValid` (because `AllInitMsgsInLogsDB` could fail), violating `AllVerifiedCrossValid()` at the next entry to `progressAndRecord`.

**Precondition `verifiedDB.GetPendingTransition().Some? ==> TransitionIsCrossValid(verifiedDB.GetPendingTransition().value)`** (uncertain):

If a future version of the code stores a pending transition in `verifiedDB` before the cross-validity of its result is confirmed (e.g., calling `SetPendingTransition` before `progressInterop` validates the blocks), then a crash-and-restart picking up that pending transition would find `TransitionIsCrossValid` not holding. This is not the current code flow, but the condition cannot be proven satisfied purely from the Go code without formal verification.

---
