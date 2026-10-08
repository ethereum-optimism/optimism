# Rollout model: activation orderings and misconfigurations for interop expiry

`rollout.qnt` asks which **deployment orderings and misconfigurations** of the exporter design are
safe, and which activation conditions are needed. The design is on `karl/message-expiry-refunds` at
`5992028e08`. The cited code was re-checked at tip `e1b3903ab8`. The changes there are error
renames, an `UndeliveredMessageExported` event, the exporter's move to `0x4200…0030`, and a kona
getter guard; none of them changes this model. The model is a copy of `../quint/expiry.qnt` with
the atomic `upgrade(x)` split into separate events per chain and per layer, which can happen in any
order.

Results in short:
- **Every ordering of the rollout is safe** under five activation conditions, AC1–AC5; the tip's
  code already meets AC5. This is checked for every interleaving of up to 11 events, which is as
  long as the longest counterexample. The orderings include: the L1 before or after the L2; chain
  by chain; source before destination or the reverse; the locked Lagoon bundle; an exporter that
  goes live before its messenger records timestamps, or after; L1, bridge and exporter rollbacks;
  a messenger rollback before any timestamp; and a mid-rollout lockbox join.
- **Each condition is necessary**: dropping any one gives a concrete double spend.
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
| `nutLagoon(x)` | The **locked Lagoon bundle** (`op-core/nuts/bundles/lagoon_nut_bundle.json`, `fork_lock.toml` commit `fa9974a2`): messenger **1.3.1**, bridge **1.0.1**, and no exporter entry, so the exporter is untouched. With `L2CM_DOWNGRADE_GUARD`, the whole `upgradePredeploys` call reverts if the messenger or bridge is already newer (`L2ContractsManagerUtils.upgradeTo:61-67`). |
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
- `relay(m, w)`: valid iff some initiating event `e` has `now[dest] <= e + w`. The window `w` is
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
  L2CrossDomainMessenger produces a withdrawal whose recorded sender is 0x..23.
- `l1Relay(w)`: the withdrawal comes from a lockbox member other than the route. For a `new` L1 it
  needs the route's INTEROP and the exporter as the recorded sender; for an `earlier` L1, 0x..23.
  Withdrawals and deposits are never consumed: failed L1 relays and failed L2 deposits stay
  replayable (`failedMessages`).
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

**Witness invariants in `rolloutSafe`.** All must be violated. Each is also driven by a scripted
trace (or, for the last six, is a short prefix of one):

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
| 6d | joining chain has a member's L2 chain ID | **double spend** | `duplicateChainId` |

## Minimal activation conditions (checked to 11 events)

`rolloutSafe` assumes exactly AC1–AC5 and leaves every other ordering free. Each `cex` instance
drops exactly one of them, and each double-spends.

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
- **AC4 (chain IDs).** L2 chain IDs are unique among the chains that are, or become, lockbox
  members. *Necessary*: `duplicateChainId`.
- **AC5 (L1 implementation).** No L1CrossDomainMessenger that trusts 0x..23 for
  `relayUndeliveredMessage` is ever installed, even briefly: 1.3.1 L2 messengers let anyone make
  0x..23 speak, and such withdrawals never expire. *Necessary*: `l1EarlierDesign`. The tip's code
  meets it (check (c) compares against `Predeploys.UNDELIVERED_MESSAGE_EXPORTER`).

**Not needed** (no condition was required; `rolloutSafe` checks them free):
- the relative order of the L1 upgrade, the L1 INTEROP flag, the messenger, bridge and exporter
  upgrades, and lockbox joins;
- the order across chains;
- the locked Lagoon bundle before the tip bundle;
- rollbacks of the L1 messenger, the bridge and the exporter, and of the messenger before any
  timestamp.

In particular there is **no "L1 accepts facts only after …" condition**. Before a member's
exporter has its standard implementation, no withdrawal from the exporter can exist there, and an honest fact stays true
forever because the destination's clock and AC1 are monotone.

### Which conditions the code enforces

| Condition | Enforced by code | Left to process |
|---|---|---|
| AC1 | op-core `StaticConfigDependencySet.hydrate` rejects overrides > 7d (`static_depset.go:141`); kona `DependencySet` serde rejects > 7d (`genesis/src/interop/depset.rs:39`) and `get_message_expiry_window` ignores a larger override set directly (`:54-56`), and that path covers the embedded registry (`registry/src/lib.rs:52`, serde) and the host-fallback depset (`proof-interop/src/boot.rs:281-284`, serde); `op-interop-filter` rejects > 7d (`filter/config.go:80`); op-node's registry depset (`op-node/superchain/depset.go:17-35`) has no override, so W = 7; `EXPIRY_PERIOD = 8 days` | every node, filter and **absolute prestate** that judges a destination's relays must include the cap (or have no override) **before** that destination's exporter goes live; the destination's withdrawals must be finalized by an interop (super-root) proof that enforces W. A pre-cap build with an override > 8d breaks AC1. |
| AC2 | NUT path: `L2ContractsManagerUtils.upgradeTo` reverts on a semver decrease (`:61-67`), so the Lagoon bundle after the tip reverts | **governance path not enforced**: `ProxyAdmin.upgrade` (`ProxyAdmin.sol:152`) can set 1.3.1 (see BC2) |
| AC3 | genesis proxy without implementation; L2CM only sets the standard implementation | the L2 ProxyAdmin owner can set anything (the named governance assumption); `ETHLockbox.authorizePortal` (`ETHLockbox.sol:124`, `_authorizePortal:220`) checks only the shared ProxyAdmin owner and SuperchainConfig, not the joiner's history (BC4) |
| AC4 | `OPContractsManagerMigrator._validateChainSystemConfigs` (`:295-330`) rejects duplicate L2 chain IDs | `ETHLockbox.authorizePortal` does not check chain IDs (BC4) |
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
  - A resends at A-time 3;
  - B relays at B-time 10, which is valid because 10 ≤ 3 + 7.
  The rollback happens after the refund, so no re-upgrade is needed.
- **NUT path.** It is protected only by the semver guard: without it, the Lagoon bundle activating
  after the tip does the same (`lagoonWithoutGuard`).
- **Fix options.**
  - (a) 2.0.0 stops writing `sentMessages`. `downgradeHardened` checks that this makes **any**
    rollback to 1.3.1 safe: 1.3.1 can then resend only its own messages, which have `sentAt = 0`.
    The cost is that the public `sentMessages(nonce)` getter returns zero for messages sent under
    2.0.0. An alternative with the same effect is to write the hash to a new slot.
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
  - `check-nut-locks` only compares the hash to the lock, so nothing ties the interop fork's bundle
    to the expiry contracts.
- **Consequence if Lagoon ships as locked.**
  - The new L1CrossDomainMessenger (OPCM) has no counterpart.
  - Every message sent while 1.3.1 is live has `sentAt = 0` and can **never** be expired or
    refunded, even after a later upgrade.
  - 1.3.1 keeps `resendMessage` and no target rule.
  - The docs (`message-expiration.mdx`) would overstate the guarantee.
  - Commit `5992028e08` itself defers the state regeneration to "the fork that ships the exporter".
- **Required.** `just nut-snapshot-for lagoon` (the two-PR flow in `op-core/nuts/README.md`) before
  Lagoon is scheduled on any chain that should have expiry.
- **Related (not checked here).** A chain whose genesis already has 2.0.0, and on which Lagoon
  activates later, would have the Lagoon `upgradePredeploys` revert as a whole on the semver guard.
  That skips **all** of Lagoon's predeploy upgrades on that chain. It is safe in this model
  (`NoRefundAfterLagoonReverted`), but worth confirming as an operational issue.

**BC3: the window cap holds only for software that includes it (AC1).**
- **Severity.** Low.
- **Code side.**
  - The cap is enforced in every parser at the tip, including kona's host-fallback depset. The
    fallback `boot.rs:281-284` deserializes through the capped serde.
  - The fallback is also unprovable on chain: `SuperFaultDisputeGame.addLocalData:611` accepts only
    local idents 1–4, and the depset is key 8. So a cluster outside the embedded registry has no
    working on-chain proof anyway, a general misconfiguration and not specific to expiry.
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

At depth 11 `rolloutSafe` covers every interleaving of up to 11 events. That is as long as the
longest counterexample of every unsafe instance, so each attack's shape (with the one dropped
condition restored) is inside the safe check's bound. It also covers the honest refund (8 events)
combined with up to 3 arbitrary further events: rollouts, rollbacks, joins, forged withdrawals and
so on.

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

**File identity.** Every check in the tables ran on this `rollout.qnt`, byte for byte. The one
exception is an earlier full-conjunction run of `rolloutSafe`. It checked all five conjuncts at
once, which is today's `SafetyFull`, completed **depth 9** (in 22 min) and was then stopped to split
the property. It ran on a version that differed only in two ways: it had not yet added the
witness `val`s `NoJoin` … `NoResend`, and it still called today's `SafetyFull` `Safety`.

Scripted traces (`./run.sh test`, Quint 0.33.0, Rust backend): all pass. That is 18 in
`rolloutSafe`, 2 in `rolloutWindowAtP`, 2 in `downgradeHardened`, and 1 counterexample in each of
the 8 unsafe instances.

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
  - The tip bundle overwrites a rogue exporter, which only removes behavior.
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

**v1** (this version): not externally reviewed yet. Changes made while building it:
- **Non-vacuity.** Every safe instance has witness invariants. The runner fails if Apalache reports
  any of them as holding. There are also scripted `witness*` / `blocked*` / `cex` traces that
  `run.sh test` asserts. Witnesses now cover every activation event and attacker move:
  `NoJoin`, `NoLateInteropEnable`, `NoRogueExporterEver`, `NoRogueWithdrawal`,
  `NoMessengerWithdrawal`, `NoResend`.
- **AC2 scoping.** It is checked rather than argued: `rolloutSafe` allows a governance rollback
  while the chain has no timestamped message (`NoRefundAfterMessengerRollback`).
- **Clocks.** They are folded into the time-reading actions; there is no `tick`. This cuts the
  honest refund from 9 to 8 steps and the longest counterexample from 13 to 11.
- **Split safety property.** `Safety` and `SafetyFull` are separate, so that the deep check is not
  dominated by `ExpiredImpliesNeverRelayable`. The first full-conjunction run of `rolloutSafe`
  completed depth 9 before it was stopped (see **Results**).
