import L1cdmEvm.OuterFinal

/-! # Inner trace (the self-call), segment 1: dispatcher and ABI decode of
`sendMessage(address,bytes,uint32)` on the fixed calldata `sendMessageCd H t` (pc 0 → 3158) -/

namespace L1cdmEvm

set_option maxRecDepth 100000

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach l1cdmBlocks L1cdmEvm.SymMem

theorem iseg_entry {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256} {H t : UInt256}
    (env : List UInt256)
    (hcode : I.code = l1cdmRuntime) (hcd : I.calldata = sendMessageCd H t) :
    ∃ m aw k C, Frame env m 0 (swriteU (csyms 128) [] 64) ∧
      RD l1cdmRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 3158)
        [UInt256.ofNat 100000, UInt256.ofNat 68, UInt256.ofNat 132,
          UInt256.ofNat 376793390874373408599387495934666716005045108771, UInt256.ofNat 766,
          UInt256.ofNat 1035673643] m aw ByteArray.empty σ k C := by
  have hM : I.calldata = Mem [H, t] sendMessageSyms := by rw [hcd, sendMessageSyms_eq]
  have hsz : I.calldata.size = 228 := by rw [hM, Mem_size]; decide
  have r0 := RD.initState (σ := σ) (σ₀ := σ₀) (A := A) (g := g) hcode
  have r1 := l1cdm_block_0_fallthrough (R := []) (by simp) (by rw [hsz]; decide) r0
  have hsel : uInt256OfByteArray (I.calldata.readBytes (UInt256.ofNat 0).toNat 32) =
      UInt256.ofNat 27921706179853611545923559487109582912368054471931530464421775953236175355904 :=
    cdload_eq hM 0 (by norm_num) _ (by decide +kernel)
  have r2 := l1cdm_block_13_taken (by simp) (by rw [hsel]; decide) (by kjump_dest) r1
  simp only [l1cdm_block_13_taken_stack, hsel,
    show UInt256.shiftRight (UInt256.ofNat
      27921706179853611545923559487109582912368054471931530464421775953236175355904) (UInt256.ofNat 224)
      = UInt256.ofNat 0x3dbb202b by decide] at r2
  have r3 := l1cdm_block_258_fallthrough (by simp) (by decide) r2
  have r4 := l1cdm_block_270_taken (by simp) (by decide) (by kjump_dest) r3
  have r5 := l1cdm_block_329_fallthrough (by simp) (by decide) r4
  have r6 := l1cdm_block_341_fallthrough (by simp) (by decide) r5
  have r7 := l1cdm_block_352_taken (by simp) (by decide) (by kjump_dest) r6
  have r8 := l1cdm_block_830 (by simp) (by kjump_dest) r7
  simp only [l1cdm_block_830_stack, hsz] at r8
  have r9 := l1cdm_block_9439_taken (by simp) (by decide) (by kjump_dest) r8
  simp only [l1cdm_block_9439_taken_stack] at r9
  have r10 := l1cdm_block_9461 (by simp) (by kjump_dest) r9
  have h4 : uInt256OfByteArray (I.calldata.readBytes (UInt256.ofNat 4).toNat 32) =
      UInt256.ofNat 376793390874373408599387495934666716005045108771 :=
    cdload_eq hM 4 (by norm_num) _ (by decide +kernel)
  simp only [l1cdm_block_9461_stack, h4] at r10
  have r11 := l1cdm_block_9304_taken (by simp) (by decide) (by kjump_dest) r10
  have r12 := l1cdm_block_9338 (by simp) (by kjump_dest) r11
  simp only [l1cdm_block_9338_stack] at r12
  have h36 : uInt256OfByteArray (I.calldata.readBytes (UInt256.ofNat 4 + UInt256.ofNat 32).toNat 32) =
      UInt256.ofNat 96 := cdload_eq hM 36 (by norm_num) _ (by decide +kernel)
  have r13 := l1cdm_block_9472_taken (by simp) (by rw [h36]; decide) (by kjump_dest) r12
  simp only [l1cdm_block_9472_taken_stack, h36] at r13
  have r14 := l1cdm_block_9500 (by simp) (by kjump_dest) r13
  simp only [l1cdm_block_9500_stack, show UInt256.ofNat 4 + UInt256.ofNat 96 = UInt256.ofNat 100 by decide]
    at r14
  have r15 := l1cdm_block_9341_taken (by simp) (by decide) (by kjump_dest) r14
  simp only [l1cdm_block_9341_taken_stack] at r15
  have h100 : uInt256OfByteArray (I.calldata.readBytes (UInt256.ofNat 100).toNat 32) =
      UInt256.ofNat 68 := cdload_eq hM 100 (by norm_num) _ (by decide +kernel)
  have r16 := l1cdm_block_9359_taken (by simp) (by rw [h100]; decide) (by kjump_dest) r15
  simp only [l1cdm_block_9359_taken_stack, h100] at r16
  have r17 := l1cdm_block_9383_taken (by simp) (by decide) (by kjump_dest) r16
  simp only [l1cdm_block_9383_taken_stack, show UInt256.ofNat 100 + UInt256.ofNat 32 = UInt256.ofNat 132
    by decide] at r17
  have r18 := l1cdm_block_9407 (by simp) (by kjump_dest) r17
  simp only [l1cdm_block_9407_stack] at r18
  have r19 := l1cdm_block_9512 (by simp) (by kjump_dest) r18
  simp only [l1cdm_block_9512_stack, show UInt256.ofNat 4 + UInt256.ofNat 64 = UInt256.ofNat 68 by decide]
    at r19
  have h68 : uInt256OfByteArray (I.calldata.readBytes (UInt256.ofNat 68).toNat 32) =
      UInt256.ofNat 100000 := cdload_eq hM 68 (by norm_num) _ (by decide +kernel)
  have r20 := l1cdm_block_9414_taken (by simp) (by rw [h68]; decide) (by kjump_dest) r19
  simp only [l1cdm_block_9414_taken_stack, h68] at r20
  have r21 := l1cdm_block_9434 (by simp) (by kjump_dest) r20
  simp only [l1cdm_block_9434_stack] at r21
  have r22 := l1cdm_block_9531 (by simp) (by kjump_dest) r21
  simp only [l1cdm_block_9531_stack] at r22
  have r23 := l1cdm_block_844 (by simp) (by kjump_dest) r22
  have hF0 := frame_mstore_const (frame_nil env ByteArray.empty 0) 128 (UInt256.ofNat 128) rfl 64
  exact ⟨_, _, _, _, hF0, r23⟩

end L1cdmEvm
