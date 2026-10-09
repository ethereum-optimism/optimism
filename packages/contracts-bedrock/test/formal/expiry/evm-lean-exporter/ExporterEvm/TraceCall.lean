import ExporterEvm.CallMem

/-!
# Trace segment 4 (pc 439 → 710): `L2CrossDomainMessenger.sendMessage(...)`

`abi.encodeCall(IL1CrossDomainMessenger.relayUndeliveredMessage, (H, block.timestamp))`
(pc 439), the ABI encoding of `sendMessage(_sourceMessenger, that, _minGasLimit)` (pc 1299, 1132,
the copy loop with 3 iterations, 1346), solc's `EXTCODESIZE` check on 0x4200…0007 (pc 664) and
the `CALL` (pc 693). Revert outputs: empty (no code at the messenger, call depth) or the failed
call's own return data.
-/

namespace ExporterEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach exporterBlocks Mem

/-- The stack at pc 439 below the hash. -/
abbrev staticTail (I : ExecutionEnv) : List UInt256 :=
  [argMinGas I, argMsgLen I, UInt256.ofNat (msgPos I + 32), argTarget I, argSender I, argNonce I,
    argSource I, argSrcMessenger I, UInt256.ofNat 78, exportSelector]

theorem sub_ofNat_add (a c : ℕ) (h : a + c < 2 ^ 256) :
    UInt256.sub (UInt256.ofNat (a + c)) (UInt256.ofNat a) = UInt256.ofNat c := by
  rw [ofNat_sub_ofNat _ _ (by omega) (by omega), Nat.add_sub_cancel_left]

theorem big68 : UInt256.ofNat
    115792089237316195423570985008687907853269984665640564039457584007913129640004 = UInt256.ofNat 68 := by
  decide +kernel

abbrev sendSelW : UInt256 :=
  UInt256.ofNat 27921706179853611545923559487109582912280325424209726016810124687257672613888

theorem sendSel_lit : UInt256.land (UInt256.shiftLeft (UInt256.ofNat 1035673643) (UInt256.ofNat 224))
    (UInt256.ofNat 115792089210356248756420345214020892766250353992003419616917011526809519390720) = sendSelW := by
  decide +kernel

abbrev relSelW : UInt256 :=
  UInt256.ofNat 24938299286184694734990173794665248623958599392764557417163592631308932612096

theorem sendMessageCd_eq (t H ts gl : UInt256) :
    sendMessageCd t H ts gl = ofL ((wb sendSelW).take 4 ++ (wb t ++ (wb (UInt256.ofNat 96) ++ (wb gl ++
      (wb (UInt256.ofNat 68) ++ ((wb relSelW).take 4 ++ (wb H ++ (wb ts ++ List.replicate 28 0)))))))) := by
  unfold sendMessageCd relayMessage
  rw [show sendMessageSelector = ofL ((wb sendSelW).take 4) by decide +kernel,
    show relaySelector = ofL ((wb relSelW).take 4) by decide +kernel, zeroes_ofL]
  simp only [toByteArray_eq_ofL, append_ofL, List.append_assoc]

theorem land_mask32_of_lt {x : UInt256} (h : x.toNat < 2 ^ 32) :
    UInt256.land x (UInt256.ofNat 4294967295) = x := by
  have := (Words.uint32_clean_iff x).mpr h
  rw [Words.eq_ne0] at this
  exact this.symm

/-- The revert outputs of this segment. -/
def CallRevert (σ₀ : AccountMap) (I : ExecutionEnv) (H : UInt256) (σ₁ : AccountMap) (o : ByteArray) : Prop :=
  o = ByteArray.empty ∨
  (extCodeSizeWord σ₁ l2cdmWord ≠ ⟨0⟩ ∧
    ∃ σ₂ rd, CallTo σ₀ I l2cdm (sendMessageCd (argSrcMessenger I) H (tsWord I) (argMinGas I)) σ₁ σ₂ false rd ∧
      o = bubble rd)

set_option maxHeartbeats 4000000 in
theorem seg_call {σ₀ σ₁ : AccountMap} {σ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256}
    {aw : UInt256} {k C : ℕ} {H : UInt256} {B : List UInt8} {fp : ℕ} {rd : ByteArray}
    (hargs : ArgsOk I) (hB : FmpMem B fp) (hfp : fp < 2 ^ 140)
    (h : RD exporterRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 439)
        (H :: staticTail I) (ofL B) aw rd σ₁ k C) :
    RDrevP exporterRuntime g (initState σ σ₀ g A I) (CallRevert σ₀ I H σ₁) ∨
    (extCodeSizeWord σ₁ l2cdmWord ≠ ⟨0⟩ ∧
      ∃ σ₂ oC, CallTo σ₀ I l2cdm (sendMessageCd (argSrcMessenger I) H (tsWord I) (argMinGas I)) σ₁ σ₂ true oC ∧
      ∃ Q aw' k' C', 164 ≤ Q.length ∧
        RD exporterRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 710)
          (UInt256.isZero ⟨1⟩ :: UInt256.ofNat (fp + 328) :: UInt256.ofNat 1035673643 :: l2cdmWord :: H ::
            staticTail I)
          (ofL (pre (setFmp B (fp + 100)) fp ++ Q)) aw' oC σ₂ k' C') := by
  have h96 := hB.len96
  have hge := hB.ge
  have h100 : 100 + fp = fp + 100 := by omega
  obtain ⟨_, _, _, r1⟩ := exporter_block_439_packed (by simp) h
  simp only [exporter_block_439_stack, exporter_block_439_memory] at r1
  simp (disch := first | omega | decide | (simp; omega) | simp) only [memLoad_fmp hB, ofNat_add_ofNat,
    toNat_small, h100, memLoad_W2 B fp hB, sub_ofNat_add, big68, Nat.reduceAdd, head3 B fp hB,
    write_fmp_pre' B _ fp _ h96 hge, sendSel_lit, memLoad_rel_pre, wb_merge',
    memLoad_setFmp_pre B _ fp _ h96 hge, write_rel_pre] at r1
  msimp at r1
  have hS := hargs.srcMessengerClean
  have r2 := exporter_block_663 (by simp) (by kjump_dest) r1
  simp only [exporter_block_663_stack] at r2
  obtain ⟨_, _, _, r3⟩ := exporter_block_1299_packed (by simp) (by kjump_dest) r2
  simp only [exporter_block_1299_stack, exporter_block_1299_memory] at r3
  have e1 : 4 + (fp + 100) = fp + 104 := by omega
  have e2 : fp + 104 + 32 = fp + 136 := by omega
  have e3 : fp + 104 + 96 = fp + 200 := by omega
  simp (disch := first | omega | decide | (simp; omega) | simp) only [e1, ofNat_add_ofNat, toNat_small, e2, e3,
    write_rel_pre, Words.land_mask_of_lt hS] at r3
  msimp at r3
  obtain ⟨_, _, _, r4⟩ := exporter_block_1132_packed (by simp) r3
  simp only [exporter_block_1132_stack, exporter_block_1132_memory] at r4
  simp (disch := first | omega | decide | (simp; omega) | simp) only [memLoad_rel_pre0, toNat_small,
    write_rel_pre] at r4
  msimp at r4
  -- the 3-iteration copy of `relayUndeliveredMessage(H, t)` (68 bytes)
  generalize hQ9 : (wb (UInt256.ofNat 68) ++ (List.take 4 (wb (UInt256.ofNat
    24938299286184694734990173794665248623958599392764557417163592631308932612096)) ++ (wb H ++
    (wb (UInt256.ofNat I.header.timestamp) ++ (List.take 4 (wb sendSelW) ++ (wb (argSrcMessenger I) ++
    (wb (UInt256.ofNat 96) ++ (List.replicate 32 0 ++ wb (UInt256.ofNat 68))))))))) = Q9 at r4
  have hQ9l : Q9.length = 232 := by rw [← hQ9]; simp
  have hsrc : ((pre (setFmp B (fp + 100)) fp ++ Q9).drop (fp + 32)).take (32 * 3) = (Q9.drop 32).take 96 := by
    rw [drop_pre_append]
  obtain ⟨_, _, _, r5⟩ := copy_loop (ee := I) (Pre := pre (setFmp B (fp + 100)) fp ++ Q9)
    (Mp := (Q9.drop 32).take 96) (v := fp) (pos := fp + 200) (L := 68) (n := 3)
    (Lw := UInt256.ofNat 68) rfl (by norm_num) (by norm_num) hsrc (by simp [hQ9l])
    (by simp [hQ9l]; try omega) (by omega) 3 0 (by omega) _ _ _
    (by simpa only [Nat.mul_zero, List.take_zero, List.append_nil] using r4) (by simp)
  obtain ⟨_, _, _, r6⟩ := copy_tail (L := 68) (n := 3) rfl (by norm_num) (by norm_num)
    (by simp [hQ9l]) (by simp [hQ9l]; try omega) (by omega) (by simp) (by kjump_dest) r5
  unfold padTail at r6
  rw [if_pos (by norm_num)] at r6
  subst hQ9
  obtain ⟨_, _, _, r7⟩ := exporter_block_1346_packed (by simp) (by kjump_dest) r6
  simp only [exporter_block_1346_stack, exporter_block_1346_memory] at r7
  have e4 : fp + 104 + 64 = fp + 168 := by omega
  have e5 : fp + 200 + 32 + 32 * 3 = fp + 328 := by omega
  simp (disch := first | omega | decide | (simp; omega) | simp) only [ofNat_add_ofNat, toNat_small, e4, e5,
    List.append_assoc, write_rel_pre, land_mask32_of_lt hargs.minGasClean] at r7
  msimp at r7
  have hdec : decode exporterRuntime (UInt256.ofNat 693) = some (.CALL, .none) := by evm_kdecide
  by_cases hx : extCodeSizeWord σ₁ (UInt256.ofNat 376793390874373408599387495934666716005045108743) = ⟨0⟩
  · obtain ⟨_, _, _, r8⟩ := exporter_block_664_fallthrough_packed (by simp) (by rw [hx]; decide) r7
    simp only [exporter_block_664_fallthrough_stack] at r8
    left
    exact (RD.rev00 r8 (by evm_kdecide) (by evm_kdecide) (by evm_kdecide) (by stk_ov)).mono
      (fun _ ho => Or.inl ho)
  obtain ⟨_, _, _, r8⟩ := exporter_block_664_taken_packed (by simp)
    (by rw [Words.isZero_isZero_of_ne hx]; decide) (by kjump_dest) r7
  simp only [exporter_block_664_taken_stack, memLoad_setFmp_pre B _ fp _ h96 hge,
    show fp + 328 = (fp + 100) + 228 by omega, sub_ofNat_add _ _ (show fp + 100 + 228 < 2 ^ 256 by omega)] at r8
  obtain ⟨_, _, _, r9⟩ := exporter_block_690_packed (by simp) r8
  simp only [exporter_block_690_stack] at r9
  by_cases hd : I.depth.val < 1024
  swap
  · have hd' : I.depth = 1024 := by
      apply Fin.ext; have := I.depth.isLt; simp only [Fin.val_ofNat] at *; omega
    obtain ⟨k4, C4, r10⟩ := RD.callDepthLimit r9 hdec hd' (by simp)
    have r10' := RD.normalizePC (pc' := UInt256.ofNat 694) r10 (by decide)
    have r11 := exporter_block_694_fallthrough (by simp) (by decide) r10'
    simp only [exporter_block_694_fallthrough_stack] at r11
    left
    refine (RD.bubble r11 (by simp) (by evm_kdecide) (by evm_kdecide) (by evm_kdecide) (by evm_kdecide)
      (by evm_kdecide) (by evm_kdecide) (by evm_kdecide) (by stk_ov)).mono ?_
    intro o' ho'
    rw [bubble_empty] at ho'
    exact Or.inl ho'
  obtain ⟨σ₂, z, oC, A_in, callGas, k4, C4, ⟨g'', A', hΘ⟩, r10, hosize⟩ := RD.call r9 hdec hd (by simp)
  rw [toNat_small (by omega), toNat_small (n := 228) (by norm_num), read_rel_pre _ _ _ 100 228 (by norm_num)
    (by norm_num) (by simp)] at hΘ
  msimp at hΘ
  rw [← sendMessageCd_eq] at hΘ
  have hcall : CallTo σ₀ I l2cdm (sendMessageCd (argSrcMessenger I) H (tsWord I) (argMinGas I)) σ₁ σ₂ z oC :=
    ⟨A_in, callGas, g'', A', hΘ⟩
  rw [Words.min0_toNat, write_len0] at r10
  have r10' := RD.normalizePC (pc' := UInt256.ofNat 694) r10 (by decide)
  have hos : oC.size < 2 ^ 256 := by have := hosize; rw [Words.size_eq] at this; exact this
  cases z with
  | false =>
    simp only [Bool.false_eq_true, if_false] at r10'
    have r11 := exporter_block_694_fallthrough (by simp) (by decide) r10'
    simp only [exporter_block_694_fallthrough_stack] at r11
    left
    refine (RD.bubble r11 hos (by evm_kdecide) (by evm_kdecide) (by evm_kdecide) (by evm_kdecide)
      (by evm_kdecide) (by evm_kdecide) (by evm_kdecide) (by stk_ov)).mono ?_
    intro o' ho'
    exact Or.inr ⟨hx, σ₂, oC, hcall, ho'⟩
  | true =>
    simp only [if_true] at r10'
    have r11 := exporter_block_694_taken (by simp) (by decide) (by kjump_dest) r10'
    simp only [exporter_block_694_taken_stack] at r11
    right
    exact ⟨hx, σ₂, oC, hcall, _, _, _, _, by simp, r11⟩

end ExporterEvm
