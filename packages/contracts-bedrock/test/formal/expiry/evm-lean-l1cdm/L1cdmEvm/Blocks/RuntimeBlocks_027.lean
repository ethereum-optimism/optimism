import Reasoning.Reach
import L1cdmEvm.Bytecode

open Solm ABI Ethereum Ethereum.EVM
open Reasoning.Theory Reasoning.Reach

namespace l1cdmBlocks

/-- Final stack for bytecode block summary `l1cdm_block_9903_taken`. -/
def l1cdm_block_9903_taken_stack {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: (x4 + x1) :: x2 :: x3 :: x4 :: x5 :: R)

/-- Automatically generated RD summary for bytecode block at pc 9903. -/
theorem l1cdm_block_9903_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hcond : (UInt256.slt ((x4 + x1) + (UInt256.ofNat 31)) x5) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9923) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9903) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9923) (l1cdm_block_9903_taken_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (R := R)) mem aw rdata σ (k + 13) (C + ((43))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup2 (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup6 (by evm_kdecide) (by evm_ov)
  have r4 := r3.add (by evm_kdecide) (by evm_ov)
  have r5 := r4.swap2 (by evm_kdecide) (by evm_ov)
  have r6 := r5.pop (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup6 (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 31) (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup4 (by evm_kdecide) (by evm_ov)
  have r10 := r9.add (by evm_kdecide) (by evm_ov)
  have r11 := r10.slt (by evm_kdecide) (by evm_ov)
  have r12 := r11.push2 (UInt256.ofNat 9923) (by evm_kdecide) (by evm_ov)
  have r13 := r12.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 9923)) r13 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_9903_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hcond : (UInt256.slt ((x4 + x1) + (UInt256.ofNat 31)) x5) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9923) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9903) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9923) (l1cdm_block_9903_taken_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_9903_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_9903_fallthrough`. -/
def l1cdm_block_9903_fallthrough_stack {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: (x4 + x1) :: x2 :: x3 :: x4 :: x5 :: R)

/-- Automatically generated RD summary for bytecode block at pc 9903. -/
theorem l1cdm_block_9903_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hcond : (UInt256.slt ((x4 + x1) + (UInt256.ofNat 31)) x5) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9903) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9919) (l1cdm_block_9903_fallthrough_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (R := R)) mem aw rdata σ (k + 13) (C + ((43))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup2 (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup6 (by evm_kdecide) (by evm_ov)
  have r4 := r3.add (by evm_kdecide) (by evm_ov)
  have r5 := r4.swap2 (by evm_kdecide) (by evm_ov)
  have r6 := r5.pop (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup6 (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 31) (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup4 (by evm_kdecide) (by evm_ov)
  have r10 := r9.add (by evm_kdecide) (by evm_ov)
  have r11 := r10.slt (by evm_kdecide) (by evm_ov)
  have r12 := r11.push2 (UInt256.ofNat 9923) (by evm_kdecide) (by evm_ov)
  have r13 := r12.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 9919)) r13 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_9903_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hcond : (UInt256.slt ((x4 + x1) + (UInt256.ofNat 31)) x5) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9903) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9919) (l1cdm_block_9903_fallthrough_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_9903_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 9919. -/
theorem l1cdm_block_9919 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9919) R mem aw rdata σ k C)
    : RDrev L1cdmEvm.l1cdmRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `l1cdm_block_9923_taken`. -/
def l1cdm_block_9923_taken_stack {ee : ExecutionEnv} {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes x1.toNat 32)) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 9923. -/
theorem l1cdm_block_9923_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes x1.toNat 32)) x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9941) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9923) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9941) (l1cdm_block_9923_taken_stack (ee := ee) (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 9) (C + ((32))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup2 (by evm_kdecide) (by evm_ov)
  have r3 := r2.calldataload (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup2 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup2 (by evm_kdecide) (by evm_ov)
  have r6 := r5.gt (by evm_kdecide) (by evm_ov)
  have r7 := r6.iszero (by evm_kdecide) (by evm_ov)
  have r8 := r7.push2 (UInt256.ofNat 9941) (by evm_kdecide) (by evm_ov)
  have r9 := r8.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 9941)) r9 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_9923_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes x1.toNat 32)) x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9941) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9923) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9941) (l1cdm_block_9923_taken_stack (ee := ee) (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_9923_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_9923_fallthrough`. -/
def l1cdm_block_9923_fallthrough_stack {ee : ExecutionEnv} {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes x1.toNat 32)) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 9923. -/
theorem l1cdm_block_9923_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes x1.toNat 32)) x0)) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9923) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9934) (l1cdm_block_9923_fallthrough_stack (ee := ee) (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 9) (C + ((32))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup2 (by evm_kdecide) (by evm_ov)
  have r3 := r2.calldataload (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup2 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup2 (by evm_kdecide) (by evm_ov)
  have r6 := r5.gt (by evm_kdecide) (by evm_ov)
  have r7 := r6.iszero (by evm_kdecide) (by evm_ov)
  have r8 := r7.push2 (UInt256.ofNat 9941) (by evm_kdecide) (by evm_ov)
  have r9 := r8.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 9934)) r9 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_9923_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes x1.toNat 32)) x0)) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9923) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9934) (l1cdm_block_9923_fallthrough_stack (ee := ee) (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_9923_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_9934`. -/
def l1cdm_block_9934_stack {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 9941) :: R)

/-- Automatically generated RD summary for bytecode block at pc 9934. -/
theorem l1cdm_block_9934 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9813) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9934) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9813) (l1cdm_block_9934_stack (R := R)) mem aw rdata σ (k + 3) (C + ((14))) := by
  let r0 := h
  have r1 := r0.push2 (UInt256.ofNat 9941) (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 9813) (by evm_kdecide) (by evm_ov)
  have r3 := r2.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 9813)) r3 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_9934_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9813) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9934) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9813) (l1cdm_block_9934_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_9934 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_9941_taken`. -/
def l1cdm_block_9941_taken_stack {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((memLoad (UInt256.ofNat 64) mem) :: ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 9941. -/
theorem l1cdm_block_9941_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.lor (UInt256.lt ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) (memLoad (UInt256.ofNat 64) mem)) (UInt256.gt ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) x1))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10011) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9941) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10011) (l1cdm_block_9941_taken_stack (mem := mem) (x0 := x0) (x1 := x1) (R := R)) mem (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 26) (C + ((83) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r3 := RD.genMload r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 31) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup3 (by evm_kdecide) (by evm_ov)
  have r6 := r5.add (by evm_kdecide) (by evm_ov)
  have r7 := r6.pushConst (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r8 := r7.swap1 (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup2 (by evm_kdecide) (by evm_ov)
  have r10 := r9.and (by evm_kdecide) (by evm_ov)
  have r11 := r10.push1 (UInt256.ofNat 63) (by evm_kdecide) (by evm_ov)
  have r12 := r11.add (by evm_kdecide) (by evm_ov)
  have r13 := r12.and (by evm_kdecide) (by evm_ov)
  have r14 := r13.dup2 (by evm_kdecide) (by evm_ov)
  have r15 := r14.add (by evm_kdecide) (by evm_ov)
  have r16 := r15.swap1 (by evm_kdecide) (by evm_ov)
  have r17 := r16.dup4 (by evm_kdecide) (by evm_ov)
  have r18 := r17.dup3 (by evm_kdecide) (by evm_ov)
  have r19 := r18.gt (by evm_kdecide) (by evm_ov)
  have r20 := r19.dup2 (by evm_kdecide) (by evm_ov)
  have r21 := r20.dup4 (by evm_kdecide) (by evm_ov)
  have r22 := r21.lt (by evm_kdecide) (by evm_ov)
  have r23 := r22.or (by evm_kdecide) (by evm_ov)
  have r24 := r23.iszero (by evm_kdecide) (by evm_ov)
  have r25 := r24.push2 (UInt256.ofNat 10011) (by evm_kdecide) (by evm_ov)
  have r26 := r25.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10011)) r26 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_9941_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.lor (UInt256.lt ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) (memLoad (UInt256.ofNat 64) mem)) (UInt256.gt ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) x1))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10011) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9941) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10011) (l1cdm_block_9941_taken_stack (mem := mem) (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_9941_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_9941_fallthrough`. -/
def l1cdm_block_9941_fallthrough_stack {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((memLoad (UInt256.ofNat 64) mem) :: ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 9941. -/
theorem l1cdm_block_9941_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.lor (UInt256.lt ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) (memLoad (UInt256.ofNat 64) mem)) (UInt256.gt ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) x1))) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9941) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10004) (l1cdm_block_9941_fallthrough_stack (mem := mem) (x0 := x0) (x1 := x1) (R := R)) mem (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 26) (C + ((83) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r3 := RD.genMload r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 31) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup3 (by evm_kdecide) (by evm_ov)
  have r6 := r5.add (by evm_kdecide) (by evm_ov)
  have r7 := r6.pushConst (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r8 := r7.swap1 (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup2 (by evm_kdecide) (by evm_ov)
  have r10 := r9.and (by evm_kdecide) (by evm_ov)
  have r11 := r10.push1 (UInt256.ofNat 63) (by evm_kdecide) (by evm_ov)
  have r12 := r11.add (by evm_kdecide) (by evm_ov)
  have r13 := r12.and (by evm_kdecide) (by evm_ov)
  have r14 := r13.dup2 (by evm_kdecide) (by evm_ov)
  have r15 := r14.add (by evm_kdecide) (by evm_ov)
  have r16 := r15.swap1 (by evm_kdecide) (by evm_ov)
  have r17 := r16.dup4 (by evm_kdecide) (by evm_ov)
  have r18 := r17.dup3 (by evm_kdecide) (by evm_ov)
  have r19 := r18.gt (by evm_kdecide) (by evm_ov)
  have r20 := r19.dup2 (by evm_kdecide) (by evm_ov)
  have r21 := r20.dup4 (by evm_kdecide) (by evm_ov)
  have r22 := r21.lt (by evm_kdecide) (by evm_ov)
  have r23 := r22.or (by evm_kdecide) (by evm_ov)
  have r24 := r23.iszero (by evm_kdecide) (by evm_ov)
  have r25 := r24.push2 (UInt256.ofNat 10011) (by evm_kdecide) (by evm_ov)
  have r26 := r25.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10004)) r26 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_9941_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.lor (UInt256.lt ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) (memLoad (UInt256.ofNat 64) mem)) (UInt256.gt ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) x1))) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9941) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10004) (l1cdm_block_9941_fallthrough_stack (mem := mem) (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_9941_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10004`. -/
def l1cdm_block_10004_stack {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 10011) :: R)

/-- Automatically generated RD summary for bytecode block at pc 10004. -/
theorem l1cdm_block_10004 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9813) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10004) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9813) (l1cdm_block_10004_stack (R := R)) mem aw rdata σ (k + 3) (C + ((14))) := by
  let r0 := h
  have r1 := r0.push2 (UInt256.ofNat 10011) (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 9813) (by evm_kdecide) (by evm_ov)
  have r3 := r2.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 9813)) r3 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10004_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9813) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10004) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9813) (l1cdm_block_10004_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10004 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final memory for bytecode block summary `l1cdm_block_10011_taken`. -/
def l1cdm_block_10011_taken_memory {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} : ByteArray :=
  (x2.toByteArray.write 0 (x1.toByteArray.write 0 mem (UInt256.ofNat 64).toNat 32) x0.toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 10011. -/
theorem l1cdm_block_10011_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt ((x4 + x2) + (UInt256.ofNat 32)) x8)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10036) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10011) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10036) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) (l1cdm_block_10011_taken_memory (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2)) (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) x0 (⟨32⟩ : UInt256)) rdata σ (k + 17) (C + ((56) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) x0 (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup2 (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r4 := RD.genMstore r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup3 (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup2 (by evm_kdecide) (by evm_ov)
  have r7 := RD.genMstore r6 (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup9 (by evm_kdecide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup5 (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup8 (by evm_kdecide) (by evm_ov)
  have r12 := r11.add (by evm_kdecide) (by evm_ov)
  have r13 := r12.add (by evm_kdecide) (by evm_ov)
  have r14 := r13.gt (by evm_kdecide) (by evm_ov)
  have r15 := r14.iszero (by evm_kdecide) (by evm_ov)
  have r16 := r15.push2 (UInt256.ofNat 10036) (by evm_kdecide) (by evm_ov)
  have r17 := r16.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10036)) r17 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10011_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt ((x4 + x2) + (UInt256.ofNat 32)) x8)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10036) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10011) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10036) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) (l1cdm_block_10011_taken_memory (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10011_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final memory for bytecode block summary `l1cdm_block_10011_fallthrough`. -/
def l1cdm_block_10011_fallthrough_memory {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} : ByteArray :=
  (x2.toByteArray.write 0 (x1.toByteArray.write 0 mem (UInt256.ofNat 64).toNat 32) x0.toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 10011. -/
theorem l1cdm_block_10011_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt ((x4 + x2) + (UInt256.ofNat 32)) x8)) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10011) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10032) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) (l1cdm_block_10011_fallthrough_memory (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2)) (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) x0 (⟨32⟩ : UInt256)) rdata σ (k + 17) (C + ((56) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) x0 (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup2 (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r4 := RD.genMstore r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup3 (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup2 (by evm_kdecide) (by evm_ov)
  have r7 := RD.genMstore r6 (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup9 (by evm_kdecide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup5 (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup8 (by evm_kdecide) (by evm_ov)
  have r12 := r11.add (by evm_kdecide) (by evm_ov)
  have r13 := r12.add (by evm_kdecide) (by evm_ov)
  have r14 := r13.gt (by evm_kdecide) (by evm_ov)
  have r15 := r14.iszero (by evm_kdecide) (by evm_ov)
  have r16 := r15.push2 (UInt256.ofNat 10036) (by evm_kdecide) (by evm_ov)
  have r17 := r16.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10032)) r17 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10011_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt ((x4 + x2) + (UInt256.ofNat 32)) x8)) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10011) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10032) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) (l1cdm_block_10011_fallthrough_memory (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10011_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 10032. -/
theorem l1cdm_block_10032 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10032) R mem aw rdata σ k C)
    : RDrev L1cdmEvm.l1cdmRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `l1cdm_block_10036`. -/
def l1cdm_block_10036_stack {x0 : UInt256} {x5 : UInt256} {x7 : UInt256} {R : List UInt256} : List UInt256 :=
  ((x7 + (UInt256.ofNat 32)) :: (UInt256.ofNat 10075) :: x5 :: x0 :: x7 :: R)

/-- Final memory for bytecode block summary `l1cdm_block_10036`. -/
def l1cdm_block_10036_memory {ee : ExecutionEnv} {mem : ByteArray} {x0 : UInt256} {x2 : UInt256} {x4 : UInt256} : ByteArray :=
  ((UInt256.ofNat 0).toByteArray.write 0 (ee.calldata.write (x4 + (UInt256.ofNat 32)).toNat mem (x0 + (UInt256.ofNat 32)).toNat x2.toNat) ((x0 + x2) + (UInt256.ofNat 32)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 10036. -/
theorem l1cdm_block_10036 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9477) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10036) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9477) (l1cdm_block_10036_stack (x0 := x0) (x5 := x5) (x7 := x7) (R := R)) (l1cdm_block_10036_memory (ee := ee) (mem := mem) (x0 := x0) (x2 := x2) (x4 := x4)) (M (M aw (x0 + (UInt256.ofNat 32)) x2) ((x0 + x2) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) rdata σ (k + 30) (C + ((84) + (memExpansionCost aw (x0 + (UInt256.ofNat 32)) x2) + (3 + 3 * ((x2.toNat + 31) / 32)) + (memExpansionCost (M aw (x0 + (UInt256.ofNat 32)) x2) ((x0 + x2) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup3 (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup7 (by evm_kdecide) (by evm_ov)
  have r5 := r4.add (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup4 (by evm_kdecide) (by evm_ov)
  have r8 := r7.add (by evm_kdecide) (by evm_ov)
  have r9 := RD.genCalldatacopy r8 (by evm_kdecide) (by evm_ov)
  have r10 := r9.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r11 := r10.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r12 := r11.dup5 (by evm_kdecide) (by evm_ov)
  have r13 := r12.dup4 (by evm_kdecide) (by evm_ov)
  have r14 := r13.add (by evm_kdecide) (by evm_ov)
  have r15 := r14.add (by evm_kdecide) (by evm_ov)
  have r16 := RD.genMstore r15 (by evm_kdecide) (by evm_ov)
  have r17 := r16.dup1 (by evm_kdecide) (by evm_ov)
  have r18 := r17.swap7 (by evm_kdecide) (by evm_ov)
  have r19 := r18.pop (by evm_kdecide) (by evm_ov)
  have r20 := r19.pop (by evm_kdecide) (by evm_ov)
  have r21 := r20.pop (by evm_kdecide) (by evm_ov)
  have r22 := r21.pop (by evm_kdecide) (by evm_ov)
  have r23 := r22.pop (by evm_kdecide) (by evm_ov)
  have r24 := r23.pop (by evm_kdecide) (by evm_ov)
  have r25 := r24.push2 (UInt256.ofNat 10075) (by evm_kdecide) (by evm_ov)
  have r26 := r25.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r27 := r26.dup5 (by evm_kdecide) (by evm_ov)
  have r28 := r27.add (by evm_kdecide) (by evm_ov)
  have r29 := r28.push2 (UInt256.ofNat 9477) (by evm_kdecide) (by evm_ov)
  have r30 := r29.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 9477)) r30 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10036_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9477) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10036) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9477) (l1cdm_block_10036_stack (x0 := x0) (x5 := x5) (x7 := x7) (R := R)) (l1cdm_block_10036_memory (ee := ee) (mem := mem) (x0 := x0) (x2 := x2) (x4 := x4)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10036 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10075`. -/
def l1cdm_block_10075_stack {x0 : UInt256} {x2 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: x2 :: R)

/-- Automatically generated RD summary for bytecode block at pc 10075. -/
theorem l1cdm_block_10075 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x5 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10075) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 x5 (l1cdm_block_10075_stack (x0 := x0) (x2 := x2) (R := R)) mem aw rdata σ (k + 9) (C + ((27))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pop (by evm_kdecide) (by evm_ov)
  have r4 := r3.swap3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.pop (by evm_kdecide) (by evm_ov)
  have r6 := r5.swap3 (by evm_kdecide) (by evm_ov)
  have r7 := r6.swap1 (by evm_kdecide) (by evm_ov)
  have r8 := r7.pop (by evm_kdecide) (by evm_ov)
  have r9 := r8.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r9 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10075_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x5 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10075) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 x5 (l1cdm_block_10075_stack (x0 := x0) (x2 := x2) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10075 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10084_taken`. -/
def l1cdm_block_10084_taken_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 10084. -/
theorem l1cdm_block_10084_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 192))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10111) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10084) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10111) (l1cdm_block_10084_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 16) (C + ((53))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup1 (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 192) (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup9 (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup11 (by evm_kdecide) (by evm_ov)
  have r12 := r11.sub (by evm_kdecide) (by evm_ov)
  have r13 := r12.slt (by evm_kdecide) (by evm_ov)
  have r14 := r13.iszero (by evm_kdecide) (by evm_ov)
  have r15 := r14.push2 (UInt256.ofNat 10111) (by evm_kdecide) (by evm_ov)
  have r16 := r15.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10111)) r16 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10084_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 192))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10111) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10084) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10111) (l1cdm_block_10084_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10084_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10084_fallthrough`. -/
def l1cdm_block_10084_fallthrough_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 10084. -/
theorem l1cdm_block_10084_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 192))) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10084) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10107) (l1cdm_block_10084_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 16) (C + ((53))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup1 (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 192) (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup9 (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup11 (by evm_kdecide) (by evm_ov)
  have r12 := r11.sub (by evm_kdecide) (by evm_ov)
  have r13 := r12.slt (by evm_kdecide) (by evm_ov)
  have r14 := r13.iszero (by evm_kdecide) (by evm_ov)
  have r15 := r14.push2 (UInt256.ofNat 10111) (by evm_kdecide) (by evm_ov)
  have r16 := r15.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10107)) r16 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10084_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 192))) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10084) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10107) (l1cdm_block_10084_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10084_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 10107. -/
theorem l1cdm_block_10107 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10107) R mem aw rdata σ k C)
    : RDrev L1cdmEvm.l1cdmRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `l1cdm_block_10111`. -/
def l1cdm_block_10111_stack {ee : ExecutionEnv} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {x7 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes (x7 + (UInt256.ofNat 32)).toNat 32)) :: (UInt256.ofNat 10129) :: (uInt256OfByteArray (ee.calldata.readBytes (x7 + (UInt256.ofNat 32)).toNat 32)) :: x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: (uInt256OfByteArray (ee.calldata.readBytes x7.toNat 32)) :: x7 :: R)

/-- Automatically generated RD summary for bytecode block at pc 10111. -/
theorem l1cdm_block_10111 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9367) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10111) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9367) (l1cdm_block_10111_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x7 := x7) (R := R)) mem aw rdata σ (k + 13) (C + ((41))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup8 (by evm_kdecide) (by evm_ov)
  have r3 := r2.calldataload (by evm_kdecide) (by evm_ov)
  have r4 := r3.swap7 (by evm_kdecide) (by evm_ov)
  have r5 := r4.pop (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup9 (by evm_kdecide) (by evm_ov)
  have r8 := r7.add (by evm_kdecide) (by evm_ov)
  have r9 := r8.calldataload (by evm_kdecide) (by evm_ov)
  have r10 := r9.push2 (UInt256.ofNat 10129) (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup2 (by evm_kdecide) (by evm_ov)
  have r12 := r11.push2 (UInt256.ofNat 9367) (by evm_kdecide) (by evm_ov)
  have r13 := r12.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 9367)) r13 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10111_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9367) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10111) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9367) (l1cdm_block_10111_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x7 := x7) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10111 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10129`. -/
def l1cdm_block_10129_stack {ee : ExecutionEnv} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {x7 : UInt256} {x8 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes (x8 + (UInt256.ofNat 64)).toNat 32)) :: (UInt256.ofNat 10145) :: (uInt256OfByteArray (ee.calldata.readBytes (x8 + (UInt256.ofNat 64)).toNat 32)) :: x1 :: x2 :: x3 :: x4 :: x5 :: x0 :: x7 :: x8 :: R)

/-- Automatically generated RD summary for bytecode block at pc 10129. -/
theorem l1cdm_block_10129 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9367) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10129) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9367) (l1cdm_block_10129_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x7 := x7) (x8 := x8) (R := R)) mem aw rdata σ (k + 11) (C + ((35))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap6 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pop (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup9 (by evm_kdecide) (by evm_ov)
  have r6 := r5.add (by evm_kdecide) (by evm_ov)
  have r7 := r6.calldataload (by evm_kdecide) (by evm_ov)
  have r8 := r7.push2 (UInt256.ofNat 10145) (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup2 (by evm_kdecide) (by evm_ov)
  have r10 := r9.push2 (UInt256.ofNat 9367) (by evm_kdecide) (by evm_ov)
  have r11 := r10.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 9367)) r11 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10129_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9367) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10129) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9367) (l1cdm_block_10129_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x7 := x7) (x8 := x8) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10129 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10145_taken`. -/
def l1cdm_block_10145_taken_stack {ee : ExecutionEnv} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x6 : UInt256} {x7 : UInt256} {x8 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes (x8 + (UInt256.ofNat 160)).toNat 32)) :: x1 :: x2 :: (uInt256OfByteArray (ee.calldata.readBytes (x8 + (UInt256.ofNat 128)).toNat 32)) :: (uInt256OfByteArray (ee.calldata.readBytes (x8 + (UInt256.ofNat 96)).toNat 32)) :: x0 :: x6 :: x7 :: x8 :: R)

/-- Automatically generated RD summary for bytecode block at pc 10145. -/
theorem l1cdm_block_10145_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 11 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes (x8 + (UInt256.ofNat 160)).toNat 32)) (UInt256.ofNat 18446744073709551615))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10187) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10145) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10187) (l1cdm_block_10145_taken_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x6 := x6) (x7 := x7) (x8 := x8) (R := R)) mem aw rdata σ (k + 25) (C + ((77))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap5 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pop (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 96) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup9 (by evm_kdecide) (by evm_ov)
  have r6 := r5.add (by evm_kdecide) (by evm_ov)
  have r7 := r6.calldataload (by evm_kdecide) (by evm_ov)
  have r8 := r7.swap4 (by evm_kdecide) (by evm_ov)
  have r9 := r8.pop (by evm_kdecide) (by evm_ov)
  have r10 := r9.push1 (UInt256.ofNat 128) (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup9 (by evm_kdecide) (by evm_ov)
  have r12 := r11.add (by evm_kdecide) (by evm_ov)
  have r13 := r12.calldataload (by evm_kdecide) (by evm_ov)
  have r14 := r13.swap3 (by evm_kdecide) (by evm_ov)
  have r15 := r14.pop (by evm_kdecide) (by evm_ov)
  have r16 := r15.push1 (UInt256.ofNat 160) (by evm_kdecide) (by evm_ov)
  have r17 := r16.dup9 (by evm_kdecide) (by evm_ov)
  have r18 := r17.add (by evm_kdecide) (by evm_ov)
  have r19 := r18.calldataload (by evm_kdecide) (by evm_ov)
  have r20 := r19.pushConst (UInt256.ofNat 18446744073709551615) (width := 8) (op := .PUSH8) (by decide) (by evm_kdecide) (by evm_ov)
  have r21 := r20.dup2 (by evm_kdecide) (by evm_ov)
  have r22 := r21.gt (by evm_kdecide) (by evm_ov)
  have r23 := r22.iszero (by evm_kdecide) (by evm_ov)
  have r24 := r23.push2 (UInt256.ofNat 10187) (by evm_kdecide) (by evm_ov)
  have r25 := r24.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10187)) r25 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10145_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 11 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes (x8 + (UInt256.ofNat 160)).toNat 32)) (UInt256.ofNat 18446744073709551615))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10187) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10145) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10187) (l1cdm_block_10145_taken_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x6 := x6) (x7 := x7) (x8 := x8) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10145_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

end l1cdmBlocks
