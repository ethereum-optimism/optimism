import Reasoning.Reach
import ExporterEvm.Bytecode

open Solm ABI Ethereum Ethereum.EVM
open Reasoning.Theory Reasoning.Reach

namespace exporterBlocks

/-- Automatically generated RD summary for bytecode block at pc 339. -/
theorem exporter_block_339 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 339) R mem aw rdata σ k C)
    : RDrev ExporterEvm.exporterRuntime g s0 := by
  let r0 := h
  have r1 := r0.returndatasize (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := RD.genReturndatacopy r3 (by evm_kdecide) (by
    have hz : UInt256.ofNat 0 = (⟨0⟩ : UInt256) := by decide
    simpa only [hz] using returnDataCopyFullGuard rdata) (by evm_ov)
  have r5 := r4.returndatasize (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  exact RD.genRev r6 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `exporter_block_348`. -/
def exporter_block_348_stack {mem : ByteArray} {rdata : ByteArray} {R : List UInt256} : List UInt256 :=
  ((memLoad (UInt256.ofNat 64) mem) :: ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat rdata.size)) :: (UInt256.ofNat 384) :: R)

/-- Final memory for bytecode block summary `exporter_block_348`. -/
def exporter_block_348_memory {mem : ByteArray} {rdata : ByteArray} : ByteArray :=
  (((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat rdata.size) + (UInt256.ofNat 31)) (UInt256.lnot (UInt256.ofNat 31)))).toByteArray.write 0 mem (UInt256.ofNat 64).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 348. -/
theorem exporter_block_348 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1265) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 348) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1265) (exporter_block_348_stack (mem := mem) (rdata := rdata) (R := R)) (exporter_block_348_memory (mem := mem) (rdata := rdata)) (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 28) (C + ((81) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := r2.pop (by evm_kdecide) (by evm_ov)
  have r4 := r3.pop (by evm_kdecide) (by evm_ov)
  have r5 := r4.pop (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r7 := RD.genMload r6 (by evm_kdecide) (by evm_ov)
  have r8 := r7.returndatasize (by evm_kdecide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 31) (by evm_kdecide) (by evm_ov)
  have r10 := r9.not (by evm_kdecide) (by evm_ov)
  have r11 := r10.push1 (UInt256.ofNat 31) (by evm_kdecide) (by evm_ov)
  have r12 := r11.dup3 (by evm_kdecide) (by evm_ov)
  have r13 := r12.add (by evm_kdecide) (by evm_ov)
  have r14 := r13.and (by evm_kdecide) (by evm_ov)
  have r15 := r14.dup3 (by evm_kdecide) (by evm_ov)
  have r16 := r15.add (by evm_kdecide) (by evm_ov)
  have r17 := r16.dup1 (by evm_kdecide) (by evm_ov)
  have r18 := r17.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r19 := RD.genMstore r18 (by evm_kdecide) (by evm_ov)
  have r20 := r19.pop (by evm_kdecide) (by evm_ov)
  have r21 := r20.dup2 (by evm_kdecide) (by evm_ov)
  have r22 := r21.add (by evm_kdecide) (by evm_ov)
  have r23 := r22.swap1 (by evm_kdecide) (by evm_ov)
  have r24 := r23.push2 (UInt256.ofNat 384) (by evm_kdecide) (by evm_ov)
  have r25 := r24.swap2 (by evm_kdecide) (by evm_ov)
  have r26 := r25.swap1 (by evm_kdecide) (by evm_ov)
  have r27 := r26.push2 (UInt256.ofNat 1265) (by evm_kdecide) (by evm_ov)
  have r28 := r27.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1265)) r28 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_348_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1265) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 348) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1265) (exporter_block_348_stack (mem := mem) (rdata := rdata) (R := R)) (exporter_block_348_memory (mem := mem) (rdata := rdata)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_348 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_384_taken`. -/
def exporter_block_384_taken_stack {R : List UInt256} : List UInt256 :=
  R

/-- Automatically generated RD summary for bytecode block at pc 384. -/
theorem exporter_block_384_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (UInt256.isZero x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 439) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 384) (x0 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 439) (exporter_block_384_taken_stack (R := R)) mem aw rdata σ (k + 4) (C + ((17))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.iszero (by evm_kdecide) (by evm_ov)
  have r3 := r2.push2 (UInt256.ofNat 439) (by evm_kdecide) (by evm_ov)
  have r4 := r3.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 439)) r4 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_384_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (UInt256.isZero x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 439) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 384) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 439) (exporter_block_384_taken_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_384_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_384_fallthrough`. -/
def exporter_block_384_fallthrough_stack {R : List UInt256} : List UInt256 :=
  R

/-- Automatically generated RD summary for bytecode block at pc 384. -/
theorem exporter_block_384_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (UInt256.isZero x0) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 384) (x0 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 390) (exporter_block_384_fallthrough_stack (R := R)) mem aw rdata σ (k + 4) (C + ((17))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.iszero (by evm_kdecide) (by evm_ov)
  have r3 := r2.push2 (UInt256.ofNat 439) (by evm_kdecide) (by evm_ov)
  have r4 := r3.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 390)) r4 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_384_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (UInt256.isZero x0) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 384) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 390) (exporter_block_384_fallthrough_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_384_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 390. -/
theorem exporter_block_390 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 390) R mem aw rdata σ k C)
    : RDrev ExporterEvm.exporterRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r2 := RD.genMload r1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pushConst (UInt256.ofNat 92618038157931011697939822198512234313800822569940725936850136633179623129088) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup2 (by evm_kdecide) (by evm_ov)
  have r5 := RD.genMstore r4 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r7 := r6.add (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r9 := RD.genMload r8 (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup1 (by evm_kdecide) (by evm_ov)
  have r11 := r10.swap2 (by evm_kdecide) (by evm_ov)
  have r12 := r11.sub (by evm_kdecide) (by evm_ov)
  have r13 := r12.swap1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r13 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `exporter_block_439`. -/
def exporter_block_439_stack {ee : ExecutionEnv} {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {x6 : UInt256} {x7 : UInt256} {x8 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 1299) :: ((UInt256.ofNat 4) + (memLoad (UInt256.ofNat 64) ((UInt256.lor (UInt256.ofNat 24938299286184694734990173794665248623958599392764557417163592631308932612096) (UInt256.land (UInt256.ofNat 26959946667150639794667015087019630673637144422540572481103610249215) (memLoad ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)) + (UInt256.ofNat 32)) (((UInt256.ofNat 100) + (memLoad (UInt256.ofNat 64) mem)).toByteArray.write 0 (((UInt256.sub ((UInt256.ofNat 100) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32))) + (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904)).toByteArray.write 0 ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)).toNat 32) (UInt256.ofNat 64).toNat 32)))).toByteArray.write 0 (((UInt256.ofNat 100) + (memLoad (UInt256.ofNat 64) mem)).toByteArray.write 0 (((UInt256.sub ((UInt256.ofNat 100) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32))) + (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904)).toByteArray.write 0 ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)).toNat 32) (UInt256.ofNat 64).toNat 32) ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)) + (UInt256.ofNat 32)).toNat 32))) :: x1 :: (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)) :: x8 :: (UInt256.ofNat 664) :: (UInt256.ofNat 1035673643) :: (UInt256.ofNat 376793390874373408599387495934666716005045108743) :: x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R)

/-- Final memory for bytecode block summary `exporter_block_439`. -/
def exporter_block_439_memory {ee : ExecutionEnv} {mem : ByteArray} {x0 : UInt256} : ByteArray :=
  ((UInt256.land (UInt256.shiftLeft (UInt256.ofNat 1035673643) (UInt256.ofNat 224)) (UInt256.ofNat 115792089210356248756420345214020892766250353992003419616917011526809519390720)).toByteArray.write 0 ((UInt256.lor (UInt256.ofNat 24938299286184694734990173794665248623958599392764557417163592631308932612096) (UInt256.land (UInt256.ofNat 26959946667150639794667015087019630673637144422540572481103610249215) (memLoad ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)) + (UInt256.ofNat 32)) (((UInt256.ofNat 100) + (memLoad (UInt256.ofNat 64) mem)).toByteArray.write 0 (((UInt256.sub ((UInt256.ofNat 100) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32))) + (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904)).toByteArray.write 0 ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)).toNat 32) (UInt256.ofNat 64).toNat 32)))).toByteArray.write 0 (((UInt256.ofNat 100) + (memLoad (UInt256.ofNat 64) mem)).toByteArray.write 0 (((UInt256.sub ((UInt256.ofNat 100) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32))) + (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904)).toByteArray.write 0 ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)).toNat 32) (UInt256.ofNat 64).toNat 32) ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)) + (UInt256.ofNat 32)).toNat 32) (memLoad (UInt256.ofNat 64) ((UInt256.lor (UInt256.ofNat 24938299286184694734990173794665248623958599392764557417163592631308932612096) (UInt256.land (UInt256.ofNat 26959946667150639794667015087019630673637144422540572481103610249215) (memLoad ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)) + (UInt256.ofNat 32)) (((UInt256.ofNat 100) + (memLoad (UInt256.ofNat 64) mem)).toByteArray.write 0 (((UInt256.sub ((UInt256.ofNat 100) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32))) + (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904)).toByteArray.write 0 ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)).toNat 32) (UInt256.ofNat 64).toNat 32)))).toByteArray.write 0 (((UInt256.ofNat 100) + (memLoad (UInt256.ofNat 64) mem)).toByteArray.write 0 (((UInt256.sub ((UInt256.ofNat 100) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32))) + (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904)).toByteArray.write 0 ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)).toNat 32) (UInt256.ofNat 64).toNat 32) ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)) + (UInt256.ofNat 32)).toNat 32)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 439. -/
theorem exporter_block_439 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 18 ≤ 1024)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 439) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 663) (exporter_block_439_stack (ee := ee) (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (x7 := x7) (x8 := x8) (R := R)) (exporter_block_439_memory (ee := ee) (mem := mem) (x0 := x0)) (M (M (M (M (M (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) ((UInt256.lor (UInt256.ofNat 24938299286184694734990173794665248623958599392764557417163592631308932612096) (UInt256.land (UInt256.ofNat 26959946667150639794667015087019630673637144422540572481103610249215) (memLoad ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)) + (UInt256.ofNat 32)) (((UInt256.ofNat 100) + (memLoad (UInt256.ofNat 64) mem)).toByteArray.write 0 (((UInt256.sub ((UInt256.ofNat 100) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32))) + (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904)).toByteArray.write 0 ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)).toNat 32) (UInt256.ofNat 64).toNat 32)))).toByteArray.write 0 (((UInt256.ofNat 100) + (memLoad (UInt256.ofNat 64) mem)).toByteArray.write 0 (((UInt256.sub ((UInt256.ofNat 100) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32))) + (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904)).toByteArray.write 0 ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)).toNat 32) (UInt256.ofNat 64).toNat 32) ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)) + (UInt256.ofNat 32)).toNat 32)) (⟨32⟩ : UInt256)) rdata σ (k + 64) (C + ((189) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) ((UInt256.lor (UInt256.ofNat 24938299286184694734990173794665248623958599392764557417163592631308932612096) (UInt256.land (UInt256.ofNat 26959946667150639794667015087019630673637144422540572481103610249215) (memLoad ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)) + (UInt256.ofNat 32)) (((UInt256.ofNat 100) + (memLoad (UInt256.ofNat 64) mem)).toByteArray.write 0 (((UInt256.sub ((UInt256.ofNat 100) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32))) + (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904)).toByteArray.write 0 ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)).toNat 32) (UInt256.ofNat 64).toNat 32)))).toByteArray.write 0 (((UInt256.ofNat 100) + (memLoad (UInt256.ofNat 64) mem)).toByteArray.write 0 (((UInt256.sub ((UInt256.ofNat 100) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32))) + (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904)).toByteArray.write 0 ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)).toNat 32) (UInt256.ofNat 64).toNat 32) ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 (x0.toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32)) + (UInt256.ofNat 32)).toNat 32)) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r3 := RD.genMload r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 36) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup2 (by evm_kdecide) (by evm_ov)
  have r6 := r5.add (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup3 (by evm_kdecide) (by evm_ov)
  have r8 := r7.swap1 (by evm_kdecide) (by evm_ov)
  have r9 := RD.genMstore r8 (by evm_kdecide) (by evm_ov)
  have r10 := r9.timestamp (by evm_kdecide) (by evm_ov)
  have r11 := r10.push1 (UInt256.ofNat 68) (by evm_kdecide) (by evm_ov)
  have r12 := r11.dup3 (by evm_kdecide) (by evm_ov)
  have r13 := r12.add (by evm_kdecide) (by evm_ov)
  have r14 := RD.genMstore r13 (by evm_kdecide) (by evm_ov)
  have r15 := r14.push20 (UInt256.ofNat 376793390874373408599387495934666716005045108743) (by evm_kdecide) (by evm_ov)
  have r16 := r15.swap1 (by evm_kdecide) (by evm_ov)
  have r17 := r16.push4 (UInt256.ofNat 1035673643) (by evm_kdecide) (by evm_ov)
  have r18 := r17.swap1 (by evm_kdecide) (by evm_ov)
  have r19 := r18.dup12 (by evm_kdecide) (by evm_ov)
  have r20 := r19.swap1 (by evm_kdecide) (by evm_ov)
  have r21 := r20.push1 (UInt256.ofNat 100) (by evm_kdecide) (by evm_ov)
  have r22 := r21.add (by evm_kdecide) (by evm_ov)
  have r23 := r22.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r24 := r23.dup1 (by evm_kdecide) (by evm_ov)
  have r25 := RD.genMload r24 (by evm_kdecide) (by evm_ov)
  have r26 := r25.pushConst (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r27 := r26.dup2 (by evm_kdecide) (by evm_ov)
  have r28 := r27.dup5 (by evm_kdecide) (by evm_ov)
  have r29 := r28.sub (by evm_kdecide) (by evm_ov)
  have r30 := r29.add (by evm_kdecide) (by evm_ov)
  have r31 := r30.dup2 (by evm_kdecide) (by evm_ov)
  have r32 := RD.genMstore r31 (by evm_kdecide) (by evm_ov)
  have r33 := r32.swap2 (by evm_kdecide) (by evm_ov)
  have r34 := r33.dup2 (by evm_kdecide) (by evm_ov)
  have r35 := RD.genMstore r34 (by evm_kdecide) (by evm_ov)
  have r36 := r35.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r37 := r36.dup3 (by evm_kdecide) (by evm_ov)
  have r38 := r37.add (by evm_kdecide) (by evm_ov)
  have r39 := r38.dup1 (by evm_kdecide) (by evm_ov)
  have r40 := RD.genMload r39 (by evm_kdecide) (by evm_ov)
  have r41 := r40.pushConst (UInt256.ofNat 26959946667150639794667015087019630673637144422540572481103610249215) (width := 28) (op := .PUSH28) (by decide) (by evm_kdecide) (by evm_ov)
  have r42 := r41.and (by evm_kdecide) (by evm_ov)
  have r43 := r42.pushConst (UInt256.ofNat 24938299286184694734990173794665248623958599392764557417163592631308932612096) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r44 := r43.or (by evm_kdecide) (by evm_ov)
  have r45 := r44.swap1 (by evm_kdecide) (by evm_ov)
  have r46 := RD.genMstore r45 (by evm_kdecide) (by evm_ov)
  have r47 := RD.genMload r46 (by evm_kdecide) (by evm_ov)
  have r48 := r47.pushConst (UInt256.ofNat 115792089210356248756420345214020892766250353992003419616917011526809519390720) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r49 := r48.push1 (UInt256.ofNat 224) (by evm_kdecide) (by evm_ov)
  have r50 := r49.dup6 (by evm_kdecide) (by evm_ov)
  have r51 := r50.swap1 (by evm_kdecide) (by evm_ov)
  have r52 := r51.shl (by evm_kdecide) (by evm_ov)
  have r53 := r52.and (by evm_kdecide) (by evm_ov)
  have r54 := r53.dup2 (by evm_kdecide) (by evm_ov)
  have r55 := RD.genMstore r54 (by evm_kdecide) (by evm_ov)
  have r56 := r55.push2 (UInt256.ofNat 664) (by evm_kdecide) (by evm_ov)
  have r57 := r56.swap3 (by evm_kdecide) (by evm_ov)
  have r58 := r57.swap2 (by evm_kdecide) (by evm_ov)
  have r59 := r58.swap1 (by evm_kdecide) (by evm_ov)
  have r60 := r59.dup8 (by evm_kdecide) (by evm_ov)
  have r61 := r60.swap1 (by evm_kdecide) (by evm_ov)
  have r62 := r61.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r63 := r62.add (by evm_kdecide) (by evm_ov)
  have r64 := r63.push2 (UInt256.ofNat 1299) (by evm_kdecide) (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 663)) r64 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_439_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 18 ≤ 1024)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 439) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 663) (exporter_block_439_stack (ee := ee) (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (x7 := x7) (x8 := x8) (R := R)) (exporter_block_439_memory (ee := ee) (mem := mem) (x0 := x0)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_439 hstack h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_663`. -/
def exporter_block_663_stack {R : List UInt256} : List UInt256 :=
  R

/-- Automatically generated RD summary for bytecode block at pc 663. -/
theorem exporter_block_663 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 1 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains x0 = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 663) (x0 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 x0 (exporter_block_663_stack (R := R)) mem aw rdata σ (k + 1) (C + ((8))) := by
  let r0 := h
  have r1 := r0.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r1 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_663_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 1 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains x0 = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 663) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 x0 (exporter_block_663_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_663 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_664_taken`. -/
def exporter_block_664_taken_stack {mem : ByteArray} {σ : AccountMap} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero (extCodeSizeWord σ x2)) :: x2 :: (UInt256.ofNat 0) :: (memLoad (UInt256.ofNat 64) mem) :: (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem)) :: (memLoad (UInt256.ofNat 64) mem) :: (UInt256.ofNat 0) :: x0 :: x1 :: x2 :: R)

/-- Automatically generated RD summary for bytecode block at pc 664. -/
theorem exporter_block_664_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero (extCodeSizeWord σ x2))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 690) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 664) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 690) (exporter_block_664_taken_stack (mem := mem) (σ := σ) (x0 := x0) (x1 := x1) (x2 := x2) (R := R)) mem (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r4 := RD.genMload r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup4 (by evm_kdecide) (by evm_ov)
  have r7 := r6.sub (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup2 (by evm_kdecide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup8 (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r12⟩ := r11.extcodesize (by evm_kdecide) (by evm_ov)
  have r13 := r12.iszero (by evm_kdecide) (by evm_ov)
  have r14 := r13.dup1 (by evm_kdecide) (by evm_ov)
  have r15 := r14.iszero (by evm_kdecide) (by evm_ov)
  have r16 := r15.push2 (UInt256.ofNat 690) (by evm_kdecide) (by evm_ov)
  have r17 := r16.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 690)) r17 (by evm_kdecide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_664_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero (extCodeSizeWord σ x2))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 690) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 664) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 690) (exporter_block_664_taken_stack (mem := mem) (σ := σ) (x0 := x0) (x1 := x1) (x2 := x2) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k0, C0, h0⟩ := exporter_block_664_taken hstack hcond hvalid h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_664_fallthrough`. -/
def exporter_block_664_fallthrough_stack {mem : ByteArray} {σ : AccountMap} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero (extCodeSizeWord σ x2)) :: x2 :: (UInt256.ofNat 0) :: (memLoad (UInt256.ofNat 64) mem) :: (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem)) :: (memLoad (UInt256.ofNat 64) mem) :: (UInt256.ofNat 0) :: x0 :: x1 :: x2 :: R)

/-- Automatically generated RD summary for bytecode block at pc 664. -/
theorem exporter_block_664_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero (extCodeSizeWord σ x2))) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 664) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 686) (exporter_block_664_fallthrough_stack (mem := mem) (σ := σ) (x0 := x0) (x1 := x1) (x2 := x2) (R := R)) mem (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r4 := RD.genMload r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup4 (by evm_kdecide) (by evm_ov)
  have r7 := r6.sub (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup2 (by evm_kdecide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup8 (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r12⟩ := r11.extcodesize (by evm_kdecide) (by evm_ov)
  have r13 := r12.iszero (by evm_kdecide) (by evm_ov)
  have r14 := r13.dup1 (by evm_kdecide) (by evm_ov)
  have r15 := r14.iszero (by evm_kdecide) (by evm_ov)
  have r16 := r15.push2 (UInt256.ofNat 690) (by evm_kdecide) (by evm_ov)
  have r17 := r16.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 686)) r17 (by evm_kdecide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_664_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero (extCodeSizeWord σ x2))) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 664) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 686) (exporter_block_664_fallthrough_stack (mem := mem) (σ := σ) (x0 := x0) (x1 := x1) (x2 := x2) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k0, C0, h0⟩ := exporter_block_664_fallthrough hstack hcond h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 686. -/
theorem exporter_block_686 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 686) R mem aw rdata σ k C)
    : RDrev ExporterEvm.exporterRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `exporter_block_690`. -/
def exporter_block_690_stack {g : Sat256} {C : ℕ} {R : List UInt256} : List UInt256 :=
  (((g.subNat (C + ((3)) + 2)).toUInt256) :: R)

/-- Automatically generated RD summary for bytecode block at pc 690. -/
theorem exporter_block_690 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 1 ≤ 1024)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 690) (x0 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 693) (exporter_block_690_stack (g := g) (C := C) (R := R)) mem aw rdata σ (k + 3) (C + ((5))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := RD.genGas (RD.normalizeCounters (k' := k + 2) (C' := C + ((3))) r2 (by omega) (by omega)) (by evm_kdecide) (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 693)) r3 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_690_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 1 ≤ 1024)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 690) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 693) (exporter_block_690_stack (g := g) (C := C) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_690 hstack h)
  exact ⟨_, k', C', h'⟩

/- Unsupported instruction boundary at pc 693: call (0xf1). No RD transition is asserted. Summaries resume at pc 694 from a fresh symbolic RD state. -/

/-- Final stack for bytecode block summary `exporter_block_694_taken`. -/
def exporter_block_694_taken_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 694. -/
theorem exporter_block_694_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 710) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 694) (x0 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 710) (exporter_block_694_taken_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.iszero (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.iszero (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 710) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 710)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_694_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 710) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 694) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 710) (exporter_block_694_taken_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_694_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_694_fallthrough`. -/
def exporter_block_694_fallthrough_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 694. -/
theorem exporter_block_694_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 694) (x0 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 701) (exporter_block_694_fallthrough_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.iszero (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.iszero (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 710) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 701)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_694_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 694) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 701) (exporter_block_694_fallthrough_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_694_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 701. -/
theorem exporter_block_701 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 701) R mem aw rdata σ k C)
    : RDrev ExporterEvm.exporterRuntime g s0 := by
  let r0 := h
  have r1 := r0.returndatasize (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := RD.genReturndatacopy r3 (by evm_kdecide) (by
    have hz : UInt256.ofNat 0 = (⟨0⟩ : UInt256) := by decide
    simpa only [hz] using returnDataCopyFullGuard rdata) (by evm_ov)
  have r5 := r4.returndatasize (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  exact RD.genRev r6 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `exporter_block_710`. -/
def exporter_block_710_stack {x4 : UInt256} {R : List UInt256} : List UInt256 :=
  (x4 :: R)

/-- Final memory for bytecode block summary `exporter_block_710`. -/
def exporter_block_710_memory {ee : ExecutionEnv} {mem : ByteArray} {x12 : UInt256} : ByteArray :=
  ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 ((UInt256.land x12 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 32)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 710. -/
theorem exporter_block_710 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 x10 x11 x12 x13 : UInt256} {R : List UInt256}
    (hstack : R.length + 17 ≤ 1024)
    (hperm : ee.perm = true)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains x13 = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 710) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: x10 :: x11 :: x12 :: x13 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 x13 (exporter_block_710_stack (x4 := x4) (R := R)) (exporter_block_710_memory (ee := ee) (mem := mem) (x12 := x12)) (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 ((UInt256.land x12 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 32)).toNat 32)) (UInt256.sub ((UInt256.ofNat 64) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 ((UInt256.land x12 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 32)).toNat 32)))) rdata σ (k + 43) (C + ((116) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 ((UInt256.land x12 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 32)).toNat 32)) (UInt256.sub ((UInt256.ofNat 64) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 ((UInt256.land x12 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 32)).toNat 32)))) + (375 + 8 * (UInt256.sub ((UInt256.ofNat 64) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat ee.header.timestamp).toByteArray.write 0 ((UInt256.land x12 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 32)).toNat 32))).toNat + 3 * 375))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := r2.pop (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := RD.genMload r5 (by evm_kdecide) (by evm_ov)
  have r7 := r6.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup14 (by evm_kdecide) (by evm_ov)
  have r9 := r8.and (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup2 (by evm_kdecide) (by evm_ov)
  have r11 := RD.genMstore r10 (by evm_kdecide) (by evm_ov)
  have r12 := r11.timestamp (by evm_kdecide) (by evm_ov)
  have r13 := r12.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r14 := r13.dup3 (by evm_kdecide) (by evm_ov)
  have r15 := r14.add (by evm_kdecide) (by evm_ov)
  have r16 := RD.genMstore r15 (by evm_kdecide) (by evm_ov)
  have r17 := r16.dup12 (by evm_kdecide) (by evm_ov)
  have r18 := r17.swap4 (by evm_kdecide) (by evm_ov)
  have r19 := r18.pop (by evm_kdecide) (by evm_ov)
  have r20 := r19.dup5 (by evm_kdecide) (by evm_ov)
  have r21 := r20.swap3 (by evm_kdecide) (by evm_ov)
  have r22 := r21.pop (by evm_kdecide) (by evm_ov)
  have r23 := r22.pushConst (UInt256.ofNat 14269514428659123455382506993809231955739883855484288135542831042926398405741) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r24 := r23.swap2 (by evm_kdecide) (by evm_ov)
  have r25 := r24.add (by evm_kdecide) (by evm_ov)
  have r26 := r25.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r27 := RD.genMload r26 (by evm_kdecide) (by evm_ov)
  have r28 := r27.dup1 (by evm_kdecide) (by evm_ov)
  have r29 := r28.swap2 (by evm_kdecide) (by evm_ov)
  have r30 := r29.sub (by evm_kdecide) (by evm_ov)
  have r31 := r30.swap1 (by evm_kdecide) (by evm_ov)
  have r32 := RD.genLog3 r31 (by evm_kdecide) hperm (by evm_ov)
  have r33 := r32.swap9 (by evm_kdecide) (by evm_ov)
  have r34 := r33.swap8 (by evm_kdecide) (by evm_ov)
  have r35 := r34.pop (by evm_kdecide) (by evm_ov)
  have r36 := r35.pop (by evm_kdecide) (by evm_ov)
  have r37 := r36.pop (by evm_kdecide) (by evm_ov)
  have r38 := r37.pop (by evm_kdecide) (by evm_ov)
  have r39 := r38.pop (by evm_kdecide) (by evm_ov)
  have r40 := r39.pop (by evm_kdecide) (by evm_ov)
  have r41 := r40.pop (by evm_kdecide) (by evm_ov)
  have r42 := r41.pop (by evm_kdecide) (by evm_ov)
  have r43 := r42.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r43 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_710_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 x10 x11 x12 x13 : UInt256} {R : List UInt256}
    (hstack : R.length + 17 ≤ 1024)
    (hperm : ee.perm = true)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains x13 = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 710) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: x10 :: x11 :: x12 :: x13 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 x13 (exporter_block_710_stack (x4 := x4) (R := R)) (exporter_block_710_memory (ee := ee) (mem := mem) (x12 := x12)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_710 hstack hperm hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_808`. -/
def exporter_block_808_stack {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {R : List UInt256} : List UInt256 :=
  (((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) :: x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: (UInt256.ofNat 837) :: (UInt256.ofNat 0) :: x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R)

/-- Automatically generated RD summary for bytecode block at pc 808. -/
theorem exporter_block_808 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 16 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1368) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 808) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1368) (exporter_block_808_stack (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (R := R)) mem (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 22) (C + ((69) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup7 (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup7 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup7 (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup7 (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup7 (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup7 (by evm_kdecide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r10 := RD.genMload r9 (by evm_kdecide) (by evm_ov)
  have r11 := r10.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r12 := r11.add (by evm_kdecide) (by evm_ov)
  have r13 := r12.push2 (UInt256.ofNat 837) (by evm_kdecide) (by evm_ov)
  have r14 := r13.swap7 (by evm_kdecide) (by evm_ov)
  have r15 := r14.swap6 (by evm_kdecide) (by evm_ov)
  have r16 := r15.swap5 (by evm_kdecide) (by evm_ov)
  have r17 := r16.swap4 (by evm_kdecide) (by evm_ov)
  have r18 := r17.swap3 (by evm_kdecide) (by evm_ov)
  have r19 := r18.swap2 (by evm_kdecide) (by evm_ov)
  have r20 := r19.swap1 (by evm_kdecide) (by evm_ov)
  have r21 := r20.push2 (UInt256.ofNat 1368) (by evm_kdecide) (by evm_ov)
  have r22 := r21.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1368)) r22 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_808_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 16 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1368) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 808) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1368) (exporter_block_808_stack (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_808 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_837`. -/
def exporter_block_837_stack {mem : ByteArray} {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((keccakWord ((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (memLoad (UInt256.ofNat 64) mem) (x0.toByteArray.write 0 ((UInt256.sub (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem)) (UInt256.ofNat 32)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) (UInt256.ofNat 64).toNat 32)) (x0.toByteArray.write 0 ((UInt256.sub (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem)) (UInt256.ofNat 32)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) (UInt256.ofNat 64).toNat 32)) :: R)

/-- Final memory for bytecode block summary `exporter_block_837`. -/
def exporter_block_837_memory {mem : ByteArray} {x0 : UInt256} : ByteArray :=
  (x0.toByteArray.write 0 ((UInt256.sub (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem)) (UInt256.ofNat 32)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) (UInt256.ofNat 64).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 837. -/
theorem exporter_block_837 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains x8 = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 837) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 x8 (exporter_block_837_stack (mem := mem) (x0 := x0) (R := R)) (exporter_block_837_memory (mem := mem) (x0 := x0)) (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (memLoad (UInt256.ofNat 64) mem) (x0.toByteArray.write 0 ((UInt256.sub (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem)) (UInt256.ofNat 32)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) (UInt256.ofNat 64).toNat 32))) rdata σ (k + 30) (C + ((83) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (memLoad (UInt256.ofNat 64) mem) (x0.toByteArray.write 0 ((UInt256.sub (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem)) (UInt256.ofNat 32)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) (UInt256.ofNat 64).toNat 32))) + (30 + 6 * (((memLoad (memLoad (UInt256.ofNat 64) mem) (x0.toByteArray.write 0 ((UInt256.sub (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem)) (UInt256.ofNat 32)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) (UInt256.ofNat 64).toNat 32)).toNat + 31) / 32)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r3 := RD.genMload r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup2 (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup4 (by evm_kdecide) (by evm_ov)
  have r7 := r6.sub (by evm_kdecide) (by evm_ov)
  have r8 := r7.sub (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup2 (by evm_kdecide) (by evm_ov)
  have r10 := RD.genMstore r9 (by evm_kdecide) (by evm_ov)
  have r11 := r10.swap1 (by evm_kdecide) (by evm_ov)
  have r12 := r11.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r13 := RD.genMstore r12 (by evm_kdecide) (by evm_ov)
  have r14 := r13.dup1 (by evm_kdecide) (by evm_ov)
  have r15 := RD.genMload r14 (by evm_kdecide) (by evm_ov)
  have r16 := r15.swap1 (by evm_kdecide) (by evm_ov)
  have r17 := r16.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r18 := r17.add (by evm_kdecide) (by evm_ov)
  have r19 := RD.genKeccak256 r18 (by evm_kdecide) (by evm_ov)
  have r20 := r19.swap1 (by evm_kdecide) (by evm_ov)
  have r21 := r20.pop (by evm_kdecide) (by evm_ov)
  have r22 := r21.swap7 (by evm_kdecide) (by evm_ov)
  have r23 := r22.swap6 (by evm_kdecide) (by evm_ov)
  have r24 := r23.pop (by evm_kdecide) (by evm_ov)
  have r25 := r24.pop (by evm_kdecide) (by evm_ov)
  have r26 := r25.pop (by evm_kdecide) (by evm_ov)
  have r27 := r26.pop (by evm_kdecide) (by evm_ov)
  have r28 := r27.pop (by evm_kdecide) (by evm_ov)
  have r29 := r28.pop (by evm_kdecide) (by evm_ov)
  have r30 := r29.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r30 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_837_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains x8 = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 837) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 x8 (exporter_block_837_stack (mem := mem) (x0 := x0) (R := R)) (exporter_block_837_memory (mem := mem) (x0 := x0)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_837 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_871_taken`. -/
def exporter_block_871_taken_stack {ee : ExecutionEnv} {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes x0.toNat 32)) :: x0 :: R)

/-- Automatically generated RD summary for bytecode block at pc 871. -/
theorem exporter_block_871_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.eq (uInt256OfByteArray (ee.calldata.readBytes x0.toNat 32)) (UInt256.land (uInt256OfByteArray (ee.calldata.readBytes x0.toNat 32)) (UInt256.ofNat 1461501637330902918203684832716283019655932542975))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 907) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 871) (x0 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 907) (exporter_block_871_taken_stack (ee := ee) (x0 := x0) (R := R)) mem aw rdata σ (k + 10) (C + ((35))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.calldataload (by evm_kdecide) (by evm_ov)
  have r4 := r3.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup2 (by evm_kdecide) (by evm_ov)
  have r6 := r5.and (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup2 (by evm_kdecide) (by evm_ov)
  have r8 := r7.eq (by evm_kdecide) (by evm_ov)
  have r9 := r8.push2 (UInt256.ofNat 907) (by evm_kdecide) (by evm_ov)
  have r10 := r9.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 907)) r10 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_871_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.eq (uInt256OfByteArray (ee.calldata.readBytes x0.toNat 32)) (UInt256.land (uInt256OfByteArray (ee.calldata.readBytes x0.toNat 32)) (UInt256.ofNat 1461501637330902918203684832716283019655932542975))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 907) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 871) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 907) (exporter_block_871_taken_stack (ee := ee) (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_871_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_871_fallthrough`. -/
def exporter_block_871_fallthrough_stack {ee : ExecutionEnv} {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes x0.toNat 32)) :: x0 :: R)

/-- Automatically generated RD summary for bytecode block at pc 871. -/
theorem exporter_block_871_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.eq (uInt256OfByteArray (ee.calldata.readBytes x0.toNat 32)) (UInt256.land (uInt256OfByteArray (ee.calldata.readBytes x0.toNat 32)) (UInt256.ofNat 1461501637330902918203684832716283019655932542975))) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 871) (x0 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 903) (exporter_block_871_fallthrough_stack (ee := ee) (x0 := x0) (R := R)) mem aw rdata σ (k + 10) (C + ((35))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.calldataload (by evm_kdecide) (by evm_ov)
  have r4 := r3.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup2 (by evm_kdecide) (by evm_ov)
  have r6 := r5.and (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup2 (by evm_kdecide) (by evm_ov)
  have r8 := r7.eq (by evm_kdecide) (by evm_ov)
  have r9 := r8.push2 (UInt256.ofNat 907) (by evm_kdecide) (by evm_ov)
  have r10 := r9.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 903)) r10 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_871_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.eq (uInt256OfByteArray (ee.calldata.readBytes x0.toNat 32)) (UInt256.land (uInt256OfByteArray (ee.calldata.readBytes x0.toNat 32)) (UInt256.ofNat 1461501637330902918203684832716283019655932542975))) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 871) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 903) (exporter_block_871_fallthrough_stack (ee := ee) (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_871_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 903. -/
theorem exporter_block_903 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 903) R mem aw rdata σ k C)
    : RDrev ExporterEvm.exporterRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

end exporterBlocks
