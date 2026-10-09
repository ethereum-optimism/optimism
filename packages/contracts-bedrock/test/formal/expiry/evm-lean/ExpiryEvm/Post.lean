import ExpiryEvm.TraceStore
import Reasoning.WordArithmetic

/-! # The post-state of a successful run satisfies `ExpirePost` -/

namespace ExpiryEvm

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

theorem getD_of_get? {σ : AccountMap} {a : AccountAddress} {acc : Account}
    (h : σ.get? a = some acc) : σ.getD a default = acc := by
  have this : σ[a]? = some acc := by rw [← Std.ExtTreeMap.get?_eq_getElem?]; exact h
  rw [Std.ExtTreeMap.getD_eq_getD_getElem?, this]; rfl

theorem finalMap_post {σ σ₂ : AccountMap} {I : ExecutionEnv}
    (hst2 : accountStorageStateEq σ σ₂) (hcd2 : accountCodeStateEq σ σ₂)
    (hae : ¬ AlreadyExpired σ I) (hs0 : sentAt σ I ≠ ⟨0⟩) : ExpirePost σ (finalMap σ₂ I) I := by
  have hS : storageWord σ₂ I.codeOwner (sentAtSlot (argHash I)) = sentAt σ I :=
    storageWord_eq_of_storageEq hst2 _ _
  obtain ⟨acc, hacc⟩ : ∃ acc, σ₂.get? I.codeOwner = some acc := by
    cases h : σ₂.get? I.codeOwner with
    | some acc => exact ⟨acc, rfl⟩
    | none =>
      exfalso; apply hs0; rw [← hS, ← optWord_eq, h]; rfl
  have hfin : finalMap σ₂ I = σ₂.insert I.codeOwner
      { acc with storage := (acc.storage.insert (expiredSlot (argHash I))
          (setTrueWord (storageWord σ₂ I.codeOwner (expiredSlot (argHash I))))) } := by
    unfold finalMap sstoreAccountMap
    rw [hacc]
    simp only [Option.option, setTrueWord_ne_default]
    rfl
  have hgacc : σ₂.getD I.codeOwner default = acc := getD_of_get? hacc
  have hget : ∀ a, (finalMap σ₂ I).getD a default =
      if a = I.codeOwner then { acc with storage := (acc.storage.insert (expiredSlot (argHash I))
          (setTrueWord (storageWord σ₂ I.codeOwner (expiredSlot (argHash I))))) }
      else σ₂.getD a default := by
    intro a
    rw [hfin, Std.ExtTreeMap.getD_insert]
    by_cases ha : a = I.codeOwner
    · subst ha; simp
    · rw [if_neg ha, if_neg]
      intro hc; apply ha; exact (Std.LawfulEqCmp.eq_of_compare hc).symm
  refine ⟨Or.inr ⟨hae, ?_⟩, ?_, ?_, ?_⟩
  · rw [hget, if_pos rfl]
    show acc.storage.insert _ _ = _
    rw [storageWord_eq_of_storageEq hst2, (hst2 I.codeOwner).1, hgacc]
  · intro a ha
    rw [hget, if_neg ha, (hst2 a).1]
  · intro a
    rw [hget]
    split
    · next ha => subst ha; show acc.tstorage = _; rw [(hst2 I.codeOwner).2, hgacc]
    · rw [(hst2 a).2]
  · intro a
    rw [hget]
    split
    · next ha => subst ha; show _ = acc.code; rw [hcd2 I.codeOwner, hgacc]
    · rw [hcd2 a]

/-- The early return leaves the account map of the static calls in place, which matches `σ` on
    every storage, transient storage and code. -/
theorem alreadyExpired_post {σ σ₂ : AccountMap} {I : ExecutionEnv}
    (hst2 : accountStorageStateEq σ σ₂) (hcd2 : accountCodeStateEq σ σ₂)
    (hae : AlreadyExpired σ I) : ExpirePost σ σ₂ I := by
  refine ⟨Or.inl ⟨hae, ?_⟩, ?_, ?_, hcd2⟩
  · rw [(hst2 I.codeOwner).1]
  · intro a _; rw [(hst2 a).1]
  · intro a; rw [(hst2 a).2]

end ExpiryEvm
