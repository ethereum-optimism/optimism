import BridgeEvm.Refund
import Reasoning.WordArithmetic

/-! # The bridge's storage after a successful `refundETH`

`RefundRun` records the bridge's account map right after the `SSTORE` as
`storedMap σ₁ I H = sstoreAccountMap I.codeOwner σ₁ (refundedSlot H) (setTrueWord old)`.
`storedMap_post` spells it out: exactly the slot `refunded[H]` changes, to a word whose low byte
is 1 (so `refunded[H]` reads true and a second `refundETH` for the same `H` fails the
`AlreadyRefunded` check). EVMLean's `SSTORE` (EquiVM's `sstoreAccountMap`) does nothing on an
account that is absent from the account map, so the lemma assumes the executing account is
present in `σ₁` — always the case for a deployed contract (the bridge proxy has code).

What happens to the bridge's storage *after* the store, during the `mint` call and the SafeSend
`CREATE`, is not constrained by the bytecode proof (it is whatever the callee code does);
`refundETH_bridgeStorage` derives the final storage under two explicit frame hypotheses. -/

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

/-- **The store.** If the executing account is present in `σ₁` (the account map after the
    `expiredMessages` static call, which has `σ`'s storage and code), then in `storedMap σ₁ I H`
    the bridge's storage is `σ`'s with exactly `refunded[H]` replaced by `setTrueWord old`, every
    other account's storage is `σ`'s, all code is `σ`'s, and `refunded[H]` now reads true. -/
theorem storedMap_post {σ σ₁ : AccountMap} {I : ExecutionEnv} {H : UInt256} {acc : Account}
    (hst : accountStorageStateEq σ σ₁) (hcd : accountCodeStateEq σ σ₁)
    (hex : σ₁.get? I.codeOwner = some acc) :
    ((storedMap σ₁ I H).getD I.codeOwner default).storage =
      (σ.getD I.codeOwner default).storage.insert (refundedSlot H)
        (setTrueWord (refundedWord σ I H)) ∧
    (∀ a, a ≠ I.codeOwner → ((storedMap σ₁ I H).getD a default).storage = (σ.getD a default).storage) ∧
    accountCodeStateEq σ (storedMap σ₁ I H) ∧
    UInt256.land (UInt256.ofNat 255) (refundedWord (storedMap σ₁ I H) I H) = UInt256.ofNat 1 := by
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

/-- **Final bridge storage under frame hypotheses.** If successful `mint` calls to ETHLiquidity
    and successful SafeSend creations do not change the bridge's storage (`MintFrame`,
    `CreateFrame` — assumptions about the callee/init code, not proved here), then after a
    successful `refundETH` the bridge's storage is the pre-state's with exactly `refunded[H]`
    set to true. -/
def MintFrame (σ₀ : AccountMap) (I : ExecutionEnv) : Prop :=
  ∀ σc σ' o, CallTo σ₀ I ethLiq (mintCalldata (argAmount I)) σc σ' true o →
    (σ'.getD I.codeOwner default).storage = (σc.getD I.codeOwner default).storage

def CreateFrame (σ₀ : AccountMap) (I : ExecutionEnv) : Prop :=
  ∀ σc x σ' rd', CreateStep I σ₀ σc (argAmount I) (safeSendDeploy (argFrom I)) x σ' rd' →
    x ≠ UInt256.ofNat 0 → (σ'.getD I.codeOwner default).storage = (σc.getD I.codeOwner default).storage

theorem refundETH_bridgeStorage {σ σ₀ σ' : AccountMap} {I : ExecutionEnv}
    (hrun : RefundRun σ σ₀ I σ') (hmint : MintFrame σ₀ I) (hcreate : CreateFrame σ₀ I)
    (hex : ∀ σ₁, accountStorageStateEq σ σ₁ → (σ₁.get? I.codeOwner).isSome) :
    (σ'.getD I.codeOwner default).storage =
      (σ.getD I.codeOwner default).storage.insert (refundedSlot (refundHash I))
        (setTrueWord (refundedWord σ I (refundHash I))) := by
  obtain ⟨_, _, σ₁, oE, _, _, _, hst, hcd, _, _, σ₃, oM, hm, x, rd', hcs, hx⟩ := hrun
  obtain ⟨acc, hacc⟩ := Option.isSome_iff_exists.mp (hex σ₁ hst)
  rw [hcreate _ _ _ _ hcs hx, hmint _ _ _ hm]
  exact (storedMap_post hst hcd hacc).1

end BridgeEvm
