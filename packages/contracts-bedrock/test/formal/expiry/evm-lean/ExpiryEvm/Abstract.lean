import ExpiryEvm.ExpireMessage
import Reasoning.WordArithmetic

/-!
# Bridge to the abstract `expire` action

`../lean/Expiry/Model.lean` (the protocol model; a v2 is being written concurrently, so this file
does not import it — it is also on a different Lean toolchain) defines

```lean
| .expire f, s => s.deposits f ∧ s.sentAt f.toL1 f.hash ≠ 0 ∧ expiredBy cfg (s.sentAt f.toL1 f.hash) f.time
-- next:
| .expire f, s => { s with expired := fun c h => s.expired c h ∨ (c = f.toL1 ∧ h = f.hash) }
-- with expiredBy cfg sent t = sent + cfg.contractPeriod < t   (cfg.expireGe = false)
```

This file restates exactly that action for one chain `z = f.toL1` (`AbsView`, `absGuard`,
`absNext`) and proves that the compiled code refines it, through the abstraction `viewOf`
of the messenger's storage:

* `Model.State.sentAt z h`  ↦  `sentMessageTimestamps[h]` (as `ℕ`),
* `Model.State.expired z h` ↦  `expiredMessages[h]` (Solidity's bool read: low byte ≠ 0),
* `Model.State.deposits f`  ↦  "the call is the L2CrossDomainMessenger relaying an L1→L2
  message whose L1 sender is this chain's L1CrossDomainMessenger":
  `I.source = l2cdm ∧ xDomainMessageSender() = otherMessenger()`. That the L2CrossDomainMessenger
  only sets `xDomainMessageSender` to the L1 sender of a deposit it is relaying is a property of
  the L2CrossDomainMessenger (not verified here; it is the meaning of the `ReturnsAddress`
  summaries' values `vS`, `vO`).
* `cfg.contractPeriod` ↦ `P_contract` (the compiled constant), `f.hash` ↦ `argHash I`,
  `f.time` ↦ `(argTime I).toNat`.

The model works with ideal hashes; the code with `keccak256` storage slots. The bridge needs that
no other mapping key's slot collides with `expiredMessages[H]` (`NoSlotCollision H`, a
per-key instance of keccak collision resistance; it is not refutable in general, unlike a global
injectivity claim on 64-byte inputs, which would be false by counting).
-/

namespace ExpiryEvm.Abstract

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach Mem

/-- The part of the model state that `expire` reads and writes, for one chain. -/
structure AbsView where
  sentAt : UInt256 → ℕ
  expired : UInt256 → Prop

/-- `Model.expiredBy` with `expireGe = false`. -/
def expiredBy (P sent t : ℕ) : Prop := sent + P < t

/-- `Model.guard (.expire f)` for `f = ⟨z, h, t⟩`, with `deposits f` given as `deposit`. -/
def absGuard (P : ℕ) (deposit : Prop) (v : AbsView) (h : UInt256) (t : ℕ) : Prop :=
  deposit ∧ v.sentAt h ≠ 0 ∧ expiredBy P (v.sentAt h) t

/-- `Model.next (.expire f)`, restricted to chain `z`. -/
def absNext (v : AbsView) (h : UInt256) : AbsView :=
  { v with expired := fun h' => v.expired h' ∨ h' = h }

/-- The abstraction of the messenger's storage at address `a`. -/
def viewOf (σ : AccountMap) (a : AccountAddress) : AbsView where
  sentAt H := (storageWord σ a (sentAtSlot H)).toNat
  expired H := UInt256.land (storageWord σ a (expiredSlot H)) (UInt256.ofNat 0xff) ≠ ⟨0⟩

/-- Per-key keccak collision freedom needed to read the abstract maps back. -/
def NoSlotCollision (H : UInt256) : Prop :=
  (∀ H', H' ≠ H → expiredSlot H' ≠ expiredSlot H) ∧ (∀ H', sentAtSlot H' ≠ expiredSlot H)

/-- The concrete meaning of `deposits ⟨z, H, t⟩` at this call. -/
def DepositCall (I : ExecutionEnv) (vO vS : AccountAddress) : Prop :=
  I.source = l2cdm ∧ vS = vO

theorem setTrueWord_testBit0 (old : UInt256) : (setTrueWord old).toNat.testBit 0 = true := by
  unfold setTrueWord
  set y := UInt256.land (UInt256.ofNat
      115792089237316195423570985008687907853269984665640564039457584007913129639680) old
  have hlt : Nat.lor (UInt256.ofNat 1).toNat y.toNat < UInt256.size := by
    simpa [UInt256.size] using
      nat_lor_lt_two_pow (a := (UInt256.ofNat 1).toNat) (b := y.toNat) (k := 256)
        (by change (UInt256.ofNat 1).val.val < UInt256.size; exact (UInt256.ofNat 1).val.isLt)
        (by change y.val.val < UInt256.size; exact y.val.isLt)
  change (Nat.lor (UInt256.ofNat 1).toNat y.toNat % UInt256.size).testBit 0 = true
  rw [Nat.mod_eq_of_lt hlt, show (UInt256.ofNat 1).toNat = 1 by decide]
  change (1 ||| y.toNat).testBit 0 = true
  rw [Nat.testBit_lor]; simp

theorem land_setTrueWord_ff (old : UInt256) :
    UInt256.land (setTrueWord old) (UInt256.ofNat 0xff) ≠ ⟨0⟩ := by
  intro h
  have h1 := congrArg UInt256.toNat h
  change Nat.land (setTrueWord old).toNat (UInt256.ofNat 0xff).toNat % UInt256.size = 0 at h1
  rw [show (UInt256.ofNat 0xff).toNat = 255 by decide] at h1
  have hle : Nat.land (setTrueWord old).toNat 255 ≤ 255 := Nat.and_le_right
  rw [Nat.mod_eq_of_lt (by unfold UInt256.size; omega)] at h1
  have hb : (Nat.land (setTrueWord old).toNat 255).testBit 0 = true := by
    change ((setTrueWord old).toNat &&& 255).testBit 0 = true
    rw [Nat.testBit_and, setTrueWord_testBit0]; decide
  rw [h1] at hb
  simp at hb

theorem storageWord_insert_ne {st : Storage} {s s' v : UInt256} (h : s ≠ s') :
    (st.insert s' v).getD s ⟨0⟩ = st.getD s ⟨0⟩ := by
  rw [Std.ExtTreeMap.getD_insert, if_neg]
  intro hc; exact h (Std.LawfulEqCmp.eq_of_compare hc).symm

theorem storageWord_insert_self {st : Storage} {s v : UInt256} :
    (st.insert s v).getD s ⟨0⟩ = v := by
  rw [Std.ExtTreeMap.getD_insert, if_pos (Std.ReflCmp.compare_self)]

/-- The effect of a successful run on the abstraction: exactly `absNext`. -/
theorem post_view {σ σ' : AccountMap} {I : ExecutionEnv} (hpost : ExpirePost σ σ' I)
    (hnc : NoSlotCollision (argHash I)) :
    (∀ H, (viewOf σ' I.codeOwner).sentAt H = (viewOf σ I.codeOwner).sentAt H) ∧
    (∀ H, (viewOf σ' I.codeOwner).expired H ↔ (absNext (viewOf σ I.codeOwner) (argHash I)).expired H) := by
  have hs : ∀ s, storageWord σ' I.codeOwner s =
      ((σ.getD I.codeOwner default).storage.insert (expiredSlot (argHash I))
        (setTrueWord (storageWord σ I.codeOwner (expiredSlot (argHash I))))).getD s ⟨0⟩ := by
    intro s; unfold storageWord; rw [hpost.self_storage]; rfl
  refine ⟨fun H => ?_, fun H => ?_⟩
  · show (storageWord σ' I.codeOwner (sentAtSlot H)).toNat = (storageWord σ I.codeOwner (sentAtSlot H)).toNat
    rw [hs, storageWord_insert_ne (hnc.2 H)]; rfl
  · show UInt256.land (storageWord σ' I.codeOwner (expiredSlot H)) (UInt256.ofNat 0xff) ≠ ⟨0⟩ ↔
      (UInt256.land (storageWord σ I.codeOwner (expiredSlot H)) (UInt256.ofNat 0xff) ≠ ⟨0⟩ ∨
        H = argHash I)
    by_cases hH : H = argHash I
    · subst hH
      rw [hs, storageWord_insert_self]
      exact ⟨fun _ => Or.inr rfl, fun _ => land_setTrueWord_ff _⟩
    · rw [hs, storageWord_insert_ne (hnc.1 H hH)]
      exact ⟨fun h => Or.inl h, fun h => h.resolve_right hH⟩

/-- **Refinement (soundness).** A successful run of the compiled `expireMessage(H, t)` is an
    abstract `expire ⟨z, H, t⟩` step: the abstract guard held before (with `deposits` read as
    `DepositCall`), the abstract state after is `absNext`, and no other account's storage
    changed. -/
theorem refines_expire {σ σ₀ σ' : AccountMap} {A A' : Substate} {I : ExecutionEnv}
    {g g' : UInt256} {o : ByteArray} {vO vS : AccountAddress}
    (hcode : I.code = l2tol2Runtime) (hsel : selectorWord I = expireSelector)
    (hcds : I.calldata.size < 2 ^ 256)
    (hO : ReturnsAddress σ σ₀ I otherMessengerCalldata vO)
    (hX : ReturnsAddress σ σ₀ I xDomainMessageSenderCalldata vS)
    (hnc : NoSlotCollision (argHash I))
    (hres : Ξ σ σ₀ g A I = .ok (.success (σ', g', A') o)) :
    absGuard P_contract (DepositCall I vO vS) (viewOf σ I.codeOwner) (argHash I) (argTime I).toNat ∧
    (∀ H, (viewOf σ' I.codeOwner).sentAt H = (viewOf σ I.codeOwner).sentAt H) ∧
    (∀ H, (viewOf σ' I.codeOwner).expired H ↔
      (absNext (viewOf σ I.codeOwner) (argHash I)).expired H) ∧
    (∀ a, a ≠ I.codeOwner → (σ'.getD a default).storage = (σ.getD a default).storage) := by
  obtain ⟨_, hc, hpost, _⟩ := expireMessage_success hcode hsel hcds hO hX hres
  refine ⟨⟨⟨hc.callerIsL2cdm, hc.senderIsOther⟩, ?_, hc.expired⟩, (post_view hpost hnc).1,
    (post_view hpost hnc).2, hpost.other_storage⟩
  intro h0
  apply hc.wasSent
  exact Words.ext_iff.mpr h0

/-- **Refinement (completeness).** If the abstract `expire ⟨z, H, t⟩` is enabled (guard with
    `deposits` read as `DepositCall`) and the concrete side conditions hold (no ETH, well-formed
    calldata, `sentAt + P` fits in 256 bits, not a static call), the compiled code performs the
    abstract step, unless it runs out of gas or one of its calls to the L2CrossDomainMessenger
    fails. -/
theorem complete_expire {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : UInt256}
    {vO vS : AccountAddress}
    (hcode : I.code = l2tol2Runtime) (hsel : selectorWord I = expireSelector)
    (hcds : I.calldata.size < 2 ^ 256)
    (hO : ReturnsAddress σ σ₀ I otherMessengerCalldata vO)
    (hX : ReturnsAddress σ σ₀ I xDomainMessageSenderCalldata vS)
    (hnc : NoSlotCollision (argHash I))
    (hguard : absGuard P_contract (DepositCall I vO vS) (viewOf σ I.codeOwner) (argHash I)
      (argTime I).toNat)
    (hval : I.weiValue = ⟨0⟩) (hlen : 68 ≤ I.calldata.size ∧ I.calldata.size < 2 ^ 255 + 4)
    (hov : (sentAt σ I).toNat + P_contract < 2 ^ 256) (hperm : I.perm = true) :
    Ξ σ σ₀ g A I = .error .OutOfGass ∨
    (∃ σ' g' A', Ξ σ σ₀ g A I = .ok (.success (σ', g', A') ByteArray.empty) ∧
      (∀ H, (viewOf σ' I.codeOwner).sentAt H = (viewOf σ I.codeOwner).sentAt H) ∧
      (∀ H, (viewOf σ' I.codeOwner).expired H ↔
        (absNext (viewOf σ I.codeOwner) (argHash I)).expired H)) ∨
    ((∃ g' o, Ξ σ σ₀ g A I = .ok (.revert g' o)) ∧ CallFailed σ σ₀ I) := by
  obtain ⟨⟨hsrc, hSO⟩, hs0, hexp⟩ := hguard
  have hc : ExpireConds σ I vO vS :=
    ⟨hval, hlen, hsrc, hSO, fun h => hs0 (by
      show (storageWord σ I.codeOwner (sentAtSlot (argHash I))).toNat = 0
      have : sentAt σ I = ⟨0⟩ := h
      unfold sentAt at this; rw [this]; rfl), hov, hexp⟩
  rcases expireMessage_complete (g := g) (A := A) hcode hsel hcds hO hX hc hperm with
    h | ⟨σ', g', A', h, hpost⟩ | h
  · exact Or.inl h
  · exact Or.inr (Or.inl ⟨σ', g', A', h, (post_view hpost hnc).1, (post_view hpost hnc).2⟩)
  · exact Or.inr (Or.inr h)

end ExpiryEvm.Abstract
