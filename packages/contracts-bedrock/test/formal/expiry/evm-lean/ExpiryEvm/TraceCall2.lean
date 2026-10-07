import ExpiryEvm.TraceCall1

/-! # Trace segment 3: the `xDomainMessageSender()` static call and the sender check
(pc 2321 → pc 2552) -/

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
    {aw : UInt256} {k C : ℕ} {t H : UInt256} {vO vS : AccountAddress}
    (hX : ReturnsAddress σ σ₀ I xDomainMessageSenderCalldata vS)
    (hst1 : accountStorageStateEq σ σ₁) (hcd1 : accountCodeStateEq σ σ₁)
    (h : RD l2tol2Runtime I g (initState σ σ₀ g A I) (UInt256.ofNat 2321)
        [addrWord vO, t, H, UInt256.ofNat 587, expireSelector]
        (wordsMem [⟨0⟩, ⟨0⟩, ⟨160⟩, ⟨0⟩, addrWord vO]) aw (abiAddress vO) σ₁ k C) :
    (RDrev l2tol2Runtime g (initState σ σ₀ g A I) ∧ (vS ≠ vO ∨ CallFailed σ σ₀ I)) ∨
    (vS = vO ∧ ∃ σ₂ aw' k' C', accountStorageStateEq σ σ₂ ∧ accountCodeStateEq σ σ₂ ∧
      RD l2tol2Runtime I g (initState σ σ₀ g A I) (UInt256.ofNat 2552)
        [t, H, UInt256.ofNat 587, expireSelector]
        (wordsMem [⟨0⟩, ⟨0⟩, ⟨192⟩, ⟨0⟩, addrWord vO, addrWord vS]) aw' (abiAddress vS) σ₂ k' C') := by
  have r2 := l2tol2_block_2321 (by simp) h
  have hA : memLoad (UInt256.ofNat 64) (wordsMem [⟨0⟩, ⟨0⟩, ⟨160⟩, ⟨0⟩, addrWord vO]) = ⟨160⟩ :=
    memLoad_wordsMem _ 2 (by simp) _ (by decide)
  have hB : (UInt256.toByteArray selSenderWord).write 0 (wordsMem [⟨0⟩, ⟨0⟩, ⟨160⟩, ⟨0⟩, addrWord vO])
      (⟨160⟩ : UInt256).toNat 32 = wordsMem [⟨0⟩, ⟨0⟩, ⟨160⟩, ⟨0⟩, addrWord vO, selSenderWord] :=
    wordsMem_write_end _ _ _ rfl
  have hC : memLoad (UInt256.ofNat 64) (wordsMem [⟨0⟩, ⟨0⟩, ⟨160⟩, ⟨0⟩, addrWord vO, selSenderWord]) =
      ⟨160⟩ := memLoad_wordsMem _ 2 (by simp) _ (by decide)
  simp only [l2tol2_block_2321_stack, l2tol2_block_2321_memory, hA, hB, hC] at r2
  have hdec : decode l2tol2Runtime (UInt256.ofNat 2421) = some (.STATICCALL, .none) := by
    native_decide
  by_cases hd : I.depth.val < 1024
  · obtain ⟨σ', z, o, A_in, callGas, k3, C3, ⟨g'', A', hΘ⟩, r3, _hosize⟩ :=
      RD.solcStaticcall r2 hdec hd (by simp)
    have hin : (wordsMem [⟨0⟩, ⟨0⟩, ⟨160⟩, ⟨0⟩, addrWord vO, selSenderWord]).readWithPadding
        (⟨160⟩ : UInt256).toNat ((UInt256.ofNat 4 + ⟨160⟩).sub ⟨160⟩).toNat =
        xDomainMessageSenderCalldata := by
      rw [show ((UInt256.ofNat 4 + ⟨160⟩).sub ⟨160⟩).toNat = 4 by decide,
        wordsMem_read4 _ 5 (by simp) _ (by decide)]
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
      refine ⟨?_, Or.inr (Or.inr ⟨_, σ₁, σ', o, Or.inr rfl, hst1, hcall⟩)⟩
      simp only [Bool.false_eq_true, if_false] at r3
      have r3' := RD.normalizePC (pc' := UInt256.ofNat 2422) r3 (by decide)
      have r4 := l2tol2_block_2422_fallthrough (by simp) (by decide) r3'
      exact l2tol2_block_2429 (by simp [l2tol2_block_2422_fallthrough_stack]) r4
    | true =>
      have ho : o = abiAddress vS := hX σ₁ σ' true o hst1 hcd1 hcall rfl
      subst ho
      simp only [if_true] at r3
      have hW : (abiAddress vS).write 0 (wordsMem [⟨0⟩, ⟨0⟩, ⟨160⟩, ⟨0⟩, addrWord vO, selSenderWord])
          (⟨160⟩ : UInt256).toNat (min (UInt256.ofNat 32) (UInt256.ofNat (abiAddress vS).size)).toNat =
          wordsMem [⟨0⟩, ⟨0⟩, ⟨160⟩, ⟨0⟩, addrWord vO, addrWord vS] := by
        rw [abiAddress_size, show (min (UInt256.ofNat 32) (UInt256.ofNat 32)).toNat = 32 by decide]
        exact wordsMem_write _ 5 (by simp) _ _ (by decide)
      rw [hW] at r3
      have r3' := RD.normalizePC (pc' := UInt256.ofNat 2422) r3 (by decide)
      have r4 := l2tol2_block_2422_taken (by simp) (by decide) (by jump_dest) r3'
      simp only [l2tol2_block_2422_taken_stack] at r4
      have r5 := l2tol2_block_2436 (by simp) (by jump_dest) r4
      have hD : memLoad (UInt256.ofNat 64)
          (wordsMem [⟨0⟩, ⟨0⟩, ⟨160⟩, ⟨0⟩, addrWord vO, addrWord vS]) = ⟨160⟩ :=
        memLoad_wordsMem _ 2 (by simp) _ (by decide)
      have hE : (⟨160⟩ : UInt256) + UInt256.land (UInt256.ofNat 32 + UInt256.ofNat 31)
          (UInt256.lnot (UInt256.ofNat 31)) = ⟨192⟩ := by decide
      have hF : (UInt256.toByteArray (⟨192⟩ : UInt256)).write 0
          (wordsMem [⟨0⟩, ⟨0⟩, ⟨160⟩, ⟨0⟩, addrWord vO, addrWord vS]) (UInt256.ofNat 64).toNat 32 =
          wordsMem [⟨0⟩, ⟨0⟩, ⟨192⟩, ⟨0⟩, addrWord vO, addrWord vS] :=
        wordsMem_write _ 2 (by simp) _ _ (by decide)
      simp only [l2tol2_block_2436_stack, l2tol2_block_2436_memory, hD, abiAddress_size, hE,
        hF] at r5
      have r6 := l2tol2_block_5266_taken (by simp) (by decide) (by jump_dest) r5
      simp only [l2tol2_block_5266_taken_stack] at r6
      have r7 := l2tol2_block_5282 (by simp) (by jump_dest) r6
      have hG : memLoad ⟨160⟩ (wordsMem [⟨0⟩, ⟨0⟩, ⟨192⟩, ⟨0⟩, addrWord vO, addrWord vS]) =
          addrWord vS :=
        memLoad_wordsMem _ 5 (by simp) _ (by decide)
      simp only [l2tol2_block_5282_stack, hG] at r7
      have r8 := l2tol2_block_4389_taken (by simp)
        (by rw [addrWord_clean]; exact Words.eq_ne0.mpr rfl) (by jump_dest) r7
      have r9 := l2tol2_block_4422 (by simp) (by jump_dest) r8
      simp only [l2tol2_block_4422_stack] at r9
      have r10 := l2tol2_block_4752 (by simp) (by jump_dest) r9
      simp only [l2tol2_block_4752_stack] at r10
      have r11 := l2tol2_block_2472 (by simp) r10
      simp only [l2tol2_block_2472_stack, addrWord_clean_left] at r11
      by_cases heq : vS = vO
      · subst heq
        have r12 := l2tol2_block_2497_taken (by simp)
          (Words.isZero_ne0.mpr (Words.isZero_eq0.mpr (Words.eq_ne0.mpr rfl))) (by jump_dest) r11
        simp only [l2tol2_block_2497_taken_stack] at r12
        right
        exact ⟨rfl, σ', _, _, _, hst, hcd, r12⟩
      · left
        refine ⟨?_, Or.inl heq⟩
        have hne : addrWord vS ≠ addrWord vO := fun h => heq (addrWord_inj.mp h)
        have r12 := l2tol2_block_2497_fallthrough (by simp)
          (Words.isZero_eq0.mpr (fun h0 => (Words.isZero_ne0.mpr (Words.eq_eq0.mpr hne)) h0)) r11
        exact l2tol2_block_2503 (by simp [l2tol2_block_2497_fallthrough_stack]) r12
  · have hd' : I.depth = 1024 := by
      apply Fin.ext; have := I.depth.isLt; omega
    obtain ⟨k3, C3, r3⟩ := RD.solcStaticcallDepthLimit r2 hdec hd' (by simp)
    left
    refine ⟨?_, Or.inr (Or.inl (by rw [hd']; rfl))⟩
    have r4 := l2tol2_block_2422_fallthrough (by simp) (by decide) r3
    exact l2tol2_block_2429 (by simp [l2tol2_block_2422_fallthrough_stack]) r4

end ExpiryEvm
