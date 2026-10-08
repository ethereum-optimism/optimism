import Reasoning.Reach
import ExporterEvm.Bytecode

open Solm ABI Ethereum Ethereum.EVM
open Reasoning.Theory Reasoning.Reach

namespace exporterBlocks

/-- Final stack for bytecode block summary `exporter_block_907`. -/
def exporter_block_907_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: R)

/-- Automatically generated RD summary for bytecode block at pc 907. -/
theorem exporter_block_907 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains x2 = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 907) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 x2 (exporter_block_907_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((17))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap2 (by evm_kdecide) (by evm_ov)
  have r3 := r2.swap1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.pop (by evm_kdecide) (by evm_ov)
  have r5 := r4.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r5 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_907_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains x2 = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 907) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 x2 (exporter_block_907_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_907 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_912_taken`. -/
def exporter_block_912_taken_stack {ee : ExecutionEnv} {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes x0.toNat 32)) :: x0 :: R)

/-- Automatically generated RD summary for bytecode block at pc 912. -/
theorem exporter_block_912_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.eq (uInt256OfByteArray (ee.calldata.readBytes x0.toNat 32)) (UInt256.land (uInt256OfByteArray (ee.calldata.readBytes x0.toNat 32)) (UInt256.ofNat 4294967295))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 907) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 912) (x0 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 907) (exporter_block_912_taken_stack (ee := ee) (x0 := x0) (R := R)) mem aw rdata σ (k + 10) (C + ((35))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.calldataload (by evm_kdecide) (by evm_ov)
  have r4 := r3.push4 (UInt256.ofNat 4294967295) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup2 (by evm_kdecide) (by evm_ov)
  have r6 := r5.and (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup2 (by evm_kdecide) (by evm_ov)
  have r8 := r7.eq (by evm_kdecide) (by evm_ov)
  have r9 := r8.push2 (UInt256.ofNat 907) (by evm_kdecide) (by evm_ov)
  have r10 := r9.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 907)) r10 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_912_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.eq (uInt256OfByteArray (ee.calldata.readBytes x0.toNat 32)) (UInt256.land (uInt256OfByteArray (ee.calldata.readBytes x0.toNat 32)) (UInt256.ofNat 4294967295))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 907) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 912) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 907) (exporter_block_912_taken_stack (ee := ee) (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_912_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_912_fallthrough`. -/
def exporter_block_912_fallthrough_stack {ee : ExecutionEnv} {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes x0.toNat 32)) :: x0 :: R)

/-- Automatically generated RD summary for bytecode block at pc 912. -/
theorem exporter_block_912_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.eq (uInt256OfByteArray (ee.calldata.readBytes x0.toNat 32)) (UInt256.land (uInt256OfByteArray (ee.calldata.readBytes x0.toNat 32)) (UInt256.ofNat 4294967295))) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 912) (x0 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 928) (exporter_block_912_fallthrough_stack (ee := ee) (x0 := x0) (R := R)) mem aw rdata σ (k + 10) (C + ((35))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.calldataload (by evm_kdecide) (by evm_ov)
  have r4 := r3.push4 (UInt256.ofNat 4294967295) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup2 (by evm_kdecide) (by evm_ov)
  have r6 := r5.and (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup2 (by evm_kdecide) (by evm_ov)
  have r8 := r7.eq (by evm_kdecide) (by evm_ov)
  have r9 := r8.push2 (UInt256.ofNat 907) (by evm_kdecide) (by evm_ov)
  have r10 := r9.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 928)) r10 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_912_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.eq (uInt256OfByteArray (ee.calldata.readBytes x0.toNat 32)) (UInt256.land (uInt256OfByteArray (ee.calldata.readBytes x0.toNat 32)) (UInt256.ofNat 4294967295))) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 912) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 928) (exporter_block_912_fallthrough_stack (ee := ee) (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_912_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 928. -/
theorem exporter_block_928 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 928) R mem aw rdata σ k C)
    : RDrev ExporterEvm.exporterRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `exporter_block_932_taken`. -/
def exporter_block_932_taken_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 932. -/
theorem exporter_block_932_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 224))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 960) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 932) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 960) (exporter_block_932_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 17) (C + ((56))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup1 (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup1 (by evm_kdecide) (by evm_ov)
  have r10 := r9.push1 (UInt256.ofNat 224) (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup10 (by evm_kdecide) (by evm_ov)
  have r12 := r11.dup12 (by evm_kdecide) (by evm_ov)
  have r13 := r12.sub (by evm_kdecide) (by evm_ov)
  have r14 := r13.slt (by evm_kdecide) (by evm_ov)
  have r15 := r14.iszero (by evm_kdecide) (by evm_ov)
  have r16 := r15.push2 (UInt256.ofNat 960) (by evm_kdecide) (by evm_ov)
  have r17 := r16.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 960)) r17 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_932_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 224))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 960) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 932) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 960) (exporter_block_932_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_932_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_932_fallthrough`. -/
def exporter_block_932_fallthrough_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 932. -/
theorem exporter_block_932_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 224))) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 932) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 956) (exporter_block_932_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 17) (C + ((56))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup1 (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup1 (by evm_kdecide) (by evm_ov)
  have r10 := r9.push1 (UInt256.ofNat 224) (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup10 (by evm_kdecide) (by evm_ov)
  have r12 := r11.dup12 (by evm_kdecide) (by evm_ov)
  have r13 := r12.sub (by evm_kdecide) (by evm_ov)
  have r14 := r13.slt (by evm_kdecide) (by evm_ov)
  have r15 := r14.iszero (by evm_kdecide) (by evm_ov)
  have r16 := r15.push2 (UInt256.ofNat 960) (by evm_kdecide) (by evm_ov)
  have r17 := r16.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 956)) r17 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_932_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 224))) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 932) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 956) (exporter_block_932_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_932_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 956. -/
theorem exporter_block_956 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 956) R mem aw rdata σ k C)
    : RDrev ExporterEvm.exporterRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `exporter_block_960`. -/
def exporter_block_960_stack {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {x6 : UInt256} {x7 : UInt256} {x8 : UInt256} {R : List UInt256} : List UInt256 :=
  (x8 :: (UInt256.ofNat 969) :: x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R)

/-- Automatically generated RD summary for bytecode block at pc 960. -/
theorem exporter_block_960 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 871) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 960) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 871) (exporter_block_960_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (x7 := x7) (x8 := x8) (R := R)) mem aw rdata σ (k + 5) (C + ((18))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 969) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup10 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 871) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 871)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_960_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 871) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 960) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 871) (exporter_block_960_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (x7 := x7) (x8 := x8) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_960 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_969`. -/
def exporter_block_969_stack {ee : ExecutionEnv} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {x9 : UInt256} {R : List UInt256} : List UInt256 :=
  ((x9 + (UInt256.ofNat 96)) :: (UInt256.ofNat 997) :: x1 :: x2 :: x3 :: x4 :: x5 :: (uInt256OfByteArray (ee.calldata.readBytes (x9 + (UInt256.ofNat 64)).toNat 32)) :: (uInt256OfByteArray (ee.calldata.readBytes (x9 + (UInt256.ofNat 32)).toNat 32)) :: x0 :: x9 :: R)

/-- Automatically generated RD summary for bytecode block at pc 969. -/
theorem exporter_block_969 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 871) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 969) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 871) (exporter_block_969_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x9 := x9) (R := R)) mem aw rdata σ (k + 21) (C + ((63))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap8 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pop (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup10 (by evm_kdecide) (by evm_ov)
  have r6 := r5.add (by evm_kdecide) (by evm_ov)
  have r7 := r6.calldataload (by evm_kdecide) (by evm_ov)
  have r8 := r7.swap7 (by evm_kdecide) (by evm_ov)
  have r9 := r8.pop (by evm_kdecide) (by evm_ov)
  have r10 := r9.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup10 (by evm_kdecide) (by evm_ov)
  have r12 := r11.add (by evm_kdecide) (by evm_ov)
  have r13 := r12.calldataload (by evm_kdecide) (by evm_ov)
  have r14 := r13.swap6 (by evm_kdecide) (by evm_ov)
  have r15 := r14.pop (by evm_kdecide) (by evm_ov)
  have r16 := r15.push2 (UInt256.ofNat 997) (by evm_kdecide) (by evm_ov)
  have r17 := r16.push1 (UInt256.ofNat 96) (by evm_kdecide) (by evm_ov)
  have r18 := r17.dup11 (by evm_kdecide) (by evm_ov)
  have r19 := r18.add (by evm_kdecide) (by evm_ov)
  have r20 := r19.push2 (UInt256.ofNat 871) (by evm_kdecide) (by evm_ov)
  have r21 := r20.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 871)) r21 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_969_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 871) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 969) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 871) (exporter_block_969_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x9 := x9) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_969 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_997`. -/
def exporter_block_997_stack {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x6 : UInt256} {x7 : UInt256} {x8 : UInt256} {x9 : UInt256} {R : List UInt256} : List UInt256 :=
  ((x9 + (UInt256.ofNat 128)) :: (UInt256.ofNat 1011) :: x1 :: x2 :: x3 :: x4 :: x0 :: x6 :: x7 :: x8 :: x9 :: R)

/-- Automatically generated RD summary for bytecode block at pc 997. -/
theorem exporter_block_997 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 871) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 997) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 871) (exporter_block_997_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x6 := x6) (x7 := x7) (x8 := x8) (x9 := x9) (R := R)) mem aw rdata σ (k + 9) (C + ((29))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap5 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pop (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 1011) (by evm_kdecide) (by evm_ov)
  have r5 := r4.push1 (UInt256.ofNat 128) (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup11 (by evm_kdecide) (by evm_ov)
  have r7 := r6.add (by evm_kdecide) (by evm_ov)
  have r8 := r7.push2 (UInt256.ofNat 871) (by evm_kdecide) (by evm_ov)
  have r9 := r8.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 871)) r9 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_997_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 871) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 997) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 871) (exporter_block_997_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x6 := x6) (x7 := x7) (x8 := x8) (x9 := x9) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_997 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_1011_taken`. -/
def exporter_block_1011_taken_stack {ee : ExecutionEnv} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x5 : UInt256} {x6 : UInt256} {x7 : UInt256} {x8 : UInt256} {x9 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 18446744073709551615) :: (uInt256OfByteArray (ee.calldata.readBytes (x9 + (UInt256.ofNat 160)).toNat 32)) :: x1 :: x2 :: x3 :: x0 :: x5 :: x6 :: x7 :: x8 :: x9 :: R)

/-- Automatically generated RD summary for bytecode block at pc 1011. -/
theorem exporter_block_1011_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes (x9 + (UInt256.ofNat 160)).toNat 32)) (UInt256.ofNat 18446744073709551615))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1040) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1011) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1040) (exporter_block_1011_taken_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x5 := x5) (x6 := x6) (x7 := x7) (x8 := x8) (x9 := x9) (R := R)) mem aw rdata σ (k + 14) (C + ((46))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap4 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pop (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 160) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup10 (by evm_kdecide) (by evm_ov)
  have r6 := r5.add (by evm_kdecide) (by evm_ov)
  have r7 := r6.calldataload (by evm_kdecide) (by evm_ov)
  have r8 := r7.pushConst (UInt256.ofNat 18446744073709551615) (width := 8) (op := .PUSH8) (by decide) (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup1 (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup3 (by evm_kdecide) (by evm_ov)
  have r11 := r10.gt (by evm_kdecide) (by evm_ov)
  have r12 := r11.iszero (by evm_kdecide) (by evm_ov)
  have r13 := r12.push2 (UInt256.ofNat 1040) (by evm_kdecide) (by evm_ov)
  have r14 := r13.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1040)) r14 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_1011_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes (x9 + (UInt256.ofNat 160)).toNat 32)) (UInt256.ofNat 18446744073709551615))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1040) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1011) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1040) (exporter_block_1011_taken_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x5 := x5) (x6 := x6) (x7 := x7) (x8 := x8) (x9 := x9) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_1011_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_1011_fallthrough`. -/
def exporter_block_1011_fallthrough_stack {ee : ExecutionEnv} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x5 : UInt256} {x6 : UInt256} {x7 : UInt256} {x8 : UInt256} {x9 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 18446744073709551615) :: (uInt256OfByteArray (ee.calldata.readBytes (x9 + (UInt256.ofNat 160)).toNat 32)) :: x1 :: x2 :: x3 :: x0 :: x5 :: x6 :: x7 :: x8 :: x9 :: R)

/-- Automatically generated RD summary for bytecode block at pc 1011. -/
theorem exporter_block_1011_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes (x9 + (UInt256.ofNat 160)).toNat 32)) (UInt256.ofNat 18446744073709551615))) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1011) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1036) (exporter_block_1011_fallthrough_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x5 := x5) (x6 := x6) (x7 := x7) (x8 := x8) (x9 := x9) (R := R)) mem aw rdata σ (k + 14) (C + ((46))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap4 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pop (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 160) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup10 (by evm_kdecide) (by evm_ov)
  have r6 := r5.add (by evm_kdecide) (by evm_ov)
  have r7 := r6.calldataload (by evm_kdecide) (by evm_ov)
  have r8 := r7.pushConst (UInt256.ofNat 18446744073709551615) (width := 8) (op := .PUSH8) (by decide) (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup1 (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup3 (by evm_kdecide) (by evm_ov)
  have r11 := r10.gt (by evm_kdecide) (by evm_ov)
  have r12 := r11.iszero (by evm_kdecide) (by evm_ov)
  have r13 := r12.push2 (UInt256.ofNat 1040) (by evm_kdecide) (by evm_ov)
  have r14 := r13.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1036)) r14 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_1011_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes (x9 + (UInt256.ofNat 160)).toNat 32)) (UInt256.ofNat 18446744073709551615))) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1011) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1036) (exporter_block_1011_fallthrough_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x5 := x5) (x6 := x6) (x7 := x7) (x8 := x8) (x9 := x9) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_1011_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1036. -/
theorem exporter_block_1036 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1036) R mem aw rdata σ k C)
    : RDrev ExporterEvm.exporterRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `exporter_block_1040_taken`. -/
def exporter_block_1040_taken_stack {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {x6 : UInt256} {x7 : UInt256} {x8 : UInt256} {x9 : UInt256} {x10 : UInt256} {x11 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: (x10 + x1) :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: x10 :: x11 :: R)

/-- Automatically generated RD summary for bytecode block at pc 1040. -/
theorem exporter_block_1040_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 x10 x11 : UInt256} {R : List UInt256}
    (hstack : R.length + 15 ≤ 1024)
    (hcond : (UInt256.slt ((x10 + x1) + (UInt256.ofNat 31)) x11) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1060) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1040) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: x10 :: x11 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1060) (exporter_block_1040_taken_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (x7 := x7) (x8 := x8) (x9 := x9) (x10 := x10) (x11 := x11) (R := R)) mem aw rdata σ (k + 13) (C + ((43))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup2 (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup12 (by evm_kdecide) (by evm_ov)
  have r4 := r3.add (by evm_kdecide) (by evm_ov)
  have r5 := r4.swap2 (by evm_kdecide) (by evm_ov)
  have r6 := r5.pop (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup12 (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 31) (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup4 (by evm_kdecide) (by evm_ov)
  have r10 := r9.add (by evm_kdecide) (by evm_ov)
  have r11 := r10.slt (by evm_kdecide) (by evm_ov)
  have r12 := r11.push2 (UInt256.ofNat 1060) (by evm_kdecide) (by evm_ov)
  have r13 := r12.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1060)) r13 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_1040_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 x10 x11 : UInt256} {R : List UInt256}
    (hstack : R.length + 15 ≤ 1024)
    (hcond : (UInt256.slt ((x10 + x1) + (UInt256.ofNat 31)) x11) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1060) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1040) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: x10 :: x11 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1060) (exporter_block_1040_taken_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (x7 := x7) (x8 := x8) (x9 := x9) (x10 := x10) (x11 := x11) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_1040_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_1040_fallthrough`. -/
def exporter_block_1040_fallthrough_stack {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {x6 : UInt256} {x7 : UInt256} {x8 : UInt256} {x9 : UInt256} {x10 : UInt256} {x11 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: (x10 + x1) :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: x10 :: x11 :: R)

/-- Automatically generated RD summary for bytecode block at pc 1040. -/
theorem exporter_block_1040_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 x10 x11 : UInt256} {R : List UInt256}
    (hstack : R.length + 15 ≤ 1024)
    (hcond : (UInt256.slt ((x10 + x1) + (UInt256.ofNat 31)) x11) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1040) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: x10 :: x11 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1056) (exporter_block_1040_fallthrough_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (x7 := x7) (x8 := x8) (x9 := x9) (x10 := x10) (x11 := x11) (R := R)) mem aw rdata σ (k + 13) (C + ((43))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup2 (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup12 (by evm_kdecide) (by evm_ov)
  have r4 := r3.add (by evm_kdecide) (by evm_ov)
  have r5 := r4.swap2 (by evm_kdecide) (by evm_ov)
  have r6 := r5.pop (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup12 (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 31) (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup4 (by evm_kdecide) (by evm_ov)
  have r10 := r9.add (by evm_kdecide) (by evm_ov)
  have r11 := r10.slt (by evm_kdecide) (by evm_ov)
  have r12 := r11.push2 (UInt256.ofNat 1060) (by evm_kdecide) (by evm_ov)
  have r13 := r12.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1056)) r13 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_1040_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 x10 x11 : UInt256} {R : List UInt256}
    (hstack : R.length + 15 ≤ 1024)
    (hcond : (UInt256.slt ((x10 + x1) + (UInt256.ofNat 31)) x11) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1040) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: x10 :: x11 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1056) (exporter_block_1040_fallthrough_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (x7 := x7) (x8 := x8) (x9 := x9) (x10 := x10) (x11 := x11) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_1040_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1056. -/
theorem exporter_block_1056 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1056) R mem aw rdata σ k C)
    : RDrev ExporterEvm.exporterRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `exporter_block_1060_taken`. -/
def exporter_block_1060_taken_stack {ee : ExecutionEnv} {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes x1.toNat 32)) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 1060. -/
theorem exporter_block_1060_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes x1.toNat 32)) x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1075) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1060) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1075) (exporter_block_1060_taken_stack (ee := ee) (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 9) (C + ((32))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup2 (by evm_kdecide) (by evm_ov)
  have r3 := r2.calldataload (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup2 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup2 (by evm_kdecide) (by evm_ov)
  have r6 := r5.gt (by evm_kdecide) (by evm_ov)
  have r7 := r6.iszero (by evm_kdecide) (by evm_ov)
  have r8 := r7.push2 (UInt256.ofNat 1075) (by evm_kdecide) (by evm_ov)
  have r9 := r8.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1075)) r9 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_1060_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes x1.toNat 32)) x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1075) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1060) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1075) (exporter_block_1060_taken_stack (ee := ee) (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_1060_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `exporter_block_1060_fallthrough`. -/
def exporter_block_1060_fallthrough_stack {ee : ExecutionEnv} {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes x1.toNat 32)) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 1060. -/
theorem exporter_block_1060_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes x1.toNat 32)) x0)) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1060) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1071) (exporter_block_1060_fallthrough_stack (ee := ee) (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 9) (C + ((32))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup2 (by evm_kdecide) (by evm_ov)
  have r3 := r2.calldataload (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup2 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup2 (by evm_kdecide) (by evm_ov)
  have r6 := r5.gt (by evm_kdecide) (by evm_ov)
  have r7 := r6.iszero (by evm_kdecide) (by evm_ov)
  have r8 := r7.push2 (UInt256.ofNat 1075) (by evm_kdecide) (by evm_ov)
  have r9 := r8.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1071)) r9 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_1060_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes x1.toNat 32)) x0)) = (UInt256.ofNat 0))
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1060) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1071) (exporter_block_1060_fallthrough_stack (ee := ee) (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_1060_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1071. -/
theorem exporter_block_1071 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1071) R mem aw rdata σ k C)
    : RDrev ExporterEvm.exporterRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Automatically generated RD summary for bytecode block at pc 1075. -/
theorem exporter_block_1075_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 x10 x11 x12 : UInt256} {R : List UInt256}
    (hstack : R.length + 17 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt ((x2 + x0) + (UInt256.ofNat 32)) x12)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1093) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1075) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: x10 :: x11 :: x12 :: R) mem aw rdata σ k C)
    : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1093) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: x10 :: x11 :: x12 :: R) mem aw rdata σ (k + 11) (C + ((38))) := by
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
  have r11 := r10.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1093)) r11 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem exporter_block_1075_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 x10 x11 x12 : UInt256} {R : List UInt256}
    (hstack : R.length + 17 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt ((x2 + x0) + (UInt256.ofNat 32)) x12)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExporterEvm.exporterRuntime 0).contains (UInt256.ofNat 1093) = true)
    (h : RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1075) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: x10 :: x11 :: x12 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExporterEvm.exporterRuntime ee g s0 (UInt256.ofNat 1093) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: x10 :: x11 :: x12 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (exporter_block_1075_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

end exporterBlocks
