import L1cdmEvm.Tails
import Ethereum.Theory.StaticStorage

/-! # Outer trace, segments 6–8: `portal.ethLockbox()`, `.authorizedPortals(callerPortal)` (check
(b)) and `msg.sender.xDomainMessageSender()` (check (c)) (pc 2440 → 2973) -/

namespace L1cdmEvm

set_option maxRecDepth 100000

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach l1cdmBlocks L1cdmEvm.SymMem

theorem div_exp0 (x : UInt256) : UInt256.div x (UInt256.exp (UInt256.ofNat 256) (UInt256.ofNat 0)) = x := by
  rw [show UInt256.exp (UInt256.ofNat 256) (UInt256.ofNat 0) = UInt256.ofNat 1 by decide +kernel]
  apply u_ext
  unfold UInt256.div
  simp only [UInt256.toNat]
  show _ / 1 = _
  exact Nat.div_one _

theorem seg_lockbox {σ σ1 σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256}
    {m : ByteArray} {Bw aw : UInt256} {rdata : ByteArray} {k C : ℕ} {t H wP : UInt256}
    (hb : CallBound σ σ₀ I (addrOf (storageWord σ I.codeOwner portalSlot)) ethLockboxCd)
    (hst : accountStorageStateEq σ σ1) (hcd : accountCodeStateEq σ σ1)
    (hF : Fmp m Bw) (hB : 96 ≤ Bw.toNat) (hBb : Bw.toNat < 2 ^ 40)
    (h : RD l1cdmRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 2440)
      [UInt256.ofNat 0, wP, UInt256.ofNat I.source.val, t, H, UInt256.ofNat 766, relaySelector] m aw rdata σ1 k C) :
    RDrev l1cdmRuntime g (initState σ σ₀ g A I) ∨
    (∃ wL, CallReturned σ σ₀ I (addrOf (storageWord σ I.codeOwner portalSlot)) ethLockboxCd wL ∧ CleanAddr wL ∧ ∃ σ2 m2 Bw2 aw2 rd2 k2 C2, accountStorageStateEq σ σ2 ∧
      accountCodeStateEq σ σ2 ∧ Fmp m2 Bw2 ∧ 96 ≤ Bw2.toNat ∧ Bw2.toNat < Bw.toNat + 2 ^ 33 ∧
      RD l1cdmRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 2585)
        [wL, wP, UInt256.ofNat I.source.val, t, H, UInt256.ofNat 766, relaySelector] m2 aw2 rd2 σ2 k2 C2) := by
  obtain ⟨gw, k1, C1, r1⟩ := l1cdm_block_2440 (by simp) h
  simp only [l1cdm_block_2440_stack, l1cdm_block_2440_memory, fmp_load hF (UInt256.ofNat 64) rfl,
    optWord_eq, storageWord_eq_of_storageEq hst, div_exp0] at r1
  obtain ⟨hF1, hin⟩ := callmem_sel hF hB
    (UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 3062023236)) (UInt256.ofNat 224))
    (0xb682c444 * 2 ^ 224) 0xb682c444 (by decide) (by decide +kernel) _ rfl
  generalize hm1 : (UInt256.toByteArray _).write 0 m Bw.toNat 32 = m1 at r1 hF1 hin
  simp only [fmp_load hF1 (UInt256.ofNat 64) rfl, u_sub_add_comm] at r1
  have hdec : decode l1cdmRuntime (UInt256.ofNat 2532) = some (.STATICCALL, .none) := by evm_kdecide
  by_cases hd : I.depth.val < 1024
  swap
  · have hd' : I.depth = 1024 := by
      apply Fin.ext; have := I.depth.isLt; omega
    obtain ⟨k3, C3, r3⟩ := RD.solcStaticcallDepthLimit r1 hdec hd' (by simp)
    left
    exact l1cdm_block_2540 (by simp [l1cdm_block_2533_fallthrough_stack])
      (l1cdm_block_2533_fallthrough (by simp) (by decide) r3)
  obtain ⟨σ', z, o, A_in, callGas, k3, C3, ⟨g'', A', hΘ⟩, r3, _hosize⟩ :=
    RD.solcStaticcall r1 hdec hd (by simp)
  rw [show (UInt256.ofNat 4).toNat = 4 from rfl, hin, ofUInt256_mask_land, ofUInt256_mask_land] at hΘ
  have hcall : StaticCallFrom σ₀ I (addrOf (storageWord σ I.codeOwner portalSlot)) ethLockboxCd σ1 σ' z o :=
    ⟨A_in, callGas, g'', A', hΘ⟩
  have hst' : accountStorageStateEq σ σ' :=
    accountStorageStateEq_trans hst (Theta_static_accountStorageStateEq hΘ.symm)
  have hcd' : accountCodeStateEq σ σ' :=
    accountCodeStateEq_trans hcd (Theta_static_accountCodeStateEq hΘ.symm)
  cases z with
  | false =>
    left
    simp only [Bool.false_eq_true, if_false] at r3
    exact l1cdm_block_2540 (by simp [l1cdm_block_2533_fallthrough_stack])
      (l1cdm_block_2533_fallthrough (by simp) (by decide) r3)
  | true =>
    have hosz := hb σ1 σ' o hst hcd hcall
    have hfirst := firstWord_spec o
    generalize firstWord o = wL at hfirst
    simp only [if_true] at r3
    have r4 := l1cdm_block_2533_taken (by simp) (by decide) (by kjump_dest) r3
    simp only [l1cdm_block_2533_taken_stack] at r4
    have r5 := l1cdm_block_2549 (by simp) (by kjump_dest) r4
    simp only [l1cdm_block_2549_stack, l1cdm_block_2549_memory,
      fmp_load (fmp_keep_out hF1 hB o (by omega)) (UInt256.ofNat 64) rfl] at r5
    have hAC := fun h32 => aftercall hF1 hB (by omega) o wL h32 (by omega) (hfirst h32)
    rcases tail_addr (w := wL) (by simp) (by kjump_dest) hosz (fun h32 => (hAC h32).2.2.1) r5 with
      hrev | ⟨h32, hclean, aw6, k6, C6, r6⟩
    · exact Or.inl hrev
    obtain ⟨_, hF3, _, hnb⟩ := hAC h32
    exact Or.inr ⟨wL, ⟨σ1, σ', o, hst, hcd, hcall, h32, hfirst h32⟩, hclean, σ', _, _, _, _, _, _, hst', hcd', hF3, by omega, by omega, r6⟩

theorem seg_auth {σ σ1 σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256}
    {m : ByteArray} {Bw aw : UInt256} {rdata : ByteArray} {k C : ℕ} {t H wP wL : UInt256}
    (hb : CallBound σ σ₀ I (addrOf wL) (authorizedPortalsCd wP)) (hP : CleanAddr wP)
    (hst : accountStorageStateEq σ σ1) (hcd : accountCodeStateEq σ σ1)
    (hF : Fmp m Bw) (hB : 96 ≤ Bw.toNat) (hBb : Bw.toNat < 2 ^ 40)
    (h : RD l1cdmRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 2585)
      [wL, wP, UInt256.ofNat I.source.val, t, H, UInt256.ofNat 766, relaySelector] m aw rdata σ1 k C) :
    RDrev l1cdmRuntime g (initState σ σ₀ g A I) ∨
    (∃ wA, CallReturned σ σ₀ I (addrOf wL) (authorizedPortalsCd wP) wA ∧ wA = UInt256.ofNat 1 ∧ ∃ σ2 m2 Bw2 aw2 rd2 k2 C2, accountStorageStateEq σ σ2 ∧
      accountCodeStateEq σ σ2 ∧ Fmp m2 Bw2 ∧ 96 ≤ Bw2.toNat ∧ Bw2.toNat < Bw.toNat + 2 ^ 33 ∧
      RD l1cdmRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 2739)
        [UInt256.ofNat 0, wP, UInt256.ofNat I.source.val, t, H, UInt256.ofNat 766, relaySelector] m2 aw2 rd2 σ2 k2 C2) := by
  have r1 := l1cdm_block_2585 (by simp) h
  simp only [l1cdm_block_2585_stack, l1cdm_block_2585_memory, fmp_load hF (UInt256.ofNat 64) rfl,
    land_mask_clean hP] at r1
  have hoff4 : (Bw + UInt256.ofNat 4).toNat = Bw.toNat + 4 :=
    u_toNat_add _ _ (by rw [u_toNat_ofNat (by norm_num)]; omega)
  obtain ⟨hF1, hin⟩ := callmem_sel_arg hF hB (by omega) _ (0x0fd11077 * 2 ^ 224) 0x0fd11077 rfl wP
    (by decide +kernel) _ rfl _ hoff4
  generalize hm1 : (UInt256.toByteArray _).write 0 ((UInt256.toByteArray _).write 0 m Bw.toNat 32) _ 32 = m1
    at r1 hF1 hin
  simp only [fmp_load hF1 (UInt256.ofNat 64) rfl, u_sub_add_comm] at r1
  have hdec : decode l1cdmRuntime (UInt256.ofNat 2678) = some (.STATICCALL, .none) := by evm_kdecide
  by_cases hd : I.depth.val < 1024
  swap
  · have hd' : I.depth = 1024 := by
      apply Fin.ext; have := I.depth.isLt; omega
    obtain ⟨k3, C3, r3⟩ := RD.solcStaticcallDepthLimit r1 hdec hd' (by simp)
    left
    exact l1cdm_block_2686 (by simp [l1cdm_block_2679_fallthrough_stack])
      (l1cdm_block_2679_fallthrough (by simp) (by decide) r3)
  obtain ⟨σ', z, o, A_in, callGas, k3, C3, ⟨g'', A', hΘ⟩, r3, _hosize⟩ :=
    RD.solcStaticcall r1 hdec hd (by simp)
  rw [show (UInt256.ofNat 36).toNat = 36 from rfl, hin, ofUInt256_mask_land] at hΘ
  have hcall : StaticCallFrom σ₀ I (addrOf wL) (authorizedPortalsCd wP) σ1 σ' z o :=
    ⟨A_in, callGas, g'', A', hΘ⟩
  have hst' : accountStorageStateEq σ σ' :=
    accountStorageStateEq_trans hst (Theta_static_accountStorageStateEq hΘ.symm)
  have hcd' : accountCodeStateEq σ σ' :=
    accountCodeStateEq_trans hcd (Theta_static_accountCodeStateEq hΘ.symm)
  cases z with
  | false =>
    left
    simp only [Bool.false_eq_true, if_false] at r3
    exact l1cdm_block_2686 (by simp [l1cdm_block_2679_fallthrough_stack])
      (l1cdm_block_2679_fallthrough (by simp) (by decide) r3)
  | true =>
    have hosz := hb σ1 σ' o hst hcd hcall
    have hfirst := firstWord_spec o
    generalize firstWord o = wA at hfirst
    simp only [if_true] at r3
    have r4 := l1cdm_block_2679_taken (by simp) (by decide) (by kjump_dest) r3
    simp only [l1cdm_block_2679_taken_stack] at r4
    have r5 := l1cdm_block_2695 (by simp) (by kjump_dest) r4
    simp only [l1cdm_block_2695_stack, l1cdm_block_2695_memory,
      fmp_load (fmp_keep_out hF1 hB o (by omega)) (UInt256.ofNat 64) rfl] at r5
    have hAC := fun h32 => aftercall hF1 hB (by omega) o wA h32 (by omega) (hfirst h32)
    rcases tail_bool (w := wA) (by simp) (by kjump_dest) hosz (fun h32 => (hAC h32).2.2.1) r5 with
      hrev | ⟨h32, hbool, aw6, k6, C6, r6⟩
    · exact Or.inl hrev
    obtain ⟨_, hF3, _, hnb⟩ := hAC h32
    have r7 := l1cdm_block_2731 (by simp) r6
    simp only [l1cdm_block_2731_stack] at r7
    by_cases hz : wA = UInt256.ofNat 0
    · left
      exact rev_2665 (by simp) (by rw [hz]; decide) r7
    have hz' : UInt256.isZero wA = UInt256.ofNat 0 := Words.isZero_eq0.mpr hz
    have r8 := l1cdm_block_2733_fallthrough (by simp) hz' r7
    rw [hz'] at r8
    exact Or.inr ⟨wA, ⟨σ1, σ', o, hst, hcd, hcall, h32, hfirst h32⟩, bool_true hbool hz, σ', _, _, _, _, _, _, hst', hcd', hF3, by omega, by omega, r8⟩

theorem seg_xsender {σ σ1 σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256}
    {m : ByteArray} {Bw aw : UInt256} {rdata : ByteArray} {k C : ℕ} {t H wP : UInt256}
    (hb : CallBound σ σ₀ I I.source xDomainMessageSenderCd)
    (hst : accountStorageStateEq σ σ1) (hcd : accountCodeStateEq σ σ1)
    (hF : Fmp m Bw) (hB : 96 ≤ Bw.toNat) (hBb : Bw.toNat < 2 ^ 40)
    (h : RD l1cdmRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 2739)
      [UInt256.ofNat 0, wP, UInt256.ofNat I.source.val, t, H, UInt256.ofNat 766, relaySelector] m aw rdata σ1 k C) :
    RDrev l1cdmRuntime g (initState σ σ₀ g A I) ∨
    (∃ wX, CallReturned σ σ₀ I I.source xDomainMessageSenderCd wX ∧ wX = exporterWord ∧ ∃ σ2 m2 Bw2 aw2 rd2 k2 C2, accountStorageStateEq σ σ2 ∧
      accountCodeStateEq σ σ2 ∧ Fmp m2 Bw2 ∧ 96 ≤ Bw2.toNat ∧ Bw2.toNat < Bw.toNat + 2 ^ 33 ∧
      RD l1cdmRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 2973)
        [wP, UInt256.ofNat I.source.val, t, H, UInt256.ofNat 766, relaySelector] m2 aw2 rd2 σ2 k2 C2) := by
  have r1 := l1cdm_block_2739 (by simp) h
  simp only [l1cdm_block_2739_stack, l1cdm_block_2739_memory, fmp_load hF (UInt256.ofNat 64) rfl] at r1
  obtain ⟨hF1, hin⟩ := callmem_sel hF hB
    (UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 1848208965)) (UInt256.ofNat 224))
    (0x6e296e45 * 2 ^ 224) 0x6e296e45 (by decide) (by decide +kernel) _ rfl
  generalize hm1 : (UInt256.toByteArray _).write 0 m Bw.toNat 32 = m1 at r1 hF1 hin
  simp only [fmp_load hF1 (UInt256.ofNat 64) rfl, u_sub_add_comm] at r1
  have hdec : decode l1cdmRuntime (UInt256.ofNat 2840) = some (.STATICCALL, .none) := by evm_kdecide
  by_cases hd : I.depth.val < 1024
  swap
  · have hd' : I.depth = 1024 := by
      apply Fin.ext; have := I.depth.isLt; omega
    obtain ⟨k3, C3, r3⟩ := RD.solcStaticcallDepthLimit r1 hdec hd' (by simp)
    left
    exact l1cdm_block_2848 (by simp [l1cdm_block_2841_fallthrough_stack])
      (l1cdm_block_2841_fallthrough (by simp) (by decide) r3)
  obtain ⟨σ', z, o, A_in, callGas, k3, C3, ⟨g'', A', hΘ⟩, r3, _hosize⟩ :=
    RD.solcStaticcall r1 hdec hd (by simp)
  rw [show (UInt256.ofNat 4).toNat = 4 from rfl, hin, ofUInt256_mask_land,
    AccountAddress.ofUInt256_ofNat I.source] at hΘ
  have hcall : StaticCallFrom σ₀ I I.source xDomainMessageSenderCd σ1 σ' z o :=
    ⟨A_in, callGas, g'', A', hΘ⟩
  have hst' : accountStorageStateEq σ σ' :=
    accountStorageStateEq_trans hst (Theta_static_accountStorageStateEq hΘ.symm)
  have hcd' : accountCodeStateEq σ σ' :=
    accountCodeStateEq_trans hcd (Theta_static_accountCodeStateEq hΘ.symm)
  cases z with
  | false =>
    left
    simp only [Bool.false_eq_true, if_false] at r3
    exact l1cdm_block_2848 (by simp [l1cdm_block_2841_fallthrough_stack])
      (l1cdm_block_2841_fallthrough (by simp) (by decide) r3)
  | true =>
    have hosz := hb σ1 σ' o hst hcd hcall
    have hfirst := firstWord_spec o
    generalize firstWord o = wX at hfirst
    simp only [if_true] at r3
    have r4 := l1cdm_block_2841_taken (by simp) (by decide) (by kjump_dest) r3
    simp only [l1cdm_block_2841_taken_stack] at r4
    have r5 := l1cdm_block_2857 (by simp) (by kjump_dest) r4
    simp only [l1cdm_block_2857_stack, l1cdm_block_2857_memory,
      fmp_load (fmp_keep_out hF1 hB o (by omega)) (UInt256.ofNat 64) rfl] at r5
    have hAC := fun h32 => aftercall hF1 hB (by omega) o wX h32 (by omega) (hfirst h32)
    rcases tail_addr (w := wX) (by simp) (by kjump_dest) hosz (fun h32 => (hAC h32).2.2.1) r5 with
      hrev | ⟨h32, hclean, aw6, k6, C6, r6⟩
    · exact Or.inl hrev
    obtain ⟨_, hF3, _, hnb⟩ := hAC h32
    have r7 := l1cdm_block_2893 (by simp) r6
    simp only [l1cdm_block_2893_stack] at r7
    by_cases heq : UInt256.isZero (UInt256.eq (UInt256.land
        (UInt256.ofNat 1461501637330902918203684832716283019655932542975) wX)
        (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975)
          (UInt256.ofNat 376793390874373408599387495934666716005045108784))) = UInt256.ofNat 0
    · have r8 := l1cdm_block_2918_taken (by simp) (by rw [heq]; decide) (by kjump_dest) r7
      simp only [l1cdm_block_2918_taken_stack] at r8
      have hwx : wX = exporterWord := by
        have := Words.eq_ne0.mp (Words.isZero_eq0.mp heq)
        rw [land_mask_clean hclean] at this
        rw [this]; decide
      exact Or.inr ⟨wX, ⟨σ1, σ', o, hst, hcd, hcall, h32, hfirst h32⟩, hwx, σ', _, _, _, _, _, _, hst', hcd', hF3, by omega, by omega, r8⟩
    · left
      exact rev_2850 (by simp) heq r7

end L1cdmEvm
