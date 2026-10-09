# Lean 4 proof: per-message interop expiry is safe (v2.3, exporter design)

**What this certifies.** This proves the safety of the **exporter design**, as it stands at
`89a3d565ad` (the code citations below are at that commit). In that design:

- `relayUndeliveredMessage` trusts only the `UndeliveredMessageExporter` at
  `Predeploys.UNDELIVERED_MESSAGE_EXPORTER`;
- P_contract is each messenger's stored expiry period, set by `initialize`; every production
  deployment and upgrade sets 8 days;
- W_protocol ≤ 7 days.

The theorem: under the stated hypotheses, ETH is never both delivered on a message's destination
and refunded on its (standard) source. It holds for any number of chains, sources, messages and
lockbox joins, any length of time, and any attacker-chosen relay targets.

**Exporter address.** The model refers to the exporter only as `Sender.exporter`, never by a
hardcoded address. Since `52ff613e14` the constant is
`packages/contracts-bedrock/src/libraries/Predeploys.sol:118`
(`UNDELIVERED_MESSAGE_EXPORTER = 0x4200…0030`; it was `0x4200…002E` at `5992028e08`). Nothing in the
proof depends on which slot it uses. The citations below are at `5992028e08`; at `52ff613e14` only
the exporter address and comments changed. At the contracts tip `448d31ad19` (`c7c51d79e2` plus the messenger's version
string) the further differences are error names, the `UndeliveredMessageExported` event and NatSpec;
none changes a modeled guard or effect.

**Where each design change is, at `89a3d565ad`.** All paths are under the repo root;
`cb/` = `packages/contracts-bedrock/`.

1. **Exporter predeploy:** `cb/src/L2/UndeliveredMessageExporter.sol:24` (the contract), with
   `exportUndeliveredMessage` at `:56-88`. It hashes with `_destination: block.chainid` (`:69`),
   reverts if `successfulMessages(H)` (`:77`), and makes exactly one call:
   `L2CrossDomainMessenger.sendMessage(_sourceMessenger, relayUndeliveredMessage(H, block.timestamp), _minGasLimit)`
   (`:81`). It is registered in `Predeploys.sol:471`.
2. **Sender check (c) moved to the exporter:** `cb/src/L1/L1CrossDomainMessenger.sol:126`
   (`xDomainMessageSender() != Predeploys.UNDELIVERED_MESSAGE_EXPORTER` reverts). Checks (a) and
   (b) are at `:124-125`.
3. **INTEROP gate:** `cb/src/L1/L1CrossDomainMessenger.sol:119`
   (`if (!systemConfig.isFeatureEnabled(Features.INTEROP)) revert`). This is the receiving chain's
   own SystemConfig, which matches the model's `cfg.interop w.fact.toL1`.
   **Pause check** (added after `89a3d565ad`; `L1CrossDomainMessenger.sol:122` at `0a88e080e6`): right after the gate,
   `relayUndeliveredMessage` reverts `L1CrossDomainMessenger_Paused` while the receiving chain is
   paused (`paused()` reads its ETHLockbox, `ETHLockbox.sol:112-113`: a local or global
   SuperchainConfig pause). The model's `s.paused w.fact.toL1 = false` in the `l1Relay` guard.
4. **P_contract, a deployment parameter:** the messenger's stored expiry period, which
   `initialize` sets (bounded to `0 < P ≤ 365 days`), with the strict check
   `if (_undeliveredAt <= sentAt + period) revert` in `expireMessage`. Every upgrade initializes
   the production `Constants.L2_TO_L2_MESSAGE_EXPIRY_PERIOD = 8 days`; only a test network's genesis
   can set another value, and its next upgrade resets it. The model's `contractPeriod : Chain → Nat`
   is that value per chain, fixed for the whole execution (as it is on every production chain), and
   `SafeConfig.window` bounds every destination's window by every source's period. The production
   pin and the 7-day cap (item 5) discharge it: `production_window`. A test network's override, and
   its reset by an upgrade, are checked in the bounded `../rollout/` model (AC6). **The model assumes
   that every live messenger has its initialized period** (rollout AC7). `initialize` rejects 0, but
   `ProxyAdmin.upgrade` (`src/universal/ProxyAdmin.sol:152`) only stores the implementation address
   (`Proxy.upgradeTo`, `src/universal/Proxy.sol:60`, `:104`) and leaves `expiryPeriod` (storage
   slot 5) as it was. A messenger proxy upgraded that way from 1.3.1, which never ran this
   `initialize`, has period 0, and `expireMessage` then accepts any fact dated after the send: a
   double spend (`../rollout/`, `govDirectUpgradeUninitialized`). The L2ContractsManager always
   upgrades it with `upgradeToAndCall` → `initialize` (`L2ContractsManager.sol:415-423`); a direct
   upgrade by the L2 ProxyAdmin owner is the process half of AC7. `expireMessage` also returns
   early when the message is already expired; that is a stuttering step here (`expired` already
   holds), so the model does not change.
5. **W_protocol ≤ 7-day cap:**
   - Go: `op-core/interop/depset/static_depset.go:141-142`, in `hydrate`, rejects an override above `MessageExpiryTimeSecondsInterop = 604800` (`:15`). The validity rule is at `op-core/interop/depset/links.go:73`.
   - kona: `rust/kona/crates/protocol/genesis/src/interop/depset.rs:47-56`: the override is a `MessageExpiryOverride`, whose `TryFrom<u64>` (also used by serde) rejects a value above `MESSAGE_EXPIRY_WINDOW` (`constants.rs:5`, 7 days), so a larger override cannot be built or parsed.

**Defense in depth.** The messenger's unsafe-target rule now also rejects the
L2ToL1MessagePasser: `_isUnsafeTarget` at `cb/src/L2/L2ToL2CrossDomainMessenger.sol:318-320`,
used on send (`:199`) and relay (`:251`). Safety does not depend on it
(`safety_without_targetRule`).

**Older contracts are a different configuration.** The contracts at `37b44c48c7`, before the
landing, are `cfgMessengerTrusted` (`trusted = messenger`). That configuration is **not** covered:
`cex_messengerTrusted` shows it double-spends without v1's activation premise.

The model mirrors `../quint/expiry.qnt`, with the same action and property names where they exist.

- **Dependencies:** core Lean 4.34.1 only.
- **Holes:** no `sorry`, `admit` or `native_decide`.
- **Axioms:** no `axiom` declarations. The build check `#assert_headline` admits Lean's three
  standard axioms (`propext`, `Classical.choice`, `Quot.sound`), so a later proof could pick up
  `Classical.choice` through a tactic without failing the build; the footprints it prints (below)
  contain no `Classical.choice`.
- **Axiom report:** `#print axioms` lists only `propext` and `Quot.sound`, or nothing at all.
- **Bounds:** none. Chains, bodies and hashes are arbitrary types, time is `Nat`, and executions are finite but of any length.

## Build

```
lake build        # under 15 s from clean on a 32-core Linux build host; prints the #print axioms report
```

| File | Contents |
| --- | --- |
| `Expiry/Model.lean` | Types, `Config`, `State`, actions (`guard`, `next`, `Step`), `Reach`, `Init`, `GovInit`, `HashInjective`, `SafeConfig`, property definitions |
| `Expiry/Invariant.lean` | `HistW` (any config with `exporterGovernance`), `Inv` (state invariant), and their preservation proofs |
| `Expiry/Safety.lean` | Main theorems |
| `Expiry/Counterexamples.lean` | Concrete instance: witnesses, safe variants, counterexamples |
| `Expiry/NonVacuity.lean` | One `nonvacuous_<T>` witness per headline theorem `T` |
| `Expiry/Axioms.lean` | `#assert_headline` (axioms + witness check, fails the build) and `#print axioms` |

## The model

**Messages.** A message is the preimage `(d, z, b)`: destination `d`, source `z`, and the rest
`b : Body` (nonce, sender, target, payload). The contracts know chains only by their L2 chain IDs
(`block.chainid`), so the model separates `Chain` from `cfg.chainId : Chain → Nat`. The hash is
`cfg.msgHash d z b = cfg.hash (chainId d) (chainId z) b`. The exporter on chain y hashes with
`chainId y`, and nothing on L1 ties the exporting chain to the hash's destination except that
chain ID.

**Senders and withdrawals.** Every L2→L1 message is a `Withdrawal = (origin, sender, fact)`.
`sender` is the L2 contract that called `L2CrossDomainMessenger.sendMessage`, which L1 exposes as
`xDomainMessageSender()`:

- `exporter`: `Predeploys.UNDELIVERED_MESSAGE_EXPORTER`;
- `messenger`: 0x..23;
- `user a`: any other address.

`fact = (toL1, hash, time)` is `relayUndeliveredMessage(hash, time)` addressed to chain `toL1`'s
L1CrossDomainMessenger (L1CDM).

**The attacker acts through targets.** A relayed message calls `cfg.decode b`:

- `.l2cdm f`: the target is the L2CDM, so 0x..23 becomes the recorded sender of `f`.
- `.other`: any other target.

A user contract reached that way sends withdrawals under its own address (`userWithdrawal`).
Calling the exporter runs only its export. A passer withdrawal is executed by the portal, so on L1
the caller is the portal, which is not an L1CDM.

Every theorem holds for **every** `decode`, so in particular for one that reaches every call.

**Non-standard chains** run arbitrary code:

- `arbitraryCode`: a withdrawal with any sender;
- `arbitraryEvent`: any SentMessage event;
- `arbitraryRefund`: the chain's own bridge pays anything.

Their relays are unconstrained.

**State.**

| Field | Meaning |
| --- | --- |
| `clock y` | Per-chain block timestamps |
| `upgraded y` | The network upgrade has run on chain y |
| `lockbox z y` | y's portal is authorized in z's ETHLockbox |
| `sentAt z h` | `sentMessageTimestamps` on chain z |
| `events z h e` | SentMessage events of chain z, with z's timestamp e |
| `relayed x h` | `successfulMessages` on chain x |
| `withdrawals` | Never removed |
| `deposits` | `expireMessage(hash, time)` deposits to chain `f.toL1`; never removed |
| `expired z h`, `refunded z h` | The contract storage |
| `refunds z h` | Ghost payout counter |
| `paused y` | Chain y is paused: its L1CDM's `relayUndeliveredMessage` reverts. Arbitrary at genesis |

**Actions.** `guard` gives the enabling condition and `next` the effect.

| Action (Quint name) | Guard | Effect |
| --- | --- | --- |
| `tick y t` | `clock y < t` | `clock y := t` (each chain's clock is monotone on its own) |
| `upgrade y` | y standard, not yet upgraded | `upgraded y`: the exporter proxy gets its implementation; the messenger records timestamps and has the target rule |
| `exporterGovernanceUpgrade y f` | `exporterGovernance = false` (excluded by `SafeConfig`) | y's L2 governance replaces its exporter: a withdrawal with sender `exporter` and any fact `f` |
| `join z y` | `govCheck → standard y` | `lockbox z y` |
| `send z d b` | `d ≠ z`; fresh hash (unique nonce); after the upgrade with the target rule, the target is not the L2CDM | event at `clock z`; `sentAt := clock z` only if z is upgraded |
| `resend z d b` | `cfg.resend`, `sentAt ≠ 0`, and `¬expired` if it restarts | event at `clock z`; `sentAt := clock z` if it restarts |
| `resendLegacy z d b` | z not upgraded, and H was sent (the old permissionless `resendMessage`, allowed when `sentMessages[nonce] == H`) | event at `clock z`; no timestamp |
| `relay x z b` (with `.l2cdm f` this is `relayToL2CrossDomainMessenger`) | `¬relayed x H`; if x is standard, there is an event `e` of z for H with `clock x ≤ e + W_x`; if x is standard, upgraded and has the target rule, the target is not the L2CDM | `relayed x H`; plus `(x, messenger, f)` when `decode b = .l2cdm f` |
| `exportUndelivered y z b route` | y standard and upgraded, `¬relayed y (hash y z b)` | withdrawal `(y, cfg.trusted, (route, hash y z b, clock y))` |
| `userWithdrawal y a f` | none | `(y, user a, f)` |
| `arbitraryCode y s f` | y not standard | `(y, s, f)` |
| `arbitraryEvent z h t` | z not standard | event `(z, h, t)` |
| `l1Relay w` | w exists; `unsafeTargetCheck → origin ≠ toL1`; `lockboxCheck → lockbox toL1 origin`, read at relay time; `senderCheck → sender = trusted`; interop gate on `toL1`; `toL1` not paused | deposit `w.fact` to `toL1` |
| `fakeCaller f` | `¬realMessengerCheck ∨ ¬lockboxCheck ∨ ¬sysConfigConsistent` | deposit `f` |
| `l1cdmSelfRelay f` | `¬unsafeTargetCheck` | deposit `f` |
| `expire f` | deposit; `sentAt toL1 hash ≠ 0`; `expiredBy` (`sentAt + P < time`, or `≤` if `expireGe`) | `expired toL1 hash` |
| `refund z d b` | `isBridge b`; `expired z (hash d z b)`; `¬refunded` | `refunded`, `refunds += 1` |
| `arbitraryRefund z h` | z not standard | `refunded z h`, `refunds += 1` |
| `pause y`, `unpause y` | none (any time, any order) | `paused y := true` / `false` |

`fakeCaller` has a three-way guard because the checks only work together. A contract that is not
an L1CDM can return a fake portal whose fake SystemConfig names it. That passes the
real-messenger check, and only the lockbox check (an authorized portal's SystemConfig names the
real L1CDM) stops it. Conversely, a fake caller returning a real authorized portal fails the
real-messenger check.

`refund` rebuilds the hash with source = z (`block.chainid`), so only the true source's refund
matches, and a fact routed to the wrong chain is harmless.

**Pause.** A relay rejected by the pause reverts inside the calling L1CDM's `relayMessage`, which
records the message in `failedMessages` (`CrossDomainMessenger.sol:308`); anyone can replay it
after unpause (`:260`), with the original sender and payload, so the fact keeps its time. In the
model withdrawals are never removed, so a paused chain only disables `l1Relay` until it is unpaused.
Per-chain flags with free `pause`/`unpause` over-approximate the real pause, which is shared by every
chain on one ETHLockbox (and global through the SuperchainConfig). `fakeCaller` and
`l1cdmSelfRelay` (which exist only when a check is dropped) are left ungated, which can only add
executions. A pause of the withdrawal's origin chain (the portal's finalization and the caller's
`relayMessage` also revert while it is paused) is not modeled; it too only delays.

## Hypotheses

Every hypothesis is an explicit argument or structure field.

| Hypothesis | Lean | Real-world fact |
| --- | --- | --- |
| Exporter is trusted | `SafeConfig.trusted` | `relayUndeliveredMessage` checks `xDomainMessageSender() == Predeploys.UNDELIVERED_MESSAGE_EXPORTER` (`L1CrossDomainMessenger.sol:114`). |
| Member governance keeps the standard exporter | `SafeConfig.exporterGovernance` | Each cluster chain's L2 governance (its L2 ProxyAdmin owner) can upgrade its own exporter, and could thereby forge facts for any destination. The design trusts it not to. Lockbox authorization compares the portals' **L1** ProxyAdmin owners, and the exporter is upgraded by the **L2** ProxyAdmin owner, a separate role; the trust is comparable because a member's L2 governance can already make arbitrary withdrawals from the shared lockbox by changing its own L2 state. Without it: `cex_exporterGovernanceUpgrade`. |
| Real-messenger check | `SafeConfig.realMessengerCheck` | `IL1CDM(msg.sender).portal().systemConfig().l1CrossDomainMessenger() == msg.sender`. |
| Lockbox check | `SafeConfig.lockboxCheck` | `portal.ethLockbox().authorizedPortals(callerPortal)`, read at L1 relay time. |
| Sender check | `SafeConfig.senderCheck` | The `xDomainMessageSender()` comparison. |
| L1CDM self-target rule | `SafeConfig.unsafeTargetCheck` | `L1CrossDomainMessenger._isUnsafeTarget` blocks relays to itself and its portal. So an L1CDM is the L1 sender of an L1→L2 message to the L2 messenger only through `relayUndeliveredMessage`. This is encoded by deposits arising only from `l1Relay` (and from `fakeCaller` / `l1cdmSelfRelay` when a check is dropped). |
| SystemConfig consistency (governance) | `SafeConfig.sysConfigConsistent` | For every portal authorized in a lockbox, `systemConfig.l1CrossDomainMessenger()` is that chain's real L1CDM. |
| Governance join rule | `SafeConfig.govCheck`, plus `hg : GovInit cfg s₀` | Only **standard** chains are ever authorized in a lockbox. A standard chain ran the standard predeploys for its whole history, and its relays obey the protocol window. |
| Windows | `SafeConfig.window : ∀ z d, W_d + (if expireGe then 1 else 0) ≤ P_z` | W_d ≤ 7 days (config cap) ≤ P_z = 8 days (the production period) with the strict check; `production_window` derives it from those two pins. With `≥` it requires W_d < P_z. |
| No non-restarting resend | `SafeConfig.resend` | `resendMessage` is removed after the upgrade. A restarting resend would also be safe. Pre-upgrade resends are always modeled (`resendLegacy`). |
| Idealized hash | `hinj : HashInjective cfg.hash` | See "The hash" below. |
| Unique chain IDs (governance) | `hid : ChainIdUnique cfg := ∀ c c', standard c → chainId c = chainId c' → c = c'` | Every standard chain has a chain ID that no other modeled chain uses. Standard chains are those that are or can become lockbox members, and the protected sources; the modeled chains are every chain whose withdrawals or relays the protocol can see. This is stronger than any on-chain check: OPCM's migrator rejects duplicates only among the chains it migrates, so uniqueness against non-members (and in members' dependency sets) is a configuration obligation (rollout AC4). Without this, a member sharing B's ID exports "not relayed" for a hash delivered on B (`cex_duplicateChainId`), with no hash collision. |
| Genesis | `h0 : Init s₀` | At every chain's genesis nothing has been sent, relayed, withdrawn or deposited, and no chain is upgraded. Clocks and initial lockbox memberships are arbitrary. |

The messenger's unsafe-target rule (`cfg.targetRule`) is **not** a hypothesis. The design keeps it as
defense in depth, and safety does not depend on it:

- `safety_without_targetRule` proves the full `safety` conjunction for configurations with
  `targetRule = false`.
- The rule's own property is `messengerSilentAfterUpgrade`: with the rule, an upgraded standard
  chain's 0x..23 never initiates a withdrawal.
- `messengerSpeaks_without_targetRule` shows that property fails without the rule.
- The L2ToL1MessagePasser path (0x..23 → passer) is not modeled as a sender here. Since
  `5992028e08` the messenger rejects that target too (`_isUnsafeTarget`,
  `L2ToL2CrossDomainMessenger.sol:318-320`). It is checked on bytecode by Halmos
  (`check_UnsafeTargetRule_*_passer`, `check_OnlyExportReachesL1_relay_passer`) and Kontrol
  (`prove_sendMessage_rejectsPasser`, `prove_relayMessage_rejectsPasser`).

**Deployment assumption: historical inertness.** The exporter address is not code-free before the upgrade.
`scripts/L2Genesis.s.sol:226-240` (`setPredeployProxies`) etches the `Proxy` at every proxied
predeploy slot (admin = ProxyAdmin), and sets an implementation only for supported predeploys.
With no implementation, every call reverts (`src/universal/Proxy.sol:124-126`, `_doProxyCall`:
`require(impl != address(0))`). So the exporter is silent before its upgrade because its proxy has
no implementation.

The residual assumption is that **no ProxyAdmin action set an implementation at the exporter address before the
network upgrade, on any chain that is or becomes a lockbox member**. In the model, `Init` says no
chain is upgraded at genesis, and for a standard chain `upgrade` is the only action that gives the
exporter an implementation, as long as `exporterGovernance` holds (`exporterGovernanceUpgrade` is the
member-governance action that `SafeConfig` excludes).

`exporterSilentBeforeUpgrade` and `joinNeedsNoHistoryCheck` derive silence **within this action
model**. They do not establish deployment history; historical inertness, together with the
governance join rule, is what covers that.

**The hash.** `HashInjective` is a global statement about the L2-to-L2 message hash on
(destination chain ID, source chain ID, rest): `hash d z b = hash d' z' b' → d = d' ∧ z = z' ∧ b = b'`.
Chains are recovered from IDs only through `ChainIdUnique`.
keccak256 with 256-bit output cannot satisfy this on unbounded inputs. It is an idealization (an
ideal, collision-free hash): the proof relies on the absence of collisions among the message
preimages that occur in an execution, but the hypothesis as stated is global.

The hypothesis says nothing about L1 replay envelopes (versioned withdrawal and message hashes).
Their integrity is a separate external obligation (see the table below). `cex_hashCollision`
shows the hypothesis is needed.

## Theorem statements (`Expiry/Safety.lean`)

The safety theorems take
`(hc : SafeConfig cfg) (hinj : HashInjective cfg.hash) (hid : ChainIdUnique cfg) (h0 : Init s₀) (hg : GovInit cfg s₀) (hr : Reach cfg s₀ s)`.

```lean
def Config.msgHash cfg d z b := cfg.hash (cfg.chainId d) (cfg.chainId z) b
-- In the statements below, `cfg.hash d z b` is written for `cfg.msgHash d z b`.
def expiredBy cfg z sent t := if cfg.expireGe then sent + cfg.contractPeriod z ≤ t
                              else sent + cfg.contractPeriod z < t
def NoDoubleSpend cfg s := ∀ d z b, cfg.standard z →
  ¬ (s.relayed d (cfg.hash d z b) ∧ s.refunded z (cfg.hash d z b))
def RefundImpliesExpired cfg s := ∀ z h, cfg.standard z → s.refunded z h → s.expired z h
def AtMostOneRefund cfg s := ∀ z h, cfg.standard z → s.refunds z h ≤ 1
def ExportedBy cfg s₀ s y f := ∃ z b s₁ s₂, Reach cfg s₀ s₁ ∧
  Step cfg (.exportUndelivered y z b f.toL1) s₁ s₂ ∧ Reach cfg s₂ s ∧
  f.hash = cfg.hash y z b ∧ f.time = s₁.clock y
def NoForgedFact cfg s₀ s := ∀ f, s.deposits f → ∃ y, cfg.standard y ∧ ExportedBy cfg s₀ s y f
def withinWindow cfg s x z h t := ∃ e, s.events z h e ∧ t ≤ e + cfg.protocolWindow x

-- Silence derived within the model. Any configuration with exporterGovernance; needs only
-- (h0 : Init s₀) (hgov) (hr : Reach cfg s₀ s).
-- Informative when cfg.trusted = .exporter; otherwise exportUndelivered records the other sender
-- and the statement is vacuous.
theorem exporterSilentBeforeUpgrade (hgov : cfg.exporterGovernance = true) (w) (hw : s.withdrawals w) (hsnd : w.sender = .exporter)
    (hstd : cfg.standard w.origin) :
    ∃ z b s₁ s₂, Reach cfg s₀ s₁ ∧ Step cfg (.exportUndelivered w.origin z b w.fact.toL1) s₁ s₂ ∧
      Reach cfg s₂ s ∧ s₁.upgraded w.origin = true ∧ ¬ s₁.relayed w.origin (cfg.hash w.origin z b) ∧
      w.fact.hash = cfg.hash w.origin z b ∧ w.fact.time = s₁.clock w.origin

-- Joins need no history re-check (same scope as above).
theorem joinNeedsNoHistoryCheck (h0 : Init s₀) (hgov : cfg.exporterGovernance = true) (hr : Reach cfg s₀ s)
    (hj : Step cfg (.join z y) s s') (hy : cfg.standard y) (w) (hw : s'.withdrawals w)
    (ho : w.origin = y) (hsnd : w.sender = .exporter) :
    ∃ z' b s₁ s₂, Reach cfg s₀ s₁ ∧ Step cfg (.exportUndelivered y z' b w.fact.toL1) s₁ s₂ ∧
      Reach cfg s₂ s' ∧ s₁.upgraded y = true ∧ w.fact.hash = cfg.hash y z' b

theorem noDoubleSpend         ... : NoDoubleSpend cfg s
theorem refundImpliesExpired  ... : RefundImpliesExpired cfg s
theorem atMostOneRefund       ... : AtMostOneRefund cfg s
theorem noForgedFact          ... : NoForgedFact cfg s₀ s

theorem expiredImpliesNeverRelayable ... (d z b) (hz : cfg.standard z)
    (he : s.expired z (cfg.hash d z b)) :
    ¬ s.relayed d (cfg.hash d z b) ∧
    (∀ t, s.clock d ≤ t → ¬ withinWindow cfg s d z (cfg.hash d z b) t) ∧
    (∀ s', Reach cfg s s' →
      s'.expired z (cfg.hash d z b) ∧ ¬ s'.relayed d (cfg.hash d z b) ∧
      ∀ t, s'.clock d ≤ t → ¬ withinWindow cfg s' d z (cfg.hash d z b) t)

theorem expired_no_relay_step ... (d z b) (hz) (he) (s' s'') (hr' : Reach cfg s s')
    (hstep : Step cfg (.relay d z b) s' s'') : False

theorem onlyDestinationCanExport ... (d z b) (hz : cfg.standard z)
    (he : s.expired z (cfg.hash d z b)) :
    ∃ f, s.deposits f ∧ f.toL1 = z ∧ f.hash = cfg.hash d z b ∧
      expiredBy cfg (s.sentAt z (cfg.hash d z b)) f.time ∧
      ∃ s₁ s₂, Reach cfg s₀ s₁ ∧ Step cfg (.exportUndelivered d z b z) s₁ s₂ ∧ Reach cfg s₂ s ∧
        s₁.upgraded d = true ∧ s₁.clock d = f.time ∧ ¬ s₁.relayed d (cfg.hash d z b)

-- Same conclusion as `safety`, for configurations without the messenger's target rule.
theorem safety_without_targetRule (hc : SafeConfig cfg) (_htr : cfg.targetRule = false) ... :
    <the safety conjunction below>

-- Defense in depth (MessengerSilentAfterUpgrade, matches the Quint invariant).
theorem messengerSilentAfterUpgrade (htr : cfg.targetRule = true)
    (htrust : cfg.trusted ≠ .messenger) (hs : Step cfg a s s') (w) (hnew : s'.withdrawals w)
    (hold : ¬ s.withdrawals w) (hsnd : w.sender = .messenger) (hstd : cfg.standard w.origin) :
    s.upgraded w.origin = false

-- A pause only delays a fact (any configuration; needs only `Reach`).
theorem pauseOnlyDelays (hr : Reach cfg s s') (w) (hg : guard cfg (.l1Relay w) s)
    (hu : s'.paused w.fact.toL1 = false) : guard cfg (.l1Relay w) s'

theorem safety ... : NoDoubleSpend cfg s ∧ RefundImpliesExpired cfg s ∧ AtMostOneRefund cfg s ∧
    NoForgedFact cfg s₀ s ∧
    (∀ d z b, cfg.standard z → s.expired z (cfg.hash d z b) →
      ¬ s.relayed d (cfg.hash d z b) ∧
      ∀ t, s.clock d ≤ t → ¬ withinWindow cfg s d z (cfg.hash d z b) t)
```

All the refund and double-spend properties are scoped to standard chains `z`. A non-standard chain
can fake SentMessage events for its own messages and its bridge can pay anything
(`arbitraryRefund`). Under `GovInit` and `govCheck`, every lockbox member is standard.

### Proof structure

**`HistW`** (any config with `exporterGovernance = true`, from `Init`): a withdrawal whose sender is the exporter and whose origin
is standard was created by an `exportUndelivered` step. The other withdrawal-creating steps record
a different sender:

- `relay` → messenger;
- `userWithdrawal` → user;
- `arbitraryCode` → only on non-standard chains.

**`Inv`** (safe config). No part of it reads `paused`, so `pause` and `unpause` preserve it
trivially, and the invariant is the same as before pauses were modeled:

- `sentAt z h ≤ clock z`, and `sentAt ≠ 0` implies the chain is upgraded and has an event.
- On standard sources, events are at or before `sentAt`, or `sentAt = 0`. Legacy resends only happen while `sentAt = 0`, and that stays 0 forever.
- On a standard chain, `relayed x (hash x z b)` implies an event of z.
- Lockbox members are standard.
- Every exporter withdrawal from a standard chain, and every deposit, is `Honest`: it has destination y, it is not from y's future, and if `expiredBy` holds for `sentAt` of a standard source, the message is not relayed on y.
- Expired ⇒ such a deposit.
- On standard chains: refunded ⇒ expired, and refunds ≤ 1.

The key cases:

- **`relay`:** relaying at y-time `≥ f.time` needs an event `e ≤ sentAt`, and
  `f.time ≤ clock y ≤ e + W_y ≤ sentAt + W_y < f.time` (by `expiredBy` and `window`) is a
  contradiction.
- **`send`:** the hash is fresh, so nothing has relayed it.
- **Restarting `resend`:** `sentAt` only grows (source-local monotonicity).

Only per-chain clock monotonicity is used. No cross-chain clock comparability is assumed.

## Witnesses and counterexamples (`Expiry/Counterexamples.lean`)

**The concrete instance.**

- Chains are `Nat`: 0 = A, 1 = B, 2 = C (standard) and 3 = D (arbitrary code).
- At genesis A and B are authorized in every lockbox.
- `hash = id` (injective), W = 7 and P = 8, in days.
- Body 9 decodes to an L2CDM call carrying the forged fact "(B, A, 5) undelivered at 100".

| Theorem | What it shows |
| --- | --- |
| `refund_reachable` | All hypotheses hold, and a refund is reachable. |
| `relay_reachable_at_edge` | A relay is reachable from genesis at exec − init = W. |
| `lateJoin_multiSource_reachable` | C joins after genesis (no history check) and is refunded. A message from source B is routed back to B and refunded there. |
| `forgery_reachable_but_harmless` | 0x..23 and a user contract both send the forged fact (reachable), and `NoDoubleSpend` holds in every continuation. |
| `legacyResend_reachable` | A pre-upgrade send plus a legacy resend 29 days later is reachable. The message has `sentAt = 0`, so it can never expire. |
| `messengerSpeaks_without_targetRule` | With `targetRule = false` (a `SafeConfig`), a reachable relay on upgraded B makes 0x..23 initiate a withdrawal. The defense-in-depth property fails, but safety holds. |
| `safe_variants` | `SafeConfig` holds with: no target rule; P = W = 7; per-destination windows; a restarting resend; the single-field `≥` mutation with P = 8, W = 7 (still safe because of the margin). |

Each `cex_*` except `cex_hashCollision` and `cex_duplicateChainId` proves `Cex cfg trace`, which is:

- genesis, `GovInit` and injectivity hold;
- `¬ SafeConfig cfg`;
- the trace is valid and reachable;
- the final state violates `NoDoubleSpend`.

Each config is `base` with **one** field changed, with two exceptions:

- `cex_nonStrict` changes two fields (`expireGe` and `contractPeriod := fun _ => 7`), because `≥` alone with P = 8 is safe.
- `cex_hashCollision` keeps every `SafeConfig` field and drops injectivity instead.

| Theorem | Dropped assumption | Execution |
| --- | --- | --- |
| `cex_messengerTrusted` | Exporter trusted. This is the **contracts at 37b44c48c7**. | Before B's upgrade, B relays body 9 to the L2CDM, so 0x..23 sends the forged fact. After the upgrades A sends, B relays, the pre-staged fact goes through l1Relay, then expire and refund. |
| `cex_nonstandardJoin` | Governance join rule | D signs the forged fact as the "exporter" and then joins A's lockbox. Its old withdrawal becomes trusted. |
| `cex_periodBelowWindow` | P ≥ W, on every source (here only A's messenger is deployed with P = 6 < W = 7; the others with 8) | B exports at its time 8 > 1 + 6 and relays at 8 ≤ 1 + 7. One mis-deployed source suffices. |
| `cex_nonStrict` | Strict `>` at P = W = 7 | The same edge execution. |
| `cex_resendNoRestart` | No non-restarting resend | Resend at A-time 10; export and expire against sentAt = 1; relay using the event at 10. |
| `cex_noRealMessengerCheck` | Real-messenger check | `fakeCaller`: a contract returning a real authorized portal. |
| `cex_noLockboxCheck` | Lockbox check | D's withdrawal, relayed by D's L1CDM. |
| `cex_noLockboxCheck_fakePortal` | Lockbox check | `fakeCaller`: a fake portal whose fake SystemConfig names the caller passes the real-messenger check. |
| `cex_sysConfigInconsistent` | SystemConfig consistency | `fakeCaller` passes both identity checks. |
| `cex_noUnsafeTargetCheck` | L1CDM self-target rule | `l1cdmSelfRelay`: A's L1CDM sends `expireMessage` as itself. |
| `cex_noSenderCheck` | Sender check | A user contract on B sends the forged fact. |
| `cex_hashCollision` | Injectivity (`SafeConfig` holds) | C honestly exports a colliding hash after B relayed. |
| `cex_exporterGovernanceUpgrade` | Member governance keeps the standard exporter | After B's upgrade, B's L2 ProxyAdmin owner replaces B's exporter with arbitrary code, which sends the forged fact as the exporter. B is a lockbox member, so A accepts it. |
| `cex_duplicateChainId` | Unique chain IDs (`SafeConfig`, `HashInjective` and `GovInit` hold) | C is standard but has B's chain ID. B relays A's message; C joins A's lockbox and exports the same hash; expire and refund. |

## Non-vacuity (`Expiry/NonVacuity.lean`, checked by `lake build`)

For every headline theorem `T` of `Safety.lean`, `nonvacuous_T` takes the concrete instance above,
shows that **all** of `T`'s hypotheses hold **jointly** on one explicit execution, and applies `T`
there, so the conclusion is instantiated where its antecedents actually occur. `Axioms.lean` runs
`#assert_headline T` on the same list of names it reports: the build fails unless `T` uses only
`propext`, `Classical.choice`, `Quot.sound`; `nonvacuous_T` exists and uses only those axioms;
`T` occurs in its proof term; and its statement is closed (no universe parameters, no leading
binder, no mention of `T`, which rules out witnesses such as `@T = @T` or one with an extra
undischarged premise). These are syntactic guards. That each witness really establishes all of
`T`'s hypotheses on one execution and applies `T` there was checked by review (R1, R2 and R3 in
the v2.2–v2.3 round each confirmed it for every witness). All witnesses are kernel-checked
(`simp`/`decide`; no `native_decide`); their footprint is `[propext, Quot.sound]`.

| `T` | Witness execution | Instantiated conclusion |
| --- | --- | --- |
| `exporterSilentBeforeUpgrade` | refund execution `NV.sR` (A sends to B, B exports, L1 relay, expire, refund); exporter withdrawal from B | the export step, with B upgraded and the message unrelayed |
| `joinNeedsNoHistoryCheck` | C upgraded, exports, then `join 0 2` (C joins A's lockbox) | C's honest export before the join |
| `noDoubleSpend` | `NV.sR`: the refund happened; a relay of the same message is reachable in another execution (`relay_reachable_at_edge`) | no relay in the refund state |
| `refundImpliesExpired`, `atMostOneRefund`, `noForgedFact` | `NV.sR` | expired; refunds = 1 ≤ 1; the deposit was exported by a standard chain |
| `expiredImpliesNeverRelayable`, `onlyDestinationCanExport` | `NV.sR` (message expired) | not relayed, outside the window from now on; B's export step |
| `expired_no_relay_step` | concludes `False`, so its hypotheses are jointly unsatisfiable **by design**: all but the relay step hold at `NV.sR`, and the relay step alone is enabled in another reachable state | no relay step is enabled from `NV.sR` |
| `safety` | `NV.sR` | the first four conjuncts at the protected message (the fifth is `expiredImpliesNeverRelayable`'s conclusion, instantiated by its own witness) |
| `safety_without_targetRule` | refund execution of `cfgNoTargetRule` (so `targetRule = false` holds too) | no relay in that refund state |
| `messengerSilentAfterUpgrade` | B (not upgraded) relays C's body-9 message: a new withdrawal with sender 0x..23 | B was not upgraded |
| `pauseOnlyDelays` | B exports the fact for A's message; A is paused (the L1 relay of B's withdrawal is then disabled) and unpaused | the relay is enabled again; continuing, the same fact expires and refunds A's message, and `noDoubleSpend` holds there |

Hypotheses that quantify over all states or values (`SafeConfig.window`, `HashInjective`,
`ChainIdUnique`, `Init`, `GovInit`) are all satisfied by the instance; none is unsatisfiable.

## Axiom report (from `lake build`)

`#assert_headline` prints each headline theorem's axioms and its witness's axioms (every witness:
`[propext, Quot.sound]`); the counterexamples use `#print axioms`. The footprints are:

```
'Expiry.exporterSilentBeforeUpgrade' does not depend on any axioms
'Expiry.joinNeedsNoHistoryCheck' does not depend on any axioms
'Expiry.noDoubleSpend' depends on axioms: [propext, Quot.sound]
'Expiry.refundImpliesExpired' depends on axioms: [propext, Quot.sound]
'Expiry.atMostOneRefund' depends on axioms: [propext, Quot.sound]
'Expiry.noForgedFact' depends on axioms: [propext, Quot.sound]
'Expiry.expiredImpliesNeverRelayable' depends on axioms: [propext, Quot.sound]
'Expiry.expired_no_relay_step' depends on axioms: [propext, Quot.sound]
'Expiry.onlyDestinationCanExport' depends on axioms: [propext, Quot.sound]
'Expiry.safety' depends on axioms: [propext, Quot.sound]
'Expiry.safety_without_targetRule' depends on axioms: [propext, Quot.sound]
'Expiry.messengerSilentAfterUpgrade' does not depend on any axioms
'Expiry.pauseOnlyDelays' does not depend on any axioms
'Expiry.Examples.safe_variants' does not depend on any axioms
every other witness and cex_* theorem: [propext, Quot.sound]
```

## Where W and P are enforced

See "Where each design change is" at the top. In short, at `5992028e08`:

- **P:** `L2ToL2CrossDomainMessenger.sol:75` and `:273`.
- **W cap:** Go `static_depset.go:141-142`, kona `depset.rs:44-45` (and, since `d36e37862b`, the getter's fallback).
- **W validity rule:** Go `links.go:73`. The extra rule `init ≤ exec` is at `links.go:70` and is omitted, which only makes the model more permissive.

## Discharged outside Lean

The proof names below are from the sibling directories, which were re-targeted to the exporter
design (each sibling README names the commit it checked: `c7c51d79e2`, or `448d31ad19` for the
layers pinned to the messenger's bytecode). Only proofs that closed are cited; the Kontrol proofs
that did not close are listed in `../kontrol/README.md` and in the top-level README's open
obligations.

| Obligation | Where |
| --- | --- |
| **L1 identity and reverse binding.** `relayUndeliveredMessage` succeeds only if the caller's portal's SystemConfig names the caller, the portal is authorized in this chain's lockbox, and `xDomainMessageSender()` is the trusted sender. It then deposits exactly `expireMessage(H, t)`. | `halmos/L1CDMExpiryHalmos.t.sol` (`check_relayUndelivered_iff_and_deposit`), `kontrol/solc0815/L1CrossDomainMessengerExpiry.k.sol` (`prove_relayUndeliveredMessage_spec`: every getter answer symbolic, pointer getters fixed stand-ins; the fully symbolic pointer variant `symbolicPortalChain` did not close) |
| **SystemConfig ↔ L1CDM consistency** for authorized portals. | Governance (lockbox authorization, SystemConfig ownership); the `sysConfigConsistent` field |
| **`xDomainMessageSender` semantics.** On L1 it is the `msg.sender` that called the L2CDM for that withdrawal. On L2, the L2CDM's `xDomainMessageSender()` equals `otherMessenger` only for deposits sent by this chain's L1CDM. | Standard CrossDomainMessenger code; Halmos `check_L1_relayGate_and_delivery`, `check_L2_relayGate_and_delivery`, `check_L1_sendMessage_senderFieldIsCaller`, `check_L1_xDomainMessageSender_revertsOutsideRelay` (with the portal-delivery and envelope assumptions in `../halmos/README.md`). `prove_expireMessage_spec` and `check_expire_iff` show only that `expireMessage` checks the getter answers it is given. |
| **The L1CDM is an L1→L2 sender to the L2 messenger only through `relayUndeliveredMessage`** (`_isUnsafeTarget`). | Standard code; encoded by deposits arising only from `l1Relay` / `fakeCaller` / `l1cdmSelfRelay`, the last two only when a check is dropped |
| **Replay envelopes.** Failed L1 relays and failed L2 deposits replay only the original message (versioned hash). This is not covered by `HashInjective`. | Standard code; withdrawals and deposits are conservatively never removed |
| **Exporter code.** H uses destination = `block.chainid`; it requires `!successfulMessages(H)`; it calls only `L2CDM.sendMessage(sourceMessenger, relayUndeliveredMessage(H, block.timestamp), gas)`. | `check_export_binding`, `check_exporter_anyCalldata_onlyExportPayload` (Halmos: finite message lengths, canonical encodings); `prove_exporter_onlyCallsL2CDMWithFixedPayload` (Kontrol: 600-byte messages); `../evm-lean-exporter` (symbolic lengths, calldata < 2^63, chain ID 1 only). Together these do not cover arbitrary lengths at arbitrary chain IDs; the general case rests on code inspection. |
| **Messenger code.** Non-static external calls happen only in `relayMessage`; `successfulMessages` is set before the relayed target call; the target rules hold; `sentAt = block.timestamp`. | `prove_relayMessage_*` (except `selfTarget_neverCallsL2CDMOrPasser`, which did not close; Halmos excludes that target too), `prove_sendMessage_*`, `check_UnsafeTargetRule_*`, `check_OnlyExportReachesL1_*` |
| **Expiry check.** Strict boundary, `sentAt ≠ 0`. | `check_expire_boundary`, `check_expire_iff_unbounded` |
| **P ≥ W cap.** | `check_contractWindowCoversProtocolCap`, `prove_expiryPeriod_atLeastProtocolWindow` |
| **Refund preimage binding.** Source = `block.chainid`, sender = target = bridge, `relayETH(from, to, amount)`, pays at most once. The model keeps the source binding and abstracts the rest as `isBridge`. | `prove_refundETH_preimageBinding` with `prove_refundETH_alreadyRefundedReverts` (single use; the two-call `prove_refundETH_singleUse` did not close), `check_refund_iff_effects_singleUse` |
| **Nonce freshness.** | The `send` freshness guard, with `HashInjective` |
| **Historical inertness of the exporter address.** | Deployment assumption (above) |
| **Member L2 governance keeps the standard exporter.** | Governance assumption, the `exporterGovernance` field (comparable to the shared ETHLockbox's trust; see Hypotheses) |

## Named assumptions not modeled as transitions

- **Every live messenger has its initialized period (rollout AC7).** `contractPeriod z` is the
  period `initialize` stored, fixed for the whole execution. A messenger proxy upgraded to the expiry
  messenger by `ProxyAdmin.upgrade` alone, from a version that never ran `initialize` (1.3.1), has
  period 0 and double-spends; see item 4 at the top and `../rollout/` (`govDirectUpgradeUninitialized`).
  The L2ContractsManager enforces it on its own path; a direct upgrade by governance is process.
- **W activation and time-varying W.** The protocol rule `exec − init ≤ W_d` is enforced on a destination before that destination's exporter goes live. W_d never later rises above P. The model has fixed W_d from genesis; activation of the W rule and changes to W are not modeled.
- **Preimage hardness of predeploy addresses.** No EOA, and no aliased L1 address (`AddressAliasHelper`), equals `Predeploys.UNDELIVERED_MESSAGE_EXPORTER` or 0x..23. This is why `userWithdrawal` can only record `user a` and never a predeploy as sender.
- **Pre-Bedrock legacy withdrawals.** Legacy (pre-Bedrock) L2→L1 messages are irrelevant: none carries `relayUndeliveredMessage` from the exporter. The model's genesis is the Bedrock-era history.
- **Historical inertness of the exporter address** and the **governance assumptions** (standard-only joins, unique chain IDs, SystemConfig consistency, member governance keeps the standard exporter): see "Hypotheses" above.

## Modeling choices and what is NOT modeled

- **Clocks.** Each chain's clock is monotone; timestamps are `Nat`.
  - Overflow: Solidity 0.8 `sentAt + P` is checked arithmetic and reverts on overflow, which only prevents expiry (the safe direction).
  - The rule `initTimestamp ≤ execTimestamp` (`links.go:70`) is omitted, which only makes the model more permissive.
- **Finality.** `withdrawals` and the initiating events are those of the canonical histories. L1 reorgs beyond finality are out of scope.
- **Lockbox and configuration changes.** Joins and pauses are modeled. Leaving a lockbox, arbitrary later upgrades (in particular a messenger live without its initialized period, rollout AC7), and changes to the windows or gate are not modeled.
- **Interop gate.** It only restricts `l1Relay`. Safety does not depend on it.
- **EVM and economics.** Gas, value, reentrancy, call failures, balances and `ETHLiquidity` are left to the other tools.

## Review log

| Round | Reviewer | Finding | Disposition |
| --- | --- | --- | --- |
| v1 | R2, R3 | Attacker completeness assumed: forging disabled by `SafeConfig` flags | v2: target-level relay (`decode`), senders recorded per call, `userWithdrawal`/`arbitraryCode`; "only the exporter speaks" derived |
| v1 | R2, R3 | Activation premise assumed, not enforced | v2: genesis-based model with `upgrade`; `exporterSilentBeforeUpgrade` derived; residual is the governance plus historical-inertness assumption (v2.1) |
| v1 | R2, R3 | Global injectivity is unsatisfiable for keccak | Reworded as an idealization (v2; tightened in v2.1) |
| v1 | R3 | P < W counterexample used W = 9 > cap | v2: P = 6, W = 7 |
| v1 | R3, R2 | `NoForgedFact` not a named conjunct | v2: `noForgedFact`, a conjunct of `safety` |
| v1 | R3 | Refund binding, per-chain clocks, overflow | v2: source-bound `refund`, per-chain clocks; overflow noted |
| v1 | R2, R3 | README 7/8-day drift; no interop gate | v2: two parameters W_d and P; gate modeled; code citations |
| v1 | R3 | `externalRawTrust` models a non-existent contract | v2: dropped |
| v1 | R1 | Cite W cap and P code | v2: "Where W and P are enforced" |
| v1 | R1 | `>=` boundary counterexample | v2: `cex_nonStrict` |
| v1 | R1 | Per-destination windows | v2: `protocolWindow : Chain → Nat` |
| v1 | R1 | Witness that a relay is reachable | v2: `relay_reachable_at_edge` |
| v1 | R1 | SystemConfig ↔ L1CDM consistency | v2: README; v2.1: `sysConfigConsistent` field and counterexample |
| v1 | R1 | Joins need no history re-check | v2: `joinNeedsNoHistoryCheck`; governance counterexample |
| v2 | R1 M1, R3 M1 | "0x..2E never had code" is false: the genesis Proxy is at every slot | v2.1: wording fixed; silence comes from the empty proxy implementation; historical-inertness assumption stated; theorems scoped to "within the model" |
| v2 | R1 M2 | `fakeCaller` mis-attributed: a fake portal passes the real-messenger check | v2.1: guard `¬real ∨ ¬lockbox ∨ ¬sysConfig`; `cex_noLockboxCheck_fakePortal`, `cex_sysConfigInconsistent` |
| v2 | R3 M2 | RefundImpliesExpired / AtMostOneRefund claimed for all chains | v2.1: scoped to standard chains; `arbitraryRefund` for non-standard bridges |
| v2 | R2 M | Pre-upgrade resends not representable | v2.1: `resendLegacy` modeled; `legacyResend_reachable` |
| v2 | R2 L | `cex_nonStrict` changes two fields | v2.1: wording fixed; single-field `≥` (P = 8) proved safe in `safe_variants` (window hypothesis generalized) |
| v2 | R1 L1 | README overstated `HashInjective` | v2.1: README matches the formal statement; envelope integrity listed as a separate obligation |
| v2 | R1 L2 | `_isUnsafeTarget` labeling | v2.1: the safety-relevant fact is stated; `l1cdmSelfRelay` plus `cex_noUnsafeTargetCheck` |
| v2 | R1 H1, R2, R3 | Make clear what is certified | v2.1: first paragraph; 37b44c48c7 = `cfgMessengerTrusted`; list of code changes that must land |
| v2.1 | design decision | Keep the target rule as defense in depth; safety must not depend on it | v2.1: `safety_without_targetRule`, `messengerSilentAfterUpgrade`, `messengerSpeaks_without_targetRule`; passer path delegated to Halmos/Kontrol |
| v2.1 | coordinator (Quint v2 review) | Chains are identified by chain ID only; duplicate IDs among lockbox members allow a double spend without any hash collision | v2.1: `Chain` separated from `chainId`; hash on chain IDs; hypothesis `ChainIdUnique`; `cex_duplicateChainId`; W activation, predeploy preimage hardness and pre-Bedrock withdrawals listed as named assumptions |
| v2.2 | coordinator (design landed) | Cite the landed code; refer to the exporter by its constant; name the member-governance exporter-upgrade assumption; note the passer target rule | v2.2: citations at `5992028e08`; no hardcoded exporter address; `exporterGovernance` field, `exporterGovernanceUpgrade` action, `cex_exporterGovernanceUpgrade`; passer rule noted |
| v2 | R1 L4 | "Every configuration" is vacuous when trusted ≠ exporter | v2.1: qualified in the docstring and README |
| v2.3 | coordinator (non-vacuity audit, all layers) | Every headline theorem needs an automated, joint satisfiability witness | v2.3: `NonVacuity.lean` (one kernel-checked `nonvacuous_*` per headline theorem, each applying its theorem); `#assert_headline` in `Axioms.lean` fails the build if one is missing; no hypothesis found unsatisfiable |
| v2.3 review | R1, R2, R3 | (all) No critical or high finding and no contract issue; the governance action and `cex_exporterGovernanceUpgrade` (one field changed) check out; all twelve witnesses establish their theorem's hypotheses jointly and apply it; the `expired_no_relay_step` exception is sound | — |
| v2.3 review | R1 M2, R2 M, R3 M | `#assert_headline` only checks that `T` occurs in the witness; `@T = @T` would pass | Guard strengthened: the witness statement must be closed (no universe parameters, no leading binder, no mention of `T`); negative cases (`@T = @T`, an extra premise) fail the build. README and docstring call it a syntactic guard; witness adequacy is by review |
| v2.3 review | R1 M1, R2 L, R3 L | The axiom transcript was no longer printed by `lake build` | `#assert_headline` prints both footprints; section retitled; wording on `Classical` tied to the printed footprints |
| v2.3 review | R2 M, R3 M, R1 L | `ChainIdUnique` is stronger than OPCM's migration check | README: uniqueness over all modeled chains is a configuration obligation (rollout AC4); OPCM covers only migrated chains. Not weakened in the proof |
| v2.3 review | R2 M, R3 M | "Discharged outside Lean" overstated the exporter coverage | Each tool's scope stated (finite lengths, 600 bytes, chain ID 1); the general case rests on code inspection |
| v2.3 review | R1 L, R2 L, R3 M | `xDomainMessageSender` row cited proofs that mock the getters | Re-cited to Halmos's relay-gate, sender-field and outside-relay checks |
| v2.3 review | R1 L, R2 L, R3 L | Descriptions of `HistW` and silence omitted `exporterGovernance`; action table lacked the governance action | Docstrings, README and action table updated |
| v2.3 review | R1 L, R2 L, R3 L | `nonvacuous_safety` instantiates four of five conjuncts; `Cex` description wrong for two counterexamples | Descriptions narrowed; the two exceptions named |
| v2.3 review | R1 L, R3 L | Lockbox authorization compares L1 ProxyAdmin owners; the exporter is upgraded by the L2 one | Trust comparison reworded |
| v2.3 review | R3 L | "External calls only in `relayMessage`" ignores static reads | "Non-static external calls"; "before the relayed target call" |
| v2.3 review | R3 L | `SafeConfig.window` bounds W on non-standard destinations too | Accepted: an unnecessary restriction on configurations, not a gap; not changed |
| v2.3 review | R1 L | Kontrol passer names; contract tip not mentioned; supporting lemmas not on the headline list | Names fixed; tip sentence added; `expired_core`/`expired_not_relayable` are helpers, not headline results |
| v2.4 | coordinator (new contract behaviour) | `relayUndeliveredMessage` reverts while the receiving chain is paused | `paused` field, `pause`/`unpause` actions, `l1Relay` guard; every theorem re-proved with `Inv` unchanged; new headline `pauseOnlyDelays` with witness `nonvacuous_pauseOnlyDelays` (a fact relayed after a pause/unpause cycle still expires and refunds) |
| v2.4 | coordinator | `ProxyAdmin.upgrade` of the messenger leaves the stored period, 0 on a proxy never initialized for expiry | Stated as the model's assumption that every live messenger has its initialized period (rollout AC7, with its counterexample); `contractPeriod` stays fixed |
