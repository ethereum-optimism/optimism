import BridgeEvm.TraceStore
import BridgeEvm.Create

namespace BridgeEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach ethbridgeBlocks Mem

/-- The bytes `CODECOPY(fp, 3030, 89)` copies are SafeSend's creation code. -/
theorem code_safeSend :
    (ethbridgeRuntime.data.toList.drop 3030).take 89 = safeSendInitcode.data.toList := by
  decide +kernel

set_option maxRecDepth 100000 in
theorem ethbridgeRuntime_size : ethbridgeRuntime.size = 3131 := by decide +kernel

theorem safeSendDeploy_eq (r : UInt256) :
    safeSendDeploy r = ofL (safeSendInitcode.data.toList ++ wb r) := by
  unfold safeSendDeploy; rw [eq_ofL safeSendInitcode, toByteArray_eq_ofL, append_ofL]; rfl

set_option maxHeartbeats 4000000 in
/-- Trace segment 5 (pc 2147 → STOP): `new SafeSend{value: amount}(payable(from))` — `CODECOPY`
    of SafeSend's creation code to the free-memory pointer, the ABI-encoded `from` after it, the
    `CREATE` (via `RD.create`), the zero-address check, the `RefundETH` `LOG3`, and `STOP`. Either
    the run reverts (the creation failed: `CREATE` pushed 0), or `CREATE` ran with value
    `amount` and init code exactly `safeSendDeploy from`, pushed a nonzero `x`, and the run
    halts successfully with the account map `σ'` the creation produced. -/
theorem seg_create {σ σ₀ σ₃ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256}
    {aw : UInt256} {k C : ℕ} {H : UInt256} {P : List UInt8} {fp : ℕ} {oM : ByteArray}
    (hfrom : (argFrom I).toNat < 2 ^ 160) (hperm : I.perm = true)
    (hP : P.length = fp) (hP64 : (P.drop 64).take 32 = wb (UInt256.ofNat fp)) (hfp : 676 ≤ fp)
    (hfpb : fp < 2 ^ 200)
    (h : RD ethbridgeRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 2147)
        [UInt256.isZero (⟨1⟩ : UInt256), UInt256.ofNat 36 + UInt256.ofNat fp, UInt256.ofNat 2691771752,
          ethLiqWord, H, argAmount I, argTo I, argFrom I, argNonce I, argDest I, UInt256.ofNat 127,
          refundSelector]
        (ofL (P ++ ((wb mintSelWord).take 4 ++ wb (argAmount I)))) aw oM σ₃ k C) :
    RDrev ethbridgeRuntime g (initState σ σ₀ g A I) ∨
    ∃ x σ' rd', CreateStep I σ₀ σ₃ (argAmount I) (safeSendDeploy (argFrom I)) x σ' rd' ∧
      x ≠ UInt256.ofNat 0 ∧ RDret ethbridgeRuntime g (initState σ σ₀ g A I) σ' ByteArray.empty := by
  have hml : ∀ Q, memLoad (UInt256.ofNat 64) (ofL (P ++ Q)) = UInt256.ofNat fp := fun Q => by
    rw [memLoad_ofL _ _ (by simp; rw [toNat_fp0 (by norm_num)]; omega), toNat_fp0 (by norm_num),
      List.drop_append_of_le_length (by omega), List.take_append_of_le_length (by simp; omega), hP64,
      ofNat_fromBE_wb]
  obtain ⟨_, _, _, r1⟩ := ethbridge_block_2147_packed (by simp) (by kjump_dest) h
  simp only [ethbridge_block_2147_stack, hml] at r1
  obtain ⟨_, _, _, r2⟩ := ethbridge_block_2377_packed (by simp) (by kjump_dest) r1
  simp only [ethbridge_block_2377_stack, ethbridge_block_2377_memory] at r2
  rw [toNat_fp0 hfpb, toNat_fp0 (show (3030:ℕ) < 2 ^ 200 by norm_num),
    toNat_fp0 (show (89:ℕ) < 2 ^ 200 by norm_num),
    write_rel_src _ _ _ _ _ 0 _ (by decide) (by rw [ethbridgeRuntime_size]; norm_num) (by simp [hP]),
    code_safeSend] at r2
  have hss : safeSendInitcode.data.toList.length = 89 := by decide +kernel
  simp only [List.length_append, List.length_take, length_wb, Nat.zero_sub, List.replicate_zero,
    List.append_nil, List.take_zero, List.nil_append, Nat.zero_add,
    List.drop_eq_nil_of_le (show (List.take 4 (wb mintSelWord) ++ wb (argAmount I)).length ≤ 89 by simp)] at r2
  obtain ⟨_, _, _, r3⟩ := ethbridge_block_2165_packed (by simp) r2
  simp only [ethbridge_block_2165_stack, ethbridge_block_2165_memory, Words.land_mask_of_lt hfrom] at r3
  have h200 : (2:ℕ) ^ 200 = 1606938044258990275541962092341162602522202993782792835301376 := by norm_num
  rw [ofNat_add_ofNat, toNat_lit (by omega),
    write_rel _ _ _ _ (89) (by simp [hP]; omega)] at r3
  simp only [hml, List.length_append, hss, Nat.sub_self, List.replicate_zero, List.append_nil,
    List.take_of_length_le (show safeSendInitcode.data.toList.length ≤ 89 by rw [hss]),
    List.drop_eq_nil_of_le (show safeSendInitcode.data.toList.length ≤ 89 + 32 by rw [hss]; omega),
    ofNat_add_ofNat] at r3
  rw [show UInt256.sub (UInt256.ofNat (32 + (89 + fp))) (UInt256.ofNat fp) = UInt256.ofNat 121 by
    rw [ofNat_sub_ofNat _ _ (by omega) (by omega)]; congr 1; omega] at r3
  have hdec : decode ethbridgeRuntime (UInt256.ofNat 2203) = some (.CREATE, .none) := by evm_kdecide
  obtain ⟨x, σ', rd', k4, C4, hcs, r4⟩ := RD.create r3 hdec hperm (by decide) (by simp)
  rw [toNat_lit (by omega), toNat_lit (by norm_num),
    read_rel _ _ _ 0 121 (by simp [hP]) (by decide) (by decide) (by simp [hss]),
    List.drop_zero, List.take_of_length_le (by simp [hss]), ← safeSendDeploy_eq] at hcs
  have r4' := RD.normalizePC (pc' := UInt256.ofNat 2204) r4 (by decide)
  by_cases hx : x = UInt256.ofNat 0
  · have r5 := ethbridge_block_2204_fallthrough (by simp) (by rw [hx]; decide) r4'
    exact Or.inl (ethbridge_block_2214 (by simp [ethbridge_block_2204_fallthrough_stack]) r5)
  have r5 := ethbridge_block_2204_taken (by simp) (by rw [Words.isZero_isZero_of_ne hx]; decide)
    (by kjump_dest) r4'
  simp only [ethbridge_block_2204_taken_stack] at r5
  obtain ⟨_, _, _, r6⟩ := ethbridge_block_2223_packed (by simp) (by kjump_dest) r5
  simp only [ethbridge_block_2223_stack] at r6
  obtain ⟨_, _, _, r7⟩ := ethbridge_block_2298_packed (by simp) hperm (by kjump_dest) r6
  simp only [ethbridge_block_2298_stack] at r7
  exact Or.inr ⟨x, σ', rd', hcs, hx, ethbridge_block_127 (by simp) r7⟩

end BridgeEvm
