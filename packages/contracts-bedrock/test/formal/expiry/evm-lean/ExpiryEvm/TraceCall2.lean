import ExpiryEvm.TraceCall1

/-! # Trace segment 3: the `xDomainMessageSender()` static call and the sender check
(pc 1872 → pc 2103) -/

namespace ExpiryEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach l2tol2Blocks Mem

/-- The selector word solc writes for `xDomainMessageSender()`. -/
abbrev selSenderWord : UInt256 :=
  UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 1848208965))
    (UInt256.ofNat 224)

theorem selSender_bytes :
    (UInt256.toByteArray selSenderWord).extract 0 4 = xDomainMessageSenderCalldata := by
  decide +kernel

theorem addrWord_clean_left (v : AccountAddress) :
    UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (addrWord v) =
      addrWord v :=
  solcAddrMask_clean_left (by rw [addrWord_toNat]; exact v.isLt)

theorem addrWord_inj {v w : AccountAddress} : addrWord v = addrWord w ↔ v = w := by
  constructor
  · intro h
    have := congrArg UInt256.toNat h
    rw [addrWord_toNat, addrWord_toNat] at this
    exact Fin.ext this
  · intro h; rw [h]

theorem seg_call2 {σ σ₁ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256}
    {aw : UInt256} {k C : ℕ} {t H : UInt256} {vO vS : AccountAddress} {o₁ : ByteArray} {j : ℕ}
    (hX : ReturnsAddress σ σ₀ I xDomainMessageSenderCalldata vS)
    (hst1 : accountStorageStateEq σ σ₁) (hcd1 : accountCodeStateEq σ σ₁)
    (hcall1 : L2cdmStaticCall σ₀ I otherMessengerCalldata σ σ₁ true o₁)
    (hj5 : 5 ≤ j) (hjb : j < 2 ^ 140)
    (h : RD l2tol2Runtime I g (initState σ σ₀ g A I) (UInt256.ofNat 1872)
        [addrWord vO, t, H, UInt256.ofNat 567, expireSelector]
        (wordsMem [⟨0⟩, ⟨0⟩, UInt256.ofNat (32 * j), ⟨0⟩, addrWord vO]) aw o₁ σ₁ k C) :
    (RDrev l2tol2Runtime g (initState σ σ₀ g A I) ∧ (vS ≠ vO ∨ CallFailed σ σ₀ I)) ∨
    (vS = vO ∧ ∃ σ₂ o₂ rest aw' k' C', accountStorageStateEq σ σ₂ ∧ accountCodeStateEq σ σ₂ ∧
      RD l2tol2Runtime I g (initState σ σ₀ g A I) (UInt256.ofNat 2103)
        [t, H, UInt256.ofNat 567, expireSelector]
        (wordsMem (⟨0⟩ :: ⟨0⟩ :: rest)) aw' o₂ σ₂ k' C') := by
  obtain ⟨kk, rfl⟩ : ∃ kk, j = 5 + kk := ⟨j - 5, by omega⟩
  set P := UInt256.ofNat (32 * (5 + kk)) with hP
  have hPn : P.toNat = 32 * (5 + kk) := Words.toNat_ofNat (by rw [Words.size_eq]; omega)
  have r2 := l2tol2_block_1872 (by simp) h
  have hA : memLoad (UInt256.ofNat 64) (wordsMem [⟨0⟩, ⟨0⟩, P, ⟨0⟩, addrWord vO]) = P :=
    memLoad_wordsMem _ 2 (by simp) _ (by decide)
  have hB : (UInt256.toByteArray selSenderWord).write 0 (wordsMem [⟨0⟩, ⟨0⟩, P, ⟨0⟩, addrWord vO])
      P.toNat 32 = wordsMem (memList P (addrWord vO) kk selSenderWord) :=
    wordsMem_write_gap _ _ _ _ (by rw [hPn]; simp)
  have hC : memLoad (UInt256.ofNat 64) (wordsMem (memList P (addrWord vO) kk selSenderWord)) = P := by
    rw [memLoad_wordsMem _ 2 (by simp only [memList_length]; omega) _ (by decide), memList_get2]
  simp only [l2tol2_block_1872_stack, l2tol2_block_1872_memory, hA, hB, hC] at r2
  have hdec : decode l2tol2Runtime (UInt256.ofNat 1972) = some (.STATICCALL, .none) := by
    native_decide
  by_cases hd : I.depth.val < 1024
  · obtain ⟨σ', z, o, A_in, callGas, k3, C3, ⟨g'', A', hΘ⟩, r3, _hosize⟩ :=
      RD.solcStaticcall r2 hdec hd (by simp)
    have hin : (wordsMem (memList P (addrWord vO) kk selSenderWord)).readWithPadding
        P.toNat ((UInt256.ofNat 4 + P).sub P).toNat = xDomainMessageSenderCalldata := by
      rw [hP, Words.add_sub_cancel_left' 4 _ (by omega), show (UInt256.ofNat 4).toNat = 4 by decide,
        ← hP, wordsMem_read4 _ (5 + kk) (by simp only [memList_length]; omega) _ (by rw [hPn]),
        memList_getLast]
      exact selSender_bytes
    rw [hin, target_l2cdm] at hΘ
    have hcall : L2cdmStaticCall σ₀ I xDomainMessageSenderCalldata σ₁ σ' z o :=
      ⟨A_in, callGas, g'', A', hΘ⟩
    have hst : accountStorageStateEq σ σ' :=
      accountStorageStateEq_trans hst1 (Theta_static_accountStorageStateEq hΘ.symm)
    have hcd : accountCodeStateEq σ σ' :=
      accountCodeStateEq_trans hcd1 (Theta_static_accountCodeStateEq hΘ.symm)
    cases z with
    | false =>
      left
      refine ⟨?_, Or.inr (Or.inr (Or.inr ⟨σ₁, o₁, σ', o, hcall1, hcall⟩))⟩
      simp only [Bool.false_eq_true, if_false] at r3
      have r3' := RD.normalizePC (pc' := UInt256.ofNat 1973) r3 (by decide)
      have r4 := l2tol2_block_1973_fallthrough (by simp) (by decide) r3'
      exact l2tol2_block_1980 (by simp [l2tol2_block_1973_fallthrough_stack]) r4
    | true =>
      obtain ⟨hsz, hpre⟩ := hX σ₁ σ' true o hst1 hcd1 hcall rfl
      have hlt : o.size < 2 ^ 138 :=
        Theta_returnData_size_lt_2pow138_of_eq _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ hΘ
          (by show 4 ≤ _; norm_num [Ethereum.EVM.maxReturnDataSizeByGas,
            Ethereum.EVM.maxReturnDataWordsByGas])
      simp only [if_true] at r3
      have hW : o.write 0 (wordsMem (memList P (addrWord vO) kk selSenderWord))
          P.toNat (min (UInt256.ofNat 32) (UInt256.ofNat o.size)).toNat =
          wordsMem (memList P (addrWord vO) kk (addrWord vS)) := by
        rw [Words.min32_toNat _ hsz (by omega),
          write_prefix32 o _ (addrWord vS) _ hsz hpre
            (by rw [wordsMem_size, memList_length, hPn]; omega),
          wordsMem_write _ (5 + kk) (by simp only [memList_length]; omega) _ _ hPn, memList_setLast]
      rw [hW] at r3
      have r3' := RD.normalizePC (pc' := UInt256.ofNat 1973) r3 (by decide)
      have r4 := l2tol2_block_1973_taken (by simp) (by decide) (by jump_dest) r3'
      simp only [l2tol2_block_1973_taken_stack] at r4
      have r5 := l2tol2_block_1987 (by simp) (by jump_dest) r4
      set P2 := UInt256.ofNat (32 * (5 + kk) + 32 * ((o.size + 31) / 32)) with hP2
      have hD : memLoad (UInt256.ofNat 64) (wordsMem (memList P (addrWord vO) kk (addrWord vS))) = P := by
        rw [memLoad_wordsMem _ 2 (by simp only [memList_length]; omega) _ (by decide), memList_get2]
      have hE : P + UInt256.land (UInt256.ofNat o.size + UInt256.ofNat 31)
          (UInt256.lnot (UInt256.ofNat 31)) = P2 := by
        rw [Words.round_up _ (by omega), hP, Words.ofNat_add _ _ (by omega)]
      have hF : (UInt256.toByteArray P2).write 0
          (wordsMem (memList P (addrWord vO) kk (addrWord vS))) (UInt256.ofNat 64).toNat 32 =
          wordsMem (memList P2 (addrWord vO) kk (addrWord vS)) := by
        rw [wordsMem_write _ 2 (by simp only [memList_length]; omega) _ _ (by decide), memList_set2]
      simp only [l2tol2_block_1987_stack, l2tol2_block_1987_memory, hD, hE, hF] at r5
      have hlen := Words.retlen_ok o.size hsz (by omega)
      rw [← Words.add_sub_cancel' (32 * (5 + kk)) o.size (by omega)] at hlen
      have r6 := l2tol2_block_4576_taken (by simp) hlen (by jump_dest) r5
      simp only [l2tol2_block_4576_taken_stack] at r6
      have r7 := l2tol2_block_4592 (by simp) (by jump_dest) r6
      have hG : memLoad (UInt256.ofNat (32 * (5 + kk))) (wordsMem (memList P2 (addrWord vO) kk (addrWord vS))) = addrWord vS := by
        rw [memLoad_wordsMem _ (5 + kk) (by simp only [memList_length]; omega) _ hPn, memList_getLast]
      simp only [l2tol2_block_4592_stack, hG] at r7
      have r8 := l2tol2_block_4055_taken (by simp)
        (by rw [addrWord_clean]; exact Words.eq_ne0.mpr rfl) (by jump_dest) r7
      have r9 := l2tol2_block_4088 (by simp) (by jump_dest) r8
      simp only [l2tol2_block_4088_stack] at r9
      have r10 := l2tol2_block_4025 (by simp) (by jump_dest) r9
      simp only [l2tol2_block_4025_stack] at r10
      have r11 := l2tol2_block_2023 (by simp) r10
      simp only [l2tol2_block_2023_stack, addrWord_clean_left] at r11
      by_cases heq : vS = vO
      · subst heq
        have r12 := l2tol2_block_2048_taken (by simp)
          (Words.isZero_ne0.mpr (Words.isZero_eq0.mpr (Words.eq_ne0.mpr rfl))) (by jump_dest) r11
        simp only [l2tol2_block_2048_taken_stack] at r12
        right
        exact ⟨rfl, σ', o, _, _, _, _, hst, hcd, r12⟩
      · left
        refine ⟨?_, Or.inl heq⟩
        have hne : addrWord vS ≠ addrWord vO := fun h => heq (addrWord_inj.mp h)
        have r12 := l2tol2_block_2048_fallthrough (by simp)
          (Words.isZero_eq0.mpr (fun h0 => (Words.isZero_ne0.mpr (Words.eq_eq0.mpr hne)) h0)) r11
        exact l2tol2_block_2054 (by simp [l2tol2_block_2048_fallthrough_stack]) r12
  · have hd' : I.depth = 1024 := by
      apply Fin.ext; have := I.depth.isLt; omega
    obtain ⟨k3, C3, r3⟩ := RD.solcStaticcallDepthLimit r2 hdec hd' (by simp)
    left
    refine ⟨?_, Or.inr (Or.inl (by rw [hd']; rfl))⟩
    have r4 := l2tol2_block_1973_fallthrough (by simp) (by decide) r3
    exact l2tol2_block_1980 (by simp [l2tol2_block_1973_fallthrough_stack]) r4

end ExpiryEvm
