import Reasoning.Reach
import ExporterEvm.Bytecode

open Solm ABI Ethereum Ethereum.EVM
open Reasoning.Theory Reasoning.Reach

namespace exporterBlocks

/-- Final stack for bytecode block summary `exporter_block_0_taken`. -/
def exporter_block_0_taken_stack {ee : ExecutionEnv} {R : List UInt256} : List UInt256 :=
  (ee.weiValue :: R)

/-- Final memory for bytecode block summary `exporter_block_0_taken`. -/
def exporter_block_0_taken_memory {mem : ByteArray} : ByteArray :=
  ((UInt256.ofNat 128).toByteArray.write 0 mem (UInt256.ofNat 64).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 0. -/
theorem exporter_block_0_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 16) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 0) R mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 16) (exporter_block_0_taken_stack (ee := ee) (R := R)) (exporter_block_0_taken_memory (mem := mem)) (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 8) (C + ((30) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 128) (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r3 := RD.genMstore r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.callvalue (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.iszero (by evm_kdecide) (by evm_ov)
  have r7 := r6.push2 (UInt256.ofNat 16) (by evm_kdecide) (by evm_ov)
  have r8 := r7.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 16)) r8 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_0_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 16) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 0) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 16) (exporter_block_0_taken_stack (ee := ee) (R := R)) (exporter_block_0_taken_memory (mem := mem)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_0_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_0_fallthrough`. -/
def exporter_block_0_fallthrough_stack {ee : ExecutionEnv} {R : List UInt256} : List UInt256 :=
  (ee.weiValue :: R)

/-- Final memory for bytecode block summary `exporter_block_0_fallthrough`. -/
def exporter_block_0_fallthrough_memory {mem : ByteArray} : ByteArray :=
  ((UInt256.ofNat 128).toByteArray.write 0 mem (UInt256.ofNat 64).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 0. -/
theorem exporter_block_0_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 0) R mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 12) (exporter_block_0_fallthrough_stack (ee := ee) (R := R)) (exporter_block_0_fallthrough_memory (mem := mem)) (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 8) (C + ((30) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 128) (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r3 := RD.genMstore r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.callvalue (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.iszero (by evm_kdecide) (by evm_ov)
  have r7 := r6.push2 (UInt256.ofNat 16) (by evm_kdecide) (by evm_ov)
  have r8 := r7.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 12)) r8 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_0_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 0) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 12) (exporter_block_0_fallthrough_stack (ee := ee) (R := R)) (exporter_block_0_fallthrough_memory (mem := mem)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_0_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 12. -/
theorem exporter_block_12 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 12) R mem aw rdata σ k C)
    : RDrev ExporterEvm.exporterRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `exporter_block_16_taken`. -/
def exporter_block_16_taken_stack {R : List UInt256} : List UInt256 :=
  R

/-- Automatically generated RD summary for bytecode block at pc 16. -/
theorem exporter_block_16_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (UInt256.lt (UInt256.ofNat ee.calldata.size) (UInt256.ofNat 4)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 54) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 16) (x0 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 54) (exporter_block_16_taken_stack (R := R)) mem aw rdata σ (k + 7) (C + ((24))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r4 := r3.calldatasize (by evm_kdecide) (by evm_ov)
  have r5 := r4.lt (by evm_kdecide) (by evm_ov)
  have r6 := r5.push2 (UInt256.ofNat 54) (by evm_kdecide) (by evm_ov)
  have r7 := r6.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 54)) r7 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_16_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (UInt256.lt (UInt256.ofNat ee.calldata.size) (UInt256.ofNat 4)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 54) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 16) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 54) (exporter_block_16_taken_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_16_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_16_fallthrough`. -/
def exporter_block_16_fallthrough_stack {R : List UInt256} : List UInt256 :=
  R

/-- Automatically generated RD summary for bytecode block at pc 16. -/
theorem exporter_block_16_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (UInt256.lt (UInt256.ofNat ee.calldata.size) (UInt256.ofNat 4)) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 16) (x0 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 26) (exporter_block_16_fallthrough_stack (R := R)) mem aw rdata σ (k + 7) (C + ((24))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r4 := r3.calldatasize (by evm_kdecide) (by evm_ov)
  have r5 := r4.lt (by evm_kdecide) (by evm_ov)
  have r6 := r5.push2 (UInt256.ofNat 54) (by evm_kdecide) (by evm_ov)
  have r7 := r6.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 26)) r7 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_16_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (UInt256.lt (UInt256.ofNat ee.calldata.size) (UInt256.ofNat 4)) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 16) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 26) (exporter_block_16_fallthrough_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_16_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_26_taken`. -/
def exporter_block_26_taken_stack {ee : ExecutionEnv} {R : List UInt256} : List UInt256 :=
  ((UInt256.shiftRight (uInt256OfByteArray (ee.calldata.readBytes (UInt256.ofNat 0).toNat 32)) (UInt256.ofNat 224)) :: R)

/-- Automatically generated RD summary for bytecode block at pc 26. -/
theorem exporter_block_26_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 409891624) (UInt256.shiftRight (uInt256OfByteArray (ee.calldata.readBytes (UInt256.ofNat 0).toNat 32)) (UInt256.ofNat 224))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 59) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 26) R mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 59) (exporter_block_26_taken_stack (ee := ee) (R := R)) mem aw rdata σ (k + 9) (C + ((34))) := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.calldataload (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 224) (by evm_kdecide) (by evm_ov)
  have r4 := r3.shr (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push4 (UInt256.ofNat 409891624) (by evm_kdecide) (by evm_ov)
  have r7 := r6.eq (by evm_kdecide) (by evm_ov)
  have r8 := r7.push2 (UInt256.ofNat 59) (by evm_kdecide) (by evm_ov)
  have r9 := r8.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 59)) r9 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_26_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 409891624) (UInt256.shiftRight (uInt256OfByteArray (ee.calldata.readBytes (UInt256.ofNat 0).toNat 32)) (UInt256.ofNat 224))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 59) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 26) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 59) (exporter_block_26_taken_stack (ee := ee) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_26_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_26_fallthrough`. -/
def exporter_block_26_fallthrough_stack {ee : ExecutionEnv} {R : List UInt256} : List UInt256 :=
  ((UInt256.shiftRight (uInt256OfByteArray (ee.calldata.readBytes (UInt256.ofNat 0).toNat 32)) (UInt256.ofNat 224)) :: R)

/-- Automatically generated RD summary for bytecode block at pc 26. -/
theorem exporter_block_26_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 409891624) (UInt256.shiftRight (uInt256OfByteArray (ee.calldata.readBytes (UInt256.ofNat 0).toNat 32)) (UInt256.ofNat 224))) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 26) R mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 43) (exporter_block_26_fallthrough_stack (ee := ee) (R := R)) mem aw rdata σ (k + 9) (C + ((34))) := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.calldataload (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 224) (by evm_kdecide) (by evm_ov)
  have r4 := r3.shr (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push4 (UInt256.ofNat 409891624) (by evm_kdecide) (by evm_ov)
  have r7 := r6.eq (by evm_kdecide) (by evm_ov)
  have r8 := r7.push2 (UInt256.ofNat 59) (by evm_kdecide) (by evm_ov)
  have r9 := r8.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 43)) r9 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_26_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 409891624) (UInt256.shiftRight (uInt256OfByteArray (ee.calldata.readBytes (UInt256.ofNat 0).toNat 32)) (UInt256.ofNat 224))) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 26) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 43) (exporter_block_26_fallthrough_stack (ee := ee) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_26_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 43. -/
theorem exporter_block_43_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 1425886544) x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 97) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 43) (x0 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 97) (x0 :: R) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.dup1 (by evm_kdecide) (by evm_ov)
  have r2 := r1.push4 (UInt256.ofNat 1425886544) (by evm_kdecide) (by evm_ov)
  have r3 := r2.eq (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 97) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 97)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_43_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 1425886544) x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 97) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 43) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 97) (x0 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_43_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 43. -/
theorem exporter_block_43_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 1425886544) x0) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 43) (x0 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 54) (x0 :: R) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.dup1 (by evm_kdecide) (by evm_ov)
  have r2 := r1.push4 (UInt256.ofNat 1425886544) (by evm_kdecide) (by evm_ov)
  have r3 := r2.eq (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 97) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 54)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_43_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 1425886544) x0) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 43) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 54) (x0 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_43_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 54. -/
theorem exporter_block_54 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 54) R mem aw rdata σ k C)
    : RDrev ExporterEvm.exporterRuntime g s0 := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r3 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `exporter_block_59`. -/
def exporter_block_59_stack {ee : ExecutionEnv} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 4) :: (UInt256.ofNat ee.calldata.size) :: (UInt256.ofNat 73) :: (UInt256.ofNat 78) :: R)

/-- Automatically generated RD summary for bytecode block at pc 59. -/
theorem exporter_block_59 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 932) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 59) R mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 932) (exporter_block_59_stack (ee := ee) (R := R)) mem aw rdata σ (k + 7) (C + ((23))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 78) (by evm_kdecide) (by evm_ov)
  have r3 := r2.push2 (UInt256.ofNat 73) (by evm_kdecide) (by evm_ov)
  have r4 := r3.calldatasize (by evm_kdecide) (by evm_ov)
  have r5 := r4.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r6 := r5.push2 (UInt256.ofNat 932) (by evm_kdecide) (by evm_ov)
  have r7 := r6.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 932)) r7 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_59_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 932) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 59) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 932) (exporter_block_59_stack (ee := ee) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_59 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 73. -/
theorem exporter_block_73 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 1 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 170) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 73) R mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 170) R mem aw rdata σ (k + 3) (C + ((12))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 170) (by evm_kdecide) (by evm_ov)
  have r3 := r2.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 170)) r3 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_73_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 1 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 170) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 73) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 170) R mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_73 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_78`. -/
def exporter_block_78_stack {mem : ByteArray} {R : List UInt256} : List UInt256 :=
  (((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) :: R)

/-- Final memory for bytecode block summary `exporter_block_78`. -/
def exporter_block_78_memory {mem : ByteArray} {x0 : UInt256} : ByteArray :=
  (x0.toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 78. -/
theorem exporter_block_78 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 78) (x0 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 88) (exporter_block_78_stack (mem := mem) (R := R)) (exporter_block_78_memory (mem := mem) (x0 := x0)) (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) rdata σ (k + 8) (C + ((22) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r3 := RD.genMload r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.swap1 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup2 (by evm_kdecide) (by evm_ov)
  have r6 := RD.genMstore r5 (by evm_kdecide) (by evm_ov)
  have r7 := r6.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r8 := r7.add (by evm_kdecide) (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 88)) r8 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_78_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 78) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 88) (exporter_block_78_stack (mem := mem) (R := R)) (exporter_block_78_memory (mem := mem) (x0 := x0)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_78 hstack h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 88. -/
theorem exporter_block_88 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 88) (x0 :: R) mem aw rdata σ k C)
    : RDret ExporterEvm.exporterRuntime g s0 σ (mem.readWithPadding (memLoad (UInt256.ofNat 64) mem).toNat (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem)).toNat) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r3 := RD.genMload r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup1 (by evm_kdecide) (by evm_ov)
  have r5 := r4.swap2 (by evm_kdecide) (by evm_ov)
  have r6 := r5.sub (by evm_kdecide) (by evm_ov)
  have r7 := r6.swap1 (by evm_kdecide) (by evm_ov)
  exact RD.genRet r7 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `exporter_block_97`. -/
def exporter_block_97_stack {mem : ByteArray} {R : List UInt256} : List UInt256 :=
  ((memLoad (UInt256.ofNat 64) mem) :: (UInt256.ofNat 157) :: R)

/-- Final memory for bytecode block summary `exporter_block_97`. -/
def exporter_block_97_memory {mem : ByteArray} : ByteArray :=
  ((UInt256.ofNat 22244937074597041345535687918817492743208931653510314877398361346409823731712).toByteArray.write 0 ((UInt256.ofNat 5).toByteArray.write 0 (((UInt256.ofNat 64) + (memLoad (UInt256.ofNat 64) mem)).toByteArray.write 0 mem (UInt256.ofNat 64).toNat 32) (memLoad (UInt256.ofNat 64) mem).toNat 32) ((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 97. -/
theorem exporter_block_97 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 157) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 97) R mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 157) (exporter_block_97_stack (mem := mem) (R := R)) (exporter_block_97_memory (mem := mem)) (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) (⟨32⟩ : UInt256)) rdata σ (k + 21) (C + ((65) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 157) (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r4 := RD.genMload r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r7 := r6.add (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r9 := RD.genMstore r8 (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup1 (by evm_kdecide) (by evm_ov)
  have r11 := r10.push1 (UInt256.ofNat 5) (by evm_kdecide) (by evm_ov)
  have r12 := r11.dup2 (by evm_kdecide) (by evm_ov)
  have r13 := RD.genMstore r12 (by evm_kdecide) (by evm_ov)
  have r14 := r13.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r15 := r14.add (by evm_kdecide) (by evm_ov)
  have r16 := r15.pushConst (UInt256.ofNat 22244937074597041345535687918817492743208931653510314877398361346409823731712) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r17 := r16.dup2 (by evm_kdecide) (by evm_ov)
  have r18 := RD.genMstore r17 (by evm_kdecide) (by evm_ov)
  have r19 := r18.pop (by evm_kdecide) (by evm_ov)
  have r20 := r19.dup2 (by evm_kdecide) (by evm_ov)
  have r21 := r20.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 157)) r21 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_97_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 157) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 97) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 157) (exporter_block_97_stack (mem := mem) (R := R)) (exporter_block_97_memory (mem := mem)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_97 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_157`. -/
def exporter_block_157_stack {mem : ByteArray} {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((memLoad (UInt256.ofNat 64) mem) :: x0 :: (UInt256.ofNat 88) :: R)

/-- Automatically generated RD summary for bytecode block at pc 157. -/
theorem exporter_block_157 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1239) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 157) (x0 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1239) (exporter_block_157_stack (mem := mem) (x0 := x0) (R := R)) mem (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 8) (C + ((27) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r3 := RD.genMload r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 88) (by evm_kdecide) (by evm_ov)
  have r5 := r4.swap2 (by evm_kdecide) (by evm_ov)
  have r6 := r5.swap1 (by evm_kdecide) (by evm_ov)
  have r7 := r6.push2 (UInt256.ofNat 1239) (by evm_kdecide) (by evm_ov)
  have r8 := r7.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1239)) r8 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_157_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1239) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 157) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1239) (exporter_block_157_stack (mem := mem) (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_157 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_170`. -/
def exporter_block_170_stack {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {x6 : UInt256} {R : List UInt256} : List UInt256 :=
  ((memLoad (UInt256.ofNat 64) mem) :: x3 :: x4 :: x5 :: x6 :: (UInt256.ofNat Ethereum.chainId) :: (UInt256.ofNat 240) :: (UInt256.ofNat 0) :: x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: R)

/-- Final memory for bytecode block summary `exporter_block_170`. -/
def exporter_block_170_memory {ee : ExecutionEnv} {mem : ByteArray} {x1 : UInt256} {x2 : UInt256} : ByteArray :=
  ((UInt256.ofNat 0).toByteArray.write 0 (ee.calldata.write x2.toNat (x1.toByteArray.write 0 (((memLoad (UInt256.ofNat 64) mem) + ((UInt256.ofNat 32) + (UInt256.mul (UInt256.div ((UInt256.ofNat 31) + x1) (UInt256.ofNat 32)) (UInt256.ofNat 32)))).toByteArray.write 0 mem (UInt256.ofNat 64).toNat 32) (memLoad (UInt256.ofNat 64) mem).toNat 32) ((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)).toNat x1.toNat) (((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) + x1).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 170. -/
theorem exporter_block_170 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 : UInt256} {R : List UInt256}
    (hstack : R.length + 24 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 808) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 170) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 808) (exporter_block_170_stack (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (R := R)) (exporter_block_170_memory (ee := ee) (mem := mem) (x1 := x1) (x2 := x2)) (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) x1) (((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) + x1) (⟨32⟩ : UInt256)) rdata σ (k + 58) (C + ((173) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) x1) + (3 + 3 * ((x1.toNat + 31) / 32)) + (memExpansionCost (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) x1) (((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) + x1) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.push2 (UInt256.ofNat 240) (by evm_kdecide) (by evm_ov)
  have r4 := r3.chainid (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup10 (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup10 (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup10 (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup10 (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup10 (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup10 (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup1 (by evm_kdecide) (by evm_ov)
  have r12 := r11.dup1 (by evm_kdecide) (by evm_ov)
  have r13 := r12.push1 (UInt256.ofNat 31) (by evm_kdecide) (by evm_ov)
  have r14 := r13.add (by evm_kdecide) (by evm_ov)
  have r15 := r14.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r16 := r15.dup1 (by evm_kdecide) (by evm_ov)
  have r17 := r16.swap2 (by evm_kdecide) (by evm_ov)
  have r18 := r17.div (by evm_kdecide) (by evm_ov)
  have r19 := r18.mul (by evm_kdecide) (by evm_ov)
  have r20 := r19.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r21 := r20.add (by evm_kdecide) (by evm_ov)
  have r22 := r21.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r23 := RD.genMload r22 (by evm_kdecide) (by evm_ov)
  have r24 := r23.swap1 (by evm_kdecide) (by evm_ov)
  have r25 := r24.dup2 (by evm_kdecide) (by evm_ov)
  have r26 := r25.add (by evm_kdecide) (by evm_ov)
  have r27 := r26.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r28 := RD.genMstore r27 (by evm_kdecide) (by evm_ov)
  have r29 := r28.dup1 (by evm_kdecide) (by evm_ov)
  have r30 := r29.swap4 (by evm_kdecide) (by evm_ov)
  have r31 := r30.swap3 (by evm_kdecide) (by evm_ov)
  have r32 := r31.swap2 (by evm_kdecide) (by evm_ov)
  have r33 := r32.swap1 (by evm_kdecide) (by evm_ov)
  have r34 := r33.dup2 (by evm_kdecide) (by evm_ov)
  have r35 := r34.dup2 (by evm_kdecide) (by evm_ov)
  have r36 := RD.genMstore r35 (by evm_kdecide) (by evm_ov)
  have r37 := r36.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r38 := r37.add (by evm_kdecide) (by evm_ov)
  have r39 := r38.dup4 (by evm_kdecide) (by evm_ov)
  have r40 := r39.dup4 (by evm_kdecide) (by evm_ov)
  have r41 := r40.dup1 (by evm_kdecide) (by evm_ov)
  have r42 := r41.dup3 (by evm_kdecide) (by evm_ov)
  have r43 := r42.dup5 (by evm_kdecide) (by evm_ov)
  have r44 := RD.genCalldatacopy r43 (by evm_kdecide) (by evm_ov)
  have r45 := r44.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r46 := r45.swap3 (by evm_kdecide) (by evm_ov)
  have r47 := r46.add (by evm_kdecide) (by evm_ov)
  have r48 := r47.swap2 (by evm_kdecide) (by evm_ov)
  have r49 := r48.swap1 (by evm_kdecide) (by evm_ov)
  have r50 := r49.swap2 (by evm_kdecide) (by evm_ov)
  have r51 := RD.genMstore r50 (by evm_kdecide) (by evm_ov)
  have r52 := r51.pop (by evm_kdecide) (by evm_ov)
  have r53 := r52.push2 (UInt256.ofNat 808) (by evm_kdecide) (by evm_ov)
  have r54 := r53.swap3 (by evm_kdecide) (by evm_ov)
  have r55 := r54.pop (by evm_kdecide) (by evm_ov)
  have r56 := r55.pop (by evm_kdecide) (by evm_ov)
  have r57 := r56.pop (by evm_kdecide) (by evm_ov)
  have r58 := r57.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 808)) r58 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_170_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 : UInt256} {R : List UInt256}
    (hstack : R.length + 24 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 808) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 170) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 808) (exporter_block_170_stack (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (R := R)) (exporter_block_170_memory (ee := ee) (mem := mem) (x1 := x1) (x2 := x2)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_170 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_240`. -/
def exporter_block_240_stack {g : Sat256} {mem : ByteArray} {aw : UInt256} {C : ℕ} {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  (((g.subNat (C + ((84) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256))) + 2)).toUInt256) :: (UInt256.ofNat 376793390874373408599387495934666716005045108771) :: (memLoad (UInt256.ofNat 64) (x0.toByteArray.write 0 ((UInt256.ofNat 80373334883193173493124541549841241460183627345106376917665715749844309508096).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)).toNat 32)) :: (UInt256.sub ((UInt256.ofNat 36) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (UInt256.ofNat 64) (x0.toByteArray.write 0 ((UInt256.ofNat 80373334883193173493124541549841241460183627345106376917665715749844309508096).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)).toNat 32))) :: (memLoad (UInt256.ofNat 64) (x0.toByteArray.write 0 ((UInt256.ofNat 80373334883193173493124541549841241460183627345106376917665715749844309508096).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)).toNat 32)) :: (UInt256.ofNat 32) :: ((UInt256.ofNat 36) + (memLoad (UInt256.ofNat 64) mem)) :: (UInt256.ofNat 2981212681) :: (UInt256.ofNat 376793390874373408599387495934666716005045108771) :: x0 :: R)

/-- Final memory for bytecode block summary `exporter_block_240`. -/
def exporter_block_240_memory {mem : ByteArray} {x0 : UInt256} : ByteArray :=
  (x0.toByteArray.write 0 ((UInt256.ofNat 80373334883193173493124541549841241460183627345106376917665715749844309508096).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 240. -/
theorem exporter_block_240 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 240) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 331) (exporter_block_240_stack (g := g) (mem := mem) (aw := aw) (C := C) (x0 := x0) (R := R)) (exporter_block_240_memory (mem := mem) (x0 := x0)) (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 30) (C + ((86) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r3 := RD.genMload r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.pushConst (UInt256.ofNat 80373334883193173493124541549841241460183627345106376917665715749844309508096) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup2 (by evm_kdecide) (by evm_ov)
  have r6 := RD.genMstore r5 (by evm_kdecide) (by evm_ov)
  have r7 := r6.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup2 (by evm_kdecide) (by evm_ov)
  have r9 := r8.add (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup3 (by evm_kdecide) (by evm_ov)
  have r11 := r10.swap1 (by evm_kdecide) (by evm_ov)
  have r12 := RD.genMstore r11 (by evm_kdecide) (by evm_ov)
  have r13 := r12.swap1 (by evm_kdecide) (by evm_ov)
  have r14 := r13.swap2 (by evm_kdecide) (by evm_ov)
  have r15 := r14.pop (by evm_kdecide) (by evm_ov)
  have r16 := r15.push20 (UInt256.ofNat 376793390874373408599387495934666716005045108771) (by evm_kdecide) (by evm_ov)
  have r17 := r16.swap1 (by evm_kdecide) (by evm_ov)
  have r18 := r17.push4 (UInt256.ofNat 2981212681) (by evm_kdecide) (by evm_ov)
  have r19 := r18.swap1 (by evm_kdecide) (by evm_ov)
  have r20 := r19.push1 (UInt256.ofNat 36) (by evm_kdecide) (by evm_ov)
  have r21 := r20.add (by evm_kdecide) (by evm_ov)
  have r22 := r21.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r23 := r22.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r24 := RD.genMload r23 (by evm_kdecide) (by evm_ov)
  have r25 := r24.dup1 (by evm_kdecide) (by evm_ov)
  have r26 := r25.dup4 (by evm_kdecide) (by evm_ov)
  have r27 := r26.sub (by evm_kdecide) (by evm_ov)
  have r28 := r27.dup2 (by evm_kdecide) (by evm_ov)
  have r29 := r28.dup7 (by evm_kdecide) (by evm_ov)
  have r30 := RD.genGas (RD.normalizeCounters (k' := k + 29) (C' := C + ((84) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) r29 (by omega) (by omega)) (by evm_kdecide) (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 331)) r30 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_240_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 240) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 331) (exporter_block_240_stack (g := g) (mem := mem) (aw := aw) (C := C) (x0 := x0) (R := R)) (exporter_block_240_memory (mem := mem) (x0 := x0)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_240 hstack h)
  exact ⟨_, k', C', h'⟩

/- Unsupported instruction boundary at pc 331: staticcall (0xfa). No RD transition is asserted. Summaries resume at pc 332 from a fresh symbolic RD state. -/

/-- Final stack for bytecode block summary `exporter_block_332_taken`. -/
def exporter_block_332_taken_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 332. -/
theorem exporter_block_332_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 348) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 332) (x0 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 348) (exporter_block_332_taken_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.iszero (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.iszero (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 348) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 348)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_332_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 348) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 332) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 348) (exporter_block_332_taken_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_332_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_332_fallthrough`. -/
def exporter_block_332_fallthrough_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 332. -/
theorem exporter_block_332_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 332) (x0 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 339) (exporter_block_332_fallthrough_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.iszero (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.iszero (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 348) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 339)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_332_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 332) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 339) (exporter_block_332_fallthrough_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_332_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

end exporterBlocks
