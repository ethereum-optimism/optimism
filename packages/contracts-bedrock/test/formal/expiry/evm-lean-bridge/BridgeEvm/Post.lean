import BridgeEvm.Refund
import Reasoning.WordArithmetic

/-! # The bridge's storage right after the `SSTORE` of a successful `refundETH`

`RefundRun` records the account map right after the `SSTORE` as
`storedMap σ₁ I H = sstoreAccountMap I.codeOwner σ₁ (refundedSlot H) (setTrueWord old)`, where
`σ₁` is the map after the `expiredMessages` static call (same storage and code as `σ`).
`storedMap_post` spells it out: exactly the slot `refunded[H]` changes, to a word whose low byte
is 1 (so `refunded[H]` reads true). EVMLean's `SSTORE` (EquiVM's `sstoreAccountMap`) is a no-op
on an account absent from the map; presence of the executing account in `σ₁` is *derived* from
the hypothesis that it has non-empty code in `σ` (static calls preserve code, proved).

`refundETH_store` packages the whole state chain of a successful run:
`σ` → (store) `σ₂` → (`mint` call) `σ₃` → (SafeSend `CREATE`) `σ'`.
What the `mint` call and the creation do to the bridge's storage is NOT constrained here: it is
whatever the callee code / init code does (no frame assumption is made or claimed). -/

namespace BridgeEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach Mem

theorem setTrueWord_ne_zero (old : UInt256) : setTrueWord old ≠ ⟨0⟩ := fun h =>
  absurd (u256_lor_eq_zero_left h) (by decide : UInt256.ofNat 1 ≠ ⟨0⟩)

theorem setTrueWord_ne_default (old : UInt256) : (setTrueWord old == default) = false := by
  have hne := setTrueWord_ne_zero old
  generalize setTrueWord old = w at hne ⊢
  cases w with
  | mk v =>
    show (v == (default : UInt256).val) = false
    cases hb : (v == (default : UInt256).val)
    · rfl
    · exfalso; apply hne
      have : v = (default : UInt256).val := by simpa using hb
      rw [this]; rfl

/-- The low byte of `setTrueWord old` is 1: Solidity reads `refunded[H]` as true. -/
theorem setTrueWord_lowByte (old : UInt256) :
    UInt256.land (UInt256.ofNat 255) (setTrueWord old) = UInt256.ofNat 1 := by
  apply Words.ext_iff.mpr
  unfold setTrueWord
  rw [Words.land_toNat, Words.lor_toNat,
    show (UInt256.ofNat
      115792089237316195423570985008687907853269984665640564039457584007913129639680) =
      UInt256.ofNat (2 ^ 256 - 2 ^ 8) by norm_num,
    u256_land_high_mask_toNat _ 8 (by norm_num)]
  rw [show (UInt256.ofNat 255).toNat = 2 ^ 8 - 1 by decide, show (UInt256.ofNat 1).toNat = 1 by decide,
    Nat.land_comm, Nat.and_two_pow_sub_one_eq_mod, Nat.lor_comm, Nat.mul_comm,
    ← Nat.two_pow_add_eq_or_of_lt (by norm_num : 1 < 2 ^ 8), Nat.mul_add_mod]

theorem getD_of_get? {σ : AccountMap} {a : AccountAddress} {acc : Account}
    (h : σ.get? a = some acc) : σ.getD a default = acc := by
  have this : σ[a]? = some acc := by rw [← Std.ExtTreeMap.get?_eq_getElem?]; exact h
  rw [Std.ExtTreeMap.getD_eq_getD_getElem?, this]; rfl

/-- An account with non-empty code in `σ` is present in every map with the same code. -/
theorem present_of_code {σ σ₁ : AccountMap} {a : AccountAddress} (hcd : accountCodeStateEq σ σ₁)
    (hc : (σ.getD a default).code.size ≠ 0) : ∃ acc, σ₁.get? a = some acc := by
  cases h : σ₁.get? a with
  | some acc => exact ⟨acc, rfl⟩
  | none =>
    exfalso; apply hc
    have this : σ₁[a]? = none := by rw [← Std.ExtTreeMap.get?_eq_getElem?]; exact h
    have hd : σ₁.getD a default = default := by
      rw [Std.ExtTreeMap.getD_eq_getD_getElem?, this]; rfl
    rw [hcd a, hd]; rfl

/-- **The store.** If the executing account has non-empty code in `σ` (a deployed contract; for
    the predeploy, the proxy), then in `storedMap σ₁ I H` (`σ₁`: same storage and code as `σ`) the
    bridge's storage is `σ`'s with exactly `refunded[H]` replaced by `setTrueWord old`, every
    other account's storage is `σ`'s, all code is `σ`'s, and `refunded[H]` now reads true. -/
theorem storedMap_post {σ σ₁ : AccountMap} {I : ExecutionEnv} {H : UInt256}
    (hst : accountStorageStateEq σ σ₁) (hcd : accountCodeStateEq σ σ₁)
    (hcode : (σ.getD I.codeOwner default).code.size ≠ 0) :
    ((storedMap σ₁ I H).getD I.codeOwner default).storage =
      (σ.getD I.codeOwner default).storage.insert (refundedSlot H)
        (setTrueWord (refundedWord σ I H)) ∧
    (∀ a, a ≠ I.codeOwner → ((storedMap σ₁ I H).getD a default).storage = (σ.getD a default).storage) ∧
    accountCodeStateEq σ (storedMap σ₁ I H) ∧
    UInt256.land (UInt256.ofNat 255) (refundedWord (storedMap σ₁ I H) I H) = UInt256.ofNat 1 := by
  obtain ⟨acc, hex⟩ := present_of_code hcd hcode
  have hrw : refundedWord σ₁ I H = refundedWord σ I H := by
    unfold refundedWord; exact storageWord_eq_of_storageEq hst _ _
  have hfin : storedMap σ₁ I H = σ₁.insert I.codeOwner
      { acc with storage := acc.storage.insert (refundedSlot H) (setTrueWord (refundedWord σ₁ I H)) } := by
    unfold storedMap sstoreAccountMap
    rw [hex]
    simp only [Option.option, setTrueWord_ne_default]
    rfl
  have hgacc : σ₁.getD I.codeOwner default = acc := getD_of_get? hex
  have hget : ∀ a, (storedMap σ₁ I H).getD a default =
      if a = I.codeOwner then
        { acc with storage := acc.storage.insert (refundedSlot H) (setTrueWord (refundedWord σ₁ I H)) }
      else σ₁.getD a default := by
    intro a
    rw [hfin, Std.ExtTreeMap.getD_insert]
    by_cases ha : a = I.codeOwner
    · subst ha; simp
    · rw [if_neg ha, if_neg]
      intro hc; apply ha; exact (Std.LawfulEqCmp.eq_of_compare hc).symm
  have hself : ((storedMap σ₁ I H).getD I.codeOwner default).storage =
      (σ.getD I.codeOwner default).storage.insert (refundedSlot H) (setTrueWord (refundedWord σ I H)) := by
    rw [hget, if_pos rfl]
    show acc.storage.insert _ _ = _
    rw [hrw, (hst I.codeOwner).1, hgacc]
  refine ⟨hself, ?_, ?_, ?_⟩
  · intro a ha
    rw [hget, if_neg ha, (hst a).1]
  · intro a
    rw [hget]
    split
    · next ha => subst ha; show _ = acc.code; rw [hcd I.codeOwner, hgacc]
    · rw [hcd a]
  · unfold refundedWord storageWord
    rw [hself, Std.ExtTreeMap.getD_insert_self]
    exact setTrueWord_lowByte _

/-- **The state chain of a successful `refundETH`.** From `RefundRun` and non-empty code at the
    executing account: there are maps `σ₂` (right after the store) and `σ₃` (after `mint`) such
    that `σ₂` is `σ` with exactly `refunded[H]` set to true (other storage and all code as in `σ`),
    the `mint(amount)` call from `σ₂` succeeded with result map `σ₃`, and the SafeSend creation
    from `σ₃` succeeded with result map `σ'`. The bridge's storage in `σ'` is not constrained
    (it depends on the callee and init code). -/
theorem refundETH_store {σ σ₀ σ' : AccountMap} {I : ExecutionEnv}
    (hrun : RefundRun σ σ₀ I σ') (hcode : (σ.getD I.codeOwner default).code.size ≠ 0) :
    ∃ σ₂ σ₃,
      ((σ₂.getD I.codeOwner default).storage =
        (σ.getD I.codeOwner default).storage.insert (refundedSlot (refundHash I))
          (setTrueWord (refundedWord σ I (refundHash I)))) ∧
      (∀ a, a ≠ I.codeOwner → (σ₂.getD a default).storage = (σ.getD a default).storage) ∧
      accountCodeStateEq σ σ₂ ∧
      UInt256.land (UInt256.ofNat 255) (refundedWord σ₂ I (refundHash I)) = UInt256.ofNat 1 ∧
      (∃ oM, CallTo σ₀ I ethLiq (mintCalldata (argAmount I)) σ₂ σ₃ true oM) ∧
      (∃ x rd', CreateStep I σ₀ σ₃ (argAmount I) (safeSendDeploy (argFrom I)) x σ' rd' ∧
        x ≠ UInt256.ofNat 0) := by
  obtain ⟨_, _, σ₁, oE, _, _, _, hst, hcd, _, _, σ₃, oM, hm, x, rd', hcs, hx⟩ := hrun
  obtain ⟨h1, h2, h3, h4⟩ := storedMap_post (H := refundHash I) hst hcd hcode
  exact ⟨_, σ₃, h1, h2, h3, h4, ⟨oM, hm⟩, ⟨x, rd', hcs, hx⟩⟩

end BridgeEvm
