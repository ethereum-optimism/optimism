import Reasoning.Reach
import ExporterEvm.Bytecode

open Solm ABI Ethereum Ethereum.EVM
open Reasoning.Theory Reasoning.Reach

namespace exporterBlocks

/-- Final stack for bytecode block summary `exporter_block_1299`. -/
def exporter_block_1299_stack {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {R : List UInt256} : List UInt256 :=
  (x2 :: (x0 + (UInt256.ofNat 96)) :: (UInt256.ofNat 1346) :: (UInt256.ofNat 0) :: x0 :: x1 :: x2 :: x3 :: R)

/-- Final memory for bytecode block summary `exporter_block_1299`. -/
def exporter_block_1299_memory {mem : ByteArray} {x0 : UInt256} {x3 : UInt256} : ByteArray :=
  ((UInt256.ofNat 96).toByteArray.write 0 ((UInt256.land x3 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 mem x0.toNat 32) (x0 + (UInt256.ofNat 32)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1299. -/
theorem exporter_block_1299 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1132) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1299) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1132) (exporter_block_1299_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (R := R)) (exporter_block_1299_memory (mem := mem) (x0 := x0) (x3 := x3)) (M (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) rdata σ (k + 19) (C + ((60) + (memExpansionCost aw x0 (⟨32⟩ : UInt256)) + (memExpansionCost (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup5 (by evm_kdecide) (by evm_ov)
  have r4 := r3.and (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup2 (by evm_kdecide) (by evm_ov)
  have r6 := RD.genMstore r5 (by evm_kdecide) (by evm_ov)
  have r7 := r6.push1 (UInt256.ofNat 96) (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup3 (by evm_kdecide) (by evm_ov)
  have r10 := r9.add (by evm_kdecide) (by evm_ov)
  have r11 := RD.genMstore r10 (by evm_kdecide) (by evm_ov)
  have r12 := r11.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r13 := r12.push2 (UInt256.ofNat 1346) (by evm_kdecide) (by evm_ov)
  have r14 := r13.push1 (UInt256.ofNat 96) (by evm_kdecide) (by evm_ov)
  have r15 := r14.dup4 (by evm_kdecide) (by evm_ov)
  have r16 := r15.add (by evm_kdecide) (by evm_ov)
  have r17 := r16.dup6 (by evm_kdecide) (by evm_ov)
  have r18 := r17.push2 (UInt256.ofNat 1132) (by evm_kdecide) (by evm_ov)
  have r19 := r18.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1132)) r19 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_1299_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1132) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1299) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1132) (exporter_block_1299_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (R := R)) (exporter_block_1299_memory (mem := mem) (x0 := x0) (x3 := x3)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_1299 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_1346`. -/
def exporter_block_1346_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: R)

/-- Final memory for bytecode block summary `exporter_block_1346`. -/
def exporter_block_1346_memory {mem : ByteArray} {x2 : UInt256} {x3 : UInt256} : ByteArray :=
  ((UInt256.land x3 (UInt256.ofNat 4294967295)).toByteArray.write 0 mem (x2 + (UInt256.ofNat 64)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1346. -/
theorem exporter_block_1346 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains x6 = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1346) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 x6 (exporter_block_1346_stack (x0 := x0) (R := R)) (exporter_block_1346_memory (mem := mem) (x2 := x2) (x3 := x3)) (M aw (x2 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) rdata σ (k + 17) (C + ((49) + (memExpansionCost aw (x2 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pop (by evm_kdecide) (by evm_ov)
  have r4 := r3.push4 (UInt256.ofNat 4294967295) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup4 (by evm_kdecide) (by evm_ov)
  have r6 := r5.and (by evm_kdecide) (by evm_ov)
  have r7 := r6.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup4 (by evm_kdecide) (by evm_ov)
  have r9 := r8.add (by evm_kdecide) (by evm_ov)
  have r10 := RD.genMstore r9 (by evm_kdecide) (by evm_ov)
  have r11 := r10.swap5 (by evm_kdecide) (by evm_ov)
  have r12 := r11.swap4 (by evm_kdecide) (by evm_ov)
  have r13 := r12.pop (by evm_kdecide) (by evm_ov)
  have r14 := r13.pop (by evm_kdecide) (by evm_ov)
  have r15 := r14.pop (by evm_kdecide) (by evm_ov)
  have r16 := r15.pop (by evm_kdecide) (by evm_ov)
  have r17 := r16.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r17 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_1346_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains x6 = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1346) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 x6 (exporter_block_1346_stack (x0 := x0) (R := R)) (exporter_block_1346_memory (mem := mem) (x2 := x2) (x3 := x3)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_1346 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_1368`. -/
def exporter_block_1368_stack {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {x6 : UInt256} {R : List UInt256} : List UInt256 :=
  (x1 :: (x0 + (UInt256.ofNat 192)) :: (UInt256.ofNat 1443) :: (UInt256.ofNat 0) :: x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: R)

/-- Final memory for bytecode block summary `exporter_block_1368`. -/
def exporter_block_1368_memory {mem : ByteArray} {x0 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {x6 : UInt256} : ByteArray :=
  ((UInt256.ofNat 192).toByteArray.write 0 ((UInt256.land x2 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.land x3 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 (x4.toByteArray.write 0 (x5.toByteArray.write 0 (x6.toByteArray.write 0 mem x0.toNat 32) (x0 + (UInt256.ofNat 32)).toNat 32) (x0 + (UInt256.ofNat 64)).toNat 32) (x0 + (UInt256.ofNat 96)).toNat 32) (x0 + (UInt256.ofNat 128)).toNat 32) (x0 + (UInt256.ofNat 160)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1368. -/
theorem exporter_block_1368 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1132) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1368) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1132) (exporter_block_1368_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (R := R)) (exporter_block_1368_memory (mem := mem) (x0 := x0) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6)) (M (M (M (M (M (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 96)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 128)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 160)) (⟨32⟩ : UInt256)) rdata σ (k + 43) (C + ((131) + (memExpansionCost aw x0 (⟨32⟩ : UInt256)) + (memExpansionCost (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 96)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 96)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 128)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 96)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 128)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 160)) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup7 (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup2 (by evm_kdecide) (by evm_ov)
  have r4 := RD.genMstore r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup6 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup3 (by evm_kdecide) (by evm_ov)
  have r8 := r7.add (by evm_kdecide) (by evm_ov)
  have r9 := RD.genMstore r8 (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup5 (by evm_kdecide) (by evm_ov)
  have r11 := r10.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r12 := r11.dup3 (by evm_kdecide) (by evm_ov)
  have r13 := r12.add (by evm_kdecide) (by evm_ov)
  have r14 := RD.genMstore r13 (by evm_kdecide) (by evm_ov)
  have r15 := r14.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r16 := r15.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r17 := r16.dup1 (by evm_kdecide) (by evm_ov)
  have r18 := r17.dup7 (by evm_kdecide) (by evm_ov)
  have r19 := r18.and (by evm_kdecide) (by evm_ov)
  have r20 := r19.push1 (UInt256.ofNat 96) (by evm_kdecide) (by evm_ov)
  have r21 := r20.dup5 (by evm_kdecide) (by evm_ov)
  have r22 := r21.add (by evm_kdecide) (by evm_ov)
  have r23 := RD.genMstore r22 (by evm_kdecide) (by evm_ov)
  have r24 := r23.dup1 (by evm_kdecide) (by evm_ov)
  have r25 := r24.dup6 (by evm_kdecide) (by evm_ov)
  have r26 := r25.and (by evm_kdecide) (by evm_ov)
  have r27 := r26.push1 (UInt256.ofNat 128) (by evm_kdecide) (by evm_ov)
  have r28 := r27.dup5 (by evm_kdecide) (by evm_ov)
  have r29 := r28.add (by evm_kdecide) (by evm_ov)
  have r30 := RD.genMstore r29 (by evm_kdecide) (by evm_ov)
  have r31 := r30.pop (by evm_kdecide) (by evm_ov)
  have r32 := r31.push1 (UInt256.ofNat 192) (by evm_kdecide) (by evm_ov)
  have r33 := r32.push1 (UInt256.ofNat 160) (by evm_kdecide) (by evm_ov)
  have r34 := r33.dup4 (by evm_kdecide) (by evm_ov)
  have r35 := r34.add (by evm_kdecide) (by evm_ov)
  have r36 := RD.genMstore r35 (by evm_kdecide) (by evm_ov)
  have r37 := r36.push2 (UInt256.ofNat 1443) (by evm_kdecide) (by evm_ov)
  have r38 := r37.push1 (UInt256.ofNat 192) (by evm_kdecide) (by evm_ov)
  have r39 := r38.dup4 (by evm_kdecide) (by evm_ov)
  have r40 := r39.add (by evm_kdecide) (by evm_ov)
  have r41 := r40.dup5 (by evm_kdecide) (by evm_ov)
  have r42 := r41.push2 (UInt256.ofNat 1132) (by evm_kdecide) (by evm_ov)
  have r43 := r42.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1132)) r43 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_1368_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1132) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1368) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1132) (exporter_block_1368_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (R := R)) (exporter_block_1368_memory (mem := mem) (x0 := x0) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_1368 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_1443`. -/
def exporter_block_1443_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: R)

/-- Automatically generated RD summary for bytecode block at pc 1443. -/
theorem exporter_block_1443 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains x9 = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1443) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 x9 (exporter_block_1443_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 12) (C + ((31))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap9 (by evm_kdecide) (by evm_ov)
  have r3 := r2.swap8 (by evm_kdecide) (by evm_ov)
  have r4 := r3.pop (by evm_kdecide) (by evm_ov)
  have r5 := r4.pop (by evm_kdecide) (by evm_ov)
  have r6 := r5.pop (by evm_kdecide) (by evm_ov)
  have r7 := r6.pop (by evm_kdecide) (by evm_ov)
  have r8 := r7.pop (by evm_kdecide) (by evm_ov)
  have r9 := r8.pop (by evm_kdecide) (by evm_ov)
  have r10 := r9.pop (by evm_kdecide) (by evm_ov)
  have r11 := r10.pop (by evm_kdecide) (by evm_ov)
  have r12 := r11.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r12 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_1443_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains x9 = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1443) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 x9 (exporter_block_1443_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_1443 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1455. -/
theorem exporter_block_1455 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1455) R mem aw rdata σ k C)
    : RDinvalid ExporterEvm.exporterRuntime g s0 := by
  let r0 := h
  exact RD.invalid r0 (by evm_kdecide)

end exporterBlocks
