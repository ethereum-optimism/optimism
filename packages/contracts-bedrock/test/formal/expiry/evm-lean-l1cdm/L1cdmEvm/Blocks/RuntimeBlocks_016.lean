import Reasoning.Reach
import L1cdmEvm.Bytecode

open Solm ABI Ethereum Ethereum.EVM
open Reasoning.Theory Reasoning.Reach

namespace l1cdmBlocks

/-- Final stack for bytecode block summary `l1cdm_block_4205_fallthrough`. -/
def l1cdm_block_4205_fallthrough_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: R)

/-- Automatically generated RD summary for bytecode block at pc 4205. -/
theorem l1cdm_block_4205_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.land x0 (UInt256.ofNat 1461501637330902918203684832716283019655932542975))) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4205) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4236) (l1cdm_block_4205_fallthrough_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 9) (C + ((31))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pop (by evm_kdecide) (by evm_ov)
  have r4 := r3.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup2 (by evm_kdecide) (by evm_ov)
  have r6 := r5.and (by evm_kdecide) (by evm_ov)
  have r7 := r6.iszero (by evm_kdecide) (by evm_ov)
  have r8 := r7.push2 (UInt256.ofNat 4353) (by evm_kdecide) (by evm_ov)
  have r9 := r8.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 4236)) r9 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_4205_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.land x0 (UInt256.ofNat 1461501637330902918203684832716283019655932542975))) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4205) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4236) (l1cdm_block_4205_fallthrough_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_4205_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_4236`. -/
def l1cdm_block_4236_stack {g : Sat256} {mem : ByteArray} {aw : UInt256} {C : ℕ} {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  (((g.subNat (C + ((69) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256))) + 2)).toUInt256) :: (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) x0) :: (memLoad (UInt256.ofNat 64) ((UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 2376452955)) (UInt256.ofNat 224)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)) :: (UInt256.sub ((UInt256.ofNat 4) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (UInt256.ofNat 64) ((UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 2376452955)) (UInt256.ofNat 224)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32))) :: (memLoad (UInt256.ofNat 64) ((UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 2376452955)) (UInt256.ofNat 224)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)) :: (UInt256.ofNat 32) :: ((UInt256.ofNat 4) + (memLoad (UInt256.ofNat 64) mem)) :: (UInt256.ofNat 2376452955) :: (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) x0) :: x0 :: R)

/-- Final memory for bytecode block summary `l1cdm_block_4236`. -/
def l1cdm_block_4236_memory {mem : ByteArray} : ByteArray :=
  ((UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 2376452955)) (UInt256.ofNat 224)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 4236. -/
theorem l1cdm_block_4236 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4236) (x0 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4293) (l1cdm_block_4236_stack (g := g) (mem := mem) (aw := aw) (C := C) (x0 := x0) (R := R)) (l1cdm_block_4236_memory (mem := mem)) (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 24) (C + ((71) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.dup1 (by evm_kdecide) (by evm_ov)
  have r2 := r1.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r3 := r2.and (by evm_kdecide) (by evm_ov)
  have r4 := r3.push4 (UInt256.ofNat 2376452955) (by evm_kdecide) (by evm_ov)
  have r5 := r4.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r6 := RD.genMload r5 (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup2 (by evm_kdecide) (by evm_ov)
  have r8 := r7.push4 (UInt256.ofNat 4294967295) (by evm_kdecide) (by evm_ov)
  have r9 := r8.and (by evm_kdecide) (by evm_ov)
  have r10 := r9.push1 (UInt256.ofNat 224) (by evm_kdecide) (by evm_ov)
  have r11 := r10.shl (by evm_kdecide) (by evm_ov)
  have r12 := r11.dup2 (by evm_kdecide) (by evm_ov)
  have r13 := RD.genMstore r12 (by evm_kdecide) (by evm_ov)
  have r14 := r13.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r15 := r14.add (by evm_kdecide) (by evm_ov)
  have r16 := r15.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r17 := r16.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r18 := RD.genMload r17 (by evm_kdecide) (by evm_ov)
  have r19 := r18.dup1 (by evm_kdecide) (by evm_ov)
  have r20 := r19.dup4 (by evm_kdecide) (by evm_ov)
  have r21 := r20.sub (by evm_kdecide) (by evm_ov)
  have r22 := r21.dup2 (by evm_kdecide) (by evm_ov)
  have r23 := r22.dup7 (by evm_kdecide) (by evm_ov)
  have r24 := RD.genGas (RD.normalizeCounters (k' := k + 23) (C' := C + ((69) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) r23 (by omega) (by omega)) (by evm_kdecide) (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 4293)) r24 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_4236_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4236) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4293) (l1cdm_block_4236_stack (g := g) (mem := mem) (aw := aw) (C := C) (x0 := x0) (R := R)) (l1cdm_block_4236_memory (mem := mem)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_4236 hstack h)
  exact ⟨_, k', C', h'⟩

/- Unsupported instruction boundary at pc 4293: staticcall (0xfa). No RD transition is asserted. Summaries resume at pc 4294 from a fresh symbolic RD state. -/

/-- Final stack for bytecode block summary `l1cdm_block_4294_taken`. -/
def l1cdm_block_4294_taken_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 4294. -/
theorem l1cdm_block_4294_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 4310) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4294) (x0 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4310) (l1cdm_block_4294_taken_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.iszero (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.iszero (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 4310) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 4310)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_4294_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 4310) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4294) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4310) (l1cdm_block_4294_taken_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_4294_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_4294_fallthrough`. -/
def l1cdm_block_4294_fallthrough_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 4294. -/
theorem l1cdm_block_4294_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4294) (x0 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4301) (l1cdm_block_4294_fallthrough_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.iszero (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.iszero (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 4310) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 4301)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_4294_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4294) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4301) (l1cdm_block_4294_fallthrough_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_4294_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 4301. -/
theorem l1cdm_block_4301 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4301) R mem aw rdata σ k C)
    : RDrev L1cdmEvm.l1cdmRuntime g s0 := by
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

/-- Final stack for bytecode block summary `l1cdm_block_4310`. -/
def l1cdm_block_4310_stack {mem : ByteArray} {rdata : ByteArray} {R : List UInt256} : List UInt256 :=
  ((memLoad (UInt256.ofNat 64) mem) :: ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat rdata.size)) :: (UInt256.ofNat 4346) :: R)

/-- Final memory for bytecode block summary `l1cdm_block_4310`. -/
def l1cdm_block_4310_memory {mem : ByteArray} {rdata : ByteArray} : ByteArray :=
  (((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat rdata.size) + (UInt256.ofNat 31)) (UInt256.lnot (UInt256.ofNat 31)))).toByteArray.write 0 mem (UInt256.ofNat 64).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 4310. -/
theorem l1cdm_block_4310 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10218) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4310) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10218) (l1cdm_block_4310_stack (mem := mem) (rdata := rdata) (R := R)) (l1cdm_block_4310_memory (mem := mem) (rdata := rdata)) (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 28) (C + ((81) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
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
  have r24 := r23.push2 (UInt256.ofNat 4346) (by evm_kdecide) (by evm_ov)
  have r25 := r24.swap2 (by evm_kdecide) (by evm_ov)
  have r26 := r25.swap1 (by evm_kdecide) (by evm_ov)
  have r27 := r26.push2 (UInt256.ofNat 10218) (by evm_kdecide) (by evm_ov)
  have r28 := r27.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10218)) r28 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_4310_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10218) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4310) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10218) (l1cdm_block_4310_stack (mem := mem) (rdata := rdata) (R := R)) (l1cdm_block_4310_memory (mem := mem) (rdata := rdata)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_4310 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_4346`. -/
def l1cdm_block_4346_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: R)

/-- Automatically generated RD summary for bytecode block at pc 4346. -/
theorem l1cdm_block_4346 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x4 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4346) (x0 :: x1 :: x2 :: x3 :: x4 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 x4 (l1cdm_block_4346_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 7) (C + ((21))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap3 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pop (by evm_kdecide) (by evm_ov)
  have r4 := r3.pop (by evm_kdecide) (by evm_ov)
  have r5 := r4.pop (by evm_kdecide) (by evm_ov)
  have r6 := r5.swap1 (by evm_kdecide) (by evm_ov)
  have r7 := r6.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r7 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_4346_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x4 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4346) (x0 :: x1 :: x2 :: x3 :: x4 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 x4 (l1cdm_block_4346_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_4346 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 4353. -/
theorem l1cdm_block_4353 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4353) R mem aw rdata σ k C)
    : RDrev L1cdmEvm.l1cdmRuntime g s0 := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r3 := RD.genMload r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.pushConst (UInt256.ofNat 23126736453864174354243441582698999348695820928149712665524827982056963178496) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup2 (by evm_kdecide) (by evm_ov)
  have r6 := RD.genMstore r5 (by evm_kdecide) (by evm_ov)
  have r7 := r6.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r8 := r7.add (by evm_kdecide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r10 := RD.genMload r9 (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup1 (by evm_kdecide) (by evm_ov)
  have r12 := r11.swap2 (by evm_kdecide) (by evm_ov)
  have r13 := r12.sub (by evm_kdecide) (by evm_ov)
  have r14 := r13.swap1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r14 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `l1cdm_block_4403_taken`. -/
def l1cdm_block_4403_taken_stack {ee : ExecutionEnv} {σ : AccountMap} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero (UInt256.land (UInt256.ofNat 255) (UInt256.div (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 0) (⟨0⟩ : UInt256))) (UInt256.ofNat 374144419156711147060143317175368453031918731001856)))) :: (UInt256.ofNat 3) :: R)

/-- Automatically generated RD summary for bytecode block at pc 4403. -/
theorem l1cdm_block_4403_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero (UInt256.land (UInt256.ofNat 255) (UInt256.div (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 0) (⟨0⟩ : UInt256))) (UInt256.ofNat 374144419156711147060143317175368453031918731001856))))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 4511) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4403) R mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4511) (l1cdm_block_4403_taken_stack (ee := ee) (σ := σ) (R := R)) mem aw rdata σ k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pushConst (UInt256.ofNat 3) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r4⟩ := RD.sload r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.pushConst (UInt256.ofNat 374144419156711147060143317175368453031918731001856) (width := 22) (op := .PUSH22) (by decide) (by evm_kdecide) (by evm_ov)
  have r6 := r5.swap1 (by evm_kdecide) (by evm_ov)
  have r7 := r6.div (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 255) (by evm_kdecide) (by evm_ov)
  have r9 := r8.and (by evm_kdecide) (by evm_ov)
  have r10 := r9.iszero (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup1 (by evm_kdecide) (by evm_ov)
  have r12 := r11.iszero (by evm_kdecide) (by evm_ov)
  have r13 := r12.push2 (UInt256.ofNat 4511) (by evm_kdecide) (by evm_ov)
  have r14 := r13.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 4511)) r14 (by evm_kdecide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_4403_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero (UInt256.land (UInt256.ofNat 255) (UInt256.div (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 0) (⟨0⟩ : UInt256))) (UInt256.ofNat 374144419156711147060143317175368453031918731001856))))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 4511) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4403) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4511) (l1cdm_block_4403_taken_stack (ee := ee) (σ := σ) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k0, C0, h0⟩ := l1cdm_block_4403_taken hstack hcond hvalid h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_4403_fallthrough`. -/
def l1cdm_block_4403_fallthrough_stack {ee : ExecutionEnv} {σ : AccountMap} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero (UInt256.land (UInt256.ofNat 255) (UInt256.div (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 0) (⟨0⟩ : UInt256))) (UInt256.ofNat 374144419156711147060143317175368453031918731001856)))) :: (UInt256.ofNat 3) :: R)

/-- Automatically generated RD summary for bytecode block at pc 4403. -/
theorem l1cdm_block_4403_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero (UInt256.land (UInt256.ofNat 255) (UInt256.div (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 0) (⟨0⟩ : UInt256))) (UInt256.ofNat 374144419156711147060143317175368453031918731001856))))) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4403) R mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4475) (l1cdm_block_4403_fallthrough_stack (ee := ee) (σ := σ) (R := R)) mem aw rdata σ k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pushConst (UInt256.ofNat 3) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r4⟩ := RD.sload r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.pushConst (UInt256.ofNat 374144419156711147060143317175368453031918731001856) (width := 22) (op := .PUSH22) (by decide) (by evm_kdecide) (by evm_ov)
  have r6 := r5.swap1 (by evm_kdecide) (by evm_ov)
  have r7 := r6.div (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 255) (by evm_kdecide) (by evm_ov)
  have r9 := r8.and (by evm_kdecide) (by evm_ov)
  have r10 := r9.iszero (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup1 (by evm_kdecide) (by evm_ov)
  have r12 := r11.iszero (by evm_kdecide) (by evm_ov)
  have r13 := r12.push2 (UInt256.ofNat 4511) (by evm_kdecide) (by evm_ov)
  have r14 := r13.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 4475)) r14 (by evm_kdecide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_4403_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero (UInt256.land (UInt256.ofNat 255) (UInt256.div (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 0) (⟨0⟩ : UInt256))) (UInt256.ofNat 374144419156711147060143317175368453031918731001856))))) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4403) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4475) (l1cdm_block_4403_fallthrough_stack (ee := ee) (σ := σ) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k0, C0, h0⟩ := l1cdm_block_4403_fallthrough hstack hcond h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_4475`. -/
def l1cdm_block_4475_stack {ee : ExecutionEnv} {σ : AccountMap} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.lt (UInt256.land (UInt256.div (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 0) (⟨0⟩ : UInt256))) (UInt256.ofNat 1461501637330902918203684832716283019655932542976)) (UInt256.ofNat 255)) (UInt256.land x1 (UInt256.ofNat 255))) :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 4475. -/
theorem l1cdm_block_4475 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4475) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4511) (l1cdm_block_4475_stack (ee := ee) (σ := σ) (x1 := x1) (R := R)) mem aw rdata σ k' C' := by
  let r0 := h
  have r1 := r0.pop (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r3⟩ := RD.sload r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 255) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup4 (by evm_kdecide) (by evm_ov)
  have r7 := r6.and (by evm_kdecide) (by evm_ov)
  have r8 := r7.pushConst (UInt256.ofNat 1461501637330902918203684832716283019655932542976) (width := 21) (op := .PUSH21) (by decide) (by evm_kdecide) (by evm_ov)
  have r9 := r8.swap1 (by evm_kdecide) (by evm_ov)
  have r10 := r9.swap3 (by evm_kdecide) (by evm_ov)
  have r11 := r10.div (by evm_kdecide) (by evm_ov)
  have r12 := r11.and (by evm_kdecide) (by evm_ov)
  have r13 := r12.lt (by evm_kdecide) (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 4511)) r13 (by evm_kdecide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_4475_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4475) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4511) (l1cdm_block_4475_stack (ee := ee) (σ := σ) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k0, C0, h0⟩ := l1cdm_block_4475 hstack h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_4511_taken`. -/
def l1cdm_block_4511_taken_stack {R : List UInt256} : List UInt256 :=
  R

/-- Automatically generated RD summary for bytecode block at pc 4511. -/
theorem l1cdm_block_4511_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : x0 ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 4656) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4511) (x0 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4656) (l1cdm_block_4511_taken_stack (R := R)) mem aw rdata σ (k + 3) (C + ((14))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 4656) (by evm_kdecide) (by evm_ov)
  have r3 := r2.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 4656)) r3 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_4511_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : x0 ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 4656) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4511) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4656) (l1cdm_block_4511_taken_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_4511_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_4511_fallthrough`. -/
def l1cdm_block_4511_fallthrough_stack {R : List UInt256} : List UInt256 :=
  R

/-- Automatically generated RD summary for bytecode block at pc 4511. -/
theorem l1cdm_block_4511_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : x0 = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4511) (x0 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4516) (l1cdm_block_4511_fallthrough_stack (R := R)) mem aw rdata σ (k + 3) (C + ((14))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 4656) (by evm_kdecide) (by evm_ov)
  have r3 := r2.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 4516)) r3 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_4511_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : x0 = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4511) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4516) (l1cdm_block_4511_fallthrough_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_4511_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_4516`. -/
def l1cdm_block_4516_stack {mem : ByteArray} {R : List UInt256} : List UInt256 :=
  (((UInt256.ofNat 132) + (memLoad (UInt256.ofNat 64) mem)) :: R)

/-- Final memory for bytecode block summary `l1cdm_block_4516`. -/
def l1cdm_block_4516_memory {mem : ByteArray} : ByteArray :=
  ((UInt256.ofNat 45445297051470054334538976711054531813460623115156291166328260229624781340672).toByteArray.write 0 ((UInt256.ofNat 33213918945522163348297488160619434111254143694905912425159868126486596838753).toByteArray.write 0 ((UInt256.ofNat 46).toByteArray.write 0 ((UInt256.ofNat 32).toByteArray.write 0 ((UInt256.ofNat 3963877391197344453575983046348115674221700746820753546331534351508065746944).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 4516. -/
theorem l1cdm_block_4516 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4516) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4647) (l1cdm_block_4516_stack (mem := mem) (R := R)) (l1cdm_block_4516_memory (mem := mem)) (M (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)) (⟨32⟩ : UInt256)) rdata σ (k + 27) (C + ((81) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r2 := RD.genMload r1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pushConst (UInt256.ofNat 3963877391197344453575983046348115674221700746820753546331534351508065746944) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup2 (by evm_kdecide) (by evm_ov)
  have r5 := RD.genMstore r4 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r7 := r6.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup3 (by evm_kdecide) (by evm_ov)
  have r9 := r8.add (by evm_kdecide) (by evm_ov)
  have r10 := RD.genMstore r9 (by evm_kdecide) (by evm_ov)
  have r11 := r10.push1 (UInt256.ofNat 46) (by evm_kdecide) (by evm_ov)
  have r12 := r11.push1 (UInt256.ofNat 36) (by evm_kdecide) (by evm_ov)
  have r13 := r12.dup3 (by evm_kdecide) (by evm_ov)
  have r14 := r13.add (by evm_kdecide) (by evm_ov)
  have r15 := RD.genMstore r14 (by evm_kdecide) (by evm_ov)
  have r16 := r15.pushConst (UInt256.ofNat 33213918945522163348297488160619434111254143694905912425159868126486596838753) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r17 := r16.push1 (UInt256.ofNat 68) (by evm_kdecide) (by evm_ov)
  have r18 := r17.dup3 (by evm_kdecide) (by evm_ov)
  have r19 := r18.add (by evm_kdecide) (by evm_ov)
  have r20 := RD.genMstore r19 (by evm_kdecide) (by evm_ov)
  have r21 := r20.pushConst (UInt256.ofNat 45445297051470054334538976711054531813460623115156291166328260229624781340672) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r22 := r21.push1 (UInt256.ofNat 100) (by evm_kdecide) (by evm_ov)
  have r23 := r22.dup3 (by evm_kdecide) (by evm_ov)
  have r24 := r23.add (by evm_kdecide) (by evm_ov)
  have r25 := RD.genMstore r24 (by evm_kdecide) (by evm_ov)
  have r26 := r25.push1 (UInt256.ofNat 132) (by evm_kdecide) (by evm_ov)
  have r27 := r26.add (by evm_kdecide) (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 4647)) r27 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_4516_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4516) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4647) (l1cdm_block_4516_stack (mem := mem) (R := R)) (l1cdm_block_4516_memory (mem := mem)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_4516 hstack h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 4647. -/
theorem l1cdm_block_4647 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4647) (x0 :: R) mem aw rdata σ k C)
    : RDrev L1cdmEvm.l1cdmRuntime g s0 := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r3 := RD.genMload r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup1 (by evm_kdecide) (by evm_ov)
  have r5 := r4.swap2 (by evm_kdecide) (by evm_ov)
  have r6 := r5.sub (by evm_kdecide) (by evm_ov)
  have r7 := r6.swap1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r7 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `l1cdm_block_4656`. -/
def l1cdm_block_4656_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 4792) :: x0 :: R)

/-- Automatically generated RD summary for bytecode block at pc 4656. -/
theorem l1cdm_block_4656 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hperm : ee.perm = true)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 8139) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4656) (x0 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 8139) (l1cdm_block_4656_stack (x0 := x0) (R := R)) mem aw rdata (sstoreAccountMap ee.codeOwner σ (UInt256.ofNat 0) (UInt256.lor (UInt256.ofNat 374144419156711147060143317175368453031918731001856) (UInt256.lor (UInt256.land (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 0) (⟨0⟩ : UInt256))) (UInt256.ofNat 115792089237316195423570889229178105372547240187155051977849890856373925707775)) (UInt256.land (UInt256.mul (UInt256.ofNat 1461501637330902918203684832716283019655932542976) (UInt256.land x0 (UInt256.ofNat 255))) (UInt256.ofNat 115792089237316195423570889601861022891927484329094684320502060868636724166655))))) k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r4⟩ := RD.sload r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.pushConst (UInt256.ofNat 115792089237316195423570889601861022891927484329094684320502060868636724166655) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 255) (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup5 (by evm_kdecide) (by evm_ov)
  have r8 := r7.and (by evm_kdecide) (by evm_ov)
  have r9 := r8.pushConst (UInt256.ofNat 1461501637330902918203684832716283019655932542976) (width := 21) (op := .PUSH21) (by decide) (by evm_kdecide) (by evm_ov)
  have r10 := r9.mul (by evm_kdecide) (by evm_ov)
  have r11 := r10.and (by evm_kdecide) (by evm_ov)
  have r12 := r11.pushConst (UInt256.ofNat 115792089237316195423570889229178105372547240187155051977849890856373925707775) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r13 := r12.swap1 (by evm_kdecide) (by evm_ov)
  have r14 := r13.swap2 (by evm_kdecide) (by evm_ov)
  have r15 := r14.and (by evm_kdecide) (by evm_ov)
  have r16 := r15.or (by evm_kdecide) (by evm_ov)
  have r17 := r16.pushConst (UInt256.ofNat 374144419156711147060143317175368453031918731001856) (width := 22) (op := .PUSH22) (by decide) (by evm_kdecide) (by evm_ov)
  have r18 := r17.or (by evm_kdecide) (by evm_ov)
  have r19 := r18.swap1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r20⟩ := RD.sstore r19 hperm (by evm_kdecide) (by evm_ov)
  have r21 := r20.push2 (UInt256.ofNat 4792) (by evm_kdecide) (by evm_ov)
  have r22 := r21.push2 (UInt256.ofNat 8139) (by evm_kdecide) (by evm_ov)
  have r23 := r22.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 8139)) r23 (by evm_kdecide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_4656_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hperm : ee.perm = true)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 8139) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4656) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 8139) (l1cdm_block_4656_stack (x0 := x0) (R := R)) mem aw' rdata (sstoreAccountMap ee.codeOwner σ (UInt256.ofNat 0) (UInt256.lor (UInt256.ofNat 374144419156711147060143317175368453031918731001856) (UInt256.lor (UInt256.land (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 0) (⟨0⟩ : UInt256))) (UInt256.ofNat 115792089237316195423570889229178105372547240187155051977849890856373925707775)) (UInt256.land (UInt256.mul (UInt256.ofNat 1461501637330902918203684832716283019655932542976) (UInt256.land x0 (UInt256.ofNat 255))) (UInt256.ofNat 115792089237316195423570889601861022891927484329094684320502060868636724166655))))) k' C' := by
  obtain ⟨k0, C0, h0⟩ := l1cdm_block_4656 hstack hperm hvalid h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_4792`. -/
def l1cdm_block_4792_stack {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 376793390874373408599387495934666716005045108743) :: (UInt256.ofNat 4906) :: x0 :: x1 :: x2 :: R)

/-- Automatically generated RD summary for bytecode block at pc 4792. -/
theorem l1cdm_block_4792 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hperm : ee.perm = true)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 8270) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4792) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 8270) (l1cdm_block_4792_stack (x0 := x0) (x1 := x1) (x2 := x2) (R := R)) mem aw rdata (sstoreAccountMap ee.codeOwner (sstoreAccountMap ee.codeOwner σ (UInt256.ofNat 254) (UInt256.lor (UInt256.land (UInt256.ofNat 115792089237316195423570985007226406215939081747436879206741300988257197096960) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 254) (⟨0⟩ : UInt256)))) (UInt256.land x2 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)))) (UInt256.ofNat 252) (UInt256.lor (UInt256.land (UInt256.ofNat 115792089237316195423570985007226406215939081747436879206741300988257197096960) ((sstoreAccountMap ee.codeOwner σ (UInt256.ofNat 254) (UInt256.lor (UInt256.land (UInt256.ofNat 115792089237316195423570985007226406215939081747436879206741300988257197096960) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 254) (⟨0⟩ : UInt256)))) (UInt256.land x2 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)))).get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 252) (⟨0⟩ : UInt256)))) (UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)))) k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 254) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r4⟩ := RD.sload r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup1 (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup7 (by evm_kdecide) (by evm_ov)
  have r8 := r7.and (by evm_kdecide) (by evm_ov)
  have r9 := r8.pushConst (UInt256.ofNat 115792089237316195423570985007226406215939081747436879206741300988257197096960) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r10 := r9.swap3 (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup4 (by evm_kdecide) (by evm_ov)
  have r12 := r11.and (by evm_kdecide) (by evm_ov)
  have r13 := r12.or (by evm_kdecide) (by evm_ov)
  have r14 := r13.swap1 (by evm_kdecide) (by evm_ov)
  have r15 := r14.swap3 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r16⟩ := RD.sstore r15 hperm (by evm_kdecide) (by evm_ov)
  have r17 := r16.push1 (UInt256.ofNat 252) (by evm_kdecide) (by evm_ov)
  have r18 := r17.dup1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r19⟩ := RD.sload r18 (by evm_kdecide) (by evm_ov)
  have r20 := r19.swap3 (by evm_kdecide) (by evm_ov)
  have r21 := r20.dup6 (by evm_kdecide) (by evm_ov)
  have r22 := r21.and (by evm_kdecide) (by evm_ov)
  have r23 := r22.swap3 (by evm_kdecide) (by evm_ov)
  have r24 := r23.swap1 (by evm_kdecide) (by evm_ov)
  have r25 := r24.swap2 (by evm_kdecide) (by evm_ov)
  have r26 := r25.and (by evm_kdecide) (by evm_ov)
  have r27 := r26.swap2 (by evm_kdecide) (by evm_ov)
  have r28 := r27.swap1 (by evm_kdecide) (by evm_ov)
  have r29 := r28.swap2 (by evm_kdecide) (by evm_ov)
  have r30 := r29.or (by evm_kdecide) (by evm_ov)
  have r31 := r30.swap1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r32⟩ := RD.sstore r31 hperm (by evm_kdecide) (by evm_ov)
  have r33 := r32.push2 (UInt256.ofNat 4906) (by evm_kdecide) (by evm_ov)
  have r34 := r33.push20 (UInt256.ofNat 376793390874373408599387495934666716005045108743) (by evm_kdecide) (by evm_ov)
  have r35 := r34.push2 (UInt256.ofNat 8270) (by evm_kdecide) (by evm_ov)
  have r36 := r35.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 8270)) r36 (by evm_kdecide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_4792_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hperm : ee.perm = true)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 8270) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4792) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 8270) (l1cdm_block_4792_stack (x0 := x0) (x1 := x1) (x2 := x2) (R := R)) mem aw' rdata (sstoreAccountMap ee.codeOwner (sstoreAccountMap ee.codeOwner σ (UInt256.ofNat 254) (UInt256.lor (UInt256.land (UInt256.ofNat 115792089237316195423570985007226406215939081747436879206741300988257197096960) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 254) (⟨0⟩ : UInt256)))) (UInt256.land x2 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)))) (UInt256.ofNat 252) (UInt256.lor (UInt256.land (UInt256.ofNat 115792089237316195423570985007226406215939081747436879206741300988257197096960) ((sstoreAccountMap ee.codeOwner σ (UInt256.ofNat 254) (UInt256.lor (UInt256.land (UInt256.ofNat 115792089237316195423570985007226406215939081747436879206741300988257197096960) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 254) (⟨0⟩ : UInt256)))) (UInt256.land x2 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)))).get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 252) (⟨0⟩ : UInt256)))) (UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)))) k' C' := by
  obtain ⟨k0, C0, h0⟩ := l1cdm_block_4792 hstack hperm hvalid h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_4906`. -/
def l1cdm_block_4906_stack {R : List UInt256} : List UInt256 :=
  R

/-- Final memory for bytecode block summary `l1cdm_block_4906`. -/
def l1cdm_block_4906_memory {mem : ByteArray} {x0 : UInt256} : ByteArray :=
  ((UInt256.land x0 (UInt256.ofNat 255)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 4906. -/
theorem l1cdm_block_4906 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hperm : ee.perm = true)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x3 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4906) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 x3 (l1cdm_block_4906_stack (R := R)) (l1cdm_block_4906_memory (mem := mem) (x0 := x0)) (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) ((UInt256.land x0 (UInt256.ofNat 255)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)) (UInt256.sub ((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (UInt256.ofNat 64) ((UInt256.land x0 (UInt256.ofNat 255)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)))) rdata (sstoreAccountMap ee.codeOwner σ (UInt256.ofNat 0) (UInt256.land (UInt256.ofNat 115792089237316195423570889601861022891927484329094684320502060868636724166655) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 0) (⟨0⟩ : UInt256))))) k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r4⟩ := RD.sload r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.pushConst (UInt256.ofNat 115792089237316195423570889601861022891927484329094684320502060868636724166655) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r6 := r5.and (by evm_kdecide) (by evm_ov)
  have r7 := r6.swap1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r8⟩ := RD.sstore r7 hperm (by evm_kdecide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r10 := RD.genMload r9 (by evm_kdecide) (by evm_ov)
  have r11 := r10.push1 (UInt256.ofNat 255) (by evm_kdecide) (by evm_ov)
  have r12 := r11.dup3 (by evm_kdecide) (by evm_ov)
  have r13 := r12.and (by evm_kdecide) (by evm_ov)
  have r14 := r13.dup2 (by evm_kdecide) (by evm_ov)
  have r15 := RD.genMstore r14 (by evm_kdecide) (by evm_ov)
  have r16 := r15.pushConst (UInt256.ofNat 57512143604608921510564439283751233207941214245504845198923540334447261918360) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r17 := r16.swap1 (by evm_kdecide) (by evm_ov)
  have r18 := r17.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r19 := r18.add (by evm_kdecide) (by evm_ov)
  have r20 := r19.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r21 := RD.genMload r20 (by evm_kdecide) (by evm_ov)
  have r22 := r21.dup1 (by evm_kdecide) (by evm_ov)
  have r23 := r22.swap2 (by evm_kdecide) (by evm_ov)
  have r24 := r23.sub (by evm_kdecide) (by evm_ov)
  have r25 := r24.swap1 (by evm_kdecide) (by evm_ov)
  have r26 := RD.genLog1 r25 (by evm_kdecide) hperm (by evm_ov)
  have r27 := r26.pop (by evm_kdecide) (by evm_ov)
  have r28 := r27.pop (by evm_kdecide) (by evm_ov)
  have r29 := r28.pop (by evm_kdecide) (by evm_ov)
  have r30 := r29.jump (by evm_kdecide) hvalid (by evm_ov)
  exact ⟨_, _, r30⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_4906_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hperm : ee.perm = true)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x3 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4906) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 x3 (l1cdm_block_4906_stack (R := R)) (l1cdm_block_4906_memory (mem := mem) (x0 := x0)) aw' rdata (sstoreAccountMap ee.codeOwner σ (UInt256.ofNat 0) (UInt256.land (UInt256.ofNat 115792089237316195423570889601861022891927484329094684320502060868636724166655) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 0) (⟨0⟩ : UInt256))))) k' C' := by
  obtain ⟨k0, C0, h0⟩ := l1cdm_block_4906 hstack hperm hvalid h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_5005`. -/
def l1cdm_block_5005_stack {ee : ExecutionEnv} {mem : ByteArray} {σ : AccountMap} {R : List UInt256} {gasWord0 : UInt256} : List UInt256 :=
  (gasWord0 :: (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 254) (⟨0⟩ : UInt256)))) :: (memLoad (UInt256.ofNat 64) ((UInt256.ofNat 41880202175123281672023411390868823785620507377596298514233450382794225090560).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)) :: ((UInt256.sub (memLoad (UInt256.ofNat 64) mem) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat 41880202175123281672023411390868823785620507377596298514233450382794225090560).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32))) + (UInt256.ofNat 4)) :: (memLoad (UInt256.ofNat 64) ((UInt256.ofNat 41880202175123281672023411390868823785620507377596298514233450382794225090560).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)) :: (UInt256.ofNat 32) :: ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)) :: (UInt256.ofNat 1553423035) :: (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 254) (⟨0⟩ : UInt256)))) :: (UInt256.ofNat 0) :: R)

/-- Final memory for bytecode block summary `l1cdm_block_5005`. -/
def l1cdm_block_5005_memory {mem : ByteArray} : ByteArray :=
  ((UInt256.ofNat 41880202175123281672023411390868823785620507377596298514233450382794225090560).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 5005. -/
theorem l1cdm_block_5005 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 5005) R mem aw rdata σ k C)
    : ∃ (gasWord0 : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 5100) (l1cdm_block_5005_stack (ee := ee) (mem := mem) (σ := σ) (R := R) (gasWord0 := gasWord0)) (l1cdm_block_5005_memory (mem := mem)) (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 254) (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r3⟩ := RD.sload r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := RD.genMload r5 (by evm_kdecide) (by evm_ov)
  have r7 := r6.pushConst (UInt256.ofNat 41880202175123281672023411390868823785620507377596298514233450382794225090560) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup2 (by evm_kdecide) (by evm_ov)
  have r9 := RD.genMstore r8 (by evm_kdecide) (by evm_ov)
  have r10 := r9.swap1 (by evm_kdecide) (by evm_ov)
  have r11 := RD.genMload r10 (by evm_kdecide) (by evm_ov)
  have r12 := r11.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r13 := r12.swap3 (by evm_kdecide) (by evm_ov)
  have r14 := r13.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r15 := r14.and (by evm_kdecide) (by evm_ov)
  have r16 := r15.swap2 (by evm_kdecide) (by evm_ov)
  have r17 := r16.push4 (UInt256.ofNat 1553423035) (by evm_kdecide) (by evm_ov)
  have r18 := r17.swap2 (by evm_kdecide) (by evm_ov)
  have r19 := r18.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r20 := r19.dup1 (by evm_kdecide) (by evm_ov)
  have r21 := r20.dup4 (by evm_kdecide) (by evm_ov)
  have r22 := r21.add (by evm_kdecide) (by evm_ov)
  have r23 := r22.swap3 (by evm_kdecide) (by evm_ov)
  have r24 := r23.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r25 := r24.swap3 (by evm_kdecide) (by evm_ov)
  have r26 := r25.swap2 (by evm_kdecide) (by evm_ov)
  have r27 := r26.swap1 (by evm_kdecide) (by evm_ov)
  have r28 := r27.dup3 (by evm_kdecide) (by evm_ov)
  have r29 := r28.swap1 (by evm_kdecide) (by evm_ov)
  have r30 := r29.sub (by evm_kdecide) (by evm_ov)
  have r31 := r30.add (by evm_kdecide) (by evm_ov)
  have r32 := r31.dup2 (by evm_kdecide) (by evm_ov)
  have r33 := r32.dup7 (by evm_kdecide) (by evm_ov)
  have r34 := RD.genGas r33 (by evm_kdecide) (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 5100)) r34 (by evm_kdecide)
  exact ⟨_, _, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_5005_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 5005) R mem aw rdata σ k C)
    : ∃ (gasWord0 : UInt256) (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 5100) (l1cdm_block_5005_stack (ee := ee) (mem := mem) (σ := σ) (R := R) (gasWord0 := gasWord0)) (l1cdm_block_5005_memory (mem := mem)) aw' rdata σ k' C' := by
  obtain ⟨gasWord0, k0, C0, h0⟩ := l1cdm_block_5005 hstack h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨gasWord0, _, k', C', h'⟩

/- Unsupported instruction boundary at pc 5100: staticcall (0xfa). No RD transition is asserted. Summaries resume at pc 5101 from a fresh symbolic RD state. -/

/-- Final stack for bytecode block summary `l1cdm_block_5101_taken`. -/
def l1cdm_block_5101_taken_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 5101. -/
theorem l1cdm_block_5101_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 5117) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 5101) (x0 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 5117) (l1cdm_block_5101_taken_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.iszero (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.iszero (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 5117) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 5117)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_5101_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 5117) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 5101) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 5117) (l1cdm_block_5101_taken_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_5101_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

end l1cdmBlocks
