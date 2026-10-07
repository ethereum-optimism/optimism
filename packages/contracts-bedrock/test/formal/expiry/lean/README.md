# Lean 4 proof: per-message interop expiry is safe (v2.1, exporter design)

**What this certifies.** This proves the safety of the **exporter design**, which is pending on
`karl/message-expiry-refunds` and not yet on the branch. In that design:

- `relayUndeliveredMessage` trusts only the new `UndeliveredMessageExporter` at 0x4200..002E;
- P_contract = 8 days;
- W_protocol ≤ 7 days.

The theorem: under the stated hypotheses, ETH is never both delivered on a message's destination
and refunded on its (standard) source. It holds for any number of chains, sources, messages and
lockbox joins, any length of time, and any attacker-chosen relay targets.

**The contracts at `37b44c48c7` are a different configuration.** They are
`cfgMessengerTrusted` (`trusted = messenger`): `relayUndeliveredMessage` trusts 0x..23. That
configuration is **not** covered. `cex_messengerTrusted` shows it double-spends without v1's
activation premise ("no legacy forged withdrawal from 0x..23 on any lockbox chain").

The theorem applies to deployed code only once all of these land:

1. The `UndeliveredMessageExporter` predeploy at 0x4200..002E. Exports move out of the messenger.
2. `L1CrossDomainMessenger.relayUndeliveredMessage` checks `xDomainMessageSender() == 0x4200..002E`, not 0x..23.
3. The INTEROP feature gate in `relayUndeliveredMessage`.
4. P_contract = 8 days in `L2ToL2CrossDomainMessenger` (today `MESSAGE_EXPIRY_WINDOW = 7 days`).
5. A W_protocol ≤ 7-day cap on the dependency-set override, in both op-core (Go) and kona (Rust).

The model mirrors `../quint/expiry.qnt`, with the same action and property names where they exist.

- **Dependencies:** core Lean 4.34.1 only.
- **Holes:** no `sorry`, `admit` or `native_decide`.
- **Axioms:** no `axiom` declarations, and no use of `Classical`.
- **Axiom report:** `#print axioms` lists only `propext` and `Quot.sound`, or nothing at all.
- **Bounds:** none. Chains, bodies and hashes are arbitrary types, time is `Nat`, and executions are finite but of any length.

## Build

```
lake build        # 2–6 s from clean on hel1; prints the #print axioms report
```

| File | Contents |
| --- | --- |
| `Expiry/Model.lean` | Types, `Config`, `State`, actions (`guard`, `next`, `Step`), `Reach`, `Init`, `GovInit`, `HashInjective`, `SafeConfig`, property definitions |
| `Expiry/Invariant.lean` | `HistW` (any config), `Inv` (state invariant), and their preservation proofs |
| `Expiry/Safety.lean` | Main theorems |
| `Expiry/Counterexamples.lean` | Concrete instance: witnesses, safe variants, counterexamples |
| `Expiry/Axioms.lean` | `#print axioms` for each theorem |

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

- `exporter`: 0x..2E;
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

**Actions.** `guard` gives the enabling condition and `next` the effect.

| Action (Quint name) | Guard | Effect |
| --- | --- | --- |
| `tick y t` | `clock y < t` | `clock y := t` (each chain's clock is monotone on its own) |
| `upgrade y` | y standard, not yet upgraded | `upgraded y`: the exporter proxy gets its implementation; the messenger records timestamps and has the target rule |
| `join z y` | `govCheck → standard y` | `lockbox z y` |
| `send z d b` | `d ≠ z`; fresh hash (unique nonce); after the upgrade with the target rule, the target is not the L2CDM | event at `clock z`; `sentAt := clock z` only if z is upgraded |
| `resend z d b` | `cfg.resend`, `sentAt ≠ 0`, and `¬expired` if it restarts | event at `clock z`; `sentAt := clock z` if it restarts |
| `resendLegacy z d b` | z not upgraded, and H was sent (the old permissionless `resendMessage`, allowed when `sentMessages[nonce] == H`) | event at `clock z`; no timestamp |
| `relay x z b` (with `.l2cdm f` this is `relayToL2CrossDomainMessenger`) | `¬relayed x H`; if x is standard, there is an event `e` of z for H with `clock x ≤ e + W_x`; if x is standard, upgraded and has the target rule, the target is not the L2CDM | `relayed x H`; plus `(x, messenger, f)` when `decode b = .l2cdm f` |
| `exportUndelivered y z b route` | y standard and upgraded, `¬relayed y (hash y z b)` | withdrawal `(y, cfg.trusted, (route, hash y z b, clock y))` |
| `userWithdrawal y a f` | none | `(y, user a, f)` |
| `arbitraryCode y s f` | y not standard | `(y, s, f)` |
| `arbitraryEvent z h t` | z not standard | event `(z, h, t)` |
| `l1Relay w` | w exists; `unsafeTargetCheck → origin ≠ toL1`; `lockboxCheck → lockbox toL1 origin`, read at relay time; `senderCheck → sender = trusted`; interop gate on `toL1` | deposit `w.fact` to `toL1` |
| `fakeCaller f` | `¬realMessengerCheck ∨ ¬lockboxCheck ∨ ¬sysConfigConsistent` | deposit `f` |
| `l1cdmSelfRelay f` | `¬unsafeTargetCheck` | deposit `f` |
| `expire f` | deposit; `sentAt toL1 hash ≠ 0`; `expiredBy` (`sentAt + P < time`, or `≤` if `expireGe`) | `expired toL1 hash` |
| `refund z d b` | `isBridge b`; `expired z (hash d z b)`; `¬refunded` | `refunded`, `refunds += 1` |
| `arbitraryRefund z h` | z not standard | `refunded z h`, `refunds += 1` |

`fakeCaller` has a three-way guard because the checks only work together. A contract that is not
an L1CDM can return a fake portal whose fake SystemConfig names it. That passes the
real-messenger check, and only the lockbox check (an authorized portal's SystemConfig names the
real L1CDM) stops it. Conversely, a fake caller returning a real authorized portal fails the
real-messenger check.

`refund` rebuilds the hash with source = z (`block.chainid`), so only the true source's refund
matches, and a fact routed to the wrong chain is harmless.

## Hypotheses

Every hypothesis is an explicit argument or structure field.

| Hypothesis | Lean | Real-world fact |
| --- | --- | --- |
| Exporter is trusted | `SafeConfig.trusted` | `relayUndeliveredMessage` checks `xDomainMessageSender() == 0x4200..002E` (pending). |
| Real-messenger check | `SafeConfig.realMessengerCheck` | `IL1CDM(msg.sender).portal().systemConfig().l1CrossDomainMessenger() == msg.sender`. |
| Lockbox check | `SafeConfig.lockboxCheck` | `portal.ethLockbox().authorizedPortals(callerPortal)`, read at L1 relay time. |
| Sender check | `SafeConfig.senderCheck` | The `xDomainMessageSender()` comparison. |
| L1CDM self-target rule | `SafeConfig.unsafeTargetCheck` | `L1CrossDomainMessenger._isUnsafeTarget` blocks relays to itself and its portal. So an L1CDM is the L1 sender of an L1→L2 message to the L2 messenger only through `relayUndeliveredMessage`. This is encoded by deposits arising only from `l1Relay` (and from `fakeCaller` / `l1cdmSelfRelay` when a check is dropped). |
| SystemConfig consistency (governance) | `SafeConfig.sysConfigConsistent` | For every portal authorized in a lockbox, `systemConfig.l1CrossDomainMessenger()` is that chain's real L1CDM. |
| Governance join rule | `SafeConfig.govCheck`, plus `hg : GovInit cfg s₀` | Only **standard** chains are ever authorized in a lockbox. A standard chain ran the standard predeploys for its whole history, and its relays obey the protocol window. |
| Windows | `SafeConfig.window : ∀ d, W_d + (if expireGe then 1 else 0) ≤ P` | W_d ≤ 7 days (config cap) ≤ P = 8 days with the strict check. With `≥` it requires W_d < P. |
| No non-restarting resend | `SafeConfig.resend` | `resendMessage` is removed after the upgrade. A restarting resend would also be safe. Pre-upgrade resends are always modeled (`resendLegacy`). |
| Idealized hash | `hinj : HashInjective cfg.hash` | See "The hash" below. |
| Unique chain IDs (governance) | `hid : ChainIdUnique cfg := ∀ c c', standard c → chainId c = chainId c' → c = c'` | Every standard chain has a chain ID that no other chain uses. Standard chains are those that are or can become lockbox members, and the protected sources. OPCM checks duplicate IDs on migration. Without this, a member sharing B's ID exports "not relayed" for a hash delivered on B (`cex_duplicateChainId`), with no hash collision. |
| Genesis | `h0 : Init s₀` | At every chain's genesis nothing has been sent, relayed, withdrawn or deposited, and no chain is upgraded. Clocks and initial lockbox memberships are arbitrary. |

The messenger's unsafe-target rule (`cfg.targetRule`) is **not** a hypothesis. Karl kept it as
defense in depth, and safety does not depend on it:

- `safety_without_targetRule` proves the full `safety` conjunction for configurations with
  `targetRule = false`.
- The rule's own property is `messengerSilentAfterUpgrade`: with the rule, an upgraded standard
  chain's 0x..23 never initiates a withdrawal.
- `messengerSpeaks_without_targetRule` shows that property fails without the rule.
- The L2ToL1MessagePasser path (0x..23 → passer) is not modeled as a sender here. It is checked on
  bytecode by Halmos/Kontrol (`check_OnlyExportReachesL1_relay_passer_PENDING`,
  `prove_relayMessage_neverMakesMessengerCallPasser_PENDING`, the `*_passer_PENDING` checks).

**Deployment assumption: historical inertness.** 0x4200..002E is not code-free before the upgrade.
`scripts/L2Genesis.s.sol:226-240` (`setPredeployProxies`) etches the `Proxy` at every proxied
predeploy slot (admin = ProxyAdmin), and sets an implementation only for supported predeploys.
With no implementation, every call reverts (`src/universal/Proxy.sol:124-126`, `_doProxyCall`:
`require(impl != address(0))`). So the exporter is silent before its upgrade because its proxy has
no implementation.

The residual assumption is that **no ProxyAdmin action set an implementation at 0x..2E before the
network upgrade, on any chain that is or becomes a lockbox member**. In the model, `Init` says no
chain is upgraded at genesis, and for a standard chain `upgrade` is the only action that gives the
exporter an implementation.

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
def expiredBy cfg sent t := if cfg.expireGe then sent + cfg.contractPeriod ≤ t
                            else sent + cfg.contractPeriod < t
def NoDoubleSpend cfg s := ∀ d z b, cfg.standard z →
  ¬ (s.relayed d (cfg.hash d z b) ∧ s.refunded z (cfg.hash d z b))
def RefundImpliesExpired cfg s := ∀ z h, cfg.standard z → s.refunded z h → s.expired z h
def AtMostOneRefund cfg s := ∀ z h, cfg.standard z → s.refunds z h ≤ 1
def ExportedBy cfg s₀ s y f := ∃ z b s₁ s₂, Reach cfg s₀ s₁ ∧
  Step cfg (.exportUndelivered y z b f.toL1) s₁ s₂ ∧ Reach cfg s₂ s ∧
  f.hash = cfg.hash y z b ∧ f.time = s₁.clock y
def NoForgedFact cfg s₀ s := ∀ f, s.deposits f → ∃ y, cfg.standard y ∧ ExportedBy cfg s₀ s y f
def withinWindow cfg s x z h t := ∃ e, s.events z h e ∧ t ≤ e + cfg.protocolWindow x

-- Silence derived within the model. Any configuration; needs only (h0 : Init s₀) (hr : Reach cfg s₀ s).
-- Informative when cfg.trusted = .exporter; otherwise exportUndelivered records the other sender
-- and the statement is vacuous.
theorem exporterSilentBeforeUpgrade (w) (hw : s.withdrawals w) (hsnd : w.sender = .exporter)
    (hstd : cfg.standard w.origin) :
    ∃ z b s₁ s₂, Reach cfg s₀ s₁ ∧ Step cfg (.exportUndelivered w.origin z b w.fact.toL1) s₁ s₂ ∧
      Reach cfg s₂ s ∧ s₁.upgraded w.origin = true ∧ ¬ s₁.relayed w.origin (cfg.hash w.origin z b) ∧
      w.fact.hash = cfg.hash w.origin z b ∧ w.fact.time = s₁.clock w.origin

-- Joins need no history re-check (same scope as above).
theorem joinNeedsNoHistoryCheck (h0 : Init s₀) (hr : Reach cfg s₀ s)
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

**`HistW`** (any config, from `Init`): a withdrawal whose sender is the exporter and whose origin
is standard was created by an `exportUndelivered` step. The other withdrawal-creating steps record
a different sender:

- `relay` → messenger;
- `userWithdrawal` → user;
- `arbitraryCode` → only on non-standard chains.

**`Inv`** (safe config):

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

Each `cex_*` proves `Cex cfg trace`, which is:

- genesis, `GovInit` and injectivity hold;
- `¬ SafeConfig cfg`;
- the trace is valid and reachable;
- the final state violates `NoDoubleSpend`.

Each config is `base` with **one** field changed, with two exceptions:

- `cex_nonStrict` changes two fields (`expireGe` and `contractPeriod := 7`), because `≥` alone with P = 8 is safe.
- `cex_hashCollision` keeps every `SafeConfig` field and drops injectivity instead.

| Theorem | Dropped assumption | Execution |
| --- | --- | --- |
| `cex_messengerTrusted` | Exporter trusted. This is the **contracts at 37b44c48c7**. | Before B's upgrade, B relays body 9 to the L2CDM, so 0x..23 sends the forged fact. After the upgrades A sends, B relays, the pre-staged fact goes through l1Relay, then expire and refund. |
| `cex_nonstandardJoin` | Governance join rule | D signs the forged fact as the "exporter" and then joins A's lockbox. Its old withdrawal becomes trusted. |
| `cex_periodBelowWindow` | P ≥ W (here P = 6 < W = 7) | B exports at its time 8 > 1 + 6 and relays at 8 ≤ 1 + 7. |
| `cex_nonStrict` | Strict `>` at P = W = 7 | The same edge execution. |
| `cex_resendNoRestart` | No non-restarting resend | Resend at A-time 10; export and expire against sentAt = 1; relay using the event at 10. |
| `cex_noRealMessengerCheck` | Real-messenger check | `fakeCaller`: a contract returning a real authorized portal. |
| `cex_noLockboxCheck` | Lockbox check | D's withdrawal, relayed by D's L1CDM. |
| `cex_noLockboxCheck_fakePortal` | Lockbox check | `fakeCaller`: a fake portal whose fake SystemConfig names the caller passes the real-messenger check. |
| `cex_sysConfigInconsistent` | SystemConfig consistency | `fakeCaller` passes both identity checks. |
| `cex_noUnsafeTargetCheck` | L1CDM self-target rule | `l1cdmSelfRelay`: A's L1CDM sends `expireMessage` as itself. |
| `cex_noSenderCheck` | Sender check | A user contract on B sends the forged fact. |
| `cex_hashCollision` | Injectivity (`SafeConfig` holds) | C honestly exports a colliding hash after B relayed. |
| `cex_duplicateChainId` | Unique chain IDs (`SafeConfig`, `HashInjective` and `GovInit` hold) | C is standard but has B's chain ID. B relays A's message; C joins A's lockbox and exports the same hash; expire and refund. |

## `#print axioms` (from `lake build`)

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
'Expiry.Examples.safe_variants' does not depend on any axioms
every other witness and cex_* theorem: [propext, Quot.sound]
```

## Where W and P are enforced

The design change had **not** landed when this was written. The branch tip
`karl/message-expiry-refunds`, fetched on 2026-10-07, is `37b44c48c7`. At that commit:

- **Contract period.** `packages/contracts-bedrock/src/L2/L2ToL2CrossDomainMessenger.sol:75` has `MESSAGE_EXPIRY_WINDOW = 7 days`. The strict check is at `:319`.
  - *Intended:* 8 days (P_contract).
- **Protocol window, Go.** The default is `op-core/interop/depset/static_depset.go:15` (604800 s). The override is applied at `:165-170`; the validity rule is at `op-core/interop/depset/links.go:73`. The override is uncapped.
  - *Intended:* reject overrides above 7 days.
- **Protocol window, kona.** `rust/kona/crates/protocol/genesis/src/interop/constants.rs:5` and `rust/kona/crates/protocol/genesis/src/interop/depset.rs:30-35`. The override is uncapped.
  - *Intended:* reject overrides above 7 days.
- **L1 relay.** `src/L1/L1CrossDomainMessenger.sol:104-120` trusts 0x..23 and has no INTEROP gate.
  - *Intended:* trust 0x..2E and add the gate.

## Discharged outside Lean

The proof names below are from the sibling directories. They target the 37b44c48c7 contracts and
must be re-targeted to the exporter once it lands.

| Obligation | Where |
| --- | --- |
| **L1 identity and reverse binding.** `relayUndeliveredMessage` succeeds only if the caller's portal's SystemConfig names the caller, the portal is authorized in this chain's lockbox, and `xDomainMessageSender()` is the trusted sender. It then deposits exactly `expireMessage(H, t)`. | `halmos/L1CDMExpiryHalmos.t.sol` (`check_relayUndelivered_iff_and_deposit`), `kontrol/l1` (`prove_relayUndeliveredMessage_spec`, `prove_relayUndeliveredMessage_symbolicPortalChain`) |
| **SystemConfig ↔ L1CDM consistency** for authorized portals. | Governance (lockbox authorization, SystemConfig ownership); the `sysConfigConsistent` field |
| **`xDomainMessageSender` semantics.** On L1 it is the `msg.sender` that called the L2CDM for that withdrawal. On L2, the L2CDM's `xDomainMessageSender()` equals `otherMessenger` only for deposits sent by this chain's L1CDM. | Standard CrossDomainMessenger code; `prove_expireMessage_spec`, `check_expire_iff` |
| **The L1CDM is an L1→L2 sender to the L2 messenger only through `relayUndeliveredMessage`** (`_isUnsafeTarget`). | Standard code; encoded by deposits arising only from `l1Relay` / `fakeCaller` / `l1cdmSelfRelay`, the last two only when a check is dropped |
| **Replay envelopes.** Failed L1 relays and failed L2 deposits replay only the original message (versioned hash). This is not covered by `HashInjective`. | Standard code; withdrawals and deposits are conservatively never removed |
| **Exporter code.** H uses destination = `block.chainid`; it requires `!successfulMessages(H)`; it calls only `L2CDM.sendMessage(sourceMessenger, relayUndeliveredMessage(H, block.timestamp), gas)`. | To be re-targeted from `check_export_binding`, `prove_exportUndeliveredMessage_reachesL2CDM` |
| **Messenger code.** External calls happen only in `relayMessage`; `successfulMessages` is set before the call; the target rules hold; `sentAt = block.timestamp`. | `prove_relayMessage_*`, `prove_sendMessage_*`, `check_UnsafeTargetRule_*`, `check_OnlyExportReachesL1_*` |
| **Expiry check.** Strict boundary, `sentAt ≠ 0`. | `check_expire_boundary`, `check_expire_iff_unbounded` |
| **P ≥ W cap.** | `check_contractWindowCoversProtocolCap`, `prove_expiryWindow_atLeastProtocolWindow` |
| **Refund preimage binding.** Source = `block.chainid`, sender = target = bridge, `relayETH(from, to, amount)`, pays at most once. The model keeps the source binding and abstracts the rest as `isBridge`. | `prove_refundETH_preimageBinding`, `prove_refundETH_singleUse`, `check_refund_iff_effects_singleUse` |
| **Nonce freshness.** | The `send` freshness guard, with `HashInjective` |
| **Historical inertness of 0x..2E.** | Deployment assumption (above) |

## Named assumptions not modeled as transitions

- **W activation and time-varying W.** The protocol rule `exec − init ≤ W_d` is enforced on a destination before that destination's exporter goes live. W_d never later rises above P. The model has fixed W_d from genesis; activation of the W rule and changes to W are not modeled.
- **Preimage hardness of predeploy addresses.** No EOA, and no aliased L1 address (`AddressAliasHelper`), equals 0x..2E or 0x..23. This is why `userWithdrawal` can only record `user a` and never a predeploy as sender.
- **Pre-Bedrock legacy withdrawals.** Legacy (pre-Bedrock) L2→L1 messages are irrelevant: none carries `relayUndeliveredMessage` from 0x..2E. The model's genesis is the Bedrock-era history.
- **Historical inertness of 0x..2E** and the **governance assumptions** (standard-only joins, unique chain IDs, SystemConfig consistency): see "Hypotheses" above.

## Modeling choices and what is NOT modeled

- **Clocks.** Each chain's clock is monotone; timestamps are `Nat`.
  - Overflow: Solidity 0.8 `sentAt + P` is checked arithmetic and reverts on overflow, which only prevents expiry (the safe direction).
  - The rule `initTimestamp ≤ execTimestamp` (`links.go:70`) is omitted, which only makes the model more permissive.
- **Finality.** `withdrawals` and the initiating events are those of the canonical histories. L1 reorgs beyond finality are out of scope.
- **Lockbox and configuration changes.** Joins are modeled. Leaving a lockbox, arbitrary later upgrades, and changes to the windows or gate are not modeled.
- **Interop gate.** It only restricts `l1Relay`. Safety does not depend on it.
- **EVM and economics.** Gas, value, reentrancy, call failures, balances and `ETHLiquidity` are left to the other tools.

## Review log

| Round | Reviewer | Finding | Disposition |
| --- | --- | --- | --- |
| v1 | Codex astra, Codex sol | Attacker completeness assumed: forging disabled by `SafeConfig` flags | v2: target-level relay (`decode`), senders recorded per call, `userWithdrawal`/`arbitraryCode`; "only the exporter speaks" derived |
| v1 | astra, sol | Activation premise assumed, not enforced | v2: genesis-based model with `upgrade`; `exporterSilentBeforeUpgrade` derived; residual is the governance plus historical-inertness assumption (v2.1) |
| v1 | astra, sol | Global injectivity is unsatisfiable for keccak | Reworded as an idealization (v2; tightened in v2.1) |
| v1 | sol | P < W counterexample used W = 9 > cap | v2: P = 6, W = 7 |
| v1 | sol, astra | `NoForgedFact` not a named conjunct | v2: `noForgedFact`, a conjunct of `safety` |
| v1 | sol | Refund binding, per-chain clocks, overflow | v2: source-bound `refund`, per-chain clocks; overflow noted |
| v1 | astra, sol | README 7/8-day drift; no interop gate | v2: two parameters W_d and P; gate modeled; code citations |
| v1 | sol | `externalRawTrust` models a non-existent contract | v2: dropped |
| v1 | Claude | Cite W cap and P code | v2: "Where W and P are enforced" |
| v1 | Claude | `>=` boundary counterexample | v2: `cex_nonStrict` |
| v1 | Claude | Per-destination windows | v2: `protocolWindow : Chain → Nat` |
| v1 | Claude | Witness that a relay is reachable | v2: `relay_reachable_at_edge` |
| v1 | Claude | SystemConfig ↔ L1CDM consistency | v2: README; v2.1: `sysConfigConsistent` field and counterexample |
| v1 | Claude | Joins need no history re-check | v2: `joinNeedsNoHistoryCheck`; governance counterexample |
| v2 | Claude M1, sol M1 | "0x..2E never had code" is false: the genesis Proxy is at every slot | v2.1: wording fixed; silence comes from the empty proxy implementation; historical-inertness assumption stated; theorems scoped to "within the model" |
| v2 | Claude M2 | `fakeCaller` mis-attributed: a fake portal passes the real-messenger check | v2.1: guard `¬real ∨ ¬lockbox ∨ ¬sysConfig`; `cex_noLockboxCheck_fakePortal`, `cex_sysConfigInconsistent` |
| v2 | sol M2 | RefundImpliesExpired / AtMostOneRefund claimed for all chains | v2.1: scoped to standard chains; `arbitraryRefund` for non-standard bridges |
| v2 | astra M | Pre-upgrade resends not representable | v2.1: `resendLegacy` modeled; `legacyResend_reachable` |
| v2 | astra L | `cex_nonStrict` changes two fields | v2.1: wording fixed; single-field `≥` (P = 8) proved safe in `safe_variants` (window hypothesis generalized) |
| v2 | Claude L1 | README overstated `HashInjective` | v2.1: README matches the formal statement; envelope integrity listed as a separate obligation |
| v2 | Claude L2 | `_isUnsafeTarget` labeling | v2.1: the safety-relevant fact is stated; `l1cdmSelfRelay` plus `cex_noUnsafeTargetCheck` |
| v2 | Claude H1, astra, sol | Make clear what is certified | v2.1: first paragraph; 37b44c48c7 = `cfgMessengerTrusted`; list of code changes that must land |
| v2.1 | coordinator (Karl's decision) | Keep the target rule as defense in depth; safety must not depend on it | v2.1: `safety_without_targetRule`, `messengerSilentAfterUpgrade`, `messengerSpeaks_without_targetRule`; passer path delegated to Halmos/Kontrol |
| v2.1 | coordinator (Quint v2 review) | Chains are identified by chain ID only; duplicate IDs among lockbox members allow a double spend without any hash collision | v2.1: `Chain` separated from `chainId`; hash on chain IDs; hypothesis `ChainIdUnique`; `cex_duplicateChainId`; W activation, predeploy preimage hardness and pre-Bedrock withdrawals listed as named assumptions |
| v2 | Claude L4 | "Every configuration" is vacuous when trusted ≠ exporter | v2.1: qualified in the docstring and README |
