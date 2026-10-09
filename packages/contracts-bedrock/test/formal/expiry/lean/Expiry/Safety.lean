import Expiry.Invariant

/-!
Main theorems. They hold over every execution of any length from genesis, for any number of
chains and messages, any decode of message bodies (attacker-chosen targets and calldata), and
any lockbox joins, under explicit hypotheses:

* `hc   : SafeConfig cfg`         -- the v2 design: exporter trusted, the three L1 checks,
                                     L1CDM self-target rule, SystemConfig consistency, the
                                     governance join rule, member governance keeps the
                                     standard exporter, ∀ d, W_d ≤ P (W_d < P with `≥`),
                                     no non-restarting resend
* `hinj : HashInjective cfg.hash` -- idealized collision-free hash on (dest ID, source ID, rest)
* `hid  : ChainIdUnique cfg`      -- standard chains have chain IDs no other chain uses
* `h0   : Init s₀`                -- genesis: nothing sent/relayed/withdrawn, nothing upgraded
* `hg   : GovInit cfg s₀`         -- only standard chains are authorized at genesis
* `hr   : Reach cfg s₀ s`

`exporterSilentBeforeUpgrade` needs only `Init`, `Reach` and `exporterGovernance = true`: it holds in
every configuration where member governance keeps the standard exporter.
-/
namespace Expiry

set_option linter.unusedSectionVars false

variable {Chain Body Hash : Type} [DecidableEq Chain] [DecidableEq Hash]
variable {cfg : Config Chain Body Hash}

/-! ### Activation is derived, not assumed -/

/-- In every configuration whose member governance keeps the standard exporter
(`exporterGovernance`): each withdrawal whose recorded sender is the exporter, from a
standard chain, was made by that chain's `exportUndelivered` step, earlier in this execution, at a
state where the chain was already upgraded and the message was unrelayed there. So, within this
action model, the exporter never spoke before its chain's upgrade, and nothing else can make it
speak. It is informative for the design (`cfg.trusted = .exporter`); with another trusted sender
`exportUndelivered` records that sender, and the statement is vacuous. It does not establish real
deployment history: that is the "historical inertness" assumption (README). -/
theorem exporterSilentBeforeUpgrade {s₀ s : State Chain Hash} (h0 : Init s₀)
    (hgov : cfg.exporterGovernance = true) (hr : Reach cfg s₀ s) (w : Withdrawal Chain Hash) (hw : s.withdrawals w)
    (hsnd : w.sender = .exporter) (hstd : cfg.standard w.origin) :
    ∃ z b s₁ s₂, Reach cfg s₀ s₁ ∧ Step cfg (.exportUndelivered w.origin z b w.fact.toL1) s₁ s₂ ∧
      Reach cfg s₂ s ∧ s₁.upgraded w.origin = true ∧
      ¬ s₁.relayed w.origin (cfg.msgHash w.origin z b) ∧
      w.fact.hash = cfg.msgHash w.origin z b ∧ w.fact.time = s₁.clock w.origin := by
  obtain ⟨z, b, s₁, s₂, h1, h2, h3, h4, h5⟩ := (histW_reach h0 hgov hr).wd w hw hsnd hstd
  exact ⟨z, b, s₁, s₂, h1, h2, h3, h2.1.2.1, h2.1.2.2, h4, h5⟩

/-- Lockbox membership changes need no history re-check: when a standard chain y is authorized
(at any time, in any configuration with `exporterGovernance`), every exporter withdrawal y ever made is an honest export
made after y's own upgrade. -/
theorem joinNeedsNoHistoryCheck {s₀ s s' : State Chain Hash} (h0 : Init s₀)
    (hgov : cfg.exporterGovernance = true) (hr : Reach cfg s₀ s) {z y : Chain} (hj : Step cfg (.join z y) s s') (hy : cfg.standard y)
    (w : Withdrawal Chain Hash) (hw : s'.withdrawals w) (ho : w.origin = y)
    (hsnd : w.sender = .exporter) :
    ∃ z' b s₁ s₂, Reach cfg s₀ s₁ ∧ Step cfg (.exportUndelivered y z' b w.fact.toL1) s₁ s₂ ∧
      Reach cfg s₂ s' ∧ s₁.upgraded y = true ∧ w.fact.hash = cfg.msgHash y z' b := by
  subst ho
  obtain ⟨z', b, s₁, s₂, h1, h2, h3, h4, _, h6, _⟩ :=
    exporterSilentBeforeUpgrade h0 hgov (Reach.tail _ hr hj) w hw hsnd hy
  exact ⟨z', b, s₁, s₂, h1, h2, h3, h4, h6⟩

/-! ### Safety -/

/-- Core lemma: a message expired on a standard source z has a deposit fact exported by its own
destination d, which is standard, on which the message is not relayed. -/
theorem expired_core (hinj : HashInjective cfg.hash) (hid : ChainIdUnique cfg) {s : State Chain Hash} (hI : Inv cfg s)
    {d z : Chain} {b : Body} (hz : cfg.standard z) (he : s.expired z (cfg.msgHash d z b)) :
    ∃ f, s.deposits f ∧ f.toL1 = z ∧ f.hash = cfg.msgHash d z b ∧ cfg.standard d ∧
      s.sentAt z (cfg.msgHash d z b) ≠ 0 ∧
      expiredBy cfg z (s.sentAt z (cfg.msgHash d z b)) f.time ∧
      f.time ≤ s.clock d ∧ ¬ s.relayed d (cfg.msgHash d z b) := by
  obtain ⟨f, hf, hto, hfh, hsz, hlt⟩ := hI.exp _ _ he
  obtain ⟨y, hy, z₀, b₀, hh, ht, hok⟩ := hI.dep f hf
  obtain ⟨rfl, rfl, rfl⟩ := msgHash_inj hinj hid (hh.symm.trans hfh) (Or.inl hy) (Or.inr hz)
  refine ⟨f, hf, hto, hfh, hy, hsz, hlt, ht, ?_⟩
  have := hok hz (hfh ▸ hsz) (hfh ▸ hlt)
  rw [hfh] at this
  exact this

/-- NoDoubleSpend. -/
theorem noDoubleSpend (hc : SafeConfig cfg) (hinj : HashInjective cfg.hash) (hid : ChainIdUnique cfg)
    {s₀ s : State Chain Hash} (h0 : Init s₀) (hg : GovInit cfg s₀) (hr : Reach cfg s₀ s) :
    NoDoubleSpend cfg s := by
  intro d z b hz ⟨hrel, href⟩
  have hI := (inv_reach hc hinj hid h0 hg hr).1
  obtain ⟨_, _, _, _, _, _, _, _, hnr⟩ := expired_core hinj hid hI hz (hI.ref _ _ hz href)
  exact hnr hrel

theorem refundImpliesExpired (hc : SafeConfig cfg) (hinj : HashInjective cfg.hash) (hid : ChainIdUnique cfg)
    {s₀ s : State Chain Hash} (h0 : Init s₀) (hg : GovInit cfg s₀) (hr : Reach cfg s₀ s) :
    RefundImpliesExpired cfg s :=
  (inv_reach hc hinj hid h0 hg hr).1.ref

theorem atMostOneRefund (hc : SafeConfig cfg) (hinj : HashInjective cfg.hash) (hid : ChainIdUnique cfg)
    {s₀ s : State Chain Hash} (h0 : Init s₀) (hg : GovInit cfg s₀) (hr : Reach cfg s₀ s) :
    AtMostOneRefund cfg s :=
  (inv_reach hc hinj hid h0 hg hr).1.refunds_le

/-- NoForgedFact: every expireMessage deposit was exported by a standard chain's exporter. -/
theorem noForgedFact (hc : SafeConfig cfg) (hinj : HashInjective cfg.hash) (hid : ChainIdUnique cfg)
    {s₀ s : State Chain Hash} (h0 : Init s₀) (hg : GovInit cfg s₀) (hr : Reach cfg s₀ s) :
    NoForgedFact cfg s₀ s :=
  (inv_reach hc hinj hid h0 hg hr).2.2

/-- Current-state part of ExpiredImpliesNeverRelayable. -/
theorem expired_not_relayable (hc : SafeConfig cfg) (hinj : HashInjective cfg.hash) (hid : ChainIdUnique cfg)
    {s : State Chain Hash} (hI : Inv cfg s) {d z : Chain} {b : Body} (hz : cfg.standard z)
    (he : s.expired z (cfg.msgHash d z b)) :
    ¬ s.relayed d (cfg.msgHash d z b) ∧
    ∀ t, s.clock d ≤ t → ¬ withinWindow cfg s d z (cfg.msgHash d z b) t := by
  obtain ⟨f, _, _, _, _, hsz, hlt, ht, hnr⟩ := expired_core hinj hid hI hz he
  refine ⟨hnr, ?_⟩
  intro t hle ⟨e, hev, hwin⟩
  have := expiredBy_window (hc.window z d) hlt
  rcases hI.ev_le _ _ e hev hz with h0 | h0
  · exact hsz h0
  · omega

/-- ExpiredImpliesNeverRelayable: a message expired on a standard source is not relayed on its
destination, no initiating event is within the destination's window at any destination time from
now on, and the same holds in every state of every extension of the execution. -/
theorem expiredImpliesNeverRelayable (hc : SafeConfig cfg) (hinj : HashInjective cfg.hash) (hid : ChainIdUnique cfg)
    {s₀ s : State Chain Hash} (h0 : Init s₀) (hg : GovInit cfg s₀) (hr : Reach cfg s₀ s)
    (d z : Chain) (b : Body) (hz : cfg.standard z) (he : s.expired z (cfg.msgHash d z b)) :
    ¬ s.relayed d (cfg.msgHash d z b) ∧
    (∀ t, s.clock d ≤ t → ¬ withinWindow cfg s d z (cfg.msgHash d z b) t) ∧
    (∀ s', Reach cfg s s' →
      s'.expired z (cfg.msgHash d z b) ∧ ¬ s'.relayed d (cfg.msgHash d z b) ∧
      ∀ t, s'.clock d ≤ t → ¬ withinWindow cfg s' d z (cfg.msgHash d z b) t) := by
  have hI := (inv_reach hc hinj hid h0 hg hr).1
  obtain ⟨h1, h2⟩ := expired_not_relayable hc hinj hid hI hz he
  refine ⟨h1, h2, fun s' hr' => ?_⟩
  have hI' := (inv_reach hc hinj hid h0 hg (Reach.trans hr hr')).1
  have he' := expired_reach hr' he
  exact ⟨he', expired_not_relayable hc hinj hid hI' hz he'⟩

/-- Corollary: no relay step of an expired message is ever enabled in any extension. -/
theorem expired_no_relay_step (hc : SafeConfig cfg) (hinj : HashInjective cfg.hash) (hid : ChainIdUnique cfg)
    {s₀ s : State Chain Hash} (h0 : Init s₀) (hg : GovInit cfg s₀) (hr : Reach cfg s₀ s)
    (d z : Chain) (b : Body) (hz : cfg.standard z) (he : s.expired z (cfg.msgHash d z b))
    (s' s'' : State Chain Hash) (hr' : Reach cfg s s')
    (hstep : Step cfg (.relay d z b) s' s'') : False := by
  obtain ⟨_, hnr, _⟩ :=
    (expiredImpliesNeverRelayable hc hinj hid h0 hg hr d z b hz he).2.2 s'' (Reach.tail _ hr' hstep)
  obtain ⟨_, rfl⟩ := hstep
  exact hnr (Or.inr ⟨rfl, rfl⟩)

/-- OnlyDestinationCanExport: a message expired on a standard source z was expired by a deposit
that its own destination d exported, routed to z, by an `exportUndelivered d z b z` step earlier in
this execution, at d's time `f.time`, while the message was unrelayed on d. -/
theorem onlyDestinationCanExport (hc : SafeConfig cfg) (hinj : HashInjective cfg.hash) (hid : ChainIdUnique cfg)
    {s₀ s : State Chain Hash} (h0 : Init s₀) (hg : GovInit cfg s₀) (hr : Reach cfg s₀ s)
    (d z : Chain) (b : Body) (hz : cfg.standard z) (he : s.expired z (cfg.msgHash d z b)) :
    ∃ f, s.deposits f ∧ f.toL1 = z ∧ f.hash = cfg.msgHash d z b ∧
      expiredBy cfg z (s.sentAt z (cfg.msgHash d z b)) f.time ∧
      ∃ s₁ s₂, Reach cfg s₀ s₁ ∧ Step cfg (.exportUndelivered d z b z) s₁ s₂ ∧ Reach cfg s₂ s ∧
        s₁.upgraded d = true ∧ s₁.clock d = f.time ∧ ¬ s₁.relayed d (cfg.msgHash d z b) := by
  obtain ⟨hI, _, hD⟩ := inv_reach hc hinj hid h0 hg hr
  obtain ⟨f, hf, hto, hfh, _, _, hlt, _, _⟩ := expired_core hinj hid hI hz he
  obtain ⟨y, hy, z₁, b₁, s₁, s₂, h1, h2, h3, h4, h5⟩ := hD f hf
  obtain ⟨rfl, rfl, rfl⟩ := msgHash_inj hinj hid (h4.symm.trans hfh) (Or.inl hy) (Or.inr hz)
  rw [hto] at h2
  exact ⟨f, hf, hto, hfh, hlt, s₁, s₂, h1, h2, h3, h2.1.2.1, h5.symm, h2.1.2.2⟩

/-- All safety properties together (the Quint `Safety`). -/
theorem safety (hc : SafeConfig cfg) (hinj : HashInjective cfg.hash) (hid : ChainIdUnique cfg)
    {s₀ s : State Chain Hash} (h0 : Init s₀) (hg : GovInit cfg s₀) (hr : Reach cfg s₀ s) :
    NoDoubleSpend cfg s ∧ RefundImpliesExpired cfg s ∧ AtMostOneRefund cfg s ∧
    NoForgedFact cfg s₀ s ∧
    (∀ d z b, cfg.standard z → s.expired z (cfg.msgHash d z b) →
      ¬ s.relayed d (cfg.msgHash d z b) ∧
      ∀ t, s.clock d ≤ t → ¬ withinWindow cfg s d z (cfg.msgHash d z b) t) :=
  ⟨noDoubleSpend hc hinj hid h0 hg hr, refundImpliesExpired hc hinj hid h0 hg hr,
   atMostOneRefund hc hinj hid h0 hg hr, noForgedFact hc hinj hid h0 hg hr,
   fun _ _ _ hz he => expired_not_relayable hc hinj hid (inv_reach hc hinj hid h0 hg hr).1 hz he⟩

/-- Safety does not depend on the messenger's unsafe-target rule (kept as defense in depth): the
same conclusions hold for a configuration without it. -/
theorem safety_without_targetRule (hc : SafeConfig cfg) (_htr : cfg.targetRule = false)
    (hinj : HashInjective cfg.hash) (hid : ChainIdUnique cfg)
    {s₀ s : State Chain Hash} (h0 : Init s₀) (hg : GovInit cfg s₀) (hr : Reach cfg s₀ s) :
    NoDoubleSpend cfg s ∧ RefundImpliesExpired cfg s ∧ AtMostOneRefund cfg s ∧
    NoForgedFact cfg s₀ s ∧
    (∀ d z b, cfg.standard z → s.expired z (cfg.msgHash d z b) →
      ¬ s.relayed d (cfg.msgHash d z b) ∧
      ∀ t, s.clock d ≤ t → ¬ withinWindow cfg s d z (cfg.msgHash d z b) t) :=
  safety hc hinj hid h0 hg hr

/-- PauseOnlyDelays: pausing a chain only delays the facts routed to it. An L1 relay of a
withdrawal that is enabled in some state is enabled again in every later state of the execution
in which the receiving chain is not paused (in any configuration): the withdrawal stays (a relay
reverted by the pause lands in the caller's `failedMessages` and can be replayed), lockbox
authorizations only grow, and the fact's content, its time included, is fixed at export. -/
theorem pauseOnlyDelays {s s' : State Chain Hash} (hr : Reach cfg s s') (w : Withdrawal Chain Hash)
    (hg : guard cfg (.l1Relay w) s) (hu : s'.paused w.fact.toL1 = false) :
    guard cfg (.l1Relay w) s' := by
  simp only [guard] at hg ⊢
  obtain ⟨hw, ht, hl, hsnd, hi, _⟩ := hg
  exact ⟨withdrawals_reach hr hw, ht, fun h => lockbox_reach hr (hl h), hsnd, hi, hu⟩

/-- MessengerSilentAfterUpgrade (defense in depth, the unsafe-target rule's own property): with
the rule, no step creates a withdrawal whose recorded sender is the L2ToL2CrossDomainMessenger
from an upgraded standard chain. (In the earlier design the export function lived in the
messenger and recorded it as sender, hence `cfg.trusted ≠ .messenger`.) The
L2ToL1MessagePasser path is checked on bytecode by Halmos/Kontrol, not here. -/
theorem messengerSilentAfterUpgrade (htr : cfg.targetRule = true)
    (htrust : cfg.trusted ≠ .messenger) {a : Action Chain Body Hash} {s s' : State Chain Hash}
    (hs : Step cfg a s s') (w : Withdrawal Chain Hash) (hnew : s'.withdrawals w)
    (hold : ¬ s.withdrawals w) (hsnd : w.sender = .messenger) (hstd : cfg.standard w.origin) :
    s.upgraded w.origin = false := by
  obtain ⟨hg, rfl⟩ := hs
  cases a with
  | relay x z b =>
    rcases hnew with hnew | hnew
    · exact (hold hnew).elim
    · simp only [guard] at hg
      obtain ⟨_, _, hrule⟩ := hg
      cases hd : cfg.decode b with
      | l2cdm f =>
        rw [hd] at hnew hrule
        simp only [callOut] at hnew
        subst hnew
        cases hu : s.upgraded x
        · rfl
        · exact (hrule hstd hu htr trivial).elim
      | other => rw [hd] at hnew; exact hnew.elim
  | exportUndelivered y z b route =>
    rcases hnew with hnew | rfl
    · exact (hold hnew).elim
    · exact (htrust hsnd).elim
  | userWithdrawal y a f =>
    rcases hnew with hnew | rfl
    · exact (hold hnew).elim
    · cases hsnd
  | arbitraryCode y snd f =>
    rcases hnew with hnew | rfl
    · exact (hold hnew).elim
    · exact (hg hstd).elim
  | exporterGovernanceUpgrade y f =>
    rcases hnew with hnew | rfl
    · exact (hold hnew).elim
    · cases hsnd
  | _ => exact (hold hnew).elim

end Expiry
