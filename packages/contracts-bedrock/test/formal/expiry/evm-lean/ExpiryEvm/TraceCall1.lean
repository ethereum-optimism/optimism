import ExpiryEvm.TraceEntry
import ExpiryEvm.Blocks.RuntimeBlocks_008
import ExpiryEvm.Blocks.RuntimeBlocks_009
import ExpiryEvm.Blocks.RuntimeBlocks_012
import ExpiryEvm.Blocks.RuntimeBlocks_013
import ExpiryEvm.Blocks.RuntimeBlocks_014
import ExpiryEvm.Blocks.RuntimeBlocks_016
import Ethereum.Theory.StaticStorage

/-! # Trace segment 2: caller check and the `otherMessenger()` static call (pc 2162 → pc 2321) -/

namespace ExpiryEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach l2tol2Blocks Mem

/-- The ABI word of an address. -/
abbrev addrWord (v : AccountAddress) : UInt256 := UInt256.ofNat v.val

theorem addrWord_toNat (v : AccountAddress) : (addrWord v).toNat = v.val :=
  Words.toNat_ofNat (by have := v.isLt; unfold AccountAddress.size at this; unfold UInt256.size; omega)

theorem addrWord_clean (v : AccountAddress) :
    UInt256.land (addrWord v) (UInt256.ofNat 1461501637330902918203684832716283019655932542975) =
      addrWord v :=
  solcAddrMask_clean (by rw [addrWord_toNat]; exact v.isLt)

theorem abiAddress_size (v : AccountAddress) : (abiAddress v).size = 32 := toByteArray_size _

theorem source_eq_iff (I : ExecutionEnv) :
    UInt256.ofNat I.source.val = l2cdmWord ↔ I.source = l2cdm := by
  constructor
  · intro h; rw [← AccountAddress.ofUInt256_ofNat I.source, h]; rfl
  · intro h; rw [h]; decide

/-- The selector word solc writes for `otherMessenger()`. -/
abbrev selOtherWord : UInt256 :=
  UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 3679477120))
    (UInt256.ofNat 224)

theorem selOther_bytes : (UInt256.toByteArray selOtherWord).extract 0 4 = otherMessengerCalldata := by
  decide +kernel

theorem target_l2cdm :
    AccountAddress.ofUInt256 (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975)
      (UInt256.ofNat 376793390874373408599387495934666716005045108743)) = l2cdm := by
  decide

theorem seg_call1 {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256}
    {aw : UInt256} {k C : ℕ} {t H : UInt256} {vO : AccountAddress}
    (hO : ReturnsAddress σ σ₀ I otherMessengerCalldata vO)
    (h : RD l2tol2Runtime I g (initState σ σ₀ g A I) (UInt256.ofNat 2162)
      [t, H, UInt256.ofNat 587, expireSelector] (wordsMem [⟨0⟩, ⟨0⟩, ⟨128⟩]) aw ByteArray.empty σ k C) :
    (RDrev l2tol2Runtime g (initState σ σ₀ g A I) ∧ (I.source ≠ l2cdm ∨ CallFailed σ σ₀ I)) ∨
    (I.source = l2cdm ∧ ∃ σ₁ aw' k' C', accountStorageStateEq σ σ₁ ∧ accountCodeStateEq σ σ₁ ∧
      RD l2tol2Runtime I g (initState σ σ₀ g A I) (UInt256.ofNat 2321)
        [addrWord vO, t, H, UInt256.ofNat 587, expireSelector]
        (wordsMem [⟨0⟩, ⟨0⟩, ⟨160⟩, ⟨0⟩, addrWord vO]) aw' (abiAddress vO) σ₁ k' C') := by
  by_cases hc : UInt256.isZero (UInt256.eq (UInt256.ofNat 376793390874373408599387495934666716005045108743)
      (UInt256.ofNat I.source.val)) = UInt256.ofNat 0
  · -- caller is the L2CrossDomainMessenger
    have hsrc : I.source = l2cdm := by
      rw [← source_eq_iff]; exact (Words.eq_ne0.mp (Words.isZero_eq0.mp hc)).symm
    obtain ⟨aw1, k1, C1, r1⟩ := l2tol2_block_2162_fallthrough_packed (by simp) hc h
    simp only [l2tol2_block_2162_fallthrough_stack] at r1
    have r2 := l2tol2_block_2192 (by simp) r1
    have hA : memLoad (UInt256.ofNat 64) (wordsMem [⟨0⟩, ⟨0⟩, ⟨128⟩]) = ⟨128⟩ :=
      memLoad_wordsMem _ 2 (by simp) _ (by decide)
    have hB : (UInt256.toByteArray selOtherWord).write 0 (wordsMem [⟨0⟩, ⟨0⟩, ⟨128⟩]) (⟨128⟩ : UInt256).toNat 32 =
        wordsMem [⟨0⟩, ⟨0⟩, ⟨128⟩, ⟨0⟩, selOtherWord] :=
      wordsMem_write_gap1 _ _ _ (by decide)
    have hC : memLoad (UInt256.ofNat 64) (wordsMem [⟨0⟩, ⟨0⟩, ⟨128⟩, ⟨0⟩, selOtherWord]) = ⟨128⟩ :=
      memLoad_wordsMem _ 2 (by simp) _ (by decide)
    simp only [l2tol2_block_2192_stack, l2tol2_block_2192_memory, hA, hB, hC] at r2
    have hdec : decode l2tol2Runtime (UInt256.ofNat 2270) = some (.STATICCALL, .none) := by
      native_decide
    by_cases hd : I.depth.val < 1024
    · obtain ⟨σ', z, o, A_in, callGas, k3, C3, ⟨g'', A', hΘ⟩, r3, _hosize⟩ :=
        RD.solcStaticcall r2 hdec hd (by simp)
      have hin : (wordsMem [⟨0⟩, ⟨0⟩, ⟨128⟩, ⟨0⟩, selOtherWord]).readWithPadding
          (⟨128⟩ : UInt256).toNat ((UInt256.ofNat 4 + ⟨128⟩).sub ⟨128⟩).toNat = otherMessengerCalldata := by
        rw [show ((UInt256.ofNat 4 + ⟨128⟩).sub ⟨128⟩).toNat = 4 by decide,
          wordsMem_read4 _ 4 (by simp) _ (by decide)]
        exact selOther_bytes
      rw [hin, target_l2cdm] at hΘ
      have hcall : L2cdmStaticCall σ₀ I otherMessengerCalldata σ σ' z o :=
        ⟨A_in, callGas, g'', A', hΘ⟩
      have hst : accountStorageStateEq σ σ' := Theta_static_accountStorageStateEq hΘ.symm
      have hcd : accountCodeStateEq σ σ' := Theta_static_accountCodeStateEq hΘ.symm
      cases z with
      | false =>
        left
        refine ⟨?_, Or.inr (Or.inr ⟨_, σ, σ', o, Or.inl rfl, accountStorageStateEq_refl σ, hcall⟩)⟩
        simp only [Bool.false_eq_true, if_false] at r3
        have r4 := l2tol2_block_2271_fallthrough (by simp) (by decide) r3
        exact l2tol2_block_2278 (by simp [l2tol2_block_2271_fallthrough_stack]) r4
      | true =>
        have ho : o = abiAddress vO := hO σ σ' true o (accountStorageStateEq_refl σ)
          (accountCodeStateEq_refl σ) hcall rfl
        subst ho
        simp only [if_true] at r3
        have hW : (abiAddress vO).write 0 (wordsMem [⟨0⟩, ⟨0⟩, ⟨128⟩, ⟨0⟩, selOtherWord])
            (⟨128⟩ : UInt256).toNat (min (UInt256.ofNat 32) (UInt256.ofNat (abiAddress vO).size)).toNat =
            wordsMem [⟨0⟩, ⟨0⟩, ⟨128⟩, ⟨0⟩, addrWord vO] := by
          rw [abiAddress_size, show (min (UInt256.ofNat 32) (UInt256.ofNat 32)).toNat = 32 by decide]
          exact wordsMem_write _ 4 (by simp) _ _ (by decide)
        rw [hW] at r3
        have r3' := RD.normalizePC (pc' := UInt256.ofNat 2271) r3 (by decide)
        have r4 := l2tol2_block_2271_taken (by simp) (by decide) (by jump_dest) r3'
        simp only [l2tol2_block_2271_taken_stack] at r4
        have r5 := l2tol2_block_2285 (by simp) (by jump_dest) r4
        have hD : memLoad (UInt256.ofNat 64) (wordsMem [⟨0⟩, ⟨0⟩, ⟨128⟩, ⟨0⟩, addrWord vO]) = ⟨128⟩ :=
          memLoad_wordsMem _ 2 (by simp) _ (by decide)
        have hE : (⟨128⟩ : UInt256) + UInt256.land (UInt256.ofNat 32 + UInt256.ofNat 31)
            (UInt256.lnot (UInt256.ofNat 31)) = ⟨160⟩ := by decide
        have hF : (UInt256.toByteArray (⟨160⟩ : UInt256)).write 0
            (wordsMem [⟨0⟩, ⟨0⟩, ⟨128⟩, ⟨0⟩, addrWord vO]) (UInt256.ofNat 64).toNat 32 =
            wordsMem [⟨0⟩, ⟨0⟩, ⟨160⟩, ⟨0⟩, addrWord vO] :=
          wordsMem_write _ 2 (by simp) _ _ (by decide)
        simp only [l2tol2_block_2285_stack, l2tol2_block_2285_memory, hD, abiAddress_size, hE,
          hF] at r5
        have r6 := l2tol2_block_5266_taken (by simp) (by decide) (by jump_dest) r5
        simp only [l2tol2_block_5266_taken_stack] at r6
        have r7 := l2tol2_block_5282 (by simp) (by jump_dest) r6
        have hG : memLoad ⟨128⟩ (wordsMem [⟨0⟩, ⟨0⟩, ⟨160⟩, ⟨0⟩, addrWord vO]) = addrWord vO :=
          memLoad_wordsMem _ 4 (by simp) _ (by decide)
        simp only [l2tol2_block_5282_stack, hG] at r7
        have r8 := l2tol2_block_4389_taken (by simp)
          (by rw [addrWord_clean]; exact Words.eq_ne0.mpr rfl) (by jump_dest) r7
        have r9 := l2tol2_block_4422 (by simp) (by jump_dest) r8
        simp only [l2tol2_block_4422_stack] at r9
        have r10 := l2tol2_block_4752 (by simp) (by jump_dest) r9
        simp only [l2tol2_block_4752_stack] at r10
        right
        exact ⟨hsrc, σ', _, _, _, hst, hcd, r10⟩
    · have hd' : I.depth = 1024 := by
        apply Fin.ext; have := I.depth.isLt; simp only [Fin.val_ofNat] at *; omega
      obtain ⟨k3, C3, r3⟩ := RD.solcStaticcallDepthLimit r2 hdec hd' (by simp)
      left
      refine ⟨?_, Or.inr (Or.inl (by rw [hd']; rfl))⟩
      have r4 := l2tol2_block_2271_fallthrough (by simp) (by decide) r3
      exact l2tol2_block_2278 (by simp [l2tol2_block_2271_fallthrough_stack]) r4
  · -- caller is not the L2CrossDomainMessenger: NotOtherMessenger
    left
    refine ⟨?_, Or.inl ?_⟩
    · have r1 := l2tol2_block_2162_taken (by simp) hc (by jump_dest) h
      simp only [l2tol2_block_2162_taken_stack] at r1
      have r2 := l2tol2_block_2497_fallthrough (by simp) (Words.isZero_eq0.mpr hc) r1
      exact l2tol2_block_2503 (by simp [l2tol2_block_2497_fallthrough_stack]) r2
    · intro hsrc
      apply hc
      apply Words.isZero_eq0.mpr
      exact Words.eq_ne0.mpr ((source_eq_iff I).mpr hsrc).symm

end ExpiryEvm
