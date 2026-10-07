/-
Abstract protocol model of per-message interop expiry, design v2 (exporter predeploy).

Mirrors the Quint model `../quint/expiry.qnt` (same action and property names where they exist),
but chains, message bodies and hashes are arbitrary types and time, chains, messages and
execution length are unbounded.

The model is at the level of "who can make which L2->L1 message exist": every withdrawal carries
the L2 address that called L2CrossDomainMessenger.sendMessage (the L2CDM records msg.sender and L1
exposes it as xDomainMessageSender). The attacker acts through targets: a relayed message calls an
attacker-chosen target with attacker-chosen calldata (`Config.decode`), any user contract can send
any withdrawal under its own address, and a non-standard chain can do anything. That only the
exporter predeploy makes the trusted sender speak, and that it never spoke before the upgrade, are
theorems (`Safety.lean`), not assumptions. They are derived within this action model; they do not
establish real deployment history (see `Init` and the README's deployment assumption).

A message is identified by its preimage `(d, z, b)`: destination `d`, source `z`, rest `b : Body`
(nonce, sender, target, payload). Its hash is `cfg.msgHash d z b = cfg.hash (chainId d) (chainId z) b`:
the contracts know chains only by their chain IDs (`block.chainid`).
-/
namespace Expiry

/-- The L2 contract that called L2CrossDomainMessenger.sendMessage. On L1 it is the relaying
L1CrossDomainMessenger's `xDomainMessageSender()`. -/
inductive Sender where
  /-- 0x4200..002E UndeliveredMessageExporter. Before its upgrade the address holds the genesis
  Proxy with no implementation, which reverts every call (Proxy.sol `_doProxyCall`). -/
  | exporter
  /-- 0x4200..0023 L2ToL2CrossDomainMessenger. -/
  | messenger
  /-- Any other address: an EOA or a user contract. -/
  | user (a : Nat)
  deriving DecidableEq, Repr

/-- The payload `relayUndeliveredMessage(hash, time)` addressed to chain `toL1`'s
L1CrossDomainMessenger (the exporter's caller-chosen `sourceMessenger`). Honest meaning: "hash,
computed with the exporting chain as destination, had not been relayed there at its time `time`". -/
structure Fact (Chain Hash : Type) where
  toL1 : Chain
  hash : Hash
  time : Nat
  deriving DecidableEq

/-- An L2->L1 message sent through chain `origin`'s L2CrossDomainMessenger by `sender`. Only
messages carrying a `relayUndeliveredMessage` payload are tracked; others cannot reach it. -/
structure Withdrawal (Chain Hash : Type) where
  origin : Chain
  sender : Sender
  fact : Fact Chain Hash
  deriving DecidableEq

/-- What a relayed message's target call does, as far as L2->L1 messages go. -/
inductive Call (Chain Hash : Type) where
  /-- Target = L2CrossDomainMessenger, calldata = sendMessage(toL1's L1CDM,
  relayUndeliveredMessage(h, t)): the L2CDM records 0x..23 as the sender. -/
  | l2cdm (f : Fact Chain Hash)
  /-- Any other target. A user contract it calls can send withdrawals under its own address
  (`userWithdrawal`). The exporter, if called, only runs its export (`exportUndelivered`). The
  L2ToL1MessagePasser makes a withdrawal that the portal executes itself (msg.sender = portal on
  L1), which fails the real-messenger check. -/
  | other

def Call.isL2CDM {Chain Hash : Type} : Call Chain Hash → Prop
  | .l2cdm _ => True
  | .other => False

/-- Parameters of the world and of the code. Every assumption of the safety theorem is a property
of these fields (`SafeConfig`, `HashInjective`) or of the genesis state (`Init`, `GovInit`). -/
structure Config (Chain Body Hash : Type) where
  /-- L2 chain ID (uint256) of each chain. -/
  chainId : Chain → Nat
  /-- Message hash of (destination chain ID, source chain ID, rest). -/
  hash : Nat → Nat → Body → Hash
  /-- Target and calldata of a message body, as far as the relay's call goes. The theorems hold for
  every `decode`, so also for one that reaches every `Call` (the attacker chooses the body). -/
  decode : Body → Call Chain Hash
  /-- Bodies that SuperchainETHBridge.refundETH can rebuild (sender = target = bridge,
  relayETH(from, to, amount)). -/
  isBridge : Body → Prop
  /-- Chains that run the standard predeploy code over their whole history (exporter, messenger,
  L2CDM) and whose relays obey the protocol window. Other chains run arbitrary code. -/
  standard : Chain → Prop
  /-- SystemConfig INTEROP feature of the chain whose L1CDM receives `relayUndeliveredMessage`. -/
  interop : Chain → Prop
  /-- W_protocol per destination: a relay on `x` is valid iff exec - init ≤ protocolWindow x. -/
  protocolWindow : Chain → Nat
  /-- P_contract: expireMessage requires t > sentAt + P_contract (`≥` if `expireGe`). -/
  contractPeriod : Nat
  /-- The sender `relayUndeliveredMessage` trusts: `exporter` (v2) or `messenger` (earlier design,
  in which the export function lived in the messenger). -/
  trusted : Sender
  /-- After its upgrade the messenger rejects target == L2CrossDomainMessenger on send and relay. -/
  targetRule : Bool
  /-- The three checks of relayUndeliveredMessage, and the lockbox-join governance rule. -/
  realMessengerCheck : Bool
  lockboxCheck : Bool
  senderCheck : Bool
  govCheck : Bool
  /-- L1CDM._isUnsafeTarget: the L1CDM never relays to itself (or its portal), so it is the L1
  sender of an L1->L2 message to the L2 messenger only through relayUndeliveredMessage. -/
  unsafeTargetCheck : Bool
  /-- Governance: for every portal authorized in a lockbox, `systemConfig.l1CrossDomainMessenger()`
  is that chain's real L1CDM. -/
  sysConfigConsistent : Bool
  /-- Boundary mutation: `t ≥ sentAt + P` instead of `t > sentAt + P`. -/
  expireGe : Bool
  /-- The removed resendMessage, and whether it restarts the recorded timestamp. -/
  resend : Bool
  resendRestarts : Bool

/-- Protocol state. Sets are predicates. -/
structure State (Chain Hash : Type) where
  /-- Per-chain block timestamps. -/
  clock : Chain → Nat
  /-- The network upgrade happened on this chain (exporter has code, messenger records
  timestamps and has the target rule). -/
  upgraded : Chain → Bool
  /-- `lockbox z y`: chain y's portal is authorized in chain z's ETHLockbox. -/
  lockbox : Chain → Chain → Prop
  /-- `sentMessageTimestamps` of each chain (0 = none). -/
  sentAt : Chain → Hash → Nat
  /-- `events z h e`: chain z emitted a SentMessage event for h at its time e. -/
  events : Chain → Hash → Nat → Prop
  /-- `successfulMessages` of each chain. -/
  relayed : Chain → Hash → Prop
  /-- L2->L1 messages in the finalized canonical histories. Never removed: withdrawals do not
  expire on L1 and a failed L1 relay can be replayed. -/
  withdrawals : Withdrawal Chain Hash → Prop
  /-- expireMessage(hash, time) deposits sent to chain `f.toL1`. Never removed (failed deposits
  can be replayed). -/
  deposits : Fact Chain Hash → Prop
  expired : Chain → Hash → Prop
  refunded : Chain → Hash → Prop
  /-- Ghost: refundETH payouts per (chain, hash). -/
  refunds : Chain → Hash → Nat

/-- Actions; names follow the Quint model (`export` is a keyword in both languages). -/
inductive Action (Chain Body Hash : Type) where
  /-- Chain y's clock advances to t. -/
  | tick (y : Chain) (t : Nat)
  /-- The network upgrade on standard chain y. -/
  | upgrade (y : Chain)
  /-- Governance authorizes chain y's portal in chain z's lockbox. -/
  | join (z y : Chain)
  /-- Chain z: sendMessage of message (d, z, b) (e.g. SuperchainETHBridge.sendETH). -/
  | send (z d : Chain) (b : Body)
  /-- Chain z: the removed resendMessage (post-upgrade variants, switch `Config.resend`). -/
  | resend (z d : Chain) (b : Body)
  /-- Chain z before its upgrade: the old permissionless resendMessage (allowed when
  `sentMessages[nonce] == H`): a new initiating event, no timestamp record. -/
  | resendLegacy (z d : Chain) (b : Body)
  /-- Chain x: relayMessage of (x, z, b), followed by the call `decode b`. With
  `decode b = .l2cdm f` this is the Quint `relayToL2CrossDomainMessenger`. -/
  | relay (x z : Chain) (b : Body)
  /-- Chain y: the exporter with preimage (y, z, b), routed to chain `route`'s L1CDM. -/
  | exportUndelivered (y z : Chain) (b : Body) (route : Chain)
  /-- Anyone on chain y: L2CDM.sendMessage with any payload; the sender is the caller. -/
  | userWithdrawal (y : Chain) (a : Nat) (f : Fact Chain Hash)
  /-- A non-standard chain: a withdrawal with any recorded sender. -/
  | arbitraryCode (y : Chain) (s : Sender) (f : Fact Chain Hash)
  /-- A non-standard chain: a SentMessage event for any hash at any time. -/
  | arbitraryEvent (z : Chain) (h : Hash) (t : Nat)
  /-- L1: w is proven, finalized and relayed by w.origin's L1CDM into
  `w.fact.toL1`'s L1CDM.relayUndeliveredMessage, which deposits expireMessage to that chain. -/
  | l1Relay (w : Withdrawal Chain Hash)
  /-- L1: a contract that is not a real L1CDM calls relayUndeliveredMessage (e.g. returning a
  fake portal whose fake SystemConfig names it). -/
  | fakeCaller (f : Fact Chain Hash)
  /-- L1: an L1CDM relays a withdrawal whose target is itself, calling its own sendMessage, so
  it becomes the L1 sender of an arbitrary expireMessage deposit. -/
  | l1cdmSelfRelay (f : Fact Chain Hash)
  /-- A non-standard chain's bridge marks any hash refunded and pays out. -/
  | arbitraryRefund (z : Chain) (h : Hash)
  /-- Chain `f.toL1`: the deposit runs expireMessage. -/
  | expire (f : Fact Chain Hash)
  /-- Chain z: SuperchainETHBridge.refundETH; it rebuilds the hash of (d, z, b). -/
  | refund (z d : Chain) (b : Body)

variable {Chain Body Hash : Type} [DecidableEq Chain] [DecidableEq Hash]

/-- The hash of message (d, z, b): computed from the chain IDs. -/
def Config.msgHash (cfg : Config Chain Body Hash) (d z : Chain) (b : Body) : Hash :=
  cfg.hash (cfg.chainId d) (cfg.chainId z) b

/-- Point update of a two-argument map. -/
def upd2 {α β γ : Type} [DecidableEq α] [DecidableEq β] (f : α → β → γ) (a : α) (b : β) (v : γ) :
    α → β → γ :=
  fun a' b' => if a' = a ∧ b' = b then v else f a' b'

/-- Point update of a one-argument map. -/
def upd1 {α γ : Type} [DecidableEq α] (f : α → γ) (a : α) (v : γ) : α → γ :=
  fun a' => if a' = a then v else f a'

/-- expireMessage's period check. -/
def expiredBy (cfg : Config Chain Body Hash) (sent t : Nat) : Prop :=
  if cfg.expireGe then sent + cfg.contractPeriod ≤ t else sent + cfg.contractPeriod < t

/-- The withdrawals a relay's target call creates on chain x. -/
def callOut (x : Chain) : Call Chain Hash → Withdrawal Chain Hash → Prop
  | .l2cdm f, w => w = ⟨x, .messenger, f⟩
  | .other, _ => False

/-- Some initiating event of h on z is within x's protocol window at x-time t. -/
def withinWindow (cfg : Config Chain Body Hash) (s : State Chain Hash) (x z : Chain) (h : Hash)
    (t : Nat) : Prop :=
  ∃ e, s.events z h e ∧ t ≤ e + cfg.protocolWindow x

/-- Enabling condition of each action. -/
def guard (cfg : Config Chain Body Hash) : Action Chain Body Hash → State Chain Hash → Prop
  | .tick y t, s => s.clock y < t
  | .upgrade y, s => cfg.standard y ∧ s.upgraded y = false
  | .join _ y, _ => cfg.govCheck = true → cfg.standard y
  | .send z d b, s =>
      d ≠ z ∧ (¬ ∃ e, s.events z (cfg.msgHash d z b) e) ∧
      (s.upgraded z = true → cfg.targetRule = true → ¬ (cfg.decode b).isL2CDM)
  | .resend z d b, s => cfg.resend = true ∧ s.sentAt z (cfg.msgHash d z b) ≠ 0 ∧
      (cfg.resendRestarts = true → ¬ s.expired z (cfg.msgHash d z b))
  | .resendLegacy z d b, s => s.upgraded z = false ∧ ∃ e, s.events z (cfg.msgHash d z b) e
  | .relay x z b, s =>
      ¬ s.relayed x (cfg.msgHash x z b) ∧
      (cfg.standard x → withinWindow cfg s x z (cfg.msgHash x z b) (s.clock x)) ∧
      (cfg.standard x → s.upgraded x = true → cfg.targetRule = true → ¬ (cfg.decode b).isL2CDM)
  | .exportUndelivered y z b _, s =>
      cfg.standard y ∧ s.upgraded y = true ∧ ¬ s.relayed y (cfg.msgHash y z b)
  | .userWithdrawal _ _ _, _ => True
  | .arbitraryCode y _ _, _ => ¬ cfg.standard y
  | .arbitraryEvent z _ _, _ => ¬ cfg.standard z
  | .l1Relay w, s =>
      s.withdrawals w ∧
      -- L1CDM._isUnsafeTarget: a messenger never relays to itself.
      (cfg.unsafeTargetCheck = true → w.origin ≠ w.fact.toL1) ∧
      (cfg.lockboxCheck = true → s.lockbox w.fact.toL1 w.origin) ∧
      (cfg.senderCheck = true → w.sender = cfg.trusted) ∧
      cfg.interop w.fact.toL1
  | .fakeCaller _, _ =>
      cfg.realMessengerCheck = false ∨ cfg.lockboxCheck = false ∨ cfg.sysConfigConsistent = false
  | .l1cdmSelfRelay _, _ => cfg.unsafeTargetCheck = false
  | .arbitraryRefund z _, _ => ¬ cfg.standard z
  | .expire f, s =>
      s.deposits f ∧ s.sentAt f.toL1 f.hash ≠ 0 ∧ expiredBy cfg (s.sentAt f.toL1 f.hash) f.time
  | .refund z d b, s =>
      cfg.isBridge b ∧ s.expired z (cfg.msgHash d z b) ∧ ¬ s.refunded z (cfg.msgHash d z b)

/-- Effect of each action. -/
def next (cfg : Config Chain Body Hash) : Action Chain Body Hash → State Chain Hash →
    State Chain Hash
  | .tick y t, s => { s with clock := upd1 s.clock y t }
  | .upgrade y, s => { s with upgraded := upd1 s.upgraded y true }
  | .join z y, s => { s with lockbox := fun a c => s.lockbox a c ∨ (a = z ∧ c = y) }
  | .send z d b, s =>
      { s with
        sentAt := if s.upgraded z then upd2 s.sentAt z (cfg.msgHash d z b) (s.clock z) else s.sentAt
        events := fun c h e => s.events c h e ∨ (c = z ∧ h = cfg.msgHash d z b ∧ e = s.clock z) }
  | .resend z d b, s =>
      { s with
        sentAt := if cfg.resendRestarts && s.upgraded z then
            upd2 s.sentAt z (cfg.msgHash d z b) (s.clock z) else s.sentAt
        events := fun c h e => s.events c h e ∨ (c = z ∧ h = cfg.msgHash d z b ∧ e = s.clock z) }
  | .resendLegacy z d b, s =>
      { s with events := fun c h e => s.events c h e ∨ (c = z ∧ h = cfg.msgHash d z b ∧ e = s.clock z) }
  | .relay x z b, s =>
      { s with
        relayed := fun c h => s.relayed c h ∨ (c = x ∧ h = cfg.msgHash x z b)
        withdrawals := fun w => s.withdrawals w ∨ callOut x (cfg.decode b) w }
  | .exportUndelivered y z b route, s =>
      { s with withdrawals := fun w =>
          s.withdrawals w ∨ w = ⟨y, cfg.trusted, ⟨route, cfg.msgHash y z b, s.clock y⟩⟩ }
  | .userWithdrawal y a f, s =>
      { s with withdrawals := fun w => s.withdrawals w ∨ w = ⟨y, .user a, f⟩ }
  | .arbitraryCode y snd f, s =>
      { s with withdrawals := fun w => s.withdrawals w ∨ w = ⟨y, snd, f⟩ }
  | .arbitraryEvent z h t, s =>
      { s with events := fun c h' e => s.events c h' e ∨ (c = z ∧ h' = h ∧ e = t) }
  | .l1Relay w, s => { s with deposits := fun f => s.deposits f ∨ f = w.fact }
  | .fakeCaller f, s => { s with deposits := fun f' => s.deposits f' ∨ f' = f }
  | .l1cdmSelfRelay f, s => { s with deposits := fun f' => s.deposits f' ∨ f' = f }
  | .arbitraryRefund z h, s =>
      { s with
        refunded := fun c h' => s.refunded c h' ∨ (c = z ∧ h' = h)
        refunds := upd2 s.refunds z h (s.refunds z h + 1) }
  | .expire f, s => { s with expired := fun c h => s.expired c h ∨ (c = f.toL1 ∧ h = f.hash) }
  | .refund z d b, s =>
      { s with
        refunded := fun c h => s.refunded c h ∨ (c = z ∧ h = cfg.msgHash d z b)
        refunds := upd2 s.refunds z (cfg.msgHash d z b) (s.refunds z (cfg.msgHash d z b) + 1) }

/-- One labelled transition. -/
def Step (cfg : Config Chain Body Hash) (a : Action Chain Body Hash) (s s' : State Chain Hash) :
    Prop :=
  guard cfg a s ∧ s' = next cfg a s

/-- Executions of any finite length. -/
inductive Reach (cfg : Config Chain Body Hash) : State Chain Hash → State Chain Hash → Prop
  | refl (s : State Chain Hash) : Reach cfg s s
  | tail {s s₁ s₂ : State Chain Hash} (a : Action Chain Body Hash) :
      Reach cfg s s₁ → Step cfg a s₁ s₂ → Reach cfg s s₂

/-- Genesis of all chains: nothing sent, relayed or withdrawn yet, and no chain upgraded: the
exporter address holds the genesis Proxy with no implementation, so every call to it reverts.
Clocks and initial lockbox memberships are arbitrary. For a standard chain, `upgrade` is the only
action that gives the exporter an implementation; that no ProxyAdmin action did so earlier is the
deployment assumption "historical inertness" (README). -/
structure Init (s : State Chain Hash) : Prop where
  not_upgraded : ∀ y, s.upgraded y = false
  sentAt_zero : ∀ z h, s.sentAt z h = 0
  events_empty : ∀ z h e, ¬ s.events z h e
  relayed_empty : ∀ x h, ¬ s.relayed x h
  withdrawals_empty : ∀ w, ¬ s.withdrawals w
  deposits_empty : ∀ f, ¬ s.deposits f
  expired_none : ∀ z h, ¬ s.expired z h
  refunded_none : ∀ z h, ¬ s.refunded z h
  refunds_zero : ∀ z h, s.refunds z h = 0

/-- Governance at genesis: only standard chains are authorized in any lockbox. -/
def GovInit (cfg : Config Chain Body Hash) (s : State Chain Hash) : Prop :=
  ∀ z y, s.lockbox z y → cfg.standard y

/-- Idealized hash: injective on L2-to-L2 message preimages (destination, source, rest). This is
an idealization of keccak256 (see README); integrity of L1 replay envelopes is a separate external
obligation. -/
def HashInjective (hash : Nat → Nat → Body → Hash) : Prop :=
  ∀ d z b d' z' b', hash d z b = hash d' z' b' → d = d' ∧ z = z' ∧ b = b'

/-- Governance: every standard chain (every chain that is or can become a lockbox member, and every
protected source) has a chain ID no other chain uses (OPCM checks duplicate IDs on migration). -/
def ChainIdUnique (cfg : Config Chain Body Hash) : Prop :=
  ∀ c c', cfg.standard c → cfg.chainId c = cfg.chainId c' → c = c'

/-- The v2 design and its safe variants. The target rule is not required. -/
structure SafeConfig (cfg : Config Chain Body Hash) : Prop where
  trusted : cfg.trusted = .exporter
  realMessengerCheck : cfg.realMessengerCheck = true
  lockboxCheck : cfg.lockboxCheck = true
  senderCheck : cfg.senderCheck = true
  govCheck : cfg.govCheck = true
  unsafeTargetCheck : cfg.unsafeTargetCheck = true
  sysConfigConsistent : cfg.sysConfigConsistent = true
  /-- W_d < P with `≥`, W_d ≤ P with the strict check. -/
  window : ∀ d, cfg.protocolWindow d + (if cfg.expireGe then 1 else 0) ≤ cfg.contractPeriod
  resend : cfg.resend = false ∨ cfg.resendRestarts = true

/-! ## Properties (names match the Quint model) -/

/-- ETH is never both delivered on the destination d and refunded on the (standard) source z. -/
def NoDoubleSpend (cfg : Config Chain Body Hash) (s : State Chain Hash) : Prop :=
  ∀ d z b, cfg.standard z → ¬ (s.relayed d (cfg.msgHash d z b) ∧ s.refunded z (cfg.msgHash d z b))

/-- On standard chains (a non-standard chain's bridge can do anything: `arbitraryRefund`). -/
def RefundImpliesExpired (cfg : Config Chain Body Hash) (s : State Chain Hash) : Prop :=
  ∀ z h, cfg.standard z → s.refunded z h → s.expired z h

def AtMostOneRefund (cfg : Config Chain Body Hash) (s : State Chain Hash) : Prop :=
  ∀ z h, cfg.standard z → s.refunds z h ≤ 1

/-- Fact `f` was produced by an `exportUndelivered y z b f.toL1` step on chain y earlier in the
execution from s₀ to s, with hash (y, z, b) and y's clock at that step. -/
def ExportedBy (cfg : Config Chain Body Hash) (s₀ s : State Chain Hash) (y : Chain)
    (f : Fact Chain Hash) : Prop :=
  ∃ z b s₁ s₂, Reach cfg s₀ s₁ ∧ Step cfg (.exportUndelivered y z b f.toL1) s₁ s₂ ∧
    Reach cfg s₂ s ∧ f.hash = cfg.msgHash y z b ∧ f.time = s₁.clock y

/-- Every deposit A's L2 messenger can act on came from a standard chain's exporter. -/
def NoForgedFact (cfg : Config Chain Body Hash) (s₀ s : State Chain Hash) : Prop :=
  ∀ f, s.deposits f → ∃ y, cfg.standard y ∧ ExportedBy cfg s₀ s y f

end Expiry
