import L1cdmEvm.Tails
import Ethereum.Theory.StaticStorage

/-! # Outer trace, segment 5: `callerPortal.systemConfig().l1CrossDomainMessenger()` and check (a)
(pc 2237 → 2377) -/

namespace L1cdmEvm

set_option maxRecDepth 100000

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach l1cdmBlocks L1cdmEvm.SymMem

theorem seg_callerMsgr {σ σ1 σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256}
    {m : ByteArray} {Bw aw : UInt256} {rdata : ByteArray} {k C : ℕ} {t H wP wS : UInt256}
    (hb : CallBound σ σ₀ I (addrOf wS) l1CrossDomainMessengerCd)
    (hst : accountStorageStateEq σ σ1) (hcd : accountCodeStateEq σ σ1)
    (hF : Fmp m Bw) (hB : 96 ≤ Bw.toNat) (hBb : Bw.toNat < 2 ^ 40)
    (h : RD l1cdmRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 2237)
      [wS, UInt256.land addrMask (UInt256.ofNat I.source.val), wP, UInt256.ofNat I.source.val, t, H, UInt256.ofNat 766,
        relaySelector] m aw rdata σ1 k C) :
    RDrev l1cdmRuntime g (initState σ σ₀ g A I) ∨
    (∃ wM, CallReturned σ σ₀ I (addrOf wS) l1CrossDomainMessengerCd wM ∧ wM = addrWord I.source ∧ ∃ σ2 m2 Bw2 aw2 rd2 k2 C2, accountStorageStateEq σ σ2 ∧
      accountCodeStateEq σ σ2 ∧ Fmp m2 Bw2 ∧ 96 ≤ Bw2.toNat ∧ Bw2.toNat < Bw.toNat + 2 ^ 33 ∧
      RD l1cdmRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 2377)
        [UInt256.ofNat 0, wP, UInt256.ofNat I.source.val, t, H, UInt256.ofNat 766, relaySelector] m2 aw2 rd2 σ2 k2 C2) := by
  have r1 := l1cdm_block_2237 (by simp) h
  simp only [l1cdm_block_2237_stack, l1cdm_block_2237_memory, fmp_load hF (UInt256.ofNat 64) rfl] at r1
  obtain ⟨hF1, hin⟩ := callmem_sel hF hB
    (UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 2802948201)) (UInt256.ofNat 224))
    (0xa7119869 * 2 ^ 224) 0xa7119869 (by decide) (by decide +kernel) _ rfl
  generalize hm1 : (UInt256.toByteArray _).write 0 m Bw.toNat 32 = m1 at r1 hF1 hin
  simp only [fmp_load hF1 (UInt256.ofNat 64) rfl, u_sub_add_comm] at r1
  have hdec : decode l1cdmRuntime (UInt256.ofNat 2294) = some (.STATICCALL, .none) := by evm_kdecide
  by_cases hd : I.depth.val < 1024
  swap
  · have hd' : I.depth = 1024 := by
      apply Fin.ext; have := I.depth.isLt; omega
    obtain ⟨k3, C3, r3⟩ := RD.solcStaticcallDepthLimit r1 hdec hd' (by simp)
    left
    exact l1cdm_block_2302 (by simp [l1cdm_block_2295_fallthrough_stack])
      (l1cdm_block_2295_fallthrough (by simp) (by decide) r3)
  obtain ⟨σ', z, o, A_in, callGas, k3, C3, ⟨g'', A', hΘ⟩, r3, _hosize⟩ :=
    RD.solcStaticcall r1 hdec hd (by simp)
  rw [show (UInt256.ofNat 4).toNat = 4 from rfl, hin, ofUInt256_mask_land] at hΘ
  have hcall : StaticCallFrom σ₀ I (addrOf wS) l1CrossDomainMessengerCd σ1 σ' z o :=
    ⟨A_in, callGas, g'', A', hΘ⟩
  have hst' : accountStorageStateEq σ σ' :=
    accountStorageStateEq_trans hst (Theta_static_accountStorageStateEq hΘ.symm)
  have hcd' : accountCodeStateEq σ σ' :=
    accountCodeStateEq_trans hcd (Theta_static_accountCodeStateEq hΘ.symm)
  cases z with
  | false =>
    left
    simp only [Bool.false_eq_true, if_false] at r3
    exact l1cdm_block_2302 (by simp [l1cdm_block_2295_fallthrough_stack])
      (l1cdm_block_2295_fallthrough (by simp) (by decide) r3)
  | true =>
    have hosz := hb σ1 σ' o hst hcd hcall
    have hfirst := firstWord_spec o
    generalize firstWord o = wM at hfirst
    simp only [if_true] at r3
    have r4 := l1cdm_block_2295_taken (by simp) (by decide) (by kjump_dest) r3
    simp only [l1cdm_block_2295_taken_stack] at r4
    have r5 := l1cdm_block_2311 (by simp) (by kjump_dest) r4
    simp only [l1cdm_block_2311_stack, l1cdm_block_2311_memory,
      fmp_load (fmp_keep_out hF1 hB o (by omega)) (UInt256.ofNat 64) rfl] at r5
    have hAC := fun h32 => aftercall hF1 hB (by omega) o wM h32 (by omega) (hfirst h32)
    rcases tail_addr (w := wM) (by simp) (by kjump_dest) hosz (fun h32 => (hAC h32).2.2.1) r5 with
      hrev | ⟨h32, hclean, aw6, k6, C6, r6⟩
    · exact Or.inl hrev
    obtain ⟨_, hF3, _, hnb⟩ := hAC h32
    by_cases heq : UInt256.isZero (UInt256.eq (UInt256.land
        (UInt256.ofNat 1461501637330902918203684832716283019655932542975) wM)
        (UInt256.land addrMask (UInt256.ofNat I.source.val))) = UInt256.ofNat 0
    · have r7 := l1cdm_block_2347_fallthrough (by simp) heq r6
      simp only [l1cdm_block_2347_fallthrough_stack, heq] at r7
      have hwm : wM = addrWord I.source := by
        have := Words.eq_ne0.mp (Words.isZero_eq0.mp heq)
        rw [land_mask_clean hclean, show addrMask = UInt256.ofNat
          1461501637330902918203684832716283019655932542975 from rfl,
          land_mask_clean (source_clean I)] at this
        exact this
      exact Or.inr ⟨wM, ⟨σ1, σ', o, hst, hcd, hcall, h32, hfirst h32⟩, hwm, σ', _, _, _, _, _, _, hst', hcd', hF3, by omega, by omega, r7⟩
    · left
      have r7 := l1cdm_block_2347_taken (by simp) heq (by kjump_dest) r6
      simp only [l1cdm_block_2347_taken_stack] at r7
      exact rev_2665 (by simp) heq r7

end L1cdmEvm
