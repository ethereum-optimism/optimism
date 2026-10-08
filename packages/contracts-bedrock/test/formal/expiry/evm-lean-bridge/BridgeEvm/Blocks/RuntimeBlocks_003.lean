import Reasoning.Reach
import BridgeEvm.Bytecode

open Solm ABI Ethereum Ethereum.EVM
open Reasoning.Theory Reasoning.Reach

namespace ethbridgeBlocks

/-- Final stack for bytecode block summary `ethbridge_block_316`. -/
def ethbridge_block_316_stack {ee : ExecutionEnv} {mem : ByteArray} {σ : AccountMap} {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.land (UInt256.ofNat 255) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) (x0.toByteArray.write 0 ((UInt256.ofNat 0).toByteArray.write 0 mem (UInt256.ofNat 32).toNat 32) (UInt256.ofNat 0).toNat 32)) (⟨0⟩ : UInt256)))) :: x1 :: R)

/-- Final memory for bytecode block summary `ethbridge_block_316`. -/
def ethbridge_block_316_memory {mem : ByteArray} {x0 : UInt256} : ByteArray :=
  (x0.toByteArray.write 0 ((UInt256.ofNat 0).toByteArray.write 0 mem (UInt256.ofNat 32).toNat 32) (UInt256.ofNat 0).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 316. -/
theorem ethbridge_block_316 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains x1 = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 316) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 x1 (ethbridge_block_316_stack (ee := ee) (mem := mem) (σ := σ) (x0 := x0) (x1 := x1) (R := R)) (ethbridge_block_316_memory (mem := mem) (x0 := x0)) (M (M (M aw (UInt256.ofNat 32) (⟨32⟩ : UInt256)) (UInt256.ofNat 0) (⟨32⟩ : UInt256)) (UInt256.ofNat 0) (UInt256.ofNat 64)) rdata σ k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup2 (by evm_kdecide) (by evm_ov)
  have r5 := r4.swap1 (by evm_kdecide) (by evm_ov)
  have r6 := RD.genMstore r5 (by evm_kdecide) (by evm_ov)
  have r7 := r6.swap1 (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup2 (by evm_kdecide) (by evm_ov)
  have r9 := RD.genMstore r8 (by evm_kdecide) (by evm_ov)
  have r10 := r9.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r11 := r10.swap1 (by evm_kdecide) (by evm_ov)
  have r12 := RD.genKeccak256 r11 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r13⟩ := RD.sload r12 (by evm_kdecide) (by evm_ov)
  have r14 := r13.push1 (UInt256.ofNat 255) (by evm_kdecide) (by evm_ov)
  have r15 := r14.and (by evm_kdecide) (by evm_ov)
  have r16 := r15.dup2 (by evm_kdecide) (by evm_ov)
  have r17 := r16.jump (by evm_kdecide) hvalid (by evm_ov)
  exact ⟨_, _, r17⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_316_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains x1 = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 316) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 x1 (ethbridge_block_316_stack (ee := ee) (mem := mem) (σ := σ) (x0 := x0) (x1 := x1) (R := R)) (ethbridge_block_316_memory (mem := mem) (x0 := x0)) aw' rdata σ k' C' := by
  obtain ⟨k0, C0, h0⟩ := ethbridge_block_316 hstack hvalid h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_337`. -/
def ethbridge_block_337_stack {mem : ByteArray} {R : List UInt256} : List UInt256 :=
  (((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) :: R)

/-- Final memory for bytecode block summary `ethbridge_block_337`. -/
def ethbridge_block_337_memory {mem : ByteArray} {x0 : UInt256} : ByteArray :=
  ((UInt256.isZero (UInt256.isZero x0)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 337. -/
theorem ethbridge_block_337 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 215) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 337) (x0 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 215) (ethbridge_block_337_stack (mem := mem) (R := R)) (ethbridge_block_337_memory (mem := mem) (x0 := x0)) (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) rdata σ (k + 12) (C + ((39) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r3 := RD.genMload r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.swap1 (by evm_kdecide) (by evm_ov)
  have r5 := r4.iszero (by evm_kdecide) (by evm_ov)
  have r6 := r5.iszero (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup2 (by evm_kdecide) (by evm_ov)
  have r8 := RD.genMstore r7 (by evm_kdecide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r10 := r9.add (by evm_kdecide) (by evm_ov)
  have r11 := r10.push2 (UInt256.ofNat 215) (by evm_kdecide) (by evm_ov)
  have r12 := r11.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 215)) r12 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_337_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 215) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 337) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 215) (ethbridge_block_337_stack (mem := mem) (R := R)) (ethbridge_block_337_memory (mem := mem) (x0 := x0)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_337 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 353. -/
theorem ethbridge_block_353_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 376793390874373408599387495934666716005045108771) (UInt256.ofNat ee.source.val)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 430) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 353) R mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 430) R mem aw rdata σ (k + 6) (C + ((22))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.caller (by evm_kdecide) (by evm_ov)
  have r3 := r2.push20 (UInt256.ofNat 376793390874373408599387495934666716005045108771) (by evm_kdecide) (by evm_ov)
  have r4 := r3.eq (by evm_kdecide) (by evm_ov)
  have r5 := r4.push2 (UInt256.ofNat 430) (by evm_kdecide) (by evm_ov)
  have r6 := r5.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 430)) r6 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_353_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 376793390874373408599387495934666716005045108771) (UInt256.ofNat ee.source.val)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 430) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 353) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 430) R mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_353_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 353. -/
theorem ethbridge_block_353_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 376793390874373408599387495934666716005045108771) (UInt256.ofNat ee.source.val)) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 353) R mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 381) R mem aw rdata σ (k + 6) (C + ((22))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.caller (by evm_kdecide) (by evm_ov)
  have r3 := r2.push20 (UInt256.ofNat 376793390874373408599387495934666716005045108771) (by evm_kdecide) (by evm_ov)
  have r4 := r3.eq (by evm_kdecide) (by evm_ov)
  have r5 := r4.push2 (UInt256.ofNat 430) (by evm_kdecide) (by evm_ov)
  have r6 := r5.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 381)) r6 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_353_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat 376793390874373408599387495934666716005045108771) (UInt256.ofNat ee.source.val)) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 353) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 381) R mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_353_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 381. -/
theorem ethbridge_block_381 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 381) R mem aw rdata σ k C)
    : RDrev BridgeEvm.ethbridgeRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r2 := RD.genMload r1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pushConst (UInt256.ofNat 59118985759084958080972419199848406897631161684926069324597705158744233476096) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
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

/-- Final stack for bytecode block summary `ethbridge_block_430`. -/
def ethbridge_block_430_stack {g : Sat256} {mem : ByteArray} {aw : UInt256} {C : ℕ} {R : List UInt256} : List UInt256 :=
  (((g.subNat (C + ((76) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256))) + 2)).toUInt256) :: (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (UInt256.ofNat 376793390874373408599387495934666716005045108771)) :: (memLoad (UInt256.ofNat 64) ((UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 2033634286)) (UInt256.ofNat 224)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)) :: (UInt256.sub ((UInt256.ofNat 4) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (UInt256.ofNat 64) ((UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 2033634286)) (UInt256.ofNat 224)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32))) :: (memLoad (UInt256.ofNat 64) ((UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 2033634286)) (UInt256.ofNat 224)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)) :: (UInt256.ofNat 64) :: ((UInt256.ofNat 4) + (memLoad (UInt256.ofNat 64) mem)) :: (UInt256.ofNat 2033634286) :: (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (UInt256.ofNat 376793390874373408599387495934666716005045108771)) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: R)

/-- Final memory for bytecode block summary `ethbridge_block_430`. -/
def ethbridge_block_430_memory {mem : ByteArray} : ByteArray :=
  ((UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 2033634286)) (UInt256.ofNat 224)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 430. -/
theorem ethbridge_block_430 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 11 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 430) R mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 510) (ethbridge_block_430_stack (g := g) (mem := mem) (aw := aw) (C := C) (R := R)) (ethbridge_block_430_memory (mem := mem)) (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 27) (C + ((78) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push20 (UInt256.ofNat 376793390874373408599387495934666716005045108771) (by evm_kdecide) (by evm_ov)
  have r5 := r4.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r6 := r5.and (by evm_kdecide) (by evm_ov)
  have r7 := r6.push4 (UInt256.ofNat 2033634286) (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r9 := RD.genMload r8 (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup2 (by evm_kdecide) (by evm_ov)
  have r11 := r10.push4 (UInt256.ofNat 4294967295) (by evm_kdecide) (by evm_ov)
  have r12 := r11.and (by evm_kdecide) (by evm_ov)
  have r13 := r12.push1 (UInt256.ofNat 224) (by evm_kdecide) (by evm_ov)
  have r14 := r13.shl (by evm_kdecide) (by evm_ov)
  have r15 := r14.dup2 (by evm_kdecide) (by evm_ov)
  have r16 := RD.genMstore r15 (by evm_kdecide) (by evm_ov)
  have r17 := r16.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r18 := r17.add (by evm_kdecide) (by evm_ov)
  have r19 := r18.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r20 := r19.dup1 (by evm_kdecide) (by evm_ov)
  have r21 := RD.genMload r20 (by evm_kdecide) (by evm_ov)
  have r22 := r21.dup1 (by evm_kdecide) (by evm_ov)
  have r23 := r22.dup4 (by evm_kdecide) (by evm_ov)
  have r24 := r23.sub (by evm_kdecide) (by evm_ov)
  have r25 := r24.dup2 (by evm_kdecide) (by evm_ov)
  have r26 := r25.dup7 (by evm_kdecide) (by evm_ov)
  have r27 := RD.genGas (RD.normalizeCounters (k' := k + 26) (C' := C + ((76) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) r26 (by omega) (by omega)) (by evm_kdecide) (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 510)) r27 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_430_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 11 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 430) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 510) (ethbridge_block_430_stack (g := g) (mem := mem) (aw := aw) (C := C) (R := R)) (ethbridge_block_430_memory (mem := mem)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_430 hstack h)
  exact ⟨_, k', C', h'⟩

/- Unsupported instruction boundary at pc 510: staticcall (0xfa). No RD transition is asserted. Summaries resume at pc 511 from a fresh symbolic RD state. -/

/-- Final stack for bytecode block summary `ethbridge_block_511_taken`. -/
def ethbridge_block_511_taken_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 511. -/
theorem ethbridge_block_511_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 527) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 511) (x0 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 527) (ethbridge_block_511_taken_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.iszero (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.iszero (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 527) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 527)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_511_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 527) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 511) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 527) (ethbridge_block_511_taken_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_511_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_511_fallthrough`. -/
def ethbridge_block_511_fallthrough_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 511. -/
theorem ethbridge_block_511_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 511) (x0 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 518) (ethbridge_block_511_fallthrough_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.iszero (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.iszero (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 527) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 518)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_511_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 511) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 518) (ethbridge_block_511_fallthrough_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_511_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 518. -/
theorem ethbridge_block_518 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 518) R mem aw rdata σ k C)
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

/-- Final stack for bytecode block summary `ethbridge_block_527`. -/
def ethbridge_block_527_stack {mem : ByteArray} {rdata : ByteArray} {R : List UInt256} : List UInt256 :=
  ((memLoad (UInt256.ofNat 64) mem) :: ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat rdata.size)) :: (UInt256.ofNat 563) :: R)

/-- Final memory for bytecode block summary `ethbridge_block_527`. -/
def ethbridge_block_527_memory {mem : ByteArray} {rdata : ByteArray} : ByteArray :=
  (((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat rdata.size) + (UInt256.ofNat 31)) (UInt256.lnot (UInt256.ofNat 31)))).toByteArray.write 0 mem (UInt256.ofNat 64).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 527. -/
theorem ethbridge_block_527 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2775) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 527) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2775) (ethbridge_block_527_stack (mem := mem) (rdata := rdata) (R := R)) (ethbridge_block_527_memory (mem := mem) (rdata := rdata)) (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 28) (C + ((81) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
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
  have r24 := r23.push2 (UInt256.ofNat 563) (by evm_kdecide) (by evm_ov)
  have r25 := r24.swap2 (by evm_kdecide) (by evm_ov)
  have r26 := r25.swap1 (by evm_kdecide) (by evm_ov)
  have r27 := r26.push2 (UInt256.ofNat 2775) (by evm_kdecide) (by evm_ov)
  have r28 := r27.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2775)) r28 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_527_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2775) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 527) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2775) (ethbridge_block_527_stack (mem := mem) (rdata := rdata) (R := R)) (ethbridge_block_527_memory (mem := mem) (rdata := rdata)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_527 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_563_taken`. -/
def ethbridge_block_563_taken_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 563. -/
theorem ethbridge_block_563_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat ee.codeOwner.val) (UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 647) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 563) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 647) (ethbridge_block_563_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 13) (C + ((41))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.swap3 (by evm_kdecide) (by evm_ov)
  have r4 := r3.pop (by evm_kdecide) (by evm_ov)
  have r5 := r4.swap1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.pop (by evm_kdecide) (by evm_ov)
  have r7 := r6.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup3 (by evm_kdecide) (by evm_ov)
  have r9 := r8.and (by evm_kdecide) (by evm_ov)
  have r10 := r9.address (by evm_kdecide) (by evm_ov)
  have r11 := r10.eq (by evm_kdecide) (by evm_ov)
  have r12 := r11.push2 (UInt256.ofNat 647) (by evm_kdecide) (by evm_ov)
  have r13 := r12.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 647)) r13 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_563_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat ee.codeOwner.val) (UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 647) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 563) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 647) (ethbridge_block_563_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_563_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_563_fallthrough`. -/
def ethbridge_block_563_fallthrough_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 563. -/
theorem ethbridge_block_563_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat ee.codeOwner.val) (UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975))) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 563) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 598) (ethbridge_block_563_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 13) (C + ((41))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.swap3 (by evm_kdecide) (by evm_ov)
  have r4 := r3.pop (by evm_kdecide) (by evm_ov)
  have r5 := r4.swap1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.pop (by evm_kdecide) (by evm_ov)
  have r7 := r6.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup3 (by evm_kdecide) (by evm_ov)
  have r9 := r8.and (by evm_kdecide) (by evm_ov)
  have r10 := r9.address (by evm_kdecide) (by evm_ov)
  have r11 := r10.eq (by evm_kdecide) (by evm_ov)
  have r12 := r11.push2 (UInt256.ofNat 647) (by evm_kdecide) (by evm_ov)
  have r13 := r12.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 598)) r13 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_563_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.eq (UInt256.ofNat ee.codeOwner.val) (UInt256.land x1 (UInt256.ofNat 1461501637330902918203684832716283019655932542975))) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 563) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 598) (ethbridge_block_563_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_563_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 598. -/
theorem ethbridge_block_598 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 598) R mem aw rdata σ k C)
    : RDrev BridgeEvm.ethbridgeRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r2 := RD.genMload r1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pushConst (UInt256.ofNat 85096452711721854164415499578997880960372747910265822600311586384743474659328) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
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

/-- Final stack for bytecode block summary `ethbridge_block_647_taken`. -/
def ethbridge_block_647_taken_stack {mem : ByteArray} {σ : AccountMap} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero (extCodeSizeWord σ (UInt256.ofNat 376793390874373408599387495934666716005045108773))) :: (UInt256.ofNat 376793390874373408599387495934666716005045108773) :: (UInt256.ofNat 0) :: (memLoad (UInt256.ofNat 64) (x2.toByteArray.write 0 ((UInt256.ofNat 72570022874062638528011751457397263716769196454539065078543251854057308946432).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)).toNat 32)) :: (UInt256.sub ((UInt256.ofNat 36) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (UInt256.ofNat 64) (x2.toByteArray.write 0 ((UInt256.ofNat 72570022874062638528011751457397263716769196454539065078543251854057308946432).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)).toNat 32))) :: (memLoad (UInt256.ofNat 64) (x2.toByteArray.write 0 ((UInt256.ofNat 72570022874062638528011751457397263716769196454539065078543251854057308946432).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)).toNat 32)) :: (UInt256.ofNat 0) :: ((UInt256.ofNat 36) + (memLoad (UInt256.ofNat 64) mem)) :: (UInt256.ofNat 2691771752) :: (UInt256.ofNat 376793390874373408599387495934666716005045108773) :: x0 :: x1 :: x2 :: R)

/-- Final memory for bytecode block summary `ethbridge_block_647_taken`. -/
def ethbridge_block_647_taken_memory {mem : ByteArray} {x2 : UInt256} : ByteArray :=
  (x2.toByteArray.write 0 ((UInt256.ofNat 72570022874062638528011751457397263716769196454539065078543251854057308946432).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 647. -/
theorem ethbridge_block_647_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 15 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero (extCodeSizeWord σ (UInt256.ofNat 376793390874373408599387495934666716005045108773)))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 749) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 647) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 749) (ethbridge_block_647_taken_stack (mem := mem) (σ := σ) (x0 := x0) (x1 := x1) (x2 := x2) (R := R)) (ethbridge_block_647_taken_memory (mem := mem) (x2 := x2)) (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r3 := RD.genMload r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.pushConst (UInt256.ofNat 72570022874062638528011751457397263716769196454539065078543251854057308946432) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup2 (by evm_kdecide) (by evm_ov)
  have r6 := RD.genMstore r5 (by evm_kdecide) (by evm_ov)
  have r7 := r6.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup2 (by evm_kdecide) (by evm_ov)
  have r9 := r8.add (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup5 (by evm_kdecide) (by evm_ov)
  have r11 := r10.swap1 (by evm_kdecide) (by evm_ov)
  have r12 := RD.genMstore r11 (by evm_kdecide) (by evm_ov)
  have r13 := r12.push20 (UInt256.ofNat 376793390874373408599387495934666716005045108773) (by evm_kdecide) (by evm_ov)
  have r14 := r13.swap1 (by evm_kdecide) (by evm_ov)
  have r15 := r14.push4 (UInt256.ofNat 2691771752) (by evm_kdecide) (by evm_ov)
  have r16 := r15.swap1 (by evm_kdecide) (by evm_ov)
  have r17 := r16.push1 (UInt256.ofNat 36) (by evm_kdecide) (by evm_ov)
  have r18 := r17.add (by evm_kdecide) (by evm_ov)
  have r19 := r18.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r20 := r19.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r21 := RD.genMload r20 (by evm_kdecide) (by evm_ov)
  have r22 := r21.dup1 (by evm_kdecide) (by evm_ov)
  have r23 := r22.dup4 (by evm_kdecide) (by evm_ov)
  have r24 := r23.sub (by evm_kdecide) (by evm_ov)
  have r25 := r24.dup2 (by evm_kdecide) (by evm_ov)
  have r26 := r25.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r27 := r26.dup8 (by evm_kdecide) (by evm_ov)
  have r28 := r27.dup1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r29⟩ := r28.extcodesize (by evm_kdecide) (by evm_ov)
  have r30 := r29.iszero (by evm_kdecide) (by evm_ov)
  have r31 := r30.dup1 (by evm_kdecide) (by evm_ov)
  have r32 := r31.iszero (by evm_kdecide) (by evm_ov)
  have r33 := r32.push2 (UInt256.ofNat 749) (by evm_kdecide) (by evm_ov)
  have r34 := r33.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 749)) r34 (by evm_kdecide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_647_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 15 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero (extCodeSizeWord σ (UInt256.ofNat 376793390874373408599387495934666716005045108773)))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 749) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 647) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 749) (ethbridge_block_647_taken_stack (mem := mem) (σ := σ) (x0 := x0) (x1 := x1) (x2 := x2) (R := R)) (ethbridge_block_647_taken_memory (mem := mem) (x2 := x2)) aw' rdata σ k' C' := by
  obtain ⟨k0, C0, h0⟩ := ethbridge_block_647_taken hstack hcond hvalid h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_647_fallthrough`. -/
def ethbridge_block_647_fallthrough_stack {mem : ByteArray} {σ : AccountMap} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero (extCodeSizeWord σ (UInt256.ofNat 376793390874373408599387495934666716005045108773))) :: (UInt256.ofNat 376793390874373408599387495934666716005045108773) :: (UInt256.ofNat 0) :: (memLoad (UInt256.ofNat 64) (x2.toByteArray.write 0 ((UInt256.ofNat 72570022874062638528011751457397263716769196454539065078543251854057308946432).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)).toNat 32)) :: (UInt256.sub ((UInt256.ofNat 36) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (UInt256.ofNat 64) (x2.toByteArray.write 0 ((UInt256.ofNat 72570022874062638528011751457397263716769196454539065078543251854057308946432).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)).toNat 32))) :: (memLoad (UInt256.ofNat 64) (x2.toByteArray.write 0 ((UInt256.ofNat 72570022874062638528011751457397263716769196454539065078543251854057308946432).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)).toNat 32)) :: (UInt256.ofNat 0) :: ((UInt256.ofNat 36) + (memLoad (UInt256.ofNat 64) mem)) :: (UInt256.ofNat 2691771752) :: (UInt256.ofNat 376793390874373408599387495934666716005045108773) :: x0 :: x1 :: x2 :: R)

/-- Final memory for bytecode block summary `ethbridge_block_647_fallthrough`. -/
def ethbridge_block_647_fallthrough_memory {mem : ByteArray} {x2 : UInt256} : ByteArray :=
  (x2.toByteArray.write 0 ((UInt256.ofNat 72570022874062638528011751457397263716769196454539065078543251854057308946432).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 647. -/
theorem ethbridge_block_647_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 15 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero (extCodeSizeWord σ (UInt256.ofNat 376793390874373408599387495934666716005045108773)))) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 647) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 745) (ethbridge_block_647_fallthrough_stack (mem := mem) (σ := σ) (x0 := x0) (x1 := x1) (x2 := x2) (R := R)) (ethbridge_block_647_fallthrough_memory (mem := mem) (x2 := x2)) (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r3 := RD.genMload r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.pushConst (UInt256.ofNat 72570022874062638528011751457397263716769196454539065078543251854057308946432) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup2 (by evm_kdecide) (by evm_ov)
  have r6 := RD.genMstore r5 (by evm_kdecide) (by evm_ov)
  have r7 := r6.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup2 (by evm_kdecide) (by evm_ov)
  have r9 := r8.add (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup5 (by evm_kdecide) (by evm_ov)
  have r11 := r10.swap1 (by evm_kdecide) (by evm_ov)
  have r12 := RD.genMstore r11 (by evm_kdecide) (by evm_ov)
  have r13 := r12.push20 (UInt256.ofNat 376793390874373408599387495934666716005045108773) (by evm_kdecide) (by evm_ov)
  have r14 := r13.swap1 (by evm_kdecide) (by evm_ov)
  have r15 := r14.push4 (UInt256.ofNat 2691771752) (by evm_kdecide) (by evm_ov)
  have r16 := r15.swap1 (by evm_kdecide) (by evm_ov)
  have r17 := r16.push1 (UInt256.ofNat 36) (by evm_kdecide) (by evm_ov)
  have r18 := r17.add (by evm_kdecide) (by evm_ov)
  have r19 := r18.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r20 := r19.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r21 := RD.genMload r20 (by evm_kdecide) (by evm_ov)
  have r22 := r21.dup1 (by evm_kdecide) (by evm_ov)
  have r23 := r22.dup4 (by evm_kdecide) (by evm_ov)
  have r24 := r23.sub (by evm_kdecide) (by evm_ov)
  have r25 := r24.dup2 (by evm_kdecide) (by evm_ov)
  have r26 := r25.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r27 := r26.dup8 (by evm_kdecide) (by evm_ov)
  have r28 := r27.dup1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r29⟩ := r28.extcodesize (by evm_kdecide) (by evm_ov)
  have r30 := r29.iszero (by evm_kdecide) (by evm_ov)
  have r31 := r30.dup1 (by evm_kdecide) (by evm_ov)
  have r32 := r31.iszero (by evm_kdecide) (by evm_ov)
  have r33 := r32.push2 (UInt256.ofNat 749) (by evm_kdecide) (by evm_ov)
  have r34 := r33.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 745)) r34 (by evm_kdecide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_647_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 15 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero (extCodeSizeWord σ (UInt256.ofNat 376793390874373408599387495934666716005045108773)))) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 647) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 745) (ethbridge_block_647_fallthrough_stack (mem := mem) (σ := σ) (x0 := x0) (x1 := x1) (x2 := x2) (R := R)) (ethbridge_block_647_fallthrough_memory (mem := mem) (x2 := x2)) aw' rdata σ k' C' := by
  obtain ⟨k0, C0, h0⟩ := ethbridge_block_647_fallthrough hstack hcond h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 745. -/
theorem ethbridge_block_745 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 745) R mem aw rdata σ k C)
    : RDrev BridgeEvm.ethbridgeRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `ethbridge_block_749`. -/
def ethbridge_block_749_stack {g : Sat256} {C : ℕ} {R : List UInt256} : List UInt256 :=
  (((g.subNat (C + ((3)) + 2)).toUInt256) :: R)

/-- Automatically generated RD summary for bytecode block at pc 749. -/
theorem ethbridge_block_749 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 1 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 749) (x0 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 752) (ethbridge_block_749_stack (g := g) (C := C) (R := R)) mem aw rdata σ (k + 3) (C + ((5))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := RD.genGas (RD.normalizeCounters (k' := k + 2) (C' := C + ((3))) r2 (by omega) (by omega)) (by evm_kdecide) (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 752)) r3 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_749_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 1 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 749) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 752) (ethbridge_block_749_stack (g := g) (C := C) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_749 hstack h)
  exact ⟨_, k', C', h'⟩

/- Unsupported instruction boundary at pc 752: call (0xf1). No RD transition is asserted. Summaries resume at pc 753 from a fresh symbolic RD state. -/

/-- Final stack for bytecode block summary `ethbridge_block_753_taken`. -/
def ethbridge_block_753_taken_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 753. -/
theorem ethbridge_block_753_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 769) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 753) (x0 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 769) (ethbridge_block_753_taken_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.iszero (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.iszero (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 769) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 769)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_753_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 769) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 753) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 769) (ethbridge_block_753_taken_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_753_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_753_fallthrough`. -/
def ethbridge_block_753_fallthrough_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 753. -/
theorem ethbridge_block_753_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 753) (x0 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 760) (ethbridge_block_753_fallthrough_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.iszero (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.iszero (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 769) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 760)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_753_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 753) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 760) (ethbridge_block_753_fallthrough_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_753_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 760. -/
theorem ethbridge_block_760 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 760) R mem aw rdata σ k C)
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

end ethbridgeBlocks
