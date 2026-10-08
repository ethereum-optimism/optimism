import Reasoning.Reach
import BridgeEvm.Bytecode

open Solm ABI Ethereum Ethereum.EVM
open Reasoning.Theory Reasoning.Reach

namespace ethbridgeBlocks

/-- Final stack for bytecode block summary `ethbridge_block_769`. -/
def ethbridge_block_769_stack {mem : ByteArray} {x4 : UInt256} {x5 : UInt256} {x6 : UInt256} {x7 : UInt256} {R : List UInt256} : List UInt256 :=
  ((memLoad (UInt256.ofNat 64) mem) :: (UInt256.ofNat 787) :: x7 :: x6 :: x4 :: x5 :: x6 :: x7 :: R)

/-- Automatically generated RD summary for bytecode block at pc 769. -/
theorem ethbridge_block_769 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2377) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 769) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2377) (ethbridge_block_769_stack (mem := mem) (x4 := x4) (x5 := x5) (x6 := x6) (x7 := x7) (R := R)) mem (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 13) (C + ((38) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := r2.pop (by evm_kdecide) (by evm_ov)
  have r4 := r3.pop (by evm_kdecide) (by evm_ov)
  have r5 := r4.pop (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup3 (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup5 (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r9 := RD.genMload r8 (by evm_kdecide) (by evm_ov)
  have r10 := r9.push2 (UInt256.ofNat 787) (by evm_kdecide) (by evm_ov)
  have r11 := r10.swap1 (by evm_kdecide) (by evm_ov)
  have r12 := r11.push2 (UInt256.ofNat 2377) (by evm_kdecide) (by evm_ov)
  have r13 := r12.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2377)) r13 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_769_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2377) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 769) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2377) (ethbridge_block_769_stack (mem := mem) (x4 := x4) (x5 := x5) (x6 := x6) (x7 := x7) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_769 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_787`. -/
def ethbridge_block_787_stack {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {R : List UInt256} : List UInt256 :=
  (x2 :: (memLoad (UInt256.ofNat 64) ((UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 mem x0.toNat 32)) :: (UInt256.sub ((UInt256.ofNat 32) + x0) (memLoad (UInt256.ofNat 64) ((UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 mem x0.toNat 32))) :: x2 :: R)

/-- Final memory for bytecode block summary `ethbridge_block_787`. -/
def ethbridge_block_787_memory {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} : ByteArray :=
  ((UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 mem x0.toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 787. -/
theorem ethbridge_block_787 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 787) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 825) (ethbridge_block_787_stack (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2) (R := R)) (ethbridge_block_787_memory (mem := mem) (x0 := x0) (x1 := x1)) (M (M aw x0 (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 16) (C + ((46) + (memExpansionCost aw x0 (⟨32⟩ : UInt256)) + (memExpansionCost (M aw x0 (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
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
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 825)) r16 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_787_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 787) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 825) (ethbridge_block_787_stack (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2) (R := R)) (ethbridge_block_787_memory (mem := mem) (x0 := x0) (x1 := x1)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_787 hstack h)
  exact ⟨_, k', C', h'⟩

/- Unsupported instruction boundary at pc 825: unsupported_f0 (0xf0). No RD transition is asserted. Summaries resume at pc 826 from a fresh symbolic RD state. -/

/-- Final stack for bytecode block summary `ethbridge_block_826_taken`. -/
def ethbridge_block_826_taken_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: x0 :: R)

/-- Automatically generated RD summary for bytecode block at pc 826. -/
theorem ethbridge_block_826_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 845) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 826) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 845) (ethbridge_block_826_taken_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 8) (C + ((30))) := by
  let r0 := h
  have r1 := r0.swap1 (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.iszero (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.iszero (by evm_kdecide) (by evm_ov)
  have r7 := r6.push2 (UInt256.ofNat 845) (by evm_kdecide) (by evm_ov)
  have r8 := r7.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 845)) r8 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_826_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 845) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 826) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 845) (ethbridge_block_826_taken_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_826_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_826_fallthrough`. -/
def ethbridge_block_826_fallthrough_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: x0 :: R)

/-- Automatically generated RD summary for bytecode block at pc 826. -/
theorem ethbridge_block_826_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 826) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 836) (ethbridge_block_826_fallthrough_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 8) (C + ((30))) := by
  let r0 := h
  have r1 := r0.swap1 (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.iszero (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.iszero (by evm_kdecide) (by evm_ov)
  have r7 := r6.push2 (UInt256.ofNat 845) (by evm_kdecide) (by evm_ov)
  have r8 := r7.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 836)) r8 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_826_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 826) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 836) (ethbridge_block_826_fallthrough_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_826_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 836. -/
theorem ethbridge_block_836 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 836) R mem aw rdata σ k C)
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

/-- Final stack for bytecode block summary `ethbridge_block_845`. -/
def ethbridge_block_845_stack {mem : ByteArray} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {x6 : UInt256} {R : List UInt256} : List UInt256 :=
  (((UInt256.ofNat 64) + (memLoad (UInt256.ofNat 64) mem)) :: (UInt256.ofNat 103706163223300859913136768077276593731280083329096637435724948933980193084249) :: (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) x6) :: (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) x5) :: x2 :: x3 :: x4 :: x5 :: x6 :: R)

/-- Final memory for bytecode block summary `ethbridge_block_845`. -/
def ethbridge_block_845_memory {mem : ByteArray} {x2 : UInt256} {x4 : UInt256} : ByteArray :=
  (x2.toByteArray.write 0 (x4.toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 32)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 845. -/
theorem ethbridge_block_845 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 951) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 845) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 951) (ethbridge_block_845_stack (mem := mem) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (R := R)) (ethbridge_block_845_memory (mem := mem) (x2 := x2) (x4 := x4)) (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) rdata σ (k + 29) (C + ((88) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := r2.pop (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup4 (by evm_kdecide) (by evm_ov)
  have r5 := r4.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r6 := r5.and (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup6 (by evm_kdecide) (by evm_ov)
  have r8 := r7.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r9 := r8.and (by evm_kdecide) (by evm_ov)
  have r10 := r9.pushConst (UInt256.ofNat 103706163223300859913136768077276593731280083329096637435724948933980193084249) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup6 (by evm_kdecide) (by evm_ov)
  have r12 := r11.dup5 (by evm_kdecide) (by evm_ov)
  have r13 := r12.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r14 := RD.genMload r13 (by evm_kdecide) (by evm_ov)
  have r15 := r14.push2 (UInt256.ofNat 951) (by evm_kdecide) (by evm_ov)
  have r16 := r15.swap3 (by evm_kdecide) (by evm_ov)
  have r17 := r16.swap2 (by evm_kdecide) (by evm_ov)
  have r18 := r17.swap1 (by evm_kdecide) (by evm_ov)
  have r19 := r18.swap2 (by evm_kdecide) (by evm_ov)
  have r20 := r19.dup3 (by evm_kdecide) (by evm_ov)
  have r21 := RD.genMstore r20 (by evm_kdecide) (by evm_ov)
  have r22 := r21.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r23 := r22.dup3 (by evm_kdecide) (by evm_ov)
  have r24 := r23.add (by evm_kdecide) (by evm_ov)
  have r25 := RD.genMstore r24 (by evm_kdecide) (by evm_ov)
  have r26 := r25.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r27 := r26.add (by evm_kdecide) (by evm_ov)
  have r28 := r27.swap1 (by evm_kdecide) (by evm_ov)
  have r29 := r28.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 951)) r29 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_845_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 951) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 845) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 951) (ethbridge_block_845_stack (mem := mem) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (R := R)) (ethbridge_block_845_memory (mem := mem) (x2 := x2) (x4 := x4)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_845 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_951`. -/
def ethbridge_block_951_stack {R : List UInt256} : List UInt256 :=
  R

/-- Automatically generated RD summary for bytecode block at pc 951. -/
theorem ethbridge_block_951 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hperm : ee.perm = true)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains x9 = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 951) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 x9 (ethbridge_block_951_stack (R := R)) mem (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem))) rdata σ (k + 14) (C + ((37) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem))) + (375 + 8 * (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem)).toNat + 3 * 375))) := by
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
  have r14 := r13.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r14 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_951_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hperm : ee.perm = true)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains x9 = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 951) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 x9 (ethbridge_block_951_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_951 hstack hperm hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_966_taken`. -/
def ethbridge_block_966_taken_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 966. -/
theorem ethbridge_block_966_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 1045) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 966) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1045) (ethbridge_block_966_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 7) (C + ((26))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup4 (by evm_kdecide) (by evm_ov)
  have r5 := r4.and (by evm_kdecide) (by evm_ov)
  have r6 := r5.push2 (UInt256.ofNat 1045) (by evm_kdecide) (by evm_ov)
  have r7 := r6.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1045)) r7 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_966_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 1045) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 966) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1045) (ethbridge_block_966_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_966_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_966_fallthrough`. -/
def ethbridge_block_966_fallthrough_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 966. -/
theorem ethbridge_block_966_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 966) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 996) (ethbridge_block_966_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 7) (C + ((26))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup4 (by evm_kdecide) (by evm_ov)
  have r5 := r4.and (by evm_kdecide) (by evm_ov)
  have r6 := r5.push2 (UInt256.ofNat 1045) (by evm_kdecide) (by evm_ov)
  have r7 := r6.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 996)) r7 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_966_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 966) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 996) (ethbridge_block_966_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_966_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 996. -/
theorem ethbridge_block_996 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 996) R mem aw rdata σ k C)
    : RDrev BridgeEvm.ethbridgeRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r2 := RD.genMload r1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pushConst (UInt256.ofNat 98233406313227496322093762137447883647064756579170498572149413615135132483584) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
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

/-- Final stack for bytecode block summary `ethbridge_block_1045_taken`. -/
def ethbridge_block_1045_taken_stack {ee : ExecutionEnv} {mem : ByteArray} {σ : AccountMap} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero (extCodeSizeWord σ (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (UInt256.ofNat 376793390874373408599387495934666716005045108773)))) :: (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (UInt256.ofNat 376793390874373408599387495934666716005045108773)) :: ee.weiValue :: (memLoad (UInt256.ofNat 64) ((UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 1155501680)) (UInt256.ofNat 224)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)) :: (UInt256.sub ((UInt256.ofNat 4) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (UInt256.ofNat 64) ((UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 1155501680)) (UInt256.ofNat 224)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32))) :: (memLoad (UInt256.ofNat 64) ((UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 1155501680)) (UInt256.ofNat 224)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)) :: (UInt256.ofNat 0) :: ((UInt256.ofNat 4) + (memLoad (UInt256.ofNat 64) mem)) :: ee.weiValue :: (UInt256.ofNat 1155501680) :: (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (UInt256.ofNat 376793390874373408599387495934666716005045108773)) :: R)

/-- Final memory for bytecode block summary `ethbridge_block_1045_taken`. -/
def ethbridge_block_1045_taken_memory {mem : ByteArray} : ByteArray :=
  ((UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 1155501680)) (UInt256.ofNat 224)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1045. -/
theorem ethbridge_block_1045_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero (extCodeSizeWord σ (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (UInt256.ofNat 376793390874373408599387495934666716005045108773))))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 1137) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1045) R mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1137) (ethbridge_block_1045_taken_stack (ee := ee) (mem := mem) (σ := σ) (R := R)) (ethbridge_block_1045_taken_memory (mem := mem)) (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push20 (UInt256.ofNat 376793390874373408599387495934666716005045108773) (by evm_kdecide) (by evm_ov)
  have r3 := r2.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r4 := r3.and (by evm_kdecide) (by evm_ov)
  have r5 := r4.push4 (UInt256.ofNat 1155501680) (by evm_kdecide) (by evm_ov)
  have r6 := r5.callvalue (by evm_kdecide) (by evm_ov)
  have r7 := r6.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r8 := RD.genMload r7 (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup3 (by evm_kdecide) (by evm_ov)
  have r10 := r9.push4 (UInt256.ofNat 4294967295) (by evm_kdecide) (by evm_ov)
  have r11 := r10.and (by evm_kdecide) (by evm_ov)
  have r12 := r11.push1 (UInt256.ofNat 224) (by evm_kdecide) (by evm_ov)
  have r13 := r12.shl (by evm_kdecide) (by evm_ov)
  have r14 := r13.dup2 (by evm_kdecide) (by evm_ov)
  have r15 := RD.genMstore r14 (by evm_kdecide) (by evm_ov)
  have r16 := r15.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r17 := r16.add (by evm_kdecide) (by evm_ov)
  have r18 := r17.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r19 := r18.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r20 := RD.genMload r19 (by evm_kdecide) (by evm_ov)
  have r21 := r20.dup1 (by evm_kdecide) (by evm_ov)
  have r22 := r21.dup4 (by evm_kdecide) (by evm_ov)
  have r23 := r22.sub (by evm_kdecide) (by evm_ov)
  have r24 := r23.dup2 (by evm_kdecide) (by evm_ov)
  have r25 := r24.dup6 (by evm_kdecide) (by evm_ov)
  have r26 := r25.dup9 (by evm_kdecide) (by evm_ov)
  have r27 := r26.dup1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r28⟩ := r27.extcodesize (by evm_kdecide) (by evm_ov)
  have r29 := r28.iszero (by evm_kdecide) (by evm_ov)
  have r30 := r29.dup1 (by evm_kdecide) (by evm_ov)
  have r31 := r30.iszero (by evm_kdecide) (by evm_ov)
  have r32 := r31.push2 (UInt256.ofNat 1137) (by evm_kdecide) (by evm_ov)
  have r33 := r32.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1137)) r33 (by evm_kdecide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_1045_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero (extCodeSizeWord σ (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (UInt256.ofNat 376793390874373408599387495934666716005045108773))))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 1137) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1045) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1137) (ethbridge_block_1045_taken_stack (ee := ee) (mem := mem) (σ := σ) (R := R)) (ethbridge_block_1045_taken_memory (mem := mem)) aw' rdata σ k' C' := by
  obtain ⟨k0, C0, h0⟩ := ethbridge_block_1045_taken hstack hcond hvalid h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_1045_fallthrough`. -/
def ethbridge_block_1045_fallthrough_stack {ee : ExecutionEnv} {mem : ByteArray} {σ : AccountMap} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero (extCodeSizeWord σ (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (UInt256.ofNat 376793390874373408599387495934666716005045108773)))) :: (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (UInt256.ofNat 376793390874373408599387495934666716005045108773)) :: ee.weiValue :: (memLoad (UInt256.ofNat 64) ((UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 1155501680)) (UInt256.ofNat 224)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)) :: (UInt256.sub ((UInt256.ofNat 4) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (UInt256.ofNat 64) ((UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 1155501680)) (UInt256.ofNat 224)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32))) :: (memLoad (UInt256.ofNat 64) ((UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 1155501680)) (UInt256.ofNat 224)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)) :: (UInt256.ofNat 0) :: ((UInt256.ofNat 4) + (memLoad (UInt256.ofNat 64) mem)) :: ee.weiValue :: (UInt256.ofNat 1155501680) :: (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (UInt256.ofNat 376793390874373408599387495934666716005045108773)) :: R)

/-- Final memory for bytecode block summary `ethbridge_block_1045_fallthrough`. -/
def ethbridge_block_1045_fallthrough_memory {mem : ByteArray} : ByteArray :=
  ((UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 1155501680)) (UInt256.ofNat 224)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1045. -/
theorem ethbridge_block_1045_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero (extCodeSizeWord σ (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (UInt256.ofNat 376793390874373408599387495934666716005045108773))))) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1045) R mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1133) (ethbridge_block_1045_fallthrough_stack (ee := ee) (mem := mem) (σ := σ) (R := R)) (ethbridge_block_1045_fallthrough_memory (mem := mem)) (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push20 (UInt256.ofNat 376793390874373408599387495934666716005045108773) (by evm_kdecide) (by evm_ov)
  have r3 := r2.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r4 := r3.and (by evm_kdecide) (by evm_ov)
  have r5 := r4.push4 (UInt256.ofNat 1155501680) (by evm_kdecide) (by evm_ov)
  have r6 := r5.callvalue (by evm_kdecide) (by evm_ov)
  have r7 := r6.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r8 := RD.genMload r7 (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup3 (by evm_kdecide) (by evm_ov)
  have r10 := r9.push4 (UInt256.ofNat 4294967295) (by evm_kdecide) (by evm_ov)
  have r11 := r10.and (by evm_kdecide) (by evm_ov)
  have r12 := r11.push1 (UInt256.ofNat 224) (by evm_kdecide) (by evm_ov)
  have r13 := r12.shl (by evm_kdecide) (by evm_ov)
  have r14 := r13.dup2 (by evm_kdecide) (by evm_ov)
  have r15 := RD.genMstore r14 (by evm_kdecide) (by evm_ov)
  have r16 := r15.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r17 := r16.add (by evm_kdecide) (by evm_ov)
  have r18 := r17.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r19 := r18.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r20 := RD.genMload r19 (by evm_kdecide) (by evm_ov)
  have r21 := r20.dup1 (by evm_kdecide) (by evm_ov)
  have r22 := r21.dup4 (by evm_kdecide) (by evm_ov)
  have r23 := r22.sub (by evm_kdecide) (by evm_ov)
  have r24 := r23.dup2 (by evm_kdecide) (by evm_ov)
  have r25 := r24.dup6 (by evm_kdecide) (by evm_ov)
  have r26 := r25.dup9 (by evm_kdecide) (by evm_ov)
  have r27 := r26.dup1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r28⟩ := r27.extcodesize (by evm_kdecide) (by evm_ov)
  have r29 := r28.iszero (by evm_kdecide) (by evm_ov)
  have r30 := r29.dup1 (by evm_kdecide) (by evm_ov)
  have r31 := r30.iszero (by evm_kdecide) (by evm_ov)
  have r32 := r31.push2 (UInt256.ofNat 1137) (by evm_kdecide) (by evm_ov)
  have r33 := r32.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1133)) r33 (by evm_kdecide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_1045_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero (extCodeSizeWord σ (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (UInt256.ofNat 376793390874373408599387495934666716005045108773))))) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1045) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1133) (ethbridge_block_1045_fallthrough_stack (ee := ee) (mem := mem) (σ := σ) (R := R)) (ethbridge_block_1045_fallthrough_memory (mem := mem)) aw' rdata σ k' C' := by
  obtain ⟨k0, C0, h0⟩ := ethbridge_block_1045_fallthrough hstack hcond h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1133. -/
theorem ethbridge_block_1133 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1133) R mem aw rdata σ k C)
    : RDrev BridgeEvm.ethbridgeRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `ethbridge_block_1137`. -/
def ethbridge_block_1137_stack {g : Sat256} {C : ℕ} {R : List UInt256} : List UInt256 :=
  (((g.subNat (C + ((3)) + 2)).toUInt256) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1137. -/
theorem ethbridge_block_1137 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 1 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1137) (x0 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1140) (ethbridge_block_1137_stack (g := g) (C := C) (R := R)) mem aw rdata σ (k + 3) (C + ((5))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := RD.genGas (RD.normalizeCounters (k' := k + 2) (C' := C + ((3))) r2 (by omega) (by omega)) (by evm_kdecide) (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1140)) r3 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_1137_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 1 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1137) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1140) (ethbridge_block_1137_stack (g := g) (C := C) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_1137 hstack h)
  exact ⟨_, k', C', h'⟩

/- Unsupported instruction boundary at pc 1140: call (0xf1). No RD transition is asserted. Summaries resume at pc 1141 from a fresh symbolic RD state. -/

/-- Final stack for bytecode block summary `ethbridge_block_1141_taken`. -/
def ethbridge_block_1141_taken_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1141. -/
theorem ethbridge_block_1141_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 1157) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1141) (x0 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1157) (ethbridge_block_1141_taken_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.iszero (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.iszero (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 1157) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1157)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_1141_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 1157) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1141) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1157) (ethbridge_block_1141_taken_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_1141_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_1141_fallthrough`. -/
def ethbridge_block_1141_fallthrough_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1141. -/
theorem ethbridge_block_1141_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1141) (x0 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1148) (ethbridge_block_1141_fallthrough_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.iszero (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.iszero (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 1157) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1148)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_1141_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1141) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1148) (ethbridge_block_1141_fallthrough_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_1141_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1148. -/
theorem ethbridge_block_1148 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1148) R mem aw rdata σ k C)
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

/-- Final stack for bytecode block summary `ethbridge_block_1157`. -/
def ethbridge_block_1157_stack {ee : ExecutionEnv} {mem : ByteArray} {x2 : UInt256} {x3 : UInt256} {x5 : UInt256} {x6 : UInt256} {x7 : UInt256} {R : List UInt256} : List UInt256 :=
  (x3 :: (memLoad (UInt256.ofNat 64) ((UInt256.lor (UInt256.ofNat 35758974700130516083418609194394918359861923750008682890162270758914934964224) (UInt256.land (UInt256.ofNat 26959946667150639794667015087019630673637144422540572481103610249215) (memLoad ((memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) + (UInt256.ofNat 32)) (((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 132)).toByteArray.write 0 (((UInt256.ofNat 100) + (UInt256.sub (memLoad (UInt256.ofNat 64) mem) (memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)))).toByteArray.write 0 (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32) (memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)).toNat 32) (UInt256.ofNat 64).toNat 32)))).toByteArray.write 0 (((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 132)).toByteArray.write 0 (((UInt256.ofNat 100) + (UInt256.sub (memLoad (UInt256.ofNat 64) mem) (memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)))).toByteArray.write 0 (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32) (memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)).toNat 32) (UInt256.ofNat 64).toNat 32) ((memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) + (UInt256.ofNat 32)).toNat 32)) :: (memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) :: x2 :: (UInt256.ofNat 1884746783) :: (UInt256.ofNat 376793390874373408599387495934666716005045108771) :: x5 :: x6 :: x7 :: R)

/-- Final memory for bytecode block summary `ethbridge_block_1157`. -/
def ethbridge_block_1157_memory {ee : ExecutionEnv} {mem : ByteArray} {x7 : UInt256} : ByteArray :=
  ((UInt256.ofNat 50812672750763740129390437241472713969985730859689736670738357706895684272128).toByteArray.write 0 ((UInt256.lor (UInt256.ofNat 35758974700130516083418609194394918359861923750008682890162270758914934964224) (UInt256.land (UInt256.ofNat 26959946667150639794667015087019630673637144422540572481103610249215) (memLoad ((memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) + (UInt256.ofNat 32)) (((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 132)).toByteArray.write 0 (((UInt256.ofNat 100) + (UInt256.sub (memLoad (UInt256.ofNat 64) mem) (memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)))).toByteArray.write 0 (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32) (memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)).toNat 32) (UInt256.ofNat 64).toNat 32)))).toByteArray.write 0 (((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 132)).toByteArray.write 0 (((UInt256.ofNat 100) + (UInt256.sub (memLoad (UInt256.ofNat 64) mem) (memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)))).toByteArray.write 0 (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32) (memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)).toNat 32) (UInt256.ofNat 64).toNat 32) ((memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) + (UInt256.ofNat 32)).toNat 32) (memLoad (UInt256.ofNat 64) ((UInt256.lor (UInt256.ofNat 35758974700130516083418609194394918359861923750008682890162270758914934964224) (UInt256.land (UInt256.ofNat 26959946667150639794667015087019630673637144422540572481103610249215) (memLoad ((memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) + (UInt256.ofNat 32)) (((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 132)).toByteArray.write 0 (((UInt256.ofNat 100) + (UInt256.sub (memLoad (UInt256.ofNat 64) mem) (memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)))).toByteArray.write 0 (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32) (memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)).toNat 32) (UInt256.ofNat 64).toNat 32)))).toByteArray.write 0 (((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 132)).toByteArray.write 0 (((UInt256.ofNat 100) + (UInt256.sub (memLoad (UInt256.ofNat 64) mem) (memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)))).toByteArray.write 0 (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32) (memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)).toNat 32) (UInt256.ofNat 64).toNat 32) ((memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) + (UInt256.ofNat 32)).toNat 32)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1157. -/
theorem ethbridge_block_1157 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1157) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1363) (ethbridge_block_1157_stack (ee := ee) (mem := mem) (x2 := x2) (x3 := x3) (x5 := x5) (x6 := x6) (x7 := x7) (R := R)) (ethbridge_block_1157_memory (ee := ee) (mem := mem) (x7 := x7)) (M (M (M (M (M (M (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) ((UInt256.lor (UInt256.ofNat 35758974700130516083418609194394918359861923750008682890162270758914934964224) (UInt256.land (UInt256.ofNat 26959946667150639794667015087019630673637144422540572481103610249215) (memLoad ((memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) + (UInt256.ofNat 32)) (((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 132)).toByteArray.write 0 (((UInt256.ofNat 100) + (UInt256.sub (memLoad (UInt256.ofNat 64) mem) (memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)))).toByteArray.write 0 (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32) (memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)).toNat 32) (UInt256.ofNat 64).toNat 32)))).toByteArray.write 0 (((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 132)).toByteArray.write 0 (((UInt256.ofNat 100) + (UInt256.sub (memLoad (UInt256.ofNat 64) mem) (memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)))).toByteArray.write 0 (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32) (memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)).toNat 32) (UInt256.ofNat 64).toNat 32) ((memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) + (UInt256.ofNat 32)).toNat 32)) (⟨32⟩ : UInt256)) rdata σ (k + 64) (C + ((185) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M (M (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) ((UInt256.lor (UInt256.ofNat 35758974700130516083418609194394918359861923750008682890162270758914934964224) (UInt256.land (UInt256.ofNat 26959946667150639794667015087019630673637144422540572481103610249215) (memLoad ((memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) + (UInt256.ofNat 32)) (((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 132)).toByteArray.write 0 (((UInt256.ofNat 100) + (UInt256.sub (memLoad (UInt256.ofNat 64) mem) (memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)))).toByteArray.write 0 (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32) (memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)).toNat 32) (UInt256.ofNat 64).toNat 32)))).toByteArray.write 0 (((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 132)).toByteArray.write 0 (((UInt256.ofNat 100) + (UInt256.sub (memLoad (UInt256.ofNat 64) mem) (memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)))).toByteArray.write 0 (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32) (memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)).toNat 32) (UInt256.ofNat 64).toNat 32) ((memLoad (UInt256.ofNat 64) (ee.weiValue.toByteArray.write 0 ((UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.ofNat ee.source.val).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) + (UInt256.ofNat 32)).toNat 32)) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := r2.pop (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := RD.genMload r5 (by evm_kdecide) (by evm_ov)
  have r7 := r6.caller (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 36) (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup3 (by evm_kdecide) (by evm_ov)
  have r10 := r9.add (by evm_kdecide) (by evm_ov)
  have r11 := RD.genMstore r10 (by evm_kdecide) (by evm_ov)
  have r12 := r11.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r13 := r12.dup9 (by evm_kdecide) (by evm_ov)
  have r14 := r13.and (by evm_kdecide) (by evm_ov)
  have r15 := r14.push1 (UInt256.ofNat 68) (by evm_kdecide) (by evm_ov)
  have r16 := r15.dup3 (by evm_kdecide) (by evm_ov)
  have r17 := r16.add (by evm_kdecide) (by evm_ov)
  have r18 := RD.genMstore r17 (by evm_kdecide) (by evm_ov)
  have r19 := r18.callvalue (by evm_kdecide) (by evm_ov)
  have r20 := r19.push1 (UInt256.ofNat 100) (by evm_kdecide) (by evm_ov)
  have r21 := r20.dup1 (by evm_kdecide) (by evm_ov)
  have r22 := r21.dup4 (by evm_kdecide) (by evm_ov)
  have r23 := r22.add (by evm_kdecide) (by evm_ov)
  have r24 := r23.swap2 (by evm_kdecide) (by evm_ov)
  have r25 := r24.swap1 (by evm_kdecide) (by evm_ov)
  have r26 := r25.swap2 (by evm_kdecide) (by evm_ov)
  have r27 := RD.genMstore r26 (by evm_kdecide) (by evm_ov)
  have r28 := r27.dup3 (by evm_kdecide) (by evm_ov)
  have r29 := RD.genMload r28 (by evm_kdecide) (by evm_ov)
  have r30 := r29.dup1 (by evm_kdecide) (by evm_ov)
  have r31 := r30.dup4 (by evm_kdecide) (by evm_ov)
  have r32 := r31.sub (by evm_kdecide) (by evm_ov)
  have r33 := r32.swap1 (by evm_kdecide) (by evm_ov)
  have r34 := r33.swap2 (by evm_kdecide) (by evm_ov)
  have r35 := r34.add (by evm_kdecide) (by evm_ov)
  have r36 := r35.dup2 (by evm_kdecide) (by evm_ov)
  have r37 := RD.genMstore r36 (by evm_kdecide) (by evm_ov)
  have r38 := r37.push1 (UInt256.ofNat 132) (by evm_kdecide) (by evm_ov)
  have r39 := r38.swap1 (by evm_kdecide) (by evm_ov)
  have r40 := r39.swap2 (by evm_kdecide) (by evm_ov)
  have r41 := r40.add (by evm_kdecide) (by evm_ov)
  have r42 := r41.dup3 (by evm_kdecide) (by evm_ov)
  have r43 := RD.genMstore r42 (by evm_kdecide) (by evm_ov)
  have r44 := r43.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r45 := r44.dup2 (by evm_kdecide) (by evm_ov)
  have r46 := r45.add (by evm_kdecide) (by evm_ov)
  have r47 := r46.dup1 (by evm_kdecide) (by evm_ov)
  have r48 := RD.genMload r47 (by evm_kdecide) (by evm_ov)
  have r49 := r48.pushConst (UInt256.ofNat 26959946667150639794667015087019630673637144422540572481103610249215) (width := 28) (op := .PUSH28) (by decide) (by evm_kdecide) (by evm_ov)
  have r50 := r49.and (by evm_kdecide) (by evm_ov)
  have r51 := r50.pushConst (UInt256.ofNat 35758974700130516083418609194394918359861923750008682890162270758914934964224) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r52 := r51.or (by evm_kdecide) (by evm_ov)
  have r53 := r52.swap1 (by evm_kdecide) (by evm_ov)
  have r54 := RD.genMstore r53 (by evm_kdecide) (by evm_ov)
  have r55 := r54.swap1 (by evm_kdecide) (by evm_ov)
  have r56 := RD.genMload r55 (by evm_kdecide) (by evm_ov)
  have r57 := r56.pushConst (UInt256.ofNat 50812672750763740129390437241472713969985730859689736670738357706895684272128) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r58 := r57.dup2 (by evm_kdecide) (by evm_ov)
  have r59 := RD.genMstore r58 (by evm_kdecide) (by evm_ov)
  have r60 := r59.push20 (UInt256.ofNat 376793390874373408599387495934666716005045108771) (by evm_kdecide) (by evm_ov)
  have r61 := r60.swap5 (by evm_kdecide) (by evm_ov)
  have r62 := r61.pop (by evm_kdecide) (by evm_ov)
  have r63 := r62.push4 (UInt256.ofNat 1884746783) (by evm_kdecide) (by evm_ov)
  have r64 := r63.swap4 (by evm_kdecide) (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1363)) r64 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_1157_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1157) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1363) (ethbridge_block_1157_stack (ee := ee) (mem := mem) (x2 := x2) (x3 := x3) (x5 := x5) (x6 := x6) (x7 := x7) (R := R)) (ethbridge_block_1157_memory (ee := ee) (mem := mem) (x7 := x7)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_1157 hstack h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_1363`. -/
def ethbridge_block_1363_stack {ee : ExecutionEnv} {x1 : UInt256} {x2 : UInt256} {x4 : UInt256} {x5 : UInt256} {x6 : UInt256} {x7 : UInt256} {R : List UInt256} : List UInt256 :=
  (((UInt256.ofNat 4) + x1) :: x2 :: (UInt256.ofNat ee.codeOwner.val) :: x7 :: (UInt256.ofNat 1380) :: x4 :: x5 :: x6 :: x7 :: R)

/-- Automatically generated RD summary for bytecode block at pc 1363. -/
theorem ethbridge_block_1363 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2821) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1363) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2821) (ethbridge_block_1363_stack (ee := ee) (x1 := x1) (x2 := x2) (x4 := x4) (x5 := x5) (x6 := x6) (x7 := x7) (R := R)) mem aw rdata σ (k + 12) (C + ((38))) := by
  let r0 := h
  have r1 := r0.pop (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 1380) (by evm_kdecide) (by evm_ov)
  have r3 := r2.swap3 (by evm_kdecide) (by evm_ov)
  have r4 := r3.pop (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup7 (by evm_kdecide) (by evm_ov)
  have r6 := r5.swap2 (by evm_kdecide) (by evm_ov)
  have r7 := r6.address (by evm_kdecide) (by evm_ov)
  have r8 := r7.swap2 (by evm_kdecide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r10 := r9.add (by evm_kdecide) (by evm_ov)
  have r11 := r10.push2 (UInt256.ofNat 2821) (by evm_kdecide) (by evm_ov)
  have r12 := r11.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2821)) r12 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_1363_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2821) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1363) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2821) (ethbridge_block_1363_stack (ee := ee) (x1 := x1) (x2 := x2) (x4 := x4) (x5 := x5) (x6 := x6) (x7 := x7) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_1363 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_1380`. -/
def ethbridge_block_1380_stack {g : Sat256} {mem : ByteArray} {aw : UInt256} {C : ℕ} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {R : List UInt256} : List UInt256 :=
  (((g.subNat (C + ((28) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256))) + 2)).toUInt256) :: x2 :: (UInt256.ofNat 0) :: (memLoad (UInt256.ofNat 64) mem) :: (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem)) :: (memLoad (UInt256.ofNat 64) mem) :: (UInt256.ofNat 32) :: x0 :: x1 :: x2 :: R)

/-- Automatically generated RD summary for bytecode block at pc 1380. -/
theorem ethbridge_block_1380 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1380) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1394) (ethbridge_block_1380_stack (g := g) (mem := mem) (aw := aw) (C := C) (x0 := x0) (x1 := x1) (x2 := x2) (R := R)) mem (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 11) (C + ((30) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r4 := RD.genMload r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup4 (by evm_kdecide) (by evm_ov)
  have r7 := r6.sub (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup2 (by evm_kdecide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup8 (by evm_kdecide) (by evm_ov)
  have r11 := RD.genGas (RD.normalizeCounters (k' := k + 10) (C' := C + ((28) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) r10 (by omega) (by omega)) (by evm_kdecide) (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1394)) r11 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_1380_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1380) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1394) (ethbridge_block_1380_stack (g := g) (mem := mem) (aw := aw) (C := C) (x0 := x0) (x1 := x1) (x2 := x2) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_1380 hstack h)
  exact ⟨_, k', C', h'⟩

/- Unsupported instruction boundary at pc 1394: call (0xf1). No RD transition is asserted. Summaries resume at pc 1395 from a fresh symbolic RD state. -/

end ethbridgeBlocks
