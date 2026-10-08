import Reasoning.Reach
import ExporterEvm.Bytecode

open Solm ABI Ethereum Ethereum.EVM
open Reasoning.Theory Reasoning.Reach

namespace exporterBlocks

/-- Automatically generated RD summary for bytecode block at pc 1075. -/
theorem exporter_block_1075_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 x10 x11 x12 : UInt256} {R : List UInt256}
    (hstack : R.length + 17 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt ((x2 + x0) + (UInt256.ofNat 32)) x12)) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1075) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: x10 :: x11 :: x12 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1089) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: x10 :: x11 :: x12 :: R) mem aw rdata σ (k + 11) (C + ((38))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup13 (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup6 (by evm_kdecide) (by evm_ov)
  have r6 := r5.add (by evm_kdecide) (by evm_ov)
  have r7 := r6.add (by evm_kdecide) (by evm_ov)
  have r8 := r7.gt (by evm_kdecide) (by evm_ov)
  have r9 := r8.iszero (by evm_kdecide) (by evm_ov)
  have r10 := r9.push2 (UInt256.ofNat 1093) (by evm_kdecide) (by evm_ov)
  have r11 := r10.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1089)) r11 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_1075_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 x10 x11 x12 : UInt256} {R : List UInt256}
    (hstack : R.length + 17 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt ((x2 + x0) + (UInt256.ofNat 32)) x12)) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1075) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: x10 :: x11 :: x12 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1089) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: x10 :: x11 :: x12 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_1075_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1089. -/
theorem exporter_block_1089 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1089) R mem aw rdata σ k C)
    : RDrev ExporterEvm.exporterRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `exporter_block_1093`. -/
def exporter_block_1093_stack {x0 : UInt256} {x2 : UInt256} {x3 : UInt256} {x6 : UInt256} {x7 : UInt256} {x8 : UInt256} {x9 : UInt256} {x10 : UInt256} {x11 : UInt256} {R : List UInt256} : List UInt256 :=
  ((x11 + (UInt256.ofNat 192)) :: (UInt256.ofNat 1117) :: x3 :: x0 :: (x2 + (UInt256.ofNat 32)) :: x6 :: x7 :: x8 :: x9 :: x10 :: x11 :: R)

/-- Automatically generated RD summary for bytecode block at pc 1093. -/
theorem exporter_block_1093 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 x10 x11 : UInt256} {R : List UInt256}
    (hstack : R.length + 14 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 912) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1093) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: x10 :: x11 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 912) (exporter_block_1093_stack (x0 := x0) (x2 := x2) (x3 := x3) (x6 := x6) (x7 := x7) (x8 := x8) (x9 := x9) (x10 := x10) (x11 := x11) (R := R)) mem aw rdata σ (k + 18) (C + ((52))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup4 (by evm_kdecide) (by evm_ov)
  have r4 := r3.add (by evm_kdecide) (by evm_ov)
  have r5 := r4.swap6 (by evm_kdecide) (by evm_ov)
  have r6 := r5.pop (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup1 (by evm_kdecide) (by evm_ov)
  have r8 := r7.swap5 (by evm_kdecide) (by evm_ov)
  have r9 := r8.pop (by evm_kdecide) (by evm_ov)
  have r10 := r9.pop (by evm_kdecide) (by evm_ov)
  have r11 := r10.pop (by evm_kdecide) (by evm_ov)
  have r12 := r11.pop (by evm_kdecide) (by evm_ov)
  have r13 := r12.push2 (UInt256.ofNat 1117) (by evm_kdecide) (by evm_ov)
  have r14 := r13.push1 (UInt256.ofNat 192) (by evm_kdecide) (by evm_ov)
  have r15 := r14.dup11 (by evm_kdecide) (by evm_ov)
  have r16 := r15.add (by evm_kdecide) (by evm_ov)
  have r17 := r16.push2 (UInt256.ofNat 912) (by evm_kdecide) (by evm_ov)
  have r18 := r17.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 912)) r18 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_1093_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 x10 x11 : UInt256} {R : List UInt256}
    (hstack : R.length + 14 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 912) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1093) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: x10 :: x11 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 912) (exporter_block_1093_stack (x0 := x0) (x2 := x2) (x3 := x3) (x6 := x6) (x7 := x7) (x8 := x8) (x9 := x9) (x10 := x10) (x11 := x11) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_1093 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_1117`. -/
def exporter_block_1117_stack {x0 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {x6 : UInt256} {x7 : UInt256} {x8 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R)

/-- Automatically generated RD summary for bytecode block at pc 1117. -/
theorem exporter_block_1117 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 x10 x11 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains x11 = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1117) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: x10 :: x11 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 x11 (exporter_block_1117_stack (x0 := x0) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (x7 := x7) (x8 := x8) (R := R)) mem aw rdata σ (k + 15) (C + ((45))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pop (by evm_kdecide) (by evm_ov)
  have r4 := r3.swap3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.swap6 (by evm_kdecide) (by evm_ov)
  have r6 := r5.swap9 (by evm_kdecide) (by evm_ov)
  have r7 := r6.pop (by evm_kdecide) (by evm_ov)
  have r8 := r7.swap3 (by evm_kdecide) (by evm_ov)
  have r9 := r8.swap6 (by evm_kdecide) (by evm_ov)
  have r10 := r9.swap9 (by evm_kdecide) (by evm_ov)
  have r11 := r10.swap1 (by evm_kdecide) (by evm_ov)
  have r12 := r11.swap4 (by evm_kdecide) (by evm_ov)
  have r13 := r12.swap7 (by evm_kdecide) (by evm_ov)
  have r14 := r13.pop (by evm_kdecide) (by evm_ov)
  have r15 := r14.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r15 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_1117_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 x10 x11 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains x11 = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1117) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: x10 :: x11 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 x11 (exporter_block_1117_stack (x0 := x0) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (x7 := x7) (x8 := x8) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_1117 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_1132`. -/
def exporter_block_1132_stack {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 0) :: (memLoad x0 mem) :: (UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Final memory for bytecode block summary `exporter_block_1132`. -/
def exporter_block_1132_memory {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} : ByteArray :=
  ((memLoad x0 mem).toByteArray.write 0 mem x1.toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1132. -/
theorem exporter_block_1132 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1132) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1142) (exporter_block_1132_stack (mem := mem) (x0 := x0) (x1 := x1) (R := R)) (exporter_block_1132_memory (mem := mem) (x0 := x0) (x1 := x1)) (M (M aw x0 (⟨32⟩ : UInt256)) x1 (⟨32⟩ : UInt256)) rdata σ (k + 8) (C + ((22) + (memExpansionCost aw x0 (⟨32⟩ : UInt256)) + (memExpansionCost (M aw x0 (⟨32⟩ : UInt256)) x1 (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup2 (by evm_kdecide) (by evm_ov)
  have r4 := RD.genMload r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup5 (by evm_kdecide) (by evm_ov)
  have r7 := RD.genMstore r6 (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1142)) r8 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_1132_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1132) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1142) (exporter_block_1132_stack (mem := mem) (x0 := x0) (x1 := x1) (R := R)) (exporter_block_1132_memory (mem := mem) (x0 := x0) (x1 := x1)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_1132 hstack h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1142. -/
theorem exporter_block_1142_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.lt x0 x1)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1170) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1142) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1170) (x0 :: x1 :: R) mem aw rdata σ (k + 7) (C + ((26))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup2 (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.lt (by evm_kdecide) (by evm_ov)
  have r5 := r4.iszero (by evm_kdecide) (by evm_ov)
  have r6 := r5.push2 (UInt256.ofNat 1170) (by evm_kdecide) (by evm_ov)
  have r7 := r6.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1170)) r7 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_1142_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.lt x0 x1)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1170) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1142) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1170) (x0 :: x1 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_1142_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1142. -/
theorem exporter_block_1142_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.lt x0 x1)) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1142) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1151) (x0 :: x1 :: R) mem aw rdata σ (k + 7) (C + ((26))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup2 (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.lt (by evm_kdecide) (by evm_ov)
  have r5 := r4.iszero (by evm_kdecide) (by evm_ov)
  have r6 := r5.push2 (UInt256.ofNat 1170) (by evm_kdecide) (by evm_ov)
  have r7 := r6.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1151)) r7 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_1142_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.lt x0 x1)) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1142) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1151) (x0 :: x1 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_1142_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_1151`. -/
def exporter_block_1151_stack {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {R : List UInt256} : List UInt256 :=
  (((UInt256.ofNat 32) + x0) :: x1 :: x2 :: x3 :: x4 :: R)

/-- Final memory for bytecode block summary `exporter_block_1151`. -/
def exporter_block_1151_memory {mem : ByteArray} {x0 : UInt256} {x3 : UInt256} {x4 : UInt256} : ByteArray :=
  ((memLoad ((UInt256.ofNat 32) + (x3 + x0)) mem).toByteArray.write 0 mem ((UInt256.ofNat 32) + (x0 + x4)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1151. -/
theorem exporter_block_1151 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1142) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1151) (x0 :: x1 :: x2 :: x3 :: x4 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1142) (exporter_block_1151_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (R := R)) (exporter_block_1151_memory (mem := mem) (x0 := x0) (x3 := x3) (x4 := x4)) (M (M aw ((UInt256.ofNat 32) + (x3 + x0)) (⟨32⟩ : UInt256)) ((UInt256.ofNat 32) + (x0 + x4)) (⟨32⟩ : UInt256)) rdata σ (k + 16) (C + ((53) + (memExpansionCost aw ((UInt256.ofNat 32) + (x3 + x0)) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw ((UInt256.ofNat 32) + (x3 + x0)) (⟨32⟩ : UInt256)) ((UInt256.ofNat 32) + (x0 + x4)) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup2 (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup6 (by evm_kdecide) (by evm_ov)
  have r4 := r3.add (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup2 (by evm_kdecide) (by evm_ov)
  have r6 := r5.add (by evm_kdecide) (by evm_ov)
  have r7 := RD.genMload r6 (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup7 (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup4 (by evm_kdecide) (by evm_ov)
  have r10 := r9.add (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup3 (by evm_kdecide) (by evm_ov)
  have r12 := r11.add (by evm_kdecide) (by evm_ov)
  have r13 := RD.genMstore r12 (by evm_kdecide) (by evm_ov)
  have r14 := r13.add (by evm_kdecide) (by evm_ov)
  have r15 := r14.push2 (UInt256.ofNat 1142) (by evm_kdecide) (by evm_ov)
  have r16 := r15.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1142)) r16 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_1151_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1142) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1151) (x0 :: x1 :: x2 :: x3 :: x4 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1142) (exporter_block_1151_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (R := R)) (exporter_block_1151_memory (mem := mem) (x0 := x0) (x3 := x3) (x4 := x4)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_1151 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1170. -/
theorem exporter_block_1170_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt x0 x1)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1188) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1170) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1188) (x0 :: x1 :: R) mem aw rdata σ (k + 7) (C + ((26))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup2 (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.gt (by evm_kdecide) (by evm_ov)
  have r5 := r4.iszero (by evm_kdecide) (by evm_ov)
  have r6 := r5.push2 (UInt256.ofNat 1188) (by evm_kdecide) (by evm_ov)
  have r7 := r6.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1188)) r7 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_1170_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt x0 x1)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1188) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1170) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1188) (x0 :: x1 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_1170_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1170. -/
theorem exporter_block_1170_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt x0 x1)) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1170) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1179) (x0 :: x1 :: R) mem aw rdata σ (k + 7) (C + ((26))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup2 (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.gt (by evm_kdecide) (by evm_ov)
  have r5 := r4.iszero (by evm_kdecide) (by evm_ov)
  have r6 := r5.push2 (UInt256.ofNat 1188) (by evm_kdecide) (by evm_ov)
  have r7 := r6.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1179)) r7 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_1170_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt x0 x1)) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1170) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1179) (x0 :: x1 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_1170_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Final memory for bytecode block summary `exporter_block_1179`. -/
def exporter_block_1179_memory {mem : ByteArray} {x1 : UInt256} {x4 : UInt256} : ByteArray :=
  ((UInt256.ofNat 0).toByteArray.write 0 mem ((x4 + x1) + (UInt256.ofNat 32)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1179. -/
theorem exporter_block_1179 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1179) (x0 :: x1 :: x2 :: x3 :: x4 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1188) (x0 :: x1 :: x2 :: x3 :: x4 :: R) (exporter_block_1179_memory (mem := mem) (x1 := x1) (x4 := x4)) (M aw ((x4 + x1) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) rdata σ (k + 7) (C + ((21) + (memExpansionCost aw ((x4 + x1) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup4 (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup8 (by evm_kdecide) (by evm_ov)
  have r5 := r4.add (by evm_kdecide) (by evm_ov)
  have r6 := r5.add (by evm_kdecide) (by evm_ov)
  have r7 := RD.genMstore r6 (by evm_kdecide) (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1188)) r7 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_1179_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1179) (x0 :: x1 :: x2 :: x3 :: x4 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1188) (x0 :: x1 :: x2 :: x3 :: x4 :: R) (exporter_block_1179_memory (mem := mem) (x1 := x1) (x4 := x4)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_1179 hstack h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_1188`. -/
def exporter_block_1188_stack {x1 : UInt256} {x4 : UInt256} {R : List UInt256} : List UInt256 :=
  (((UInt256.ofNat 32) + ((UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) ((UInt256.ofNat 31) + x1)) + x4)) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1188. -/
theorem exporter_block_1188 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains x5 = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1188) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 x5 (exporter_block_1188_stack (x1 := x1) (x4 := x4) (R := R)) mem aw rdata σ (k + 17) (C + ((51))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 31) (by evm_kdecide) (by evm_ov)
  have r4 := r3.add (by evm_kdecide) (by evm_ov)
  have r5 := r4.pushConst (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r6 := r5.and (by evm_kdecide) (by evm_ov)
  have r7 := r6.swap3 (by evm_kdecide) (by evm_ov)
  have r8 := r7.swap1 (by evm_kdecide) (by evm_ov)
  have r9 := r8.swap3 (by evm_kdecide) (by evm_ov)
  have r10 := r9.add (by evm_kdecide) (by evm_ov)
  have r11 := r10.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r12 := r11.add (by evm_kdecide) (by evm_ov)
  have r13 := r12.swap3 (by evm_kdecide) (by evm_ov)
  have r14 := r13.swap2 (by evm_kdecide) (by evm_ov)
  have r15 := r14.pop (by evm_kdecide) (by evm_ov)
  have r16 := r15.pop (by evm_kdecide) (by evm_ov)
  have r17 := r16.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r17 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_1188_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains x5 = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1188) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 x5 (exporter_block_1188_stack (x1 := x1) (x4 := x4) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_1188 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_1239`. -/
def exporter_block_1239_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  (x1 :: (x0 + (UInt256.ofNat 32)) :: (UInt256.ofNat 1258) :: (UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Final memory for bytecode block summary `exporter_block_1239`. -/
def exporter_block_1239_memory {mem : ByteArray} {x0 : UInt256} : ByteArray :=
  ((UInt256.ofNat 32).toByteArray.write 0 mem x0.toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1239. -/
theorem exporter_block_1239 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1132) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1239) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1132) (exporter_block_1239_stack (x0 := x0) (x1 := x1) (R := R)) (exporter_block_1239_memory (mem := mem) (x0 := x0)) (M aw x0 (⟨32⟩ : UInt256)) rdata σ (k + 12) (C + ((39) + (memExpansionCost aw x0 (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup2 (by evm_kdecide) (by evm_ov)
  have r4 := RD.genMstore r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r6 := r5.push2 (UInt256.ofNat 1258) (by evm_kdecide) (by evm_ov)
  have r7 := r6.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup4 (by evm_kdecide) (by evm_ov)
  have r9 := r8.add (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup5 (by evm_kdecide) (by evm_ov)
  have r11 := r10.push2 (UInt256.ofNat 1132) (by evm_kdecide) (by evm_ov)
  have r12 := r11.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1132)) r12 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_1239_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1132) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1239) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1132) (exporter_block_1239_stack (x0 := x0) (x1 := x1) (R := R)) (exporter_block_1239_memory (mem := mem) (x0 := x0)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_1239 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_1258`. -/
def exporter_block_1258_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: R)

/-- Automatically generated RD summary for bytecode block at pc 1258. -/
theorem exporter_block_1258 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains x4 = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1258) (x0 :: x1 :: x2 :: x3 :: x4 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 x4 (exporter_block_1258_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 7) (C + ((21))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap4 (by evm_kdecide) (by evm_ov)
  have r3 := r2.swap3 (by evm_kdecide) (by evm_ov)
  have r4 := r3.pop (by evm_kdecide) (by evm_ov)
  have r5 := r4.pop (by evm_kdecide) (by evm_ov)
  have r6 := r5.pop (by evm_kdecide) (by evm_ov)
  have r7 := r6.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r7 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_1258_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains x4 = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1258) (x0 :: x1 :: x2 :: x3 :: x4 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 x4 (exporter_block_1258_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_1258 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_1265_taken`. -/
def exporter_block_1265_taken_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 1265. -/
theorem exporter_block_1265_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 32))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1283) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1265) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1283) (exporter_block_1265_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 10) (C + ((35))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup5 (by evm_kdecide) (by evm_ov)
  have r6 := r5.sub (by evm_kdecide) (by evm_ov)
  have r7 := r6.slt (by evm_kdecide) (by evm_ov)
  have r8 := r7.iszero (by evm_kdecide) (by evm_ov)
  have r9 := r8.push2 (UInt256.ofNat 1283) (by evm_kdecide) (by evm_ov)
  have r10 := r9.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1283)) r10 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_1265_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 32))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1283) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1265) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1283) (exporter_block_1265_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_1265_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_1265_fallthrough`. -/
def exporter_block_1265_fallthrough_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 1265. -/
theorem exporter_block_1265_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 32))) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1265) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1279) (exporter_block_1265_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 10) (C + ((35))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup5 (by evm_kdecide) (by evm_ov)
  have r6 := r5.sub (by evm_kdecide) (by evm_ov)
  have r7 := r6.slt (by evm_kdecide) (by evm_ov)
  have r8 := r7.iszero (by evm_kdecide) (by evm_ov)
  have r9 := r8.push2 (UInt256.ofNat 1283) (by evm_kdecide) (by evm_ov)
  have r10 := r9.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1279)) r10 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_1265_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 32))) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1265) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1279) (exporter_block_1265_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_1265_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1279. -/
theorem exporter_block_1279 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1279) R mem aw rdata σ k C)
    : RDrev ExporterEvm.exporterRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `exporter_block_1283_taken`. -/
def exporter_block_1283_taken_stack {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((memLoad x1 mem) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 1283. -/
theorem exporter_block_1283_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.eq (memLoad x1 mem) (UInt256.isZero (UInt256.isZero (memLoad x1 mem)))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1258) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1283) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1258) (exporter_block_1283_taken_stack (mem := mem) (x0 := x0) (x1 := x1) (R := R)) mem (M aw x1 (⟨32⟩ : UInt256)) rdata σ (k + 10) (C + ((35) + (memExpansionCost aw x1 (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup2 (by evm_kdecide) (by evm_ov)
  have r3 := RD.genMload r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup1 (by evm_kdecide) (by evm_ov)
  have r5 := r4.iszero (by evm_kdecide) (by evm_ov)
  have r6 := r5.iszero (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup2 (by evm_kdecide) (by evm_ov)
  have r8 := r7.eq (by evm_kdecide) (by evm_ov)
  have r9 := r8.push2 (UInt256.ofNat 1258) (by evm_kdecide) (by evm_ov)
  have r10 := r9.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1258)) r10 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_1283_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.eq (memLoad x1 mem) (UInt256.isZero (UInt256.isZero (memLoad x1 mem)))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1258) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1283) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1258) (exporter_block_1283_taken_stack (mem := mem) (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_1283_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_1283_fallthrough`. -/
def exporter_block_1283_fallthrough_stack {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((memLoad x1 mem) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 1283. -/
theorem exporter_block_1283_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.eq (memLoad x1 mem) (UInt256.isZero (UInt256.isZero (memLoad x1 mem)))) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1283) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1295) (exporter_block_1283_fallthrough_stack (mem := mem) (x0 := x0) (x1 := x1) (R := R)) mem (M aw x1 (⟨32⟩ : UInt256)) rdata σ (k + 10) (C + ((35) + (memExpansionCost aw x1 (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup2 (by evm_kdecide) (by evm_ov)
  have r3 := RD.genMload r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup1 (by evm_kdecide) (by evm_ov)
  have r5 := r4.iszero (by evm_kdecide) (by evm_ov)
  have r6 := r5.iszero (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup2 (by evm_kdecide) (by evm_ov)
  have r8 := r7.eq (by evm_kdecide) (by evm_ov)
  have r9 := r8.push2 (UInt256.ofNat 1258) (by evm_kdecide) (by evm_ov)
  have r10 := r9.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1295)) r10 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_1283_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.eq (memLoad x1 mem) (UInt256.isZero (UInt256.isZero (memLoad x1 mem)))) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1283) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1295) (exporter_block_1283_fallthrough_stack (mem := mem) (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_1283_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1295. -/
theorem exporter_block_1295 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1295) R mem aw rdata σ k C)
    : RDrev ExporterEvm.exporterRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

end exporterBlocks
