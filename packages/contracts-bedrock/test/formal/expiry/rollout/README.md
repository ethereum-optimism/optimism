# Rollout model: activation orderings and misconfigurations for interop expiry

`rollout.qnt` asks which **deployment orderings and misconfigurations** of the exporter design are
safe, and which activation conditions are needed. The design is on `karl/message-expiry-refunds` at
`5992028e08`. The cited code was re-checked at tip `e1b3903ab8`. The changes there are error
renames, an `UndeliveredMessageExported` event, the exporter's move to `0x4200…0030`, and a kona
getter guard; none of them changes this model. The model is a copy of `../quint/expiry.qnt` with
the atomic `upgrade(x)` split into separate events per chain and per layer, which can happen in any
order.

Results in short:
- **No ordering violates safety within the bound** when five activation conditions, AC1–AC5,
  hold; the tip's code already meets AC5. The bound: Apalache found no violation in any execution
  of up to 11 events of this finite model (3 chains, 3 messages, times up to 16 days). That depth is
  not a completeness threshold; see **Results**. The orderings explored include: the L1 before or after the L2; chain
  by chain; source before destination or the reverse; the locked Lagoon bundle; an exporter that
  goes live before its messenger records timestamps, or after; L1, bridge and exporter rollbacks;
  a messenger rollback before any timestamp; and a mid-rollout lockbox join.
- **Each condition is individually indispensable**: dropping any one of them, with the others
  kept, gives a concrete double spend. This does not make AC1–AC5 the weakest possible conditions
  (see **Activation conditions**).
- **One condition is only partly enforced by code.** AC2 says that once a source chain's messenger
  has recorded a timestamp, it is never rolled back below 2.0.0. The L2ContractsManager's semver
  guard enforces this on the NUT path. The L2 ProxyAdmin owner's `upgrade` path has no guard, and a
  rollback there gives a double spend. Deleting one line (2.0.0's `sentMessages[nonce] = …` write)
  makes every rollback safe, and that is checked too (`downgradeHardened`).
- **The locked Lagoon NUT bundle**, which is the bundle of the interop activation fork, still ships
  the pre-expiry contracts. It is safe, but it disables the feature, permanently for the messages
  sent while it is in force. See BC1.

All statements below are about this model. "Checked" means Apalache bounded model checking to the
stated depth (section **Results**). "Trace" means a scripted `run` that `quint test` executes; it
needs no Java.

## What is modeled

**Chains, messages and time.**
- Chains A, B and C all run standard code unless an event says otherwise. A and B start in the
  shared ETHLockbox, already migrated with INTEROP set on L1. C starts outside and can `join` at any
  point.
- There are three bridge sends: `m1` goes A→B, `m2` goes B→A, `m3` goes A→C.
- Time is in days. Clocks are per chain and only move forward. `MAX_TIME` = 16, and `INF` = 99
  stands for "no W rule".
- There is no separate `tick` action. The actions that read a chain's clock (`send`, `resend`,
  `relay`, `exportStd`) take the time `t` they happen at, which may be any `t ≥` that chain's clock,
  and move the clock to `t`. No other action reads a clock, so this gives the same reachable
  behaviors with fewer steps. Among the properties, only `ExpiredImpliesNeverRelayable` reads a
  clock, and a later clock only makes it easier to satisfy.

**Initial state (before the rollout).** Every L2 runs messenger 1.3.1, the bridge without
`refundETH`, and an exporter proxy with no implementation. Every L1 runs the old
L1CrossDomainMessenger, which has no `relayUndeliveredMessage`.

**Activation events.** Each is enabled on every chain at every step, so Apalache explores every
interleaving up to the depth bound.

| Event | Stands for (code) |
|---|---|
| `l1Set(x, v)` | OPCM upgrade or rollback of x's L1CrossDomainMessenger. The versions are: `old` (no `relayUndeliveredMessage`); `new` (tip: INTEROP gate plus checks (a), (b) and (c), with the exporter as the trusted sender); `earlier` (only with `L1_EARLIER_DESIGN`: the unreleased PR-branch design before `1086b6de3e` that trusts 0x..23, modeled without the INTEROP gate; `cf3d730591` added the gate, which delays the attack only until the route's INTEROP is set, as it already is for A and B). |
| `l1EnableInterop(x)` | `SystemConfig.setFeature(INTEROP)`, set by OPCM migrate. It can't be cleared. |
| `nutTip(x)` | The tip's bundle (`snapshots/upgrades/current-upgrade-bundle.json`). The L2ContractsManager installs messenger 2.0.0, bridge 1.1.0 and the standard exporter 1.0.0 in one transaction. |
| `nutLagoon(x)` | The **locked Lagoon bundle** (`op-core/nuts/bundles/lagoon_nut_bundle.json`, `fork_lock.toml` commit `fa9974a2`): messenger **1.3.1**, bridge **1.0.1**, and no exporter entry, so the exporter is untouched. With `L2CM_DOWNGRADE_GUARD`, the whole `upgradePredeploys` call reverts if the messenger or bridge is already newer. The guard that runs at Lagoon is the one in the locked bundle's own L2ContractsManager (`fa9974a2:L2ContractsManagerUtils.sol:49-62`, and `:132` for `upgradeToAndCall`). In the real bundle the messenger and bridge upgrades also require L1Block INTEROP (`fa9974a2:L2ContractsManager.sol:397`), which op-node sets only for a multi-chain dependency set (`op-node/rollup/derive/attributes.go:171-182`). `nutLagoon` ignores that gate, which can only add executions. |
| `setMessengerNew(x)`, `setBridge(x, b)`, `setExporter(x, v)` | One predeploy at a time: a split bundle, a later fork, or governance through `ProxyAdmin.upgrade`. The bridge can go in either direction. The exporter can be set to `none`, `std`, or `rogue` (non-standard code); `rogue` is allowed only per `ROGUE_EXPORTER_MEMBER` / `ROGUE_EXPORTER_OUTSIDER`. |
| `govMessengerDowngrade(x)` | The L2 ProxyAdmin owner sets the messenger back to 1.3.1. `ProxyAdmin.upgrade` (`src/universal/ProxyAdmin.sol:152`) has no version check. `GOV_MSGR_DOWNGRADE` is `never`, `beforeTimestamps` (only while no message from x has a timestamp; this is what `rolloutSafe` allows) or `any`. |
| `join(x)` | `ETHLockbox.authorizePortal` or OPCM migrate. With `JOIN_REQUIRES_CLEAN_EXPORTER`, a chain can join only if its exporter address never ran non-standard code. |

**Protocol actions.** These are as in `../quint/expiry.qnt`, but guarded by the per-chain
deployment state:
- `send(m)`: both messenger versions send, and only 2.0.0 records `sentAt`. Both write
  `sentMessages[nonce] = H`; `NEW_WRITES_SENT_MESSAGES = false` models the hardening that stops
  2.0.0 from writing it.
- `resend(m)`: 1.3.1 only. It requires `sentMessages[nonce] == H` (1.3.1
  `L2ToL2CrossDomainMessenger.sol:192` at `fa9974a2`), re-emits the event at the current time, and
  never touches `sentMessageTimestamps`.
- `relay(m, w)`: enabled if some initiating event `e` has `t <= e + w`. That is only the upper
  bound of the real rule. The real rule also requires `init ≤ exec`, dependency-set membership and
  activation checks (`op-core/interop/depset/links.go:47-74`). Dropping them is conservative: it
  only adds relays. The window `w` is
  picked **per relay** from the destination's window set: `WIN_BEFORE[d]` until d's exporter has
  first had an implementation, and `WIN_AFTER[d]` from then on. A set therefore covers W changing
  over time, a W rule that isn't active yet (`INF`), and a host-chosen depset (kona's fallback;
  see AC1).
- `exportStd(x, m, route)`: needs x's exporter to be `std`. The hash binds `CHAIN_ID[x]`, and the
  export reverts if x relayed the message. It may name any preimage, including one not sent yet,
  and any route.
- `exportRogue(x, …)`: needs x's exporter to be `rogue`. The fact can have any content, and the
  recorded sender is the exporter.
- `messengerForge(x, …)`: needs messenger 1.3.1 on x, which has no target rule. A relay to the
  L2CrossDomainMessenger produces a withdrawal whose recorded sender is 0x..23. This one event
  stands for the attacker's initiating send plus its relay.
- `l1Relay(w)`: the withdrawal comes from a lockbox member other than the route. For a `new` L1 it
  needs the route's INTEROP and the exporter as the recorded sender; for an `earlier` L1, 0x..23.
  Withdrawals and deposits are never consumed. Failed L1 relays and failed L2 deposits really do
  stay replayable (`failedMessages`). Successful ones cannot be replayed in reality
  (`CrossDomainMessenger.sol`), so keeping them is a conservative over-approximation.
- `expire(dep)`: needs messenger 2.0.0 on the source. On 1.3.1 the call reverts and stays
  replayable. It also requires `sentAt != 0` and `at > sentAt + P`.
- `refund(m)`: needs bridge 1.1.0 **and** messenger 2.0.0 on the source, since `refundETH` reads
  `expiredMessages`. The message must be expired and not yet refunded; `refunded` persists in
  storage across bridge rollbacks.

## Properties

The names are the same as in `../quint`.
- `Safety` is `NoDoubleSpend ∧ RefundImpliesExpired ∧ NoForgedFact ∧ AtMostOneRefund`. This is what
  the deep checks use.
- `SafetyFull` is `Safety ∧ ExpiredImpliesNeverRelayable`. The extra conjunct is stated against
  every window in `WIN_AFTER[dest]`. It is stronger than the double-spend property and by far the
  most expensive for the solver, so it is checked to a smaller depth (see **Results**).

## Non-vacuity

Every safe result comes with automated checks that its interesting states are reachable, so
`Safety` does not hold just because nothing happens. There are two kinds, and `run.sh` fails if
either one does not come out as expected:
- **Witness invariants**, checked by Apalache (`run.sh verify`). Each one says that some state
  never happens, so each must be **violated**. The runner fails if Apalache reports one as holding.
- **Scripted traces**, run with `quint test` (`run.sh test`). `witness*` traces drive a concrete
  ordering and assert that `Safety` holds and the witness is violated at the end. `blocked*` traces
  assert that a step a condition forbids is disabled. `cex` traces reproduce each double spend.

**Witness invariants in `rolloutSafe`.** All must be violated, and each is also driven by a
scripted trace. The last six have their own short traces: `witnessLateInteropEnable`,
`witnessMessengerWithdrawalRejected`, `witnessRogueOutsiderRejected`, `witnessResendOn131`, plus
`witnessRefundM3JoinMidRollout` for `NoJoin`:

| Witness | Ordering it shows is reachable *and* safe |
|---|---|
| `NoRefundEver`, `NoRefundOfM2` | the honest refund, in both directions (source upgraded before destination and the reverse) |
| `NoRefundOfM3` | a refund for a destination (C) that **joins the lockbox mid-rollout**; in the trace, C exports before it joins |
| `NoEdgeRelay` | a relay at exactly `exec − init = 7` after the destination's exporter is live |
| `NoLateRelayBeforeExporter` | a relay with `exec − init > 7` (no W rule) **before** the destination's exporter is live |
| `NoDeferredL1Fact` | a fact exported while the route's **L1 was not upgraded** is later accepted by that L1 (the L2 goes before the L1). Apalache's witness stops at the L1 acceptance (4 steps); the scripted trace carries it through to the expiry and the refund |
| `NoRefundExporterFirst` | the destination's **exporter goes live while the source still runs 1.3.1** (no timestamps), then a refund |
| `NoRefundAfterLagoon` | the **locked Lagoon bundle activates first**, then the tip bundle, then a refund |
| `NoRefundAfterLagoonReverted` | the Lagoon bundle after the tip bundle: the guard reverts it (the trace asserts that 2.0.0 stays), then a refund |
| `NoRefundAfterL1Rollback`, `NoRefundAfterBridgeRollback`, `NoRefundAfterExporterRemoved` | L1 rolled back and re-upgraded; bridge rolled back and re-upgraded; exporter removed after it exported |
| `NoRefundAfterMessengerRollback` | the source's messenger rolled back to 1.3.1 **before any timestamp** and re-upgraded, then a refund (AC2's scoping) |
| `NoJoin`, `NoLateInteropEnable` | a lockbox join; INTEROP enabled on L1 for a chain that started without it |
| `NoRogueExporterEver`, `NoRogueWithdrawal` | non-standard exporter code (an outsider, in `rolloutSafe`) and a withdrawal from it (AC3's scoping) |
| `NoMessengerWithdrawal`, `NoResend` | a 0x..23 withdrawal through a 1.3.1 messenger without the target rule; a `resendMessage` on 1.3.1 |

**Other safe instances.** In `rolloutWindowAtP`, `NoRefundEver`, `NoEdgeRelay`, `NoRefundOfM3`
and `NoRefundAfterLagoon` must be violated; the trace `witnessRelayAtP` relays at exactly
`exec − init = 8`. In `downgradeHardened`, `NoRefundEver`, `NoRefundAfterMessengerRollback` (here
a rollback after timestamps) and `NoResend` must be violated. Its traces show the rollback after a
refund, where the resend is then impossible, and a 1.3.1 resend of its own message.

**Unsafe instances.** The violated `NoDoubleSpend` is itself the reachability result. Each one also
has a scripted `cex` trace.

The `blocked*` traces in `rolloutSafe` check that the conditions really bind. These steps are
disabled there:
- the governance downgrade after a timestamped send;
- `resend` after the Lagoon bundle on 2.0.0;
- a relay with an uncapped window after the exporter;
- the join of a chain whose exporter was rogue;
- the earlier L1 design.

## Orderings and misconfigurations: results

"Safe" means `Safety` is checked to hold in `rolloutSafe` for every interleaving of up to **11** events (see **Results**),
where the ordering is one of the interleavings explored. The witness column shows that the
ordering is actually exercised. "Double spend" means `NoDoubleSpend` is violated: Apalache finds a
counterexample and the scripted `cex` trace reproduces it.

| # | Ordering / misconfiguration (task item) | Result | Instance / witness |
|---|---|---|---|
| 1a | L1 upgraded before L2 (relayUndeliveredMessage live, exporter has no implementation) | safe: no trusted-sender withdrawal can exist | `rolloutSafe` |
| 1b | L2 before L1 (facts exported while route's L1 is old) | safe: the facts are honest and stay true; they are accepted later | `NoDeferredL1Fact` |
| 1c | chain by chain, source before destination and the reverse | safe | `NoRefundOfM2`, `NoRefundExporterFirst` |
| 2a | locked Lagoon bundle activates first (1.3.1, bridge 1.0.1, no exporter), tip later | safe; messages sent under 1.3.1 never expire (BC1) | `NoRefundAfterLagoon` |
| 2b | Lagoon bundle after tip (guard on) | safe: the whole L2CM call reverts | `NoRefundAfterLagoonReverted` |
| 2c | Lagoon bundle after tip **without** the semver guard | **double spend** (11 steps) | `lagoonWithoutGuard` |
| 3a | L2 messenger rolled back to 1.3.1 by governance after messages were sent with timestamps | **double spend** (11 steps) | `govMessengerDowngrade` |
| 3b | 3a with the hardening "2.0.0 does not write `sentMessages`" | safe | `downgradeHardened` |
| 3b' | messenger rolled back to 1.3.1 before the chain recorded any timestamp, re-upgraded later | safe | `NoRefundAfterMessengerRollback` |
| 3c | L1 messenger rolled back to old and re-upgraded | safe | `NoRefundAfterL1Rollback` |
| 3d | L1 messenger set to the earlier design (trusts 0x..23) at any point | **double spend** | `l1EarlierDesign` |
| 3e | bridge rolled back / exporter removed | safe | `NoRefundAfterBridgeRollback`, `NoRefundAfterExporterRemoved` |
| 4a | W ≤ 7 per relay, changing between relays, after the exporter is live (covers kona host-fallback depset, which is serde-capped) | safe | `rolloutSafe` (`WIN_AFTER = {5, 7}`) |
| 4b | W = P = 8 | safe (strict `>` in `expireMessage`) | `rolloutWindowAtP` |
| 4c | W > P on one chain (C uses 9) after its exporter is live | **double spend** | `windowAbovePOnC` |
| 4d | no W rule on a destination whose exporter is live | **double spend** | `noWindowRuleOnB` |
| 4e | no W rule (or any W) **before** the destination's exporter is ever live | safe | `NoLateRelayBeforeExporter` |
| 5a | exporter live before the source messenger records timestamps | safe: those messages have `sentAt = 0` and never expire | `NoRefundExporterFirst` |
| 5b | source records timestamps before the destination's exporter is live | safe | `rolloutSafe` |
| 6a | chain joins the lockbox mid-rollout (clean history) | safe | `NoRefundOfM3` |
| 6b | chain with past non-standard exporter code joins | **double spend** | `rogueThenJoin` |
| 6c | lockbox member's governance installs non-standard exporter code | **double spend** | `rogueExporterMember` |
| 6d | a chain that can relay members' messages shares a member's L2 chain ID, whether or not it joins | **double spend** | `duplicateChainId` (Apalache's shortest counterexample, 9 steps, has no join: trace `cexNonMember`; trace `cex` has a join) |

## Activation conditions (checked to 11 events)

`rolloutSafe` assumes AC1–AC5 and leaves every other modeled event free. Each unsafe instance
relaxes exactly one of them, keeps the others, and double-spends. So the set is **individually
indispensable under the modeled relaxations**. It is not shown to be the weakest characterization.
For example, briefly installing the earlier L1 implementation without any forged withdrawal ever
being relayed would be safe. So would rolling back the messenger after all of its timestamped
messages had been relayed. Both are outside AC5 and AC2 as stated.

- **AC1 (window).** For every chain d, from the moment d's exporter first has an implementation,
  every relay on d is judged with `W ≤ P` (P = 8 days), forever after.
  - *Scoped*: before that moment, d may relay with any window, including none
    (`NoLateRelayBeforeExporter`).
  - *Tight*: W = P is safe (`rolloutWindowAtP`).
  - *Necessary*: `noWindowRuleOnB` (no rule) and `windowAbovePOnC` (W = 9 on one chain).
- **AC2 (no resend after timestamps).** Once a source chain's messenger has recorded a timestamp
  for some message, it never again runs code that can re-emit that message's `SentMessage`. For
  the released versions: never below 2.0.0.
  - *Scoped*: a rollback before the chain recorded any timestamp is safe; `rolloutSafe` allows it
    (`NoRefundAfterMessengerRollback`).
  - *Necessary*: `govMessengerDowngrade`, and `lagoonWithoutGuard` for the NUT path.
  - *Can be discharged by code*: `downgradeHardened`.
- **AC3 (exporter provenance).** On every chain that is, or later becomes, a member of the
  lockbox, the exporter address only ever ran the standard implementation or nothing.
  - *Scoped*: outsiders may run anything while they are outside (`ROGUE_EXPORTER_OUTSIDER = true`
    in `rolloutSafe`); they just can't join afterwards.
  - *Necessary*: `rogueExporterMember`, `rogueThenJoin`.
- **AC4 (chain IDs).** No chain that can relay messages sent from a lockbox member shares the L2
  chain ID of a lockbox member, or of another such chain. In practice: L2 chain IDs are unique
  across members and every chain in their dependency sets. `rolloutSafe` assumes all IDs are
  unique.
  - Uniqueness among members alone is **not** enough. In `duplicateChainId`'s shortest
    counterexample (9 steps, trace `cexNonMember`), C has B's chain ID and never joins.
    1. B (a member) exports "not relayed here" for m3, which is addressed to that chain ID.
    2. A refunds m3.
    3. C relays m3 and mints the ETH again on C.
  - *Necessary*: `duplicateChainId`.
- **AC5 (L1 implementation).** No L1CrossDomainMessenger that trusts 0x..23 for
  `relayUndeliveredMessage` is ever installed, even briefly: 1.3.1 L2 messengers let anyone make
  0x..23 speak, and such withdrawals never expire. *Necessary*: `l1EarlierDesign`. The tip's code
  meets it (check (c) compares against `Predeploys.UNDELIVERED_MESSAGE_EXPORTER`).

**Not needed within the bound** (no condition was required; `rolloutSafe` leaves them free):
- the relative order of the L1 upgrade, the messenger, bridge and exporter upgrades, and lockbox
  joins;
- the order across chains;
- the locked Lagoon bundle before the tip bundle;
- rollbacks of the L1 messenger, the bridge and the exporter, and of the messenger before any
  timestamp.

**The L1 INTEROP flag is argued, not checked.** Both message sources, A and B, start with INTEROP
set, and only C can enable it later. C is never a source, so the late-enable path never reaches an
expiry, and `NoLateInteropEnable` shows reachability only. The argument: the gate only adds a revert
to `relayUndeliveredMessage` (`L1CrossDomainMessenger.sol:113`). A reverted L1 relay stays
replayable in the caller's `failedMessages`, so enabling it later can only delay a fact. Exercising
this needs a model change (a source starting without INTEROP) and a rerun.

In particular there is **no "L1 accepts facts only after …" condition**. Before a member's
exporter has its standard implementation, no withdrawal from the exporter can exist there, and an honest fact stays true
forever because the destination's clock and AC1 are monotone.

### Which conditions the code enforces

| Condition | Enforced by code | Left to process |
|---|---|---|
| AC1 | op-core `StaticConfigDependencySet.hydrate` rejects overrides > 7d (`static_depset.go:141`); kona `DependencySet` serde rejects > 7d (`genesis/src/interop/depset.rs:39`) and `get_message_expiry_window` ignores a larger override set directly (`:54-56`), and that path covers the embedded registry (`registry/src/lib.rs:52`, serde) and the host-fallback depset (`proof-interop/src/boot.rs:281-284`, serde); `op-interop-filter` rejects > 7d (`filter/config.go:80`); op-node's registry depset (`op-node/superchain/depset.go:17-35`) has no override, so W = 7; `EXPIRY_PERIOD = 8 days` | every node, filter and **absolute prestate** that judges a destination's relays must include the cap (or have no override) **before** that destination's exporter goes live; the destination's withdrawals must be finalized by an interop (super-root) proof that enforces W. A pre-cap build with an override > 8d breaks AC1. |
| AC2 | NUT path: `L2ContractsManagerUtils.upgradeTo` reverts on a semver decrease (`:61-67`), so the Lagoon bundle after the tip reverts | **governance path not enforced**: `ProxyAdmin.upgrade` (`ProxyAdmin.sol:152`) can set 1.3.1 (see BC2) |
| AC3 | genesis proxy without implementation; L2CM only sets the standard implementation | the L2 ProxyAdmin owner can set anything (the named governance assumption); `ETHLockbox.authorizePortal` (`ETHLockbox.sol:124`, `_authorizePortal:220`) checks only the shared ProxyAdmin owner and SuperchainConfig, not the joiner's history (BC4) |
| AC4 | `OPContractsManagerMigrator._validateChainSystemConfigs` (`:295-330`) rejects duplicate L2 chain IDs among the chains it migrates | `ETHLockbox.authorizePortal` does not check chain IDs (BC4); nothing on chain checks the IDs of non-member chains in a member's dependency set (dependency-set configuration) |
| AC5 | tip L1CrossDomainMessenger check (c) | never deploy an L1CrossDomainMessenger built from a PR-branch commit before `1086b6de3e` (e.g. `37b44c48c7`, `cf3d730591`), whose check (c) is `L2_TO_L2_CROSS_DOMAIN_MESSENGER` |

## Bug candidates (ranked by "could this be a real bug")

**BC2: a messenger rollback by governance double-spends (AC2, governance path).**
- **Severity.** Real hazard, medium. It needs governance, but an *honest* emergency rollback of
  `L2ToL2CrossDomainMessenger` to the previous release is enough; no malice is required.
- **Evidence.**
  - 1.3.1 has `resendMessage`, which checks only `sentMessages[_nonce] == H`
    (`fa9974a2:src/L2/L2ToL2CrossDomainMessenger.sol:173-196`).
  - 2.0.0 still writes `sentMessages[nonce] = messageHash_` (`L2ToL2CrossDomainMessenger.sol:189` at `e1b3903ab8`),
    but nothing in 2.0.0 or in the monorepo's Go, Rust or TypeScript code reads it; only the
    public getter exposes it.
  - `sentMessageTimestamps` keeps the original time.
- **Trace** (`govMessengerDowngrade::cex`, 11 steps):
  - send `m1` at A-time 1;
  - B exports at B-time 10;
  - the fact is relayed on L1, `m1` expires on A and is refunded;
  - A's messenger is rolled back to 1.3.1;
  - A resends at A-time 12, after the refund;
  - B relays at B-time 15, which is valid because 15 ≤ 12 + 7.
  The rollback happens after the refund, so no re-upgrade is needed. The model's clocks are
  abstract: they do not tie an L2 block's timestamp to its L1 origin or to withdrawal finality. So
  "day 10" for the export does not mean a refund at day 10. The attack needs only that the resend
  comes after the refund and the relay within 7 days of the resend, and every real schedule allows
  that.
- **NUT path.** It is protected only by the semver guard: without it, the Lagoon bundle activating
  after the tip does the same (`lagoonWithoutGuard`).
- **Fix options.**
  - (a) 2.0.0 stops writing `sentMessages`. `downgradeHardened` checks that this makes **any**
    rollback to 1.3.1 safe: 1.3.1 can then resend only its own messages, which have `sentAt = 0`.
    The cost is that the public `sentMessages(nonce)` getter returns zero for messages sent under
    2.0.0. That breaks an existing test assertion (`test/L2/L2ToL2CrossDomainMessenger.t.sol:231`),
    and a rolled-back 1.3.1 can no longer resend messages sent under 2.0.0.
    - The fix must ship in the **first** 2.0.0 deployed on any chain. `downgradeHardened` assumes
      it from genesis, and entries an unhardened 2.0.0 has already written stay resendable.
    - Storage layout and nonces must be preserved.
    - A rollback to 1.3.1 still needs AC5, because 1.3.1 has no target rule.
    - An alternative with the same effect is to write the hash to a new slot.
  - (b) Document AC2 as a governance obligation next to the exporter assumption.

**BC1: the locked Lagoon bundle still ships the pre-expiry L2 contracts.**
- **Severity.** Not a safety bug (checked safe); a feature and process bug.
- **Evidence.**
  - Lagoon is the interop activation fork: `op-node/rollup/toggles.go` `IsInterop = IsLagoon`, and
    `derive/attributes.go:171` runs the Lagoon bundle at activation.
  - `fork_lock.toml` pins `lagoon` to commit `fa9974a2` (2026-05-29). Its bundle deploys
    L2ToL2CrossDomainMessenger **1.3.1** and SuperchainETHBridge **1.0.1**, and has **no
    UndeliveredMessageExporter** entry. These are the version strings in the bundle's initcode;
    `current-upgrade-bundle.json` has 2.0.0, 1.1.0 and exporter 1.0.0.
  - `check-nut-locks` (`ops/scripts/check-nut-locks/main.go`; `justfile`; CI) checks the bundle hash
    against the lock, that the commit is recorded and is an ancestor of `origin/develop`, that the
    pre-fork state file exists, and that every bundle file is locked. None of these checks requires
    the interop fork's bundle to contain the expiry contracts.
- **Consequence if Lagoon ships as locked.**
  - The new L1CrossDomainMessenger (OPCM) has no counterpart.
  - Every message sent while 1.3.1 is live has `sentAt = 0` and can **never** be expired or
    refunded, even after a later upgrade.
  - 1.3.1 keeps `resendMessage` and no target rule.
  - The docs (`message-expiration.mdx`) would overstate the guarantee.
  - Commit `5992028e08` itself defers the state regeneration to "the fork that ships the exporter".
- **Required.** `just nut-snapshot-for lagoon` (the two-PR flow in `op-core/nuts/README.md`) before
  Lagoon is scheduled on any chain that should have expiry.
- **Related: a genesis-vs-NUT consistency issue, not expiry safety.** Take a chain whose genesis
  is built from current code and on which Lagoon activates later.
  - The locked bundle's L2ContractsManager hits its semver downgrade guard. It can do so even before
    the interop block, and even on a single-chain dependency set: tip fee vaults are 1.7.0 against
    the bundle's 1.6.1, and L2DevFeatureFlags is 1.3.0 against 1.0.0. These are version strings in
    the bundle initcode against `custom:semver` at the tip. With the interop gate set, the messenger
    (2.0.0 against 1.3.1) does the same.
  - `L2ProxyAdmin.upgradePredeploys` then reverts (`L2ProxyAdmin.sol:52-55`). The upgrade deposit
    gets a failed receipt and every proxy change in it is undone. The block stays valid.
  - The bundle's separate implementation-deployment deposits and the interop setFeature and
    funding wrappers still execute (`op-node/rollup/derive/lagoon_activation_transactions.go`).
    On such an interop chain, L2 INTEROP ends up enabled while the predeploys keep their genesis
    code.
  - Nothing retries the upgrade.
  - In this model the messenger case is safe (`NoRefundAfterLagoonReverted`). The general case
    is an operational issue for chains whose genesis predates or postdates their bundle.

**BC3: the window cap holds only for software that includes it (AC1).**
- **Severity.** Low.
- **Code side.**
  - The cap is enforced in every parser at the tip, including kona's host-fallback depset. The
    fallback `boot.rs:281-284` deserializes through the capped serde.
  - The fallback is also unprovable on chain: `SuperFaultDisputeGame.addLocalData:611` accepts only
    local idents 1–4 (cases at `:620-632`), and the depset is key 8 (`local_keys.rs:53`). Keys 5–7
    (chain ID, rollup and L1 config fallbacks) have the same gap. Caller-localized keys mean no
    one else can supply the data. A bisection that reaches that read cannot be stepped on chain,
    so the game is decided by its clocks rather than by the program, and an unchallenged proposal
    still wins by timeout. A cluster outside the prestate's embedded registry therefore has no
    working on-chain proof. This is a general misconfiguration, not specific to expiry.
- **Process side.**
  - Before any destination's exporter goes live, no node, interop filter or deployed absolute
    prestate judging its relays may run a pre-cap build with an override above 8 days.
  - Every embedded `depsets.json` today has `overrideMessageExpiryWindow: null`, so W = 7.

**BC4: the lockbox join path skips the migrator's checks (AC3, AC4).**
- **Severity.** Low; this is governance.
- **Evidence.** `ETHLockbox.authorizePortal` checks the shared ProxyAdmin owner and the
  SuperchainConfig only. The duplicate-chain-ID check exists only in
  `OPContractsManagerMigrator._validateChainSystemConfigs`. Neither path can see whether the
  joiner's exporter address ever ran non-standard code.
- **Required.** State AC3 and AC4 as governance obligations for every join.
- **Exporter address.** The address moved from 0x..2E to 0x..30 (at `e1b3903ab8`). That adds a check: the new slot must
  never have had an implementation on any current or future member (AC3).

**Not a bug (checked):** every other ordering in the table.

## Results

The checks use Apalache 0.62.1 through `quint verify` (Quint 0.33.0). Each one ran under a hard
memory cap (`systemd-run --user --scope -p MemoryMax=16G -p MemorySwapMax=0`), with at most two at
once, on a shared 32-core Linux host that was heavily loaded by other jobs, so the times are only
indicative. The honest refund takes 8 steps. The longest counterexamples (`govMessengerDowngrade`,
`lagoonWithoutGuard`) take 11.

**Checks expected to hold.** A depth is "complete" when Apalache finished checking every execution
of that many steps: its log reached `Step k: picking a transition`. Two checks were started with
`--max-steps=12` and stopped by hand once depth 11 was complete, because depth 12 was projected to
take another 10 hours or more. Their logs end with the depth-11 marker; nothing was violated.

| Instance | Invariant | Complete depth | Time to that depth |
|---|---|---|---|
| `rolloutSafe` | `Safety` | **11** | 4 h 24 min |
| `rolloutWindowAtP` | `Safety` | **11** | 3 h 44 min |
| `downgradeHardened` | `Safety` | **11** (run with `--max-steps=11`, finished: no violation) | 4 h 14 min |
| `rolloutSafe` | `SafetyFull` (adds `ExpiredImpliesNeverRelayable`) | **10** (run with `--max-steps=10`, finished: no violation) | 3 h 12 min |

**What depth 11 does and does not show.**
- At depth 11, `rolloutSafe` has no violation in any execution of up to 11 events of this finite
  model. That covers the honest refund (8 events) combined with up to 3 arbitrary further events:
  rollouts, rollbacks, joins, forged withdrawals and so on.
- Depth 11 is also the length of the longest counterexample found against any relaxed condition.
  That is a heuristic for choosing the depth, **not** a completeness threshold: a longer
  execution that combines several permitted moves is not excluded.
- `SafetyFull` is checked only to depth 10.
- The model is finite (3 chains, 3 messages, times up to 16 days). The Lean model in `../lean`
  proves the fully-upgraded design for unbounded executions; the rollout model has no unbounded
  counterpart.

**Checks expected to be violated** (`--max-steps=15`; Apalache stops at the first counterexample,
whose length is given):

| Instance | Invariant | Counterexample length | Time |
|---|---|---|---|
| `noWindowRuleOnB` | `NoDoubleSpend` | 9 | 99 s |
| `windowAbovePOnC` | `NoDoubleSpend` | 10 | 341 s |
| `govMessengerDowngrade` | `NoDoubleSpend` | 11 | 2585 s |
| `lagoonWithoutGuard` | `NoDoubleSpend` | 11 | 2281 s |
| `rogueExporterMember` | `NoDoubleSpend` | 9 | 75 s |
| `rogueThenJoin` | `NoDoubleSpend` | 10 | 241 s |
| `duplicateChainId` | `NoDoubleSpend` | 9 | 72 s |
| `l1EarlierDesign` | `NoDoubleSpend` | 8 | 37 s |
| `rolloutSafe` | `NoRefundEver`, `NoRefundOfM2`, `NoRefundExporterFirst` | 8 | 78 s, 94 s, 47 s |
| `rolloutSafe` | `NoRefundOfM3`, `NoRefundAfterLagoon`, `NoRefundAfterLagoonReverted`, `NoRefundAfterL1Rollback`, `NoRefundAfterBridgeRollback`, `NoRefundAfterExporterRemoved` | 9 | 119 s, 350 s, 104 s, 58 s, 160 s, 188 s |
| `rolloutSafe` | `NoRefundAfterMessengerRollback` | 10 | 3072 s |
| `rolloutSafe` | `NoDeferredL1Fact`, `NoEdgeRelay`, `NoLateRelayBeforeExporter` | 4, 3, 2 | 12 s, 11 s, 10 s |
| `rolloutSafe` | `NoJoin`, `NoLateInteropEnable`, `NoRogueExporterEver`, `NoMessengerWithdrawal`, `NoRogueWithdrawal`, `NoResend` | 1, 1, 1, 1, 2, 2 | ≤ 10 s each |
| `rolloutWindowAtP` | `NoRefundEver`, `NoEdgeRelay`, `NoRefundOfM3`, `NoRefundAfterLagoon` | 8, 3, 9, 9 | 53 s, 8 s, 126 s, 267 s |
| `downgradeHardened` | `NoRefundEver`, `NoRefundAfterMessengerRollback`, `NoResend` | 8, 9, 2 | 53 s, 156 s, 8 s |

**File identity.** All the Apalache checks above ran on the version of `rollout.qnt` with SHA-256
`abd1847e2ed97225f21951a57393297307bb961bcc2afab9126bf6d0c541c892`. The review round (see
**Review log**) changed it in three ways only, none of which affects the invariants or actions
Apalache checks:
- added `run` traces: `cexNonMember`, `witnessLateInteropEnable`,
  `witnessMessengerWithdrawalRejected`, `witnessRogueOutsiderRejected`, `witnessResendOn131`;
- retimed two existing `cex` traces (resend at 12, relay at 15);
- edited the header comment.

The one other result is an earlier full-conjunction run of `rolloutSafe`. It checked all five
conjuncts at once, which is today's `SafetyFull`, completed **depth 9** (in 22 min) and was then
stopped to split the property. It ran on a version that differed only in two ways: it had not yet
added the witness `val`s `NoJoin` … `NoResend`, and it still called today's `SafetyFull` `Safety`.
The Apalache logs are kept on the checking host and not committed (`*.log` is gitignored).

Scripted traces (`./run.sh test`, Quint 0.33.0, Rust backend): all pass. That is 22 in
`rolloutSafe`, 2 in `rolloutWindowAtP`, 2 in `downgradeHardened`, 2 counterexamples in
`duplicateChainId`, and 1 in each of the other 7 unsafe instances.

## Run

```sh
./run.sh test                               # scripted traces only (no Java), seconds
./run.sh verify                             # Apalache: DEPTH=15 (violations), SAFE_DEPTH=11, FULL_DEPTH=10, JOBS=8
ONLY='^govMessengerDowngrade ' ./run.sh verify   # one instance
# Shared host: launch.sh waits for >= 40 GB available, then runs `run.sh verify` with JOBS=2
# under `systemd-run --user --scope -p MemoryMax=16G -p MemorySwapMax=0`; q.sh is `mise exec node@22 -- quint`.
nohup setsid ./launch.sh > verify.out 2>&1 &
```

## Assumptions and limits (not shown by this model)

- **Inherited from `../quint`**:
  - per-chain monotone clocks;
  - finality;
  - collision-free hashing, abstracted as message identity plus destination;
  - L1 checks (a), (b) and (c) as guards, with no fake L1 callers; `../quint` covers those, and
    here all three checks are on;
  - no EOA at a predeploy address;
  - gas, amounts, pauses and proof delays not modeled;
  - one lockbox.
- **Window semantics.** The window a relay is judged by is one value per relay, drawn from the
  destination's set. Disagreement between nodes and the proof is not modeled. Withdrawals are taken
  from the chain as the proof finalizes it, and any W the finalizing proof could apply must be in
  the set. The switch from `WIN_BEFORE` to `WIN_AFTER` is the first time the destination's exporter
  has *any* implementation, rogue included.
- **Versions.**
  - Only the released L2 versions 1.3.1 and 2.0.0 and the tip bridge and exporter are modeled. Any
    future messenger version must be re-checked against AC2: it must not re-emit `SentMessage` for
    a hash with a timestamp, and it must keep `P ≥ W`.
  - P is fixed at 8 for 2.0.0. A messenger version with a different P is not modeled.
  - The earlier-design L2 messenger (`37b44c48c7`, with export in the messenger) is not modeled.
    Its exports come from 0x..23, which the tip L1 rejects.
- **NUT bundles.**
  - They are modeled as their effect on the three predeploys. The L2CM guard is modeled for the
    messenger and bridge. The exporter isn't in the Lagoon bundle.
  - `nutTip` overwrites a rogue exporter unconditionally. The real L2ContractsManager calls
    `version()` on the current implementation and reverts the whole upgrade if that call fails or
    returns a higher semver. The model is more permissive here, and `rogueEver` stays set in any
    case.
  - Bundle *execution failure modes* other than the semver guard are not modeled (gas, other
    predeploys).
- **`isInterop` gating.** The L2CM installs interop predeploys only when `L1Block.isFeatureEnabled(INTEROP)`
  (`L2ContractsManager.sol:190`). A chain where this is false is a chain where `nutTip` doesn't
  happen, which is one of the explored interleavings.
- **Bounds.** Three chains, three messages, `MAX_TIME` = 16, windows {5, 7, 8, 9, INF} as the
  instances configure them, and the depths in **Results**: 11 for the safe instances, 15 for the
  violations. There is no unbounded proof of the rollout model. The Lean
  model in `../lean` proves the fully-upgraded design for unbounded executions.

## Review log

Reviewers are named R1 (a fresh-context reviewer) and R2 and R3 (independent model-based reviewers).

**v1**: changes made while building it.
- **Non-vacuity.** Every safe instance has witness invariants. The runner fails if Apalache reports
  any of them as holding. There are also scripted `witness*` / `blocked*` / `cex` traces that
  `run.sh test` asserts.
- **AC2 scoping.** It is checked rather than argued: `rolloutSafe` allows a rollback before any
  timestamp.
- **Clocks.** They are folded into the time-reading actions.
- **Split safety property.** `Safety` and `SafetyFull` are separate, so that the deep check is not
  dominated by `ExpiredImpliesNeverRelayable`.

**v1 review** (R1, R2, R3: a correctness and statement-fidelity audit against the cited code at
`e1b3903ab8` and `fa9974a2`). None of them found a critical issue. Each finding and what became of
it:
- **R1 (high): AC4 was too weak.** It required uniqueness only among members, but a non-member
  with a member's chain ID double-spends: Apalache's 9-step counterexample has no join. AC4 is
  restated, row 6d is corrected, and the trace `cexNonMember` is added.
- **R1, R2, R3: the bound was overclaimed.** "Every ordering is safe", "minimal" and "each
  attack's shape is inside the bound" overstated it. The wording now says: no violation in
  executions of ≤ 11 events of the finite model; depth 11 is a heuristic, not a completeness
  threshold; the conditions are individually indispensable, not the weakest.
- **R1, R2: the late L1 INTEROP enable was vacuous.** Only C could enable it, and C is never a
  source. The claim that its order is free is withdrawn; it is argued instead (the gate only adds
  a replayable revert). Checking it needs a model change.
- **R1, R3: the Lagoon guard citation and the side note.**
  - The cited guard is now the locked bundle's own L2ContractsManager (`fa9974a2`).
  - `nutLagoon` ignoring the INTEROP gate is stated as a conservative choice.
  - The side note is confirmed and corrected. The revert can come from the fee vaults or
    L2DevFeatureFlags, even on a single-chain set. The deposit fails, the block stays valid, the
    wrappers and implementation deployments survive, and nothing retries.
- **R1, R2, R3: the hardening needed caveats.** It must ship with the first 2.0.0; it breaks the
  getter and a test; a rollback still needs AC5. Added.
- **R1, R2: clocks and abstractions.**
  - The BC2 trace's resend time is now after the refund (resend at 12, relay at 15).
  - The abstract clocks, the missing `init ≤ exec` and dependency checks, the replay
    over-approximation and the compression in `messengerForge` are now stated.
- **R1: wrong trace claim.** The "short prefix" claim for the last six witnesses was false. Four
  traces are added.
- **R1, R2, R3: `check-nut-locks`.** It does more than compare hashes; the description is
  corrected, and the conclusion stands.
- **R1: rogue exporter overwrite.** The `nutTip` overwrite is now described as more permissive
  than the code.
- **R1: header comment.** The comment's tip reference is fixed.
- **R2, R3: results reproducibility.** Results can't be reproduced from the repo, so the model's
  digest and where the logs are kept are recorded.
- **Confirmed by all three:**
  - BC2's mechanism (1.3.1 resend checks only `sentMessages`; 2.0.0 writes it; `ProxyAdmin.upgrade`
    has no version check; the semver guard is only on the NUT path);
  - BC1's pin and bundle contents;
  - BC3's key-8 finding. A game that needs key 8 can't execute that step on chain, so it is decided
    by the clocks.
  - All eight counterexample mechanisms match real code paths under their relaxations.
