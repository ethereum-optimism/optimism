import Expiry.Counterexamples

/-!
# Non-vacuity witnesses, one per headline theorem

For every headline theorem `T` of `Safety.lean` (the list in `Axioms.lean`), `nonvacuous_T`
exhibits one concrete instance (`Examples.base`, genesis `Examples.s0`, explicit executions) in
which **all** of `T`'s hypotheses hold **jointly**, and then applies `T` to that instance, so `T`'s
conclusion is instantiated on a state where its antecedents are actually met (a refund happened, a
message is expired, an exporter withdrawal exists, ...). Where the conclusion is a negative
statement, the witness also shows that the forbidden event is otherwise reachable (e.g. a relay of
the same message in another execution), so the theorem is not true merely because the event can
never happen.

`expired_no_relay_step` concludes `False`: its hypotheses are jointly unsatisfiable by design (that
is the theorem). Its witness shows that every hypothesis except the relay step holds jointly, that
the relay step alone is enabled in another reachable state, and applies the theorem.

`Axioms.lean` fails the build if a headline theorem has no `nonvacuous_` witness, if the witness
does not apply the theorem, or if the witness uses an axiom beyond `propext`, `Classical.choice`,
`Quot.sound`. Every witness here is checked by the kernel (`simp`/`decide`; no `native_decide`).
-/

namespace Expiry

open Expiry.Examples

set_option linter.unusedSimpArgs false

attribute [local simp] Call.isL2CDM Config.msgHash

namespace NV

/-! ### Witness executions -/

/-- The refund execution of `Examples.refund_reachable`: A sends body 5 to B, B exports it at time
10, the fact is relayed on L1, A expires the message and refunds it. -/
def sR : State Nat H := run base refundTrace s0

theorem sR_reach : Reach base s0 sR :=
  reach_run _ _ _ (by simp [Valid, refundTrace, guard, next, base, s0, upd1, upd2, expiredBy, fB, mAB])

theorem sR_refunded : sR.refunded 0 mAB := by
  simp [sR, run, refundTrace, next, base, s0, upd1, upd2, fB, mAB]

theorem sR_expired : sR.expired 0 mAB := by
  simp [sR, run, refundTrace, next, base, s0, upd1, upd2, fB, mAB]

theorem sR_refunds : sR.refunds 0 mAB = 1 := by
  simp [sR, run, refundTrace, next, base, s0, upd1, upd2, fB, mAB]

theorem sR_deposit : sR.deposits fB := by
  simp [sR, run, refundTrace, next, base, s0, upd1, upd2, fB, mAB]

theorem sR_withdrawal : sR.withdrawals ⟨1, .exporter, fB⟩ := by
  simp [sR, run, refundTrace, next, base, s0, upd1, upd2, fB, mAB]

/-- A relay of the same message `mAB` on B (the event `NoDoubleSpend` pairs with the refund). -/
theorem relay_mAB_reachable : ∃ s, Reach base s0 s ∧ s.relayed 1 mAB := by
  obtain ⟨s, hr, hrel, _⟩ := relay_reachable_at_edge
  exact ⟨s, hr, hrel⟩

/-- `joinNeedsNoHistoryCheck`: C (standard) is upgraded and exports a fact for A's message to C,
and only then is C authorized in A's lockbox (`join 0 2`). -/
def joinPre : List (Action Nat Nat H) :=
  [.upgrade 0, .upgrade 2, .send 0 2 6, .tick 2 10, .exportUndelivered 2 0 6 0]

def sJ : State Nat H := run base joinPre s0

theorem sJ_reach : Reach base s0 sJ :=
  reach_run _ _ _ (by simp [Valid, joinPre, guard, next, base, s0, upd1, upd2])

/-- `messengerSilentAfterUpgrade`: before B's upgrade, B relays C's attacker message (body 9, target
the L2CrossDomainMessenger), so B's 0x..23 sends a withdrawal. -/
def sM : State Nat H := run base [.send 2 1 9] s0

theorem sM_reach : Reach base s0 sM :=
  reach_run _ _ _ (by simp [Valid, guard, next, base, s0])

/-- The safe configuration without the messenger's target rule (`safety_without_targetRule`). -/
theorem cfgNoTargetRule_safe : SafeConfig cfgNoTargetRule :=
  ⟨rfl, rfl, rfl, rfl, rfl, rfl, rfl, rfl, fun _ => show 7 + 0 ≤ 8 by decide, Or.inl rfl⟩

def sRn : State Nat H := run cfgNoTargetRule refundTrace s0

theorem sRn_reach : Reach cfgNoTargetRule s0 sRn :=
  reach_run _ _ _ (by
    simp [Valid, refundTrace, cfgNoTargetRule, guard, next, base, s0, upd1, upd2, expiredBy, fB, mAB])

theorem sRn_refunded : sRn.refunded 0 mAB := by
  simp [sRn, run, refundTrace, cfgNoTargetRule, next, base, s0, upd1, upd2, fB, mAB]

end NV

open NV

/-! ### Activation -/

/-- All hypotheses of `exporterSilentBeforeUpgrade` hold jointly (an exporter withdrawal from the
standard chain B exists after the refund execution), and the theorem yields the export step. -/
theorem nonvacuous_exporterSilentBeforeUpgrade :
    let w : Withdrawal Nat H := ⟨1, .exporter, fB⟩
    Init s0 ∧ base.exporterGovernance = true ∧ Reach base s0 sR ∧ sR.withdrawals w ∧
    w.sender = .exporter ∧ base.standard w.origin ∧
    ∃ z b s₁ s₂, Reach base s0 s₁ ∧ Step base (.exportUndelivered w.origin z b w.fact.toL1) s₁ s₂ ∧
      Reach base s₂ sR ∧ s₁.upgraded w.origin = true ∧
      ¬ s₁.relayed w.origin (base.msgHash w.origin z b) ∧
      w.fact.hash = base.msgHash w.origin z b ∧ w.fact.time = s₁.clock w.origin := by
  intro w
  have hstd : base.standard w.origin := show (1 : Nat) ≤ 2 by decide
  exact ⟨s0_init, rfl, sR_reach, sR_withdrawal, rfl, hstd,
    exporterSilentBeforeUpgrade s0_init rfl sR_reach w sR_withdrawal rfl hstd⟩

/-- All hypotheses of `joinNeedsNoHistoryCheck` hold jointly: C exported before it joins A's
lockbox, and the join step is enabled; the theorem yields C's honest export. -/
theorem nonvacuous_joinNeedsNoHistoryCheck :
    let s' := next base (.join 0 2) sJ
    let w : Withdrawal Nat H := ⟨2, .exporter, fC⟩
    Init s0 ∧ base.exporterGovernance = true ∧ Reach base s0 sJ ∧ Step base (.join 0 2) sJ s' ∧
    base.standard 2 ∧ s'.withdrawals w ∧ w.origin = 2 ∧ w.sender = .exporter ∧
    ∃ z' b s₁ s₂, Reach base s0 s₁ ∧ Step base (.exportUndelivered 2 z' b w.fact.toL1) s₁ s₂ ∧
      Reach base s₂ s' ∧ s₁.upgraded 2 = true ∧ w.fact.hash = base.msgHash 2 z' b := by
  intro s' w
  have hj : Step base (.join 0 2) sJ s' := ⟨fun _ => show (2 : Nat) ≤ 2 by decide, rfl⟩
  have hy : base.standard 2 := show (2 : Nat) ≤ 2 by decide
  have hw : s'.withdrawals w := by
    simp [s', w, sJ, joinPre, run, next, base, s0, upd1, upd2, fC]
  exact ⟨s0_init, rfl, sJ_reach, hj, hy, hw, rfl, rfl,
    joinNeedsNoHistoryCheck s0_init rfl sJ_reach hj hy w hw rfl rfl⟩

/-! ### Safety -/

/-- All hypotheses of `noDoubleSpend` hold jointly on the refund execution; the refund half of the
forbidden pair happens there, the relay half is reachable in another execution, and the theorem
rules out the relay in the refund state. -/
theorem nonvacuous_noDoubleSpend :
    SafeConfig base ∧ HashInjective base.hash ∧ ChainIdUnique base ∧ Init s0 ∧ GovInit base s0 ∧
    Reach base s0 sR ∧ sR.refunded 0 (base.msgHash 1 0 5) ∧
    (∃ s, Reach base s0 s ∧ s.relayed 1 (base.msgHash 1 0 5)) ∧
    ¬ sR.relayed 1 (base.msgHash 1 0 5) := by
  refine ⟨base_safe, inj_id, base_idu, s0_init, s0_gov _ rfl, sR_reach, sR_refunded,
    relay_mAB_reachable, fun hrel => ?_⟩
  exact noDoubleSpend base_safe inj_id base_idu s0_init (s0_gov _ rfl) sR_reach 1 0 5
    (show (0 : Nat) ≤ 2 by decide) ⟨hrel, sR_refunded⟩

/-- `refundImpliesExpired`, applied where a refund happened. -/
theorem nonvacuous_refundImpliesExpired :
    SafeConfig base ∧ HashInjective base.hash ∧ ChainIdUnique base ∧ Init s0 ∧ GovInit base s0 ∧
    Reach base s0 sR ∧ sR.refunded 0 mAB ∧ sR.expired 0 mAB := by
  exact ⟨base_safe, inj_id, base_idu, s0_init, s0_gov _ rfl, sR_reach, sR_refunded,
    refundImpliesExpired base_safe inj_id base_idu s0_init (s0_gov _ rfl) sR_reach 0 mAB
      (show (0 : Nat) ≤ 2 by decide) sR_refunded⟩

/-- `atMostOneRefund`, applied where one refund happened (the bound is attained). -/
theorem nonvacuous_atMostOneRefund :
    SafeConfig base ∧ HashInjective base.hash ∧ ChainIdUnique base ∧ Init s0 ∧ GovInit base s0 ∧
    Reach base s0 sR ∧ sR.refunds 0 mAB = 1 ∧ sR.refunds 0 mAB ≤ 1 := by
  exact ⟨base_safe, inj_id, base_idu, s0_init, s0_gov _ rfl, sR_reach, sR_refunds,
    atMostOneRefund base_safe inj_id base_idu s0_init (s0_gov _ rfl) sR_reach 0 mAB
      (show (0 : Nat) ≤ 2 by decide)⟩

/-- `noForgedFact`, applied to an existing deposit: it was exported by a standard chain. -/
theorem nonvacuous_noForgedFact :
    SafeConfig base ∧ HashInjective base.hash ∧ ChainIdUnique base ∧ Init s0 ∧ GovInit base s0 ∧
    Reach base s0 sR ∧ sR.deposits fB ∧ ∃ y, base.standard y ∧ ExportedBy base s0 sR y fB := by
  exact ⟨base_safe, inj_id, base_idu, s0_init, s0_gov _ rfl, sR_reach, sR_deposit,
    noForgedFact base_safe inj_id base_idu s0_init (s0_gov _ rfl) sR_reach fB sR_deposit⟩

/-- `expiredImpliesNeverRelayable`, applied where the message is expired (and to the trivial
extension `sR → sR`). -/
theorem nonvacuous_expiredImpliesNeverRelayable :
    SafeConfig base ∧ HashInjective base.hash ∧ ChainIdUnique base ∧ Init s0 ∧ GovInit base s0 ∧
    Reach base s0 sR ∧ base.standard 0 ∧ sR.expired 0 (base.msgHash 1 0 5) ∧
    ¬ sR.relayed 1 (base.msgHash 1 0 5) ∧
    (∀ t, sR.clock 1 ≤ t → ¬ withinWindow base sR 1 0 (base.msgHash 1 0 5) t) ∧
    (sR.expired 0 (base.msgHash 1 0 5) ∧ ¬ sR.relayed 1 (base.msgHash 1 0 5)) := by
  have hz : base.standard 0 := show (0 : Nat) ≤ 2 by decide
  obtain ⟨h1, h2, h3⟩ :=
    expiredImpliesNeverRelayable base_safe inj_id base_idu s0_init (s0_gov _ rfl) sR_reach 1 0 5 hz
      sR_expired
  obtain ⟨h4, h5, _⟩ := h3 sR (Reach.refl _)
  exact ⟨base_safe, inj_id, base_idu, s0_init, s0_gov _ rfl, sR_reach, hz, sR_expired, h1, h2, h4, h5⟩

/-- `expired_no_relay_step` concludes `False`, so its hypotheses cannot all hold (that is the
theorem). Every hypothesis except the relay step holds jointly (`s' = sR`); the relay step of the
same message is enabled in another reachable state (before expiry); and the theorem, applied,
says no relay step of it is enabled from the expired state. -/
theorem nonvacuous_expired_no_relay_step :
    SafeConfig base ∧ HashInjective base.hash ∧ ChainIdUnique base ∧ Init s0 ∧ GovInit base s0 ∧
    Reach base s0 sR ∧ base.standard 0 ∧ sR.expired 0 (base.msgHash 1 0 5) ∧ Reach base sR sR ∧
    (∃ s s'', Reach base s0 s ∧ Step base (.relay 1 0 5) s s'') ∧
    ¬ ∃ s'', Step base (.relay 1 0 5) sR s'' := by
  have hz : base.standard 0 := show (0 : Nat) ≤ 2 by decide
  refine ⟨base_safe, inj_id, base_idu, s0_init, s0_gov _ rfl, sR_reach, hz, sR_expired,
    Reach.refl _, ?_, fun ⟨s'', hs⟩ =>
      expired_no_relay_step base_safe inj_id base_idu s0_init (s0_gov _ rfl) sR_reach 1 0 5 hz
        sR_expired sR s'' (Reach.refl _) hs⟩
  let pre : List (Action Nat Nat H) := [.upgrade 0, .upgrade 1, .send 0 1 5, .tick 1 8]
  refine ⟨run base pre s0, next base (.relay 1 0 5) (run base pre s0),
    reach_run _ _ _ (by simp [pre, Valid, guard, next, base, s0, upd1, upd2]), ?_, rfl⟩
  simp [pre, run, guard, next, base, s0, upd1, upd2, withinWindow]

/-- `onlyDestinationCanExport`, applied where the message is expired: the export step by B. -/
theorem nonvacuous_onlyDestinationCanExport :
    SafeConfig base ∧ HashInjective base.hash ∧ ChainIdUnique base ∧ Init s0 ∧ GovInit base s0 ∧
    Reach base s0 sR ∧ base.standard 0 ∧ sR.expired 0 (base.msgHash 1 0 5) ∧
    ∃ f, sR.deposits f ∧ f.toL1 = 0 ∧ f.hash = base.msgHash 1 0 5 ∧
      expiredBy base (sR.sentAt 0 (base.msgHash 1 0 5)) f.time ∧
      ∃ s₁ s₂, Reach base s0 s₁ ∧ Step base (.exportUndelivered 1 0 5 0) s₁ s₂ ∧ Reach base s₂ sR ∧
        s₁.upgraded 1 = true ∧ s₁.clock 1 = f.time ∧ ¬ s₁.relayed 1 (base.msgHash 1 0 5) := by
  have hz : base.standard 0 := show (0 : Nat) ≤ 2 by decide
  exact ⟨base_safe, inj_id, base_idu, s0_init, s0_gov _ rfl, sR_reach, hz, sR_expired,
    onlyDestinationCanExport base_safe inj_id base_idu s0_init (s0_gov _ rfl) sR_reach 1 0 5 hz
      sR_expired⟩

/-- `safety`, applied on the refund execution, with its first four conjuncts instantiated at `mAB`
(the fifth is `expiredImpliesNeverRelayable`'s conclusion; see its witness). -/
theorem nonvacuous_safety :
    SafeConfig base ∧ HashInjective base.hash ∧ ChainIdUnique base ∧ Init s0 ∧ GovInit base s0 ∧
    Reach base s0 sR ∧ sR.refunded 0 mAB ∧ sR.deposits fB ∧
    ¬ sR.relayed 1 (base.msgHash 1 0 5) ∧ sR.expired 0 mAB ∧ sR.refunds 0 mAB ≤ 1 ∧
    (∃ y, base.standard y ∧ ExportedBy base s0 sR y fB) := by
  have hz : base.standard 0 := show (0 : Nat) ≤ 2 by decide
  obtain ⟨hnd, hre, hamo, hnf, _⟩ :=
    safety base_safe inj_id base_idu s0_init (s0_gov _ rfl) sR_reach
  exact ⟨base_safe, inj_id, base_idu, s0_init, s0_gov _ rfl, sR_reach, sR_refunded, sR_deposit,
    fun hrel => hnd 1 0 5 hz ⟨hrel, sR_refunded⟩, hre 0 mAB hz sR_refunded, hamo 0 mAB hz,
    hnf fB sR_deposit⟩

/-- `safety_without_targetRule`, on the refund execution of the configuration without the target
rule (all its hypotheses, including `targetRule = false`, hold jointly). -/
theorem nonvacuous_safety_without_targetRule :
    SafeConfig cfgNoTargetRule ∧ cfgNoTargetRule.targetRule = false ∧
    HashInjective cfgNoTargetRule.hash ∧ ChainIdUnique cfgNoTargetRule ∧ Init s0 ∧
    GovInit cfgNoTargetRule s0 ∧ Reach cfgNoTargetRule s0 sRn ∧ sRn.refunded 0 mAB ∧
    ¬ sRn.relayed 1 (cfgNoTargetRule.msgHash 1 0 5) := by
  have hid : ChainIdUnique cfgNoTargetRule := fun _ _ _ h => h
  have hz : cfgNoTargetRule.standard 0 := show (0 : Nat) ≤ 2 by decide
  obtain ⟨hnd, _⟩ := safety_without_targetRule cfgNoTargetRule_safe rfl inj_id hid s0_init
    (s0_gov _ rfl) sRn_reach
  exact ⟨cfgNoTargetRule_safe, rfl, inj_id, hid, s0_init, s0_gov _ rfl, sRn_reach, sRn_refunded,
    fun hrel => hnd 1 0 5 hz ⟨hrel, sRn_refunded⟩⟩

/-- All hypotheses of `messengerSilentAfterUpgrade` hold jointly: a relay step on B creates a new
withdrawal whose sender is 0x..23 (B not yet upgraded); the theorem says B was not upgraded. -/
theorem nonvacuous_messengerSilentAfterUpgrade :
    let s' := next base (.relay 1 2 9) sM
    base.targetRule = true ∧ base.trusted ≠ .messenger ∧ Reach base s0 sM ∧
    Step base (.relay 1 2 9) sM s' ∧ s'.withdrawals wM ∧ ¬ sM.withdrawals wM ∧
    wM.sender = .messenger ∧ base.standard wM.origin ∧ sM.upgraded wM.origin = false := by
  intro s'
  have hs : Step base (.relay 1 2 9) sM s' :=
    ⟨by simp [sM, run, guard, next, base, s0, withinWindow], rfl⟩
  have hnew : s'.withdrawals wM := by simp [s', sM, run, next, base, s0, callOut, wM, fM]
  have hold : ¬ sM.withdrawals wM := by simp [sM, run, next, base, s0]
  have hstd : base.standard wM.origin := show (1 : Nat) ≤ 2 by decide
  exact ⟨rfl, by decide, sM_reach, hs, hnew, hold, rfl, hstd,
    messengerSilentAfterUpgrade rfl (by decide) hs wM hnew hold rfl hstd⟩

end Expiry
