# Halmos checks: per-message interop expiry

Symbolic checks, using Halmos 0.3.3, on the real contracts of the exporter design (base PR #23259):
`L2ToL2CrossDomainMessenger`,
`UndeliveredMessageExporter`, `L1CrossDomainMessenger` and `L2CrossDomainMessenger` (with the `CrossDomainMessenger` code they inherit),
`SuperchainETHBridge`, `ETHLiquidity` and `SafeSend`. Each check is a statement about **one call, or a short fixed
sequence of calls, made from a symbolic state**. These checks do not cover multi-chain or multi-transaction
composition; that belongs to the Quint model, Kontrol and Lean, which cross-check against this suite.

Last full run (`run.sh` and `mutants.sh`): contracts at c7c51d79e2. The logic is unchanged at 448d31ad19, where only
the messenger's version string differs.

## Files

| File | What it holds |
|---|---|
| `L2ToL2ExpiryHalmos.t.sol` (solc 0.8.25) | Groups (1) UnsafeTargetRule, (2) OnlyExportReachesL1, (5) expireMessage; storage effects and frames of sendMessage and relayMessage; relay delivery, value, context and failure. |
| `ExporterExpiryHalmos.t.sol` (solc 0.8.15) | Group (3) on the real `UndeliveredMessageExporter`: export binding, plus "any calldata ⇒ only the export payload". |
| `L1CDMExpiryHalmos.t.sol` (solc 0.8.15) | Two test contracts. `L1CDMExpiryHalmos`: group (4) relayUndeliveredMessage, and (7) the parts of sender exclusivity that live in L1CrossDomainMessenger, including its relay gate. `L2CDMGateHalmos`: the same relay gate on the real L2CrossDomainMessenger, which expireMessage's authorization trusts. |
| `RefundExpiryHalmos.t.sol` (solc 0.8.15) | Group (6) refundETH, and a composed sendETH → expire → refund check. |
| `HalmosMocks.sol` | L2-side mocks: the inbox, call recorders, L2CDM getters, a re-entrant relay target and an observing (optionally reverting) relay target. |
| `expected.tsv` | The inventory: contract, check, expected outcome. `run.sh` fails on any deviation. |
| `run.sh` | Builds and runs everything, then validates against `expected.tsv`. |
| `halmos.toml` | Default halmos options (byte lengths, solver timeout). Function annotations override them; command-line flags override both. |
| `.gitignore` | Ignores the build output (`out/`, `cache/`) and the per-contract halmos logs and JSON that `run.sh` keeps in `results/`. |
| `mutants.sh` | 46 halmos mutants and 2 forge mutants. Each one must be killed by the checks designated for it. |
| `halmos-selfdestruct.patch` | Patch to halmos 0.3.3: SELFDESTRUCT in constructors, and MAX_ETH raised to 2^200. See below. |

## How to run

```
cd packages/contracts-bedrock
uv venv /tmp/halmos-sd --python 3.12
uv pip install --link-mode copy --python /tmp/halmos-sd/bin/python halmos==0.3.3   # copy mode: never patch uv's cache
patch -d /tmp/halmos-sd/lib/python3.12/site-packages -p1 < test/formal/expiry/halmos/halmos-selfdestruct.patch
HALMOS=/tmp/halmos-sd/bin/halmos test/formal/expiry/halmos/run.sh        # about 70 min on a 32-core Linux host (ReachL1CDM ~38 min, ReachL2ToL2 ~16 min)
HALMOS=/tmp/halmos-sd/bin/halmos test/formal/expiry/halmos/mutants.sh    # about 15 min; ONLY=<regex> selects mutants
# On a shared host, cap memory (and time) per halmos process:
#   HALMOS_WRAP="systemd-run --user --scope -p MemoryMax=16G -p MemorySwapMax=0 timeout 3600" ...
```

Options and settings:
- **No foundry.toml change needed.** `run.sh` sets `FOUNDRY_SRC`, `FOUNDRY_TEST` and `FOUNDRY_SCRIPT` to this directory, with
  `FOUNDRY_OUT` and `FOUNDRY_CACHE_PATH` pointing at `out/` and `cache/` in this directory, which `.gitignore` excludes.
- **Halmos options:** `halmos.toml` sets `--default-bytes-lengths 0,1,32,33,100,132,260` and `--solver-timeout-assertion 60s`;
  halmos's own default is `--loop 2`. Because command-line flags would override the per-function annotations, `run.sh`
  passes these through the config file. The phase-2 reachability checks use smaller length sets via annotations. The one exception is `check_exporter_anyCalldata_onlyExportPayload`, which sets `@custom:halmos --loop 40`
  so the word-by-word copies of its symbolic-length `bytes` argument (up to 1024 bytes) are explored in full. You can override
  the lengths with `BYTES_LENGTHS`. The solver is yices, the default.
- **Wider lengths:** the suite is also run with `0,1,31,32,33,64,100,132,260,1024`.
- **Optional justfile recipe:** `test-halmos-expiry: ./test/formal/expiry/halmos/run.sh`.
- **What `run.sh` rejects:**
  - a missing, extra or empty result for any contract (for example after a setUp failure);
  - a halmos exit code other than 0 or 1;
  - a PASS check that halmos bounded with `--loop`;
  - an expected-FAIL check without a counterexample that halmos validated;
  - any stuck path in any check, PASS or FAIL. Halmos reports a counterexample in preference to stuck, error or timeout
    paths, so the exit code alone cannot rule them out;
  - any WARNING, ERROR or TIMEOUT line in the halmos log, except the benign "unknown deployed bytecode" (SafeSend deploys
    empty code);
  - a `contract X is Test` in the `.t.sol` files that is missing from `expected.tsv`, or the reverse.
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
| `check_UnsafeTargetRule_send` | Symbolic storage. If sendMessage succeeds, then target ∉ {0x..23, 0x..07, 0x..16} and destination ≠ chainid. | PASS |
| `check_UnsafeTargetRule_relay` | Symbolic storage, any non-harness target ≠ 0x..23. If relayMessage succeeds, then target ∉ {0x..07, 0x..16}, id.origin = 0x..23 and destination = chainid. | PASS |
| `check_UnsafeTargetRule_relay_l2cdm` | Symbolic storage. A relay to 0x..07 always reverts. | PASS |
| `check_UnsafeTargetRule_send_passer` / `_relay_passer` | Send and relay to 0x..16 revert. This rule landed at the tip; in the earlier design these were `_PENDING` checks and failed. | PASS |
| `check_INFO_relayDoesNotRejectTarget23` | relayMessage itself accepts target 0x..23. The relay checks exclude 0x..23 because every source chain's sendMessage rejects it. | FAIL |
| `check_FALSE_send_neverSucceeds` / `check_FALSE_relay_neverSucceeds` | Non-vacuity: the success paths are reachable. Relay succeeds to 0x0 and 0x..22. | FAIL |

### (2) OnlyExportReachesL1 (L2ToL2 + exporter)

The recorders at 0x..07 and 0x..16 have only a fallback, so every call is counted whatever its selector. They count
calls from 0x..23 and from the exporter separately.

| Check | Statement | Expected |
|---|---|---|
| `check_OnlyExportReachesL1_send` | Symbolic storage. sendMessage makes 0x..23 call neither 0x..07 nor 0x..16. | PASS |
| `check_OnlyExportReachesL1_relay_l2cdm` / `_relay_passer` | A relay to a target that does not re-enter makes 0x..23 call 0x..07 zero times, and 0x..16 zero times. The passer half was `_PENDING` in the earlier design and is now PASS. | PASS |
| `check_OnlyExportReachesL1_relay_reentrant` | Symbolic storage. The relay target is `ReentrantTarget`. During the relayed call it either calls `exportUndeliveredMessage` on the **exporter** (mode 1) or re-enters `sendMessage` on 0x..23 (mode 2), with symbolic arguments. A third mode (mode 3) attempts a **nested relayMessage** of another well-formed message, and that must never succeed; this is the reentrancy guard itself (mutant M36 removes only `if (_entered()) revert`). 0x..23 makes **no** call to 0x..07 or 0x..16. The exporter makes at most one call to 0x..07, exactly `sendMessage(sm', relayUndeliveredMessage(H', block.timestamp), g')`, where H' ≠ the hash being relayed and `successfulMessages[H']` was false before, i.e. H' was unrelayed at that time. | PASS |
| `check_FALSE_relay_reentrantExportNeverReachesL2CDM` | Non-vacuity: the re-entrant export does reach 0x..07. | FAIL |

At the tip, OnlyExportReachesL1 is stronger than before: **0x..23 never calls 0x..07 or 0x..16**. Every withdrawal
the trusted sender, the exporter, initiates is the export payload (group 3).

### (3) UndeliveredMessageExporter (ExporterExpiryHalmos)

The exporter is the real contract at `Predeploys.UNDELIVERED_MESSAGE_EXPORTER`; its address is never hardcoded. At
0x..23, `successfulMessages` is an arbitrary symbolic mapping; the real getter is a plain mapping read.

| Check | Statement | Expected |
|---|---|---|
| `check_export_binding` | Any caller. Export succeeds **iff** `!successfulMessages[H]`, where H = H(chainid, source, nonce, sender, target, message). It returns H. On success it makes exactly one call, to 0x..07, whose calldata (checked by keccak) is exactly `sendMessage(sourceMessenger, relayUndeliveredMessage(H, block.timestamp), minGas)`; that is the only call 0x..07 receives. On revert, no call. Never a call to 0x..16. successfulMessages is unchanged at H and at k. | PASS |
| `check_exporter_anyCalldata_onlyExportPayload` | Calldata from `svm.createCalldata("UndeliveredMessageExporter")`, from any caller. That means every non-view function, with canonical ABI encodings, symbolic arguments and `bytes` lengths from the configured set; it does not mean every byte string. Every call the exporter makes to 0x..07 (at most one) is the export payload for the decoded arguments, and it never calls 0x..16. **Observation scope:** only calls to 0x..07 and 0x..16 are recorded. A call to any other address, which would be a codeless account in the harness, is not observed. By inspection the exporter has exactly one external call besides the `successfulMessages` read. | PASS |
| `check_FALSE_export_neverCallsL2CDM`, `check_FALSE_export_hashUsesSourceAsDestination` | Non-vacuity. | FAIL |

### Storage effects and frames (L2ToL2)

| Check | Statement | Expected |
|---|---|---|
| `check_send_effects_and_frame` | Symbolic storage. Let n = messageNonce() before the call. On success: it returns H = H(dest, chainid, n, msg.sender, target, message); `sentMessageTimestamps[H] == block.timestamp`; `sentMessages[n] == H`; `messageNonce() == n+1`; sentMessageTimestamps[k≠H] and sentMessages[j≠n] are unchanged; successfulMessages and expiredMessages are unchanged at k and at H. On revert, nothing changes at k, j, H or n. | PASS |
| `check_relay_delivery_value_context_failure` | Symbolic storage, symbolic msg.value, and an observing target (`RelayProbe`, fallback only) that either returns or reverts. **Success ⇒** the target did not revert, ran with exactly msg.value (its balance is msg.value), and during the call `crossDomainMessageContext()` returned (sender, id.chainId); successfulMessages[H] is set. **Target reverts ⇒** the relay reverts and successfulMessages[H] is unchanged (the message is not consumed). **Liveness:** a non-reverting target, origin 0x..23 and an unrelayed H ⇒ the relay succeeds. **Afterwards:** the context getter reverts, so the entered flag is cleared, and a second relay in the same transaction succeeds. The sender and source transient slots are also reset, but that cannot be observed through the interface, because every context getter is `onlyEntered`. | PASS |
| `check_FALSE_relay_revertingTargetStillSucceeds` | Non-vacuity: with a valid origin and fresh storage, the target's revert is the only possible cause of failure. | FAIL |
| `check_relay_effects_and_frame` | Symbolic storage, target does not re-enter. On success, `successfulMessages[H]` goes false → true, where H = H(chainid, id.chainId, nonce, sender, target, message). successfulMessages[k≠H] is unchanged, and nonce, sentMessages[j], sentMessageTimestamps[k] and expiredMessages[k] are unchanged. On revert, nothing changes. | PASS |

### (5) expireMessage (L2ToL2)

W is read from `EXPIRY_PERIOD()` in the contract, which is 8 days at the tip: the 7-day protocol cap plus a 1-day margin.

| Check | Statement | Expected |
|---|---|---|
| `check_expire_iff` | Symbolic storage and symbolic getter answers. Success **iff** `msg.sender == 0x..07 ∧ xDomainMessageSender == otherMessenger ∧ sentAt ≠ 0 ∧ t > sentAt + W`, where sentAt = sentMessageTimestamps[H]. **Assumes sentAt ≤ 2^64−1.** Frame: expiredMessages[H] becomes true on success and is unchanged on revert. sentMessageTimestamps[H], successfulMessages[H], nonce and sentMessages[j] are unchanged, as are all observations at every H2 ≠ H. | PASS |
| `check_expire_iff_unbounded` | The same iff with no bound on sentAt, plus the conjunct sentAt ≤ 2^256−1−W, because otherwise the checked add reverts. | PASS |
| `check_expire_boundary` | Authorized call, 0 < sentAt < 2^64: t = sentAt+W reverts and t = sentAt+W+1 succeeds. | PASS |
| `check_contractWindowCoversProtocolCap` | W ≥ 7 days, i.e. P_contract ≥ W_protocol, the cap op-core and kona enforce. | PASS |
| `check_expiryPeriodIsCapPlusMargin` | W == 7 days + 1 day. | PASS |
| `check_FALSE_expire_windowIsGte`, `check_FALSE_expire_ignoresXDomainSender` | Non-vacuity. The first has its counterexample at t = sentAt+W. | FAIL |

### (4) relayUndeliveredMessage (L1CDM)

| Check | Statement | Expected |
|---|---|---|
| `check_relayUndelivered_iff_and_deposit` | Success **iff** A's SystemConfig has INTEROP enabled (symbolic) ∧ no dependency reverts ∧ (a) ∧ (b) ∧ (c). Gas is compared with an independent restatement of the baseGas formula. The possibly reverting dependencies are caller.portal(), callerPortal.systemConfig(), sysCfg.l1CrossDomainMessenger(), portalA.ethLockbox(), lockbox.authorizedPortals(), caller.xDomainMessageSender() and portalA.depositTransaction(); each has a symbolic revert flag. The three checks are: (a) the caller's SystemConfig names the caller; (b) A's lockbox authorizes the caller's portal; (c) caller.xDomainMessageSender() == `Predeploys.UNDELIVERED_MESSAGE_EXPORTER`. All answers are symbolic. On success there is exactly one deposit, sent by A's L1CDM, with to = 0x..07, value 0, isCreation false, gasLimit = baseGas(expireMessage(H,t), 100000), and data = `relayMessage(messageNonce(), A's L1CDM, 0x..23, 0, 100000, expireMessage(H,t))`; the nonce is symbolic. On revert there is no deposit. | PASS |
| `check_relayUndelivered_rejectsL2ToL2AsSender` | A caller whose xDomainMessageSender is 0x..23 is rejected, even when interop, (a), (b) and no-revert all hold. 0x..23 is no longer trusted. | PASS |
| `check_FALSE_relayUndelivered_interopGateRedundant` | Non-vacuity: the INTEROP gate matters. | FAIL |
| `check_relayUndelivered_rejectsCallerClaimingPortalA` | A contract that names A's own portal as its portal always fails (a), because A's SystemConfig names A's L1CDM. | PASS |
| `check_relayUndelivered_rejectsSelfCallOutsideRelay` | A's L1CDM calling itself while not relaying reverts: its xDomainMessageSender() reverts. | PASS |
| `check_FALSE_relayUndelivered_lockboxCheckRedundant`, `check_FALSE_relayUndelivered_neverDeposits` | Non-vacuity. | FAIL |

### (7) Sender exclusivity and the CrossDomainMessenger relay gates (L1CDM, L2CDM)

`GateIn` makes every input symbolic:
- whether the caller is the portal (on L1) or the aliased otherMessenger (on L2), or else any other address;
- `portal.l2Sender()`;
- `failedMessages[vh]` and `successfulMessages[vh]` before the call;
- paused (L1);
- nonce, `_sender` (the exporter and 0x..23 included) and `_value`;
- msg.value: equal to `_value` for portal or aliased delivery, arbitrary for any other caller;
- the messenger's own balance.

Relay targets are `GateProbe`, which has only a fallback and records the ETH it receives. It reads
`xDomainMessageSender()` through a low-level call, so a revert of that getter is recorded, not hidden.

| Check | Statement | Expected |
|---|---|---|
| `check_L1_relayGate_and_delivery` | Real L1CDM. The target runs **only if** not paused ∧ not already successful ∧ ((caller == portal ∧ portal.l2Sender() == otherMessenger) ∨ failedMessages[vh]). Whenever it runs, it receives exactly `_value`, and `xDomainMessageSender()` succeeds and returns exactly `_sender`. **No skipped delivery:** if the message becomes successful, the target ran. **No forged replay eligibility:** an unauthorized attempt changes neither failedMessages[vh] nor successfulMessages[vh]. An unauthorized attempt is anything other than an authentic first delivery with msg.value == `_value` or a replay of an already-failed message; this includes the portal with msg.value ≠ `_value`. vh is computed with an independent formula (`specVersionedHash`), not the production `Hashing`. Afterwards `xDomainMessageSender()` reverts again. | PASS |
| `check_L1_replayNeedsExactFailedEntry` / `check_L2_replayNeedsExactFailedEntry` | When only a message differing in one field (sender, value, nonce, minGasLimit or target) is marked failed, this message cannot be replayed and none of its flags change. | PASS |
| `check_L1_failedEntryNotReplayableWithAlteredField` | Two steps. Step 1: the portal delivers authentically to a failing target, so the **messenger itself** records the failure under its own key. Step 2: anyone replays with one field altered (sender, value, nonce or minGasLimit). The target never runs. This also catches a message hash that stopped binding a field (mutant M37). | PASS |
| `check_FALSE_L1_failedMessageNeverReplayable` | Non-vacuity: the exact replay of such a message does run. | FAIL |
| `check_L1_relayMessage_rejectsSelfAndPortalTargets` | relayMessage with target ∈ {A's L1CDM, A's portal} always reverts and never deposits, for every input above. | PASS |
| `check_L1_xDomainMessageSender_revertsOutsideRelay` | xDomainMessageSender() reverts when no message is being relayed. | PASS |
| `check_L1_sendMessage_senderFieldIsCaller` | sendMessage from any caller ≠ A's L1CDM, with any msg.value, makes exactly one deposit, sent by A's L1CDM. It carries msg.value, goes to 0x..07, uses gas = baseGas computed **independently** (the formula is restated in the test), and its data is relayMessage(nonce, **caller**, target, msg.value, minGas, message). | PASS |
| `check_L2_relayGate_and_delivery` | Real L2CrossDomainMessenger at 0x..07, with otherMessenger = A's L1CDM. The target runs **only if** not already successful ∧ (caller == alias(otherMessenger) ∨ failedMessages[vh]). It then receives exactly `_value` and sees `xDomainMessageSender() == _sender`. No skipped delivery, no forged replay eligibility (same frame as on L1), and the getter reverts afterwards. So expireMessage's check (msg.sender == 0x..07 ∧ xDomainMessageSender == otherMessenger) holds only while 0x..07 relays a message whose `_sender` field is A's L1CDM, delivered by the aliased A's L1CDM or replayed after failing. | PASS |
| `check_L2_relayMessage_rejectsSelfAndPasser` | The real L2CDM never relays to itself or to 0x..16, for every input above. | PASS |
| `check_FALSE_L1_probeNeverCalled`, `check_FALSE_L2_probeNeverCalled` | Non-vacuity: an authentic delivery reaches the target. | FAIL |

What these checks show:
- A's L1CDM is the `_sender` of an L1→L2 message only when it calls its own sendMessage.
- By code inspection, the only place it does that is `this.sendMessage` inside relayUndeliveredMessage. A relayed call
  to itself is ruled out by the second check.
- B's L1CDM reports xDomainMessageSender = the exporter only while relaying, through the gate, a message whose
  `_sender` field is the exporter, i.e. a withdrawal the exporter initiated. That is check (c)'s trust basis. By
  group 3, such a withdrawal carries only the export payload.
- On A, the L2CDM shows otherMessenger as the sender only for a message with that `_sender`, delivered by the aliased
  L1CDM.

**Delegated, not checked here:**
- **Event binding.** Halmos 0.3.3 has no `vm.recordLogs`, so the binding between the hash sendMessage returns and stores
  and the `SentMessage` event the destination relays is not checked here. The existing forge test
  `test/L2/L2ToL2CrossDomainMessenger.t.sol::testFuzz_sendMessage_succeeds` checks every event field. `mutants.sh` runs
  it against mutants X1 (event nonce+1) and X1b (event sender = tx.origin) and requires it to fail.
- **Portal and L2 sender encoding.**
  - L2CrossDomainMessenger encodes `msg.sender` as `_sender`. This is the same `CrossDomainMessenger.sendMessage` code
    that `check_L1_sendMessage_senderFieldIsCaller` checks on L1; only `_sendMessage` (passer vs portal) differs.
  - The portal delivers finalized withdrawals only, with `l2Sender` equal to the withdrawal's L2 sender.
  - The portal aliases the L1 caller as the deposit's L2 `from`.
  - Withdrawal finality.
  - The real OptimismPortal2, ETHLockbox and SystemConfig code.
- **No relay after export.** Inbox and protocol enforcement of the window (exec − init ≤ W_protocol), together with
  cross-chain clock agreement, is what makes an exported message unrelayable. The Lean and Quint protocol models and the
  window differential discharge that, not this suite.
- **End-to-end composition** of B's export → L1 → A's expire.

These belong to Kontrol, Lean, Quint and the forge test, or to the brief's listed assumptions.

### (6) refundETH (bridge + ETHLiquidity + SafeSend)

| Check | Statement | Expected |
|---|---|---|
| `check_refund_iff_effects_singleUse` | Symbolic expired set and symbolic refunded map, symbolic balances. Let H = H(dest, chainid, nonce, bridge, bridge, relayETH(from,to,amount)). Success **iff** `expiredMessages[H] ∧ ¬refunded[H] ∧ amount ≤ liquidity balance`. On success: refunded[H] becomes true; `from` gains amount + p2; the bridge gains p1; ETHLiquidity loses amount; and the identical second call reverts. Here p1 and p2 are symbolic ETH pre-sent to ETHLiquidity's SafeSend and to the bridge's SafeSend, which forward their whole balance. On revert, no flag or balance changes. | PASS |
| `check_refund_frame` | refunded[H2] is unchanged for every H2 ≠ H. | PASS |
| `check_sendETH_then_refund` | Uses the real L2ToL2 at 0x..23 (loaded with `vm.getCode`), with a **symbolic prior nonce** < 2^240−1. sendETH succeeds. Its hash equals refundETH's H for (dest, nonce = messageNonce() before, from, to, amount), and `sentMessageTimestamps[H] == block.timestamp`. Refund reverts before expiry. After `expiredMessages[H]` is set by storage write (expireMessage itself is group 5), the refund succeeds and pays `from` exactly `amount`. | PASS |
| `check_FALSE_refund_failsWhenExpired` | Non-vacuity: a successful refund is reachable. | FAIL |
| `check_FALSE_refund_paysTo` | Non-vacuity: the refund pays `to`. Here `to` ≠ `from`, `to` is a realistic address (not a SafeSend helper, the bridge or ETHLiquidity), and amount > 0, so the only way to fail is that `from` is paid. | FAIL |
| `check_FALSE_refund_ignoresPresentETH` | Documents why p2 is in the statement: the counterexample has p2 > 0. | FAIL |

**Preimage binding.** The expired set is an arbitrary map, so "success ⇒ expiredMessages[H]" can hold only if the
bridge queries exactly H. Mutants M18 (nonce+1) and M19 (sender = from) confirm this, and so does the composed check.

## Reachability (phase 2): whole-contract properties over call sequences

These checks target each contract as deployed, behind the real `Proxy` with admin = the L2 ProxyAdmin, at its
predeploy address. The L1 messenger is the exception: it is called directly, without its `ResolvedDelegateProxy`, which
only forwards. Each check runs a SEQUENCE of steps. A step is one call from a symbolic caller, with symbolic msg.value
and a non-decreasing symbolic timestamp, choosing symbolically among every state-changing entry point, unknown
selectors (the proxy's own `upgradeTo`, `admin`, ... included, since a non-admin call to them is forwarded) and,
where relevant, empty calldata. The properties are checked after **every** step.

**Callers.** Not the ProxyAdmin, by the governance assumption. Not address(0), which only `eth_call` can be. Not the
contract under test or its implementation, which act only through their own code. Predeploys that legitimately call
the contract are allowed: 0x..07 for expireMessage and 0x..23 for relayETH, with symbolic oracle answers.

**View functions** are not steps, because the compiler forbids state writes in them.

| Check | Statement | Expected |
|---|---|---|
| `ReachL2ToL2Halmos.check_reach_sequence2` | Two steps from the deployed state, so every state is reachable; three steps exceeded 40 minutes. Relay targets are a codeless account, 0x..07 or 0x..16; arbitrary relay targets are covered by the L2ToL2ExpiryHalmos relay checks. Two steps already cover send-then-expire and send-then-relay. At a symbolic hash K: **(E)** expiredMessages[K] never goes true → false. It goes false → true only in an expireMessage step with msg.sender == 0x..07, xDomainMessageSender == otherMessenger, h == K, sentAt ≠ 0, sentAt ≤ 2^256−1−EXPIRY_PERIOD (so a wrapped sum is a counterexample, not a dropped panic) and t > sentAt + EXPIRY_PERIOD. **(T)** sentMessageTimestamps[K] changes only in a sendMessage step that sends K, from 0 to block.timestamp, so it is never decreased or cleared. **(S)** successfulMessages[K] changes only in a relayMessage step that relays K, from false to true. Unknown selectors always revert. | PASS |
| `ReachL2ToL2Halmos.check_reach_step_symbolicStorage` | The same for one step from **fully symbolic** storage. (T) is weakened to "only a sendMessage step sending K, to block.timestamp", because an unreachable state can already hold a value for the next nonce's hash. | PASS |
| `ReachExporterHalmos.check_reach_exporter_sequence2` | Two steps over export (symbolic arguments and message), version(), unknown selectors and empty calldata; three steps exceeded 40 minutes. successfulMessages is fully symbolic, so the first step already starts from every messenger state. Every call the exporter makes to 0x..07 is exactly the export payload for that step's arguments, with H computed with block.chainid and block.timestamp, and only when `successfulMessages[H]` was false before the step. It never calls 0x..16. The proxy slots and slots 0..3 never change; the implementation has no state variables. | PASS |
| `ReachBridgeHalmos.check_reach_bridge_step` | One step from fully symbolic bridge and messenger storage, which covers every state, reachable or not; two- and three-step sequences exceeded 40 minutes, and every property here is a one-step transition property. The step is over the bridge (sendETH, relayETH, refundETH) **and** ETHLiquidity (burn, fund, mint), plus unknown selectors. refunded[K] never goes true → false, and goes false → true only in refundETH whose arguments hash to K with expiredMessages[K]. ETHLiquidity's balance decreases, i.e. a mint, only in relayETH called by 0x..23 with context sender == the bridge, or in refundETH, and by exactly the amount. With three steps the run exceeded 20 minutes. | PASS |
| `ReachL1CDMHalmos.check_reach_l1cdm_sequence2` | Two steps, each one of: sendMessage, relayMessage (any caller including A's portal; target a codeless account, A's L1CDM or A's portal, because arbitrary relay targets are the gate checks' job), relayUndeliveredMessage (any caller, including the mock messengers through symbolic aliasing), initialize, or an unknown selector, all with symbolic arguments. Three steps exceeded 40 minutes, and the earlier `createCalldata` form got stuck on symbolic offsets. A's L1CDM is the **envelope sender** of a deposit only in a relayUndeliveredMessage step. A sendMessage step's deposit carries that step's caller as the sender. Nothing else deposits, and there is at most one deposit per step. | PASS |
| `check_FALSE_reach_*` (one per contract) | Non-vacuity: expiry is reached, the exporter does call 0x..07, liquidity is minted, and the L1CDM is the self-sender. | FAIL |

## Non-vacuity

Every PASS check is paired with a **witness**: an expected-FAIL `check_FALSE_*` check of the same test contract, under
the same assumptions. Its validated counterexample shows that the branch the PASS check constrains is reachable: the
success path, the delivery, the transition, the deposit. The pairs are the 4th column of `expected.tsv`. `run.sh`
fails if a PASS check names no witness, if the witness is not an expected-FAIL check of the same contract, or if the
witness does not produce a validated counterexample in the same run.

A PASS check that asserts "always reverts" (the `reject*` checks) is paired with the witness for the success path of
the same entry point under the same harness. Its counterexample shows that the harness can reach success, so the
revert is not an artefact of the setup. The two constant checks on `EXPIRY_PERIOD` are paired with a FALSE check on the
constant's value.

| Contract | PASS check | Witness |
|---|---|---|
| `L2ToL2ExpiryHalmos` | `check_UnsafeTargetRule_send` | `check_FALSE_send_neverSucceeds` |
| `L2ToL2ExpiryHalmos` | `check_UnsafeTargetRule_send_passer` | `check_FALSE_send_neverSucceeds` |
| `L2ToL2ExpiryHalmos` | `check_UnsafeTargetRule_relay` | `check_FALSE_relay_neverSucceeds` |
| `L2ToL2ExpiryHalmos` | `check_UnsafeTargetRule_relay_l2cdm` | `check_FALSE_relay_neverSucceeds` |
| `L2ToL2ExpiryHalmos` | `check_UnsafeTargetRule_relay_passer` | `check_FALSE_relay_neverSucceeds` |
| `L2ToL2ExpiryHalmos` | `check_OnlyExportReachesL1_send` | `check_FALSE_send_neverSucceeds` |
| `L2ToL2ExpiryHalmos` | `check_OnlyExportReachesL1_relay_l2cdm` | `check_FALSE_relay_neverSucceeds` |
| `L2ToL2ExpiryHalmos` | `check_OnlyExportReachesL1_relay_passer` | `check_FALSE_relay_neverSucceeds` |
| `L2ToL2ExpiryHalmos` | `check_send_effects_and_frame` | `check_FALSE_send_neverSucceeds` |
| `L2ToL2ExpiryHalmos` | `check_relay_effects_and_frame` | `check_FALSE_relay_neverSucceeds` |
| `L2ToL2ExpiryHalmos` | `check_OnlyExportReachesL1_relay_reentrant` | `check_FALSE_relay_reentrantExportNeverReachesL2CDM` |
| `L2ToL2ExpiryHalmos` | `check_relay_delivery_value_context_failure` | `check_FALSE_relay_probeNeverRuns` |
| `L2ToL2ExpiryHalmos` | `check_expire_iff` | `check_FALSE_expire_neverSucceeds` |
| `L2ToL2ExpiryHalmos` | `check_expire_iff_unbounded` | `check_FALSE_expire_neverSucceeds` |
| `L2ToL2ExpiryHalmos` | `check_expire_boundary` | `check_FALSE_expire_windowIsGte` |
| `L2ToL2ExpiryHalmos` | `check_contractWindowCoversProtocolCap` | `check_FALSE_expiryPeriodIsProtocolCap` |
| `L2ToL2ExpiryHalmos` | `check_expiryPeriodIsCapPlusMargin` | `check_FALSE_expiryPeriodIsProtocolCap` |
| `ExporterExpiryHalmos` | `check_export_binding` | `check_FALSE_export_neverCallsL2CDM` |
| `ExporterExpiryHalmos` | `check_exporter_anyCalldata_onlyExportPayload` | `check_FALSE_exporter_anyCalldata_neverCalls` |
| `L1CDMExpiryHalmos` | `check_relayUndelivered_iff_and_deposit` | `check_FALSE_relayUndelivered_neverDeposits` |
| `L1CDMExpiryHalmos` | `check_relayUndelivered_rejectsL2ToL2AsSender` | `check_FALSE_relayUndelivered_neverDeposits` |
| `L1CDMExpiryHalmos` | `check_relayUndelivered_rejectsCallerClaimingPortalA` | `check_FALSE_relayUndelivered_neverDeposits` |
| `L1CDMExpiryHalmos` | `check_relayUndelivered_rejectsSelfCallOutsideRelay` | `check_FALSE_relayUndelivered_neverDeposits` |
| `L1CDMExpiryHalmos` | `check_L1_relayMessage_rejectsSelfAndPortalTargets` | `check_FALSE_L1_probeNeverCalled` |
| `L1CDMExpiryHalmos` | `check_L1_relayGate_and_delivery` | `check_FALSE_L1_probeNeverCalled` |
| `L1CDMExpiryHalmos` | `check_L1_replayNeedsExactFailedEntry` | `check_FALSE_L1_failedMessageNeverReplayable` |
| `L1CDMExpiryHalmos` | `check_L1_failedEntryNotReplayableWithAlteredField` | `check_FALSE_L1_failedMessageNeverReplayable` |
| `L1CDMExpiryHalmos` | `check_L1_xDomainMessageSender_revertsOutsideRelay` | `check_FALSE_L1_probeNeverCalled` |
| `L1CDMExpiryHalmos` | `check_L1_sendMessage_senderFieldIsCaller` | `check_FALSE_L1_sendMessage_neverDeposits` |
| `L2CDMGateHalmos` | `check_L2_relayGate_and_delivery` | `check_FALSE_L2_probeNeverCalled` |
| `L2CDMGateHalmos` | `check_L2_relayMessage_rejectsSelfAndPasser` | `check_FALSE_L2_probeNeverCalled` |
| `L2CDMGateHalmos` | `check_L2_replayNeedsExactFailedEntry` | `check_FALSE_L2_exactReplayNeverRuns` |
| `RefundExpiryHalmos` | `check_refund_iff_effects_singleUse` | `check_FALSE_refund_failsWhenExpired` |
| `RefundExpiryHalmos` | `check_refund_frame` | `check_FALSE_refund_failsWhenExpired` |
| `RefundExpiryHalmos` | `check_sendETH_then_refund` | `check_FALSE_refund_failsWhenExpired` |
| `ReachL2ToL2Halmos` | `check_reach_sequence2` | `check_FALSE_reach_expiredNeverSet` |
| `ReachL2ToL2Halmos` | `check_reach_step_symbolicStorage` | `check_FALSE_reach_step_noTransition` |
| `ReachExporterHalmos` | `check_reach_exporter_sequence2` | `check_FALSE_reach_exporterNeverCalls` |
| `ReachBridgeHalmos` | `check_reach_bridge_step` | `check_FALSE_reach_liquidityNeverMints` |
| `ReachL1CDMHalmos` | `check_reach_l1cdm_sequence2` | `check_FALSE_reach_l1cdmNeverSelfSender` |

## Assumptions, mocks and bounds

**Mocks.** Every mock returns whatever the test or symbolic storage says, standing in for any answer the real contract
could give.
- `MockCrossL2Inbox` at 0x..22: validateMessage always succeeds. The identifier is assumed valid; validating it is the
  protocol's job, not the messenger's.
- `Recorder` / `CallRecorder` at 0x..07 and 0x..16: fallback only. They count **committed** calls from 0x..23 and from the
  exporter, and hash their calldata. A
  call that is later reverted leaves no trace; such a call also has no effect.
- `MockL2CDMGetters` at 0x..07, for group 5: xDomainMessageSender() and otherMessenger() are symbolic. The real getter
  reverts when unset, so the mock only adds behaviours.
- L1 mocks: every getter has a symbolic revert flag and returns symbolic values. The mocks' own addresses are fixed.
- Attacker-controlled getters on the OTHER chain's contracts, which the real code must not consult: the caller's own
  `systemConfig()` (answered by `AttackerSystemConfig`) and the caller portal's `ethLockbox()` (`AttackerLockbox`).
  They never revert and give fully symbolic answers, so a mutant that consults them (K41, K42) is tested against a
  forged answer, not killed by a missing getter.
  A's SystemConfig names A's L1CDM. `paused()` is symbolic in the gate checks.
- Probes (`RelayProbe`, `GateProbe`) have only a fallback and receive, and are read with `vm.load`. No relayed selector
  can hit a getter and bypass the recorder.
- `MockExpired` at 0x..23 in the refund checks: expiredMessages is an arbitrary map.

**Relay payloads.** Only canonical, well-formed SentMessage encodings are built: the event selector plus symbolic fields.
Decoder rejection of malformed payloads is not checked.

**Export iff, ⇐ direction.** `!successfulMessages[H] ⇒ export succeeds` assumes 0x..07 accepts the call: the recorder
never reverts. The real L2CrossDomainMessenger.sendMessage reverts only if the L2ToL1MessagePasser does. The ⇒ direction
and the payload do not depend on this.

**Relay targets.**
- Symbolic targets are assumed not to be harness accounts: the test contract, `vm`, the SVM address, console, the
  CREATE2 factory, and the template deployments whose code is etched at the predeploys.
- Covered targets: any codeless account, 0x..22, 0x..07, 0x..16, `ReentrantTarget`, and `RelayProbe` (observing or reverting, with symbolic msg.value).
- `ReentrantTarget` re-enters once, with export or send. Deeper nesting and other re-entry entry points are not
  explored. relayMessage is `nonReentrant`, and expireMessage requires msg.sender == 0x..07.
- Target 0x..23 is excluded from the relay checks; see the INFO check above.
- The exporter is excluded as a **symbolic** relay target. It ignores msg.sender, so a relay that calls it is an export
  with caller 0x..23, and that is covered by `ExporterExpiryHalmos` (symbolic caller) and by the re-entrant check.
  Calling it with symbolic calldata here would only make halmos stuck on symbolic ABI offsets.

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

**Governance (named assumption).** Each cluster chain's L2 governance, i.e. its L2 ProxyAdmin owner, can upgrade its
own UndeliveredMessageExporter. An upgraded exporter could forge undelivered-message facts for any destination. This
is the same trust as the shared ETHLockbox: lockbox portals must share the proxy admin owner. These checks cover the
exporter code as deployed; they say nothing about an upgraded exporter.

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

**Expected-FAIL checks and solver timeouts.**
- Halmos 0.3.3 reports a counterexample in preference to solver timeouts on other assertion queries of the same check,
  and it does not serialize those timeouts. So for an expected-FAIL check, `run.sh` cannot rule out a timeout on some
  other path; the reported counterexample is still valid. PASS checks are fully enforced, with no timeouts, stuck
  paths or bounded loops.
- Each expected-FAIL check contains exactly one assertion of its own.
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

Before each mutant, `mutants.sh` runs `forge build --force`. Test contracts embed creation code (`new X()`), and an
incremental build can miss them; in a copied tree it did, and 16 of a reviewer's mutants falsely survived until a clean
build. For the forge mutants, the designated test must first PASS on the unmutated code. Under the mutant it must then
fail itself after a successful `setUp`; a setUp or compile failure does not count as a kill.

`mutants.sh` covers 46 halmos mutants and 2 forge mutants. Each must make **all** of its designated checks FAIL with a valid counterexample, and the
halmos process must exit 1. Stuck paths are tolerated for mutants only: a mutant can open code halmos cannot finish.
For example, M20 lets the messenger call itself with symbolic calldata. The script exits nonzero on any survivor, any sed that does not
apply, any other exit status, any halmos error or timeout, or any missing result.

| Mutant | Designated checks |
|---|---|
| M1 window `<=` → `<` (EXPIRY_PERIOD) | expire_iff, expire_iff_unbounded, expire_boundary |
| M2 relay skips the unsafe-target check | UnsafeTargetRule_relay, _relay_l2cdm, _relay_passer, OnlyExportReachesL1_relay_l2cdm, _relay_passer |
| M3 send skips the unsafe-target check; M3b send accepts 0x..23 | UnsafeTargetRule_send (and _send_passer for M3) |
| M3c the unsafe set drops 0x..16 | UnsafeTargetRule_send_passer, _relay_passer, OnlyExportReachesL1_relay_passer |
| M4 exporter destination = source; M6 exporter sends t = 0; M15 exporter sends to sourceMessenger+1 | export_binding, exporter_anyCalldata_onlyExportPayload |
| M5 exporter reads expiredMessages instead of successfulMessages | export_binding |
| M33 L1 skips the INTEROP gate; M34 L1 trusts 0x..23 instead of the exporter | relayUndelivered_iff_and_deposit (and rejectsL2ToL2AsSender for M34) |
| M7 expire compares otherMessenger() with 0 instead of xDomainMessageSender(), so it always reverts on a real chain; M7b (campaign K10) drops the check, both operands set to 0 | expire_iff, expire_iff_unbounded |
| K41 check (a) reads the caller's OWN systemConfig(); K42 check (b) reversed (the caller portal's lockbox must authorize A's portal) | relayUndelivered_iff_and_deposit. The kill comes from the gate property against the attacker's forged answer, not from a missing getter. |
| M13 send records timestamp 1; M16 send skips sentMessages | send_effects_and_frame |
| M14 relay also sets expiredMessages | relay_effects_and_frame |
| M17 expire also sets successfulMessages | expire_iff |
| M24 relay does not mark the hash before the call | relay_effects_and_frame, OnlyExportReachesL1_relay_reentrant |
| M8 drop the lockbox check; M9 gas 100001 | relayUndelivered_iff_and_deposit |
| M20 L1 relays to itself | L1_relayMessage_rejectsSelfAndPortalTargets |
| M21 xDomainMsgSender not reset after the call | L1_relayGate_and_delivery |
| M23 sender field = tx.origin | L1_sendMessage_senderFieldIsCaller, relayUndelivered_iff_and_deposit |
| M10 refund skips the refunded check; M12 refund does not mark | refund_iff_effects_singleUse |
| M11 refund pays `to`; M18 hash nonce+1; M19 hash sender = from | refund_iff_effects_singleUse, sendETH_then_refund |
| M25 relay swallows a target revert; M26 relayMessage not `nonReentrant`; M27 entered flag never cleared; M28 context sender = target; M29 relay drops msg.value | relay_delivery_value_context_failure |
| X2 L1 gate ignores l2Sender (R1's round-2 mutant); X3a replay gate deleted; M21; M30 CDM relay drops `_value` | L1_relayGate_and_delivery |
| X3b replay gate deleted; X2b L2 gate accepts any caller | L2_relayGate_and_delivery |
| M31 CDM sendMessage records `_value` 0 | L1_sendMessage_senderFieldIsCaller |
| M35 / M35b an unauthorized relay marks the message failed instead of reverting | L1_relayGate_and_delivery / L2_relayGate_and_delivery |
| M36 nonReentrant without its `if (_entered()) revert` | OnlyExportReachesL1_relay_reentrant |
| M37 `Encoding.encodeCrossDomainMessageV1` stops binding `_sender` | L1_failedEntryNotReplayableWithAlteredField |
| M32 L2CDM relays to itself | L2_relayMessage_rejectsSelfAndPasser |
| X1 event nonce+1; X1b event sender = tx.origin (forge) | `test/L2/L2ToL2CrossDomainMessenger.t.sol::testFuzz_sendMessage_succeeds` |

The reviewers' combined mutant (send writes timestamp 1 plus relay sets expiredMessages) is covered by M13 and M14
separately.

## Retarget to the exporter design

What changed from the suite for the earlier design (export inside the L2ToL2CrossDomainMessenger):
- `exportUndeliveredMessage` moved to `UndeliveredMessageExporter`. Its checks now live in `ExporterExpiryHalmos`,
  including the new "any calldata ⇒ only the export payload" check.
- relayUndeliveredMessage trusts the exporter, read from `Predeploys`, and is gated on INTEROP. The new checks are
  `rejectsL2ToL2AsSender` and `FALSE_interopGateRedundant`.
- The messenger's unsafe-target rule now includes 0x..16, so the former `_PENDING` checks are expected PASS.
- `MESSAGE_EXPIRY_WINDOW` became `EXPIRY_PERIOD` (8 days).
- SuperchainETHBridge, ETHLiquidity, CrossDomainMessenger, L2CrossDomainMessenger and TransientContext are unchanged.
- The same v3 suite also passed in full on the earlier design, before the retarget (round-2 log below).

## Review log

Round 1, 2026-10-07. Reviewers: R1 (fresh-context reviewer), R2 and R3 (independent model-based reviewers).

| # | Reviewer(s) | Finding | Disposition |
|---|---|---|---|
| 1 | R1, R2, R3 | sendMessage's timestamp write is unchecked, and relay and export have no full storage frame. A combined mutant survived. | **Fixed.** Added `check_send_effects_and_frame` and `check_relay_effects_and_frame`, and full frames in `check_export_binding` and `check_expire_iff`, all at symbolic keys. Added mutants M13–M17 and M24, all killed. |
| 2 | R1, R2, R3 | run.sh passes when checks never run, and mutants.sh does not enforce kills. | **Fixed.** Added the `expected.tsv` inventory; missing, extra or empty results fail, and so does a halmos exit code other than 0 or 1. Both cases were reproduced and are now rejected. mutants.sh now enforces kills per mutant. |
| 3 | R2, R3 | The SELFDESTRUCT patch keeps the balance when the beneficiary is the account itself, which could give an unsound refund PASS. | **Fixed.** EIP-6780 burn for the self-beneficiary; constructor-only support; `from` assumed outside the fresh range (stated). Without that assumption halmos finds the burn counterexample. |
| 4 | R2, R3 | Refund environment: pre-funded SafeSend, bridge balance, insufficient liquidity, MAX_ETH. | **Fixed.** Symbolic p1, p2 and bridge balance in the effect statement. Insufficient liquidity is now in the iff; my earlier pruning rationale was wrong for CREATE. MAX_ETH raised to 2^200, with an explicit 2^198 bound. |
| 5 | R1, R2, R3 | Relay callbacks: targets were codeless only. | **Fixed.** Added `ReentrantTarget` and `check_OnlyExportReachesL1_relay_reentrant`, with OnlyExportReachesL1 restated per call. Remaining limits (one re-entry, export or send) are stated above. |
| 6 | R1, R3 | The L1 group covers one topology. | **Partly fixed.** Added the claim-A's-portal, self-call, relay-to-self/portal and reverting-getter variants. The unexplored topologies are listed above. |
| 7 | R1 | Names read as end-to-end claims; L1 sender exclusivity is assumed. | **Fixed.** Added the (7) checks on the real L1CDM. What is delegated is listed above. The group names describe contract-level statements, defined in this file. |
| 8 | R1, R2, R3 | Low: no composed send→refund check; expected-FAIL attribution; `hashBindsFromAsSender` fails trivially; recorder misses call-then-revert; byte lengths. | **Fixed or noted.** Added `check_sendETH_then_refund`. One own assert per expected-FAIL check. Replaced `hashBindsFromAsSender`: no FALSE check against an arbitrary map can tell implementations apart, so binding rests on the PASS iff plus M18 and M19. The recorder limit is noted. Default lengths are now 0,1,32,33,100,132,260. |
| 9 | integrator | Write a README. | This file. |
| 10 | integrator | Retarget to the exporter design. | Not landed yet; see "Pending design changes". |

Round 2, 2026-10-07. Reviewers: R1 (fresh-context reviewer), R2 and R3 (independent model-based reviewers). Verdict: the round-1 fixes are real, and
nothing is critical.

| # | Reviewer(s) | Finding | Disposition |
|---|---|---|---|
| 1 | R1 (HIGH) | The CrossDomainMessenger relay gates behind check (c) and behind expireMessage's authorization were neither checked nor listed as delegated. X2 and X3 survived. | **Fixed.** Added `check_L1_relayGate_and_delivery` and the `L2CDMGateHalmos` checks on the real L2CrossDomainMessenger, with symbolic caller, l2Sender, failed and successful flags, value and paused. X2, X3a, X3b and X2b are now killed. |
| 2 | R1, R2, R3 | The send→relay binding through the emitted SentMessage event is unchecked; X1 survived. | **Delegated, and enforced.** Halmos 0.3.3 has no recordLogs or log cheatcode: I checked its cheatcode table. `mutants.sh` runs X1 and X1b against the forge test `testFuzz_sendMessage_succeeds`, which kills both. |
| 3 | R2, R3 | The L1 sender probe could pass vacuously if the getter reverts or delivery is skipped for 0x..23. | **Fixed.** Getter success is recorded through a low-level call and asserted. There is a no-skipped-delivery assertion (message becomes successful ⇒ target ran), the sender is symbolic including 0x..23, and the probe has only a fallback. |
| 4 | R2, R3 | ETH-bearing messages were not covered. | **Fixed.** Symbolic `_value`, msg.value and messenger balance in the L1 and L2 gate and self-target checks. sendMessage uses a symbolic msg.value. L2ToL2 relay forwards a symbolic msg.value. Mutants M29, M30 and M31 are killed. |
| 5 | R2 | Transient state was not checked. | **Fixed where observable.** During the call the context getter returns (sender, source). Afterwards it reverts (entered flag cleared), and a second relay in the same transaction succeeds. Mutants M26, M27 and M28 are killed. Resetting the sender and source slots cannot be observed through the interface (`onlyEntered`); this is noted. |
| 6 | R3 | Target failure propagation was not checked. | **Fixed.** A reverting RelayProbe makes the relay revert with nothing consumed, and there is a liveness assertion for a non-reverting target. M25 (swallowed revert) is killed. |
| 7 | R1, R2, R3 | run.sh accepted an expected FAIL that also had stuck paths; mutants.sh ignored the exit status; test contracts could be missing from the inventory. | **Fixed.** run.sh now requires zero stuck paths in every check, rejects any non-benign WARNING, ERROR or TIMEOUT log line, and cross-checks the source's `contract … is Test` against `expected.tsv`. mutants.sh requires exit 1 and a valid counterexample. Stuck paths are tolerated for mutants only; see "Mutation check". (Round 3 corrected this entry, which first said "zero stuck paths".) |
| 8 | R2, R3 | `check_FALSE_refund_paysTo` could fail for an unrelated reason. | **Fixed.** `to` is realistic (outside the helper ranges, not the bridge or liquidity), `to` ≠ `from`, and amount > 0. |
| 9 | R1 | Notes: export ⇐ depends on the Recorder; baseGas was taken from the contract; the composed check binds at nonce 0 only; relay value paths. | **Fixed or noted.** The export ⇐ dependence is noted. baseGas is now an independent formula in both deposit checks. The composed check uses a symbolic prior nonce. Value paths are covered (item 4). |
| 10 | R1 | "No relay after export" was not listed as delegated. | **Added** to the delegated list: Lean, Quint and the window differential. |
| 11 | integrator | Retarget to the exporter design. | **Done**; see "Retarget" above. The exporter address comes from `Predeploys`. The governance assumption is added. |

Round 3. Reviewers: R1 (fresh-context reviewer, who also ran the suite plus 27 extra mutants and vacuity probes), R2
and R3. Verdict: high confidence in the local statements, with no critical findings.

| # | Reviewer(s) | Finding | Disposition |
|---|---|---|---|
| 1 | R1 | mutants.sh trusted forge's incremental build: test contracts that embed creation code were not rebuilt, so mutants survived falsely. | **Fixed.** `forge build --force` before every halmos mutant, and a mutant that does not compile is reported. |
| 2 | R2 (HIGH) | Replay eligibility: the gate checks never asserted the post-call `failedMessages`, so an unauthorized attempt that marked a message failed would pass. | **Fixed.** Both gate checks assert that an unauthorized attempt changes neither flag; this includes the portal with msg.value ≠ `_value`. Mutants M35 and M35b are killed. |
| 3 | R3 | Storage keys came from the production `Hashing`; replay with altered fields was untested. | **Fixed.** Added an independent `specVersionedHash`, `check_{L1,L2}_replayNeedsExactFailedEntry`, and the two-step `check_L1_failedEntryNotReplayableWithAlteredField` with its FALSE witness. M37 (hash stops binding `_sender`) is killed. |
| 4 | R2, R3 | Nested relay rejection was untested. | **Fixed.** ReentrantTarget mode 3 attempts a nested relayMessage, and the check asserts it never succeeds. M36 (only `if (_entered()) revert` removed) is killed. |
| 5 | R2, R3 | Forge mutants could be "killed" by a setUp failure. | **Fixed.** The designated test must pass on the unmutated code, and under the mutant that test itself must fail with no setUp or compile failure. |
| 6 | R1, R2, R3 | Expected-FAIL timeouts, the createCalldata wording, the exporter's observation scope, whole-contract frames, the portal value-mismatch replay path, the scope of the benign-warning whitelist, and `from == 0`. | **Fixed or documented.** The timeout limitation is stated exactly. The createCalldata wording is narrowed, and the exporter's observation scope is stated. Whole-contract frames are now the phase-2 reachability checks. The value mismatch is in the gate frame. The whitelist applies only to the refund groups. `from != 0` is assumed in the refund checks, since msg.sender is never 0 on chain. |
| 7 | R1 | Stale text (old NatSpec, orphan struct doc, a sentence about relay to 0x..16, commit references). | **Fixed.** Commit hashes are replaced by design descriptions. |
| 8 | integrator | Every headline PASS property needs an automated non-vacuity check. | **Done.** Each PASS check is paired with an expected-FAIL witness in `expected.tsv`, and seven witnesses were added. `run.sh` enforces the pairing and requires every witness to produce a validated counterexample. See "Non-vacuity". |
| 9 | non-vacuity rule | The new witness pairing caught a vacuous PASS: in `ReachL1CDMHalmos`, A's SystemConfig mock kept INTEROP and paused at their concrete `false`, because they are packed with the messenger in a slot written in setUp, so symbolic storage left them concrete. relayUndeliveredMessage therefore always reverted, and the witness `check_FALSE_reach_l1cdmNeverSelfSender` PASSED. | **Fixed.** Both flags are now set from fresh symbolic values, and the witness fails with a validated counterexample. This was a harness bug, not a contract bug. |
| 10 | (self) | `run.sh` passed `--default-bytes-lengths` on the command line, which silently overrode the reachability checks' smaller per-function sets and made them time out. | **Fixed.** Defaults moved to `halmos.toml`; `BYTES_LENGTHS` remains an explicit global override. |
| 11 | (self) | The L2ToL2 reach check's unknown-selector step sent two symbolic words. Through the proxy's `upgradeToAndCall(address,bytes)` selector, the second word became a symbolic ABI offset, and halmos got stuck on it. | **Fixed.** One word of arguments, as in the other reach contracts. A non-admin call to any proxy selector is still covered: it is forwarded and must revert. |

Round 4: cross-layer mutation campaign (`../mutation`).

| # | Source | Finding | Disposition |
|---|---|---|---|
| 1 | campaign K41, K42 (HIGH: both double spends) | Killed only "for the wrong reason". The stand-ins for the other chain's messenger and portal had no `systemConfig()` or `ethLockbox()`, so every forged-caller attempt reverted on a missing getter, honest attempts included, and the forged answer was never explored. | **Fixed.** The stand-ins now answer those getters with attacker-chosen, never-reverting symbolic values (`AttackerSystemConfig`, `AttackerLockbox`). K41 and K42 are added to `mutants.sh` and killed by `check_relayUndelivered_iff_and_deposit` through an acceptance the property forbids. |
| 2 | campaign | M7 was mislabeled: it compares otherMessenger() with 0, so expireMessage always reverts on a real chain; it does not drop the check. | **Relabeled**, and the real removal (campaign K10) is added as M7b. Both are killed. |
| 3 | campaign K04 | `ReachL2ToL2Halmos` lost the wrapped `sentAt + EXPIRY_PERIOD` case to a checked-arithmetic panic (0x11) in its own assertion (E), and halmos 0.3.3 does not count that panic as a failure. | **Fixed.** (E) now bounds `sentAt <= 2^256-1-EXPIRY_PERIOD` before adding, as `check_expire_iff_unbounded` does. |

