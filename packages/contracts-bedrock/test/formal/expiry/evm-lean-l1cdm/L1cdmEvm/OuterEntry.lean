import L1cdmEvm.Spec
import L1cdmEvm.Words
import L1cdmEvm.AllBlocks

/-! # Outer trace, segment 1: dispatcher, non-payable guard, ABI decode (pc 0 → 1751) -/

namespace L1cdmEvm

set_option maxRecDepth 100000

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach l1cdmBlocks L1cdmEvm.SymMem

theorem seg_entry {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256}
    (hcode : I.code = l1cdmRuntime) (hsel : selectorWord I = relaySelector) :
    RDrev l1cdmRuntime g (initState σ σ₀ g A I) ∨
    (I.weiValue = ⟨0⟩ ∧ UInt256.lt (UInt256.ofNat I.calldata.size) (UInt256.ofNat 4) = UInt256.ofNat 0 ∧
      UInt256.isZero (UInt256.slt (UInt256.sub (UInt256.ofNat I.calldata.size) (UInt256.ofNat 4))
        (UInt256.ofNat 64)) ≠ UInt256.ofNat 0 ∧
      ∃ m aw k C, Fmp m (UInt256.ofNat 128) ∧
        RD l1cdmRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 1751)
          [argTime I, argHash I, UInt256.ofNat 766, relaySelector] m aw ByteArray.empty σ k C) := by
  have r0 := RD.initState (σ := σ) (σ₀ := σ₀) (A := A) (g := g) hcode
  have hsel' : UInt256.shiftRight (uInt256OfByteArray (I.calldata.readBytes (UInt256.ofNat 0).toNat 32))
      (UInt256.ofNat 224) = UInt256.ofNat 0x372293c3 := hsel
  by_cases hlt : UInt256.lt (UInt256.ofNat I.calldata.size) (UInt256.ofNat 4) = UInt256.ofNat 0
  swap
  · left; exact l1cdm_block_472 (by simp) (l1cdm_block_0_taken (by simp) hlt (by kjump_dest) r0)
  have r1 := l1cdm_block_0_fallthrough (R := []) (by simp) hlt r0
  have r2 := l1cdm_block_13_taken (by simp) (by rw [hsel']; decide) (by kjump_dest) r1
  simp only [l1cdm_block_13_taken_stack, hsel'] at r2
  have r3 := l1cdm_block_258_fallthrough (by simp) (by decide) r2
  have r4 := l1cdm_block_270_taken (by simp) (by decide) (by kjump_dest) r3
  have r5 := l1cdm_block_329_taken (by simp) (by decide) (by kjump_dest) r4
  by_cases hv : UInt256.isZero I.weiValue = UInt256.ofNat 0
  · left
    exact l1cdm_block_742 (by simp [l1cdm_block_734_fallthrough_stack])
      (l1cdm_block_734_fallthrough (by simp) hv r5)
  have r6 := l1cdm_block_734_taken (by simp) hv (by kjump_dest) r5
  simp only [l1cdm_block_734_taken_stack] at r6
  have r7 := l1cdm_block_746 (by simp) (by kjump_dest) r6
  simp only [l1cdm_block_746_stack] at r7
  by_cases hcd : UInt256.isZero (UInt256.slt (UInt256.sub (UInt256.ofNat I.calldata.size)
      (UInt256.ofNat 4)) (UInt256.ofNat 64)) = UInt256.ofNat 0
  · left
    exact l1cdm_block_9278 (by simp [l1cdm_block_9263_fallthrough_stack])
      (l1cdm_block_9263_fallthrough (by simp) hcd r7)
  have r8 := l1cdm_block_9263_taken (by simp) hcd (by kjump_dest) r7
  simp only [l1cdm_block_9263_taken_stack] at r8
  have r9 := l1cdm_block_9282 (by simp) (by kjump_dest) r8
  simp only [l1cdm_block_9282_stack] at r9
  have r10 := l1cdm_block_761 (by simp) (by kjump_dest) r9
  have h36 : (UInt256.ofNat 4 + UInt256.ofNat 32).toNat = 36 := by decide
  have h4 : (UInt256.ofNat 4).toNat = 4 := by decide
  rw [h36, h4] at r10
  exact Or.inr ⟨Words.isZero_ne0.mp hv, hlt, hcd, _, _, _, _, fmp_store _ _ _ (by decide), r10⟩

end L1cdmEvm
