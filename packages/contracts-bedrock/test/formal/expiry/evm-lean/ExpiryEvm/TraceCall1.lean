import ExpiryEvm.TraceEntry
import Ethereum.Theory.StaticStorage

/-! # Trace segment 2: caller check and the `otherMessenger()` static call (pc 1713 → pc 1872) -/

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
    (h : RD l2tol2Runtime I g (initState σ σ₀ g A I) (UInt256.ofNat 1713)
      [t, H, UInt256.ofNat 567, expireSelector] (wordsMem [⟨0⟩, ⟨0⟩, ⟨128⟩]) aw ByteArray.empty σ k C) :
    (RDrev l2tol2Runtime g (initState σ σ₀ g A I) ∧ (I.source ≠ l2cdm ∨ CallFailed σ σ₀ I)) ∨
    (I.source = l2cdm ∧ ∃ σ₁ o₁ j aw' k' C', accountStorageStateEq σ σ₁ ∧ accountCodeStateEq σ σ₁ ∧
      L2cdmStaticCall σ₀ I otherMessengerCalldata σ σ₁ true o₁ ∧ 5 ≤ j ∧ j < 2 ^ 140 ∧
      RD l2tol2Runtime I g (initState σ σ₀ g A I) (UInt256.ofNat 1872)
        [addrWord vO, t, H, UInt256.ofNat 567, expireSelector]
        (wordsMem [⟨0⟩, ⟨0⟩, UInt256.ofNat (32 * j), ⟨0⟩, addrWord vO]) aw' o₁ σ₁ k' C') := by
  by_cases hc : UInt256.isZero (UInt256.eq (UInt256.ofNat 376793390874373408599387495934666716005045108743)
      (UInt256.ofNat I.source.val)) = UInt256.ofNat 0
  · -- caller is the L2CrossDomainMessenger
    have hsrc : I.source = l2cdm := by
      rw [← source_eq_iff]; exact (Words.eq_ne0.mp (Words.isZero_eq0.mp hc)).symm
    obtain ⟨aw1, k1, C1, r1⟩ := l2tol2_block_1713_fallthrough_packed (by simp) hc h
    simp only [l2tol2_block_1713_fallthrough_stack] at r1
    have r2 := l2tol2_block_1743 (by simp) r1
    have hA : memLoad (UInt256.ofNat 64) (wordsMem [⟨0⟩, ⟨0⟩, ⟨128⟩]) = ⟨128⟩ :=
      memLoad_wordsMem _ 2 (by simp) _ (by decide)
    have hB : (UInt256.toByteArray selOtherWord).write 0 (wordsMem [⟨0⟩, ⟨0⟩, ⟨128⟩]) (⟨128⟩ : UInt256).toNat 32 =
        wordsMem [⟨0⟩, ⟨0⟩, ⟨128⟩, ⟨0⟩, selOtherWord] :=
      wordsMem_write_gap1 _ _ _ (by decide)
    have hC : memLoad (UInt256.ofNat 64) (wordsMem [⟨0⟩, ⟨0⟩, ⟨128⟩, ⟨0⟩, selOtherWord]) = ⟨128⟩ :=
      memLoad_wordsMem _ 2 (by simp) _ (by decide)
    simp only [l2tol2_block_1743_stack, l2tol2_block_1743_memory, hA, hB, hC] at r2
    have hdec : decode l2tol2Runtime (UInt256.ofNat 1821) = some (.STATICCALL, .none) := by
      evm_kdecide
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
        refine ⟨?_, Or.inr (Or.inr (Or.inl ⟨σ', o, hcall⟩))⟩
        simp only [Bool.false_eq_true, if_false] at r3
        have r4 := l2tol2_block_1822_fallthrough (by simp) (by decide) r3
        exact l2tol2_block_1829 (by simp [l2tol2_block_1822_fallthrough_stack]) r4
      | true =>
        obtain ⟨hsz, hpre⟩ := hO σ σ' true o (accountStorageStateEq_refl σ)
          (accountCodeStateEq_refl σ) hcall rfl
        have hlt : o.size < 2 ^ 138 :=
          Theta_returnData_size_lt_2pow138_of_eq _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ hΘ
            (by show 4 ≤ _; norm_num [Ethereum.EVM.maxReturnDataSizeByGas,
              Ethereum.EVM.maxReturnDataWordsByGas])
        simp only [if_true] at r3
        have hW : o.write 0 (wordsMem [⟨0⟩, ⟨0⟩, ⟨128⟩, ⟨0⟩, selOtherWord])
            (⟨128⟩ : UInt256).toNat (min (UInt256.ofNat 32) (UInt256.ofNat o.size)).toNat =
            wordsMem [⟨0⟩, ⟨0⟩, ⟨128⟩, ⟨0⟩, addrWord vO] := by
          rw [Words.min32_toNat _ hsz (by omega),
            write_prefix32 o _ (addrWord vO) _ hsz hpre (by rw [wordsMem_size]; decide)]
          exact wordsMem_write _ 4 (by simp) _ _ (by decide)
        rw [hW] at r3
        have r3' := RD.normalizePC (pc' := UInt256.ofNat 1822) r3 (by decide)
        have r4 := l2tol2_block_1822_taken (by simp) (by decide) (by kjump_dest) r3'
        simp only [l2tol2_block_1822_taken_stack] at r4
        have r5 := l2tol2_block_1836 (by simp) (by kjump_dest) r4
        set j := 4 + (o.size + 31) / 32 with hj
        have hD : memLoad (UInt256.ofNat 64) (wordsMem [⟨0⟩, ⟨0⟩, ⟨128⟩, ⟨0⟩, addrWord vO]) = ⟨128⟩ :=
          memLoad_wordsMem _ 2 (by simp) _ (by decide)
        have hE : (⟨128⟩ : UInt256) + UInt256.land (UInt256.ofNat o.size + UInt256.ofNat 31)
            (UInt256.lnot (UInt256.ofNat 31)) = UInt256.ofNat (32 * j) := by
          rw [Words.round_up _ (by omega), show (⟨128⟩ : UInt256) = UInt256.ofNat 128 from rfl,
            Words.ofNat_add _ _ (by omega)]
          exact congrArg UInt256.ofNat (by omega)
        have hF : (UInt256.toByteArray (UInt256.ofNat (32 * j))).write 0
            (wordsMem [⟨0⟩, ⟨0⟩, ⟨128⟩, ⟨0⟩, addrWord vO]) (UInt256.ofNat 64).toNat 32 =
            wordsMem [⟨0⟩, ⟨0⟩, UInt256.ofNat (32 * j), ⟨0⟩, addrWord vO] :=
          wordsMem_write _ 2 (by simp) _ _ (by decide)
        simp only [l2tol2_block_1836_stack, l2tol2_block_1836_memory, hD, hE, hF] at r5
        have hlen := Words.retlen_ok o.size hsz (by omega)
        rw [← Words.add_sub_cancel' 128 o.size (by omega)] at hlen
        have r6 := l2tol2_block_4576_taken (by simp) hlen (by kjump_dest) r5
        simp only [l2tol2_block_4576_taken_stack] at r6
        have r7 := l2tol2_block_4592 (by simp) (by kjump_dest) r6
        have hG : memLoad (UInt256.ofNat 128) (wordsMem [⟨0⟩, ⟨0⟩, UInt256.ofNat (32 * j), ⟨0⟩, addrWord vO]) =
            addrWord vO :=
          memLoad_wordsMem _ 4 (by simp) _ (by decide)
        simp only [l2tol2_block_4592_stack, hG] at r7
        have r8 := l2tol2_block_4055_taken (by simp)
          (by rw [addrWord_clean]; exact Words.eq_ne0.mpr rfl) (by kjump_dest) r7
        have r9 := l2tol2_block_4088 (by simp) (by kjump_dest) r8
        simp only [l2tol2_block_4088_stack] at r9
        have r10 := l2tol2_block_4025 (by simp) (by kjump_dest) r9
        simp only [l2tol2_block_4025_stack] at r10
        right
        exact ⟨hsrc, σ', o, j, _, _, _, hst, hcd, hcall, by omega, by omega, r10⟩
    · have hd' : I.depth = 1024 := by
        apply Fin.ext; have := I.depth.isLt; simp only [Fin.val_ofNat] at *; omega
      obtain ⟨k3, C3, r3⟩ := RD.solcStaticcallDepthLimit r2 hdec hd' (by simp)
      left
      refine ⟨?_, Or.inr (Or.inl (by rw [hd']; rfl))⟩
      have r4 := l2tol2_block_1822_fallthrough (by simp) (by decide) r3
      exact l2tol2_block_1829 (by simp [l2tol2_block_1822_fallthrough_stack]) r4
  · -- caller is not the L2CrossDomainMessenger: NotOtherMessenger
    left
    refine ⟨?_, Or.inl ?_⟩
    · have r1 := l2tol2_block_1713_taken (by simp) hc (by kjump_dest) h
      simp only [l2tol2_block_1713_taken_stack] at r1
      have r2 := l2tol2_block_2048_fallthrough (by simp) (Words.isZero_eq0.mpr hc) r1
      exact l2tol2_block_2054 (by simp [l2tol2_block_2048_fallthrough_stack]) r2
    · intro hsrc
      apply hc
      apply Words.isZero_eq0.mpr
      exact Words.eq_ne0.mpr ((source_eq_iff I).mpr hsrc).symm

end ExpiryEvm
