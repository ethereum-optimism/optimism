# Stateful invariant fuzzing: per-message interop expiry

Foundry invariant tests that drive the **real** contracts through sends, relays, exports, expiry
facts and refunds across several chains in one EVM, and check the expiry safety properties after
every call.

- **Target:** the PR #23259 branch at `c7c51d79e2`, the landed exporter design after the
  style-guide pass (`ContractName_`-prefixed errors and the exporter's `UndeliveredMessageExported`
  event).
  - `UndeliveredMessageExporter` sits at `Predeploys.UNDELIVERED_MESSAGE_EXPORTER`, which is
    0x..0030 (it was 0x..2E before). The harness only ever uses the constant.
  - The harness matches no error selectors. It matches only the `SentMessage`, `MessageExpired` and
    `MessagePassed` events, and a witness asserts that each one is observed on the real contracts.
  - `EXPIRY_PERIOD` is 8 days.
  - The messenger rejects both the L2CrossDomainMessenger and the L2ToL1MessagePasser as targets.
- **Files:**
  - `ExpiryHandler.sol`: the handler, its ghost state, and the model and abstraction notes in its
    natspec.
  - `ExpiryInvariants.t.sol`: setup, configurations, invariants and deterministic witnesses.
  - `mutants/`: test-only modified copies, used only by the mutant configurations and the failing
    witnesses. These are **not** the real contracts, which are never modified. The copies are:
    - `L2ToL2CrossDomainMessengerNoUnsafeTargets`: no unsafe-target rule;
    - `L2ToL2CrossDomainMessengerFaulty`: no relay destination check and no replay check;
    - `SuperchainETHBridgeFaulty`: refunds skip the expiry and already-refunded checks, and relays
      pay the sender;
    - `L2ToL2CrossDomainMessengerLegacyNoTargetRule`: the `37b44c48c7` design without its target
      rule.

    All of them except the legacy copy are copies of the `c7c51d79e2` sources.
- **Kind of evidence:** randomized stateful testing. It is not a proof: it explores bounded random
  call sequences. The deterministic witnesses and the expected-to-fail configurations show that the
  campaigns reach the relevant states, and that the checks can fail.

## The model

The model uses one EVM with the repo's real L2 genesis: `CommonTest` with `enableInterop()`. Real
proxies and implementations of these contracts are used:
- `L2ToL2CrossDomainMessenger`;
- `UndeliveredMessageExporter`;
- `SuperchainETHBridge`;
- `ETHLiquidity`;
- `L2CrossDomainMessenger`;
- `L2ToL1MessagePasser`.

The EVM plays three chains. Before each step, `vm.chainId` is set to the chain that acts:

| Chain | Role |
|---|---|
| A (901) | The **only source**. Calls `sendETH`/`sendMessage`, receives expiry facts (`expireMessage`), and runs `refundETH`. |
| B (902), C (903) | Destinations. They relay (writing `successfulMessages[H]`) and export. Any chain, A included, may call the exporter. |

Sharing one deployment between chains is sound here for three reasons:
- Each mapping is written by exactly one role. The source role writes `sentMessageTimestamps`,
  `expiredMessages` and `refunded`. The destination roles write `successfulMessages`.
- Every hash commits to both the destination and the source.
- No destination ever sends. The only way a relayed message could make a destination send (a second
  source) is a relayed `SuperchainETHBridge.sendETH`. The handler filters that call, and an
  invariant (`nestedSends == 0`) checks that the filter is complete. Such a child message would
  commit to its own source chain, so it could never touch A's hashes anyway.

Two caveats follow from the shared storage:
- `successfulMessages` is shared by B and C. A relay that wrongly succeeded on C would also look
  delivered to B. The wrong-chain relay is therefore checked directly, by the `relayOnWrongChain`
  action and the DestinationBinding flag, not through B's state.
- The `ETHLiquidity` pool is shared between the chains. Its balance therefore measures cluster-wide
  conservation. Actual delivery is checked separately: each relay must pay its recipient exactly the
  amount, and the bridge's balance must never change.

### Handler actions

The fuzz targets are the following actions. Indices pick from the ghost lists, and selector bytes
pick the variants.

- `sendETH`: A sends to B (3/4 of the time) or to C. The sender is one of 4 actors. The recipient is
  an actor (even seeds) or one of 16 pseudo-random EOAs (odd seeds). The amount is between 0 and 1000 ether.
  - The handler records the `SentMessage` payload (topics and data) that the real messenger emitted,
    and the send's timestamp.
- `warp`: advances the shared clock by up to 1 hour, up to 2 days, or up to 10 days. The boundary
  mode instead jumps to a boundary of a send:
  - `init + W - 1`, `init + W` or `init + W + 1`;
  - `init + P`, `init + P + 1`, `init + P + 2` or `init + P + 3`.
- `relayETH` / `attackerRelay`: relays a recorded message on its destination. For ETH sends, the
  handler records the recipient's and the bridge's balances around the relay. **Protocol rule
  (enforced by the handler, not by the contracts):** the handler relays only payloads that the real
  messenger emitted on A, and only while `now - init <= W_protocol`.
  - `CrossL2Inbox.validateMessage` is mocked for exactly that `(Identifier, keccak256(payload))`
    pair.
- `relayOnWrongChain`: relays an authentic, within-window ETH send payload on the other destination
  (B and C swapped). The inbox mock accepts it, so only the messenger's destination check can refuse
  it.
- `attackerSend`: an attacker on A sends a message to B. With `relayNow`, the handler relays it in
  the same step.
  - **Targets:**
    - L2CrossDomainMessenger;
    - L2ToL1MessagePasser;
    - SuperchainETHBridge;
    - ETHLiquidity;
    - the messenger itself;
    - an L1 address;
    - the exporter;
    - a pseudo-random address.
  - **Calldata:**
    - `L2CrossDomainMessenger.sendMessage(A_L1CDM, relayUndeliveredMessage(h, t))`;
    - `initiateWithdrawal(A_L1CDM, ...)` with the same word;
    - a raw `relayMessage` withdrawal claiming sender 0x..23;
    - `relayETH`;
    - `expireMessage`;
    - `exportUndeliveredMessage`;
    - junk of up to 256 bytes.
  - Here `h` is a real send's hash, and `t` is past its expiry period or arbitrary.
  - A bridge target with `sendETH` calldata is filtered out (see the model).
- `relayForgedPayloadToUnsafeTarget`: relays, on B, a payload that has **no** initiating event and
  targets the L2CrossDomainMessenger or the L2ToL1MessagePasser. This is a stronger adversary than
  the protocol allows. It checks each relay-side target rule on its own, independent of the
  send-side rule.
- `exportMessage`: calls `exportUndeliveredMessage` on the real exporter for a send or attacker
  message, with these variants:
  - **Chain:** the message's destination (5/8 of the time), or B, C or A.
  - **L1 target:** A's L1CDM (7/8 of the time), a decoy, or a random address.
  - **Source chain:** 1/8 of the time, a wrong source chain.
  - **Bias:** a bias mode prefers a send that is at or past `init + P`, unrelayed and unexpired. The
    boundary itself is included so that boundary mutants are reachable.
- `deliverFact`: delivers a captured fact to A through the **real**
  `L2CrossDomainMessenger.relayMessage`. The call is pranked as the aliased A_L1CDM, with sender =
  A_L1CDM, target 0x..23, `expireMessage(H, t)` and gas limit 100,000.
  - Three modes weaken an exported `t` to some `t' <= t`: within 2 days of `t`, anywhere in
    `[0, t]`, or exactly `init + P_contract` (the expiry boundary, when that is `<= t`). This is
    sound: `successfulMessages` only grows, so the destination could also have exported at `t'`.
    The boundary mode is what lets the CI campaign kill an off-by-one in `expireMessage`.
- `forgeExpiry`: tries every other way into `expireMessage`, with an arbitrary `undeliveredAt`. Any
  `MessageExpired` emission on these paths counts as a violation, including a repeat on a hash that
  has already expired. The paths are:
  - an L1CDM relay with an L1 sender other than A_L1CDM;
  - a direct deposit from another L1 contract;
  - any L2 caller, including the L2CrossDomainMessenger outside a relay;
  - another L1 contract's deposit into the L2CrossDomainMessenger that claims A_L1CDM as its
    sender.
- `refund`: calls `refundETH` with the right preimage (6/10 of the time) or a wrong one: amount + 1,
  another `from`, another destination, or another nonce. Calls repeat. A bias mode picks expired
  sends.

### Facts and the L1 hop (abstraction)

The L1 is not executed. Instead, the handler captures every `MessagePassed` log of the real
`L2ToL1MessagePasser`. A withdrawal becomes a **fact** only when all of these hold:
- it is sent by the L2CrossDomainMessenger;
- its inner `relayMessage` sender is `TRUSTED_SENDER`, which is `Predeploys.UNDELIVERED_MESSAGE_EXPORTER`;
- its inner target is A's L1CrossDomainMessenger;
- its calldata is `relayUndeliveredMessage(H, t)`.

This models the following behaviour of `L1CrossDomainMessenger.relayUndeliveredMessage`. With the
INTEROP feature on for A, it accepts any messenger-sent withdrawal from the exporter of any chain
whose portal is in A's lockbox. All of B, C and A are in that lockbox. It accepts nothing else.

Two other kinds of withdrawal are counted but never delivered:
- **Raw withdrawals.** The sender is 0x..23 or the exporter calling the passer directly. On L1 the
  portal calls the target itself, so the caller is a portal, not an L1CrossDomainMessenger, and an
  L1CrossDomainMessenger rejects `l2Sender != L2CrossDomainMessenger`.
- **0x..23 withdrawals through the L2CrossDomainMessenger.** These have an untrusted sender.

## Properties

The handler records violation flags for each property, and `invariant_*` functions read them. Every
property is checked after every call.

| Property | Exact check |
|---|---|
| **NoDoubleSpend** | No ETH send has both `successfulMessages[h]` (on its destination) and `refunded[h]` (on A). |
| **ETH conservation** | Pool:<br>• the `ETHLiquidity` balance is at least its initial balance;<br>• it equals `initial + sent - minted by relayETH - refunded` exactly, so there is no unexplained flow.<br>Per send:<br>• (successful relays + successful refunds) of its hash is at most 1, so minted + refunded is at most its amount;<br>• summed over sends, minted + refunded is at most the total sent.<br>Delivery:<br>• every successful ETH relay raised its recipient's balance by exactly the amount;<br>• the bridge's balance never changes across a relay or a refund. |
| **AtMostOneRelay** | Every message (ETH send or attacker message) is relayed successfully at most once (counted by the handler). |
| **DestinationBinding** | An authentic, within-window payload is never relayed on a chain other than its destination. |
| **RefundImpliesExpired** | Every successful `refundETH` saw `expiredMessages[h]` beforehand and refunded a hash that A really sent. Every refunded send is expired. |
| **ExpiredImpliesNeverRelayable** | For every expired message (ETH or attacker):<br>• `!successfulMessages[h]`;<br>• the expiry came from a delivered fact;<br>• the fact's `undeliveredAt`, A's time at expiry, and now are all `> init + W_protocol`;<br>• no relay that the protocol would accept was ever attempted on an expired message. |
| **AtMostOneRefund** | For every hash, `refundETH` succeeds at most once, and pays `from` exactly `amount`. |
| **OnlyExportReachesL1** | Every withdrawal whose L2CDM sender is the trusted sender came from an `exportUndeliveredMessage` call (a direct one, or one made by a relayed message whose target is the exporter). Each such withdrawal also says what an export on that chain at that time would say: `t == now`, H unrelayed there, and H not a known message to another chain. No export succeeds for a relayed hash. |
| **NoForgedFact / OnlyDestinationCanExport** | Every successful `expireMessage` execution (each `MessageExpired` emission, including repeats after a legitimate expiry) ran on an export fact made on the message's own destination. No `MessageExpired` is emitted outside fact delivery, and no path in `forgeExpiry` ever emits one. |
| **UnsafeTargetRule** | No message targeting the L2CrossDomainMessenger or the L2ToL1MessagePasser is ever sent (accepted by `sendMessage`) or relayed (including forged payloads with no initiating event), and 0x..23 never makes a raw withdrawal. |
| **OnlyExportInitiatesWithdrawal** | The exporter never makes a raw withdrawal. |
| sentTimestamps / model scope | `sentMessageTimestamps[h]` equals the initiating block timestamp, for ETH sends and attacker messages alike, so it is never rewritten. No destination ever emits a send (`nestedSends == 0`). |

## Configurations

| Contract | What it is | Expected |
|---|---|---|
| `ExpiryInvariants_Safety_Invariant` | Real contracts, with W_protocol = 7 days (the protocol's maximum) and P_contract = `EXPIRY_PERIOD` (8 days). | pass |
| `ExpiryInvariants_TightWindow_Invariant` | W_protocol = P_contract: the boundary case, without the 1-day margin. | pass |
| `ExpiryInvariants_NoUnsafeTargetRule_Invariant` | Uses `mutants/L2ToL2CrossDomainMessengerNoUnsafeTargets.sol`, where `_isUnsafeTarget` always returns false, together with the real exporter. Shows that safety does not depend on the messenger's target rule. | pass |
| `ExpiryInvariants_NonVacuity_Invariant` | Asserts that "no relay", "no expiry", "no refund" and "no hash refunded after a window-blocked relay attempt on that same hash" hold. | **fail** (`RUN_EXPECTED_FAIL = true`) |
| `ExpiryInvariants_UnsafeWindow_Invariant` | W_protocol = P_contract + 1 day, which drops the assumption. | **fail**: NoDoubleSpend and ExpiredImpliesNeverRelayable |
| `ExpiryInvariants_LegacyNoTargetRule_Invariant` | **Legacy design**, as a test-only variant. It uses `mutants/L2ToL2CrossDomainMessengerLegacyNoTargetRule.sol` (the `37b44c48c7` messenger, which exports itself, without its target rule) and trusts 0x..23. Relays without an initiating event are excluded. | **fail**: OnlyExportReachesL1, NoForgedFact and NoDoubleSpend |
| `*_Witness_Test` (7 contracts, 20 tests) | Deterministic paths. See below. | pass (each asserts its path) |

Expected-to-fail contracts call `vm.skip(true)` unless the constant `RUN_EXPECTED_FAIL` in
`ExpiryInvariants_TestInit` is flipped to `true` locally. It is `false` in the repo, so the default
and CI runs skip them. There are no environment knobs and no file writes in CI: `vm.env*` is
reserved for `Config.sol`.

### CI budgets

Each invariant contract pins its budget inline, because inline config overrides both the profile
and `FOUNDRY_INVARIANT_*` env vars:

| Profile | Where | runs × depth | fail-on-revert |
|---|---|---|---|
| `liteci` | all PR tests, 4 feature variants | 32 × 256 | true |
| `ciheavy` | new or changed tests on PRs, 4 feature variants | 32 × 256 | true |
| `ci` | develop | 32 × 256 | true |
| `default` | local runs | from the profile or env (64 × 500 if unset) | true |

`fail-on-revert` is pinned for every profile. Any handler revert therefore fails the campaign
instead of being silently discarded, for example a mismatched send hash, a missing `SentMessage`
log, or an `L2CrossDomainMessenger.relayMessage` delivery that reverts.

Deep runs use the `default` profile, where runs and depth are not pinned, so
`FOUNDRY_INVARIANT_RUNS` and `FOUNDRY_INVARIANT_DEPTH` still apply. See "Run".

### Non-vacuity, per invariant

Every headline invariant has two kinds of evidence, both automated and deterministic (CI runs them
on every PR, with no randomness):
- **Reach evidence:** a witness on the **real** contracts reaches the situation the invariant
  guards, and the check passes there.
- **Failing witness:** a witness puts the system into a violating state and asserts that the check
  **fails**. It does this through `runCheck(id)` in a `staticcall`, which must revert. The violating
  state comes from a test-only mutant, from a dropped assumption, or from injecting a fault into
  real state.

On top of that, the mutation runs on the real sources (below) show that the randomized `liteci`
campaign kills each real-source mutant.

| Invariant | Reach evidence (real contracts) | Failing witness |
|---|---|---|
| NoDoubleSpend | `refundAfterExpiry` (relay attempt, then refund), `relayBlocksExport` (relay, then the expiry paths) | `unsafeWindowDoubleSpend` (W > P); `legacyForgedFact` (legacy mutant); `upgradedExporterForgesFact` (governance) |
| ETH conservation | `relayAtWindowBoundary` (payouts checked), `refundAfterExpiry` | `wrongPayoutDetected` (faulty bridge pays the sender); `replayDetected` (faulty messenger) |
| AtMostOneRelay | `relayBlocksExport` (a second relay is refused) | `replayDetected` (faulty messenger, relayed twice) |
| DestinationBinding | `wrongChainRelayRejected` (an A→B payload on C is refused) | `wrongChainRelayDetected` (faulty messenger) |
| RefundImpliesExpired | `refundAfterExpiry` (refund only after expiry; wrong preimages revert) | `refundWithoutExpiryDetected` (faulty bridge) |
| ExpiredImpliesNeverRelayable | `refundAfterExpiry` (relay blocked, then expiry), `earlyFactRejected` | `unsafeWindowDoubleSpend` (fact time within W) |
| AtMostOneRefund | `refundAfterExpiry` (the repeat refund reverts) | `doubleRefundDetected` (faulty bridge) |
| OnlyExportReachesL1 | `refundAfterExpiry`, `relayedExportCall`, `exporterDesignBlocksForgery` | `legacyForgedFact`; `upgradedExporterForgesFact` |
| NoForgedFact / OnlyDestinationCanExport | `relayBlocksExport` (every `forgeExpiry` path), `wrongExportRejected`, `earlyFactRejected` | `legacyForgedFact`; `upgradedExporterForgesFact` |
| UnsafeTargetRule | `targetRuleRejects` (sends and forged payloads to both targets) | `exporterDesignBlocksForgery` (no-unsafe-targets mutant); `legacyForgedFact` |
| OnlyExportInitiatesWithdrawal | `refundAfterExpiry` (exports go through the L2CrossDomainMessenger only) | `upgradedExporterRawWithdrawal` (governance) |
| sentTimestamps / model scope | every witness that sends; attacker messages in `relayedExportCall` | `sentTimestampCorruptionDetected` (fault injection); `nestedSendDetected` (filter off) |

The non-vacuity reach checks for the randomized campaign itself (relay, expiry, refund, and refund
after a blocked relay of the same hash) are `ExpiryInvariants_NonVacuity_Invariant`, an
expected-to-fail suite run with `RUN_EXPECTED_FAIL`. The coverage table below gives per-run reach
counts at the CI budget.

### Witness list

The witnesses are:
- a blocked relay, then refund after expiry, of the same hash; a repeat refund fails, and a
  re-delivery of the honest fact stays authorized;
- ETH relays at `init + W - 1` and `init + W` succeed and pay the recipient; at `init + W + 1` the
  protocol refuses the relay;
- an authentic A→B payload relayed on C is refused, and then relays on B;
- an early fact (at `init + P`, or weakened) that is rejected;
- a relay that blocks export, with every `forgeExpiry` path failing;
- wrong-chain, wrong-source and wrong-L1-target exports that are rejected;
- target rules that reject sends to the L2CrossDomainMessenger and the passer, and forged payloads to
  each;
- a relayed call to the exporter that counts as an export and leads to a refund;
- the unsafe-window double spend;
- without the target rule, the legacy attack yields only an untrusted 0x..23 withdrawal and no fact;
- the legacy design without its target rule double-spends;
- **governance:** an upgraded exporter forges a fact and causes a double spend, and an upgraded
  exporter's raw withdrawal is detected;
- **faulty messenger:** a wrong-chain relay and a replay are detected;
- **faulty bridge:** a refund without expiry, a double refund and a wrong payout are detected;
- **fault injection:** a corrupted send timestamp and a nested send are detected.

## Assumptions, abstractions and mocks

- **Protocol relay rule (P1):** a relay is valid iff its payload was emitted by A's messenger and
  `exec - init <= W_protocol`. The handler enforces this in place of op-supernode and kona.
  `CrossL2Inbox.validateMessage` is mocked per relay.
- **Window cap:** P_contract >= W_protocol is checked in `setUp` (Go and kona cap W at 7 days). The
  `UnsafeWindow` configuration shows that this assumption is needed.
- **Clock:** the harness uses one global, monotone clock (`block.timestamp`) for simplicity. The
  argument needs less than that.
  - Both cross-chain comparisons have the same shape, a destination timestamp minus A's initiating
    timestamp: the relay rule (`exec - init <= W`) and `expireMessage` (`t > sentAt + P`). So no
    clock agreement is needed beyond what the protocol rule already uses.
  - Per-chain monotonicity of the destination's clock then gives "unrelayed at t and t > init + W
    implies never relayable".
- **L1 hop:** abstracted as described under "Facts and the L1 hop", and **not exercised** here. The
  following are not executed:
  - the L1CrossDomainMessenger's three checks, its INTEROP gate and its lockbox membership checks;
  - withdrawal proving and finality;
  - deposit gas.

  The model assumes A has the INTEROP feature on, and that every chain in the model is in A's
  lockbox. These checks are covered elsewhere:
  - `L1CrossDomainMessenger_RelayUndeliveredMessage_Test` in `test/L1/L1CrossDomainMessenger.t.sol`
    (about lines 1249–1410: success, borrowed portal, other cluster, wrong L2 sender, interop
    disabled, 0x..23 sender, not a messenger, no lockbox, own chain);
  - the Halmos layer (`../halmos/L1CDMExpiryHalmos.t.sol`, `ReachL1Halmos.t.sol`);
  - the Kontrol layer (`../kontrol/solc0815/L1CrossDomainMessengerExpiry.k.sol`).
- **Gas and replays:** the expiry deposit runs with ample gas. Gas exhaustion is not modeled.
  - **Replays in general:** a relay that runs out of gas lands in that messenger's
    `failedMessages`, and anyone can replay it later. A replay re-executes the same versioned
    message: the same sender, target, value, gas limit and calldata.
  - **Why replays are safe here:**
    - A replayed L2→L1 export withdrawal re-delivers the same honest fact, which is still true
      because `successfulMessages` only grows. A delayed or re-delivered fact is exactly what the
      `deliverFact` weakening modes and repeated deliveries cover.
    - A replayed L1→L2 `expireMessage` deposit re-runs the same check on the same `(H, t)`, which is
      what a repeated `deliverFact` does. A repeat after expiry is allowed and must stay authorized,
      which NoForgedFact checks per emission.
    - Replays cannot change the L2 sender, so they cannot turn a 0x..23 or raw withdrawal into a
      trusted-sender fact.
- **Governance (named assumption):** every cluster chain runs the real `UndeliveredMessageExporter`.
  Each cluster chain's L2 governance (its L2 ProxyAdmin owner) can upgrade its own exporter, and an
  upgraded exporter could forge facts for any destination.
  `ExpiryInvariants_GovernanceAssumptionWitness_Test` demonstrates this: a forged fact for a relayed
  send leads to an expiry and a refund, which is a double spend. This is comparable to the trust in
  the shared ETHLockbox: a member's L2 governance can already make arbitrary withdrawals from it by
  changing its own L2 state (the lockbox's own check compares the portals' L1 ProxyAdmin owners, a
  separate role). The harness does not model a malicious cluster chain.
- **Activation:** the exporter is fresh in genesis, so no withdrawal from it can predate it. The
  harness starts from the post-upgrade genesis and does not model pre-staged legacy withdrawals.
- **Hashing:** keccak is treated as collision resistant. This is implicit: ghost state is keyed by
  hash.
- **Bounds:**
  - 3 chains, 4 sender actors, 16 extra recipients (from odd seeds) and 1 attacker;
  - amounts of at most 1000 ether, and junk calldata of at most 256 bytes;
  - the call sequence length is the invariant depth.

## Results

All runs used a 32-core Linux host, under a 12–16 GB memory cap, with one forge job at a time. For
each configuration, forge runs one campaign that checks all of that contract's invariants after
every call. `fail-on-revert` is on in every run, and every passing run had 0 handler reverts.

**At `c7c51d79e2`:**

| Configuration | Profile, runs × depth | Calls | Result |
|---|---|---|---|
| whole directory: Safety (12 invariants), TightWindow, NoUnsafeTargetRule and all witnesses | `liteci`, pinned 32 × 256 | 8,192 per suite | pass in 84 s (suites run in parallel; 261 s CPU) |
| the same | `liteci` with `SYS_FEATURE__CUSTOM_GAS_TOKEN`, `DEV_FEATURE__OPTIMISM_PORTAL_INTEROP` or `DEV_FEATURE__ZK_DISPUTE_GAME` | 8,192 per suite | pass in every variant (77–78 s) |
| the same | `ci`, pinned 32 × 256 | 8,192 per suite | pass in 41 s |
| Safety, TightWindow, NoUnsafeTargetRule | `default` 512 × 256 | 131,072 each | pass, 914 s |
| Safety, NoUnsafeTargetRule | `default` 32 × 1024 | 32,768 each | pass, 769 s |
| NonVacuity | `default` 256 × 256, `RUN_EXPECTED_FAIL = true` | stops at the first failure | all 4 fail (as expected) |
| UnsafeWindow | the same | stops at the first failure | NoDoubleSpend and ExpiredImpliesNeverRelayable fail (as expected) |
| LegacyNoTargetRule | the same | stops at the first failure | NoDoubleSpend, NoForgedFact and OnlyExportReachesL1 fail (as expected) |
| Witness tests (reach and failing) | — | — | 20/20 pass |

With the repo-pinned **forge 1.8.3**, the files compile under the `cicoverage` profile (into a
separate output directory). Under `liteci` the whole directory passes in 80 s wall clock (253 s CPU):
Safety 76 s, TightWindow 76 s, NoUnsafeTargetRule 80 s, and all 20 witnesses pass. The other runs
above used forge 1.8.1, with the same sources and artifacts layout.

### Coverage

Coverage comes from `afterInvariant` stats, with `WRITE_STATS = true` set locally. Each cell gives
the fraction of runs that reached the event at least once, with the total count in parentheses:

| | Safety, liteci 32 × 256 | TightWindow, liteci | NoUnsafeTargetRule, liteci | Safety, 512 × 256 | NoUnsafeTargetRule, 512 × 256 |
|---|---|---|---|---|---|
| relays | 33/33 | 33/33 | 32/33 | 513/513 | 513/513 |
| relays blocked by window | 33/33 | 33/33 | 33/33 | 513/513 | 513/513 |
| expiries | 30/33 (49) | 28/33 (64) | 33/33 (70) | 462/513 (1,110) | 461/513 (1,171) |
| refunds | 18/33 (18) | 15/33 (20) | 17/33 (19) | 308/513 (393) | 300/513 (399) |
| attacker relays | 30/33 | 32/33 | 31/33 | 448/513 | 493/513 |
| untrusted 0x..23 withdrawals (facts in the legacy design) | 0 | 0 | 30/33 (82) | 0 | 483/513 (1,509) |
| raw withdrawals from 0x..23 / from the exporter | 0 / 0 | 0 / 0 | 31 / 0 | 0 / 0 | 505 / 0 |

Before the review fixes, the CI campaign (unpinned 64 × 32) reached an expiry in about 4 of 65 runs
and a refund in 1 of 65.

### Mutation testing on the real sources

Each mutant is a one-line edit to the real `src/` contract, applied temporarily in a scratch
worktree and never committed. Each one ran under the pinned `liteci` budget (Safety, TightWindow
and NoUnsafeTargetRule):

| Mutant (real source) | Killed under `liteci` 32 × 256 by | Trials killed |
|---|---|---|
| M1: `expireMessage` uses `<` instead of `<=` at `sentAt + P` | TightWindow (ExpiredImpliesNeverRelayable: fact time == init + W) | 6/6 with the boundary weakening mode (before that mode: 3/4 at 32 runs, 1/2 at 16 runs) |
| M2: no `SuperchainETHBridge_AlreadyRefunded` check | Safety (AtMostOneRefund, ETH conservation), TightWindow, NoUnsafeTargetRule | 4/4 |
| M3: no replay check (`successfulMessages`) | Safety (AtMostOneRelay, ETH conservation), TightWindow | 2/2 |
| M4: no relay destination check | Safety (DestinationBinding, ETH conservation), TightWindow | 2/2 |
| M5: no relay-side unsafe-target check | Safety (UnsafeTargetRule), TightWindow | 2/2 |
| M6: `relayETH` pays `_from` | Safety (ETH conservation), TightWindow, NoUnsafeTargetRule | 2/2 |
| M7: `refundETH` skips the expiry check | Safety (NoDoubleSpend, RefundImpliesExpired, ETH conservation), TightWindow, NoUnsafeTargetRule | 2/2 |
| M8: no send-side unsafe-target check | Safety (UnsafeTargetRule), TightWindow | 2/2 |

The trials ran at `52ff613e14` and `c7c51d79e2`, which have the same logic; only the error names
differ. Each trial took 30–370 s of wall clock. M1 can only be caught where W_protocol == P_contract
(TightWindow): with the 1-day margin, an off-by-one at `P` is still safe.

### Earlier commits (for reference)

**`5992028e08` and `dd0931a540`** (the same messenger and exporter bytecode, with the exporter at
0x..2E): the pre-review harness passed Safety, TightWindow and NoUnsafeTargetRule at 512 × 256 and
32 × 1024. The expected-to-fail suites failed as intended.

**`37b44c48c7`** (the earlier design, using the real contracts; the messenger exported and 0x..23
was trusted):
- Safety and TightWindow passed at 512 × 256 and at 32 × 1024.
- The no-target-rule mutant broke OnlyExportReachesL1, NoForgedFact and NoDoubleSpend.
- UnsafeWindow broke NoDoubleSpend.
- At that commit, a relayed message to the L2ToL1MessagePasser made 0x..23 a raw-withdrawal sender.
  That was fixed in `3b8d14c4ef`.

No new contract bug was found.

## Run

From `packages/contracts-bedrock`:

```sh
# What CI runs (pinned 32 x 256, fail-on-revert):
FOUNDRY_PROFILE=liteci forge test --match-path 'test/formal/expiry/invariants/*'

# Deep safety campaigns (default profile; runs and depth come from env):
FOUNDRY_INVARIANT_RUNS=512 FOUNDRY_INVARIANT_DEPTH=256 \
  forge test --match-path 'test/formal/expiry/invariants/*' \
  --match-contract 'Safety_|TightWindow|NoUnsafeTargetRule_'

# Expected-to-fail suites (non-vacuity, unsafe window, legacy design): first set
# RUN_EXPECTED_FAIL = true in ExpiryInvariants_TestInit, and delete cache/invariant.
FOUNDRY_INVARIANT_RUNS=256 FOUNDRY_INVARIANT_DEPTH=256 \
  forge test --match-path 'test/formal/expiry/invariants/*' \
  --match-contract 'NonVacuity|UnsafeWindow_|LegacyNoTargetRule_'

# Deterministic witnesses:
forge test --match-path 'test/formal/expiry/invariants/*' --match-contract Witness
```

Notes on running:
- **Coverage stats:** set `WRITE_STATS = true` locally. Each run then appends a line of counters to
  `.testdata/expiry-invariants-stats.txt`. The flag is `false` in the repo, so CI never writes files.
- **Artifacts:** `DeployUtils.getDeployedCode` (used by the L2 genesis and by the mutant etching)
  reads `forge-artifacts/` on disk, whatever `FOUNDRY_OUT` says. Run in place. With a separate
  `FOUNDRY_OUT`, genesis would load whatever was last compiled into `forge-artifacts/`.
- **Replayed failures:** forge persists failing sequences in `cache/invariant/` and replays them
  first. On a re-run, an expected-to-fail suite then reports only the replayed invariant.
- **Medusa:** not run. The harness depends on `CommonTest`, which uses ffi, the Deploy script and
  cheatcodes Medusa lacks.

## Review log

- **Round 1 (pre-merge):** reviewers R1 (fresh context), and R2 and R3 (independent model-based
  reviewers). Verdict: sound, with conservative abstractions; no critical or high findings. R1 ran
  10 mutants of the real sources. Most were killed at the then-default CI budget, but two survived
  it: `<=` → `<` at the P boundary, and removing `AlreadyRefunded`.

  Findings and how each was resolved in v2:
  1. **CI evidence was thin** (expiries and refunds rarely reached at 64 × 32). Fixed with inline
     budgets: `liteci`, `ciheavy` and `ci` at 32 × 256. The deep path is the `default` profile with
     env runs and depth. All 8 v2 mutants are killed under the `liteci` budget (table above).
  2. **`fail_on_revert` was off** in the default and CI profiles. It is now pinned inline for every
     profile.
  3. **Conservation precision:**
     - added AtMostOneRelay (per-hash relay count ≤ 1);
     - per send, relays + refunds ≤ 1 (the README had called the aggregate "per send");
     - recipient payout is checked exactly, and the bridge never keeps ETH.
  4. **Destination binding** was untested. Added `relayOnWrongChain` and the DestinationBinding
     check, and documented the shared `successfulMessages` caveat.
  5. **Passer target rule:** the forged-payload relay now covers the L2ToL1MessagePasser too.
     Acceptance of a passer target on send or relay sets `unsafeTargetAccepted`.
  6. **NoForgedFact** now checks every `MessageExpired` emission, including repeats after a
     legitimate expiry, and any emission outside fact delivery.
  7. **Shared-storage justification for nested traffic:** a relayed `bridge.sendETH` is filtered,
     and `nestedSends == 0` is asserted. This is justified in "The model".
  8. **Witnesses:** added deterministic ETH relays at `W - 1` and `W` (and a refusal at `W + 1`).
     The blocked-relay-then-refund non-vacuity check is now per hash and ordered.
  9. **Smaller fixes:**
     - `sentTimestamps` now covers attacker messages;
     - the recipient seeds really reach 16 addresses;
     - the default budget is stated as 64 × 500;
     - added the replay, clock and L1-hop pointers.
- **Round 2 (non-vacuity requirement):** every headline invariant now has deterministic reach
  evidence on the real contracts and a deterministic failing witness that asserts the check fails.
  Both run in CI; see the "Non-vacuity, per invariant" table. This added two test-only mutants
  (`L2ToL2CrossDomainMessengerFaulty`, `SuperchainETHBridgeFaulty`) and three fault injections
  (corrupted timestamp, nested send, the upgraded exporter's raw withdrawal).
- **CI kill rate for M1:** M1 (the off-by-one at `P`) survived 1 of 4 `liteci` trials. The handler
  gained an exact-boundary weakening mode in `deliverFact` (`t' = init + P`, a sound weakening), and
  M1 was then killed in 6 of 6 trials.
- **Retarget to `c7c51d79e2`:** the exporter moved to 0x..0030 and the style-guide pass renamed
  errors. The harness matches no error selectors, so the only changes were to regenerate the
  mutants from the new sources and re-run everything.
- **Repo compliance (v2):**
  - no `vm.env*` (the env knobs are replaced by constants, and expected-to-fail suites are gated
    by `RUN_EXPECTED_FAIL`);
  - no file writes in CI;
  - `abi.encodeCall` only;
  - named returns end in `_`;
  - custom errors instead of revert strings;
  - `DeployUtils.getDeployedCode` with bare names;
  - semgrep clean, `forge fmt` clean, and the test-name lint clean.

## Run log

- `37b44c48c7` (pre-review harness): the safety suite at 512 × 256 took 888 s, the 32 × 1024 run
  464 s, and the expected-fail suite 483 s.
- `dd0931a540` and `5992028e08` (pre-review harness): safety took 572–660 s, the 32 × 1024 run
  436–656 s, and the expected-fail suite 223–264 s.
- `52ff613e14` (v2 harness):
  - `liteci` took 96 s, and each feature variant 89–106 s;
  - safety took 680 s;
  - the 32 × 1024 run took 653 s;
  - the expected-fail suite took 298 s;
  - mutants M1–M8 were run at 16 and at 32 runs.
- `c7c51d79e2` (final harness, forge 1.8.1):
  - `ci` took 43 s;
  - `liteci` took 99 s, and each feature variant 79–81 s;
  - safety took 914 s, and the re-run with the boundary weakening mode 723 s;
  - the 32 × 1024 run took 769 s;
  - the expected-fail suite took 329 s, and 177 s on the re-run;
  - the mutants took 30–370 s each.
- `c7c51d79e2` (forge 1.8.3):
  - the `cicoverage` compile took 7 s;
  - `liteci` took 83 s;
  - the witnesses took 106 s, including compile;
  - six M1 trials took 30–333 s.

The host was shared with other jobs, so these are loaded wall-clock times.
