import Reasoning.Reach
import ExpiryEvm.Bytecode

open Solm ABI Ethereum Ethereum.EVM
open Reasoning.Theory Reasoning.Reach

namespace l2tol2Blocks

/-- Automatically generated RD summary for bytecode block at pc 110. -/
theorem l2tol2_block_110_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 2367498639) x0) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 110) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 121) (x0 :: R) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.dup1 (by native_decide) (by evm_ov)
  have r2 := r1.push4 (UInt256.ofNat 2367498639) (by native_decide) (by evm_ov)
  have r3 := r2.eq (by native_decide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 653) (by native_decide) (by evm_ov)
  have r5 := r4.jumpiNT (by native_decide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 121)) r5 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_110_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 2367498639) x0) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 110) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 121) (x0 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_110_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 121. -/
theorem l2tol2_block_121_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 2981212681) x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 672) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 121) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 672) (x0 :: R) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.dup1 (by native_decide) (by evm_ov)
  have r2 := r1.push4 (UInt256.ofNat 2981212681) (by native_decide) (by evm_ov)
  have r3 := r2.eq (by native_decide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 672) (by native_decide) (by evm_ov)
  have r5 := r4.jumpiT (by native_decide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 672)) r5 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_121_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 2981212681) x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 672) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 121) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 672) (x0 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_121_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 121. -/
theorem l2tol2_block_121_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 2981212681) x0) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 121) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 132) (x0 :: R) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.dup1 (by native_decide) (by evm_ov)
  have r2 := r1.push4 (UInt256.ofNat 2981212681) (by native_decide) (by evm_ov)
  have r3 := r2.eq (by native_decide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 672) (by native_decide) (by evm_ov)
  have r5 := r4.jumpiNT (by native_decide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 132)) r5 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_121_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 2981212681) x0) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 121) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 132) (x0 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_121_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 132. -/
theorem l2tol2_block_132 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 132) R mem aw rdata σ k C)
    : RDrev ExpiryEvm.l2tol2Runtime g s0 := by
  let r0 := h
  have r1 := r0.push0 (by native_decide) (by evm_ov)
  have r2 := r1.dup1 (by native_decide) (by evm_ov)
  exact RD.genRev r2 (by native_decide) (by evm_ov)

/-- Automatically generated RD summary for bytecode block at pc 135. -/
theorem l2tol2_block_135_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.gt (UInt256.ofNat 1382121276) x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 194) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 135) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 194) (x0 :: R) mem aw rdata σ (k + 6) (C + ((23))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.dup1 (by native_decide) (by evm_ov)
  have r3 := r2.push4 (UInt256.ofNat 1382121276) (by native_decide) (by evm_ov)
  have r4 := r3.gt (by native_decide) (by evm_ov)
  have r5 := r4.push2 (UInt256.ofNat 194) (by native_decide) (by evm_ov)
  have r6 := r5.jumpiT (by native_decide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 194)) r6 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_135_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.gt (UInt256.ofNat 1382121276) x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 194) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 135) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 194) (x0 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_135_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 135. -/
theorem l2tol2_block_135_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.gt (UInt256.ofNat 1382121276) x0) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 135) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 147) (x0 :: R) mem aw rdata σ (k + 6) (C + ((23))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.dup1 (by native_decide) (by evm_ov)
  have r3 := r2.push4 (UInt256.ofNat 1382121276) (by native_decide) (by evm_ov)
  have r4 := r3.gt (by native_decide) (by evm_ov)
  have r5 := r4.push2 (UInt256.ofNat 194) (by native_decide) (by evm_ov)
  have r6 := r5.jumpiNT (by native_decide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 147)) r6 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_135_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.gt (UInt256.ofNat 1382121276) x0) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 135) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 147) (x0 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_135_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 147. -/
theorem l2tol2_block_147_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 1382121276) x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 359) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 147) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 359) (x0 :: R) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.dup1 (by native_decide) (by evm_ov)
  have r2 := r1.push4 (UInt256.ofNat 1382121276) (by native_decide) (by evm_ov)
  have r3 := r2.eq (by native_decide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 359) (by native_decide) (by evm_ov)
  have r5 := r4.jumpiT (by native_decide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 359)) r5 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_147_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 1382121276) x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 359) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 147) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 359) (x0 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_147_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 147. -/
theorem l2tol2_block_147_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 1382121276) x0) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 147) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 158) (x0 :: R) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.dup1 (by native_decide) (by evm_ov)
  have r2 := r1.push4 (UInt256.ofNat 1382121276) (by native_decide) (by evm_ov)
  have r3 := r2.eq (by native_decide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 359) (by native_decide) (by evm_ov)
  have r5 := r4.jumpiNT (by native_decide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 158)) r5 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_147_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 1382121276) x0) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 147) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 158) (x0 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_147_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 158. -/
theorem l2tol2_block_158_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 1425886544) x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 397) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 158) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 397) (x0 :: R) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.dup1 (by native_decide) (by evm_ov)
  have r2 := r1.push4 (UInt256.ofNat 1425886544) (by native_decide) (by evm_ov)
  have r3 := r2.eq (by native_decide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 397) (by native_decide) (by evm_ov)
  have r5 := r4.jumpiT (by native_decide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 397)) r5 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_158_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 1425886544) x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 397) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 158) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 397) (x0 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_158_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 158. -/
theorem l2tol2_block_158_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 1425886544) x0) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 158) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 169) (x0 :: R) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.dup1 (by native_decide) (by evm_ov)
  have r2 := r1.push4 (UInt256.ofNat 1425886544) (by native_decide) (by evm_ov)
  have r3 := r2.eq (by native_decide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 397) (by native_decide) (by evm_ov)
  have r5 := r4.jumpiNT (by native_decide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 169)) r5 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_158_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 1425886544) x0) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 158) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 169) (x0 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_158_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 169. -/
theorem l2tol2_block_169_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 1810414363) x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 482) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 169) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 482) (x0 :: R) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.dup1 (by native_decide) (by evm_ov)
  have r2 := r1.push4 (UInt256.ofNat 1810414363) (by native_decide) (by evm_ov)
  have r3 := r2.eq (by native_decide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 482) (by native_decide) (by evm_ov)
  have r5 := r4.jumpiT (by native_decide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 482)) r5 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_169_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 1810414363) x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 482) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 169) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 482) (x0 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_169_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 169. -/
theorem l2tol2_block_169_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 1810414363) x0) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 169) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 180) (x0 :: R) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.dup1 (by native_decide) (by evm_ov)
  have r2 := r1.push4 (UInt256.ofNat 1810414363) (by native_decide) (by evm_ov)
  have r3 := r2.eq (by native_decide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 482) (by native_decide) (by evm_ov)
  have r5 := r4.jumpiNT (by native_decide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 180)) r5 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_169_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 1810414363) x0) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 169) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 180) (x0 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_169_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 180. -/
theorem l2tol2_block_180_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 1884746783) x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 525) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 180) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 525) (x0 :: R) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.dup1 (by native_decide) (by evm_ov)
  have r2 := r1.push4 (UInt256.ofNat 1884746783) (by native_decide) (by evm_ov)
  have r3 := r2.eq (by native_decide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 525) (by native_decide) (by evm_ov)
  have r5 := r4.jumpiT (by native_decide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 525)) r5 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_180_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 1884746783) x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 525) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 180) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 525) (x0 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_180_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 180. -/
theorem l2tol2_block_180_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 1884746783) x0) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 180) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 191) (x0 :: R) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.dup1 (by native_decide) (by evm_ov)
  have r2 := r1.push4 (UInt256.ofNat 1884746783) (by native_decide) (by evm_ov)
  have r3 := r2.eq (by native_decide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 525) (by native_decide) (by evm_ov)
  have r5 := r4.jumpiNT (by native_decide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 191)) r5 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_180_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 1884746783) x0) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 180) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 191) (x0 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_180_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 191. -/
theorem l2tol2_block_191 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 191) R mem aw rdata σ k C)
    : RDrev ExpiryEvm.l2tol2Runtime g s0 := by
  let r0 := h
  have r1 := r0.push0 (by native_decide) (by evm_ov)
  have r2 := r1.dup1 (by native_decide) (by evm_ov)
  exact RD.genRev r2 (by native_decide) (by evm_ov)

/-- Automatically generated RD summary for bytecode block at pc 194. -/
theorem l2tol2_block_194_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 409891624) x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 232) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 194) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 232) (x0 :: R) mem aw rdata σ (k + 6) (C + ((23))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.dup1 (by native_decide) (by evm_ov)
  have r3 := r2.push4 (UInt256.ofNat 409891624) (by native_decide) (by evm_ov)
  have r4 := r3.eq (by native_decide) (by evm_ov)
  have r5 := r4.push2 (UInt256.ofNat 232) (by native_decide) (by evm_ov)
  have r6 := r5.jumpiT (by native_decide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 232)) r6 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_194_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 409891624) x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 232) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 194) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 232) (x0 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_194_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 194. -/
theorem l2tol2_block_194_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 409891624) x0) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 194) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 206) (x0 :: R) mem aw rdata σ (k + 6) (C + ((23))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.dup1 (by native_decide) (by evm_ov)
  have r3 := r2.push4 (UInt256.ofNat 409891624) (by native_decide) (by evm_ov)
  have r4 := r3.eq (by native_decide) (by evm_ov)
  have r5 := r4.push2 (UInt256.ofNat 232) (by native_decide) (by evm_ov)
  have r6 := r5.jumpiNT (by native_decide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 206)) r6 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_194_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 409891624) x0) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 194) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 206) (x0 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_194_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 206. -/
theorem l2tol2_block_206_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 611927138) x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 282) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 206) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 282) (x0 :: R) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.dup1 (by native_decide) (by evm_ov)
  have r2 := r1.push4 (UInt256.ofNat 611927138) (by native_decide) (by evm_ov)
  have r3 := r2.eq (by native_decide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 282) (by native_decide) (by evm_ov)
  have r5 := r4.jumpiT (by native_decide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 282)) r5 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_206_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 611927138) x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 282) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 206) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 282) (x0 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_206_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 206. -/
theorem l2tol2_block_206_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 611927138) x0) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 206) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 217) (x0 :: R) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.dup1 (by native_decide) (by evm_ov)
  have r2 := r1.push4 (UInt256.ofNat 611927138) (by native_decide) (by evm_ov)
  have r3 := r2.eq (by native_decide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 282) (by native_decide) (by evm_ov)
  have r5 := r4.jumpiNT (by native_decide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 217)) r5 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_206_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 611927138) x0) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 206) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 217) (x0 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_206_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 217. -/
theorem l2tol2_block_217_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 956292632) x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 302) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 217) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 302) (x0 :: R) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.dup1 (by native_decide) (by evm_ov)
  have r2 := r1.push4 (UInt256.ofNat 956292632) (by native_decide) (by evm_ov)
  have r3 := r2.eq (by native_decide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 302) (by native_decide) (by evm_ov)
  have r5 := r4.jumpiT (by native_decide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 302)) r5 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_217_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 956292632) x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 302) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 217) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 302) (x0 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_217_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

end l2tol2Blocks
