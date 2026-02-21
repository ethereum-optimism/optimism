# OP Stack Deep Security Audit Report

**Scope:** op-node · op-geth · op-dispute-mon
**Branch:** `claude/op-stack-security-audit-PRwvB`
**Date:** 2026-02-21
**Methodology:** Line-by-line manual code review, System 2 thinking, full money-flow tracing L1→L2→L1

---

## Executive Summary

This security audit covers the three specified OP Stack components across ~420 Go source files. The audit traces the full lifecycle of assets: L1 deposits → L2 derivation → L2 execution → L2→L1 withdrawals → dispute game resolution → withdrawal finalization.

**16 findings** were identified:

| Severity | Count | IDs |
|----------|-------|-----|
| HIGH     | 4     | TREE-NIL, BATCH-SPAN-FORK, CLAIMS-OOB, COMMIT-SYNCING |
| MEDIUM   | 6     | FEE-FORMULA, CLOCK-OVERFLOW, UNDERFLOW-MULTI, SYSCONF-SILENT, TIMESTAMP-INT64, STALE-CACHE |
| LOW      | 6     | ISCREATION, CHANNEL-READ, NOTFOUND-STR, FIRSTRESULT, SAFE-SINGLE, ASYNC-BLOCK |

---

## Money Flow Architecture (Traced)

```
L1 User → OptimismPortal.depositTransaction()
        → TransactionDeposited event
op-node: L1Traversal → L1Retrieval → deposit_log.go (UnmarshalDepositLogEvent)
       → deposits.go (UserDeposits) → attributes.go (PreparePayloadAttributes)
       → Engine API forkchoiceUpdated + PayloadAttributes
op-geth: state_transition.go preCheck() [deposit bypasses EOA/nonce/fee]
       → AddBalance(mint) → execute() → state transition
       → block_validator.go ValidateBody/State

L2→L1 Withdrawal path:
op-node: output_root.go ComputeL2OutputRoot → proposals
op-dispute-mon: extractor.go → enrichGames → monitor.go monitorGames
             → forecast.go → resolutions.go → withdrawals.go
             → bonds/monitor.go → transform/tree.go → resolve.go
```

---

## Part 1: Deposit Flow (L1→L2)

### [LOW-1] ISCREATION — Non-standard isCreation Byte Accepted

**Location:** `op-node/rollup/derive/deposit_log.go:129`
**Severity:** Low
**Type:** Input Validation

**Code:**
```go
if opaqueData[offset] == 0 {
    dep.To = &to
}
// If non-zero → treated as contract creation (isCreation = true)
```

**Description:** The spec defines `isCreation` as a boolean (0 or 1), but the implementation treats ANY non-zero byte value as contract creation. Values 2–255 are silently accepted as contract creation requests. This deviates from the spec but has no direct exploit path since the L1 smart contract is the canonical source.

**Impact:** Low — deterministic behavior, but may confuse tooling that validates `isCreation` must be 0 or 1.

**Recommendation:** Add explicit check: `if opaqueData[offset] != 0 && opaqueData[offset] != 1 { return nil, fmt.Errorf("invalid isCreation byte: %d", opaqueData[offset]) }`

---

### [DESIGN-1] Mint Applied Before Snapshot in Deposit Failure Path

**Location:** `op-geth/core/state_transition.go:467-503`
**Severity:** Informational (Intentional Design)
**Type:** Protocol Semantics

**Code:**
```go
func (st *stateTransition) execute() (*ExecutionResult, error) {
    if mint := st.msg.Mint; mint != nil {
        st.state.AddBalance(st.msg.From, mintU256, tracing.BalanceMint) // BEFORE snapshot
    }
    snap := st.state.Snapshot()
    result, err := st.innerExecute()
    if err != nil && err != ErrGasLimitReached && st.msg.IsDepositTx {
        st.state.RevertToSnapshot(snap) // reverts execution, NOT the mint
        // nonce incremented, gas consumed
    }
}
```

**Description:** When a deposit transaction fails (not `ErrGasLimitReached`), execution is reverted to the snapshot, but the `mint` (ETH bridging) is preserved because it happened before the snapshot. This is intentional per the OP Stack spec: failed deposits must still credit the minted ETH to prevent locked funds. This is **not a bug** but a critical invariant that must be preserved.

**Verification:** Confirmed by spec: "If a deposit transaction fails, the state snapshot is reverted, except for the ETH mint."

---

## Part 2: Derivation Pipeline

### [MED-1] SYSCONF-SILENT — SystemConfig Update Errors Silently Discarded

**Location:** `op-node/rollup/derive/attributes.go:97-99`
**Severity:** Medium
**Type:** Error Handling / Information Loss

**Code:**
```go
// errors from UpdateSystemConfigWithL1Receipts are ignored as they represent malformed
// or invalid updates and there is no recovery mechanism for malformed updates
_ = UpdateSystemConfigWithL1Receipts(&sysConfig, receipts, ba.rollupCfg, info.Time())
```

**Description:** All errors from processing L1 `ConfigUpdate` events are silently discarded. If a critical config parameter (e.g., batcher address, gas limit, EIP-1559 params) fails to parse, the derivation pipeline continues with the PREVIOUS config value, and no error is propagated.

**Impact:**
- An attacker who can craft malformed `ConfigUpdate` events on L1 could prevent configuration updates from taking effect, forcing the system to operate with stale configuration
- Silent failures make debugging very difficult
- If the update contains a critical address change (e.g., new batcher), the old address is silently retained

**False Positive Check:** The comment says "there is no recovery mechanism for malformed updates, we must process past them." This is intentional design, but the silent discard (vs. at least logging at ERROR level) is problematic. The config copy-before-apply pattern (`updated` variable) is correct.

**Recommendation:** At minimum, log the error at ERROR level. Consider emitting a metric for failed config updates.

---

### [HIGH-1] BATCH-SPAN-FORK — Span Batches Bypass Fork Activation Block User-TX Restriction

**Location:** `op-node/rollup/derive/batches.go`
**Severity:** High
**Type:** Logic / Protocol Invariant Violation

**Description:** For Jovian and Interop fork activation blocks, user transactions must not be included. The check exists for singular batches:

```go
// checkSingularBatch() — lines 137-142
if (cfg.IsJovianActivationBlock(batch.Timestamp) ||
    cfg.IsInteropActivationBlock(batch.Timestamp)) &&
    len(batch.Transactions) > 0 {
    return BatchDrop
}
```

However, **`checkSpanBatch()` has NO equivalent check**. A span batch that covers the activation block can include user transactions in that specific block, violating the protocol invariant that activation blocks must be empty of user transactions.

**Attack Scenario:**
1. Sequencer submits a span batch covering blocks N-1, N (activation), N+1
2. Block N is the fork activation block — span batch includes user txs in block N
3. `checkSingularBatch()` would reject this, but `checkSpanBatch()` does not
4. User transactions execute in the activation block alongside upgrade system transactions

**Impact:** Protocol invariant violation. Depending on the activation block's upgrade transactions (which may change precompile behavior, fee params, etc.), including user transactions in the same block could lead to unexpected behavior or economic issues.

**False Positive Check:** Confirmed by reading both `checkSingularBatch()` (has check, lines 137-142) and `checkSpanBatch()` (no equivalent check for Jovian/Interop activation).

**Recommendation:** Add equivalent fork activation block check in `checkSpanBatch()`:
```go
// In checkSpanBatch(), for each block in the span:
if (cfg.IsJovianActivationBlock(blockTimestamp) || cfg.IsInteropActivationBlock(blockTimestamp)) &&
    len(blockTxs) > 0 {
    return BatchDrop
}
```

---

## Part 3: Engine API (op-node ↔ op-geth)

### [HIGH-2] COMMIT-SYNCING — CommitBlock Sets Unsafe Head on ExecutionSyncing Status

**Location:** `op-node/rollup/engine/api.go:129-144`
**Severity:** High
**Type:** Logic / Missing Status Handling

**Code:**
```go
func (e *EngineController) CommitBlock(ctx context.Context, signed *opsigner.SignedExecutionPayloadEnvelope) error {
    // ...
    status, err := e.engine.NewPayload(ctx, envelope.ExecutionPayload, envelope.ParentBeaconBlockRoot)
    if err != nil {
        return fmt.Errorf("failed to insert payload: %w", err)
    }
    switch status.Status {
    case eth.ExecutionInvalid, eth.ExecutionInvalidBlockHash:
        return &rpc.JsonError{Code: apis.BuildErrCodeInvalidInput, ...}
    case eth.ExecutionValid:
        break
    }
    // FALLS THROUGH TO HERE for ExecutionSyncing or any unhandled status:
    e.SetUnsafeHead(ref)  // ← Set even if block not yet validated!
    e.emitter.Emit(ctx, UnsafeUpdateEvent{Ref: ref})
    ...
}
```

**Description:** In Go, `switch` statements don't fall through by default. If `status.Status` is `eth.ExecutionSyncing` (or any future status), no `case` matches, the switch exits, and `SetUnsafeHead(ref)` is called unconditionally. This means a block whose validity op-geth has not yet confirmed (still syncing) gets marked as the unsafe head.

**Contrast:** `payload_process.go:onPayloadProcess()` correctly handles this with a `default:` case that emits `EngineTemporaryErrorEvent`.

**Impact:** The unsafe head could advance to a block that op-geth hasn't validated. This could cause temporary inconsistency between op-node's view of the chain and op-geth's actual state. Subsequent operations on this invalid unsafe head could fail in unexpected ways.

**False Positive Check:** Confirmed by reading both files. `payload_process.go` handles `default` correctly (line 64-68), `api.go:CommitBlock` does not (missing `default:` case).

**Recommendation:** Add a `default:` case:
```go
default:
    return &rpc.JsonError{
        Code:    apis.BuildErrCodeTemporary,
        Message: fmt.Sprintf("unexpected payload status %v", status.Status),
    }
```

---

## Part 4: P2P Layer

### [LOW-2] ASYNC-BLOCK — AsyncGossiper.Gossip() Blocks Sequencer on Slow P2P

**Location:** `op-node/rollup/async/asyncgossiper.go:73`
**Severity:** Low
**Type:** Availability / Potential DoS

**Code:**
```go
func (p *SimpleAsyncGossiper) Gossip(payload *eth.ExecutionPayloadEnvelope) {
    p.set <- payload  // Blocks until the internal loop accepts it
}
```

**Description:** `Gossip()` is a synchronous blocking send to an unbuffered channel. The internal goroutine handles one operation at a time in a `select` loop. If the current operation is a slow `SignAndPublishL2Payload()` (e.g., due to P2P network congestion), `Gossip()` blocks until it completes. If the sequencer calls `Gossip()` after building a block, it cannot proceed to build the next block until the previous gossip completes.

**Impact:** P2P network slowness can cause the sequencer to lag behind in block production. In extreme cases (hung network), the sequencer halts. This is a self-DoS risk under adverse network conditions.

**False Positive Check:** Confirmed — channel `p.set` is unbuffered (line 57: `set: make(chan *eth.ExecutionPayloadEnvelope)`). Blocking is intentional per the comment "it blocks until the async routine is able to start handling the request", but the downstream call to `SignAndPublishL2Payload` can take arbitrarily long.

**Recommendation:** Add a timeout on the `Gossip()` send, or use a buffered channel (capacity 1) with non-blocking send that drops if busy.

---

## Part 5: Sequencer Logic

### [DESIGN-2] ConfDepth Numeric Overflow (Theoretical)

**Location:** `op-node/rollup/confdepth/conf_depth.go:36`
**Severity:** Informational
**Type:** Integer Overflow (Theoretical)

**Code:**
```go
if num == 0 || c.depth == 0 || num+c.depth <= l1Head.Number {
    return c.L1Fetcher.L1BlockRefByNumber(ctx, num)
}
```

**Description:** If `num + c.depth` overflows `uint64`, the check `num+c.depth <= l1Head.Number` could produce a false positive (bypassing the confirmation depth). Requires `num` near `MaxUint64`, which is impossible with real L1 block numbers (currently ~21M).

**Verdict:** **Not exploitable in practice.** Documented for completeness.

---

## Part 6: op-geth OP-specific Changes

### [MED-2] FEE-FORMULA — 100,000,000× Operator Fee Formula Discontinuity

**Location:** `op-geth/core/types/rollup_cost.go:253-287`
**Severity:** Medium
**Type:** Math / Economic

**Code:**
```go
// Isthmus operator fee:
func newOperatorCostFuncIsthmus(...) operatorCostFunc {
    return func(gas uint64) *uint256.Int {
        fee = fee.Mul(fee, operatorFeeScalar)
        fee = fee.Div(fee, oneMillion)        // ÷ 1,000,000
        fee = fee.Add(fee, operatorFeeConstant)
    }
}

// Jovian "fix":
func newOperatorCostFuncOperatorFeeFix(...) operatorCostFunc {
    return func(gas uint64) *uint256.Int {
        fee = fee.Mul(fee, operatorFeeScalar)
        fee = fee.Mul(fee, oneHundred)        // × 100 — NOT ÷ 1,000,000!
        fee = fee.Add(fee, operatorFeeConstant)
    }
}
```

**Description:** The Isthmus formula divides by 1,000,000 (`oneMillion`), while the Jovian "fix" multiplies by 100 (`oneHundred`). The ratio between the two formulas is 100 × 1,000,000 = **100,000,000×**. This is not a cosmetic difference — the formulas produce fundamentally different results for the same inputs. The Jovian formula would produce fees 100 million times larger than Isthmus for the same scalar value.

**Context:** This is labeled "OperatorFeeFix" suggesting it's an intentional correction to the Isthmus formula. However, the magnitude of change (100,000,000×) is extreme. Even if both formulas are "correct" under different scalar conventions, the absence of a scalar migration mechanism means any operator who sets their scalar under Isthmus conventions will see a 100,000,000× fee increase at the Jovian hard fork.

**Impact:**
- At Jovian fork activation, all transactions suddenly become 100M× more expensive for operator fees
- Users/wallets may reject transactions as too expensive
- If scalar is calibrated for Jovian but system is still on Isthmus, operator collects near-zero fees

**False Positive Check:** Confirmed by reading `rollup_cost.go` lines 253-287. `oneMillion = 1_000_000` (line ~240), `oneHundred = 100` (line ~241). The division is replaced by multiplication.

**Recommendation:**
1. Document the exact scalar units for each fork version
2. Verify the intended formula is tested against economic projections
3. Ensure operators are aware of the formula change and scalar recalibration needed

---

## Part 7: op-dispute-mon

### [HIGH-3] TREE-NIL — Nil Pointer Dereference in CreateBidirectionalTree

**Location:** `op-dispute-mon/mon/transform/tree.go:21-35`
**Severity:** High
**Type:** Crash / Availability

**Code:**
```go
func CreateBidirectionalTree(claims []types.EnrichedClaim) *types.BidirectionalTree {
    claimMap := make(map[int]*types.BidirectionalClaim)
    for _, claim := range claims {
        claim := claim
        bidirectionalClaim := &types.BidirectionalClaim{Claim: &claim.Claim}
        claimMap[claim.ContractIndex] = bidirectionalClaim
        if !claim.IsRoot() {
            // SAFETY: the parent must exist in the list prior to the current claim.
            parent := claimMap[claim.ParentContractIndex]  // nil if not in map!
            parent.Children = append(parent.Children, bidirectionalClaim)  // PANIC!
        }
    }
    ...
}
```

**Description:** The SAFETY comment assumes that claims are always returned in topological order with valid parent indices. If a dispute game returns claims with:
- Out-of-order indices (parent comes after child)
- Invalid `ParentContractIndex` pointing to a non-existent claim
- Maliciously crafted claim data from a compromised or griefing game

Then `claimMap[claim.ParentContractIndex]` returns Go's zero value for a pointer: `nil`. The next line `parent.Children = append(...)` causes a **nil pointer dereference** → **panic** → **op-dispute-mon process crash**.

**Attack Scenario:**
1. Attacker creates a dispute game on-chain with malformed claim ordering
2. op-dispute-mon queries the game's claims via RPC
3. `CreateBidirectionalTree` is called with out-of-order or invalid claims
4. op-dispute-mon crashes
5. While crashed, legitimate withdrawals from ongoing games are not monitored
6. Attacker can execute fraudulent withdrawals while monitoring is down

**Impact:** Process crash causing monitoring gap. Given op-dispute-mon is the safety monitor for the dispute game system, crashes create windows where fraudulent withdrawals can go undetected.

**False Positive Check:** Confirmed — Go maps return zero value (nil for pointers) when key not found. No nil check exists before `parent.Children = append(...)`.

**Recommendation:**
```go
parent := claimMap[claim.ParentContractIndex]
if parent == nil {
    return nil, fmt.Errorf("parent claim %d not found for claim %d",
        claim.ParentContractIndex, claim.ContractIndex)
}
parent.Children = append(parent.Children, bidirectionalClaim)
```

---

### [HIGH-4] CLAIMS-OOB — Index Out of Bounds in ClaimMonitor

**Location:** `op-dispute-mon/mon/claims.go:105`
**Severity:** High
**Type:** Crash / Availability

**Code:**
```go
var parent faultTypes.Claim
if !claim.IsRoot() {
    parent = game.Claims[claim.ParentContractIndex].Claim  // No bounds check!
}
```

**Description:** Similar to `TREE-NIL`, but in `ClaimMonitor.checkGameClaims()`. `claim.ParentContractIndex` is used directly as a slice index into `game.Claims` without validating that it is within bounds. If `claim.ParentContractIndex >= len(game.Claims)`, this panics with **index out of bounds**.

**Attack Scenario:** Same as TREE-NIL — a malformed dispute game with out-of-bounds parent indices crashes op-dispute-mon.

**False Positive Check:** Confirmed — `game.Claims` is a `[]EnrichedClaim` slice. Access at unchecked index from on-chain data.

**Recommendation:**
```go
if !claim.IsRoot() {
    if claim.ParentContractIndex >= len(game.Claims) {
        c.logger.Error("Invalid parent index", "game", game.Proxy,
            "claimIndex", claim.ContractIndex, "parentIndex", claim.ParentContractIndex)
        continue
    }
    parent = game.Claims[claim.ParentContractIndex].Claim
}
```

---

### [MED-3] UNDERFLOW-MULTI — Uint64 Underflow in Timestamp Arithmetic (Multiple Locations)

**Severity:** Medium
**Type:** Integer Underflow

**Affected Locations:**
1. `op-dispute-mon/mon/resolutions.go:36` — `duration := uint64(r.clock.Now().Unix()) - game.Timestamp`
2. `op-dispute-mon/mon/claims.go:86` — `duration := uint64(now.Unix()) - game.Timestamp`
3. `op-dispute-mon/mon/bonds/monitor.go:53` — `duration := uint64(b.clock.Now().Unix()) - game.Timestamp`

**Code (representative):**
```go
duration := uint64(r.clock.Now().Unix()) - game.Timestamp
// If game.Timestamp > uint64(now.Unix()), this UNDERFLOWS to ~MaxUint64
```

**Description:** `game.Timestamp` is fetched from on-chain dispute game data. If a game is created with a timestamp in the future (due to clock skew on the L2 node, or a maliciously crafted game), `game.Timestamp > uint64(now.Unix())`, causing uint64 underflow. The `duration` becomes approximately `MaxUint64 - (game.Timestamp - now)`, a huge number.

**Downstream Effects:**
- `resolutions.go:37`: `maxDurationReached := duration >= (2 * game.MaxClockDuration)` → true (huge duration ≥ threshold)
- `claims.go:87`: `firstHalf := duration <= game.MaxClockDuration` → false (game not considered in first half)
- `bonds/monitor.go:54`: `maxDurationReached := duration >= game.MaxClockDuration + uint64(game.WETHDelay.Seconds())` → true

**Impact:**
- A game with a future timestamp is incorrectly classified as having exceeded its clock duration
- Claims may be incorrectly reported as resolvable when they aren't
- Bond withdrawal eligibility may be incorrectly computed
- Monitoring alerts may be triggered for non-issues, or genuine issues missed

**False Positive Check:** Confirmed — uint64 subtraction underflows in Go (no panic, wraps around). The `game.Timestamp` field comes from on-chain data without sanitization in monitoring code.

**Recommendation:** Add guard before each subtraction:
```go
now := uint64(r.clock.Now().Unix())
if game.Timestamp > now {
    // Game timestamp is in the future - log warning and skip
    r.log.Warn("Game has future timestamp", "game", game.Proxy, "timestamp", game.Timestamp)
    continue
}
duration := now - game.Timestamp
```

---

### [MED-4] CLOCK-OVERFLOW — MaxClockDuration Multiplication Overflow

**Location:** `op-dispute-mon/mon/resolutions.go:37`
**Severity:** Medium
**Type:** Integer Overflow

**Code:**
```go
maxDurationReached := duration >= (2 * game.MaxClockDuration)
// If MaxClockDuration > MaxUint64/2, multiplication overflows!
```

**Description:** `game.MaxClockDuration` is a `uint64` fetched from on-chain data. If `MaxClockDuration > MaxUint64/2` (≈9.2×10¹⁸ seconds ≈ 292 billion years), then `2 * game.MaxClockDuration` overflows to a small value. The condition `duration >= (small value)` is then trivially true, causing the game to be classified as "max duration reached" immediately.

**Real-World Assessment:** A dispute game's max clock duration of 292 billion years is not realistic for any production game. However, a maliciously crafted dispute game could set this value to `MaxUint64`, causing `2 * MaxUint64 = MaxUint64 - 1` after overflow (actually wraps to `MaxUint64 * 2 mod 2^64 = MaxUint64 - 1 + 1 = 0` for `MaxUint64` = 2^64-1, so `2*(2^64-1) mod 2^64 = 2^64 - 2 = MaxUint64 - 1`).

**Combined with UNDERFLOW-MULTI:** If duration underflowed to ~MaxUint64, then `maxDurationReached` comparison may or may not hold depending on exact values.

**False Positive Check:** Confirmed — `uint64` overflow is well-defined in Go (wraps modulo 2^64). A dispute game with extreme `MaxClockDuration` could trigger this.

**Recommendation:**
```go
if game.MaxClockDuration > math.MaxUint64/2 {
    // Cannot safely double, treat as not reached
    maxDurationReached = false
} else {
    maxDurationReached = duration >= 2*game.MaxClockDuration
}
```

---

### [MED-5] TIMESTAMP-INT64 — BigInt Timestamp Truncation in Withdrawal Check

**Location:** `op-dispute-mon/mon/withdrawals.go:98`
**Severity:** Medium
**Type:** Integer Truncation

**Code:**
```go
if bigs.IsPositive(withdrawalAmount.Amount) &&
   time.Unix(withdrawalAmount.Timestamp.Int64(), 0).Add(game.WETHDelay).Before(now) {
```

**Description:** `withdrawalAmount.Timestamp` is a `*big.Int` from on-chain data. `big.Int.Int64()` silently truncates values larger than `2^63 - 1` (year 292,277,026,596). If the timestamp is set to a value > MaxInt64, `Int64()` returns a meaningless (likely negative) value. `time.Unix(negative, 0)` creates a time far in the past, making the delay check `Add(game.WETHDelay).Before(now)` always true.

**Impact:** A withdrawal with a timestamp set to any value > MaxInt64 would appear immediately claimable (ignoring the `WETHDelay`). If combined with other weaknesses in the dispute game system, this could allow premature withdrawal claims to go undetected by the monitoring system.

**False Positive Check:** Confirmed — `big.Int.Int64()` spec: "Int64 returns the int64 representation of x. If x cannot be represented in an int64, the result is undefined." In practice, Go's implementation returns the low 64 bits interpreted as int64 (truncation/wrapping).

**Recommendation:**
```go
if !withdrawalAmount.Timestamp.IsInt64() {
    log.Warn("Withdrawal timestamp exceeds int64", "timestamp", withdrawalAmount.Timestamp)
    continue // or treat as future timestamp
}
withdrawTime := time.Unix(withdrawalAmount.Timestamp.Int64(), 0)
```

---

### [MED-6] STALE-CACHE — Indefinitely Stale Game Data on Enrichment Failure

**Location:** `op-dispute-mon/mon/extract/extractor.go:112-127`
**Severity:** Medium
**Type:** Logic / Staleness

**Code:**
```go
for _, game := range games {
    previousData := e.latestGameData[game.Proxy]
    if previousData != nil {
        updatedGameData[game.Proxy] = previousData  // Use stale data as default
    }
    gameCh <- game
}
// ...
for enrichedGame := range enrichedCh {
    updatedGameData[enrichedGame.Proxy] = enrichedGame  // Override if enrichment succeeded
}
e.latestGameData = updatedGameData
```

**Description:** If enrichment fails for a game (RPC error), the previous round's data is used. This is a reasonable strategy for transient failures. However, if the RPC failure is persistent (e.g., the game contract is unreachable), the monitoring system silently uses increasingly stale data indefinitely, with no alerting.

**Impact:**
- A game's state could change significantly (e.g., become resolved, bonds distributed) but op-dispute-mon continues to report stale data
- Stale data for a resolved game could cause false "bond deficit" alerts
- No maximum staleness limit is enforced

**Recommendation:** Track `lastSuccessfulEnrichmentTime` per game and alert when staleness exceeds a threshold (e.g., 2× monitor interval).

---

### [LOW-3] NOTFOUND-STR — Fragile "not found" String Matching

**Location:** `op-dispute-mon/mon/extract/output_agreement_enricher.go:101`
**Severity:** Low
**Type:** Brittleness

**Code:**
```go
if strings.Contains(strings.ToLower(rpcErr.Error()), "not found") {
    results[i] = outputResult{notFound: true}
}
```

**Description:** The "not found" determination uses string matching on the error message. Different RPC providers use different error messages ("not found", "block not found", "output not found", "unknown output"). A provider using a different phrasing would cause the enricher to treat a valid "not found" as a generic error, potentially causing the game to be incorrectly classified.

**Recommendation:** Use structured error codes (e.g., RPC error code -32000 range) rather than string matching.

---

### [LOW-4] FIRSTRESULT — Diverged Expected Root Uses First Arbitrary Result

**Locations:**
- `op-dispute-mon/mon/extract/output_agreement_enricher.go:208`
- `op-dispute-mon/mon/extract/super_agreement_enricher.go:134`
**Severity:** Low
**Type:** Logic

**Code:**
```go
// On divergence:
game.ExpectedRootClaim = firstResult.outputRoot  // First result, not majority
```

**Description:** When multiple RPC nodes disagree on the expected output root, the first node's result is used as `ExpectedRootClaim`. The "first" result depends on goroutine scheduling (non-deterministic). This means the monitoring system's view of "expected" can vary between runs when nodes disagree.

**Impact:** If a minority of nodes returns the correct root and the first goroutine happened to pick an incorrect node's result, the monitoring system disagrees with the correct claim and raises false alerts (or misses real fraud).

**Recommendation:** Use majority voting or raise an alert when nodes disagree without declaring a winner.

---

### [LOW-5] SAFE-SINGLE — Single Node Sufficient for "isSafe" Determination

**Locations:**
- `op-dispute-mon/mon/extract/output_agreement_enricher.go:213-220`
- `op-dispute-mon/mon/extract/super_agreement_enricher.go:140-146`
**Severity:** Low
**Type:** Trust Model

**Code:**
```go
atLeastOneSafe := false
for _, result := range foundResults {
    if result.isSafe {
        atLeastOneSafe = true
        break  // One node is enough!
    }
}
```

**Description:** A claim is considered "safe" if ANY single configured RPC node reports it as safe. A compromised or misconfigured node returning `isSafe=true` is sufficient to cause op-dispute-mon to agree with a potentially fraudulent claim.

**Impact:** Low — requires a compromised monitoring node. But this weakens the trust assumptions of the monitoring system. An operator running 3 nodes where 1 is compromised would have their monitoring agree with invalid claims.

**Recommendation:** Require majority of nodes to report `isSafe=true` before agreeing.

---

## Part 8: Cross-Component Analysis

### [CROSS-1] Business Logic: Deposit Censorship Window

**Location:** Cross-component (op-node derivation + sequencer)
**Severity:** Low (by design)
**Type:** Business Logic

**Description:** The sequencer CAN censor individual deposits for up to `max_sequencer_drift` (1800 seconds = 30 minutes) before the derivation pipeline forces inclusion via L1 epoch processing. However, if the sequencer continuously produces blocks near the drift limit, it can delay deposits by ~30 minutes indefinitely. This is a known and accepted trade-off in the OP Stack design.

**Verification:** `origin_selector.go` confirms this: `driftCurrent > int64(maxDrift)` forces origin advancement. The `MaxSequencerDrift` varies by fork (Bedrock: configurable, modern forks: spec-defined).

---

### [CROSS-2] Economic: Operator Fee Scalar Griefing

**Location:** `op-geth/core/types/rollup_cost.go` + L1 SystemConfig contract
**Severity:** Medium (requires privileged access)
**Type:** Economic

**Description:** The `OperatorFeeScalar` and `OperatorFeeConstant` can be updated on L1 by the SystemConfig owner. If maliciously set to maximum values (scalar=MaxUint32, constant=MaxUint64), the operator fee per transaction could be:

- Isthmus: `gas * MaxUint32 / 1,000,000 + MaxUint64` — moderate depending on gas
- Jovian: `gas * MaxUint32 * 100 + MaxUint64` — ~1.4×10¹⁴ wei per unit gas

This would make all transactions prohibitively expensive. However:
- Requires SystemConfig owner to be malicious or compromised
- Fee scalars are publicly visible on-chain
- Governance controls these parameters

**Verdict:** Design-level risk, mitigated by governance.

---

### [CROSS-3] Monitor Crash → Withdrawal Fraud Window

**Location:** `op-dispute-mon` + dispute game protocol
**Severity:** HIGH when combined with TREE-NIL/CLAIMS-OOB
**Type:** Cross-component

**Description:** The HIGH findings TREE-NIL and CLAIMS-OOB can crash op-dispute-mon. When the monitor is down:
1. Dispute games are not monitored for incorrect outcomes
2. The `Bonds.CheckBonds()` function does not run → bond deficits not detected
3. `ClaimMonitor.CheckClaims()` does not run → claims resolved against honest actors not alerted
4. `WithdrawalMonitor.CheckWithdrawals()` does not run → invalid withdrawals not flagged

The combination of these crashes with a real fraud attempt creates a critical vulnerability window.

**Recommendation:** Implement a watchdog/supervisor process for op-dispute-mon, and/or add crash recovery for malformed game data instead of panicking.

---

## Summary Table

| ID | File | Line | Severity | Type | Description |
|----|------|------|----------|------|-------------|
| HIGH-1 | batches.go | ~137 | HIGH | Logic | Span batches bypass fork activation user-tx restriction |
| HIGH-2 | engine/api.go | 129-144 | HIGH | Logic | CommitBlock accepts ExecutionSyncing without error |
| HIGH-3 | transform/tree.go | 21 | HIGH | Crash | Nil pointer dereference → op-dispute-mon crash |
| HIGH-4 | mon/claims.go | 105 | HIGH | Crash | Index out of bounds → op-dispute-mon crash |
| MED-1 | attributes.go | 97-99 | MEDIUM | Error Handling | SystemConfig errors silently discarded |
| MED-2 | rollup_cost.go | 253-287 | MEDIUM | Math/Economic | 100,000,000× operator fee formula change |
| MED-3 | resolutions.go:36, claims.go:86, bonds/monitor.go:53 | multiple | MEDIUM | Arithmetic | Uint64 underflow on future game timestamps |
| MED-4 | resolutions.go | 37 | MEDIUM | Arithmetic | 2×MaxClockDuration uint64 overflow |
| MED-5 | withdrawals.go | 98 | MEDIUM | Arithmetic | big.Int.Int64() truncation for large timestamps |
| MED-6 | extractor.go | 112-127 | MEDIUM | Logic | Stale game data used indefinitely on failure |
| LOW-1 | deposit_log.go | 129 | LOW | Input Validation | Non-spec isCreation byte values accepted |
| LOW-2 | asyncgossiper.go | 73 | LOW | Availability | Gossip() blocks sequencer on P2P slowness |
| LOW-3 | output_agreement_enricher.go | 101 | LOW | Brittleness | "not found" string matching |
| LOW-4 | output_agreement_enricher.go:208, super_agreement_enricher.go:134 | multiple | LOW | Logic | First diverged result used as expected root |
| LOW-5 | output_agreement_enricher.go:213-220, super_agreement_enricher.go:140-146 | multiple | LOW | Trust Model | Single node sufficient for isSafe |
| CROSS-3 | op-dispute-mon (multiple) | — | HIGH (combined) | Cross-component | Crash vulnerabilities create fraud windows |

---

## Files Reviewed

### op-node (key files)
- `rollup/derive/deposit_log.go` ✅
- `rollup/derive/deposits.go` ✅
- `rollup/derive/deposit_source.go` ✅
- `rollup/derive/l1_block_info.go` ✅
- `rollup/derive/attributes.go` ✅
- `rollup/derive/attributes_queue.go` ✅
- `rollup/derive/channel_bank.go` ✅
- `rollup/derive/channel.go` ✅
- `rollup/derive/channel_in_reader.go` ✅
- `rollup/derive/batches.go` ✅
- `rollup/derive/span_batch.go` ✅
- `rollup/derive/frame.go` ✅
- `rollup/derive/system_config.go` ✅
- `rollup/engine/engine_controller.go` ✅
- `rollup/engine/api.go` ✅
- `rollup/engine/events.go` ✅
- `rollup/engine/build_start.go` ✅
- `rollup/engine/payload_process.go` ✅
- `rollup/engine/payloads_queue.go` ✅
- `rollup/driver/sync_deriver.go` ✅
- `rollup/async/asyncgossiper.go` ✅
- `rollup/sequencing/origin_selector.go` ✅
- `rollup/confdepth/conf_depth.go` ✅
- `rollup/finality/altda.go` ✅
- `rollup/output_root.go` ✅
- `p2p/gossip.go` ✅
- `p2p/gating/blocking.go` ✅
- `withdrawals/proof.go` ✅

### op-geth (key files)
- `core/state_transition.go` ✅
- `core/state_processor.go` ✅
- `core/block_validator.go` ✅
- `core/types/deposit_tx.go` ✅
- `core/types/rollup_cost.go` ✅
- `core/txpool/validation.go` ✅
- `consensus/misc/eip1559/eip1559_optimism.go` ✅
- `eth/catalyst/api.go` ✅
- `eth/catalyst/api_optimism.go` ✅
- `eth/interop.go` ✅
- `miner/payload_building.go` ✅

### op-dispute-mon (key files)
- `mon/monitor.go` ✅
- `mon/forecast.go` ✅
- `mon/resolutions.go` ✅
- `mon/withdrawals.go` ✅
- `mon/claims.go` ✅
- `mon/resolve.go` ✅
- `mon/bonds/monitor.go` ✅
- `mon/bonds/collateral.go` ✅
- `mon/extract/extractor.go` ✅
- `mon/extract/output_agreement_enricher.go` ✅
- `mon/extract/super_agreement_enricher.go` ✅
- `mon/extract/claim_enricher.go` ✅
- `mon/extract/bond_enricher.go` ✅
- `mon/extract/head_enricher.go` ✅
- `mon/extract/withdrawals_enricher.go` ✅
- `mon/transform/tree.go` ✅
- `mon/resolve.go` ✅
- `mon/l2_challenges.go` ✅
- `mon/different_output_roots.go` ✅
- `mon/mixed_availability.go` ✅
- `mon/update_times.go` ✅

---

## Progress: ~85% complete

**Remaining to review:**
- `op-node/rollup/driver/driver.go` (full)
- `op-node/rollup/driver/follow_source.go`
- `op-node/p2p/app_params.go`, `app_scores.go`, `discovery.go`, `filter.go`
- `op-node/rollup/conductor/conductor.go`
- `op-node/rollup/finality/finalizer.go` (full detail)
- `op-node/rollup/chain_spec.go`
- `op-geth/core/vm/contracts.go` (OP precompiles)
- `op-geth/core/genesis.go` (OP parts)
- `op-geth/consensus/beacon/consensus.go` (OP parts)
- `op-geth/miner/worker_optimism.go` (if exists)
- `op-dispute-mon/config/config.go`
- `op-dispute-mon/metrics/metrics.go`
- `op-dispute-mon/mon/types/types.go`
- `op-dispute-mon/mon/types/honest_actors.go`
- `op-dispute-mon/mon/service.go`
- `op-dispute-mon/mon/node_endpoint_*.go`
- `op-dispute-mon/mon/mixed_safety.go`
- `op-dispute-mon/mon/extract/recipient_enricher.go`
- `op-dispute-mon/mon/extract/balance_enricher.go`
- `op-dispute-mon/mon/extract/caller.go`
