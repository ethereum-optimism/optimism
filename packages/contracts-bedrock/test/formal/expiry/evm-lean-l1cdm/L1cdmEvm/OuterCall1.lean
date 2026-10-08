import L1cdmEvm.OuterEntry
import L1cdmEvm.CallMem
import Ethereum.Theory.StaticStorage

/-! # Outer trace, segment 2: `systemConfig.isFeatureEnabled(INTEROP)` (pc 1751 → 1983) -/

namespace L1cdmEvm

set_option maxRecDepth 100000

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach l1cdmBlocks L1cdmEvm.SymMem

theorem seg_feat {σ σ1 σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256}
    {m : ByteArray} {Bw aw : UInt256} {rdata : ByteArray} {k C : ℕ} {t H : UInt256}
    (hb : CallBound σ σ₀ I (addrOf (storageWord σ I.codeOwner systemConfigSlot)) isFeatureEnabledCd)
    (hst : accountStorageStateEq σ σ1) (hcd : accountCodeStateEq σ σ1)
    (hF : Fmp m Bw) (hB : 96 ≤ Bw.toNat) (hBb : Bw.toNat < 2 ^ 40)
    (h : RD l1cdmRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 1751)
      [t, H, UInt256.ofNat 766, relaySelector] m aw rdata σ1 k C) :
    RDrev l1cdmRuntime g (initState σ σ₀ g A I) ∨
    (∃ wF, CallReturned σ σ₀ I (addrOf (storageWord σ I.codeOwner systemConfigSlot)) isFeatureEnabledCd wF ∧ wF = UInt256.ofNat 1 ∧ ∃ σ2 m2 Bw2 aw2 rd2 k2 C2, accountStorageStateEq σ σ2 ∧
      accountCodeStateEq σ σ2 ∧ Fmp m2 Bw2 ∧ 96 ≤ Bw2.toNat ∧ Bw2.toNat < Bw.toNat + 2 ^ 33 ∧
      RD l1cdmRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 1983)
        [t, H, UInt256.ofNat 766, relaySelector] m2 aw2 rd2 σ2 k2 C2) := by
  obtain ⟨gw, k1, C1, r1⟩ := l1cdm_block_1751 (by simp) h
  simp only [l1cdm_block_1751_stack, l1cdm_block_1751_memory, fmp_load hF (UInt256.ofNat 64) rfl] at r1
  have hoff4 : (Bw + UInt256.ofNat 4).toNat = Bw.toNat + 4 :=
    u_toNat_add _ _ (by rw [u_toNat_ofNat (by norm_num)]; omega)
  obtain ⟨hF1, hin⟩ := callmem_sel_arg hF hB (by omega) _
    32423676068182748933135709700898919914331922991074189947669106751585430536192 0x47af267b rfl
    (UInt256.ofNat 33157233633228845956584775098241963410894955598515501468310989770222668349440)
    (by decide +kernel) _ rfl _ hoff4
  generalize hm1 : (UInt256.toByteArray _).write 0 ((UInt256.toByteArray _).write 0 m Bw.toNat 32) _ 32 = m1
    at r1 hF1 hin
  simp only [fmp_load hF1 (UInt256.ofNat 64) rfl, optWord_eq, storageWord_eq_of_storageEq hst,
    u_sub_add_comm] at r1
  have hdec : decode l1cdmRuntime (UInt256.ofNat 1876) = some (.STATICCALL, .none) := by evm_kdecide
  by_cases hd : I.depth.val < 1024
  swap
  · have hd' : I.depth = 1024 := by
      apply Fin.ext; have := I.depth.isLt; simp only [Fin.val_ofNat] at *; omega
    obtain ⟨k3, C3, r3⟩ := RD.solcStaticcallDepthLimit r1 hdec hd' (by simp)
    left
    exact l1cdm_block_1884 (by simp [l1cdm_block_1877_fallthrough_stack])
      (l1cdm_block_1877_fallthrough (by simp) (by decide) r3)
  obtain ⟨σ', z, o, A_in, callGas, k3, C3, ⟨g'', A', hΘ⟩, r3, _hosize⟩ :=
    RD.solcStaticcall r1 hdec hd (by simp)
  rw [show (UInt256.ofNat 36).toNat = 36 from rfl, hin, ofUInt256_land_mask] at hΘ
  have hcall : StaticCallFrom σ₀ I (addrOf (storageWord σ I.codeOwner systemConfigSlot))
      isFeatureEnabledCd σ1 σ' z o := ⟨A_in, callGas, g'', A', hΘ⟩
  have hst' : accountStorageStateEq σ σ' :=
    accountStorageStateEq_trans hst (Theta_static_accountStorageStateEq hΘ.symm)
  have hcd' : accountCodeStateEq σ σ' :=
    accountCodeStateEq_trans hcd (Theta_static_accountCodeStateEq hΘ.symm)
  cases z with
  | false =>
    left
    simp only [Bool.false_eq_true, if_false] at r3
    exact l1cdm_block_1884 (by simp [l1cdm_block_1877_fallthrough_stack])
      (l1cdm_block_1877_fallthrough (by simp) (by decide) r3)
  | true =>
    have hosz := hb σ1 σ' o hst hcd hcall
    have hfirst := firstWord_spec o
    generalize firstWord o = wF at hfirst
    simp only [if_true] at r3
    have r4 := l1cdm_block_1877_taken (by simp) (by decide) (by kjump_dest) r3
    simp only [l1cdm_block_1877_taken_stack] at r4
    have r5 := l1cdm_block_1893 (by simp) (by kjump_dest) r4
    simp only [l1cdm_block_1893_stack, l1cdm_block_1893_memory,
      fmp_load (fmp_keep_out hF1 hB o (by omega)) (UInt256.ofNat 64) rfl] at r5
    by_cases h32 : 32 ≤ o.size
    swap
    · left
      have r6 := l1cdm_block_10184_fallthrough (by simp)
        (by
          rw [u_sub_add_self]
          by_contra hc
          exact h32 ((slt_small _ (by omega)).mp hc)) r5
      exact l1cdm_block_10198 (by simp [l1cdm_block_10184_fallthrough_stack]) r6
    obtain ⟨_, hF3, hload, hnb⟩ := aftercall hF1 hB (by omega) o wF h32 (by omega) (hfirst h32)
    have r6 := l1cdm_block_10184_taken (by simp)
      (by rw [u_sub_add_self]; exact (slt_small _ (by omega)).mpr h32) (by kjump_dest) r5
    simp only [l1cdm_block_10184_taken_stack] at r6
    by_cases hbool : UInt256.eq wF (UInt256.isZero (UInt256.isZero wF)) = UInt256.ofNat 0
    · left
      have r7 := l1cdm_block_10202_fallthrough (by simp) (by rw [hload]; exact hbool) r6
      exact l1cdm_block_10214 (by simp [l1cdm_block_10202_fallthrough_stack]) r7
    have r7 := l1cdm_block_10202_taken (by simp) (by rw [hload]; exact hbool) (by kjump_dest) r6
    simp only [l1cdm_block_10202_taken_stack, hload] at r7
    have r8 := l1cdm_block_8541 (by simp) (by kjump_dest) r7
    simp only [l1cdm_block_8541_stack] at r8
    by_cases hz : wF = UInt256.ofNat 0
    · left
      have r9 := l1cdm_block_1929_fallthrough (by simp) hz r8
      exact l1cdm_block_1934 (by simp [l1cdm_block_1929_fallthrough_stack]) r9
    have r9 := l1cdm_block_1929_taken (by simp) hz (by kjump_dest) r8
    simp only [l1cdm_block_1929_taken_stack] at r9
    right
    refine ⟨wF, ⟨σ1, σ', o, hst, hcd, hcall, h32, hfirst h32⟩, bool_true hbool hz, σ', _, _, _, _, _, _, hst', hcd', hF3, by omega, by omega, r9⟩

end L1cdmEvm
