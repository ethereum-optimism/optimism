import Reasoning.Reach
import BridgeEvm.Bytecode

open Solm ABI Ethereum Ethereum.EVM
open Reasoning.Theory Reasoning.Reach

namespace ethbridgeBlocks

/-- Automatically generated RD summary for bytecode block at pc 2123. -/
theorem ethbridge_block_2123 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2123) R mem aw rdata σ k C)
    : RDrev BridgeEvm.ethbridgeRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `ethbridge_block_2127`. -/
def ethbridge_block_2127_stack {g : Sat256} {C : ℕ} {R : List UInt256} : List UInt256 :=
  (((g.subNat (C + ((3)) + 2)).toUInt256) :: R)

/-- Automatically generated RD summary for bytecode block at pc 2127. -/
theorem ethbridge_block_2127 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 1 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2127) (x0 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2130) (ethbridge_block_2127_stack (g := g) (C := C) (R := R)) mem aw rdata σ (k + 3) (C + ((5))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := RD.genGas (RD.normalizeCounters (k' := k + 2) (C' := C + ((3))) r2 (by omega) (by omega)) (by evm_kdecide) (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2130)) r3 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2127_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 1 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2127) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2130) (ethbridge_block_2127_stack (g := g) (C := C) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2127 hstack h)
  exact ⟨_, k', C', h'⟩

/- Unsupported instruction boundary at pc 2130: call (0xf1). No RD transition is asserted. Summaries resume at pc 2131 from a fresh symbolic RD state. -/

/-- Final stack for bytecode block summary `ethbridge_block_2131_taken`. -/
def ethbridge_block_2131_taken_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 2131. -/
theorem ethbridge_block_2131_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2147) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2131) (x0 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2147) (ethbridge_block_2131_taken_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.iszero (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.iszero (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 2147) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2147)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2131_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2147) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2131) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2147) (ethbridge_block_2131_taken_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2131_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_2131_fallthrough`. -/
def ethbridge_block_2131_fallthrough_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 2131. -/
theorem ethbridge_block_2131_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2131) (x0 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2138) (ethbridge_block_2131_fallthrough_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.iszero (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.iszero (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 2147) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2138)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2131_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2131) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2138) (ethbridge_block_2131_fallthrough_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2131_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 2138. -/
theorem ethbridge_block_2138 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2138) R mem aw rdata σ k C)
    : RDrev BridgeEvm.ethbridgeRuntime g s0 := by
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

/-- Final stack for bytecode block summary `ethbridge_block_2147`. -/
def ethbridge_block_2147_stack {mem : ByteArray} {x4 : UInt256} {x5 : UInt256} {x6 : UInt256} {x7 : UInt256} {R : List UInt256} : List UInt256 :=
  ((memLoad (UInt256.ofNat 64) mem) :: (UInt256.ofNat 2165) :: x7 :: x5 :: x4 :: x5 :: x6 :: x7 :: R)

/-- Automatically generated RD summary for bytecode block at pc 2147. -/
theorem ethbridge_block_2147 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2377) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2147) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2377) (ethbridge_block_2147_stack (mem := mem) (x4 := x4) (x5 := x5) (x6 := x6) (x7 := x7) (R := R)) mem (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 13) (C + ((38) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := r2.pop (by evm_kdecide) (by evm_ov)
  have r4 := r3.pop (by evm_kdecide) (by evm_ov)
  have r5 := r4.pop (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup2 (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup5 (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r9 := RD.genMload r8 (by evm_kdecide) (by evm_ov)
  have r10 := r9.push2 (UInt256.ofNat 2165) (by evm_kdecide) (by evm_ov)
  have r11 := r10.swap1 (by evm_kdecide) (by evm_ov)
  have r12 := r11.push2 (UInt256.ofNat 2377) (by evm_kdecide) (by evm_ov)
  have r13 := r12.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2377)) r13 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2147_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2377) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2147) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2377) (ethbridge_block_2147_stack (mem := mem) (x4 := x4) (x5 := x5) (x6 := x6) (x7 := x7) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2147 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_2165`. -/
def ethbridge_block_2165_stack {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {R : List UInt256} : List UInt256 :=
  (x2 :: (memLoad (UInt256.ofNat 64) ((UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 mem x0.toNat 32)) :: (UInt256.sub ((UInt256.ofNat 32) + x0) (memLoad (UInt256.ofNat 64) ((UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 mem x0.toNat 32))) :: x2 :: R)

/-- Final memory for bytecode block summary `ethbridge_block_2165`. -/
def ethbridge_block_2165_memory {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} : ByteArray :=
  ((UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 mem x0.toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 2165. -/
theorem ethbridge_block_2165 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2165) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2203) (ethbridge_block_2165_stack (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2) (R := R)) (ethbridge_block_2165_memory (mem := mem) (x0 := x0) (x1 := x1)) (M (M aw x0 (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 16) (C + ((46) + (memExpansionCost aw x0 (⟨32⟩ : UInt256)) + (memExpansionCost (M aw x0 (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r3 := r2.swap1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.swap2 (by evm_kdecide) (by evm_ov)
  have r5 := r4.and (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup2 (by evm_kdecide) (by evm_ov)
  have r7 := RD.genMstore r6 (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r9 := r8.add (by evm_kdecide) (by evm_ov)
  have r10 := r9.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r11 := RD.genMload r10 (by evm_kdecide) (by evm_ov)
  have r12 := r11.dup1 (by evm_kdecide) (by evm_ov)
  have r13 := r12.swap2 (by evm_kdecide) (by evm_ov)
  have r14 := r13.sub (by evm_kdecide) (by evm_ov)
  have r15 := r14.swap1 (by evm_kdecide) (by evm_ov)
  have r16 := r15.dup3 (by evm_kdecide) (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2203)) r16 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2165_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2165) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2203) (ethbridge_block_2165_stack (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2) (R := R)) (ethbridge_block_2165_memory (mem := mem) (x0 := x0) (x1 := x1)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2165 hstack h)
  exact ⟨_, k', C', h'⟩

/- Unsupported instruction boundary at pc 2203: unsupported_f0 (0xf0). No RD transition is asserted. Summaries resume at pc 2204 from a fresh symbolic RD state. -/

/-- Final stack for bytecode block summary `ethbridge_block_2204_taken`. -/
def ethbridge_block_2204_taken_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: x0 :: R)

/-- Automatically generated RD summary for bytecode block at pc 2204. -/
theorem ethbridge_block_2204_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2223) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2204) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2223) (ethbridge_block_2204_taken_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 8) (C + ((30))) := by
  let r0 := h
  have r1 := r0.swap1 (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.iszero (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.iszero (by evm_kdecide) (by evm_ov)
  have r7 := r6.push2 (UInt256.ofNat 2223) (by evm_kdecide) (by evm_ov)
  have r8 := r7.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2223)) r8 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2204_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2223) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2204) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2223) (ethbridge_block_2204_taken_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2204_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_2204_fallthrough`. -/
def ethbridge_block_2204_fallthrough_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: x0 :: R)

/-- Automatically generated RD summary for bytecode block at pc 2204. -/
theorem ethbridge_block_2204_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2204) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2214) (ethbridge_block_2204_fallthrough_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 8) (C + ((30))) := by
  let r0 := h
  have r1 := r0.swap1 (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.iszero (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.iszero (by evm_kdecide) (by evm_ov)
  have r7 := r6.push2 (UInt256.ofNat 2223) (by evm_kdecide) (by evm_ov)
  have r8 := r7.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2214)) r8 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2204_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2204) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2214) (ethbridge_block_2204_fallthrough_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2204_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 2214. -/
theorem ethbridge_block_2214 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2214) R mem aw rdata σ k C)
    : RDrev BridgeEvm.ethbridgeRuntime g s0 := by
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

/-- Final stack for bytecode block summary `ethbridge_block_2223`. -/
def ethbridge_block_2223_stack {mem : ByteArray} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {R : List UInt256} : List UInt256 :=
  (((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) :: (UInt256.ofNat 36350953339468671940935475402750112069362522110264146369827093632130034281877) :: (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) x5) :: x2 :: x2 :: x3 :: x4 :: x5 :: R)

/-- Final memory for bytecode block summary `ethbridge_block_2223`. -/
def ethbridge_block_2223_memory {mem : ByteArray} {x3 : UInt256} : ByteArray :=
  (x3.toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 2223. -/
theorem ethbridge_block_2223 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 11 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2298) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2223) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2298) (ethbridge_block_2223_stack (mem := mem) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (R := R)) (ethbridge_block_2223_memory (mem := mem) (x3 := x3)) (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) rdata σ (k + 19) (C + ((58) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := r2.pop (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup1 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup5 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r7 := r6.and (by evm_kdecide) (by evm_ov)
  have r8 := r7.pushConst (UInt256.ofNat 36350953339468671940935475402750112069362522110264146369827093632130034281877) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup5 (by evm_kdecide) (by evm_ov)
  have r10 := r9.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r11 := RD.genMload r10 (by evm_kdecide) (by evm_ov)
  have r12 := r11.push2 (UInt256.ofNat 2298) (by evm_kdecide) (by evm_ov)
  have r13 := r12.swap2 (by evm_kdecide) (by evm_ov)
  have r14 := r13.dup2 (by evm_kdecide) (by evm_ov)
  have r15 := RD.genMstore r14 (by evm_kdecide) (by evm_ov)
  have r16 := r15.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r17 := r16.add (by evm_kdecide) (by evm_ov)
  have r18 := r17.swap1 (by evm_kdecide) (by evm_ov)
  have r19 := r18.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2298)) r19 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2223_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 11 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2298) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2223) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2298) (ethbridge_block_2223_stack (mem := mem) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (R := R)) (ethbridge_block_2223_memory (mem := mem) (x3 := x3)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2223 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_2298`. -/
def ethbridge_block_2298_stack {R : List UInt256} : List UInt256 :=
  R

/-- Automatically generated RD summary for bytecode block at pc 2298. -/
theorem ethbridge_block_2298 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 x10 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hperm : ee.perm = true)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains x10 = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2298) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: x10 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 x10 (ethbridge_block_2298_stack (R := R)) mem (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem))) rdata σ (k + 15) (C + ((39) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem))) + (375 + 8 * (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem)).toNat + 3 * 375))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r3 := RD.genMload r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup1 (by evm_kdecide) (by evm_ov)
  have r5 := r4.swap2 (by evm_kdecide) (by evm_ov)
  have r6 := r5.sub (by evm_kdecide) (by evm_ov)
  have r7 := r6.swap1 (by evm_kdecide) (by evm_ov)
  have r8 := RD.genLog3 r7 (by evm_kdecide) hperm (by evm_ov)
  have r9 := r8.pop (by evm_kdecide) (by evm_ov)
  have r10 := r9.pop (by evm_kdecide) (by evm_ov)
  have r11 := r10.pop (by evm_kdecide) (by evm_ov)
  have r12 := r11.pop (by evm_kdecide) (by evm_ov)
  have r13 := r12.pop (by evm_kdecide) (by evm_ov)
  have r14 := r13.pop (by evm_kdecide) (by evm_ov)
  have r15 := r14.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r15 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2298_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 x10 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hperm : ee.perm = true)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains x10 = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2298) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: x10 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 x10 (ethbridge_block_2298_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2298 hstack hperm hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_2314`. -/
def ethbridge_block_2314_stack {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {R : List UInt256} : List UInt256 :=
  (((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) :: x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: (UInt256.ofNat 2343) :: (UInt256.ofNat 0) :: x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R)

/-- Automatically generated RD summary for bytecode block at pc 2314. -/
theorem ethbridge_block_2314 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 16 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2942) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2314) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2942) (ethbridge_block_2314_stack (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (R := R)) mem (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 22) (C + ((69) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
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
  have r13 := r12.push2 (UInt256.ofNat 2343) (by evm_kdecide) (by evm_ov)
  have r14 := r13.swap7 (by evm_kdecide) (by evm_ov)
  have r15 := r14.swap6 (by evm_kdecide) (by evm_ov)
  have r16 := r15.swap5 (by evm_kdecide) (by evm_ov)
  have r17 := r16.swap4 (by evm_kdecide) (by evm_ov)
  have r18 := r17.swap3 (by evm_kdecide) (by evm_ov)
  have r19 := r18.swap2 (by evm_kdecide) (by evm_ov)
  have r20 := r19.swap1 (by evm_kdecide) (by evm_ov)
  have r21 := r20.push2 (UInt256.ofNat 2942) (by evm_kdecide) (by evm_ov)
  have r22 := r21.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2942)) r22 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2314_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 16 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2942) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2314) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2942) (ethbridge_block_2314_stack (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2314 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_2343`. -/
def ethbridge_block_2343_stack {mem : ByteArray} {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((keccakWord ((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (memLoad (UInt256.ofNat 64) mem) (x0.toByteArray.write 0 ((UInt256.sub (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem)) (UInt256.ofNat 32)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) (UInt256.ofNat 64).toNat 32)) (x0.toByteArray.write 0 ((UInt256.sub (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem)) (UInt256.ofNat 32)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) (UInt256.ofNat 64).toNat 32)) :: R)

/-- Final memory for bytecode block summary `ethbridge_block_2343`. -/
def ethbridge_block_2343_memory {mem : ByteArray} {x0 : UInt256} : ByteArray :=
  (x0.toByteArray.write 0 ((UInt256.sub (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem)) (UInt256.ofNat 32)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) (UInt256.ofNat 64).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 2343. -/
theorem ethbridge_block_2343 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains x8 = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2343) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 x8 (ethbridge_block_2343_stack (mem := mem) (x0 := x0) (R := R)) (ethbridge_block_2343_memory (mem := mem) (x0 := x0)) (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (memLoad (UInt256.ofNat 64) mem) (x0.toByteArray.write 0 ((UInt256.sub (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem)) (UInt256.ofNat 32)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) (UInt256.ofNat 64).toNat 32))) rdata σ (k + 30) (C + ((83) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (memLoad (UInt256.ofNat 64) mem) (x0.toByteArray.write 0 ((UInt256.sub (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem)) (UInt256.ofNat 32)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) (UInt256.ofNat 64).toNat 32))) + (30 + 6 * (((memLoad (memLoad (UInt256.ofNat 64) mem) (x0.toByteArray.write 0 ((UInt256.sub (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem)) (UInt256.ofNat 32)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) (UInt256.ofNat 64).toNat 32)).toNat + 31) / 32)))) := by
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
theorem ethbridge_block_2343_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains x8 = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2343) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 x8 (ethbridge_block_2343_stack (mem := mem) (x0 := x0) (R := R)) (ethbridge_block_2343_memory (mem := mem) (x0 := x0)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2343 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_2377`. -/
def ethbridge_block_2377_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  (((UInt256.ofNat 89) + x0) :: R)

/-- Final memory for bytecode block summary `ethbridge_block_2377`. -/
def ethbridge_block_2377_memory {mem : ByteArray} {x0 : UInt256} : ByteArray :=
  (BridgeEvm.ethbridgeRuntime.write (UInt256.ofNat 3030).toNat mem x0.toNat (UInt256.ofNat 89).toNat)

/-- Automatically generated RD summary for bytecode block at pc 2377. -/
theorem ethbridge_block_2377 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains x1 = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2377) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 x1 (ethbridge_block_2377_stack (x0 := x0) (R := R)) (ethbridge_block_2377_memory (mem := mem) (x0 := x0)) (M aw x0 (UInt256.ofNat 89)) rdata σ (k + 9) (C + ((27) + (memExpansionCost aw x0 (UInt256.ofNat 89)) + (3 + 3 * (((UInt256.ofNat 89).toNat + 31) / 32)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 89) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 3030) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup4 (by evm_kdecide) (by evm_ov)
  have r6 := RD.genCodecopy r5 (by evm_kdecide) (by evm_ov)
  have r7 := r6.add (by evm_kdecide) (by evm_ov)
  have r8 := r7.swap1 (by evm_kdecide) (by evm_ov)
  have r9 := r8.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r9 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2377_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains x1 = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2377) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 x1 (ethbridge_block_2377_stack (x0 := x0) (R := R)) (ethbridge_block_2377_memory (mem := mem) (x0 := x0)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2377 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 2389. -/
theorem ethbridge_block_2389_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq x0 (UInt256.land x0 (UInt256.ofNat 1461501637330902918203684832716283019655932542975))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2423) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2389) (x0 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2423) (x0 :: R) mem aw rdata σ (k + 8) (C + ((29))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.and (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup2 (by evm_kdecide) (by evm_ov)
  have r6 := r5.eq (by evm_kdecide) (by evm_ov)
  have r7 := r6.push2 (UInt256.ofNat 2423) (by evm_kdecide) (by evm_ov)
  have r8 := r7.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2423)) r8 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2389_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq x0 (UInt256.land x0 (UInt256.ofNat 1461501637330902918203684832716283019655932542975))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2423) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2389) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2423) (x0 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2389_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 2389. -/
theorem ethbridge_block_2389_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq x0 (UInt256.land x0 (UInt256.ofNat 1461501637330902918203684832716283019655932542975))) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2389) (x0 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2419) (x0 :: R) mem aw rdata σ (k + 8) (C + ((29))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.and (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup2 (by evm_kdecide) (by evm_ov)
  have r6 := r5.eq (by evm_kdecide) (by evm_ov)
  have r7 := r6.push2 (UInt256.ofNat 2423) (by evm_kdecide) (by evm_ov)
  have r8 := r7.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2419)) r8 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2389_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.eq x0 (UInt256.land x0 (UInt256.ofNat 1461501637330902918203684832716283019655932542975))) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2389) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2419) (x0 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2389_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 2419. -/
theorem ethbridge_block_2419 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2419) R mem aw rdata σ k C)
    : RDrev BridgeEvm.ethbridgeRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `ethbridge_block_2423`. -/
def ethbridge_block_2423_stack {R : List UInt256} : List UInt256 :=
  R

/-- Automatically generated RD summary for bytecode block at pc 2423. -/
theorem ethbridge_block_2423 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains x1 = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2423) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 x1 (ethbridge_block_2423_stack (R := R)) mem aw rdata σ (k + 3) (C + ((11))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := r2.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r3 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2423_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains x1 = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2423) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 x1 (ethbridge_block_2423_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2423 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_2426_taken`. -/
def ethbridge_block_2426_taken_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 2426. -/
theorem ethbridge_block_2426_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 96))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2447) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2426) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2447) (ethbridge_block_2426_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 12) (C + ((41))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r5 := r4.push1 (UInt256.ofNat 96) (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup5 (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup7 (by evm_kdecide) (by evm_ov)
  have r8 := r7.sub (by evm_kdecide) (by evm_ov)
  have r9 := r8.slt (by evm_kdecide) (by evm_ov)
  have r10 := r9.iszero (by evm_kdecide) (by evm_ov)
  have r11 := r10.push2 (UInt256.ofNat 2447) (by evm_kdecide) (by evm_ov)
  have r12 := r11.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2447)) r12 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2426_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 96))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2447) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2426) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2447) (ethbridge_block_2426_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2426_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

end ethbridgeBlocks
