import L1cdmEvm.Tails
import Ethereum.Theory.StaticStorage

/-! # Outer trace, segment 2b: `if (paused()) revert L1CrossDomainMessenger_Paused();` (pc 1983 → 2046)

`paused()` is the messenger's own `public view` function (`return systemConfig.paused();`), compiled
as an internal subroutine at pc 5005 shared with the external `paused()` getter: the call site pushes
the return pc 1991 and jumps there; the subroutine makes the static call to `systemConfig` (slot 254)
with calldata `paused()`, decodes a `bool` (pc 10247) and returns through pc 1746 to 1991, which
reverts unless the word is 0. -/

namespace L1cdmEvm

set_option maxRecDepth 100000

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach l1cdmBlocks L1cdmEvm.SymMem

theorem seg_paused {σ σ1 σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256}
    {m : ByteArray} {Bw aw : UInt256} {rdata : ByteArray} {k C : ℕ} {t H : UInt256}
    (hb : CallBound σ σ₀ I (addrOf (storageWord σ I.codeOwner systemConfigSlot)) pausedCd)
    (hst : accountStorageStateEq σ σ1) (hcd : accountCodeStateEq σ σ1)
    (hF : Fmp m Bw) (hB : 96 ≤ Bw.toNat) (hBb : Bw.toNat < 2 ^ 40)
    (h : RD l1cdmRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 1983)
      [t, H, UInt256.ofNat 766, relaySelector] m aw rdata σ1 k C) :
    RDrev l1cdmRuntime g (initState σ σ₀ g A I) ∨
    (∃ wZ, CallReturned σ σ₀ I (addrOf (storageWord σ I.codeOwner systemConfigSlot)) pausedCd wZ ∧
      wZ = UInt256.ofNat 0 ∧ ∃ σ2 m2 Bw2 aw2 rd2 k2 C2, accountStorageStateEq σ σ2 ∧
      accountCodeStateEq σ σ2 ∧ Fmp m2 Bw2 ∧ 96 ≤ Bw2.toNat ∧ Bw2.toNat < Bw.toNat + 2 ^ 33 ∧
      RD l1cdmRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 2046)
        [t, H, UInt256.ofNat 766, relaySelector] m2 aw2 rd2 σ2 k2 C2) := by
  -- the call site: push the return pc 1991, jump to the `paused()` subroutine
  have r0 := l1cdm_block_1983 (by simp) (by kjump_dest) h
  simp only [l1cdm_block_1983_stack] at r0
  obtain ⟨gw, k1, C1, r1⟩ := l1cdm_block_5005 (by simp) r0
  simp only [l1cdm_block_5005_stack, l1cdm_block_5005_memory, fmp_load hF (UInt256.ofNat 64) rfl] at r1
  obtain ⟨hF1, hin⟩ := callmem_sel hF hB
    (UInt256.ofNat 41880202175123281672023411390868823785620507377596298514233450382794225090560)
    (0x5c975abb * 2 ^ 224) 0x5c975abb (by norm_num) (by decide +kernel) _ rfl
  generalize hm1 : (UInt256.toByteArray _).write 0 m Bw.toNat 32 = m1 at r1 hF1 hin
  have h4 : UInt256.sub Bw Bw + UInt256.ofNat 4 = UInt256.ofNat 4 := by rw [u_sub_self]; decide
  simp only [fmp_load hF1 (UInt256.ofNat 64) rfl, optWord_eq, storageWord_eq_of_storageEq hst, h4] at r1
  have hdec : decode l1cdmRuntime (UInt256.ofNat 5100) = some (.STATICCALL, .none) := by evm_kdecide
  by_cases hd : I.depth.val < 1024
  swap
  · have hd' : I.depth = 1024 := by
      apply Fin.ext; have := I.depth.isLt; omega
    obtain ⟨k3, C3, r3⟩ := RD.solcStaticcallDepthLimit r1 hdec hd' (by simp)
    left
    exact l1cdm_block_5108 (by simp [l1cdm_block_5101_fallthrough_stack])
      (l1cdm_block_5101_fallthrough (by simp) (by decide) r3)
  obtain ⟨σ', z, o, A_in, callGas, k3, C3, ⟨g'', A', hΘ⟩, r3, _hosize⟩ :=
    RD.solcStaticcall r1 hdec hd (by simp)
  rw [show (UInt256.ofNat 4).toNat = 4 from rfl, hin, ofUInt256_mask_land] at hΘ
  have hcall : StaticCallFrom σ₀ I (addrOf (storageWord σ I.codeOwner systemConfigSlot))
      pausedCd σ1 σ' z o := ⟨A_in, callGas, g'', A', hΘ⟩
  have hst' : accountStorageStateEq σ σ' :=
    accountStorageStateEq_trans hst (Theta_static_accountStorageStateEq hΘ.symm)
  have hcd' : accountCodeStateEq σ σ' :=
    accountCodeStateEq_trans hcd (Theta_static_accountCodeStateEq hΘ.symm)
  cases z with
  | false =>
    left
    simp only [Bool.false_eq_true, if_false] at r3
    exact l1cdm_block_5108 (by simp [l1cdm_block_5101_fallthrough_stack])
      (l1cdm_block_5101_fallthrough (by simp) (by decide) r3)
  | true =>
    have hosz := hb σ1 σ' o hst hcd hcall
    have hfirst := firstWord_spec o
    generalize firstWord o = wZ at hfirst
    simp only [if_true] at r3
    have r4 := l1cdm_block_5101_taken (by simp) (by decide) (by kjump_dest) r3
    simp only [l1cdm_block_5101_taken_stack] at r4
    have r5 := l1cdm_block_5117 (by simp) (by kjump_dest) r4
    simp only [l1cdm_block_5117_stack, l1cdm_block_5117_memory,
      fmp_load (fmp_keep_out hF1 hB o (by omega)) (UInt256.ofNat 64) rfl] at r5
    have hAC := fun h32 => aftercall hF1 hB (by omega) o wZ h32 (by omega) (hfirst h32)
    rcases tail_bool (w := wZ) (by simp) (by kjump_dest) hosz (fun h32 => (hAC h32).2.2.1) r5 with
      hrev | ⟨h32, _hbool, aw6, k6, C6, r6⟩
    · exact Or.inl hrev
    obtain ⟨_, hF3, _, hnb⟩ := hAC h32
    -- return from the subroutine to pc 1991 with the decoded word on the stack
    have r7 := l1cdm_block_1746 (by simp) (by kjump_dest) r6
    simp only [l1cdm_block_1746_stack] at r7
    by_cases hz : UInt256.isZero wZ = UInt256.ofNat 0
    · left
      exact l1cdm_block_1997 (by simp [l1cdm_block_1991_fallthrough_stack])
        (l1cdm_block_1991_fallthrough (by simp) hz r7)
    have r8 := l1cdm_block_1991_taken (by simp) hz (by kjump_dest) r7
    simp only [l1cdm_block_1991_taken_stack] at r8
    have hw0 : wZ = UInt256.ofNat 0 := by
      have := Words.isZero_ne0.mp hz; rw [this]; rfl
    right
    exact ⟨wZ, ⟨σ1, σ', o, hst, hcd, hcall, h32, hfirst h32⟩, hw0, σ', _, _, _, _, _, _, hst', hcd',
      hF3, by omega, by omega, r8⟩

end L1cdmEvm
