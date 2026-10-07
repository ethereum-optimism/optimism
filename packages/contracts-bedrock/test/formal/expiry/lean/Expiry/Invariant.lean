import Expiry.Model

/-!
Inductive invariants.

* `HistW` (any configuration): every withdrawal with sender `exporter` from a standard chain was
  made by an `exportUndelivered` step of that chain earlier in the execution. Needs only `Init`
  (genesis has no withdrawals).
* `Inv` (safe configuration): the state invariant.
* `HistD` (safe configuration): every deposit fact was exported by a standard chain.
-/
namespace Expiry

set_option linter.unusedSectionVars false

variable {Chain Body Hash : Type} [DecidableEq Chain] [DecidableEq Hash]

/-! ### Map-update lemmas -/

theorem upd2_eq {α β γ : Type} [DecidableEq α] [DecidableEq β] (f : α → β → γ) (a : α) (b : β)
    (v : γ) : upd2 f a b v a b = v := by
  simp [upd2]

theorem upd2_ne {α β γ : Type} [DecidableEq α] [DecidableEq β] (f : α → β → γ) {a a' : α}
    {b b' : β} (v : γ) (h : ¬ (a' = a ∧ b' = b)) : upd2 f a b v a' b' = f a' b' := by
  simp only [upd2]; split
  · contradiction
  · rfl

theorem upd1_ne {α γ : Type} [DecidableEq α] (f : α → γ) {a a' : α} (v : γ) (h : a' ≠ a) :
    upd1 f a v a' = f a' := by
  simp only [upd1]; split
  · contradiction
  · rfl

theorem upd1_eq {α γ : Type} [DecidableEq α] (f : α → γ) (a : α) (v : γ) : upd1 f a v a = v := by
  simp [upd1]

/-! ### expiredBy lemmas -/

theorem expiredBy_mono {cfg : Config Chain Body Hash} {a b t : Nat} (hab : a ≤ b)
    (h : expiredBy cfg b t) : expiredBy cfg a t := by
  cases hge : cfg.expireGe <;> simp [expiredBy, hge] at h ⊢ <;> omega

theorem expiredBy_window {cfg : Config Chain Body Hash} {w a t : Nat}
    (hw : w + (if cfg.expireGe then 1 else 0) ≤ cfg.contractPeriod) (h : expiredBy cfg a t) :
    a + w < t := by
  cases hge : cfg.expireGe <;> simp [expiredBy, hge] at h hw ⊢ <;> omega

/-! ### Definitions -/

/-- A fact exported by chain y is consistent with the current state: its hash has y as the
destination, it is not from y's future, and if it is dated past the period of a message sent by a
standard source, that message is not relayed on y. -/
def Honest (cfg : Config Chain Body Hash) (s : State Chain Hash) (y : Chain) (f : Fact Chain Hash) :
    Prop :=
  ∃ z b, f.hash = cfg.hash y z b ∧ f.time ≤ s.clock y ∧
    (cfg.standard z → s.sentAt z f.hash ≠ 0 →
      expiredBy cfg (s.sentAt z f.hash) f.time → ¬ s.relayed y f.hash)

structure Inv (cfg : Config Chain Body Hash) (s : State Chain Hash) : Prop where
  sent_le : ∀ z h, s.sentAt z h ≤ s.clock z
  sent_up : ∀ z h, s.sentAt z h ≠ 0 → s.upgraded z = true
  ev_le : ∀ z h e, s.events z h e → cfg.standard z → s.sentAt z h = 0 ∨ e ≤ s.sentAt z h
  sent_ev : ∀ z h, s.sentAt z h ≠ 0 → ∃ e, s.events z h e
  rel_ev : ∀ x z b, s.relayed x (cfg.hash x z b) → cfg.standard x →
    ∃ e, s.events z (cfg.hash x z b) e
  gov : ∀ z y, s.lockbox z y → cfg.standard y
  wd : ∀ w, s.withdrawals w → w.sender = .exporter → cfg.standard w.origin →
    Honest cfg s w.origin w.fact
  dep : ∀ f, s.deposits f → ∃ y, cfg.standard y ∧ Honest cfg s y f
  exp : ∀ z h, s.expired z h → ∃ f, s.deposits f ∧ f.toL1 = z ∧ f.hash = h ∧
    s.sentAt z h ≠ 0 ∧ expiredBy cfg (s.sentAt z h) f.time
  ref : ∀ z h, cfg.standard z → s.refunded z h → s.expired z h
  refunds_le : ∀ z h, cfg.standard z → s.refunds z h ≤ 1
  refunds_zero : ∀ z h, cfg.standard z → ¬ s.refunded z h → s.refunds z h = 0

structure HistW (cfg : Config Chain Body Hash) (s₀ s : State Chain Hash) : Prop where
  wd : ∀ w, s.withdrawals w → w.sender = .exporter → cfg.standard w.origin →
    ExportedBy cfg s₀ s w.origin w.fact

/-! ### Reach lemmas -/

theorem Reach.trans {cfg : Config Chain Body Hash} {s₁ s₂ s₃ : State Chain Hash}
    (h₁ : Reach cfg s₁ s₂) (h₂ : Reach cfg s₂ s₃) : Reach cfg s₁ s₃ := by
  induction h₂ with
  | refl => exact h₁
  | tail a _ hs ih => exact Reach.tail a ih hs

theorem Reach.single {cfg : Config Chain Body Hash} {s s' : State Chain Hash}
    {a : Action Chain Body Hash} (h : Step cfg a s s') : Reach cfg s s' :=
  Reach.tail a (Reach.refl s) h

theorem expired_step {cfg : Config Chain Body Hash} {a : Action Chain Body Hash}
    {s s' : State Chain Hash} {z : Chain} {h : Hash} (hs : Step cfg a s s') (he : s.expired z h) :
    s'.expired z h := by
  obtain ⟨_, rfl⟩ := hs
  cases a <;> simp only [next] <;> first | exact he | exact Or.inl he

theorem expired_reach {cfg : Config Chain Body Hash} {s s' : State Chain Hash} {z : Chain}
    {h : Hash} (hr : Reach cfg s s') (he : s.expired z h) : s'.expired z h := by
  induction hr with
  | refl => exact he
  | tail _ _ hs ih => exact expired_step hs ih

theorem exportedBy_step {cfg : Config Chain Body Hash} {s₀ s s' : State Chain Hash}
    {a : Action Chain Body Hash} {y : Chain} {f : Fact Chain Hash}
    (hx : ExportedBy cfg s₀ s y f) (hs : Step cfg a s s') : ExportedBy cfg s₀ s' y f := by
  obtain ⟨z, b, s₁, s₂, h1, h2, h3, h4, h5⟩ := hx
  exact ⟨z, b, s₁, s₂, h1, h2, Reach.tail a h3 hs, h4, h5⟩

theorem callOut_sender {x : Chain} {c : Call Chain Hash} {w : Withdrawal Chain Hash}
    (h : callOut x c w) : w.sender = .messenger := by
  cases c with
  | l2cdm f => simp only [callOut] at h; subst h; rfl
  | other => exact h.elim

/-! ### HistW: holds in every configuration -/

theorem histW_reach {cfg : Config Chain Body Hash} {s₀ s : State Chain Hash} (h0 : Init s₀)
    (hr : Reach cfg s₀ s) : HistW cfg s₀ s := by
  induction hr with
  | refl => exact ⟨fun w hw => (h0.withdrawals_empty w hw).elim⟩
  | @tail s₁ s₂ a hr' hs ih =>
    refine ⟨fun w hw hsnd hstd => ?_⟩
    have hs' := hs
    obtain ⟨hg, rfl⟩ := hs
    cases a with
    | relay x z b =>
      rcases hw with hw | hw
      · exact exportedBy_step (ih.wd w hw hsnd hstd) hs'
      · rw [callOut_sender hw] at hsnd; cases hsnd
    | exportUndelivered y z b route =>
      rcases hw with hw | rfl
      · exact exportedBy_step (ih.wd w hw hsnd hstd) hs'
      · exact ⟨z, b, s₁, _, hr', hs', Reach.refl _, rfl, rfl⟩
    | userWithdrawal y a f =>
      rcases hw with hw | rfl
      · exact exportedBy_step (ih.wd w hw hsnd hstd) hs'
      · cases hsnd
    | arbitraryCode y snd f =>
      rcases hw with hw | rfl
      · exact exportedBy_step (ih.wd w hw hsnd hstd) hs'
      · exact (hg hstd).elim
    | _ => exact exportedBy_step (ih.wd w hw hsnd hstd) hs'

/-! ### Inv: preservation under the safe configuration -/

section
variable {cfg : Config Chain Body Hash}

theorem inv_init {s₀ : State Chain Hash} (h0 : Init s₀) (hg : GovInit cfg s₀) : Inv cfg s₀ where
  sent_le z h := by rw [h0.sentAt_zero z h]; exact Nat.zero_le _
  sent_up z h hne := (hne (h0.sentAt_zero z h)).elim
  ev_le z h e he := (h0.events_empty z h e he).elim
  sent_ev z h hne := (hne (h0.sentAt_zero z h)).elim
  rel_ev x _ _ hr := (h0.relayed_empty x _ hr).elim
  gov := hg
  wd w hw := (h0.withdrawals_empty w hw).elim
  dep f hf := (h0.deposits_empty f hf).elim
  exp z h he := (h0.expired_none z h he).elim
  ref z h _ hr := (h0.refunded_none z h hr).elim
  refunds_le z h _ := by rw [h0.refunds_zero z h]; exact Nat.zero_le _
  refunds_zero z h _ _ := h0.refunds_zero z h

theorem inv_step (hc : SafeConfig cfg) (hinj : HashInjective cfg.hash) {a : Action Chain Body Hash}
    {s s' : State Chain Hash} (hI : Inv cfg s) (hs : Step cfg a s s') : Inv cfg s' := by
  obtain ⟨hg, rfl⟩ := hs
  cases a with
  | tick y t =>
    simp only [guard] at hg
    have hmono : ∀ c, s.clock c ≤ upd1 s.clock y t c := by
      intro c; simp only [upd1]; split
      · rename_i h; subst h; omega
      · exact Nat.le_refl _
    have honest : ∀ y' f, Honest cfg s y' f → Honest cfg (next cfg (.tick y t) s) y' f :=
      fun y' f ⟨z, b, hh, ht, hf⟩ => ⟨z, b, hh, Nat.le_trans ht (hmono y'), hf⟩
    exact ⟨fun z h => Nat.le_trans (hI.sent_le z h) (hmono z), hI.sent_up, hI.ev_le, hI.sent_ev,
      hI.rel_ev, hI.gov, fun w hw h1 h2 => honest _ _ (hI.wd w hw h1 h2),
      fun f hf => let ⟨y', hy, hh⟩ := hI.dep f hf; ⟨y', hy, honest _ _ hh⟩,
      hI.exp, hI.ref, hI.refunds_le, hI.refunds_zero⟩
  | upgrade y =>
    refine ⟨hI.sent_le, ?_, hI.ev_le, hI.sent_ev, hI.rel_ev, hI.gov, hI.wd, hI.dep, hI.exp, hI.ref,
      hI.refunds_le, hI.refunds_zero⟩
    intro z h hne
    simp only [next, upd1]; split
    · rfl
    · exact hI.sent_up z h hne
  | join z y =>
    simp only [guard] at hg
    refine ⟨hI.sent_le, hI.sent_up, hI.ev_le, hI.sent_ev, hI.rel_ev, ?_, hI.wd, hI.dep, hI.exp,
      hI.ref, hI.refunds_le, hI.refunds_zero⟩
    intro a c hl
    rcases hl with hl | ⟨_, rfl⟩
    · exact hI.gov a c hl
    · exact hg hc.govCheck
  | send z d b =>
    simp only [guard] at hg
    obtain ⟨_, hfresh, _⟩ := hg
    have hsz : s.sentAt z (cfg.hash d z b) = 0 :=
      Decidable.byContradiction fun hne => hfresh (hI.sent_ev _ _ hne)
    -- The new sentAt differs from the old one at most at (z, H), where it is clock z or 0.
    have hS : ∀ c h, ¬ (c = z ∧ h = cfg.hash d z b) →
        (next cfg (.send z d b) s).sentAt c h = s.sentAt c h := by
      intro c h hne; simp only [next]; split
      · exact upd2_ne _ _ hne
      · rfl
    have hS' : (s.upgraded z = true ∧
          (next cfg (.send z d b) s).sentAt z (cfg.hash d z b) = s.clock z) ∨
        (next cfg (.send z d b) s).sentAt z (cfg.hash d z b) = 0 := by
      simp only [next]; split
      · rename_i hu; exact Or.inl ⟨hu, upd2_eq _ _ _ _⟩
      · exact Or.inr hsz
    have honest : ∀ y f, cfg.standard y → Honest cfg s y f →
        Honest cfg (next cfg (.send z d b) s) y f := by
      intro y f hy ⟨z₀, b₀, hh, ht, hf⟩
      refine ⟨z₀, b₀, hh, ht, fun hzs hz hlt hr => ?_⟩
      by_cases hc' : z₀ = z ∧ f.hash = cfg.hash d z b
      · obtain ⟨rfl, hfH⟩ := hc'
        obtain ⟨rfl, -, rfl⟩ := hinj _ _ _ _ _ _ (hh.symm.trans hfH)
        obtain ⟨e, he⟩ := hI.rel_ev y z₀ b₀ (hh ▸ hr) hy
        exact hfresh ⟨e, hfH ▸ hh ▸ he⟩
      · rw [hS _ _ hc'] at hz hlt
        exact hf hzs hz hlt hr
    refine ⟨?_, ?_, ?_, ?_, ?_, hI.gov, fun w hw h1 h2 => honest _ _ h2 (hI.wd w hw h1 h2),
      fun f hf => let ⟨y, hy, hh⟩ := hI.dep f hf; ⟨y, hy, honest _ _ hy hh⟩, ?_, hI.ref,
      hI.refunds_le, hI.refunds_zero⟩
    · intro c h
      by_cases hc' : c = z ∧ h = cfg.hash d z b
      · obtain ⟨rfl, rfl⟩ := hc'
        rcases hS' with ⟨_, he⟩ | he <;> rw [he]
        · exact Nat.le_refl _
        · exact Nat.zero_le _
      · rw [hS _ _ hc']; exact hI.sent_le c h
    · intro c h hne
      by_cases hc' : c = z ∧ h = cfg.hash d z b
      · obtain ⟨rfl, rfl⟩ := hc'
        rcases hS' with ⟨hu, _⟩ | he
        · exact hu
        · exact (hne he).elim
      · rw [hS _ _ hc'] at hne; exact hI.sent_up c h hne
    · intro c h e he hstd
      rcases he with he | ⟨rfl, rfl, rfl⟩
      · have hc' : ¬ (c = z ∧ h = cfg.hash d z b) := fun ⟨h1, h2⟩ =>
          hfresh ⟨e, by subst h1 h2; exact he⟩
        rw [hS _ _ hc']; exact hI.ev_le c h e he hstd
      · rcases hS' with ⟨_, he⟩ | he
        · rw [he]; exact Or.inr (Nat.le_refl _)
        · exact Or.inl he
    · intro c h hne
      by_cases hc' : c = z ∧ h = cfg.hash d z b
      · obtain ⟨rfl, rfl⟩ := hc'
        exact ⟨s.clock c, Or.inr ⟨rfl, rfl, rfl⟩⟩
      · rw [hS _ _ hc'] at hne
        obtain ⟨e, he⟩ := hI.sent_ev c h hne
        exact ⟨e, Or.inl he⟩
    · intro x z' b' hr hx
      obtain ⟨e, he⟩ := hI.rel_ev x z' b' hr hx
      exact ⟨e, Or.inl he⟩
    · intro c h he
      obtain ⟨f, hf, hto, hfh, hz, hlt⟩ := hI.exp c h he
      have hc' : ¬ (c = z ∧ h = cfg.hash d z b) := fun ⟨h1, h2⟩ => hz (by subst h1 h2; exact hsz)
      refine ⟨f, hf, hto, hfh, ?_⟩
      rw [hS _ _ hc']; exact ⟨hz, hlt⟩
  | resend z d b =>
    simp only [guard] at hg
    obtain ⟨hr, hsent, hne⟩ := hg
    have hrs : cfg.resendRestarts = true := by
      rcases hc.resend with h | h
      · rw [h] at hr; cases hr
      · exact h
    have hup : s.upgraded z = true := hI.sent_up _ _ hsent
    have hnext : (next cfg (.resend z d b) s).sentAt =
        upd2 s.sentAt z (cfg.hash d z b) (s.clock z) := by
      simp only [next, hrs, hup, Bool.and_self, ↓reduceIte]
    have hS : ∀ c h, ¬ (c = z ∧ h = cfg.hash d z b) →
        (next cfg (.resend z d b) s).sentAt c h = s.sentAt c h := by
      intro c h hne; rw [hnext]; exact upd2_ne _ _ hne
    have hS' : (next cfg (.resend z d b) s).sentAt z (cfg.hash d z b) = s.clock z := by
      rw [hnext]; exact upd2_eq _ _ _ _
    have hle := hI.sent_le z (cfg.hash d z b)
    have honest : ∀ y f, Honest cfg s y f → Honest cfg (next cfg (.resend z d b) s) y f := by
      intro y f ⟨z₀, b₀, hh, ht, hf⟩
      refine ⟨z₀, b₀, hh, ht, fun hzs hz hlt hrl => ?_⟩
      by_cases hc' : z₀ = z ∧ f.hash = cfg.hash d z b
      · obtain ⟨rfl, hfH⟩ := hc'
        rw [hfH, hS'] at hlt
        exact hf hzs (hfH ▸ hsent) (by rw [hfH]; exact expiredBy_mono hle hlt) hrl
      · rw [hS _ _ hc'] at hz hlt
        exact hf hzs hz hlt hrl
    refine ⟨?_, ?_, ?_, ?_, ?_, hI.gov, fun w hw h1 h2 => honest _ _ (hI.wd w hw h1 h2),
      fun f hf => let ⟨y, hy, hh⟩ := hI.dep f hf; ⟨y, hy, honest _ _ hh⟩, ?_, hI.ref,
      hI.refunds_le, hI.refunds_zero⟩
    · intro c h
      by_cases hc' : c = z ∧ h = cfg.hash d z b
      · obtain ⟨rfl, rfl⟩ := hc'; rw [hS']; exact Nat.le_refl _
      · rw [hS _ _ hc']; exact hI.sent_le c h
    · intro c h hne'
      by_cases hc' : c = z ∧ h = cfg.hash d z b
      · obtain ⟨rfl, rfl⟩ := hc'; exact hup
      · rw [hS _ _ hc'] at hne'; exact hI.sent_up c h hne'
    · intro c h e he hstd
      by_cases hc' : c = z ∧ h = cfg.hash d z b
      · obtain ⟨rfl, rfl⟩ := hc'
        rw [hS']
        rcases he with he | ⟨_, _, rfl⟩
        · rcases hI.ev_le _ _ e he hstd with h0 | h0
          · exact (hsent h0).elim
          · exact Or.inr (by omega)
        · exact Or.inr (Nat.le_refl _)
      · rw [hS _ _ hc']
        rcases he with he | ⟨h1, h2, _⟩
        · exact hI.ev_le c h e he hstd
        · exact (hc' ⟨h1, h2⟩).elim
    · intro c h hne'
      by_cases hc' : c = z ∧ h = cfg.hash d z b
      · obtain ⟨rfl, rfl⟩ := hc'
        exact ⟨s.clock c, Or.inr ⟨rfl, rfl, rfl⟩⟩
      · rw [hS _ _ hc'] at hne'
        obtain ⟨e, he⟩ := hI.sent_ev c h hne'
        exact ⟨e, Or.inl he⟩
    · intro x z' b' hrl hx
      obtain ⟨e, he⟩ := hI.rel_ev x z' b' hrl hx
      exact ⟨e, Or.inl he⟩
    · intro c h he
      have hc' : ¬ (c = z ∧ h = cfg.hash d z b) := fun ⟨h1, h2⟩ =>
        hne hrs (by subst h1 h2; exact he)
      obtain ⟨f, hf, hto, hfh, hz, hlt⟩ := hI.exp c h he
      refine ⟨f, hf, hto, hfh, ?_⟩
      rw [hS _ _ hc']; exact ⟨hz, hlt⟩
  | resendLegacy z d b =>
    simp only [guard] at hg
    obtain ⟨hnu, _⟩ := hg
    have hsz : s.sentAt z (cfg.hash d z b) = 0 :=
      Decidable.byContradiction fun hne => by rw [hI.sent_up _ _ hne] at hnu; cases hnu
    refine ⟨hI.sent_le, hI.sent_up, ?_, ?_, ?_, hI.gov, hI.wd, hI.dep, hI.exp, hI.ref,
      hI.refunds_le, hI.refunds_zero⟩
    · intro c h e he hstd
      rcases he with he | ⟨rfl, rfl, _⟩
      · exact hI.ev_le c h e he hstd
      · exact Or.inl hsz
    · intro c h hne
      obtain ⟨e, he⟩ := hI.sent_ev c h hne
      exact ⟨e, Or.inl he⟩
    · intro x z' b' hr hx
      obtain ⟨e, he⟩ := hI.rel_ev x z' b' hr hx
      exact ⟨e, Or.inl he⟩
  | relay x z b =>
    simp only [guard] at hg
    obtain ⟨_, hwin, _⟩ := hg
    have honest : ∀ y f, cfg.standard y → Honest cfg s y f →
        Honest cfg (next cfg (.relay x z b) s) y f := by
      intro y f hy ⟨z₀, b₀, hh, ht, hf⟩
      refine ⟨z₀, b₀, hh, ht, fun hzs hz hlt hr => ?_⟩
      rcases hr with hr | ⟨rfl, hfH⟩
      · exact hf hzs hz hlt hr
      · obtain ⟨-, rfl, rfl⟩ := hinj _ _ _ _ _ _ (hh.symm.trans hfH)
        obtain ⟨e, he, hle⟩ := hwin hy
        rw [← hfH] at he
        rcases hI.ev_le _ _ e he hzs with h0 | h0
        · exact hz h0
        · change f.time ≤ s.clock y at ht
          change expiredBy cfg (s.sentAt z₀ f.hash) f.time at hlt
          have := expiredBy_window (hc.window y) hlt
          omega
    refine ⟨hI.sent_le, hI.sent_up, hI.ev_le, hI.sent_ev, ?_, hI.gov, ?_,
      fun f hf => let ⟨y, hy, hh⟩ := hI.dep f hf; ⟨y, hy, honest _ _ hy hh⟩, hI.exp, hI.ref,
      hI.refunds_le, hI.refunds_zero⟩
    · intro x' z' b' hr hx
      rcases hr with hr | ⟨rfl, hH⟩
      · exact hI.rel_ev x' z' b' hr hx
      · obtain ⟨-, rfl, rfl⟩ := hinj _ _ _ _ _ _ hH
        obtain ⟨e, he, _⟩ := hwin hx
        exact ⟨e, he⟩
    · intro w hw h1 h2
      rcases hw with hw | hw
      · exact honest _ _ h2 (hI.wd w hw h1 h2)
      · rw [callOut_sender hw] at h1; cases h1
  | exportUndelivered y z b route =>
    simp only [guard] at hg
    obtain ⟨_, _, hnr⟩ := hg
    refine ⟨hI.sent_le, hI.sent_up, hI.ev_le, hI.sent_ev, hI.rel_ev, hI.gov, ?_, hI.dep, hI.exp,
      hI.ref, hI.refunds_le, hI.refunds_zero⟩
    intro w hw h1 h2
    rcases hw with hw | rfl
    · exact hI.wd w hw h1 h2
    · exact ⟨z, b, rfl, Nat.le_refl _, fun _ _ _ => hnr⟩
  | userWithdrawal y a f =>
    refine ⟨hI.sent_le, hI.sent_up, hI.ev_le, hI.sent_ev, hI.rel_ev, hI.gov, ?_, hI.dep, hI.exp,
      hI.ref, hI.refunds_le, hI.refunds_zero⟩
    intro w hw h1 h2
    rcases hw with hw | rfl
    · exact hI.wd w hw h1 h2
    · cases h1
  | arbitraryCode y snd f =>
    refine ⟨hI.sent_le, hI.sent_up, hI.ev_le, hI.sent_ev, hI.rel_ev, hI.gov, ?_, hI.dep, hI.exp,
      hI.ref, hI.refunds_le, hI.refunds_zero⟩
    intro w hw h1 h2
    rcases hw with hw | rfl
    · exact hI.wd w hw h1 h2
    · exact (hg h2).elim
  | arbitraryEvent z h t =>
    refine ⟨hI.sent_le, hI.sent_up, ?_, ?_, ?_, hI.gov, hI.wd, hI.dep, hI.exp, hI.ref,
      hI.refunds_le, hI.refunds_zero⟩
    · intro c h' e he hstd
      rcases he with he | ⟨rfl, _, _⟩
      · exact hI.ev_le c h' e he hstd
      · exact (hg hstd).elim
    · intro c h' hne
      obtain ⟨e, he⟩ := hI.sent_ev c h' hne
      exact ⟨e, Or.inl he⟩
    · intro x z' b' hr hx
      obtain ⟨e, he⟩ := hI.rel_ev x z' b' hr hx
      exact ⟨e, Or.inl he⟩
  | l1Relay w =>
    simp only [guard] at hg
    obtain ⟨hw, _, hlb, hsnd, _⟩ := hg
    refine ⟨hI.sent_le, hI.sent_up, hI.ev_le, hI.sent_ev, hI.rel_ev, hI.gov, hI.wd, ?_, ?_, hI.ref,
      hI.refunds_le, hI.refunds_zero⟩
    · intro f hf
      rcases hf with hf | rfl
      · exact hI.dep f hf
      · have hstd := hI.gov _ _ (hlb hc.lockboxCheck)
        have hex : w.sender = .exporter := (hsnd hc.senderCheck).trans hc.trusted
        exact ⟨w.origin, hstd, hI.wd w hw hex hstd⟩
    · intro z h he
      obtain ⟨f, hf, rest⟩ := hI.exp z h he
      exact ⟨f, Or.inl hf, rest⟩
  | fakeCaller f =>
    simp only [guard, hc.realMessengerCheck, hc.lockboxCheck, hc.sysConfigConsistent] at hg
    rcases hg with hg | hg | hg <;> cases hg
  | l1cdmSelfRelay f =>
    simp only [guard, hc.unsafeTargetCheck] at hg; cases hg
  | arbitraryRefund z h =>
    simp only [guard] at hg
    refine ⟨hI.sent_le, hI.sent_up, hI.ev_le, hI.sent_ev, hI.rel_ev, hI.gov, hI.wd, hI.dep, hI.exp,
      ?_, ?_, ?_⟩
    · intro c h' hstd hr
      rcases hr with hr | ⟨rfl, _⟩
      · exact hI.ref c h' hstd hr
      · exact (hg hstd).elim
    · intro c h' hstd
      simp only [next]
      have hc' : ¬ (c = z ∧ h' = h) := fun ⟨e, _⟩ => hg (e ▸ hstd)
      rw [upd2_ne _ _ hc']; exact hI.refunds_le c h' hstd
    · intro c h' hstd hnr'
      simp only [next]
      have hc' : ¬ (c = z ∧ h' = h) := fun ⟨e, _⟩ => hg (e ▸ hstd)
      rw [upd2_ne _ _ hc']
      exact hI.refunds_zero c h' hstd (fun r => hnr' (Or.inl r))
  | expire f =>
    simp only [guard] at hg
    obtain ⟨hf, hz, hlt⟩ := hg
    refine ⟨hI.sent_le, hI.sent_up, hI.ev_le, hI.sent_ev, hI.rel_ev, hI.gov, hI.wd, hI.dep, ?_, ?_,
      hI.refunds_le, hI.refunds_zero⟩
    · intro z h he
      rcases he with he | ⟨rfl, rfl⟩
      · exact hI.exp z h he
      · exact ⟨f, hf, rfl, rfl, hz, hlt⟩
    · intro z h hstd hr; exact Or.inl (hI.ref z h hstd hr)
  | refund z d b =>
    simp only [guard] at hg
    obtain ⟨_, he, hnr⟩ := hg
    refine ⟨hI.sent_le, hI.sent_up, hI.ev_le, hI.sent_ev, hI.rel_ev, hI.gov, hI.wd, hI.dep, hI.exp,
      ?_, ?_, ?_⟩
    · intro c h hstd hr
      rcases hr with hr | ⟨rfl, rfl⟩
      · exact hI.ref c h hstd hr
      · exact he
    · intro c h hstd
      simp only [next]
      by_cases hc' : c = z ∧ h = cfg.hash d z b
      · obtain ⟨rfl, rfl⟩ := hc'
        rw [upd2_eq, hI.refunds_zero _ _ hstd hnr]; exact Nat.le_refl _
      · rw [upd2_ne _ _ hc']; exact hI.refunds_le c h hstd
    · intro c h hstd hnr'
      simp only [next]
      have hc' : ¬ (c = z ∧ h = cfg.hash d z b) := fun e => hnr' (Or.inr e)
      rw [upd2_ne _ _ hc']
      exact hI.refunds_zero c h hstd (fun r => hnr' (Or.inl r))

/-- Main induction: `Inv`, `HistW` and the deposit history hold on every reachable state. -/
theorem inv_reach (hc : SafeConfig cfg) (hinj : HashInjective cfg.hash) {s₀ s : State Chain Hash}
    (h0 : Init s₀) (hg : GovInit cfg s₀) (hr : Reach cfg s₀ s) :
    Inv cfg s ∧ HistW cfg s₀ s ∧ NoForgedFact cfg s₀ s := by
  induction hr with
  | refl => exact ⟨inv_init h0 hg, histW_reach h0 (Reach.refl _),
      fun f hf => (h0.deposits_empty f hf).elim⟩
  | @tail s₁ s₂ a hr' hs ih =>
    obtain ⟨hI, hW, hD⟩ := ih
    refine ⟨inv_step hc hinj hI hs, histW_reach h0 (Reach.tail a hr' hs), ?_⟩
    intro f hf
    have hs' := hs
    obtain ⟨hgd, rfl⟩ := hs
    cases a with
    | l1Relay w =>
      rcases hf with hf | rfl
      · obtain ⟨y, hy, hx⟩ := hD f hf
        exact ⟨y, hy, exportedBy_step hx hs'⟩
      · simp only [guard] at hgd
        obtain ⟨hw, _, hlb, hsnd, _⟩ := hgd
        have hstd := hI.gov _ _ (hlb hc.lockboxCheck)
        have hex : w.sender = .exporter := (hsnd hc.senderCheck).trans hc.trusted
        exact ⟨w.origin, hstd, exportedBy_step (hW.wd w hw hex hstd) hs'⟩
    | fakeCaller f' =>
      simp only [guard, hc.realMessengerCheck, hc.lockboxCheck, hc.sysConfigConsistent] at hgd
      rcases hgd with hgd | hgd | hgd <;> cases hgd
    | l1cdmSelfRelay f' =>
      simp only [guard, hc.unsafeTargetCheck] at hgd; cases hgd
    | _ =>
      obtain ⟨y, hy, hx⟩ := hD f hf
      exact ⟨y, hy, exportedBy_step hx hs'⟩

end

end Expiry
