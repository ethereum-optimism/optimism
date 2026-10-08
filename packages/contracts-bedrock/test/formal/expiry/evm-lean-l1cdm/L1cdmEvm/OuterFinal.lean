import L1cdmEvm.Tails

/-! # Outer trace, segment 9: `this.sendMessage(0x4200..0023, abi.encodeCall(expireMessage, (H, t)),
100000)` and `STOP` (pc 2910 → end)

Memory: all writes are relative to `p = mload(0x40)` (symbolic: it depends on the return data sizes
of the seven view calls). The frame descriptors `D1 … D13` below follow solc's encoding step by step;
the call input is `sreadU D13 100 228`, decided equal to `sendMessageCd H t`. -/

namespace L1cdmEvm

set_option maxRecDepth 100000

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach l1cdmBlocks L1cdmEvm.SymMem

/-- The frame after `abi.encodeCall(expireMessage, (H, t))` and the `sendMessage` head. -/
def D1 : List Sym := swriteU (vsyms 0) [] 36
def D2 : List Sym := swriteU (vsyms 1) D1 68
def D3 : List Sym := swriteU (csyms 68) D2 0
def D4 : List Sym := swriteU (csel 0x763a1cb7 ++ sreadU D3 36 28) D3 32
def D5 : List Sym := swriteU (csyms (0x3dbb202b * 2 ^ 224)) D4 100
/-- The descriptors of `sendMessageCd H t` with `env = [H, t]`. -/
def sendMessageSyms : List Sym :=
  csel 0x3dbb202b ++ csyms 0x4200000000000000000000000000000000000023 ++ csyms 0x60 ++ csyms 100000 ++
    csyms 68 ++ csel 0x763a1cb7 ++ vsyms 0 ++ vsyms 1 ++ zeros 28

theorem sendMessageSyms_eq' (H t : UInt256) (rest : List UInt256) :
    Mem (H :: t :: rest) sendMessageSyms = sendMessageCd H t := by
  simp only [sendMessageSyms, Mem_append, sel4_eq_Mem, Mem_csyms, Mem_vsyms, Mem_zeros, sendMessageCd,
    expireMessageCd, ByteArray.append_assoc]
  rfl

theorem sendMessageSyms_eq (H t : UInt256) : Mem [H, t] sendMessageSyms = sendMessageCd H t := by
  simp only [sendMessageSyms, Mem_append, sel4_eq_Mem, Mem_csyms, Mem_vsyms, Mem_zeros, sendMessageCd,
    expireMessageCd, ByteArray.append_assoc]
  rfl


theorem seg_final {σ σ1 σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256}
    {m : ByteArray} {Bw aw : UInt256} {rdata : ByteArray} {k C : ℕ} {wP : UInt256}
    (hst : accountStorageStateEq σ σ1) (hcd : accountCodeStateEq σ σ1)
    (hF : Fmp m Bw) (hB : 96 ≤ Bw.toNat) (hBb : Bw.toNat < 2 ^ 40)
    (h : RD l1cdmRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 2910)
      [wP, UInt256.ofNat I.source.val, argTime I, argHash I, UInt256.ofNat 766, relaySelector] m aw rdata σ1 k C) :
    RDrev l1cdmRuntime g (initState σ σ₀ g A I) ∨
    (extCodeSizeWord σ1 (UInt256.ofNat I.codeOwner) ≠ ⟨0⟩ ∧
      ∃ σ' o, SelfCall σ₀ I (sendMessageCd (argHash I) (argTime I)) σ1 σ' true o ∧
        RDret l1cdmRuntime g (initState σ σ₀ g A I) σ' ByteArray.empty) := by
  have r1 := l1cdm_block_2910 (by simp) (by kjump_dest) h
  simp only [l1cdm_block_2910_stack, l1cdm_block_2910_memory, fmp_load hF (UInt256.ofNat 64) rfl] at r1
  have e36 : (Bw + UInt256.ofNat 36).toNat = Bw.toNat + 36 := by tnorm
  have e68 : (Bw + UInt256.ofNat 68).toNat = Bw.toNat + 68 := by tnorm
  rw [e36, e68] at r1
  -- the encodeCall's two argument words
  have hFa := fmp_keep (fmp_keep hF (UInt256.toByteArray (argHash I)) 0 (Bw.toNat + 36) 32 (by norm_num) (by rw [toByteArray_size]) (Or.inr (by omega)))
    (UInt256.toByteArray (argTime I)) 0 (Bw.toNat + 68) 32 (by norm_num) (by rw [toByteArray_size]) (Or.inr (by omega))
  have hDa : Frame [argHash I, argTime I] ((UInt256.toByteArray (argTime I)).write 0
      ((UInt256.toByteArray (argHash I)).write 0 m (Bw.toNat + 36) 32) (Bw.toNat + 68) 32) Bw.toNat D2 :=
    frame_mstore_var (frame_mstore_var (frame_nil [argHash I, argTime I] m Bw.toNat) 0 _ rfl 36) 1 _ rfl 68
  generalize (UInt256.toByteArray (argTime I)).write 0
      ((UInt256.toByteArray (argHash I)).write 0 m (Bw.toNat + 36) 32) (Bw.toNat + 68) 32 = ma at r1 hFa hDa
  have hsub : UInt256.ofNat 68 + UInt256.sub Bw Bw = UInt256.ofNat 68 := by
    rw [u_sub_self]; rfl
  simp only [fmp_load hFa (UInt256.ofNat 64) rfl, hsub] at r1
  -- the length word of the `bytes memory` at p
  have hDb := frame_mstore_const hDa 68 _ rfl 0
  rw [Nat.add_zero] at hDb
  have hFb := fmp_keep hFa (UInt256.toByteArray (UInt256.ofNat 68)) 0 Bw.toNat 32 (by norm_num)
    (by rw [toByteArray_size]) (Or.inr (by omega))
  generalize (UInt256.toByteArray (UInt256.ofNat 68)).write 0 ma Bw.toNat 32 = mb at r1 hDb hFb
  -- the free memory pointer moves to p + 100
  have hDc : Frame [argHash I, argTime I]
      ((UInt256.toByteArray (Bw + UInt256.ofNat 100)).write 0 mb (UInt256.ofNat 64).toNat 32) Bw.toNat
      (swriteU (csyms 68) D2 0) :=
    frame_write_disjoint (UInt256.toByteArray (Bw + UInt256.ofNat 100)) 0 64 32 hDb
      (by norm_num) (by rw [toByteArray_size]) (Or.inl (by omega))
  have hFc := fmp_store mb (Bw + UInt256.ofNat 100) (UInt256.ofNat 64).toNat rfl
  generalize (UInt256.toByteArray (Bw + UInt256.ofNat 100)).write 0 mb (UInt256.ofNat 64).toNat 32 = mc
    at r1 hDc hFc
  -- the selector is OR-ed into the word at p + 32
  rw [show UInt256.ofNat 53475591445150530343521212754713076481339709529486133482256841880961372127232
      = UInt256.ofNat (0x763a1cb7 * 2 ^ 224) by norm_num,
    show UInt256.ofNat 26959946667150639794667015087019630673637144422540572481103610249215
      = UInt256.ofNat (2 ^ 224 - 1) by norm_num,
    show (Bw + UInt256.ofNat 32).toNat = Bw.toNat + 32 by tnorm] at r1
  have hDd := frame_mstore_selmask hDc 0x763a1cb7 (by norm_num) (Bw + UInt256.ofNat 32) 32 (by tnorm) 32
  have hFd := fmp_keep hFc (UInt256.toByteArray (UInt256.lor (UInt256.ofNat (0x763a1cb7 * 2 ^ 224))
      (UInt256.land (UInt256.ofNat (2 ^ 224 - 1)) (memLoad (Bw + UInt256.ofNat 32) mc))))
    0 (Bw.toNat + 32) 32 (by norm_num) (by rw [toByteArray_size]) (Or.inr (by omega))
  generalize (UInt256.toByteArray (UInt256.lor (UInt256.ofNat (0x763a1cb7 * 2 ^ 224))
      (UInt256.land (UInt256.ofNat (2 ^ 224 - 1)) (memLoad (Bw + UInt256.ofNat 32) mc)))).write 0 mc
      (Bw.toNat + 32) 32 = md at r1 hDd hFd
  simp only [fmp_load hFd (UInt256.ofNat 64) rfl,
    show (Bw + UInt256.ofNat 100).toNat = Bw.toNat + 100 by tnorm] at r1
  -- the `sendMessage` selector at q = p + 100
  have hDe := frame_mstore_const hDd (0x3dbb202b * 2 ^ 224)
    (UInt256.ofNat 27921706179853611545923559487109582912280325424209726016810124687257672613888)
    (by norm_num) 100
  have hFe := fmp_keep hFd (UInt256.toByteArray (UInt256.ofNat
      27921706179853611545923559487109582912280325424209726016810124687257672613888)) 0
    (Bw.toNat + 100) 32 (by norm_num) (by rw [toByteArray_size]) (Or.inr (by omega))
  generalize (UInt256.toByteArray (UInt256.ofNat
      27921706179853611545923559487109582912280325424209726016810124687257672613888)).write 0 md
      (Bw.toNat + 100) 32 = me at r1 hDe hFe
  -- abi_encode(address, bytes, uint32) head at x0 = q + 4
  have r2 := l1cdm_block_10218 (by simp) (by kjump_dest) r1
  simp only [l1cdm_block_10218_stack, l1cdm_block_10218_memory,
    show (UInt256.ofNat 4 + (Bw + UInt256.ofNat 100)).toNat = Bw.toNat + 104 by tnorm,
    show (UInt256.ofNat 4 + (Bw + UInt256.ofNat 100) + UInt256.ofNat 32).toNat = Bw.toNat + 136 by tnorm,
    show UInt256.land (UInt256.ofNat 376793390874373408599387495934666716005045108771)
      (UInt256.ofNat 1461501637330902918203684832716283019655932542975) =
      UInt256.ofNat 0x4200000000000000000000000000000000000023 by decide] at r2
  have hDf := frame_mstore_const (frame_mstore_const hDe 0x4200000000000000000000000000000000000023 _ rfl 104)
    96 _ rfl 136
  have hFf := fmp_keep (fmp_keep hFe (UInt256.toByteArray (UInt256.ofNat 0x4200000000000000000000000000000000000023))
      0 (Bw.toNat + 104) 32 (by norm_num) (by rw [toByteArray_size]) (Or.inr (by omega)))
    (UInt256.toByteArray (UInt256.ofNat 96)) 0 (Bw.toNat + 136) 32 (by norm_num) (by rw [toByteArray_size])
    (Or.inr (by omega))
  generalize (UInt256.toByteArray (UInt256.ofNat 96)).write 0
      ((UInt256.toByteArray (UInt256.ofNat 0x4200000000000000000000000000000000000023)).write 0 me
        (Bw.toNat + 104) 32) (Bw.toNat + 136) 32 = mf at r2 hDf hFf
  rw [show UInt256.ofNat 4 + (Bw + UInt256.ofNat 100) + UInt256.ofNat 96 = Bw + UInt256.ofNat 200 by wnorm]
    at r2
  -- abi_encode_bytes: length word, then the copy loop
  have r3 := l1cdm_block_9599 (by simp) r2
  simp only [l1cdm_block_9599_stack, l1cdm_block_9599_memory,
    frame_memLoad_c hDf Bw 0 (by omega) 68 (by decide +kernel),
    show (Bw + UInt256.ofNat 200).toNat = Bw.toNat + 200 by tnorm] at r3
  have hDg := frame_mstore_const hDf 68 _ rfl 200
  have hFg := fmp_keep hFf (UInt256.toByteArray (UInt256.ofNat 68)) 0 (Bw.toNat + 200) 32 (by norm_num)
    (by rw [toByteArray_size]) (Or.inr (by omega))
  generalize (UInt256.toByteArray (UInt256.ofNat 68)).write 0 mf (Bw.toNat + 200) 32 = mg at r3 hDg hFg
  -- iteration i = 0
  have r4 := l1cdm_block_9609_fallthrough (by simp) (by decide) r3
  have r5 := l1cdm_block_9618 (by simp) (by kjump_dest) r4
  simp only [l1cdm_block_9618_stack, l1cdm_block_9618_memory,
    show (UInt256.ofNat 32 + (UInt256.ofNat 0 + (Bw + UInt256.ofNat 200))).toNat = Bw.toNat + 232 by tnorm,
    show UInt256.ofNat 32 + UInt256.ofNat 0 = UInt256.ofNat 32 by decide] at r5
  have hDh := frame_mstore_load hDg (UInt256.ofNat 32 + (Bw + UInt256.ofNat 0)) 32 (by tnorm) 232
  have hFh := fmp_keep hFg (UInt256.toByteArray (memLoad (UInt256.ofNat 32 + (Bw + UInt256.ofNat 0)) mg))
    0 (Bw.toNat + 232) 32 (by norm_num) (by rw [toByteArray_size]) (Or.inr (by omega))
  generalize (UInt256.toByteArray (memLoad (UInt256.ofNat 32 + (Bw + UInt256.ofNat 0)) mg)).write 0 mg
    (Bw.toNat + 232) 32 = mh at r5 hDh hFh
  -- iteration i = 32
  have r6 := l1cdm_block_9609_fallthrough (by simp) (by decide) r5
  have r7 := l1cdm_block_9618 (by simp) (by kjump_dest) r6
  simp only [l1cdm_block_9618_stack, l1cdm_block_9618_memory,
    show (UInt256.ofNat 32 + (UInt256.ofNat 32 + (Bw + UInt256.ofNat 200))).toNat = Bw.toNat + 264 by tnorm,
    show UInt256.ofNat 32 + UInt256.ofNat 32 = UInt256.ofNat 64 by decide] at r7
  have hDi := frame_mstore_load hDh (UInt256.ofNat 32 + (Bw + UInt256.ofNat 32)) 64 (by tnorm) 264
  have hFi := fmp_keep hFh (UInt256.toByteArray (memLoad (UInt256.ofNat 32 + (Bw + UInt256.ofNat 32)) mh))
    0 (Bw.toNat + 264) 32 (by norm_num) (by rw [toByteArray_size]) (Or.inr (by omega))
  generalize (UInt256.toByteArray (memLoad (UInt256.ofNat 32 + (Bw + UInt256.ofNat 32)) mh)).write 0 mh
    (Bw.toNat + 264) 32 = mi at r7 hDi hFi
  -- iteration i = 64
  have r8 := l1cdm_block_9609_fallthrough (by simp) (by decide) r7
  have r9 := l1cdm_block_9618 (by simp) (by kjump_dest) r8
  simp only [l1cdm_block_9618_stack, l1cdm_block_9618_memory,
    show (UInt256.ofNat 32 + (UInt256.ofNat 64 + (Bw + UInt256.ofNat 200))).toNat = Bw.toNat + 296 by tnorm,
    show UInt256.ofNat 32 + UInt256.ofNat 64 = UInt256.ofNat 96 by decide] at r9
  have hDj := frame_mstore_load hDi (UInt256.ofNat 32 + (Bw + UInt256.ofNat 64)) 96 (by tnorm) 296
  have hFj := fmp_keep hFi (UInt256.toByteArray (memLoad (UInt256.ofNat 32 + (Bw + UInt256.ofNat 64)) mi))
    0 (Bw.toNat + 296) 32 (by norm_num) (by rw [toByteArray_size]) (Or.inr (by omega))
  generalize (UInt256.toByteArray (memLoad (UInt256.ofNat 32 + (Bw + UInt256.ofNat 64)) mi)).write 0 mi
    (Bw.toNat + 296) 32 = mj at r9 hDj hFj
  -- loop exit (i = 96 ≥ 68) and the zero padding after the 68 bytes
  have r10 := l1cdm_block_9609_taken (by simp) (by decide) (by kjump_dest) r9
  have r11 := l1cdm_block_9637_fallthrough (by simp) (by decide) r10
  have r12 := l1cdm_block_9646 (by simp) r11
  simp only [l1cdm_block_9646_memory,
    show (Bw + UInt256.ofNat 200 + UInt256.ofNat 68 + UInt256.ofNat 32).toNat = Bw.toNat + 300 by tnorm] at r12
  have hDk := frame_mstore_const hDj 0 _ rfl 300
  have hFk := fmp_keep hFj (UInt256.toByteArray (UInt256.ofNat 0)) 0 (Bw.toNat + 300) 32 (by norm_num)
    (by rw [toByteArray_size]) (Or.inr (by omega))
  generalize (UInt256.toByteArray (UInt256.ofNat 0)).write 0 mj (Bw.toNat + 300) 32 = mk at r12 hDk hFk
  have r13 := l1cdm_block_9655 (by simp) (by kjump_dest) r12
  simp only [l1cdm_block_9655_stack] at r13
  -- _minGasLimit at x0 + 64, then back to the caller
  have r14 := l1cdm_block_10265 (by simp) (by kjump_dest) r13
  simp only [l1cdm_block_10265_stack, l1cdm_block_10265_memory,
    show (UInt256.ofNat 4 + (Bw + UInt256.ofNat 100) + UInt256.ofNat 64).toNat = Bw.toNat + 168 by tnorm,
    show UInt256.land (UInt256.ofNat 100000) (UInt256.ofNat 4294967295) = UInt256.ofNat 100000 by decide]
    at r14
  have hDl := frame_mstore_const hDk 100000 _ rfl 168
  have hFl := fmp_keep hFk (UInt256.toByteArray (UInt256.ofNat 100000)) 0 (Bw.toNat + 168) 32 (by norm_num)
    (by rw [toByteArray_size]) (Or.inr (by omega))
  generalize (UInt256.toByteArray (UInt256.ofNat 100000)).write 0 mk (Bw.toNat + 168) 32 = ml at r14 hDl hFl
  rw [show UInt256.land (UInt256.ofNat
      115792089237316195423570985008687907853269984665640564039457584007913129639904)
      (UInt256.ofNat 31 + UInt256.ofNat 68) = UInt256.ofNat 96 by decide,
    show UInt256.ofNat 32 + (UInt256.ofNat 96 + (Bw + UInt256.ofNat 200)) = (Bw + UInt256.ofNat 100) +
      UInt256.ofNat 228 by wnorm] at r14
  -- `extcodesize(address(this)) != 0`, then the CALL
  by_cases hcode : UInt256.isZero (UInt256.isZero (extCodeSizeWord σ1 (UInt256.ofNat I.codeOwner))) =
      UInt256.ofNat 0
  · left
    obtain ⟨k15, C15, r15⟩ := l1cdm_block_3102_fallthrough (by simp) hcode r14
    exact l1cdm_block_3124 (by simp [l1cdm_block_3102_fallthrough_stack]) r15
  obtain ⟨k15, C15, r15⟩ := l1cdm_block_3102_taken (by simp) hcode (by kjump_dest) r14
  simp only [l1cdm_block_3102_taken_stack, fmp_load hFl (UInt256.ofNat 64) rfl, u_sub_add_self] at r15
  have r16 := l1cdm_block_3128 (by simp) r15
  simp only [l1cdm_block_3128_stack] at r16
  have hdec : decode l1cdmRuntime (UInt256.ofNat 3131) = some (.CALL, .none) := by evm_kdecide
  have hecs : extCodeSizeWord σ1 (UInt256.ofNat I.codeOwner) ≠ ⟨0⟩ := by
    intro h0; apply hcode; rw [h0]; rfl
  by_cases hd : I.depth.val < 1024
  swap
  · have hd' : I.depth = 1024 := by
      apply Fin.ext; have := I.depth.isLt; omega
    obtain ⟨k17, C17, r17⟩ := RD.callDepthLimit r16 hdec hd' (by simp)
    left
    exact l1cdm_block_3139 (by simp [l1cdm_block_3132_fallthrough_stack])
      (l1cdm_block_3132_fallthrough (by simp) (by decide) r17)
  obtain ⟨σ', z, o, A_in, callGas, k17, C17, ⟨g'', A', hΘ⟩, r17, _⟩ := RD.call r16 hdec hd (by simp)
  have hin := frame_read_eq hDl 100 228 (by norm_num) sendMessageSyms (by decide +kernel) (by decide +kernel)
  rw [sendMessageSyms_eq] at hin
  rw [show (Bw + UInt256.ofNat 100).toNat = Bw.toNat + 100 by tnorm, show (UInt256.ofNat 228).toNat = 228
    from rfl, hin, AccountAddress.ofUInt256_ofNat I.codeOwner] at hΘ
  cases z with
  | false =>
    left
    simp only [Bool.false_eq_true, if_false] at r17
    exact l1cdm_block_3139 (by simp [l1cdm_block_3132_fallthrough_stack])
      (l1cdm_block_3132_fallthrough (by simp) (by decide) r17)
  | true =>
    simp only [if_true] at r17
    have r18 := l1cdm_block_3132_taken (by simp) (by decide) (by kjump_dest) r17
    simp only [l1cdm_block_3132_taken_stack] at r18
    have r19 := l1cdm_block_3148 (by simp) (by kjump_dest) r18
    simp only [l1cdm_block_3148_stack] at r19
    exact Or.inr ⟨hecs, σ', o, ⟨A_in, callGas, g'', A', hΘ⟩, l1cdm_block_766 (by simp) r19⟩

end L1cdmEvm
