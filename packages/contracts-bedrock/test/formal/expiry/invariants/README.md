# Stateful invariant fuzzing: per-message interop expiry

Foundry invariant tests that drive the **real** contracts through sends, relays, exports, expiry
facts and refunds across several chains in one EVM, and check the expiry safety properties after
every call.

- **Target:** `karl/message-expiry-refunds` at `5992028e08`, the landed exporter design.
  - `UndeliveredMessageExporter` sits at `Predeploys.UNDELIVERED_MESSAGE_EXPORTER`. The harness only
    ever uses the constant.
  - `EXPIRY_PERIOD` is 8 days.
  - The messenger rejects both the L2CrossDomainMessenger and the L2ToL1MessagePasser as targets.
  - The contracts are unchanged from `dd0931a540`, where the same campaigns were also run.
- **Files:**
  - `ExpiryHandler.sol`: the handler, its ghost state, and the model and abstraction notes in its
    natspec.
  - `ExpiryInvariants.t.sol`: setup, configurations, invariants and deterministic witnesses.
  - `mutants/`: test-only modified copies of the messenger. These are **not** the real contracts.
    The real contracts are never modified.
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

Sharing one deployment between chains is sound here for two reasons:
- Each mapping is written by exactly one role. The source role writes `sentMessageTimestamps`,
  `expiredMessages` and `refunded`. The destination roles write `successfulMessages`.
- Every hash commits to both the destination and the source.

The `ETHLiquidity` pool is shared between the chains. Its balance therefore measures cluster-wide
conservation.

### Handler actions

The fuzz targets are the following actions. Indices pick from the ghost lists, and selector bytes
pick the variants.

- `sendETH`: A sends to B (3/4 of the time) or to C. The sender is one of 4 actors. The recipient is
  an actor or one of 16 pseudo-random EOAs. The amount is between 0 and 1000 ether.
  - The handler records the `SentMessage` payload (topics and data) that the real messenger emitted,
    and the send's timestamp.
- `warp`: advances the shared clock by up to 1 hour, up to 2 days, or up to 10 days. The boundary
  mode instead jumps to a boundary of a send:
  - `init + W - 1`, `init + W` or `init + W + 1`;
  - `init + P`, `init + P + 1`, `init + P + 2` or `init + P + 3`.
- `relayETH` / `attackerRelay`: relays a recorded message on its destination. **Protocol rule
  (enforced by the handler, not by the contracts):** the handler relays only payloads that the real
  messenger emitted on A, and only while `now - init <= W_protocol`.
  - `CrossL2Inbox.validateMessage` is mocked for exactly that `(Identifier, keccak256(payload))`
    pair.
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
- `relayForgedPayloadToL2CDM`: relays, on B, a payload that targets the L2CrossDomainMessenger and
  has **no** initiating event. This is a stronger adversary than the protocol allows. It checks the
  relay-side rule on its own.
- `exportMessage`: calls `exportUndeliveredMessage` on the real exporter for a send or attacker
  message, with these variants:
  - **Chain:** the message's destination (5/8 of the time), or B, C or A.
  - **L1 target:** A's L1CDM (7/8 of the time), a decoy, or a random address.
  - **Source chain:** 1/8 of the time, a wrong source chain.
  - **Bias:** a bias mode prefers a send that is past P, unrelayed and unexpired.
- `deliverFact`: delivers a captured fact to A through the **real**
  `L2CrossDomainMessenger.relayMessage`. The call is pranked as the aliased A_L1CDM, with sender =
  A_L1CDM, target 0x..23, `expireMessage(H, t)` and gas limit 100,000.
  - Two modes weaken an exported `t` to some `t' <= t`: within 2 days of `t`, or anywhere in
    `[0, t]`. This is sound: `successfulMessages` only grows, so the destination could also have
    exported at `t'`.
- `forgeExpiry`: tries every other way into `expireMessage`, with an arbitrary `undeliveredAt`:
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
| **ETH conservation** | The `ETHLiquidity` balance is at least its initial balance. It also equals `initial + sent - minted by relayETH - refunded` exactly, which means no unexplained flow. Per send, the sum of (relayed ? amount : 0) + (refunded ? amount : 0) is at most the total sent. |
| **RefundImpliesExpired** | Every successful `refundETH` saw `expiredMessages[h]` beforehand and refunded a hash that A really sent. Every refunded send is expired. |
| **ExpiredImpliesNeverRelayable** | For every expired message (ETH or attacker):<br>• `!successfulMessages[h]`;<br>• the expiry came from a delivered fact;<br>• the fact's `undeliveredAt`, A's time at expiry, and now are all `> init + W_protocol`;<br>• no relay that the protocol would accept was ever attempted on an expired message. |
| **AtMostOneRefund** | For every hash, `refundETH` succeeds at most once, and pays `from` exactly `amount`. |
| **OnlyExportReachesL1** | Every withdrawal whose L2CDM sender is the trusted sender came from an `exportUndeliveredMessage` call (a direct one, or one made by a relayed message whose target is the exporter). Each such withdrawal also says what an export on that chain at that time would say: `t == now`, H unrelayed there, and H not a known message to another chain. No export succeeds for a relayed hash. |
| **NoForgedFact / OnlyDestinationCanExport** | Every expiry came from an export fact made on the message's own destination. No path in `forgeExpiry` ever expires anything. |
| **UnsafeTargetRule** | No message targeting the L2CrossDomainMessenger is ever sent or relayed, and 0x..23 never makes a raw withdrawal (passer target rule). |
| **OnlyExportInitiatesWithdrawal** | The exporter never makes a raw withdrawal. |
| sentTimestamps | `sentMessageTimestamps[h]` equals the send's block timestamp, so it is never rewritten. |

## Configurations

| Contract | What it is | Expected |
|---|---|---|
| `ExpiryInvariants_Safety_Invariant` | Real contracts, with W_protocol taken from env `EXPIRY_INV_W_PROTOCOL` (default 7 days) and P_contract = `EXPIRY_PERIOD` (8 days). | pass |
| `ExpiryInvariants_TightWindow_Invariant` | W_protocol = P_contract: the boundary case, without the 1-day margin. | pass |
| `ExpiryInvariants_NoUnsafeTargetRule_Invariant` | Uses `mutants/L2ToL2CrossDomainMessengerNoUnsafeTargets.sol`, where `_isUnsafeTarget` always returns false, together with the real exporter. Shows that safety does not depend on the messenger's target rule. | pass |
| `ExpiryInvariants_NonVacuity_Invariant` | Asserts that "no relay", "no expiry", "no refund" and "no refund after a window-blocked relay" hold. | **fail** (`EXPIRY_INV_EXPECT_FAIL=true`) |
| `ExpiryInvariants_UnsafeWindow_Invariant` | W_protocol = P_contract + 1 day, which drops the assumption. | **fail**: NoDoubleSpend and ExpiredImpliesNeverRelayable |
| `ExpiryInvariants_LegacyNoTargetRule_Invariant` | **Legacy design**, as a test-only variant. It uses `mutants/L2ToL2CrossDomainMessengerLegacyNoTargetRule.sol` (the `37b44c48c7` messenger, which exports itself, without its target rule) and trusts 0x..23. Relays without an initiating event are excluded. | **fail**: OnlyExportReachesL1, NoForgedFact and NoDoubleSpend |
| `*_Witness_Test` (5 contracts, 10 tests) | Deterministic paths. See below. | pass (each asserts its path) |

Expected-to-fail contracts call `vm.skip(true)` unless `EXPIRY_INV_EXPECT_FAIL=true`, so the
default and CI runs skip them.

The witnesses are:
- refund after expiry, with a repeat refund and a late relay that both fail;
- an early fact (at `init + P`, or weakened) that is rejected;
- a relay that blocks export, with every `forgeExpiry` path failing;
- wrong-chain, wrong-source and wrong-L1-target exports that are rejected;
- target rules that reject sends to the L2CrossDomainMessenger and the passer;
- a relayed call to the exporter that counts as an export and leads to a refund;
- the unsafe-window double spend;
- without the target rule, the legacy attack yields only an untrusted 0x..23 withdrawal and no fact;
- the legacy design without its target rule double-spends;
- **governance:** an upgraded exporter forges a fact and causes a double spend.

## Assumptions, abstractions and mocks

- **Protocol relay rule (P1):** a relay is valid iff its payload was emitted by A's messenger and
  `exec - init <= W_protocol`. The handler enforces this in place of op-supernode and kona.
  `CrossL2Inbox.validateMessage` is mocked per relay.
- **Window cap:** P_contract >= W_protocol is checked in `setUp` (Go and kona cap W at 7 days). The
  `UnsafeWindow` configuration shows that this assumption is needed.
- **Clock:** one global, monotone clock (`block.timestamp`) is shared by all chains, so their
  timestamps are comparable.
- **L1 hop:** abstracted as described under "Facts and the L1 hop". The following are not executed:
  - the L1CrossDomainMessenger's three checks, its INTEROP gate and its lockbox membership checks;
  - withdrawal proving and finality;
  - deposit gas.

  The model assumes A has the INTEROP feature on, and that every chain in the model is in A's
  lockbox.
- **Gas:** the expiry deposit runs with ample gas. Running out of gas in the L1 or L2 relay, and the
  replays that follow, are not modeled.
- **Governance (named assumption):** every cluster chain runs the real `UndeliveredMessageExporter`.
  Each cluster chain's L2 governance (its L2 ProxyAdmin owner) can upgrade its own exporter, and an
  upgraded exporter could forge facts for any destination.
  `ExpiryInvariants_GovernanceAssumptionWitness_Test` demonstrates this: a forged fact for a relayed
  send leads to an expiry and a refund, which is a double spend. This is the same trust as the
  shared ETHLockbox, whose portals must share the proxy admin owner. The harness does not model a
  malicious cluster chain.
- **Activation:** the exporter is fresh in genesis, so no withdrawal from it can predate it. The
  harness starts from the post-upgrade genesis and does not model pre-staged legacy withdrawals.
- **Hashing:** keccak is treated as collision resistant. This is implicit: ghost state is keyed by
  hash.
- **Bounds:**
  - 3 chains, 4 sender actors, 16 extra recipients and 1 attacker;
  - amounts of at most 1000 ether, and junk calldata of at most 256 bytes;
  - the call sequence length is the invariant depth.

## Results

All runs were on hel1. For each configuration, forge runs one campaign that checks all of that
contract's invariants after every call. Runs that pass use `fail_on_revert = true`, and every one
of them had 0 handler reverts.

**At `5992028e08` (exporter design):** see "Run log" below for each run's wall-clock time.

| Configuration | runs × depth | calls | result |
|---|---|---|---|
| Safety (10 invariants) | 512 × 256 | 131,072 | pass |
| TightWindow | 512 × 256 | 131,072 | pass |
| NoUnsafeTargetRule (mutant) | 512 × 256 | 131,072 | pass |
| Safety, NoUnsafeTargetRule | 32 × 1024 | 32,768 each | pass |
| NonVacuity | 256 × 256 | stops at the first failure | all 4 fail (as expected) |
| UnsafeWindow | 256 × 256 | stops at the first failure | NoDoubleSpend and ExpiredImpliesNeverRelayable fail (as expected) |
| LegacyNoTargetRule | 256 × 256 | stops at the first failure | NoDoubleSpend, NoForgedFact and OnlyExportReachesL1 fail (as expected) |
| Witness tests | — | — | 10/10 pass |
| `FOUNDRY_PROFILE=ci` (64 × 32) | — | — | 13 pass, 3 expected-to-fail skipped, about 15 s |

Coverage at 512 × 256, per run, from `afterInvariant` stats (`EXPIRY_INV_STATS`). Each cell gives
the fraction of runs that reached the event at least once:

| | Safety | TightWindow | NoUnsafeTargetRule |
|---|---|---|---|
| relays | 513/513 | 513/513 | 513/513 |
| relays blocked by window | 513/513 | 513/513 | 513/513 |
| exports / reverted exports | 513 / 461 | 513 / 472 | 513 / 480 |
| expiries | 478/513 | 485/513 | 476/513 |
| refunds | 315/513 (451 total) | 334/513 (480 total) | 327/513 (440 total) |
| attacker relays | 471/513 | 473/513 | 493/513 |
| untrusted 0x..23 withdrawals (attacks that are facts in the legacy design) | 0 | 0 | 510/513 (2,827 total) |
| raw withdrawals from 0x..23 / from the exporter | 0 / 0 | 0 / 0 | 267 / 0 |

The same suite at `dd0931a540` (identical contracts) gave the same results and similar coverage.

**At `37b44c48c7` (earlier design, for reference):** these runs used the real contracts. The
exporter then was the messenger and the trusted sender 0x..23.
- Safety (9 invariants) and TightWindow each passed at 512 × 256 (131,072 calls).
- Both also passed at 32 × 1024.
- The no-target-rule mutant broke OnlyExportReachesL1, NoForgedFact and NoDoubleSpend.
- UnsafeWindow broke NoDoubleSpend.
- `OnlyExportInitiatesWithdrawal` for 0x..23 failed as documented. At that commit, a relayed message
  to the L2ToL1MessagePasser made 0x..23 a raw-withdrawal sender. That was fixed in `3b8d14c4ef`.

No new contract bug was found.

## Run

From `packages/contracts-bedrock`:

```sh
# Safety campaigns (deep):
FOUNDRY_INVARIANT_FAIL_ON_REVERT=true FOUNDRY_INVARIANT_RUNS=512 FOUNDRY_INVARIANT_DEPTH=256 \
  EXPIRY_INV_STATS=.testdata/expiry-stats.txt \
  forge test --match-path 'test/formal/expiry/invariants/*' \
  --match-contract 'Safety_|TightWindow|NoUnsafeTargetRule_'

# Expected-to-fail configurations (non-vacuity, unsafe window, legacy design):
EXPIRY_INV_EXPECT_FAIL=true FOUNDRY_INVARIANT_RUNS=256 FOUNDRY_INVARIANT_DEPTH=256 \
  forge test --match-path 'test/formal/expiry/invariants/*' \
  --match-contract 'NonVacuity|UnsafeWindow_|LegacyNoTargetRule_'

# Deterministic witnesses:
forge test --match-path 'test/formal/expiry/invariants/*' --match-contract Witness
```

Notes on running:
- Without overrides, the default profile uses Foundry's invariant defaults (256 × 500), which take
  several minutes per contract. The `ci` profile (64 × 32) takes about 10 s.
- `EXPIRY_INV_STATS` must point under `.testdata/` (fs permissions). Each run then appends one line
  of counters.
- Forge persists failing sequences in `cache/invariant/` and replays them first. On a re-run, an
  expected-to-fail suite then reports only the replayed invariant. Delete `cache/invariant` first to
  see every expected failure.
- Medusa was not run: the harness depends on `CommonTest` (ffi, the Deploy script,
  `vm.getDeployedCode`, `vm.recordLogs`, `vm.mockCall`).

## Run log

- `37b44c48c7`:
  - the 4-contract safety suite at 512 × 256 took 888 s;
  - the 32 × 1024 run took 464 s;
  - the expected-fail suite took 483 s.
- `dd0931a540`:
  - safety took 572 s;
  - the 32 × 1024 run took 436 s;
  - the expected-fail suite took 223 s;
  - the `ci` profile took 9 s.
- `5992028e08`:
  - safety took 660 s;
  - the 32 × 1024 run took 656 s;
  - the expected-fail suite took 264 s (with a clean `cache/invariant`);
  - the witnesses took 12 s;
  - the `ci` profile took 20 s.
  hel1 was shared with other jobs, so these times are loaded wall-clock times.
