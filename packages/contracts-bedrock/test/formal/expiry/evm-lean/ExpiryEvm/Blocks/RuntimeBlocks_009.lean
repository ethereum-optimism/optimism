import Reasoning.Reach
import ExpiryEvm.Bytecode

open Solm ABI Ethereum Ethereum.EVM
open Reasoning.Theory Reasoning.Reach

namespace l2tol2Blocks

/-- Final stack for bytecode block summary `l2tol2_block_2422_fallthrough`. -/
def l2tol2_block_2422_fallthrough_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 2422. -/
theorem l2tol2_block_2422_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2422) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2429) (l2tol2_block_2422_fallthrough_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.iszero (by native_decide) (by evm_ov)
  have r2 := r1.dup1 (by native_decide) (by evm_ov)
  have r3 := r2.iszero (by native_decide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 2436) (by native_decide) (by evm_ov)
  have r5 := r4.jumpiNT (by native_decide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2429)) r5 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_2422_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2422) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2429) (l2tol2_block_2422_fallthrough_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_2422_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 2429. -/
theorem l2tol2_block_2429 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2429) R mem aw rdata σ k C)
    : RDrev ExpiryEvm.l2tol2Runtime g s0 := by
  let r0 := h
  have r1 := r0.returndatasize (by native_decide) (by evm_ov)
  have r2 := r1.push0 (by native_decide) (by evm_ov)
  have r3 := r2.dup1 (by native_decide) (by evm_ov)
  have r4 := RD.genReturndatacopy r3 (by native_decide) (by
    have hz : UInt256.ofNat 0 = (⟨0⟩ : UInt256) := by decide
    simpa only [hz] using returnDataCopyFullGuard rdata) (by evm_ov)
  have r5 := r4.returndatasize (by native_decide) (by evm_ov)
  have r6 := r5.push0 (by native_decide) (by evm_ov)
  exact RD.genRev r6 (by native_decide) (by evm_ov)

/-- Final stack for bytecode block summary `l2tol2_block_2436`. -/
def l2tol2_block_2436_stack {mem : ByteArray} {rdata : ByteArray} {R : List UInt256} : List UInt256 :=
  ((memLoad (UInt256.ofNat 64) mem) :: ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat rdata.size)) :: (UInt256.ofNat 2472) :: R)

/-- Final memory for bytecode block summary `l2tol2_block_2436`. -/
def l2tol2_block_2436_memory {mem : ByteArray} {rdata : ByteArray} : ByteArray :=
  (((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat rdata.size) + (UInt256.ofNat 31)) (UInt256.lnot (UInt256.ofNat 31)))).toByteArray.write 0 mem (UInt256.ofNat 64).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 2436. -/
theorem l2tol2_block_2436 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 5266) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2436) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5266) (l2tol2_block_2436_stack (mem := mem) (rdata := rdata) (R := R)) (l2tol2_block_2436_memory (mem := mem) (rdata := rdata)) (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 28) (C + ((81) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.pop (by native_decide) (by evm_ov)
  have r3 := r2.pop (by native_decide) (by evm_ov)
  have r4 := r3.pop (by native_decide) (by evm_ov)
  have r5 := r4.pop (by native_decide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 64) (by native_decide) (by evm_ov)
  have r7 := RD.genMload r6 (by native_decide) (by evm_ov)
  have r8 := r7.returndatasize (by native_decide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 31) (by native_decide) (by evm_ov)
  have r10 := r9.not (by native_decide) (by evm_ov)
  have r11 := r10.push1 (UInt256.ofNat 31) (by native_decide) (by evm_ov)
  have r12 := r11.dup3 (by native_decide) (by evm_ov)
  have r13 := r12.add (by native_decide) (by evm_ov)
  have r14 := r13.and (by native_decide) (by evm_ov)
  have r15 := r14.dup3 (by native_decide) (by evm_ov)
  have r16 := r15.add (by native_decide) (by evm_ov)
  have r17 := r16.dup1 (by native_decide) (by evm_ov)
  have r18 := r17.push1 (UInt256.ofNat 64) (by native_decide) (by evm_ov)
  have r19 := RD.genMstore r18 (by native_decide) (by evm_ov)
  have r20 := r19.pop (by native_decide) (by evm_ov)
  have r21 := r20.dup2 (by native_decide) (by evm_ov)
  have r22 := r21.add (by native_decide) (by evm_ov)
  have r23 := r22.swap1 (by native_decide) (by evm_ov)
  have r24 := r23.push2 (UInt256.ofNat 2472) (by native_decide) (by evm_ov)
  have r25 := r24.swap2 (by native_decide) (by evm_ov)
  have r26 := r25.swap1 (by native_decide) (by evm_ov)
  have r27 := r26.push2 (UInt256.ofNat 5266) (by native_decide) (by evm_ov)
  have r28 := r27.jump (by native_decide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 5266)) r28 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_2436_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 5266) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2436) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5266) (l2tol2_block_2436_stack (mem := mem) (rdata := rdata) (R := R)) (l2tol2_block_2436_memory (mem := mem) (rdata := rdata)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_2436 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_2472`. -/
def l2tol2_block_2472_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero (UInt256.eq (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) x0) x1)) :: R)

/-- Automatically generated RD summary for bytecode block at pc 2472. -/
theorem l2tol2_block_2472 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2472) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2497) (l2tol2_block_2472_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 5) (C + ((13))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by native_decide) (by evm_ov)
  have r3 := r2.and (by native_decide) (by evm_ov)
  have r4 := r3.eq (by native_decide) (by evm_ov)
  have r5 := r4.iszero (by native_decide) (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2497)) r5 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_2472_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2472) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2497) (l2tol2_block_2472_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_2472 hstack h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_2497_taken`. -/
def l2tol2_block_2497_taken_stack {R : List UInt256} : List UInt256 :=
  R

/-- Automatically generated RD summary for bytecode block at pc 2497. -/
theorem l2tol2_block_2497_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (UInt256.isZero x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 2552) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2497) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2552) (l2tol2_block_2497_taken_stack (R := R)) mem aw rdata σ (k + 4) (C + ((17))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.iszero (by native_decide) (by evm_ov)
  have r3 := r2.push2 (UInt256.ofNat 2552) (by native_decide) (by evm_ov)
  have r4 := r3.jumpiT (by native_decide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2552)) r4 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_2497_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (UInt256.isZero x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 2552) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2497) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2552) (l2tol2_block_2497_taken_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_2497_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_2497_fallthrough`. -/
def l2tol2_block_2497_fallthrough_stack {R : List UInt256} : List UInt256 :=
  R

/-- Automatically generated RD summary for bytecode block at pc 2497. -/
theorem l2tol2_block_2497_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (UInt256.isZero x0) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2497) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2503) (l2tol2_block_2497_fallthrough_stack (R := R)) mem aw rdata σ (k + 4) (C + ((17))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.iszero (by native_decide) (by evm_ov)
  have r3 := r2.push2 (UInt256.ofNat 2552) (by native_decide) (by evm_ov)
  have r4 := r3.jumpiNT (by native_decide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2503)) r4 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_2497_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (UInt256.isZero x0) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2497) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2503) (l2tol2_block_2497_fallthrough_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_2497_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 2503. -/
theorem l2tol2_block_2503 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2503) R mem aw rdata σ k C)
    : RDrev ExpiryEvm.l2tol2Runtime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 64) (by native_decide) (by evm_ov)
  have r2 := RD.genMload r1 (by native_decide) (by evm_ov)
  have r3 := r2.pushConst (UInt256.ofNat 93545514728453368016867473155542155214588639737833566317075367692838136446976) (width := 32) (op := .PUSH32) (by decide) (by native_decide) (by evm_ov)
  have r4 := r3.dup2 (by native_decide) (by evm_ov)
  have r5 := RD.genMstore r4 (by native_decide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 4) (by native_decide) (by evm_ov)
  have r7 := r6.add (by native_decide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 64) (by native_decide) (by evm_ov)
  have r9 := RD.genMload r8 (by native_decide) (by evm_ov)
  have r10 := r9.dup1 (by native_decide) (by evm_ov)
  have r11 := r10.swap2 (by native_decide) (by evm_ov)
  have r12 := r11.sub (by native_decide) (by evm_ov)
  have r13 := r12.swap1 (by native_decide) (by evm_ov)
  exact RD.genRev r13 (by native_decide) (by evm_ov)

/-- Final stack for bytecode block summary `l2tol2_block_2552_taken`. -/
def l2tol2_block_2552_taken_stack {ee : ExecutionEnv} {mem : ByteArray} {σ : AccountMap} {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (⟨0⟩ : UInt256) (UInt256.ofNat 64) ((UInt256.ofNat 3).toByteArray.write 0 (x1.toByteArray.write 0 mem (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256))) :: x0 :: x1 :: R)

/-- Final memory for bytecode block summary `l2tol2_block_2552_taken`. -/
def l2tol2_block_2552_taken_memory {mem : ByteArray} {x1 : UInt256} : ByteArray :=
  ((UInt256.ofNat 3).toByteArray.write 0 (x1.toByteArray.write 0 mem (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 2552. -/
theorem l2tol2_block_2552_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.sub (⟨0⟩ : UInt256) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (⟨0⟩ : UInt256) (UInt256.ofNat 64) ((UInt256.ofNat 3).toByteArray.write 0 (x1.toByteArray.write 0 mem (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256)))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 2624) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2552) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2624) (l2tol2_block_2552_taken_stack (ee := ee) (mem := mem) (σ := σ) (x0 := x0) (x1 := x1) (R := R)) (l2tol2_block_2552_taken_memory (mem := mem) (x1 := x1)) (M (M (M aw (⟨0⟩ : UInt256) (⟨32⟩ : UInt256)) (UInt256.ofNat 32) (⟨32⟩ : UInt256)) (⟨0⟩ : UInt256) (UInt256.ofNat 64)) rdata σ k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.push0 (by native_decide) (by evm_ov)
  have r3 := r2.dup3 (by native_decide) (by evm_ov)
  have r4 := r3.dup2 (by native_decide) (by evm_ov)
  have r5 := RD.genMstore r4 (by native_decide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 3) (by native_decide) (by evm_ov)
  have r7 := r6.push1 (UInt256.ofNat 32) (by native_decide) (by evm_ov)
  have r8 := RD.genMstore r7 (by native_decide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 64) (by native_decide) (by evm_ov)
  have r10 := r9.dup2 (by native_decide) (by evm_ov)
  have r11 := RD.genKeccak256 r10 (by native_decide) (by evm_ov)
  obtain ⟨_, _, r12⟩ := RD.sload r11 (by native_decide) (by evm_ov)
  have r13 := r12.swap1 (by native_decide) (by evm_ov)
  have r14 := r13.dup2 (by native_decide) (by evm_ov)
  have r15 := r14.swap1 (by native_decide) (by evm_ov)
  have r16 := r15.sub (by native_decide) (by evm_ov)
  have r17 := r16.push2 (UInt256.ofNat 2624) (by native_decide) (by evm_ov)
  have r18 := r17.jumpiT (by native_decide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2624)) r18 (by native_decide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_2552_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.sub (⟨0⟩ : UInt256) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (⟨0⟩ : UInt256) (UInt256.ofNat 64) ((UInt256.ofNat 3).toByteArray.write 0 (x1.toByteArray.write 0 mem (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256)))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 2624) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2552) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2624) (l2tol2_block_2552_taken_stack (ee := ee) (mem := mem) (σ := σ) (x0 := x0) (x1 := x1) (R := R)) (l2tol2_block_2552_taken_memory (mem := mem) (x1 := x1)) aw' rdata σ k' C' := by
  obtain ⟨k0, C0, h0⟩ := l2tol2_block_2552_taken hstack hcond hvalid h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_2552_fallthrough`. -/
def l2tol2_block_2552_fallthrough_stack {ee : ExecutionEnv} {mem : ByteArray} {σ : AccountMap} {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (⟨0⟩ : UInt256) (UInt256.ofNat 64) ((UInt256.ofNat 3).toByteArray.write 0 (x1.toByteArray.write 0 mem (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256))) :: x0 :: x1 :: R)

/-- Final memory for bytecode block summary `l2tol2_block_2552_fallthrough`. -/
def l2tol2_block_2552_fallthrough_memory {mem : ByteArray} {x1 : UInt256} : ByteArray :=
  ((UInt256.ofNat 3).toByteArray.write 0 (x1.toByteArray.write 0 mem (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 2552. -/
theorem l2tol2_block_2552_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.sub (⟨0⟩ : UInt256) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (⟨0⟩ : UInt256) (UInt256.ofNat 64) ((UInt256.ofNat 3).toByteArray.write 0 (x1.toByteArray.write 0 mem (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256)))) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2552) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2575) (l2tol2_block_2552_fallthrough_stack (ee := ee) (mem := mem) (σ := σ) (x0 := x0) (x1 := x1) (R := R)) (l2tol2_block_2552_fallthrough_memory (mem := mem) (x1 := x1)) (M (M (M aw (⟨0⟩ : UInt256) (⟨32⟩ : UInt256)) (UInt256.ofNat 32) (⟨32⟩ : UInt256)) (⟨0⟩ : UInt256) (UInt256.ofNat 64)) rdata σ k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.push0 (by native_decide) (by evm_ov)
  have r3 := r2.dup3 (by native_decide) (by evm_ov)
  have r4 := r3.dup2 (by native_decide) (by evm_ov)
  have r5 := RD.genMstore r4 (by native_decide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 3) (by native_decide) (by evm_ov)
  have r7 := r6.push1 (UInt256.ofNat 32) (by native_decide) (by evm_ov)
  have r8 := RD.genMstore r7 (by native_decide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 64) (by native_decide) (by evm_ov)
  have r10 := r9.dup2 (by native_decide) (by evm_ov)
  have r11 := RD.genKeccak256 r10 (by native_decide) (by evm_ov)
  obtain ⟨_, _, r12⟩ := RD.sload r11 (by native_decide) (by evm_ov)
  have r13 := r12.swap1 (by native_decide) (by evm_ov)
  have r14 := r13.dup2 (by native_decide) (by evm_ov)
  have r15 := r14.swap1 (by native_decide) (by evm_ov)
  have r16 := r15.sub (by native_decide) (by evm_ov)
  have r17 := r16.push2 (UInt256.ofNat 2624) (by native_decide) (by evm_ov)
  have r18 := r17.jumpiNT (by native_decide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2575)) r18 (by native_decide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_2552_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.sub (⟨0⟩ : UInt256) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (⟨0⟩ : UInt256) (UInt256.ofNat 64) ((UInt256.ofNat 3).toByteArray.write 0 (x1.toByteArray.write 0 mem (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256)))) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2552) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2575) (l2tol2_block_2552_fallthrough_stack (ee := ee) (mem := mem) (σ := σ) (x0 := x0) (x1 := x1) (R := R)) (l2tol2_block_2552_fallthrough_memory (mem := mem) (x1 := x1)) aw' rdata σ k' C' := by
  obtain ⟨k0, C0, h0⟩ := l2tol2_block_2552_fallthrough hstack hcond h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 2575. -/
theorem l2tol2_block_2575 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2575) R mem aw rdata σ k C)
    : RDrev ExpiryEvm.l2tol2Runtime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 64) (by native_decide) (by evm_ov)
  have r2 := RD.genMload r1 (by native_decide) (by evm_ov)
  have r3 := r2.pushConst (UInt256.ofNat 50111635953572488757873690314866451215542211556984856418569468639946345545728) (width := 32) (op := .PUSH32) (by decide) (by native_decide) (by evm_ov)
  have r4 := r3.dup2 (by native_decide) (by evm_ov)
  have r5 := RD.genMstore r4 (by native_decide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 4) (by native_decide) (by evm_ov)
  have r7 := r6.add (by native_decide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 64) (by native_decide) (by evm_ov)
  have r9 := RD.genMload r8 (by native_decide) (by evm_ov)
  have r10 := r9.dup1 (by native_decide) (by evm_ov)
  have r11 := r10.swap2 (by native_decide) (by evm_ov)
  have r12 := r11.sub (by native_decide) (by evm_ov)
  have r13 := r12.swap1 (by native_decide) (by evm_ov)
  exact RD.genRev r13 (by native_decide) (by evm_ov)

/-- Final stack for bytecode block summary `l2tol2_block_2624`. -/
def l2tol2_block_2624_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: (UInt256.ofNat 604800) :: (UInt256.ofNat 2637) :: x0 :: R)

/-- Automatically generated RD summary for bytecode block at pc 2624. -/
theorem l2tol2_block_2624 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 5293) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2624) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5293) (l2tol2_block_2624_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 6) (C + ((21))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 2637) (by native_decide) (by evm_ov)
  have r3 := r2.pushConst (UInt256.ofNat 604800) (width := 3) (op := .PUSH3) (by decide) (by native_decide) (by evm_ov)
  have r4 := r3.dup3 (by native_decide) (by evm_ov)
  have r5 := r4.push2 (UInt256.ofNat 5293) (by native_decide) (by evm_ov)
  have r6 := r5.jump (by native_decide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 5293)) r6 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_2624_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 5293) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2624) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5293) (l2tol2_block_2624_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_2624 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_2637_taken`. -/
def l2tol2_block_2637_taken_stack {x1 : UInt256} {x2 : UInt256} {R : List UInt256} : List UInt256 :=
  (x1 :: x2 :: R)

/-- Automatically generated RD summary for bytecode block at pc 2637. -/
theorem l2tol2_block_2637_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.gt x2 x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 2693) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2637) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2693) (l2tol2_block_2637_taken_stack (x1 := x1) (x2 := x2) (R := R)) mem aw rdata σ (k + 5) (C + ((20))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.dup3 (by native_decide) (by evm_ov)
  have r3 := r2.gt (by native_decide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 2693) (by native_decide) (by evm_ov)
  have r5 := r4.jumpiT (by native_decide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2693)) r5 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_2637_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.gt x2 x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 2693) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2637) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2693) (l2tol2_block_2637_taken_stack (x1 := x1) (x2 := x2) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_2637_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_2637_fallthrough`. -/
def l2tol2_block_2637_fallthrough_stack {x1 : UInt256} {x2 : UInt256} {R : List UInt256} : List UInt256 :=
  (x1 :: x2 :: R)

/-- Automatically generated RD summary for bytecode block at pc 2637. -/
theorem l2tol2_block_2637_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.gt x2 x0) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2637) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2644) (l2tol2_block_2637_fallthrough_stack (x1 := x1) (x2 := x2) (R := R)) mem aw rdata σ (k + 5) (C + ((20))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.dup3 (by native_decide) (by evm_ov)
  have r3 := r2.gt (by native_decide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 2693) (by native_decide) (by evm_ov)
  have r5 := r4.jumpiNT (by native_decide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2644)) r5 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_2637_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (UInt256.gt x2 x0) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2637) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2644) (l2tol2_block_2637_fallthrough_stack (x1 := x1) (x2 := x2) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_2637_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 2644. -/
theorem l2tol2_block_2644 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2644) R mem aw rdata σ k C)
    : RDrev ExpiryEvm.l2tol2Runtime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 64) (by native_decide) (by evm_ov)
  have r2 := RD.genMload r1 (by native_decide) (by evm_ov)
  have r3 := r2.pushConst (UInt256.ofNat 18074760117491723492711543516830290912764447529838340178131556265193451290624) (width := 32) (op := .PUSH32) (by decide) (by native_decide) (by evm_ov)
  have r4 := r3.dup2 (by native_decide) (by evm_ov)
  have r5 := RD.genMstore r4 (by native_decide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 4) (by native_decide) (by evm_ov)
  have r7 := r6.add (by native_decide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 64) (by native_decide) (by evm_ov)
  have r9 := RD.genMload r8 (by native_decide) (by evm_ov)
  have r10 := r9.dup1 (by native_decide) (by evm_ov)
  have r11 := r10.swap2 (by native_decide) (by evm_ov)
  have r12 := r11.sub (by native_decide) (by evm_ov)
  have r13 := r12.swap1 (by native_decide) (by evm_ov)
  exact RD.genRev r13 (by native_decide) (by evm_ov)

/-- Final stack for bytecode block summary `l2tol2_block_2693`. -/
def l2tol2_block_2693_stack {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {R : List UInt256} : List UInt256 :=
  (((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) ((UInt256.ofNat 4).toByteArray.write 0 (x2.toByteArray.write 0 mem (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32))) :: (UInt256.ofNat 13985141918288769522110054263011045850217665415068678124743895891648686834477) :: x2 :: x0 :: x1 :: x2 :: R)

/-- Final memory for bytecode block summary `l2tol2_block_2693`. -/
def l2tol2_block_2693_memory {mem : ByteArray} {x1 : UInt256} {x2 : UInt256} : ByteArray :=
  (x1.toByteArray.write 0 ((UInt256.ofNat 4).toByteArray.write 0 (x2.toByteArray.write 0 mem (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat 4).toByteArray.write 0 (x2.toByteArray.write 0 mem (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 2693. -/
theorem l2tol2_block_2693 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hperm : ee.perm = true)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 2799) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2693) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2799) (l2tol2_block_2693_stack (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2) (R := R)) (l2tol2_block_2693_memory (mem := mem) (x1 := x1) (x2 := x2)) (M (M (M (M (M aw (⟨0⟩ : UInt256) (⟨32⟩ : UInt256)) (UInt256.ofNat 32) (⟨32⟩ : UInt256)) (⟨0⟩ : UInt256) (UInt256.ofNat 64)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat 4).toByteArray.write 0 (x2.toByteArray.write 0 mem (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨32⟩ : UInt256)) rdata (sstoreAccountMap ee.codeOwner σ (keccakWord (⟨0⟩ : UInt256) (UInt256.ofNat 64) ((UInt256.ofNat 4).toByteArray.write 0 (x2.toByteArray.write 0 mem (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32)) (UInt256.lor (UInt256.ofNat 1) (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639680) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (⟨0⟩ : UInt256) (UInt256.ofNat 64) ((UInt256.ofNat 4).toByteArray.write 0 (x2.toByteArray.write 0 mem (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256)))))) k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.push0 (by native_decide) (by evm_ov)
  have r3 := r2.dup4 (by native_decide) (by evm_ov)
  have r4 := r3.dup2 (by native_decide) (by evm_ov)
  have r5 := RD.genMstore r4 (by native_decide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 4) (by native_decide) (by evm_ov)
  have r7 := r6.push1 (UInt256.ofNat 32) (by native_decide) (by evm_ov)
  have r8 := RD.genMstore r7 (by native_decide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 64) (by native_decide) (by evm_ov)
  have r10 := r9.swap1 (by native_decide) (by evm_ov)
  have r11 := r10.dup2 (by native_decide) (by evm_ov)
  have r12 := r11.swap1 (by native_decide) (by evm_ov)
  have r13 := RD.genKeccak256 r12 (by native_decide) (by evm_ov)
  have r14 := r13.dup1 (by native_decide) (by evm_ov)
  obtain ⟨_, _, r15⟩ := RD.sload r14 (by native_decide) (by evm_ov)
  have r16 := r15.pushConst (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639680) (width := 32) (op := .PUSH32) (by decide) (by native_decide) (by evm_ov)
  have r17 := r16.and (by native_decide) (by evm_ov)
  have r18 := r17.push1 (UInt256.ofNat 1) (by native_decide) (by evm_ov)
  have r19 := r18.or (by native_decide) (by evm_ov)
  have r20 := r19.swap1 (by native_decide) (by evm_ov)
  obtain ⟨_, _, r21⟩ := RD.sstore r20 hperm (by native_decide) (by evm_ov)
  have r22 := RD.genMload r21 (by native_decide) (by evm_ov)
  have r23 := r22.dup4 (by native_decide) (by evm_ov)
  have r24 := r23.swap1 (by native_decide) (by evm_ov)
  have r25 := r24.pushConst (UInt256.ofNat 13985141918288769522110054263011045850217665415068678124743895891648686834477) (width := 32) (op := .PUSH32) (by decide) (by native_decide) (by evm_ov)
  have r26 := r25.swap1 (by native_decide) (by evm_ov)
  have r27 := r26.push2 (UInt256.ofNat 2799) (by native_decide) (by evm_ov)
  have r28 := r27.swap1 (by native_decide) (by evm_ov)
  have r29 := r28.dup6 (by native_decide) (by evm_ov)
  have r30 := r29.dup2 (by native_decide) (by evm_ov)
  have r31 := RD.genMstore r30 (by native_decide) (by evm_ov)
  have r32 := r31.push1 (UInt256.ofNat 32) (by native_decide) (by evm_ov)
  have r33 := r32.add (by native_decide) (by evm_ov)
  have r34 := r33.swap1 (by native_decide) (by evm_ov)
  have r35 := r34.jump (by native_decide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2799)) r35 (by native_decide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_2693_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hperm : ee.perm = true)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 2799) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2693) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2799) (l2tol2_block_2693_stack (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2) (R := R)) (l2tol2_block_2693_memory (mem := mem) (x1 := x1) (x2 := x2)) aw' rdata (sstoreAccountMap ee.codeOwner σ (keccakWord (⟨0⟩ : UInt256) (UInt256.ofNat 64) ((UInt256.ofNat 4).toByteArray.write 0 (x2.toByteArray.write 0 mem (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32)) (UInt256.lor (UInt256.ofNat 1) (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639680) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (⟨0⟩ : UInt256) (UInt256.ofNat 64) ((UInt256.ofNat 4).toByteArray.write 0 (x2.toByteArray.write 0 mem (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256)))))) k' C' := by
  obtain ⟨k0, C0, h0⟩ := l2tol2_block_2693 hstack hperm hvalid h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_2799`. -/
def l2tol2_block_2799_stack {R : List UInt256} : List UInt256 :=
  R

/-- Automatically generated RD summary for bytecode block at pc 2799. -/
theorem l2tol2_block_2799 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hperm : ee.perm = true)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x6 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2799) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 x6 (l2tol2_block_2799_stack (R := R)) mem (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem))) rdata σ (k + 12) (C + ((33) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem))) + (375 + 8 * (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem)).toNat + 2 * 375))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 64) (by native_decide) (by evm_ov)
  have r3 := RD.genMload r2 (by native_decide) (by evm_ov)
  have r4 := r3.dup1 (by native_decide) (by evm_ov)
  have r5 := r4.swap2 (by native_decide) (by evm_ov)
  have r6 := r5.sub (by native_decide) (by evm_ov)
  have r7 := r6.swap1 (by native_decide) (by evm_ov)
  have r8 := RD.genLog2 r7 (by native_decide) hperm (by evm_ov)
  have r9 := r8.pop (by native_decide) (by evm_ov)
  have r10 := r9.pop (by native_decide) (by evm_ov)
  have r11 := r10.pop (by native_decide) (by evm_ov)
  have r12 := r11.jump (by native_decide) hvalid (by evm_ov)
  exact RD.normalizeCounters r12 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_2799_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hperm : ee.perm = true)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x6 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2799) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 x6 (l2tol2_block_2799_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_2799 hstack hperm hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_2812_taken`. -/
def l2tol2_block_2812_taken_stack {R : List UInt256} : List UInt256 :=
  ((⟨0⟩ : UInt256) :: (⟨0⟩ : UInt256) :: R)

/-- Automatically generated RD summary for bytecode block at pc 2812. -/
theorem l2tol2_block_2812_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.tstorage.getD (UInt256.ofNat 109101767571825637468208191185038832016786111485836296546894784897825145752586) (⟨0⟩ : UInt256))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 2902) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2812) R mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2902) (l2tol2_block_2812_taken_stack (R := R)) mem aw rdata σ (k + 7) (C + ((22) + (Ctload))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.push0 (by native_decide) (by evm_ov)
  have r3 := r2.dup1 (by native_decide) (by evm_ov)
  have r4 := r3.pushConst (UInt256.ofNat 109101767571825637468208191185038832016786111485836296546894784897825145752586) (width := 32) (op := .PUSH32) (by decide) (by native_decide) (by evm_ov)
  have r5 := r4.tload (by native_decide) (by evm_ov)
  have r6 := r5.push2 (UInt256.ofNat 2902) (by native_decide) (by evm_ov)
  have r7 := r6.jumpiT (by native_decide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2902)) r7 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_2812_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.tstorage.getD (UInt256.ofNat 109101767571825637468208191185038832016786111485836296546894784897825145752586) (⟨0⟩ : UInt256))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 2902) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2812) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2902) (l2tol2_block_2812_taken_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_2812_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_2812_fallthrough`. -/
def l2tol2_block_2812_fallthrough_stack {R : List UInt256} : List UInt256 :=
  ((⟨0⟩ : UInt256) :: (⟨0⟩ : UInt256) :: R)

/-- Automatically generated RD summary for bytecode block at pc 2812. -/
theorem l2tol2_block_2812_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.tstorage.getD (UInt256.ofNat 109101767571825637468208191185038832016786111485836296546894784897825145752586) (⟨0⟩ : UInt256))) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2812) R mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2853) (l2tol2_block_2812_fallthrough_stack (R := R)) mem aw rdata σ (k + 7) (C + ((22) + (Ctload))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.push0 (by native_decide) (by evm_ov)
  have r3 := r2.dup1 (by native_decide) (by evm_ov)
  have r4 := r3.pushConst (UInt256.ofNat 109101767571825637468208191185038832016786111485836296546894784897825145752586) (width := 32) (op := .PUSH32) (by decide) (by native_decide) (by evm_ov)
  have r5 := r4.tload (by native_decide) (by evm_ov)
  have r6 := r5.push2 (UInt256.ofNat 2902) (by native_decide) (by evm_ov)
  have r7 := r6.jumpiNT (by native_decide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2853)) r7 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_2812_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hcond : (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.tstorage.getD (UInt256.ofNat 109101767571825637468208191185038832016786111485836296546894784897825145752586) (⟨0⟩ : UInt256))) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2812) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2853) (l2tol2_block_2812_fallthrough_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_2812_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 2853. -/
theorem l2tol2_block_2853 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2853) R mem aw rdata σ k C)
    : RDrev ExpiryEvm.l2tol2Runtime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 64) (by native_decide) (by evm_ov)
  have r2 := RD.genMload r1 (by native_decide) (by evm_ov)
  have r3 := r2.pushConst (UInt256.ofNat 85323439394531048929705278224489594386944891178565538386421280045290164846592) (width := 32) (op := .PUSH32) (by decide) (by native_decide) (by evm_ov)
  have r4 := r3.dup2 (by native_decide) (by evm_ov)
  have r5 := RD.genMstore r4 (by native_decide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 4) (by native_decide) (by evm_ov)
  have r7 := r6.add (by native_decide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 64) (by native_decide) (by evm_ov)
  have r9 := RD.genMload r8 (by native_decide) (by evm_ov)
  have r10 := r9.dup1 (by native_decide) (by evm_ov)
  have r11 := r10.swap2 (by native_decide) (by evm_ov)
  have r12 := r11.sub (by native_decide) (by evm_ov)
  have r13 := r12.swap1 (by native_decide) (by evm_ov)
  exact RD.genRev r13 (by native_decide) (by evm_ov)

/-- Final stack for bytecode block summary `l2tol2_block_2902`. -/
def l2tol2_block_2902_stack {ee : ExecutionEnv} {σ : AccountMap} {R : List UInt256} : List UInt256 :=
  ((σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.tstorage.getD (UInt256.ofNat 51164317248826882881343213066748959193860054907587800719701918628625062166247) (⟨0⟩ : UInt256))) :: (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.tstorage.getD (UInt256.ofNat 83317915124952138165281597070141736405785001779305661129532428155124339880947) (⟨0⟩ : UInt256))) :: R)

/-- Automatically generated RD summary for bytecode block at pc 2902. -/
theorem l2tol2_block_2902 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x2 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2902) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 x2 (l2tol2_block_2902_stack (ee := ee) (σ := σ) (R := R)) mem aw rdata σ (k + 10) (C + ((25) + (Ctload) + (Ctload))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.pop (by native_decide) (by evm_ov)
  have r3 := r2.pop (by native_decide) (by evm_ov)
  have r4 := r3.pushConst (UInt256.ofNat 83317915124952138165281597070141736405785001779305661129532428155124339880947) (width := 32) (op := .PUSH32) (by decide) (by native_decide) (by evm_ov)
  have r5 := r4.tload (by native_decide) (by evm_ov)
  have r6 := r5.swap1 (by native_decide) (by evm_ov)
  have r7 := r6.pushConst (UInt256.ofNat 51164317248826882881343213066748959193860054907587800719701918628625062166247) (width := 32) (op := .PUSH32) (by decide) (by native_decide) (by evm_ov)
  have r8 := r7.tload (by native_decide) (by evm_ov)
  have r9 := r8.swap1 (by native_decide) (by evm_ov)
  have r10 := r9.jump (by native_decide) hvalid (by evm_ov)
  exact RD.normalizeCounters r10 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_2902_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x2 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2902) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 x2 (l2tol2_block_2902_stack (ee := ee) (σ := σ) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_2902 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

end l2tol2Blocks
