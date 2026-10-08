import ExpiryEvm.Spec
import ExpiryEvm.AllBlocks
import ExpiryEvm.Words
import ExpiryEvm.Mem

/-! # Trace segment 1: dispatcher, non-payable guard, ABI decode (pc 0 → pc 1713) -/

namespace ExpiryEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach l2tol2Blocks

theorem seg_entry {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256}
    (hcode : I.code = l2tol2Runtime) (hsel : selectorWord I = expireSelector) :
    (RDrev l2tol2Runtime g (initState σ σ₀ g A I) ∧
      ¬ (UInt256.lt (UInt256.ofNat I.calldata.size) (UInt256.ofNat 4) = UInt256.ofNat 0 ∧
        I.weiValue = ⟨0⟩ ∧
        UInt256.isZero (UInt256.slt (UInt256.sub (UInt256.ofNat I.calldata.size) (UInt256.ofNat 4))
          (UInt256.ofNat 64)) ≠ UInt256.ofNat 0)) ∨
    (UInt256.lt (UInt256.ofNat I.calldata.size) (UInt256.ofNat 4) = UInt256.ofNat 0 ∧
      I.weiValue = ⟨0⟩ ∧
      UInt256.isZero (UInt256.slt (UInt256.sub (UInt256.ofNat I.calldata.size) (UInt256.ofNat 4))
        (UInt256.ofNat 64)) ≠ UInt256.ofNat 0 ∧
      ∃ aw k C, RD l2tol2Runtime I g (initState σ σ₀ g A I) (UInt256.ofNat 1713)
        [argTime I, argHash I, UInt256.ofNat 567, expireSelector]
        (Mem.wordsMem [⟨0⟩, ⟨0⟩, ⟨128⟩]) aw ByteArray.empty σ k C) := by
  have r0 := RD.initState (σ := σ) (σ₀ := σ₀) (A := A) (g := g) hcode
  have hsel' : UInt256.shiftRight (uInt256OfByteArray (I.calldata.readBytes (⟨0⟩ : UInt256).toNat 32))
      (UInt256.ofNat 224) = UInt256.ofNat 1983519927 := hsel
  by_cases hlt : UInt256.lt (UInt256.ofNat I.calldata.size) (UInt256.ofNat 4) = UInt256.ofNat 0
  · have r1 := l2tol2_block_0_fallthrough (R := []) (by simp) hlt r0
    have r2 := l2tol2_block_13_fallthrough (by simp) (by rw [hsel']; decide) r1
    simp only [l2tol2_block_13_fallthrough_stack] at r2
    rw [hsel'] at r2
    have r3 := l2tol2_block_29_taken (by simp) (by decide) (by kjump_dest) r2
    have r4 := l2tol2_block_87_taken (by simp) (by decide) (by kjump_dest) r3
    by_cases hv : UInt256.isZero I.weiValue = UInt256.ofNat 0
    · left
      refine ⟨l2tol2_block_544 (by simp [l2tol2_block_536_fallthrough_stack])
        (l2tol2_block_536_fallthrough (by simp) hv r4), fun hc => Words.isZero_eq0.mp hv hc.2.1⟩
    · have r5 := l2tol2_block_536_taken (by simp) hv (by kjump_dest) r4
      simp only [l2tol2_block_536_taken_stack] at r5
      have r6 := l2tol2_block_547 (by simp) (by kjump_dest) r5
      simp only [l2tol2_block_547_stack] at r6
      by_cases hcd : UInt256.isZero (UInt256.slt (UInt256.sub (UInt256.ofNat I.calldata.size)
          (UInt256.ofNat 4)) (UInt256.ofNat 64)) = UInt256.ofNat 0
      · left
        refine ⟨l2tol2_block_4262 (by simp [l2tol2_block_4248_fallthrough_stack])
          (l2tol2_block_4248_fallthrough (by simp) hcd r6), fun hc => hc.2.2 hcd⟩
      · have r7 := l2tol2_block_4248_taken (by simp) hcd (by kjump_dest) r6
        simp only [l2tol2_block_4248_taken_stack] at r7
        have r8 := l2tol2_block_4265 (by simp) (by kjump_dest) r7
        simp only [l2tol2_block_4265_stack] at r8
        have r9 := l2tol2_block_562 (by simp) (by kjump_dest) r8
        right
        have hm : l2tol2_block_0_fallthrough_memory (mem := ByteArray.empty) = solcFreePtrMem := rfl
        have h36 : (UInt256.ofNat 4 + UInt256.ofNat 32).toNat = 36 := by decide
        have h4 : (UInt256.ofNat 4).toNat = 4 := by decide
        rw [hm, h36, h4, Mem.solcFreePtrMem_eq_wordsMem] at r9
        exact ⟨hlt, Words.isZero_ne0.mp hv, hcd, _, _, _, r9⟩
  · left
    exact ⟨l2tol2_block_217 (by simp) (l2tol2_block_0_taken (by simp) hlt (by kjump_dest) r0),
      fun hc => hlt hc.1⟩

end ExpiryEvm
