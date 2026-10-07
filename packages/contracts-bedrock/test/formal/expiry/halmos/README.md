# Halmos checks: per-message interop expiry

Symbolic checks, using Halmos 0.3.3, on the real contracts at 37b44c48c7:
`L2ToL2CrossDomainMessenger`, `L1CrossDomainMessenger` (with the `CrossDomainMessenger` code it inherits),
`SuperchainETHBridge`, `ETHLiquidity` and `SafeSend`. Each check is a statement about **one call, or a short fixed
sequence of calls, made from a symbolic state**. These checks do not cover multi-chain or multi-transaction
composition; that belongs to the Quint model, Kontrol and Lean, which cross-check against this suite.

## Files

| File | What it holds |
|---|---|
| `L2ToL2ExpiryHalmos.t.sol` (solc 0.8.25) | Groups (1) UnsafeTargetRule, (2) OnlyExportReachesL1, (3) export binding, (5) expireMessage, and the storage effects and frames of sendMessage and relayMessage. |
| `L1CDMExpiryHalmos.t.sol` (solc 0.8.15) | Group (4) relayUndeliveredMessage, and (7) the parts of L1 sender exclusivity that live in L1CrossDomainMessenger. |
| `RefundExpiryHalmos.t.sol` (solc 0.8.15) | Group (6) refundETH, and a composed sendETH → expire → refund check. |
| `HalmosMocks.sol` | L2-side mocks: the inbox, call recorders, L2CDM getters and a re-entrant relay target. |
| `expected.tsv` | The inventory: contract, check, expected outcome. `run.sh` fails on any deviation. |
| `run.sh` | Builds and runs everything, then validates against `expected.tsv`. |
| `mutants.sh` | 24 contract mutants. Each one must be killed by the checks designated for it. |
| `halmos-selfdestruct.patch` | Patch to halmos 0.3.3: SELFDESTRUCT in constructors, and MAX_ETH raised to 2^200. See below. |

## How to run

```
cd packages/contracts-bedrock
uv venv /tmp/halmos-sd --python 3.12
uv pip install --link-mode copy --python /tmp/halmos-sd/bin/python halmos==0.3.3   # copy mode: never patch uv's cache
patch -d /tmp/halmos-sd/lib/python3.12/site-packages -p1 < test/formal/expiry/halmos/halmos-selfdestruct.patch
HALMOS=/tmp/halmos-sd/bin/halmos test/formal/expiry/halmos/run.sh        # about 75 s on a 32-core box
HALMOS=/tmp/halmos-sd/bin/halmos test/formal/expiry/halmos/mutants.sh    # about 8 min; ONLY=<regex> selects mutants
```

Options and settings:
- **No foundry.toml change needed.** `run.sh` sets `FOUNDRY_SRC`, `FOUNDRY_TEST` and `FOUNDRY_SCRIPT` to this directory, with
  `FOUNDRY_OUT=halmos-out` and `FOUNDRY_CACHE_PATH=halmos-cache`.
- **Halmos flags:** `--loop 2 --solver-timeout-assertion 60s --default-bytes-lengths 0,1,32,33,100,132,260`. You can override
  the lengths with `BYTES_LENGTHS`. The solver is yices, the default.
- **Wider lengths:** the suite has also passed with `0,1,31,32,33,64,100,132,260,1024`.
- **Optional justfile recipe:** `test-halmos-expiry: ./test/formal/expiry/halmos/run.sh`.
- **What `run.sh` rejects:**
  - a missing, extra or empty result for any contract (for example after a setUp failure);
  - a halmos exit code other than 0 or 1;
  - a PASS check that halmos bounded with `--loop`;
  - an expected-FAIL check without a counterexample that halmos validated.
- **Stock halmos:** the refund checks ERROR (exit code 3), and the run fails.

## What each check proves

Notation: H(d, s, n, snd, tgt, msg) = `keccak256(abi.encode(d, s, n, snd, tgt, msg))`.
- **Symbolic means universally quantified.** Every `check_` parameter is symbolic, and so are block.chainid and block.timestamp
  where they are set from parameters.
- **"Symbolic storage"** means `svm.enableSymbolicStorage`: every slot starts with an arbitrary value. The statement then
  holds from every state, reachable or not.
- **Frame claims are made at symbolic keys** `k` (a hash) and `j` (a nonce), so "unchanged at k" means unchanged at every key.

### (1) UnsafeTargetRule (L2ToL2)

| Check | Statement | Expected |
|---|---|---|
| `check_UnsafeTargetRule_send` | Symbolic storage. If sendMessage succeeds, then target ∉ {0x..23, 0x..07} and destination ≠ chainid. | PASS |
| `check_UnsafeTargetRule_relay` | Symbolic storage, any non-harness target ≠ 0x..23. If relayMessage succeeds, then target ≠ 0x..07, id.origin = 0x..23 and destination = chainid. | PASS |
| `check_UnsafeTargetRule_relay_l2cdm` | Symbolic storage. A relay to 0x..07 always reverts. | PASS |
| `check_UnsafeTargetRule_send_passer_PENDING` / `_relay_passer_PENDING` | Send and relay to 0x..16 revert. This is the pending rule, not yet in 37b44c48c7. | FAIL |
| `check_INFO_relayDoesNotRejectTarget23` | relayMessage itself accepts target 0x..23. The relay checks exclude 0x..23 because every source chain's sendMessage rejects it. | FAIL |
| `check_FALSE_send_neverSucceeds` / `check_FALSE_relay_neverSucceeds` | Non-vacuity: the success paths are reachable. Relay succeeds to 0x0, 0x..22 and 0x..16. | FAIL |

### (2) OnlyExportReachesL1 (L2ToL2)

The recorders at 0x..07 and 0x..16 have only a fallback, so every call is counted whatever its selector.

| Check | Statement | Expected |
|---|---|---|
| `check_OnlyExportReachesL1_send` | Symbolic storage. sendMessage makes 0x..23 call neither 0x..07 nor 0x..16. | PASS |
| `check_OnlyExportReachesL1_relay_l2cdm` | A relay to a target that does not re-enter (codeless or predeploy mock) makes zero calls from 0x..23 to 0x..07. | PASS |
| `check_OnlyExportReachesL1_relay_reentrant` | Symbolic storage. The relay target is `ReentrantTarget`, which, during the relayed call, calls `exportUndeliveredMessage` (mode 1) or `sendMessage` (mode 2) on 0x..23 with symbolic arguments. **Every** call 0x..23 makes to 0x..07 in the whole execution (at most one) is exactly `sendMessage(sm', relayUndeliveredMessage(H', block.timestamp), g')`. Here H' = H(chainid, src', nonce', snd', tgt', msg') ≠ the hash being relayed, and `successfulMessages[H']` was false before the relay, so H' was unrelayed at the time of the call. There is no call to 0x..16. | PASS |
| `check_OnlyExportReachesL1_export_positive` | From fresh storage, export succeeds with exactly one call to 0x..07 and none to 0x..16. | PASS |
| `check_OnlyExportReachesL1_relay_passer_PENDING` | A relay never makes 0x..23 call 0x..16. This is the pending rule. | FAIL |
| `check_FALSE_export_neverCallsL2CDM`, `check_FALSE_relay_reentrantExportNeverReachesL2CDM` | Non-vacuity. | FAIL |

Together these give OnlyExportReachesL1 at the level of the L2ToL2 contract: **0x..23 calls 0x..07 only through
exportUndeliveredMessage's single call, with that exact payload**. That holds at top level, and re-entrantly from a
relay target that re-enters once. expireMessage only reads 0x..07 through view getters (STATICCALL).

### (3) exportUndeliveredMessage binding (L2ToL2)

| Check | Statement | Expected |
|---|---|---|
| `check_export_binding` | Symbolic storage. Export succeeds **iff** `!successfulMessages[H]`, where H = H(chainid, source, nonce, sender, target, message). On success it returns H and makes exactly one call from 0x..23 to 0x..07, whose calldata (checked by keccak and length) is exactly `sendMessage(sourceMessenger, relayUndeliveredMessage(H, block.timestamp), minGas)`. On revert it makes no call. In either case it writes no storage: nonce, sentMessages[j], and sentMessageTimestamps, successfulMessages and expiredMessages at k are unchanged, as is successfulMessages[H]. There is no call to 0x..16. | PASS |
| `check_FALSE_export_hashUsesSourceAsDestination` | Non-vacuity. | FAIL |

### Storage effects and frames (L2ToL2)

| Check | Statement | Expected |
|---|---|---|
| `check_send_effects_and_frame` | Symbolic storage. Let n = messageNonce() before the call. On success: it returns H = H(dest, chainid, n, msg.sender, target, message); `sentMessageTimestamps[H] == block.timestamp`; `sentMessages[n] == H`; `messageNonce() == n+1`; sentMessageTimestamps[k≠H] and sentMessages[j≠n] are unchanged; successfulMessages and expiredMessages are unchanged at k and at H. On revert, nothing changes at k, j, H or n. | PASS |
| `check_relay_effects_and_frame` | Symbolic storage, target does not re-enter. On success, `successfulMessages[H]` goes false → true, where H = H(chainid, id.chainId, nonce, sender, target, message). successfulMessages[k≠H] is unchanged, and nonce, sentMessages[j], sentMessageTimestamps[k] and expiredMessages[k] are unchanged. On revert, nothing changes. | PASS |

### (5) expireMessage (L2ToL2)

W is read from `MESSAGE_EXPIRY_WINDOW()` in the contract, so the planned 8-day constant needs only a recompile.

| Check | Statement | Expected |
|---|---|---|
| `check_expire_iff` | Symbolic storage and symbolic getter answers. Success **iff** `msg.sender == 0x..07 ∧ xDomainMessageSender == otherMessenger ∧ sentAt ≠ 0 ∧ t > sentAt + W`, where sentAt = sentMessageTimestamps[H]. **Assumes sentAt ≤ 2^64−1.** Frame: expiredMessages[H] becomes true on success and is unchanged on revert. sentMessageTimestamps[H], successfulMessages[H], nonce and sentMessages[j] are unchanged, as are all observations at every H2 ≠ H. | PASS |
| `check_expire_iff_unbounded` | The same iff with no bound on sentAt, plus the conjunct sentAt ≤ 2^256−1−W, because otherwise the checked add reverts. | PASS |
| `check_expire_boundary` | Authorized call, 0 < sentAt < 2^64: t = sentAt+W reverts and t = sentAt+W+1 succeeds. | PASS |
| `check_contractWindowCoversProtocolCap` | W ≥ 7 days, i.e. P_contract ≥ the protocol cap. | PASS |
| `check_FALSE_expire_windowIsGte`, `check_FALSE_expire_ignoresXDomainSender` | Non-vacuity. The first has its counterexample at t = sentAt+W. | FAIL |

### (4) relayUndeliveredMessage (L1CDM)

| Check | Statement | Expected |
|---|---|---|
| `check_relayUndelivered_iff_and_deposit` | Success **iff** no dependency reverts ∧ (a) ∧ (b) ∧ (c). The possibly reverting dependencies are caller.portal(), callerPortal.systemConfig(), sysCfg.l1CrossDomainMessenger(), portalA.ethLockbox(), lockbox.authorizedPortals(), caller.xDomainMessageSender() and portalA.depositTransaction(); each has a symbolic revert flag. The three checks are: (a) the caller's SystemConfig names the caller; (b) A's lockbox authorizes the caller's portal; (c) caller.xDomainMessageSender() == TRUSTED_EXPORTER (0x..23). All answers are symbolic. On success there is exactly one deposit, sent by A's L1CDM, with to = 0x..07, value 0, isCreation false, gasLimit = baseGas(expireMessage(H,t), 100000), and data = `relayMessage(messageNonce(), A's L1CDM, 0x..23, 0, 100000, expireMessage(H,t))`; the nonce is symbolic. On revert there is no deposit. | PASS |
| `check_relayUndelivered_rejectsCallerClaimingPortalA` | A contract that names A's own portal as its portal always fails (a), because A's SystemConfig names A's L1CDM. | PASS |
| `check_relayUndelivered_rejectsSelfCallOutsideRelay` | A's L1CDM calling itself while not relaying reverts: its xDomainMessageSender() reverts. | PASS |
| `check_FALSE_relayUndelivered_lockboxCheckRedundant`, `check_FALSE_relayUndelivered_neverDeposits` | Non-vacuity. | FAIL |

### (7) L1 sender exclusivity: the L1CrossDomainMessenger part

| Check | Statement | Expected |
|---|---|---|
| `check_L1_relayMessage_rejectsSelfAndPortalTargets` | relayMessage with target ∈ {A's L1CDM, A's portal} always reverts and never deposits. This holds for any caller (A's portal with any l2Sender, or anyone else), any symbolic failedMessages entry for the message, any message, and paused or not. | PASS |
| `check_L1_xDomainMessageSender_revertsOutsideRelay` | xDomainMessageSender() reverts when no message is being relayed. | PASS |
| `check_L1_xDomainMessageSender_isRelayedSender` | During a portal-delivered relay (l2Sender = 0x..07), the target observes xDomainMessageSender() == the message's `_sender`. Afterwards the getter reverts again. | PASS |
| `check_L1_sendMessage_senderFieldIsCaller` | sendMessage from any caller ≠ A's L1CDM deposits relayMessage(nonce, **caller**, target, 0, minGas, message). | PASS |
| `check_FALSE_L1_probeNeverCalled` | Non-vacuity: the relay target is reached. | FAIL |

What these checks show:
- A's L1CDM is the sender of an L1→L2 message only when it calls its own sendMessage.
- By code inspection, the only place it does that is `this.sendMessage` inside relayUndeliveredMessage. A relayed call
  to itself is ruled out by the first check.
- B's L1CDM reports xDomainMessageSender = X only while relaying a message whose `_sender` field is X.

**Delegated, not checked here:**
- L2CrossDomainMessenger encodes `msg.sender` as `_sender`. It runs the same `CrossDomainMessenger.sendMessage` code that
  `check_L1_sendMessage_senderFieldIsCaller` checks on L1; only `_sendMessage` (passer vs portal) differs.
- The portal delivers finalized withdrawals only, with `l2Sender` equal to the withdrawal's L2 sender.
- Withdrawal finality.
- The real OptimismPortal2, ETHLockbox and SystemConfig code.
- Composing B's export → L1 → A's expire end to end.

These belong to Kontrol, Lean and Quint, or to the brief's listed assumptions.

### (6) refundETH (bridge + ETHLiquidity + SafeSend)

| Check | Statement | Expected |
|---|---|---|
| `check_refund_iff_effects_singleUse` | Symbolic expired set and symbolic refunded map, symbolic balances. Let H = H(dest, chainid, nonce, bridge, bridge, relayETH(from,to,amount)). Success **iff** `expiredMessages[H] ∧ ¬refunded[H] ∧ amount ≤ liquidity balance`. On success: refunded[H] becomes true; `from` gains amount + p2; the bridge gains p1; ETHLiquidity loses amount; and the identical second call reverts. Here p1 and p2 are symbolic ETH pre-sent to ETHLiquidity's SafeSend and to the bridge's SafeSend, which forward their whole balance. On revert, no flag or balance changes. | PASS |
| `check_refund_frame` | refunded[H2] is unchanged for every H2 ≠ H. | PASS |
| `check_sendETH_then_refund` | Uses the real L2ToL2 at 0x..23 (loaded with `vm.getCode`). sendETH succeeds. Its hash equals refundETH's H for (dest, nonce = messageNonce() before, from, to, amount), and `sentMessageTimestamps[H] == block.timestamp`. Refund reverts before expiry. After `expiredMessages[H]` is set by storage write (expireMessage itself is group 5), the refund succeeds and pays `from` exactly `amount`. | PASS |
| `check_FALSE_refund_failsWhenExpired` | Non-vacuity: a successful refund is reachable. | FAIL |
| `check_FALSE_refund_paysTo` | Non-vacuity: the refund pays `to`. | FAIL |
| `check_FALSE_refund_ignoresPresentETH` | Documents why p2 is in the statement: the counterexample has p2 > 0. | FAIL |

**Preimage binding.** The expired set is an arbitrary map, so "success ⇒ expiredMessages[H]" can hold only if the
bridge queries exactly H. Mutants M18 (nonce+1) and M19 (sender = from) confirm this, and so does the composed check.

## Assumptions, mocks and bounds

**Mocks.** Every mock returns whatever the test or symbolic storage says, standing in for any answer the real contract
could give.
- `MockCrossL2Inbox` at 0x..22: validateMessage always succeeds. The identifier is assumed valid; validating it is the
  protocol's job, not the messenger's.
- `Recorder` at 0x..07 and 0x..16: fallback only. It counts **committed** calls from 0x..23 and hashes their calldata. A
  call that is later reverted leaves no trace; such a call also has no effect.
- `MockL2CDMGetters` at 0x..07, for group 5: xDomainMessageSender() and otherMessenger() are symbolic. The real getter
  reverts when unset, so the mock only adds behaviours.
- L1 mocks: every getter has a symbolic revert flag and returns symbolic values. The mocks' own addresses are fixed.
  A's SystemConfig names A's L1CDM.
- `MockExpired` at 0x..23 in the refund checks: expiredMessages is an arbitrary map.

**Relay payloads.** Only canonical, well-formed SentMessage encodings are built: the event selector plus symbolic fields.
Decoder rejection of malformed payloads is not checked.

**Relay targets.**
- Symbolic targets are assumed not to be harness accounts: the test contract, `vm`, the SVM address, console, the
  CREATE2 factory, and the template deployments whose code is etched at the predeploys.
- Covered targets: any codeless account, 0x..22, 0x..07, 0x..16, and `ReentrantTarget`.
- `ReentrantTarget` re-enters once, with export or send. Deeper nesting and other re-entry entry points are not
  explored. relayMessage is `nonReentrant`, and expireMessage requires msg.sender == 0x..07.
- Target 0x..23 is excluded from the relay checks; see the INFO check above.

**L1 topologies.**
- Explored:
  - distinct caller, caller portal, caller SystemConfig and A's portal;
  - a caller claiming A's portal;
  - a caller that is A's L1CDM, outside a relay;
  - A's L1CDM relaying to itself or to its portal;
  - every dependency reverting.
- Not explored:
  - caller.portal() returning arbitrary addresses that alias the lockbox, a SystemConfig, the caller itself or A's
    L1CDM;
  - malformed or short return data;
  - re-entry from a dependency getter;
  - the real portal, lockbox and SystemConfig code.

**Expiry.** `check_expire_iff` assumes sentAt ≤ 2^64−1, a block timestamp. `check_expire_iff_unbounded` drops that
assumption.

**Balances (refund).**
- Halmos prunes any path that reads a balance above MAX_ETH. MAX_ETH is 2^128 in stock halmos, which is below
  ETHLiquidity's 2^128−1 genesis balance plus deposits. The patch raises it to 2^200.
- The checks assume every symbolic balance, amount and pre-sent amount is ≤ 2^198. Sums of up to four of them then
  never reach the cap, so no path is silently pruned by it. Total ETH supply is about 2^87 wei.
- Insufficient liquidity is part of the iff. Halmos 0.3.3 branches on insufficient CREATE funds.

**`from` (refund).**
- `from` is assumed not to be the bridge or ETHLiquidity.
- `from` is assumed to lie outside halmos's fresh-address ranges, [0xaaaa0000, 0xaaaaffff] and [0xbbbb0000, 0xbbbbffff].
  This means `from` is not one of the SafeSend helpers created during the check. On a real chain those addresses are
  CREATE(bridge or ETHLiquidity, nonce), which only those two contracts can deploy to.
- Without this assumption, halmos finds the EIP-6780 burn counterexample, `from` = 0xaaaa0006. I checked this.

**SafeSend pre-funding.** The two SafeSend addresses are predicted as the next two CREATE addresses after a probe
deployment, and funded with symbolic p1 and p2. The prediction is confirmed by `check_FALSE_refund_ignoresPresentETH`,
which fails only because p2 lands at the bridge's SafeSend, and by the PASS check's `+ p2` term.

**keccak.** Halmos models keccak as an injective uninterpreted function. This matches the brief's collision-resistance
assumption.

**Expected-FAIL checks.**
- Each one contains exactly one assertion of its own.
- The only other assertions on their paths are shared slot or layout sanity asserts, and the PASS checks prove those
  never fail. So a counterexample means the intended assertion failed. Halmos 0.3.3 does not report which assert fired.

## The SELFDESTRUCT patch, and why it is sound for these checks

Stock halmos 0.3.3 halts any path that reaches SELFDESTRUCT. SafeSend's constructor is `selfdestruct(recipient)`, so
every refund success path needs the opcode. The patch does two things:

1. **SELFDESTRUCT inside a constructor frame (CREATE or CREATE2) only.** The account was created in this transaction,
   so EIP-6780's full semantics apply:
   - the whole balance moves to the beneficiary;
   - if the beneficiary is the account itself, the balance is **burned**;
   - the frame halts successfully with empty output, so no runtime code is deployed;
   - a static context is an error.

   The account is deleted at the end of the transaction, but it has empty code and no storage, and its balance is zero
   after the opcode. Deletion would only matter if ETH were sent to it later in the same transaction. That cannot
   happen here: the only later transfers go to the bridge or to `from`, and `from` is assumed outside the fresh range.

   SELFDESTRUCT outside a constructor still halts the path with an error, as in stock halmos. It is not modelled.
2. **MAX_ETH raised from 2^128 to 2^200.** See "Balances" above.

## Mutation check

`mutants.sh` covers 24 mutants. Each must make **all** of its designated checks FAIL with a valid counterexample. The
script exits nonzero on any survivor, any sed that does not apply, any halmos error or timeout, or any missing result.

| Mutant | Designated checks |
|---|---|
| M1 window `<=` → `<` | expire_iff, expire_iff_unbounded, expire_boundary |
| M2 relay accepts 0x..07 | UnsafeTargetRule_relay, UnsafeTargetRule_relay_l2cdm, OnlyExportReachesL1_relay_l2cdm |
| M3 / M3b send accepts 0x..07 / 0x..23 | UnsafeTargetRule_send |
| M4 export destination = source; M5 export skips the relayed check; M6 export sends t = 0; M15 export bumps the nonce | export_binding |
| M7 expire skips the xDomainMessageSender check | expire_iff, expire_iff_unbounded |
| M13 send records timestamp 1; M16 send skips sentMessages | send_effects_and_frame |
| M14 relay also sets expiredMessages | relay_effects_and_frame |
| M17 expire also sets successfulMessages | expire_iff |
| M24 relay does not mark the hash before the call | relay_effects_and_frame, OnlyExportReachesL1_relay_reentrant |
| M8 drop the lockbox check; M9 gas 100001 | relayUndelivered_iff_and_deposit |
| M20 L1 relays to itself | L1_relayMessage_rejectsSelfAndPortalTargets |
| M21 xDomainMsgSender not reset after the call | L1_xDomainMessageSender_isRelayedSender |
| M23 sender field = tx.origin | L1_sendMessage_senderFieldIsCaller, relayUndelivered_iff_and_deposit |
| M10 refund skips the refunded check; M12 refund does not mark | refund_iff_effects_singleUse |
| M11 refund pays `to`; M18 hash nonce+1; M19 hash sender = from | refund_iff_effects_singleUse, sendETH_then_refund |

The reviewers' combined mutant (send writes timestamp 1 plus relay sets expiredMessages) is covered by M13 and M14
separately.

## Pending design changes

As of the last poll, the tip of `karl/message-expiry-refunds` is still 37b44c48c7. Two changes are planned:

- **8-day expiry period.** W is read from the contract, so this needs only a recompile.
- **Exporter predeploy at 0x..2E plus an INTEROP gate.** When it lands I will add two checks:
  - the exporter only ever calls L2CDM.sendMessage, with the fixed payload, for all symbolic inputs (reusing
    `check_export_binding` and the Recorder);
  - relayUndeliveredMessage rejects 0x..23 and accepts only 0x..2E, together with checks (a) and (b) and the interop
    gate. That needs `TRUSTED_EXPORTER` updated and a gate flag on A's SystemConfig mock.

## Review log

Round 1, 2026-10-07. Reviewers: Claude, Codex gpt-6-astra and gpt-6.1-sol.

| # | Reviewer(s) | Finding | Disposition |
|---|---|---|---|
| 1 | all three | sendMessage's timestamp write is unchecked, and relay and export have no full storage frame. A combined mutant survived. | **Fixed.** Added `check_send_effects_and_frame` and `check_relay_effects_and_frame`, and full frames in `check_export_binding` and `check_expire_iff`, all at symbolic keys. Added mutants M13–M17 and M24, all killed. |
| 2 | all three | run.sh passes when checks never run, and mutants.sh does not enforce kills. | **Fixed.** Added the `expected.tsv` inventory; missing, extra or empty results fail, and so does a halmos exit code other than 0 or 1. Both cases were reproduced and are now rejected. mutants.sh now enforces kills per mutant. |
| 3 | astra, sol | The SELFDESTRUCT patch keeps the balance when the beneficiary is the account itself, which could give an unsound refund PASS. | **Fixed.** EIP-6780 burn for the self-beneficiary; constructor-only support; `from` assumed outside the fresh range (stated). Without that assumption halmos finds the burn counterexample. |
| 4 | astra, sol | Refund environment: pre-funded SafeSend, bridge balance, insufficient liquidity, MAX_ETH. | **Fixed.** Symbolic p1, p2 and bridge balance in the effect statement. Insufficient liquidity is now in the iff; my earlier pruning rationale was wrong for CREATE. MAX_ETH raised to 2^200, with an explicit 2^198 bound. |
| 5 | astra, sol, Claude | Relay callbacks: targets were codeless only. | **Fixed.** Added `ReentrantTarget` and `check_OnlyExportReachesL1_relay_reentrant`, with OnlyExportReachesL1 restated per call. Remaining limits (one re-entry, export or send) are stated above. |
| 6 | sol, Claude | The L1 group covers one topology. | **Partly fixed.** Added the claim-A's-portal, self-call, relay-to-self/portal and reverting-getter variants. The unexplored topologies are listed above. |
| 7 | Claude | Names read as end-to-end claims; L1 sender exclusivity is assumed. | **Fixed.** Added the (7) checks on the real L1CDM. What is delegated is listed above. The group names describe contract-level statements, defined in this file. |
| 8 | all three | Low: no composed send→refund check; expected-FAIL attribution; `hashBindsFromAsSender` fails trivially; recorder misses call-then-revert; byte lengths. | **Fixed or noted.** Added `check_sendETH_then_refund`. One own assert per expected-FAIL check. Replaced `hashBindsFromAsSender`: no FALSE check against an arbitrary map can tell implementations apart, so binding rests on the PASS iff plus M18 and M19. The recorder limit is noted. Default lengths are now 0,1,32,33,100,132,260. |
| 9 | integrator | Write a README. | This file. |
| 10 | integrator | Retarget to the exporter design. | Not landed yet; see "Pending design changes". |
