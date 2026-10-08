import BridgeEvm.Spec
import BridgeEvm.Words
import BridgeEvm.AllBlocks

namespace BridgeEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach ethbridgeBlocks Mem

theorem mem0 : ethbridge_block_0_fallthrough_memory (mem := ByteArray.empty) =
    ofL (List.replicate 64 0 ++ wb (UInt256.ofNat 128)) := by
  unfold ethbridge_block_0_fallthrough_memory
  rw [empty_ofL, write_wb]
  mnorm

/-- The stack at pc 1539 (start of `refundETH`'s body): the five decoded arguments, the return
    pc and the selector. -/
abbrev entryStack (I : ExecutionEnv) : List UInt256 :=
  [argAmount I, argTo I, argFrom I, argNonce I, argDest I, UInt256.ofNat 127, refundSelector]

/-- Memory at pc 1539: the free-memory pointer `0x80` at `0x40`. -/
abbrev entryMem : List UInt8 := List.replicate 64 0 ++ wb (UInt256.ofNat 128)

theorem seg_entry {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256}
    (hcode : I.code = ethbridgeRuntime) (hsel : selectorWord I = refundSelector)
    (hcds : I.calldata.size < 2 ^ 256) :
    (RDrev ethbridgeRuntime g (initState σ σ₀ g A I) ∧ ¬ ArgsOk I) ∨
    (ArgsOk I ∧ ∃ aw k C, RD ethbridgeRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 1539)
        (entryStack I) (ofL entryMem) aw ByteArray.empty σ k C) := by
  have hcdok := Words.calldata_ok_iff _ 160 hcds (by norm_num)
  have r0 := RD.initState (σ := σ) (σ₀ := σ₀) (A := A) (g := g) hcode
  have hsel' : UInt256.shiftRight (uInt256OfByteArray (I.calldata.readBytes (UInt256.ofNat 0).toNat 32))
      (UInt256.ofNat 224) = UInt256.ofNat 3782899563 := hsel
  by_cases hlt : UInt256.lt (UInt256.ofNat I.calldata.size) (UInt256.ofNat 4) = UInt256.ofNat 0
  swap
  · left
    refine ⟨ethbridge_block_90 (by simp) (ethbridge_block_0_taken (by simp) hlt (by kjump_dest) r0),
      fun hc => hlt (hcdok.mpr hc.calldataLen).1⟩
  have r1 := ethbridge_block_0_fallthrough (R := []) (by simp) hlt r0
  have r2 := ethbridge_block_13_fallthrough (by simp) (by rw [hsel']; decide) r1
  simp only [ethbridge_block_13_fallthrough_stack] at r2
  rw [hsel'] at r2
  have r3 := ethbridge_block_30_fallthrough (by simp) (by decide) r2
  have r4 := ethbridge_block_41_taken (by simp) (by decide) (by kjump_dest) r3
  by_cases hv : UInt256.isZero I.weiValue = UInt256.ofNat 0
  · left
    refine ⟨ethbridge_block_265 (by simp [ethbridge_block_257_fallthrough_stack])
      (ethbridge_block_257_fallthrough (by simp) hv r4), fun hc => Words.isZero_eq0.mp hv hc.noValue⟩
  have r5 := ethbridge_block_257_taken (by simp) hv (by kjump_dest) r4
  simp only [ethbridge_block_257_taken_stack] at r5
  have r6 := ethbridge_block_269 (by simp) (by kjump_dest) r5
  simp only [ethbridge_block_269_stack] at r6
  by_cases hcd : UInt256.isZero (UInt256.slt (UInt256.sub (UInt256.ofNat I.calldata.size)
      (UInt256.ofNat 4)) (UInt256.ofNat 160)) = UInt256.ofNat 0
  · left
    refine ⟨ethbridge_block_2688 (by simp [ethbridge_block_2668_fallthrough_stack])
      (ethbridge_block_2668_fallthrough (by simp) hcd r6), fun hc => (hcdok.mpr hc.calldataLen).2 hcd⟩
  have hlen := hcdok.mp ⟨hlt, hcd⟩
  have r7 := ethbridge_block_2668_taken (by simp) hcd (by kjump_dest) r6
  simp only [ethbridge_block_2668_taken_stack] at r7
  have r8 := ethbridge_block_2692 (by simp) (by kjump_dest) r7
  simp only [ethbridge_block_2692_stack] at r8
  by_cases hf : UInt256.eq (argFrom I) (UInt256.land (argFrom I) addrMask) = UInt256.ofNat 0
  · left
    refine ⟨ethbridge_block_2419 (by simp) (ethbridge_block_2389_fallthrough (by simp) hf r8),
      fun hc => (Words.addr_clean_iff _).mpr hc.fromClean hf⟩
  have r9 := ethbridge_block_2389_taken (by simp) hf (by kjump_dest) r8
  have r10 := ethbridge_block_2423 (by simp) (by kjump_dest) r9
  simp only [ethbridge_block_2423_stack] at r10
  have r11 := ethbridge_block_2717 (by simp) (by kjump_dest) r10
  simp only [ethbridge_block_2717_stack] at r11
  by_cases ht : UInt256.eq (argTo I) (UInt256.land (argTo I) addrMask) = UInt256.ofNat 0
  · left
    refine ⟨ethbridge_block_2419 (by simp) (ethbridge_block_2389_fallthrough (by simp) ht r11),
      fun hc => (Words.addr_clean_iff _).mpr hc.toClean ht⟩
  have r12 := ethbridge_block_2389_taken (by simp) ht (by kjump_dest) r11
  have r13 := ethbridge_block_2423 (by simp) (by kjump_dest) r12
  simp only [ethbridge_block_2423_stack] at r13
  have r14 := ethbridge_block_2733 (by simp) (by kjump_dest) r13
  simp only [ethbridge_block_2733_stack] at r14
  have r15 := ethbridge_block_284 (by simp) (by kjump_dest) r14
  rw [mem0] at r15
  right
  exact ⟨⟨Words.isZero_ne0.mp hv, hlen, (Words.addr_clean_iff _).mp hf, (Words.addr_clean_iff _).mp ht⟩,
    _, _, _, r15⟩

end BridgeEvm
