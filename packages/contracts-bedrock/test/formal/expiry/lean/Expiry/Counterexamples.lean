import Expiry.Safety

/-!
Non-vacuity checks and counterexamples on a concrete instance.

Instance (`base`): chains are `Nat` (0 = A, 1 = B, 2 = C, 3 = D). A, B, C are standard; D runs
arbitrary code. At genesis A and B are authorized in every lockbox; C may join later. Bodies are
`Nat`, the hash is the identity on (destination, source, body), so it is injective. Body 9 decodes to
a call to the L2CrossDomainMessenger carrying the forged fact "(B, A, 5) was undelivered at time
100, route to A". Every other body decodes to `other`. Time unit = 1 day, W_d = 7 for every d,
P = 8. The protected message is `mAB = (1, 0, 5)`: A sends body 5 to B.

* Witnesses (all hypotheses of the safety theorem hold):
  - `refund_reachable`
  - `relay_reachable_at_edge`
  - `lateJoin_multiSource_reachable`
  - `forgery_reachable_but_harmless`
  - `safe_variants`
* `cex_*`: one assumption dropped (one `Config` field changed, except `cex_nonStrict`, which
  changes two; or injectivity dropped). Each gives an explicit execution from genesis that ends
  with `¬ NoDoubleSpend`.
-/
namespace Expiry.Examples

open Expiry

set_option linter.unusedSectionVars false
set_option linter.unusedSimpArgs false

attribute [local simp] Call.isL2CDM

section Run
variable {Chain Body Hash : Type} [DecidableEq Chain] [DecidableEq Hash]

def run (cfg : Config Chain Body Hash) : List (Action Chain Body Hash) → State Chain Hash →
    State Chain Hash
  | [], s => s
  | a :: as, s => run cfg as (next cfg a s)

def Valid (cfg : Config Chain Body Hash) : List (Action Chain Body Hash) → State Chain Hash → Prop
  | [], _ => True
  | a :: as, s => guard cfg a s ∧ Valid cfg as (next cfg a s)

theorem reach_run (cfg : Config Chain Body Hash) :
    ∀ (as : List (Action Chain Body Hash)) (s : State Chain Hash),
      Valid cfg as s → Reach cfg s (run cfg as s)
  | [], s, _ => Reach.refl s
  | _ :: as, _, ⟨hg, hv⟩ => Reach.trans (Reach.single ⟨hg, rfl⟩) (reach_run cfg as _ hv)

end Run

abbrev H := Nat × Nat × Nat

def fM : Fact Nat H := ⟨0, (1, 0, 5), 100⟩

def base : Config Nat Nat H where
  hash := fun d z b => (d, z, b)
  decode := fun b => if b = 9 then .l2cdm fM else .other
  isBridge := fun _ => True
  standard := fun c => c ≤ 2
  interop := fun _ => True
  protocolWindow := fun _ => 7
  contractPeriod := 8
  trusted := .exporter
  targetRule := true
  realMessengerCheck := true
  lockboxCheck := true
  senderCheck := true
  govCheck := true
  unsafeTargetCheck := true
  sysConfigConsistent := true
  expireGe := false
  resend := false
  resendRestarts := false

/-- Genesis. -/
def s0 : State Nat H where
  clock := fun _ => 1
  upgraded := fun _ => false
  lockbox := fun _ y => y ≤ 1
  sentAt := fun _ _ => 0
  events := fun _ _ _ => False
  relayed := fun _ _ => False
  withdrawals := fun _ => False
  deposits := fun _ => False
  expired := fun _ _ => False
  refunded := fun _ _ => False
  refunds := fun _ _ => 0

theorem s0_init : Init s0 :=
  ⟨fun _ => rfl, fun _ _ => rfl, fun _ _ _ h => h, fun _ _ h => h, fun _ h => h, fun _ h => h,
   fun _ _ h => h, fun _ _ h => h, fun _ _ => rfl⟩

theorem s0_gov (cfg : Config Nat Nat H) (hstd : cfg.standard = fun c => c ≤ 2) :
    GovInit cfg s0 := by
  intro z y h; rw [hstd]; simp only [s0] at h; show y ≤ 2; omega

theorem base_safe : SafeConfig base :=
  ⟨rfl, rfl, rfl, rfl, rfl, rfl, rfl, fun _ => show 7 + 0 ≤ 8 by decide, Or.inl rfl⟩

theorem inj_id : HashInjective (fun (d z b : Nat) => ((d, z, b) : H)) := by
  intro d z b d' z' b' h
  simp only [Prod.mk.injEq] at h
  exact h

def mAB : H := (1, 0, 5)

/-! ### Witnesses in the safe design -/

def fB : Fact Nat H := ⟨0, mAB, 10⟩

def refundTrace : List (Action Nat Nat H) :=
  [.upgrade 0, .upgrade 1, .send 0 1 5, .tick 1 10, .exportUndelivered 1 0 5 0,
   .l1Relay ⟨1, .exporter, fB⟩, .expire fB, .refund 0 1 5]

/-- All hypotheses hold and a refund is reachable. -/
theorem refund_reachable :
    SafeConfig base ∧ HashInjective base.hash ∧ Init s0 ∧ GovInit base s0 ∧
    ∃ s, Reach base s0 s ∧ s.expired 0 mAB ∧ s.refunded 0 mAB ∧ s.refunds 0 mAB = 1 := by
  refine ⟨base_safe, inj_id, s0_init, s0_gov _ rfl, run base refundTrace s0,
    reach_run _ _ _ ?_, ?_, ?_, ?_⟩
  · simp [Valid, refundTrace, guard, next, base, s0, upd1, upd2, expiredBy, fB, mAB]
  all_goals simp [run, refundTrace, next, base, s0, upd1, upd2, fB, mAB]

/-- A relay is reachable from genesis, exactly at the window edge (exec - init = W). -/
theorem relay_reachable_at_edge :
    ∃ s, Reach base s0 s ∧ s.relayed 1 mAB ∧ s.clock 1 = s.sentAt 0 mAB + 7 := by
  refine ⟨run base [.upgrade 0, .upgrade 1, .send 0 1 5, .tick 1 8, .relay 1 0 5] s0,
    reach_run _ _ _ ?_, ?_, ?_⟩
  · simp [Valid, guard, next, base, s0, upd1, upd2, withinWindow]
  · simp [run, next, base, s0, upd1, upd2, mAB]
  · simp [run, next, base, s0, upd1, upd2, mAB]

def fC : Fact Nat H := ⟨0, (2, 0, 6), 10⟩
def fBA : Fact Nat H := ⟨1, (0, 1, 6), 10⟩

def lateTrace : List (Action Nat Nat H) :=
  [.upgrade 0, .upgrade 1, .upgrade 2, .join 0 2,
   -- A -> C, C joined A's lockbox after genesis; no history check needed.
   .send 0 2 6, .tick 2 10, .exportUndelivered 2 0 6 0, .l1Relay ⟨2, .exporter, fC⟩,
   .expire fC, .refund 0 2 6,
   -- B -> A, exported by A and routed to B (the source).
   .send 1 0 6, .tick 0 10, .exportUndelivered 0 1 6 1, .l1Relay ⟨0, .exporter, fBA⟩,
   .expire fBA, .refund 1 0 6]

/-- A late-joining standard chain and a second source chain both get refunds. -/
theorem lateJoin_multiSource_reachable :
    ∃ s, Reach base s0 s ∧ s.refunded 0 (2, 0, 6) ∧ s.refunded 1 (0, 1, 6) := by
  refine ⟨run base lateTrace s0, reach_run _ _ _ ?_, ?_, ?_⟩
  · simp [Valid, lateTrace, guard, next, base, s0, upd1, upd2, expiredBy, fC, fBA]
  all_goals simp [run, lateTrace, next, base, s0, upd1, upd2, fC, fBA]

/-- The attacker can make 0x..23 send a withdrawal carrying the forged fact (a pre-upgrade relay
of a message whose target is the L2CrossDomainMessenger), and any user can send one. Both exist in
a reachable state of the safe design, and the safety theorem still applies to every
continuation. -/
theorem forgery_reachable_but_harmless :
    ∃ s, Reach base s0 s ∧ s.withdrawals ⟨1, .messenger, fM⟩ ∧ s.withdrawals ⟨1, .user 42, fM⟩ ∧
      ∀ s', Reach base s s' → NoDoubleSpend base s' := by
  refine ⟨run base [.send 2 1 9, .relay 1 2 9, .userWithdrawal 1 42 fM] s0, ?_, ?_, ?_, ?_⟩
  · exact reach_run _ _ _ (by simp [Valid, guard, next, base, s0, withinWindow])
  · simp [run, next, base, s0, callOut]
  · simp [run, next, base, s0]
  · intro s' hr'
    exact noDoubleSpend base_safe inj_id s0_init (s0_gov _ rfl)
      (Reach.trans (reach_run _ _ _ (by simp [Valid, guard, next, base, s0, withinWindow])) hr')

/-- Variants covered by the theorem: no target rule (defense in depth only), P = W (no margin),
per-destination windows, a resend that restarts the timestamp, and the single-field `≥` mutation
with the 1-day margin (W = 7 < P = 8). -/
theorem safe_variants :
    SafeConfig { base with targetRule := false } ∧
    SafeConfig { base with contractPeriod := 7 } ∧
    SafeConfig { base with protocolWindow := fun d => if d = 1 then 3 else 7 } ∧
    SafeConfig { base with resend := true, resendRestarts := true } ∧
    SafeConfig { base with expireGe := true } := by
  refine ⟨⟨rfl, rfl, rfl, rfl, rfl, rfl, rfl, fun _ => show 7 + 0 ≤ 8 by decide, Or.inl rfl⟩,
    ⟨rfl, rfl, rfl, rfl, rfl, rfl, rfl, fun _ => show 7 + 0 ≤ 7 by decide, Or.inl rfl⟩,
    ⟨rfl, rfl, rfl, rfl, rfl, rfl, rfl, fun d => ?_, Or.inl rfl⟩,
    ⟨rfl, rfl, rfl, rfl, rfl, rfl, rfl, fun _ => show 7 + 0 ≤ 8 by decide, Or.inr rfl⟩,
    ⟨rfl, rfl, rfl, rfl, rfl, rfl, rfl, fun _ => show 7 + 1 ≤ 8 by decide, Or.inl rfl⟩⟩
  show (if d = 1 then 3 else 7) + 0 ≤ 8
  split <;> decide

/-- Pre-upgrade history: A sends and the old permissionless resendMessage re-emits the event much
later; no timestamp is ever recorded for it, so it can never expire (`expire` needs sentAt ≠ 0). -/
theorem legacyResend_reachable :
    ∃ s, Reach base s0 s ∧ s.events 0 mAB 1 ∧ s.events 0 mAB 30 ∧ s.sentAt 0 mAB = 0 := by
  refine ⟨run base [.send 0 1 5, .tick 0 30, .resendLegacy 0 1 5] s0, reach_run _ _ _ ?_,
    ?_, ?_, ?_⟩
  · simp [Valid, guard, next, base, s0, upd1, mAB]
  all_goals simp [run, next, base, s0, upd1, mAB]

/-! ### Counterexamples: each drops one assumption -/

/-- Shape of every counterexample: genesis, governance at genesis and injectivity hold, the config
is not safe, and an execution reaches a double spend of a standard source's message. -/
def Cex (cfg : Config Nat Nat H) (as : List (Action Nat Nat H)) : Prop :=
  Init s0 ∧ GovInit cfg s0 ∧ HashInjective cfg.hash ∧ ¬ SafeConfig cfg ∧
  Valid cfg as s0 ∧ Reach cfg s0 (run cfg as s0) ∧ ¬ NoDoubleSpend cfg (run cfg as s0)

def wM : Withdrawal Nat H := ⟨1, .messenger, fM⟩

/-- The earlier design (relayUndeliveredMessage trusts 0x..23): before B's upgrade, B relays an
attacker message whose target is the L2CrossDomainMessenger, so 0x..23 sends the forged fact. After
the upgrade A's real message is relayed on B, and then the pre-staged fact expires it. -/
def cfgMessengerTrusted : Config Nat Nat H := { base with trusted := .messenger }

def messengerTrustedTrace : List (Action Nat Nat H) :=
  [.send 2 1 9, .relay 1 2 9, .upgrade 0, .upgrade 1, .send 0 1 5, .relay 1 0 5, .l1Relay wM,
   .expire fM, .refund 0 1 5]

theorem cex_messengerTrusted : Cex cfgMessengerTrusted messengerTrustedTrace := by
  have hv : Valid cfgMessengerTrusted messengerTrustedTrace s0 := by
    simp [Valid, messengerTrustedTrace, cfgMessengerTrusted, guard, next, base, s0, upd1, upd2,
      withinWindow, expiredBy, callOut, wM, fM]
  refine ⟨s0_init, s0_gov _ rfl, inj_id, (fun h => by cases h.trusted), hv, reach_run _ _ _ hv, ?_⟩
  intro h
  exact h 1 0 5 (show (0 : Nat) ≤ 2 by decide) (by
    simp [run, messengerTrustedTrace, cfgMessengerTrusted, next, base, s0, upd1, upd2, callOut,
      wM, fM])

def fD : Fact Nat H := fM

/-- Governance rule dropped: D (non-standard) runs arbitrary code at the exporter address, signs
the forged fact as the exporter, and is then authorized in A's lockbox. Its old withdrawal becomes
trusted. -/
def cfgNoGov : Config Nat Nat H := { base with govCheck := false }

def noGovTrace : List (Action Nat Nat H) :=
  [.arbitraryCode 3 .exporter fD, .join 0 3, .upgrade 0, .upgrade 1, .send 0 1 5, .relay 1 0 5,
   .l1Relay ⟨3, .exporter, fD⟩, .expire fD, .refund 0 1 5]

theorem cex_nonstandardJoin : Cex cfgNoGov noGovTrace := by
  have hv : Valid cfgNoGov noGovTrace s0 := by
    simp [Valid, noGovTrace, cfgNoGov, guard, next, base, s0, upd1, upd2, withinWindow, expiredBy,
      fD, fM]
  refine ⟨s0_init, s0_gov _ rfl, inj_id, (fun h => by cases h.govCheck), hv, reach_run _ _ _ hv, ?_⟩
  intro h
  exact h 1 0 5 (show (0 : Nat) ≤ 2 by decide) (by
    simp [run, noGovTrace, cfgNoGov, next, base, s0, upd1, upd2, fD, fM])

def fEdge : Fact Nat H := ⟨0, mAB, 8⟩

def edgeTrace : List (Action Nat Nat H) :=
  [.upgrade 0, .upgrade 1, .send 0 1 5, .tick 1 8, .exportUndelivered 1 0 5 0,
   .l1Relay ⟨1, .exporter, fEdge⟩, .expire fEdge, .refund 0 1 5, .relay 1 0 5]

/-- P_contract < W_protocol, within the 7-day cap (W = 7, P = 6): B exports at its time 8 > 1 + 6,
and B relays at time 8 ≤ 1 + 7. -/
def cfgPBelowW : Config Nat Nat H := { base with contractPeriod := 6 }

theorem cex_periodBelowWindow : Cex cfgPBelowW edgeTrace := by
  have hv : Valid cfgPBelowW edgeTrace s0 := by
    simp [Valid, edgeTrace, cfgPBelowW, guard, next, base, s0, upd1, upd2, withinWindow,
      expiredBy, fEdge, mAB]
  refine ⟨s0_init, s0_gov _ rfl, inj_id, fun h => absurd (h.window 0) (by decide), hv,
    reach_run _ _ _ hv, ?_⟩
  intro h
  exact h 1 0 5 (show (0 : Nat) ≤ 2 by decide) (by simp [run, edgeTrace, cfgPBelowW, next, base, s0, upd1, upd2])

/-- Non-strict check `t ≥ sentAt + P` with P = W = 7 (two fields changed: with P = 8 the `≥`
mutation alone is still safe, see `safe_variants`): the same edge execution. -/
def cfgNonStrict : Config Nat Nat H := { base with contractPeriod := 7, expireGe := true }

theorem cex_nonStrict : Cex cfgNonStrict edgeTrace := by
  have hv : Valid cfgNonStrict edgeTrace s0 := by
    simp [Valid, edgeTrace, cfgNonStrict, guard, next, base, s0, upd1, upd2, withinWindow,
      expiredBy, fEdge, mAB]
  refine ⟨s0_init, s0_gov _ rfl, inj_id, (fun h => absurd (h.window 0) (by decide)), hv, reach_run _ _ _ hv, ?_⟩
  intro h
  exact h 1 0 5 (show (0 : Nat) ≤ 2 by decide) (by simp [run, edgeTrace, cfgNonStrict, next, base, s0, upd1, upd2])

/-- resendMessage that does not restart the timestamp: the resent event at A's time 10 is within
B's window at B's time 10, although the message expired against sentAt = 1. -/
def cfgResend : Config Nat Nat H := { base with resend := true }

def resendTrace : List (Action Nat Nat H) :=
  [.upgrade 0, .upgrade 1, .send 0 1 5, .tick 0 10, .tick 1 10, .resend 0 1 5,
   .exportUndelivered 1 0 5 0, .l1Relay ⟨1, .exporter, fB⟩, .expire fB, .refund 0 1 5,
   .relay 1 0 5]

theorem cex_resendNoRestart : Cex cfgResend resendTrace := by
  have hv : Valid cfgResend resendTrace s0 := by
    simp [Valid, resendTrace, cfgResend, guard, next, base, s0, upd1, upd2, withinWindow,
      expiredBy, fB, mAB]
  refine ⟨s0_init, s0_gov _ rfl, inj_id, fun h => ?_, hv, reach_run _ _ _ hv, ?_⟩
  · rcases h.resend with h | h <;> cases h
  · intro h
    exact h 1 0 5 (show (0 : Nat) ≤ 2 by decide) (by simp [run, resendTrace, cfgResend, next, base, s0, upd1, upd2])

/-- Real-messenger check dropped: any L1 contract calls relayUndeliveredMessage directly. -/
def cfgNoRealMessenger : Config Nat Nat H := { base with realMessengerCheck := false }

def fakeTrace : List (Action Nat Nat H) :=
  [.upgrade 0, .upgrade 1, .send 0 1 5, .relay 1 0 5, .fakeCaller fM, .expire fM, .refund 0 1 5]

theorem cex_noRealMessengerCheck : Cex cfgNoRealMessenger fakeTrace := by
  have hv : Valid cfgNoRealMessenger fakeTrace s0 := by
    simp [Valid, fakeTrace, cfgNoRealMessenger, guard, next, base, s0, upd1, upd2, withinWindow,
      expiredBy, fM]
  refine ⟨s0_init, s0_gov _ rfl, inj_id, (fun h => by cases h.realMessengerCheck), hv,
    reach_run _ _ _ hv, ?_⟩
  intro h
  exact h 1 0 5 (show (0 : Nat) ≤ 2 by decide) (by
    simp [run, fakeTrace, cfgNoRealMessenger, next, base, s0, upd1, upd2, fM])

/-- Lockbox check dropped, variant 1: D (outside every lockbox) signs the fact as the exporter and
D's real L1CDM relays it. -/
def cfgNoLockbox : Config Nat Nat H := { base with lockboxCheck := false }

def noLockboxTrace : List (Action Nat Nat H) :=
  [.arbitraryCode 3 .exporter fD, .upgrade 0, .upgrade 1, .send 0 1 5, .relay 1 0 5,
   .l1Relay ⟨3, .exporter, fD⟩, .expire fD, .refund 0 1 5]

theorem cex_noLockboxCheck : Cex cfgNoLockbox noLockboxTrace := by
  have hv : Valid cfgNoLockbox noLockboxTrace s0 := by
    simp [Valid, noLockboxTrace, cfgNoLockbox, guard, next, base, s0, upd1, upd2, withinWindow,
      expiredBy, fD, fM]
  refine ⟨s0_init, s0_gov _ rfl, inj_id, (fun h => by cases h.lockboxCheck), hv,
    reach_run _ _ _ hv, ?_⟩
  intro h
  exact h 1 0 5 (show (0 : Nat) ≤ 2 by decide) (by
    simp [run, noLockboxTrace, cfgNoLockbox, next, base, s0, upd1, upd2, fD, fM])

/-- Lockbox check dropped, variant 2: a contract that is not an L1CDM returns a fake portal
whose fake SystemConfig names it, so it passes the real-messenger check, and reports the exporter
as `xDomainMessageSender`. Only the lockbox check (with SystemConfig consistency) stops it. -/
theorem cex_noLockboxCheck_fakePortal : Cex cfgNoLockbox fakeTrace := by
  have hv : Valid cfgNoLockbox fakeTrace s0 := by
    simp [Valid, fakeTrace, cfgNoLockbox, guard, next, base, s0, upd1, upd2, withinWindow,
      expiredBy, fM]
  refine ⟨s0_init, s0_gov _ rfl, inj_id, (fun h => by cases h.lockboxCheck), hv,
    reach_run _ _ _ hv, ?_⟩
  intro h
  exact h 1 0 5 (show (0 : Nat) ≤ 2 by decide) (by
    simp [run, fakeTrace, cfgNoLockbox, next, base, s0, upd1, upd2, fM])

/-- Governance SystemConfig consistency dropped: an authorized portal's SystemConfig names a
contract that is not that chain's L1CDM, so a fake caller passes both identity checks. -/
def cfgNoSysConfig : Config Nat Nat H := { base with sysConfigConsistent := false }

theorem cex_sysConfigInconsistent : Cex cfgNoSysConfig fakeTrace := by
  have hv : Valid cfgNoSysConfig fakeTrace s0 := by
    simp [Valid, fakeTrace, cfgNoSysConfig, guard, next, base, s0, upd1, upd2, withinWindow,
      expiredBy, fM]
  refine ⟨s0_init, s0_gov _ rfl, inj_id, (fun h => by cases h.sysConfigConsistent), hv,
    reach_run _ _ _ hv, ?_⟩
  intro h
  exact h 1 0 5 (show (0 : Nat) ≤ 2 by decide) (by
    simp [run, fakeTrace, cfgNoSysConfig, next, base, s0, upd1, upd2, fM])

/-- L1CDM self-target rule dropped: A's L1CDM relays a withdrawal targeting itself, calls its own
sendMessage, and becomes the L1 sender of an arbitrary expireMessage deposit. -/
def cfgNoUnsafeTarget : Config Nat Nat H := { base with unsafeTargetCheck := false }

def selfRelayTrace : List (Action Nat Nat H) :=
  [.upgrade 0, .upgrade 1, .send 0 1 5, .relay 1 0 5, .l1cdmSelfRelay fM, .expire fM,
   .refund 0 1 5]

theorem cex_noUnsafeTargetCheck : Cex cfgNoUnsafeTarget selfRelayTrace := by
  have hv : Valid cfgNoUnsafeTarget selfRelayTrace s0 := by
    simp [Valid, selfRelayTrace, cfgNoUnsafeTarget, guard, next, base, s0, upd1, upd2,
      withinWindow, expiredBy, fM]
  refine ⟨s0_init, s0_gov _ rfl, inj_id, (fun h => by cases h.unsafeTargetCheck), hv,
    reach_run _ _ _ hv, ?_⟩
  intro h
  exact h 1 0 5 (show (0 : Nat) ≤ 2 by decide) (by
    simp [run, selfRelayTrace, cfgNoUnsafeTarget, next, base, s0, upd1, upd2, fM])

/-- Sender check dropped: a user contract on B sends the forged fact under its own address. -/
def cfgNoSender : Config Nat Nat H := { base with senderCheck := false }

def noSenderTrace : List (Action Nat Nat H) :=
  [.userWithdrawal 1 42 fM, .upgrade 0, .upgrade 1, .send 0 1 5, .relay 1 0 5,
   .l1Relay ⟨1, .user 42, fM⟩, .expire fM, .refund 0 1 5]

theorem cex_noSenderCheck : Cex cfgNoSender noSenderTrace := by
  have hv : Valid cfgNoSender noSenderTrace s0 := by
    simp [Valid, noSenderTrace, cfgNoSender, guard, next, base, s0, upd1, upd2, withinWindow,
      expiredBy, fM]
  refine ⟨s0_init, s0_gov _ rfl, inj_id, (fun h => by cases h.senderCheck), hv,
    reach_run _ _ _ hv, ?_⟩
  intro h
  exact h 1 0 5 (show (0 : Nat) ≤ 2 by decide) (by
    simp [run, noSenderTrace, cfgNoSender, next, base, s0, upd1, upd2, fM])

/-- Hash injectivity dropped (every preimage hashes to `()`): C's honest export of a different
message expires A's message to B after B relayed it. All `SafeConfig` fields hold. -/
def cfgCollide : Config Nat Nat Unit where
  hash := fun _ _ _ => ()
  decode := fun _ => .other
  isBridge := fun _ => True
  standard := fun c => c ≤ 2
  interop := fun _ => True
  protocolWindow := fun _ => 7
  contractPeriod := 8
  trusted := .exporter
  targetRule := true
  realMessengerCheck := true
  lockboxCheck := true
  senderCheck := true
  govCheck := true
  unsafeTargetCheck := true
  sysConfigConsistent := true
  expireGe := false
  resend := false
  resendRestarts := false

def u0 : State Nat Unit where
  clock := fun _ => 1
  upgraded := fun _ => false
  lockbox := fun _ y => y ≤ 1
  sentAt := fun _ _ => 0
  events := fun _ _ _ => False
  relayed := fun _ _ => False
  withdrawals := fun _ => False
  deposits := fun _ => False
  expired := fun _ _ => False
  refunded := fun _ _ => False
  refunds := fun _ _ => 0

def fU : Fact Nat Unit := ⟨0, (), 10⟩

def collideTrace : List (Action Nat Nat Unit) :=
  [.upgrade 0, .upgrade 1, .upgrade 2, .join 0 2, .send 0 1 5, .relay 1 0 5, .tick 2 10,
   .exportUndelivered 2 0 6 0, .l1Relay ⟨2, .exporter, fU⟩, .expire fU, .refund 0 1 5]

theorem cex_hashCollision :
    SafeConfig cfgCollide ∧ ¬ HashInjective cfgCollide.hash ∧ Init u0 ∧ GovInit cfgCollide u0 ∧
    Reach cfgCollide u0 (run cfgCollide collideTrace u0) ∧
    ¬ NoDoubleSpend cfgCollide (run cfgCollide collideTrace u0) := by
  refine ⟨⟨rfl, rfl, rfl, rfl, rfl, rfl, rfl, fun _ => show 7 + 0 ≤ 8 by decide, Or.inl rfl⟩, ?_,
    ⟨fun _ => rfl, fun _ _ => rfl, fun _ _ _ h => h, fun _ _ h => h, fun _ h => h, fun _ h => h,
     fun _ _ h => h, fun _ _ h => h, fun _ _ => rfl⟩, ?_, reach_run _ _ _ ?_, ?_⟩
  · intro hinj
    exact absurd (hinj 1 0 5 2 0 6 rfl).1 (by decide)
  · intro z y h; simp only [u0] at h; show y ≤ 2; omega
  · simp [Valid, collideTrace, cfgCollide, guard, next, u0, upd1, upd2, withinWindow, expiredBy,
      fU]
  · intro h
    exact h 1 0 5 (show (0 : Nat) ≤ 2 by decide) (by simp [run, collideTrace, cfgCollide, next, u0, upd1, upd2])

/-! ### Defense in depth: the messenger's unsafe-target rule -/

/-- Without the target rule (a safe configuration), an upgraded standard chain's messenger can be
made to initiate a withdrawal: A sends body 9 to B, B is upgraded, and B's relay calls the L2CDM.
Safety still holds (`safety_without_targetRule`); only the defense-in-depth property fails. -/
def cfgNoTargetRule : Config Nat Nat H := { base with targetRule := false }

theorem messengerSpeaks_without_targetRule :
    SafeConfig cfgNoTargetRule ∧
    ∃ s s', Reach cfgNoTargetRule s0 s ∧ Step cfgNoTargetRule (.relay 1 0 9) s s' ∧
      s'.withdrawals wM ∧ ¬ s.withdrawals wM ∧ wM.sender = .messenger ∧
      cfgNoTargetRule.standard wM.origin ∧ s.upgraded wM.origin = true := by
  refine ⟨⟨rfl, rfl, rfl, rfl, rfl, rfl, rfl, fun _ => show 7 + 0 ≤ 8 by decide, Or.inl rfl⟩,
    run cfgNoTargetRule [.upgrade 0, .upgrade 1, .send 0 1 9] s0,
    next cfgNoTargetRule (.relay 1 0 9) (run cfgNoTargetRule [.upgrade 0, .upgrade 1, .send 0 1 9] s0),
    reach_run _ _ _ ?_, ⟨?_, rfl⟩, ?_, ?_, rfl, show (1 : Nat) ≤ 2 by decide, ?_⟩
  · simp [Valid, guard, next, cfgNoTargetRule, base, s0, upd1, upd2]
  · simp [run, guard, next, cfgNoTargetRule, base, s0, upd1, upd2, withinWindow]
  · simp [run, next, cfgNoTargetRule, base, s0, upd1, upd2, callOut, wM, fM]
  · simp [run, next, cfgNoTargetRule, base, s0, upd1, upd2, wM]
  · simp [run, next, cfgNoTargetRule, base, s0, upd1, upd2, wM]

end Expiry.Examples
