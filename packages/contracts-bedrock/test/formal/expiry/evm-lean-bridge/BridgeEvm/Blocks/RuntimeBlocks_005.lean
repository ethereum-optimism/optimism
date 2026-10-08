import Reasoning.Reach
import BridgeEvm.Bytecode

open Solm ABI Ethereum Ethereum.EVM
open Reasoning.Theory Reasoning.Reach

namespace ethbridgeBlocks

/-- Final stack for bytecode block summary `ethbridge_block_1395_taken`. -/
def ethbridge_block_1395_taken_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1395. -/
theorem ethbridge_block_1395_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 1411) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1395) (x0 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1411) (ethbridge_block_1395_taken_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.iszero (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.iszero (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 1411) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1411)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_1395_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 1411) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1395) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1411) (ethbridge_block_1395_taken_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_1395_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_1395_fallthrough`. -/
def ethbridge_block_1395_fallthrough_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1395. -/
theorem ethbridge_block_1395_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1395) (x0 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1402) (ethbridge_block_1395_fallthrough_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.iszero (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.iszero (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 1411) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1402)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_1395_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1395) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1402) (ethbridge_block_1395_fallthrough_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_1395_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1402. -/
theorem ethbridge_block_1402 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1402) R mem aw rdata σ k C)
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

/-- Final stack for bytecode block summary `ethbridge_block_1411`. -/
def ethbridge_block_1411_stack {mem : ByteArray} {rdata : ByteArray} {R : List UInt256} : List UInt256 :=
  ((memLoad (UInt256.ofNat 64) mem) :: ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat rdata.size)) :: (UInt256.ofNat 1447) :: R)

/-- Final memory for bytecode block summary `ethbridge_block_1411`. -/
def ethbridge_block_1411_memory {mem : ByteArray} {rdata : ByteArray} : ByteArray :=
  (((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat rdata.size) + (UInt256.ofNat 31)) (UInt256.lnot (UInt256.ofNat 31)))).toByteArray.write 0 mem (UInt256.ofNat 64).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1411. -/
theorem ethbridge_block_1411 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2883) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1411) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2883) (ethbridge_block_1411_stack (mem := mem) (rdata := rdata) (R := R)) (ethbridge_block_1411_memory (mem := mem) (rdata := rdata)) (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 28) (C + ((81) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
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
  have r24 := r23.push2 (UInt256.ofNat 1447) (by evm_kdecide) (by evm_ov)
  have r25 := r24.swap2 (by evm_kdecide) (by evm_ov)
  have r26 := r25.swap1 (by evm_kdecide) (by evm_ov)
  have r27 := r26.push2 (UInt256.ofNat 2883) (by evm_kdecide) (by evm_ov)
  have r28 := r27.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2883)) r28 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_1411_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2883) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1411) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2883) (ethbridge_block_1411_stack (mem := mem) (rdata := rdata) (R := R)) (ethbridge_block_1411_memory (mem := mem) (rdata := rdata)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_1411 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_1447`. -/
def ethbridge_block_1447_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: R)

/-- Final memory for bytecode block summary `ethbridge_block_1447`. -/
def ethbridge_block_1447_memory {ee : ExecutionEnv} {mem : ByteArray} {x2 : UInt256} : ByteArray :=
  (x2.toByteArray.write 0 (ee.weiValue.toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 32)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1447. -/
theorem ethbridge_block_1447 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (hperm : ee.perm = true)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains x4 = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1447) (x0 :: x1 :: x2 :: x3 :: x4 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 x4 (ethbridge_block_1447_stack (x0 := x0) (R := R)) (ethbridge_block_1447_memory (ee := ee) (mem := mem) (x2 := x2)) (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) (x2.toByteArray.write 0 (ee.weiValue.toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 32)).toNat 32)) (UInt256.sub ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 64)) (memLoad (UInt256.ofNat 64) (x2.toByteArray.write 0 (ee.weiValue.toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 32)).toNat 32)))) rdata σ (k + 37) (C + ((106) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) (x2.toByteArray.write 0 (ee.weiValue.toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 32)).toNat 32)) (UInt256.sub ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 64)) (memLoad (UInt256.ofNat 64) (x2.toByteArray.write 0 (ee.weiValue.toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 32)).toNat 32)))) + (375 + 8 * (UInt256.sub ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 64)) (memLoad (UInt256.ofNat 64) (x2.toByteArray.write 0 (ee.weiValue.toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 32)).toNat 32))).toNat + 3 * 375))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := RD.genMload r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.callvalue (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup2 (by evm_kdecide) (by evm_ov)
  have r7 := RD.genMstore r6 (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup2 (by evm_kdecide) (by evm_ov)
  have r10 := r9.add (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup6 (by evm_kdecide) (by evm_ov)
  have r12 := r11.swap1 (by evm_kdecide) (by evm_ov)
  have r13 := RD.genMstore r12 (by evm_kdecide) (by evm_ov)
  have r14 := r13.swap2 (by evm_kdecide) (by evm_ov)
  have r15 := r14.swap3 (by evm_kdecide) (by evm_ov)
  have r16 := r15.pop (by evm_kdecide) (by evm_ov)
  have r17 := r16.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r18 := r17.dup6 (by evm_kdecide) (by evm_ov)
  have r19 := r18.and (by evm_kdecide) (by evm_ov)
  have r20 := r19.swap2 (by evm_kdecide) (by evm_ov)
  have r21 := r20.caller (by evm_kdecide) (by evm_ov)
  have r22 := r21.swap2 (by evm_kdecide) (by evm_ov)
  have r23 := r22.pushConst (UInt256.ofNat 107467830838466476624852865317537284309165008304313857258712035819154557121538) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r24 := r23.swap2 (by evm_kdecide) (by evm_ov)
  have r25 := r24.add (by evm_kdecide) (by evm_ov)
  have r26 := r25.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r27 := RD.genMload r26 (by evm_kdecide) (by evm_ov)
  have r28 := r27.dup1 (by evm_kdecide) (by evm_ov)
  have r29 := r28.swap2 (by evm_kdecide) (by evm_ov)
  have r30 := r29.sub (by evm_kdecide) (by evm_ov)
  have r31 := r30.swap1 (by evm_kdecide) (by evm_ov)
  have r32 := RD.genLog3 r31 (by evm_kdecide) hperm (by evm_ov)
  have r33 := r32.swap3 (by evm_kdecide) (by evm_ov)
  have r34 := r33.swap2 (by evm_kdecide) (by evm_ov)
  have r35 := r34.pop (by evm_kdecide) (by evm_ov)
  have r36 := r35.pop (by evm_kdecide) (by evm_ov)
  have r37 := r36.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r37 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_1447_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (hperm : ee.perm = true)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains x4 = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1447) (x0 :: x1 :: x2 :: x3 :: x4 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 x4 (ethbridge_block_1447_stack (x0 := x0) (R := R)) (ethbridge_block_1447_memory (ee := ee) (mem := mem) (x2 := x2)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_1447 hstack hperm hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_1539`. -/
def ethbridge_block_1539_stack {ee : ExecutionEnv} {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat ee.codeOwner.val) :: (memLoad (UInt256.ofNat 64) (x0.toByteArray.write 0 ((UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) x2).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) :: x3 :: (UInt256.ofNat Ethereum.chainId) :: x4 :: (UInt256.ofNat 1699) :: (UInt256.ofNat 0) :: x0 :: x1 :: x2 :: x3 :: x4 :: R)

/-- Final memory for bytecode block summary `ethbridge_block_1539`. -/
def ethbridge_block_1539_memory {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} : ByteArray :=
  ((UInt256.lor (UInt256.ofNat 35758974700130516083418609194394918359861923750008682890162270758914934964224) (UInt256.land (UInt256.ofNat 26959946667150639794667015087019630673637144422540572481103610249215) (memLoad ((memLoad (UInt256.ofNat 64) (x0.toByteArray.write 0 ((UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) x2).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) + (UInt256.ofNat 32)) (((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 132)).toByteArray.write 0 (((UInt256.ofNat 100) + (UInt256.sub (memLoad (UInt256.ofNat 64) mem) (memLoad (UInt256.ofNat 64) (x0.toByteArray.write 0 ((UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) x2).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)))).toByteArray.write 0 (x0.toByteArray.write 0 ((UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) x2).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32) (memLoad (UInt256.ofNat 64) (x0.toByteArray.write 0 ((UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) x2).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)).toNat 32) (UInt256.ofNat 64).toNat 32)))).toByteArray.write 0 (((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 132)).toByteArray.write 0 (((UInt256.ofNat 100) + (UInt256.sub (memLoad (UInt256.ofNat 64) mem) (memLoad (UInt256.ofNat 64) (x0.toByteArray.write 0 ((UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) x2).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)))).toByteArray.write 0 (x0.toByteArray.write 0 ((UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) x2).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32) (memLoad (UInt256.ofNat 64) (x0.toByteArray.write 0 ((UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) x2).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)).toNat 32) (UInt256.ofNat 64).toNat 32) ((memLoad (UInt256.ofNat 64) (x0.toByteArray.write 0 ((UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) x2).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) + (UInt256.ofNat 32)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1539. -/
theorem ethbridge_block_1539 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1539) (x0 :: x1 :: x2 :: x3 :: x4 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1692) (ethbridge_block_1539_stack (ee := ee) (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (R := R)) (ethbridge_block_1539_memory (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2)) (M (M (M (M (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) (x0.toByteArray.write 0 ((UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) x2).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) (x0.toByteArray.write 0 ((UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) x2).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) (x0.toByteArray.write 0 ((UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) x2).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) rdata σ (k + 64) (C + ((188) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) (x0.toByteArray.write 0 ((UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) x2).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) (x0.toByteArray.write 0 ((UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) x2).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) (x0.toByteArray.write 0 ((UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) x2).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) (x0.toByteArray.write 0 ((UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) x2).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) (x0.toByteArray.write 0 ((UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) x2).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) (x0.toByteArray.write 0 ((UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) x2).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) (x0.toByteArray.write 0 ((UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) x2).toByteArray.write 0 mem ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := RD.genMload r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup6 (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup2 (by evm_kdecide) (by evm_ov)
  have r8 := r7.and (by evm_kdecide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 36) (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup4 (by evm_kdecide) (by evm_ov)
  have r11 := r10.add (by evm_kdecide) (by evm_ov)
  have r12 := RD.genMstore r11 (by evm_kdecide) (by evm_ov)
  have r13 := r12.dup5 (by evm_kdecide) (by evm_ov)
  have r14 := r13.and (by evm_kdecide) (by evm_ov)
  have r15 := r14.push1 (UInt256.ofNat 68) (by evm_kdecide) (by evm_ov)
  have r16 := r15.dup3 (by evm_kdecide) (by evm_ov)
  have r17 := r16.add (by evm_kdecide) (by evm_ov)
  have r18 := RD.genMstore r17 (by evm_kdecide) (by evm_ov)
  have r19 := r18.push1 (UInt256.ofNat 100) (by evm_kdecide) (by evm_ov)
  have r20 := r19.dup1 (by evm_kdecide) (by evm_ov)
  have r21 := r20.dup3 (by evm_kdecide) (by evm_ov)
  have r22 := r21.add (by evm_kdecide) (by evm_ov)
  have r23 := r22.dup5 (by evm_kdecide) (by evm_ov)
  have r24 := r23.swap1 (by evm_kdecide) (by evm_ov)
  have r25 := RD.genMstore r24 (by evm_kdecide) (by evm_ov)
  have r26 := r25.dup3 (by evm_kdecide) (by evm_ov)
  have r27 := RD.genMload r26 (by evm_kdecide) (by evm_ov)
  have r28 := r27.dup1 (by evm_kdecide) (by evm_ov)
  have r29 := r28.dup4 (by evm_kdecide) (by evm_ov)
  have r30 := r29.sub (by evm_kdecide) (by evm_ov)
  have r31 := r30.swap1 (by evm_kdecide) (by evm_ov)
  have r32 := r31.swap2 (by evm_kdecide) (by evm_ov)
  have r33 := r32.add (by evm_kdecide) (by evm_ov)
  have r34 := r33.dup2 (by evm_kdecide) (by evm_ov)
  have r35 := RD.genMstore r34 (by evm_kdecide) (by evm_ov)
  have r36 := r35.push1 (UInt256.ofNat 132) (by evm_kdecide) (by evm_ov)
  have r37 := r36.swap1 (by evm_kdecide) (by evm_ov)
  have r38 := r37.swap2 (by evm_kdecide) (by evm_ov)
  have r39 := r38.add (by evm_kdecide) (by evm_ov)
  have r40 := r39.swap1 (by evm_kdecide) (by evm_ov)
  have r41 := r40.swap2 (by evm_kdecide) (by evm_ov)
  have r42 := RD.genMstore r41 (by evm_kdecide) (by evm_ov)
  have r43 := r42.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r44 := r43.dup2 (by evm_kdecide) (by evm_ov)
  have r45 := r44.add (by evm_kdecide) (by evm_ov)
  have r46 := r45.dup1 (by evm_kdecide) (by evm_ov)
  have r47 := RD.genMload r46 (by evm_kdecide) (by evm_ov)
  have r48 := r47.pushConst (UInt256.ofNat 26959946667150639794667015087019630673637144422540572481103610249215) (width := 28) (op := .PUSH28) (by decide) (by evm_kdecide) (by evm_ov)
  have r49 := r48.and (by evm_kdecide) (by evm_ov)
  have r50 := r49.pushConst (UInt256.ofNat 35758974700130516083418609194394918359861923750008682890162270758914934964224) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r51 := r50.or (by evm_kdecide) (by evm_ov)
  have r52 := r51.swap1 (by evm_kdecide) (by evm_ov)
  have r53 := RD.genMstore r52 (by evm_kdecide) (by evm_ov)
  have r54 := r53.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r55 := r54.swap1 (by evm_kdecide) (by evm_ov)
  have r56 := r55.push2 (UInt256.ofNat 1699) (by evm_kdecide) (by evm_ov)
  have r57 := r56.swap1 (by evm_kdecide) (by evm_ov)
  have r58 := r57.dup8 (by evm_kdecide) (by evm_ov)
  have r59 := r58.swap1 (by evm_kdecide) (by evm_ov)
  have r60 := r59.chainid (by evm_kdecide) (by evm_ov)
  have r61 := r60.swap1 (by evm_kdecide) (by evm_ov)
  have r62 := r61.dup9 (by evm_kdecide) (by evm_ov)
  have r63 := r62.swap1 (by evm_kdecide) (by evm_ov)
  have r64 := r63.address (by evm_kdecide) (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1692)) r64 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_1539_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1539) (x0 :: x1 :: x2 :: x3 :: x4 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1692) (ethbridge_block_1539_stack (ee := ee) (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (R := R)) (ethbridge_block_1539_memory (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_1539 hstack h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_1692`. -/
def ethbridge_block_1692_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  (x1 :: x0 :: x0 :: R)

/-- Automatically generated RD summary for bytecode block at pc 1692. -/
theorem ethbridge_block_1692 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2314) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1692) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2314) (ethbridge_block_1692_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 5) (C + ((20))) := by
  let r0 := h
  have r1 := r0.swap1 (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup2 (by evm_kdecide) (by evm_ov)
  have r3 := r2.swap1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 2314) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2314)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_1692_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2314) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1692) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2314) (ethbridge_block_1692_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_1692 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_1699`. -/
def ethbridge_block_1699_stack {g : Sat256} {mem : ByteArray} {aw : UInt256} {C : ℕ} {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  (((g.subNat (C + ((84) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256))) + 2)).toUInt256) :: (UInt256.ofNat 376793390874373408599387495934666716005045108771) :: (memLoad (UInt256.ofNat 64) (x0.toByteArray.write 0 ((UInt256.ofNat 96634408021126418473330570835253591908004151213357964794993293693437865885696).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)).toNat 32)) :: (UInt256.sub ((UInt256.ofNat 36) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (UInt256.ofNat 64) (x0.toByteArray.write 0 ((UInt256.ofNat 96634408021126418473330570835253591908004151213357964794993293693437865885696).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)).toNat 32))) :: (memLoad (UInt256.ofNat 64) (x0.toByteArray.write 0 ((UInt256.ofNat 96634408021126418473330570835253591908004151213357964794993293693437865885696).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)).toNat 32)) :: (UInt256.ofNat 32) :: ((UInt256.ofNat 36) + (memLoad (UInt256.ofNat 64) mem)) :: (UInt256.ofNat 3584369406) :: (UInt256.ofNat 376793390874373408599387495934666716005045108771) :: x0 :: R)

/-- Final memory for bytecode block summary `ethbridge_block_1699`. -/
def ethbridge_block_1699_memory {mem : ByteArray} {x0 : UInt256} : ByteArray :=
  (x0.toByteArray.write 0 ((UInt256.ofNat 96634408021126418473330570835253591908004151213357964794993293693437865885696).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1699. -/
theorem ethbridge_block_1699 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1699) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1790) (ethbridge_block_1699_stack (g := g) (mem := mem) (aw := aw) (C := C) (x0 := x0) (R := R)) (ethbridge_block_1699_memory (mem := mem) (x0 := x0)) (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 30) (C + ((86) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r3 := RD.genMload r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.pushConst (UInt256.ofNat 96634408021126418473330570835253591908004151213357964794993293693437865885696) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
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
  have r18 := r17.push4 (UInt256.ofNat 3584369406) (by evm_kdecide) (by evm_ov)
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
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1790)) r30 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_1699_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1699) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1790) (ethbridge_block_1699_stack (g := g) (mem := mem) (aw := aw) (C := C) (x0 := x0) (R := R)) (ethbridge_block_1699_memory (mem := mem) (x0 := x0)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_1699 hstack h)
  exact ⟨_, k', C', h'⟩

/- Unsupported instruction boundary at pc 1790: staticcall (0xfa). No RD transition is asserted. Summaries resume at pc 1791 from a fresh symbolic RD state. -/

/-- Final stack for bytecode block summary `ethbridge_block_1791_taken`. -/
def ethbridge_block_1791_taken_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1791. -/
theorem ethbridge_block_1791_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 1807) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1791) (x0 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1807) (ethbridge_block_1791_taken_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.iszero (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.iszero (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 1807) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1807)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_1791_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 1807) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1791) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1807) (ethbridge_block_1791_taken_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_1791_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_1791_fallthrough`. -/
def ethbridge_block_1791_fallthrough_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1791. -/
theorem ethbridge_block_1791_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1791) (x0 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1798) (ethbridge_block_1791_fallthrough_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.iszero (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.iszero (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 1807) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1798)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_1791_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1791) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1798) (ethbridge_block_1791_fallthrough_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_1791_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1798. -/
theorem ethbridge_block_1798 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1798) R mem aw rdata σ k C)
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

/-- Final stack for bytecode block summary `ethbridge_block_1807`. -/
def ethbridge_block_1807_stack {mem : ByteArray} {rdata : ByteArray} {R : List UInt256} : List UInt256 :=
  ((memLoad (UInt256.ofNat 64) mem) :: ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat rdata.size)) :: (UInt256.ofNat 1843) :: R)

/-- Final memory for bytecode block summary `ethbridge_block_1807`. -/
def ethbridge_block_1807_memory {mem : ByteArray} {rdata : ByteArray} : ByteArray :=
  (((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat rdata.size) + (UInt256.ofNat 31)) (UInt256.lnot (UInt256.ofNat 31)))).toByteArray.write 0 mem (UInt256.ofNat 64).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1807. -/
theorem ethbridge_block_1807 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2908) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1807) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2908) (ethbridge_block_1807_stack (mem := mem) (rdata := rdata) (R := R)) (ethbridge_block_1807_memory (mem := mem) (rdata := rdata)) (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 28) (C + ((81) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
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
  have r24 := r23.push2 (UInt256.ofNat 1843) (by evm_kdecide) (by evm_ov)
  have r25 := r24.swap2 (by evm_kdecide) (by evm_ov)
  have r26 := r25.swap1 (by evm_kdecide) (by evm_ov)
  have r27 := r26.push2 (UInt256.ofNat 2908) (by evm_kdecide) (by evm_ov)
  have r28 := r27.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2908)) r28 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_1807_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2908) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1807) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2908) (ethbridge_block_1807_stack (mem := mem) (rdata := rdata) (R := R)) (ethbridge_block_1807_memory (mem := mem) (rdata := rdata)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_1807 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_1843_taken`. -/
def ethbridge_block_1843_taken_stack {R : List UInt256} : List UInt256 :=
  R

/-- Automatically generated RD summary for bytecode block at pc 1843. -/
theorem ethbridge_block_1843_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : x0 ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 1897) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1843) (x0 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1897) (ethbridge_block_1843_taken_stack (R := R)) mem aw rdata σ (k + 3) (C + ((14))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 1897) (by evm_kdecide) (by evm_ov)
  have r3 := r2.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1897)) r3 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_1843_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : x0 ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 1897) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1843) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1897) (ethbridge_block_1843_taken_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_1843_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_1843_fallthrough`. -/
def ethbridge_block_1843_fallthrough_stack {R : List UInt256} : List UInt256 :=
  R

/-- Automatically generated RD summary for bytecode block at pc 1843. -/
theorem ethbridge_block_1843_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : x0 = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1843) (x0 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1848) (ethbridge_block_1843_fallthrough_stack (R := R)) mem aw rdata σ (k + 3) (C + ((14))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 1897) (by evm_kdecide) (by evm_ov)
  have r3 := r2.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1848)) r3 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_1843_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : x0 = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1843) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1848) (ethbridge_block_1843_fallthrough_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_1843_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1848. -/
theorem ethbridge_block_1848 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1848) R mem aw rdata σ k C)
    : RDrev BridgeEvm.ethbridgeRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r2 := RD.genMload r1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pushConst (UInt256.ofNat 4283108933445421824756484206644851292164488533893701873295093031910181437440) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
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

/-- Final memory for bytecode block summary `ethbridge_block_1897_taken`. -/
def ethbridge_block_1897_taken_memory {mem : ByteArray} {x0 : UInt256} : ByteArray :=
  ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1897. -/
theorem ethbridge_block_1897_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.land (UInt256.ofNat 255) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256))))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 1970) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1897) (x0 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1970) (x0 :: R) (ethbridge_block_1897_taken_memory (mem := mem) (x0 := x0)) (M (M (M aw (UInt256.ofNat 0) (⟨32⟩ : UInt256)) (UInt256.ofNat 32) (⟨32⟩ : UInt256)) (UInt256.ofNat 0) (UInt256.ofNat 64)) rdata σ k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup2 (by evm_kdecide) (by evm_ov)
  have r5 := RD.genMstore r4 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup2 (by evm_kdecide) (by evm_ov)
  have r8 := r7.swap1 (by evm_kdecide) (by evm_ov)
  have r9 := RD.genMstore r8 (by evm_kdecide) (by evm_ov)
  have r10 := r9.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r11 := r10.swap1 (by evm_kdecide) (by evm_ov)
  have r12 := RD.genKeccak256 r11 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r13⟩ := RD.sload r12 (by evm_kdecide) (by evm_ov)
  have r14 := r13.push1 (UInt256.ofNat 255) (by evm_kdecide) (by evm_ov)
  have r15 := r14.and (by evm_kdecide) (by evm_ov)
  have r16 := r15.iszero (by evm_kdecide) (by evm_ov)
  have r17 := r16.push2 (UInt256.ofNat 1970) (by evm_kdecide) (by evm_ov)
  have r18 := r17.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1970)) r18 (by evm_kdecide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_1897_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.land (UInt256.ofNat 255) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256))))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 1970) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1897) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1970) (x0 :: R) (ethbridge_block_1897_taken_memory (mem := mem) (x0 := x0)) aw' rdata σ k' C' := by
  obtain ⟨k0, C0, h0⟩ := ethbridge_block_1897_taken hstack hcond hvalid h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Final memory for bytecode block summary `ethbridge_block_1897_fallthrough`. -/
def ethbridge_block_1897_fallthrough_memory {mem : ByteArray} {x0 : UInt256} : ByteArray :=
  ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1897. -/
theorem ethbridge_block_1897_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.land (UInt256.ofNat 255) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256))))) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1897) (x0 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1921) (x0 :: R) (ethbridge_block_1897_fallthrough_memory (mem := mem) (x0 := x0)) (M (M (M aw (UInt256.ofNat 0) (⟨32⟩ : UInt256)) (UInt256.ofNat 32) (⟨32⟩ : UInt256)) (UInt256.ofNat 0) (UInt256.ofNat 64)) rdata σ k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup2 (by evm_kdecide) (by evm_ov)
  have r5 := RD.genMstore r4 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup2 (by evm_kdecide) (by evm_ov)
  have r8 := r7.swap1 (by evm_kdecide) (by evm_ov)
  have r9 := RD.genMstore r8 (by evm_kdecide) (by evm_ov)
  have r10 := r9.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r11 := r10.swap1 (by evm_kdecide) (by evm_ov)
  have r12 := RD.genKeccak256 r11 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r13⟩ := RD.sload r12 (by evm_kdecide) (by evm_ov)
  have r14 := r13.push1 (UInt256.ofNat 255) (by evm_kdecide) (by evm_ov)
  have r15 := r14.and (by evm_kdecide) (by evm_ov)
  have r16 := r15.iszero (by evm_kdecide) (by evm_ov)
  have r17 := r16.push2 (UInt256.ofNat 1970) (by evm_kdecide) (by evm_ov)
  have r18 := r17.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1921)) r18 (by evm_kdecide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_1897_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.land (UInt256.ofNat 255) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256))))) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1897) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1921) (x0 :: R) (ethbridge_block_1897_fallthrough_memory (mem := mem) (x0 := x0)) aw' rdata σ k' C' := by
  obtain ⟨k0, C0, h0⟩ := ethbridge_block_1897_fallthrough hstack hcond h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1921. -/
theorem ethbridge_block_1921 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1921) R mem aw rdata σ k C)
    : RDrev BridgeEvm.ethbridgeRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r2 := RD.genMload r1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pushConst (UInt256.ofNat 19663479255927283481287023152847279242139961893411169798514943407892948058112) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
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

/-- Final stack for bytecode block summary `ethbridge_block_1970_taken`. -/
def ethbridge_block_1970_taken_stack {ee : ExecutionEnv} {mem : ByteArray} {σ : AccountMap} {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero (extCodeSizeWord (sstoreAccountMap ee.codeOwner σ (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (UInt256.lor (UInt256.ofNat 1) (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639680) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256)))))) (UInt256.ofNat 376793390874373408599387495934666716005045108773))) :: (UInt256.ofNat 376793390874373408599387495934666716005045108773) :: (UInt256.ofNat 0) :: (memLoad (UInt256.ofNat 64) (x1.toByteArray.write 0 ((UInt256.ofNat 72570022874062638528011751457397263716769196454539065078543251854057308946432).toByteArray.write 0 ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)).toNat 32) ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) + (UInt256.ofNat 4)).toNat 32)) :: (UInt256.sub ((UInt256.ofNat 36) + (memLoad (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32))) (memLoad (UInt256.ofNat 64) (x1.toByteArray.write 0 ((UInt256.ofNat 72570022874062638528011751457397263716769196454539065078543251854057308946432).toByteArray.write 0 ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)).toNat 32) ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) + (UInt256.ofNat 4)).toNat 32))) :: (memLoad (UInt256.ofNat 64) (x1.toByteArray.write 0 ((UInt256.ofNat 72570022874062638528011751457397263716769196454539065078543251854057308946432).toByteArray.write 0 ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)).toNat 32) ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) + (UInt256.ofNat 4)).toNat 32)) :: (UInt256.ofNat 0) :: ((UInt256.ofNat 36) + (memLoad (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32))) :: (UInt256.ofNat 2691771752) :: (UInt256.ofNat 376793390874373408599387495934666716005045108773) :: x0 :: x1 :: R)

/-- Final memory for bytecode block summary `ethbridge_block_1970_taken`. -/
def ethbridge_block_1970_taken_memory {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} : ByteArray :=
  (x1.toByteArray.write 0 ((UInt256.ofNat 72570022874062638528011751457397263716769196454539065078543251854057308946432).toByteArray.write 0 ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)).toNat 32) ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) + (UInt256.ofNat 4)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1970. -/
theorem ethbridge_block_1970_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 14 ≤ 1024)
    (hperm : ee.perm = true)
    (hcond : (UInt256.isZero (UInt256.isZero (extCodeSizeWord (sstoreAccountMap ee.codeOwner σ (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (UInt256.lor (UInt256.ofNat 1) (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639680) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256)))))) (UInt256.ofNat 376793390874373408599387495934666716005045108773)))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2127) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1970) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2127) (ethbridge_block_1970_taken_stack (ee := ee) (mem := mem) (σ := σ) (x0 := x0) (x1 := x1) (R := R)) (ethbridge_block_1970_taken_memory (mem := mem) (x0 := x0) (x1 := x1)) (M (M (M (M (M (M (M aw (UInt256.ofNat 0) (⟨32⟩ : UInt256)) (UInt256.ofNat 32) (⟨32⟩ : UInt256)) (UInt256.ofNat 0) (UInt256.ofNat 64)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) + (UInt256.ofNat 4)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata (sstoreAccountMap ee.codeOwner σ (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (UInt256.lor (UInt256.ofNat 1) (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639680) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256)))))) k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup2 (by evm_kdecide) (by evm_ov)
  have r5 := RD.genMstore r4 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup2 (by evm_kdecide) (by evm_ov)
  have r8 := r7.swap1 (by evm_kdecide) (by evm_ov)
  have r9 := RD.genMstore r8 (by evm_kdecide) (by evm_ov)
  have r10 := r9.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r11 := r10.swap1 (by evm_kdecide) (by evm_ov)
  have r12 := r11.dup2 (by evm_kdecide) (by evm_ov)
  have r13 := r12.swap1 (by evm_kdecide) (by evm_ov)
  have r14 := RD.genKeccak256 r13 (by evm_kdecide) (by evm_ov)
  have r15 := r14.dup1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r16⟩ := RD.sload r15 (by evm_kdecide) (by evm_ov)
  have r17 := r16.pushConst (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639680) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r18 := r17.and (by evm_kdecide) (by evm_ov)
  have r19 := r18.push1 (UInt256.ofNat 1) (by evm_kdecide) (by evm_ov)
  have r20 := r19.or (by evm_kdecide) (by evm_ov)
  have r21 := r20.swap1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r22⟩ := RD.sstore r21 hperm (by evm_kdecide) (by evm_ov)
  have r23 := RD.genMload r22 (by evm_kdecide) (by evm_ov)
  have r24 := r23.pushConst (UInt256.ofNat 72570022874062638528011751457397263716769196454539065078543251854057308946432) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r25 := r24.dup2 (by evm_kdecide) (by evm_ov)
  have r26 := RD.genMstore r25 (by evm_kdecide) (by evm_ov)
  have r27 := r26.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r28 := r27.dup2 (by evm_kdecide) (by evm_ov)
  have r29 := r28.add (by evm_kdecide) (by evm_ov)
  have r30 := r29.dup4 (by evm_kdecide) (by evm_ov)
  have r31 := r30.swap1 (by evm_kdecide) (by evm_ov)
  have r32 := RD.genMstore r31 (by evm_kdecide) (by evm_ov)
  have r33 := r32.push20 (UInt256.ofNat 376793390874373408599387495934666716005045108773) (by evm_kdecide) (by evm_ov)
  have r34 := r33.swap1 (by evm_kdecide) (by evm_ov)
  have r35 := r34.push4 (UInt256.ofNat 2691771752) (by evm_kdecide) (by evm_ov)
  have r36 := r35.swap1 (by evm_kdecide) (by evm_ov)
  have r37 := r36.push1 (UInt256.ofNat 36) (by evm_kdecide) (by evm_ov)
  have r38 := r37.add (by evm_kdecide) (by evm_ov)
  have r39 := r38.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r40 := r39.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r41 := RD.genMload r40 (by evm_kdecide) (by evm_ov)
  have r42 := r41.dup1 (by evm_kdecide) (by evm_ov)
  have r43 := r42.dup4 (by evm_kdecide) (by evm_ov)
  have r44 := r43.sub (by evm_kdecide) (by evm_ov)
  have r45 := r44.dup2 (by evm_kdecide) (by evm_ov)
  have r46 := r45.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r47 := r46.dup8 (by evm_kdecide) (by evm_ov)
  have r48 := r47.dup1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r49⟩ := r48.extcodesize (by evm_kdecide) (by evm_ov)
  have r50 := r49.iszero (by evm_kdecide) (by evm_ov)
  have r51 := r50.dup1 (by evm_kdecide) (by evm_ov)
  have r52 := r51.iszero (by evm_kdecide) (by evm_ov)
  have r53 := r52.push2 (UInt256.ofNat 2127) (by evm_kdecide) (by evm_ov)
  have r54 := r53.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2127)) r54 (by evm_kdecide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_1970_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 14 ≤ 1024)
    (hperm : ee.perm = true)
    (hcond : (UInt256.isZero (UInt256.isZero (extCodeSizeWord (sstoreAccountMap ee.codeOwner σ (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (UInt256.lor (UInt256.ofNat 1) (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639680) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256)))))) (UInt256.ofNat 376793390874373408599387495934666716005045108773)))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2127) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1970) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2127) (ethbridge_block_1970_taken_stack (ee := ee) (mem := mem) (σ := σ) (x0 := x0) (x1 := x1) (R := R)) (ethbridge_block_1970_taken_memory (mem := mem) (x0 := x0) (x1 := x1)) aw' rdata (sstoreAccountMap ee.codeOwner σ (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (UInt256.lor (UInt256.ofNat 1) (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639680) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256)))))) k' C' := by
  obtain ⟨k0, C0, h0⟩ := ethbridge_block_1970_taken hstack hperm hcond hvalid h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_1970_fallthrough`. -/
def ethbridge_block_1970_fallthrough_stack {ee : ExecutionEnv} {mem : ByteArray} {σ : AccountMap} {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero (extCodeSizeWord (sstoreAccountMap ee.codeOwner σ (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (UInt256.lor (UInt256.ofNat 1) (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639680) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256)))))) (UInt256.ofNat 376793390874373408599387495934666716005045108773))) :: (UInt256.ofNat 376793390874373408599387495934666716005045108773) :: (UInt256.ofNat 0) :: (memLoad (UInt256.ofNat 64) (x1.toByteArray.write 0 ((UInt256.ofNat 72570022874062638528011751457397263716769196454539065078543251854057308946432).toByteArray.write 0 ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)).toNat 32) ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) + (UInt256.ofNat 4)).toNat 32)) :: (UInt256.sub ((UInt256.ofNat 36) + (memLoad (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32))) (memLoad (UInt256.ofNat 64) (x1.toByteArray.write 0 ((UInt256.ofNat 72570022874062638528011751457397263716769196454539065078543251854057308946432).toByteArray.write 0 ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)).toNat 32) ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) + (UInt256.ofNat 4)).toNat 32))) :: (memLoad (UInt256.ofNat 64) (x1.toByteArray.write 0 ((UInt256.ofNat 72570022874062638528011751457397263716769196454539065078543251854057308946432).toByteArray.write 0 ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)).toNat 32) ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) + (UInt256.ofNat 4)).toNat 32)) :: (UInt256.ofNat 0) :: ((UInt256.ofNat 36) + (memLoad (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32))) :: (UInt256.ofNat 2691771752) :: (UInt256.ofNat 376793390874373408599387495934666716005045108773) :: x0 :: x1 :: R)

/-- Final memory for bytecode block summary `ethbridge_block_1970_fallthrough`. -/
def ethbridge_block_1970_fallthrough_memory {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} : ByteArray :=
  (x1.toByteArray.write 0 ((UInt256.ofNat 72570022874062638528011751457397263716769196454539065078543251854057308946432).toByteArray.write 0 ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)).toNat 32) ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) + (UInt256.ofNat 4)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1970. -/
theorem ethbridge_block_1970_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 14 ≤ 1024)
    (hperm : ee.perm = true)
    (hcond : (UInt256.isZero (UInt256.isZero (extCodeSizeWord (sstoreAccountMap ee.codeOwner σ (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (UInt256.lor (UInt256.ofNat 1) (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639680) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256)))))) (UInt256.ofNat 376793390874373408599387495934666716005045108773)))) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1970) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2123) (ethbridge_block_1970_fallthrough_stack (ee := ee) (mem := mem) (σ := σ) (x0 := x0) (x1 := x1) (R := R)) (ethbridge_block_1970_fallthrough_memory (mem := mem) (x0 := x0) (x1 := x1)) (M (M (M (M (M (M (M aw (UInt256.ofNat 0) (⟨32⟩ : UInt256)) (UInt256.ofNat 32) (⟨32⟩ : UInt256)) (UInt256.ofNat 0) (UInt256.ofNat 64)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) + (UInt256.ofNat 4)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata (sstoreAccountMap ee.codeOwner σ (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (UInt256.lor (UInt256.ofNat 1) (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639680) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256)))))) k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup2 (by evm_kdecide) (by evm_ov)
  have r5 := RD.genMstore r4 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup2 (by evm_kdecide) (by evm_ov)
  have r8 := r7.swap1 (by evm_kdecide) (by evm_ov)
  have r9 := RD.genMstore r8 (by evm_kdecide) (by evm_ov)
  have r10 := r9.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r11 := r10.swap1 (by evm_kdecide) (by evm_ov)
  have r12 := r11.dup2 (by evm_kdecide) (by evm_ov)
  have r13 := r12.swap1 (by evm_kdecide) (by evm_ov)
  have r14 := RD.genKeccak256 r13 (by evm_kdecide) (by evm_ov)
  have r15 := r14.dup1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r16⟩ := RD.sload r15 (by evm_kdecide) (by evm_ov)
  have r17 := r16.pushConst (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639680) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r18 := r17.and (by evm_kdecide) (by evm_ov)
  have r19 := r18.push1 (UInt256.ofNat 1) (by evm_kdecide) (by evm_ov)
  have r20 := r19.or (by evm_kdecide) (by evm_ov)
  have r21 := r20.swap1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r22⟩ := RD.sstore r21 hperm (by evm_kdecide) (by evm_ov)
  have r23 := RD.genMload r22 (by evm_kdecide) (by evm_ov)
  have r24 := r23.pushConst (UInt256.ofNat 72570022874062638528011751457397263716769196454539065078543251854057308946432) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r25 := r24.dup2 (by evm_kdecide) (by evm_ov)
  have r26 := RD.genMstore r25 (by evm_kdecide) (by evm_ov)
  have r27 := r26.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r28 := r27.dup2 (by evm_kdecide) (by evm_ov)
  have r29 := r28.add (by evm_kdecide) (by evm_ov)
  have r30 := r29.dup4 (by evm_kdecide) (by evm_ov)
  have r31 := r30.swap1 (by evm_kdecide) (by evm_ov)
  have r32 := RD.genMstore r31 (by evm_kdecide) (by evm_ov)
  have r33 := r32.push20 (UInt256.ofNat 376793390874373408599387495934666716005045108773) (by evm_kdecide) (by evm_ov)
  have r34 := r33.swap1 (by evm_kdecide) (by evm_ov)
  have r35 := r34.push4 (UInt256.ofNat 2691771752) (by evm_kdecide) (by evm_ov)
  have r36 := r35.swap1 (by evm_kdecide) (by evm_ov)
  have r37 := r36.push1 (UInt256.ofNat 36) (by evm_kdecide) (by evm_ov)
  have r38 := r37.add (by evm_kdecide) (by evm_ov)
  have r39 := r38.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r40 := r39.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r41 := RD.genMload r40 (by evm_kdecide) (by evm_ov)
  have r42 := r41.dup1 (by evm_kdecide) (by evm_ov)
  have r43 := r42.dup4 (by evm_kdecide) (by evm_ov)
  have r44 := r43.sub (by evm_kdecide) (by evm_ov)
  have r45 := r44.dup2 (by evm_kdecide) (by evm_ov)
  have r46 := r45.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r47 := r46.dup8 (by evm_kdecide) (by evm_ov)
  have r48 := r47.dup1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r49⟩ := r48.extcodesize (by evm_kdecide) (by evm_ov)
  have r50 := r49.iszero (by evm_kdecide) (by evm_ov)
  have r51 := r50.dup1 (by evm_kdecide) (by evm_ov)
  have r52 := r51.iszero (by evm_kdecide) (by evm_ov)
  have r53 := r52.push2 (UInt256.ofNat 2127) (by evm_kdecide) (by evm_ov)
  have r54 := r53.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2123)) r54 (by evm_kdecide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_1970_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 14 ≤ 1024)
    (hperm : ee.perm = true)
    (hcond : (UInt256.isZero (UInt256.isZero (extCodeSizeWord (sstoreAccountMap ee.codeOwner σ (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (UInt256.lor (UInt256.ofNat 1) (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639680) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256)))))) (UInt256.ofNat 376793390874373408599387495934666716005045108773)))) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 1970) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2123) (ethbridge_block_1970_fallthrough_stack (ee := ee) (mem := mem) (σ := σ) (x0 := x0) (x1 := x1) (R := R)) (ethbridge_block_1970_fallthrough_memory (mem := mem) (x0 := x0) (x1 := x1)) aw' rdata (sstoreAccountMap ee.codeOwner σ (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (UInt256.lor (UInt256.ofNat 1) (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639680) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 0).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256)))))) k' C' := by
  obtain ⟨k0, C0, h0⟩ := ethbridge_block_1970_fallthrough hstack hperm hcond h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

end ethbridgeBlocks
