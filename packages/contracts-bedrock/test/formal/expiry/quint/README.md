# Quint model: interop message expiry

> **Scope versus the code.** The exporter design landed on the PR #23259 branch at
> `5992028e08`:
> - `UndeliveredMessageExporter` at `Predeploys.UNDELIVERED_MESSAGE_EXPORTER` (0x...0030 since
>   `52ff613e14`; 0x..2E before);
> - `relayUndeliveredMessage` trusts it and has the INTEROP gate;
> - P = 8 days;
> - the Go and kona caps reject W > 7 days.
>
> The `safe` instances describe that code. The earlier commit `37b44c48c7` implemented the earlier
> design:
> - the L1 messenger trusts 0x..23 (`L1CrossDomainMessenger.sol:109-110`);
> - exports come from the L2ToL2 messenger;
> - `MESSAGE_EXPIRY_WINDOW` is 7 days, and the protocol window has no cap.
>
> That is this model's `messengerTrustedPrestaged` configuration, which **double-spends** (the
> instance uses P = 8; the earlier code's P = W = 7 is not a separate instance). Before its upgrade, a chain's messenger has no
> `relayMessage` target check, so an attacker can relay a message to
> `L2CrossDomainMessenger.sendMessage(routeL1CDM, relayUndeliveredMessage(H_future, …))` and replay the
> resulting deposit later.
>

`expiry.qnt` models the cross-chain protocol at the level of "who can make which withdrawal or
deposit exist". It covers the design in which exports come from a predeploy whose proxy has no
implementation before the upgrade, so nothing can be sent from its address (the "exporter"), and the earlier design in which the
L2ToL2CrossDomainMessenger itself was trusted, kept as a counterexample.

It is checked two ways:
- **`quint verify`**: Apalache bounded model checking, every execution up to a step bound, with SMT.
  This is the authoritative check. The bound is `DEPTH` (default 15) for the counterexamples and
  witnesses, and `SAFE_DEPTH` (default 10; `RESEND_SAFE_DEPTH`, default 9, for
  `safeResendRestarts`) for `Safety` in the safe instances, the depths recorded under **Results**.
- **`quint run`**: random simulation, a sanity pass.

`./run.sh` asserts every expected outcome and exits nonzero on any surprise, including tool errors.

## What is modeled

**Chains.**
- A, B and C run the standard code; D runs arbitrary code.
- A and B start in the shared ETHLockbox (the cluster). Any chain can be authorized later (`join`),
  and the lockbox check is evaluated when the L1 relay happens.
- Every chain has its own clock, which only moves forward (`tick(x, t)` may jump).

**Messages.** There are three bridge sends:
- `m1` goes A→B;
- `m2` goes B→A, so a fact has to be routed to the right source chain's L1 messenger;
- `m3` goes A→C, where C is outside the lockbox at genesis and can join later.

**Chain IDs.** Hashes use `CHAIN_ID`. With `UNIQUE_IDS` off, C has B's chain ID.

**Units.** Time is in days, with `PROTOCOL_WINDOW` (W) = 7 and `CONTRACT_PERIOD` (P) = 8.
`MAX_TIME` = 20.

**L2→L1 messages.** Every withdrawal records the L2 address that called the L2CrossDomainMessenger:
`exporter`, `messenger` (0x..23) or `user`. The attacker's moves are listed by hand, one per target
contract and recorded sender. That nothing can forge the trusted sender is therefore built into this
action set, not derived. The Lean model derives it from a target-level relay over an arbitrary
`decode` of calldata. What this model checks independently is the timing argument and the hash and
route bindings:

| Action | Stands for |
|---|---|
| `tick(x, t)` | Chain `x`'s clock advances to `t`. |
| `upgrade(x)` | The network upgrade on a standard chain. The exporter's proxy gets its implementation; the messenger starts recording timestamps and, with `TARGET_RULE`, gets the target rule. Since genesis the exporter address holds a proxy that reverts without an implementation (`L2Genesis` etches a proxy at every predeploy slot), so nothing can be sent from it before the upgrade. |
| `join(x)` | `x`'s portal is authorized in the lockbox. With `JOIN_REQUIRES_STANDARD`, only standard chains can join (a governance assumption). |
| `send(m)` | `sendMessage` on the source. Once per message; the timestamp is recorded only after the source's upgrade. |
| `resend(m)` | `resendMessage`. Available on a source before its upgrade, since the old messenger still has it: it re-emits the event and records no timestamp. After the upgrade it exists only in the historical variants (`RESEND`). |
| `relay(m)` | `relayMessage` on the destination. Valid iff some initiating event `e` has `now[dest] <= e + W`; cross-safety and the fault proof reject the rest. |
| `exportUndelivered(x, m, route)` | The export function on upgraded standard chain `x`. It sends `relayUndeliveredMessage(H, now[x])` to `route`'s L1 messenger; the route is chosen by the caller. The hash uses `CHAIN_ID[x]` as the destination, and the export reverts if `x` relayed that hash. The recorded sender is the trusted sender of the design. |
| `relayToL2CrossDomainMessenger(x, …)` | The attacker relays a message on standard chain `x` whose target is the L2CrossDomainMessenger, with any calldata. The recorded sender is the messenger. Possible before `x`'s upgrade (switched off only by `PRE_UPGRADE_FORGERY`, to isolate one counterexample), or after it if `TARGET_RULE` is off. |
| `userWithdrawal(x, …)` | Anyone calls `L2CrossDomainMessenger.sendMessage` with any payload. The recorded sender is the caller. |
| `arbitraryCode(x, sender, …)` | A non-standard chain produces withdrawals with any recorded sender. Only while it is outside the lockbox, so a counterexample through it shows withdrawals made before a join. |
| `l1Relay(w)` | The withdrawal is proven and finalized, then relayed into `route`'s `relayUndeliveredMessage`, which deposits `expireMessage` to `route`. It needs: `from != route` (an L1 messenger never relays to itself); `from` in the lockbox **now** (`LOCKBOX_CHECK`); the recorded sender equal to the trusted sender (`SENDER_CHECK`). |
| `fakeCaller(route, …)` | Any L1 contract F calls `relayUndeliveredMessage` directly and answers every getter itself. If F names a real, authorized portal, check (a) fails, because that portal's SystemConfig names the real messenger. If F names a fake portal whose fake SystemConfig names F, check (a) passes, but check (b) fails because the fake portal isn't in the lockbox. Check (c) always passes. So F succeeds iff (a) or (b) is off. |
| `expire(dep)` | The deposit runs `expireMessage` on `dep.to`. Only the message's source has a timestamp for its hash, and the hash also binds the destination. Requires `sentAt != 0` and `at > sentAt + P` (`>=` with `EXPIRE_GE`). |
| `refund(m)` | `refundETH`: expired and not yet refunded. |

Withdrawals and deposits are never removed. A withdrawal can be finalized at any later time, and
failed L1 and L2 relays can be replayed, so keeping them can only add executions.

## Properties

`Safety` is the conjunction of these:
- `NoDoubleSpend`: no message is both relayed on its destination and refunded on its source.
- `RefundImpliesExpired`.
- `ExpiredImpliesNeverRelayable`: an expired message was never relayed, and none of its initiating
  events is within W of the destination's current clock. That clock only moves forward, and after
  the source's upgrade no resend can add an initiating event (in `safeResendRestarts`, the
  hypothetical resend refuses expired messages), so this is permanent.
- `NoForgedFact`: every fact that expired a message was exported by that message's destination. This
  is a provenance property, stronger than safety strictly needs.
- `AtMostOneRefund`.

Defense in depth, separate from `Safety`:
- `MessengerSilentAfterUpgrade`: once a chain is upgraded, its L2ToL2CrossDomainMessenger never
  initiates a withdrawal through the L2CrossDomainMessenger. This is the target rule's own property,
  and it is meaningful only in the exporter-trusted instances: its ghost variable tracks
  `relayToL2CrossDomainMessenger`, not exports, which in the messenger-trusted instances are
  recorded with the messenger as sender. It holds in `safe` directly from the `TARGET_RULE` guard,
  and is violated in `safeNoTargetRule`, where `Safety` still holds. So safety does not depend on
  the rule. The L2ToL1MessagePasser path (raw withdrawals) is outside this model;
  Halmos and Kontrol check it on bytecode.

Non-vacuity witnesses, each of which must be violated in `safe`:
- `NoRefundEver`: a refund happens.
- `NoEdgeRelay`: a relay happens exactly at `exec - init == W` (it may be a message sent before
  the source's upgrade, which can never expire).
- `NoRefundOfM2`: a refund is routed to a source other than A.
- `NoRefundOfM3`: a refund happens for a destination that joined the lockbox after genesis.

## Instances and expected results

| Instance | Configuration | Expected |
|---|---|---|
| `safe` | exporter trusted, all three L1 checks, target rule, joins guarded, W=7, P=8 | `Safety` holds; witnesses violated |
| `safeNoTargetRule` | no target rule | holds: the rule is defense in depth in this design |
| `safeNoMargin` | P = W = 7 | holds with the strict `>` |
| `safeShorterWindow` | W = 5 | holds |
| `safeResendRestarts` | a hypothetical resend that restarts the timestamp **and** refuses an expired message | holds; restarting alone is not enough, since a resend after the refund would open a new relay window |
| `messengerTrustedPrestaged` | earlier design (0x..23 trusted), even with the target rule | double spend: forgeries made before the upgrade |
| `messengerTrustedNoTargetRule` | earlier design, no target rule, with pre-upgrade forgeries switched off | double spend on the post-upgrade path |
| `periodBelowWindow` | W = 7, P = 6 (within the cap) | double spend |
| `expireGeNoMargin` | `>=` with P = W | double spend at the boundary |
| `noRealMessengerCheck` | check (a) off | double spend: a fake caller borrowing a real portal |
| `noLockboxCheck` | check (b) off | double spend: a non-cluster chain, or a fake caller with a fake portal |
| `noSenderCheck` | check (c) off | double spend: a user withdrawal (or a pre-upgrade relay through the messenger) |
| `nonstandardJoin` | a chain that ran arbitrary code joins the lockbox | double spend: its old withdrawals |
| `resendNoRestart` | the original #22601 resend flaw | double spend |
| `duplicateChainId` | C (a standard chain that can join) has B's chain ID | double spend: member B exports "not relayed" for `m3`, which is addressed to that ID, and C relays it without joining (9 steps); C exporting for B's message needs a join first |

**Activation.** In the exporter design, the "no legacy forged facts" premise is encoded in the
action set rather than assumed separately: before a chain's upgrade its exporter proxy has no
implementation, so `exportUndelivered` requires `upgraded`, and no other action on a standard chain
records the exporter as sender. That rests on the completeness of the hand-enumerated actions; the
Lean model derives it (`exporterSilentBeforeUpgrade`). What remains is weaker
than v1's premise: no ProxyAdmin action set an implementation at the exporter address before the
upgrade, on any chain that is or becomes part of the lockbox. `messengerTrustedPrestaged`
shows the same premise failing for the earlier design.

## Results

Apalache 0.62.1 via `quint verify`, on a 32-core Linux host, each check memory-capped (8–16 GB).

**Unsafe instances.** Every one violates `NoDoubleSpend`, within the default `DEPTH` of 15. The
column is the step at which Apalache reports the shortest violation.

| Instance | Violation at step | Time |
|---|---|---|
| `noRealMessengerCheck`, `noLockboxCheck` | 6 | 22–23 s |
| `messengerTrustedPrestaged`, `noSenderCheck` | 7 | 22–30 s |
| `messengerTrustedNoTargetRule`, `nonstandardJoin` | 8 | 27–28 s |
| `periodBelowWindow`, `expireGeNoMargin`, `duplicateChainId` | 9 | 27–43 s |
| `resendNoRestart` | 11 | 346 s |

**Witnesses.** In every safe instance `NoRefundEver` and `NoRefundOfM2` are violated at step 8,
`NoRefundOfM3` at step 9 and `NoEdgeRelay` at step 3 (22–60 s each). `MessengerSilentAfterUpgrade`
holds in `safe` at depth 15 (332 s) and is violated in `safeNoTargetRule` at step 2.

**`Safety` in the safe instances is checked to a smaller depth than 15.** The symbolic state grows
quickly with depth: step 11 alone ran for more than five hours per instance without finishing. The
depths below are the last step at which Apalache finished checking every conjunct of `Safety`
for every execution of that length; the runs were then stopped.

| Instance | `Safety` holds for every execution of up to |
|---|---|
| `safe`, `safeNoTargetRule`, `safeNoMargin`, `safeShorterWindow` | 10 steps |
| `safeResendRestarts` | 9 steps |

What this bound covers: an honest refund takes 8–9 steps, and every unsafe instance except
`resendNoRestart` double-spends within 9 steps, so for those mitigations the safe instance is
checked at least as deep as the shortest attack its removal enables. The `resendNoRestart`
attack needs 11 steps, deeper than `safeResendRestarts` was checked. Unbounded safety for all
of these configurations is the Lean proof (`safety`, `safety_without_targetRule`,
`safe_variants`); this model is the bounded cross-check and the source of the counterexamples.

## Assumptions (not shown by this model)

- **Timestamps.** Per-chain timestamps are monotone, and the source records `sentAt` as the
  initiating event's timestamp (same transaction). Relay validity and expiry compare the same pair of
  clocks, so cross-chain drift doesn't matter.
- **Finality.** Withdrawals reflect their chain's canonical history.
- **Standard chains run the standard code**, and `relayMessage` only succeeds within the window.
- **Chain IDs.** No chain, inside the lockbox or not, shares a chain ID with a standard chain. This
  is the Lean model's `ChainIdUnique` (`../lean/Expiry/Model.lean`) and the rollout model's AC4.
  Uniqueness among lockbox members alone is not enough: a non-member with a member's chain ID can
  relay that member's messages (`../rollout`, trace `cexNonMember`). This model's
  `duplicateChainId` gives the duplicate ID to C, a standard chain that can join. OPCM's migration
  checks for duplicate IDs among the chains it migrates; the rest is a governance obligation.
- **The protocol W rule** is enforced on a destination before its exporter goes live, and W never
  rises above P later. Changes of W over time are not modeled.
- **Upgrades are monotone and storage-preserving.** `upgrade` happens once and is never undone,
  and every map survives it. In particular, once a message has a send timestamp, no later
  implementation may re-emit its `SentMessage`, including a rollback to the pre-expiry messenger
  with `resendMessage` (rollout AC2; `../rollout` `govMessengerDowngrade` shows the double spend).
- **Relay is pinned to the message's assigned destination chain.** The contracts accept a relay on
  any chain whose `block.chainid` equals the destination. With unique chain IDs (every safe
  instance) that is the same chain; with `UNIQUE_IDS` off the model omits relays of a message on
  the other chain with the same ID, so `duplicateChainId` under-approximates what that
  misconfiguration allows (it double-spends regardless).
- **Addresses.** No EOA or aliased L1 address equals the exporter's or the messenger's address
  (preimage hardness).
- **Exporter governance.** Each cluster chain's L2 governance (its L2 ProxyAdmin owner) can upgrade
  its own exporter, which would let it forge facts for any destination. This is comparable to the
  trust in the shared ETHLockbox: a member's L2 governance can already make arbitrary withdrawals
  from it by changing its own L2 state (the lockbox's own check compares the portals' L1 ProxyAdmin
  owners, a separate role). It is not modeled; a chain whose exporter was replaced is a non-standard
  chain, as in `nonstandardJoin`.
- **Legacy withdrawals.** Pre-Bedrock (version 0) withdrawals are irrelevant.
- **The protocol rule's `<=`** is the conservative reading.
- **One lockbox.** There is a single authorization set, while the code reads the route's own lockbox.
  Lockbox migrations and asymmetric lockboxes are not modeled.
- **Governance:**
  - only standard chains are authorized in a lockbox (`JOIN_REQUIRES_STANDARD`); the
    `nonstandardJoin` counterexample shows why it is needed;
  - an authorized portal's SystemConfig names that chain's real L1CrossDomainMessenger;
  - no ProxyAdmin set an implementation at the exporter address
    (`Predeploys.UNDELIVERED_MESSAGE_EXPORTER`) before the upgrade.
- **`RefundImpliesExpired` and `AtMostOneRefund`** follow directly from the guards of `refund`; they
  are kept as regression properties.
- **Hash binding is abstracted** as message identity, destination and source.
  - A collision-free hash is assumed.
  - The refund's preimage binding (the bridge as sender, `relayETH(from, to, amount)`) is not
    modeled. Halmos and Kontrol check it on bytecode.
- **EVM checks are abstracted into guards.** The reverse binding, the `xDomainMessageSender`
  semantics, the `expireMessage` caller check, and the L1 messenger never relaying to itself are
  taken as given here; Halmos and Kontrol check them on bytecode. The INTEROP gate (landed at
  `5992028e08`) is not modeled; it can only restrict.
- **Bounded.** Three messages, four chains, `MAX_TIME` = 20, and Apalache up to the step bounds above.
  - An honest refund takes 8–9 steps.
  - The attacks' shortest double spends take 6 steps (`fakeCaller` paths) to 11 (`resendNoRestart`).
  - The default `DEPTH` is 15. `Safety` in the safe instances was checked to 10 steps, 9 for
    `safeResendRestarts` (see Results); `SAFE_DEPTH` and `RESEND_SAFE_DEPTH` default to those.
  - The Lean proof covers unbounded chains, messages and time.
- **Not modeled:** gas, ETH amounts, pauses, proof-maturity delays and message nonces. ETH amounts
  are covered by the Foundry invariant harness.
- **The L2ToL1MessagePasser target rule** (landed in `3b8d14c4ef`) matters only for external L1
  contracts that might trust raw withdrawals from 0x..23. No protocol contract does, so it is out
  of scope here; Halmos and Kontrol check it on bytecode.

## Review log

**v1**, reviewed by R1, R2 and R3 (independent model-based reviewers). All findings were addressed in v2
except where marked "partly".
- **All reviewers:** the unsafe instances were checked against `Safety`, so the counterexamples
  stopped at an earlier conjunct. The runner now asserts that `NoDoubleSpend` itself is violated.
- **R2, R3:** `run.sh` returned 0 regardless of results. It now asserts every outcome and exits
  nonzero on any surprise.
- **R1:** lockbox membership was static. `join` is now modeled, with the check read at relay time,
  plus the governance assumption and the `nonstandardJoin` counterexample.
- **R1:** only one of the three L1 checks was exercised. `fakeCaller`, `userWithdrawal` and
  per-check instances are added.
- **R1:** `exportUndelivered` had a fixed route and a single source. Routes are now chosen by the
  caller and there are two sources.
- **R2, R3:** attacker coverage was assumed. Partly addressed: the attacker is now written per
  target and per recorded sender, and arbitrary code on non-standard chains is modeled. Completeness
  is still enumerated by hand; the Lean model derives it.
- **R2, R3:** `externalRawTrust` modeled a privileged path that doesn't exist. It is removed, and
  the scope note above replaces it.
- **R1:** the windows didn't match the code. The model is now in days with W=7, P=8, and an
  instance where P = W.
- **R1, R3:** the README and runner disagreed on depth, the results file was missing, and there
  was no boundary mutation. `DEPTH` is now consistent, the results are recorded in the top-level
  README, and `expireGeNoMargin` is added.
- **R1, R2:** a single global clock was a stronger assumption than needed. Clocks are now per
  chain.
- **R2:** the refund's preimage binding was absent. It is now an explicitly stated delegation to
  Halmos and Kontrol.

**v1 fixes from the Lean v2 review**, which applied to this model too:
- **R1:** a fake L1 caller can pass check (a) with a fake portal and SystemConfig; only check (b)
  stops it. `fakeCaller` is now guarded by `¬(a) ∨ ¬(b)`.
- **R1:** "0x..2E never had code" was wrong. The genesis proxy has no implementation, so the
  residual assumption is that no ProxyAdmin action set one before the upgrade. Both the model text and
  the README are corrected.

- **R2:** pre-upgrade resends were not representable. `resend` is now enabled before the
  source's upgrade.

**v2**, reviewed by R1, R2 and R3 (independent model-based reviewers). R3 found no critical or high issue; its
medium and low findings duplicate the runner, depth, joins, witness and documentation items below.
Each finding and what became of it:
- **R1 C1, R2:** the README didn't say the code at this commit is the model's own
  counterexample. Added the scope box at the top.
- **R1 H1:** "only the exporter can make the trusted sender speak" was described as derived; it
  is hand-enumerated here. Reworded, with a pointer to the Lean derivation.
- **R1 H2, R2:** chain-ID uniqueness was missing. Added `CHAIN_ID`, `UNIQUE_IDS`, the
  `duplicateChainId` counterexample and a governance assumption.
- **R1 M1:** W activation and W changing over time. Named as assumptions; not modeled.
- **R1 M2, R2:** depth, step counts and results were misstated. `DEPTH` is now 15, the step
  counts are corrected, and the results are in `../README.md`.
- **R1 M3, R2:** joins were vacuous in the safe instances. Added `m3` to C, which joins later,
  and the `NoRefundOfM3` witness.
- **R2:** the INTEROP gate was claimed checked on bytecode. Marked as not landed and not modeled.
- **R2:** the runner accepted unknown modes and treated tool errors as violations. It now
  validates the mode and classifies errors separately; witnesses are checked in every safe instance.
- **R1 L2, R2:** some counterexamples didn't isolate their attack:
  - `messengerTrustedNoTargetRule` now switches off pre-upgrade forgeries;
  - `arbitraryCode` only works outside the lockbox, so `nonstandardJoin` shows withdrawals made
    before the join;
  - `noLockboxCheck` keeps two paths, both described.
- **R1 L1:** two properties follow from guards. Noted.
- **R1 L4:** missing assumptions (legacy withdrawals, addresses, `<=`). Added.
- **R1 L5:** overclaiming "every finding fixed". Reworded.
- **Later additions:**
  - pre-upgrade `resend` (from the Lean v2 review);
  - `fakeCaller` passing check (a) with a fake portal (from the Lean v2 review);
  - the `MessengerSilentAfterUpgrade` defense-in-depth property (the design keeps the target rule).

**v2.1** (the exporter design, chain IDs, `m3` and joins, pre-upgrade resend, the `fakeCaller` guard,
`MessengerSilentAfterUpgrade`, the results and depth knobs), reviewed by R1, R2 and R3. None found a
critical or high issue or a way the contracts break a stated property; all confirmed that the action
guards match the code, that the safe instances' guards are not stronger than the code, that the
witnesses are substantive (`NoRefundOfM3` needs C to join), and that the chain-ID assumption matches
Lean's `ChainIdUnique`. Each finding and what became of it:
- **R1, R2, R3: `duplicateChainId`'s mechanism was misdescribed.** The 9-step violation is member B
  exporting "not relayed" for `m3` (addressed to the shared ID) and C relaying it without joining; C
  exporting for B's message needs a join (10 steps). Fixed in the instance table and the model comment.
- **R2, R3: relay is pinned to the assigned destination chain.** With duplicate IDs the contracts would
  also accept the relay on the other chain. Stated as a limitation of the `duplicateChainId` instance
  (it under-approximates that misconfiguration, which double-spends regardless); safe instances have
  unique IDs. Not changed in the model.
- **R2, R3: `safeResendRestarts` also refuses expired messages.** Restarting the timestamp alone is not
  enough. Fixed in the instance table and the `ExpiredImpliesNeverRelayable` text.
- **R2, R3: missing rollback assumption.** Added "upgrades are monotone and storage-preserving", with
  rollout AC2.
- **R1, R2, R3: "derived activation" overclaimed.** Reworded here and in the model header: the
  premise is encoded in the hand-enumerated action set; Lean derives it.
- **R1: the `UNIQUE_IDS` comment stated the weaker, members-only assumption.** Fixed.
- **R2, R3: the runner accepted a deadlock trace as the expected violation.** `classify` now requires
  quint's invariant-counterexample message and treats a deadlock as an error (checked against quint
  0.33 output).
- **R1, R2, R3: `wait -n` needs bash 4.3; `JOBS=0` hung.** The runner checks the bash version and that
  `JOBS` is a positive integer, and states the memory cap used for the results.
- **R2, R3, R1: `MessengerSilentAfterUpgrade` misses exports in messenger-trusted instances.** Scoped to
  the exporter-trusted instances, where it is checked; R1 noted it holds in `safe` directly from the
  guard.
- **R1: `NoEdgeRelay` can be met by a pre-upgrade message; `noSenderCheck` can also use a pre-upgrade
  relay; the scope box's "(with P = W)" named no instance; "resend is gone" was stale.** Fixed.
- **R1: `ExpiredImpliesNeverRelayable` catches a non-restarting resend at step 9, within the bound.**
  Noted; the README keeps the conservative statement that the `resendNoRestart` double spend (11 steps)
  is deeper than `safeResendRestarts` was checked.
- **R1: simulate results not recorded.** `./run.sh simulate` (20,000 samples, 30 steps) was re-run with
  this runner: every expected outcome, no failures.
