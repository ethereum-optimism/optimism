import BridgeEvm.TraceExpired
import BridgeEvm.KernelRun

namespace BridgeEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach ethbridgeBlocks Mem

theorem optWord_eq (σ : AccountMap) (a : AccountAddress) (s : UInt256) :
    (σ.get? a |>.option ⟨0⟩ (fun ac => ac.storage.getD s ⟨0⟩)) = storageWord σ a s := by
  unfold storageWord
  cases h : σ.get? a with
  | none =>
    have this : σ[a]? = none := by rw [← Std.ExtTreeMap.get?_eq_getElem?]; exact h
    have hd : σ.getD a default = default := by
      rw [Std.ExtTreeMap.getD_eq_getD_getElem?, this]; rfl
    rw [hd]
    simp only [Option.option]
    show (⟨0⟩ : UInt256) = (∅ : Storage).getD s ⟨0⟩
    simp
  | some ac =>
    have this : σ[a]? = some ac := by rw [← Std.ExtTreeMap.get?_eq_getElem?]; exact h
    have hd : σ.getD a default = ac := by
      rw [Std.ExtTreeMap.getD_eq_getD_getElem?, this]; rfl
    rw [hd]
    rfl

theorem refundedSlot_eq (H : UInt256) :
    UInt256.ofNat (fromByteArrayBigEndian (KEC (ofL (wb H ++ wb (UInt256.ofNat 0))))) = refundedSlot H := by
  unfold refundedSlot solcMappingSlot
  rw [← append_ofL, ← toByteArray_eq_ofL, ← toByteArray_eq_ofL, mappingSlot_single]

/-- The scratch memory solc writes for `refunded[H]` (`mstore(0, H); mstore(32, 0)`). -/
theorem scratch_mem (I : ExecutionEnv) (H : UInt256) (R : List UInt8) (fp : ℕ) (hRl : R.length = 32) :
    (UInt256.ofNat 0).toByteArray.write 0 (H.toByteArray.write 0 (ofL (expMem I H R fp))
      (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32 =
    ofL (wb H ++ (wb (UInt256.ofNat 0) ++ ((expMem I H R fp).drop 64))) := by
  msimpg [hRl]

theorem scratch_slot (L : List UInt8) (H : UInt256) :
    keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) (ofL (wb H ++ (wb (UInt256.ofNat 0) ++ L))) =
      refundedSlot H := by
  msimpg
  exact refundedSlot_eq H

theorem scratch_idem (H : UInt256) (T : List UInt8) :
    (UInt256.ofNat 0).toByteArray.write 0 (H.toByteArray.write 0 (ofL (wb H ++ (wb (UInt256.ofNat 0) ++ T)))
      (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32 = ofL (wb H ++ (wb (UInt256.ofNat 0) ++ T)) := by
  msimpg
  rw [Nat.sub_eq_zero_of_le (by omega)]
  rfl

theorem memLoad_scr (H w : UInt256) (T : List UInt8) (hT : T.take 32 = wb w) (hTl : 32 ≤ T.length) :
    memLoad (UInt256.ofNat 64) (ofL (wb H ++ (wb (UInt256.ofNat 0) ++ T))) = w := by
  msimpg
  rw [hT, ofNat_fromBE_wb]

theorem toNat_fp {fp : ℕ} (h : fp < 2 ^ 200) (c : ℕ) (hc : c < 2 ^ 200) :
    (UInt256.ofNat fp + UInt256.ofNat c).toNat = fp + c := by
  rw [ofNat_add_ofNat, toNat_ofNat_mod, Nat.mod_eq_of_lt (by omega)]

theorem toNat_fp0 {fp : ℕ} (h : fp < 2 ^ 200) : (UInt256.ofNat fp).toNat = fp := by
  rw [toNat_ofNat_mod, Nat.mod_eq_of_lt (by omega)]

theorem toNat_lit {n : ℕ}
    (h : n < 115792089237316195423570985008687907853269984665640564039457584007913129639936) :
    (UInt256.ofNat n).toNat = n := by
  rw [toNat_ofNat_mod, Nat.mod_eq_of_lt h]

/-- The `mint(amount)` call data solc writes at the free-memory pointer. -/
theorem mint_mem (L : List UInt8) (fp : ℕ) (a : UInt256) (hL : L.length ≤ fp + 32) (h : fp < 2 ^ 200) :
    a.toByteArray.write 0 (mintSelWord.toByteArray.write 0 (ofL L) (UInt256.ofNat fp).toNat 32)
      (UInt256.ofNat fp + UInt256.ofNat 4).toNat 32 =
    ofL (pre L fp ++ ((wb mintSelWord).take 4 ++ wb a)) := by
  rw [toNat_fp0 h, write_first _ _ _ hL, toNat_fp h 4 (by norm_num),
    write_rel _ _ _ _ 4 (by simp)]
  congr 2
  simp

theorem expMem_tail_take (I : ExecutionEnv) (H : UInt256) (R : List UInt8) (fp : ℕ) :
    ((expMem I H R fp).drop 64).take 32 = wb (UInt256.ofNat fp) := by
  mnorm

theorem expMem_tail_length (I : ExecutionEnv) (H : UInt256) (R : List UInt8) (fp : ℕ)
    (hRl : R.length = 32) : ((expMem I H R fp).drop 64).length = 616 := by
  simp only [List.length_drop, List.length_append, length_wb, List.length_replicate, List.length_take,
    hRl, Nat.min_def, Nat.reduceLeDiff, ↓reduceIte, Nat.reduceAdd, Nat.reduceSub]

theorem memLoad_pre_scr (H : UInt256) (T Q : List UInt8) (fp : ℕ) (hT : T.take 32 = wb (UInt256.ofNat fp))
    (hTl : T.length = 616) (hfp : 676 ≤ fp) :
    memLoad (UInt256.ofNat 64) (ofL (pre (wb H ++ (wb (UInt256.ofNat 0) ++ T)) fp ++ Q)) =
      UInt256.ofNat fp := by
  rw [memLoad_pre _ _ _ _ (by rw [toNat_ofNat_mod]; omega) (by simp [hTl]; rw [toNat_ofNat_mod]; omega)]
  rw [toNat_ofNat_mod, show 64 % 115792089237316195423570985008687907853269984665640564039457584007913129639936 = 64 by norm_num]
  simp only [List.drop_append, length_wb, List.drop_eq_nil_of_le (show (wb H).length ≤ 64 by simp),
    List.nil_append, Nat.reduceSub]
  rw [drop_wb_of_le _ _ (le_refl _), List.nil_append, List.drop_zero, hT, ofNat_fromBE_wb]

/-- The account map after `refunded[H] = true`. -/
abbrev storedMap (σ₁ : AccountMap) (I : ExecutionEnv) (H : UInt256) : AccountMap :=
  sstoreAccountMap I.codeOwner σ₁ (refundedSlot H) (setTrueWord (refundedWord σ₁ I H))

/-- The memory tail kept from `expMem` (everything from byte 64 on). -/
abbrev scrMem (I : ExecutionEnv) (H : UInt256) (R : List UInt8) (fp : ℕ) : List UInt8 :=
  wb H ++ (wb (UInt256.ofNat 0) ++ (expMem I H R fp).drop 64)

set_option maxHeartbeats 4000000 in
/-- Trace segment 4 (pc 1897 → 2147): the `refunded[H]` check, the `SSTORE`, the `EXTCODESIZE`
    check on ETHLiquidity and the `CALL` `mint(amount)`. Outcomes: revert (already refunded, no
    code at ETHLiquidity, call depth 1024, or the call failed); static-mode violation at the
    `SSTORE` when entered by `STATICCALL`; or the store happened, ETHLiquidity has code, the call
    with exactly `mintCalldata amount` succeeded, and execution continues at pc 2147. -/
theorem seg_store {σ σ₀ σ₁ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256}
    {aw : UInt256} {k C : ℕ} {H : UInt256} {R : List UInt8} {fp : ℕ} {o : ByteArray}
    (hRl : R.length = 32) (hfp : 676 ≤ fp) (hfpb : fp < 2 ^ 200)
    (h : RD ethbridgeRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 1897)
        (bodyStack I H) (ofL (expMem I H R fp)) aw o σ₁ k C) :
    RDrev ethbridgeRuntime g (initState σ σ₀ g A I) ∨
    (I.perm = false ∧ UInt256.land (UInt256.ofNat 255) (refundedWord σ₁ I H) = ⟨0⟩ ∧
      RDstatic ethbridgeRuntime g (initState σ σ₀ g A I)) ∨
    (I.perm = true ∧ UInt256.land (UInt256.ofNat 255) (refundedWord σ₁ I H) = ⟨0⟩ ∧
      extCodeSizeWord (storedMap σ₁ I H) ethLiqWord ≠ ⟨0⟩ ∧
      ∃ σ₃ oM, CallTo σ₀ I ethLiq (mintCalldata (argAmount I)) (storedMap σ₁ I H) σ₃ true oM ∧
      ∃ aw' k' C', RD ethbridgeRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 2147)
        [UInt256.isZero (⟨1⟩ : UInt256), UInt256.ofNat 36 + UInt256.ofNat fp, UInt256.ofNat 2691771752, ethLiqWord, H, argAmount I,
          argTo I, argFrom I, argNonce I, argDest I, UInt256.ofNat 127, refundSelector]
        (ofL (pre (scrMem I H R fp) fp ++ ((wb mintSelWord).take 4 ++ wb (argAmount I))))
        aw' oM σ₃ k' C') := by
  have hS := scratch_mem I H R fp hRl
  have hT := expMem_tail_take I H R fp
  have hTl := expMem_tail_length I H R fp hRl
  show _ ∨ _ ∨ (_ ∧ _ ∧ _ ∧ ∃ σ₃ oM, _ ∧ ∃ aw' k' C', RD ethbridgeRuntime I g (initState σ σ₀ g A I)
    (UInt256.ofNat 2147) _ (ofL (pre (wb H ++ (wb (UInt256.ofNat 0) ++ (expMem I H R fp).drop 64)) fp ++ _)) aw' oM σ₃ k' C')
  generalize (expMem I H R fp).drop 64 = T at hS hT hTl
  by_cases href : UInt256.isZero (UInt256.land (UInt256.ofNat 255) (refundedWord σ₁ I H)) = UInt256.ofNat 0
  · left
    obtain ⟨_, _, r1⟩ := ethbridge_block_1897_fallthrough (by simp)
      (by rw [hS, scratch_slot, optWord_eq]; exact href) h
    exact ethbridge_block_1921 (by simp) r1
  have hrf : UInt256.land (UInt256.ofNat 255) (refundedWord σ₁ I H) = ⟨0⟩ := Words.isZero_ne0.mp href
  obtain ⟨_, _, _, r1⟩ := ethbridge_block_1897_taken_packed (by simp)
    (by rw [hS, scratch_slot, optWord_eq]; exact href) (by kjump_dest) h
  simp only [ethbridge_block_1897_taken_memory, hS] at r1
  cases hp : I.perm with
  | false =>
    right; left
    refine ⟨rfl, hrf, ?_⟩
    have q := kevm_run r1 with [jumpdest, push1 (UInt256.ofNat 0), dup2, dup2]
    have q5 := RD.genMstore q (by evm_kdecide) (by evm_ov)
    have q6 := kevm_run q5 with [push1 (UInt256.ofNat 32), dup2, swap1]
    have q9 := RD.genMstore q6 (by evm_kdecide) (by evm_ov)
    have q10 := kevm_run q9 with [push1 (UInt256.ofNat 64), swap1, dup2, swap1]
    have q14 := RD.genKeccak256 q10 (by evm_kdecide) (by evm_ov)
    have q15 := kevm_run q14 with [dup1]
    obtain ⟨_, _, q16⟩ := RD.sload q15 (by evm_kdecide) (by evm_ov)
    have q17 := q16.pushConst (UInt256.ofNat
      115792089237316195423570985008687907853269984665640564039457584007913129639680)
      (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
    have q21 := kevm_run q17 with [and, push1 (UInt256.ofNat 1), or, swap1]
    exact RD.sstoreStatic q21 hp (by evm_kdecide) (by evm_ov)
  | true =>
    have hml := memLoad_scr H (UInt256.ofNat fp) T hT (by omega)
    have hst : sstoreAccountMap I.codeOwner σ₁ (refundedSlot H)
        (UInt256.lor (UInt256.ofNat 1) (UInt256.land (UInt256.ofNat
          115792089237316195423570985008687907853269984665640564039457584007913129639680)
          (storageWord σ₁ I.codeOwner (refundedSlot H)))) = storedMap σ₁ I H := rfl
    by_cases hx : extCodeSizeWord (storedMap σ₁ I H) ethLiqWord = ⟨0⟩
    · obtain ⟨_, _, r2⟩ := ethbridge_block_1970_fallthrough (by simp) hp (by
          rw [scratch_idem, scratch_slot, optWord_eq, hst]
          show UInt256.isZero (UInt256.isZero (extCodeSizeWord (storedMap σ₁ I H) ethLiqWord)) = _
          rw [hx]; decide) r1
      exact Or.inl (ethbridge_block_2123 (by simp [ethbridge_block_1970_fallthrough_stack]) r2)
    obtain ⟨_, _, _, r2⟩ := ethbridge_block_1970_taken_packed (by simp) hp (by
        rw [scratch_idem, scratch_slot, optWord_eq, hst]
        show UInt256.isZero (UInt256.isZero (extCodeSizeWord (storedMap σ₁ I H) ethLiqWord)) ≠ _
        rw [Words.isZero_isZero_of_ne hx]; decide) (by kjump_dest) r1
    simp only [ethbridge_block_1970_taken_stack, ethbridge_block_1970_taken_memory, scratch_idem,
      scratch_slot, optWord_eq, hml, hst] at r2
    rw [mint_mem _ _ _ (by simp [hTl]; omega) hfpb, memLoad_pre_scr H T _ fp hT hTl hfp,
      Words.sub_add_self'] at r2
    have r3 := ethbridge_block_2127 (by simp) r2
    simp only [ethbridge_block_2127_stack] at r3
    have hdec : decode ethbridgeRuntime (UInt256.ofNat 2130) = some (.CALL, .none) := by evm_kdecide
    by_cases hd : I.depth.val < 1024
    swap
    · have hd' : I.depth = 1024 := by
        apply Fin.ext; have := I.depth.isLt; simp only [Fin.val_ofNat] at *; omega
      obtain ⟨k4, C4, r4⟩ := RD.callDepthLimit r3 hdec hd' (by simp)
      have r4' := RD.normalizePC (pc' := UInt256.ofNat 2131) r4 (by decide)
      have r5 := ethbridge_block_2131_fallthrough (by simp) (by decide) r4'
      exact Or.inl (ethbridge_block_2138 (by simp [ethbridge_block_2131_fallthrough_stack]) r5)
    · obtain ⟨σ₃, z, oM, A_in, callGas, k4, C4, ⟨g'', A', hΘ⟩, r4, _⟩ := RD.call r3 hdec hd (by simp)
      simp only [toNat_fp0 hfpb, toNat_fp0 (show (36:ℕ) < 2 ^ 200 by norm_num)] at hΘ
      rw [read_rel _ _ _ 0 36 (by simp) (by decide) (by decide) (by simp),
        List.drop_zero, List.take_of_length_le (by simp), ← mintCalldata_eq] at hΘ
      have hcall : CallTo σ₀ I ethLiq (mintCalldata (argAmount I)) (storedMap σ₁ I H) σ₃ z oM :=
        ⟨A_in, callGas, g'', A', hΘ⟩
      rw [Words.min0_toNat, write_len0] at r4
      have r4' := RD.normalizePC (pc' := UInt256.ofNat 2131) r4 (by decide)
      cases z with
      | false =>
        simp only [Bool.false_eq_true, if_false] at r4'
        have r5 := ethbridge_block_2131_fallthrough (by simp) (by decide) r4'
        exact Or.inl (ethbridge_block_2138 (by simp [ethbridge_block_2131_fallthrough_stack]) r5)
      | true =>
        simp only [if_true] at r4'
        have r5 := ethbridge_block_2131_taken (by simp) (by decide) (by kjump_dest) r4'
        simp only [ethbridge_block_2131_taken_stack] at r5
        exact Or.inr (Or.inr ⟨rfl, hrf, hx, σ₃, oM, hcall, _, _, _, r5⟩)

end BridgeEvm
