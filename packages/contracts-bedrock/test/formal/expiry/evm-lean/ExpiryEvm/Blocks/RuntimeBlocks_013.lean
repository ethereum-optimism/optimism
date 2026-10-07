import Reasoning.Reach
import ExpiryEvm.Bytecode

open Solm ABI Ethereum Ethereum.EVM
open Reasoning.Theory Reasoning.Reach

namespace l2tol2Blocks

/-- Final stack for bytecode block summary `l2tol2_block_4422`. -/
def l2tol2_block_4422_stack {R : List UInt256} : List UInt256 :=
  R

/-- Automatically generated RD summary for bytecode block at pc 4422. -/
theorem l2tol2_block_4422 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x1 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4422) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 x1 (l2tol2_block_4422_stack (R := R)) mem aw rdata σ (k + 3) (C + ((11))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.pop (by native_decide) (by evm_ov)
  have r3 := r2.jump (by native_decide) hvalid (by evm_ov)
  exact RD.normalizeCounters r3 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_4422_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x1 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4422) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 x1 (l2tol2_block_4422_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_4422 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_4425_taken`. -/
def l2tol2_block_4425_taken_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((⟨0⟩ : UInt256) :: (⟨0⟩ : UInt256) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 4425. -/
theorem l2tol2_block_4425_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : (UInt256.slt (x0 + (UInt256.ofNat 31)) x1) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 4441) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4425) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4441) (l2tol2_block_4425_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 10) (C + ((34))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.push0 (by native_decide) (by evm_ov)
  have r3 := r2.dup1 (by native_decide) (by evm_ov)
  have r4 := r3.dup4 (by native_decide) (by evm_ov)
  have r5 := r4.push1 (UInt256.ofNat 31) (by native_decide) (by evm_ov)
  have r6 := r5.dup5 (by native_decide) (by evm_ov)
  have r7 := r6.add (by native_decide) (by evm_ov)
  have r8 := r7.slt (by native_decide) (by evm_ov)
  have r9 := r8.push2 (UInt256.ofNat 4441) (by native_decide) (by evm_ov)
  have r10 := r9.jumpiT (by native_decide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 4441)) r10 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_4425_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : (UInt256.slt (x0 + (UInt256.ofNat 31)) x1) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 4441) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4425) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4441) (l2tol2_block_4425_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_4425_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_4425_fallthrough`. -/
def l2tol2_block_4425_fallthrough_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((⟨0⟩ : UInt256) :: (⟨0⟩ : UInt256) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 4425. -/
theorem l2tol2_block_4425_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : (UInt256.slt (x0 + (UInt256.ofNat 31)) x1) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4425) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4438) (l2tol2_block_4425_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 10) (C + ((34))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.push0 (by native_decide) (by evm_ov)
  have r3 := r2.dup1 (by native_decide) (by evm_ov)
  have r4 := r3.dup4 (by native_decide) (by evm_ov)
  have r5 := r4.push1 (UInt256.ofNat 31) (by native_decide) (by evm_ov)
  have r6 := r5.dup5 (by native_decide) (by evm_ov)
  have r7 := r6.add (by native_decide) (by evm_ov)
  have r8 := r7.slt (by native_decide) (by evm_ov)
  have r9 := r8.push2 (UInt256.ofNat 4441) (by native_decide) (by evm_ov)
  have r10 := r9.jumpiNT (by native_decide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 4438)) r10 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_4425_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : (UInt256.slt (x0 + (UInt256.ofNat 31)) x1) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4425) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4438) (l2tol2_block_4425_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_4425_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 4438. -/
theorem l2tol2_block_4438 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4438) R mem aw rdata σ k C)
    : RDrev ExpiryEvm.l2tol2Runtime g s0 := by
  let r0 := h
  have r1 := r0.push0 (by native_decide) (by evm_ov)
  have r2 := r1.dup1 (by native_decide) (by evm_ov)
  exact RD.genRev r2 (by native_decide) (by evm_ov)

/-- Final stack for bytecode block summary `l2tol2_block_4441_taken`. -/
def l2tol2_block_4441_taken_stack {ee : ExecutionEnv} {x1 : UInt256} {x2 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes x2.toNat 32)) :: x1 :: x2 :: R)

/-- Automatically generated RD summary for bytecode block at pc 4441. -/
theorem l2tol2_block_4441_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes x2.toNat 32)) (UInt256.ofNat 18446744073709551615))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 4464) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4441) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4464) (l2tol2_block_4441_taken_stack (ee := ee) (x1 := x1) (x2 := x2) (R := R)) mem aw rdata σ (k + 10) (C + ((34))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.pop (by native_decide) (by evm_ov)
  have r3 := r2.dup2 (by native_decide) (by evm_ov)
  have r4 := r3.calldataload (by native_decide) (by evm_ov)
  have r5 := r4.pushConst (UInt256.ofNat 18446744073709551615) (width := 8) (op := .PUSH8) (by decide) (by native_decide) (by evm_ov)
  have r6 := r5.dup2 (by native_decide) (by evm_ov)
  have r7 := r6.gt (by native_decide) (by evm_ov)
  have r8 := r7.iszero (by native_decide) (by evm_ov)
  have r9 := r8.push2 (UInt256.ofNat 4464) (by native_decide) (by evm_ov)
  have r10 := r9.jumpiT (by native_decide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 4464)) r10 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_4441_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes x2.toNat 32)) (UInt256.ofNat 18446744073709551615))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 4464) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4441) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4464) (l2tol2_block_4441_taken_stack (ee := ee) (x1 := x1) (x2 := x2) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_4441_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_4441_fallthrough`. -/
def l2tol2_block_4441_fallthrough_stack {ee : ExecutionEnv} {x1 : UInt256} {x2 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes x2.toNat 32)) :: x1 :: x2 :: R)

/-- Automatically generated RD summary for bytecode block at pc 4441. -/
theorem l2tol2_block_4441_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes x2.toNat 32)) (UInt256.ofNat 18446744073709551615))) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4441) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4461) (l2tol2_block_4441_fallthrough_stack (ee := ee) (x1 := x1) (x2 := x2) (R := R)) mem aw rdata σ (k + 10) (C + ((34))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.pop (by native_decide) (by evm_ov)
  have r3 := r2.dup2 (by native_decide) (by evm_ov)
  have r4 := r3.calldataload (by native_decide) (by evm_ov)
  have r5 := r4.pushConst (UInt256.ofNat 18446744073709551615) (width := 8) (op := .PUSH8) (by decide) (by native_decide) (by evm_ov)
  have r6 := r5.dup2 (by native_decide) (by evm_ov)
  have r7 := r6.gt (by native_decide) (by evm_ov)
  have r8 := r7.iszero (by native_decide) (by evm_ov)
  have r9 := r8.push2 (UInt256.ofNat 4464) (by native_decide) (by evm_ov)
  have r10 := r9.jumpiNT (by native_decide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 4461)) r10 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_4441_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes x2.toNat 32)) (UInt256.ofNat 18446744073709551615))) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4441) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4461) (l2tol2_block_4441_fallthrough_stack (ee := ee) (x1 := x1) (x2 := x2) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_4441_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 4461. -/
theorem l2tol2_block_4461 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4461) R mem aw rdata σ k C)
    : RDrev ExpiryEvm.l2tol2Runtime g s0 := by
  let r0 := h
  have r1 := r0.push0 (by native_decide) (by evm_ov)
  have r2 := r1.dup1 (by native_decide) (by evm_ov)
  exact RD.genRev r2 (by native_decide) (by evm_ov)

/-- Final stack for bytecode block summary `l2tol2_block_4464_taken`. -/
def l2tol2_block_4464_taken_stack {x0 : UInt256} {x2 : UInt256} {x3 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: (x2 + (UInt256.ofNat 32)) :: x2 :: x3 :: R)

/-- Automatically generated RD summary for bytecode block at pc 4464. -/
theorem l2tol2_block_4464_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt ((x2 + x0) + (UInt256.ofNat 32)) x3)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 4487) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4464) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4487) (l2tol2_block_4464_taken_stack (x0 := x0) (x2 := x2) (x3 := x3) (R := R)) mem aw rdata σ (k + 16) (C + ((52))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 32) (by native_decide) (by evm_ov)
  have r3 := r2.dup4 (by native_decide) (by evm_ov)
  have r4 := r3.add (by native_decide) (by evm_ov)
  have r5 := r4.swap2 (by native_decide) (by evm_ov)
  have r6 := r5.pop (by native_decide) (by evm_ov)
  have r7 := r6.dup4 (by native_decide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 32) (by native_decide) (by evm_ov)
  have r9 := r8.dup3 (by native_decide) (by evm_ov)
  have r10 := r9.dup6 (by native_decide) (by evm_ov)
  have r11 := r10.add (by native_decide) (by evm_ov)
  have r12 := r11.add (by native_decide) (by evm_ov)
  have r13 := r12.gt (by native_decide) (by evm_ov)
  have r14 := r13.iszero (by native_decide) (by evm_ov)
  have r15 := r14.push2 (UInt256.ofNat 4487) (by native_decide) (by evm_ov)
  have r16 := r15.jumpiT (by native_decide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 4487)) r16 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_4464_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt ((x2 + x0) + (UInt256.ofNat 32)) x3)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 4487) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4464) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4487) (l2tol2_block_4464_taken_stack (x0 := x0) (x2 := x2) (x3 := x3) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_4464_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_4464_fallthrough`. -/
def l2tol2_block_4464_fallthrough_stack {x0 : UInt256} {x2 : UInt256} {x3 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: (x2 + (UInt256.ofNat 32)) :: x2 :: x3 :: R)

/-- Automatically generated RD summary for bytecode block at pc 4464. -/
theorem l2tol2_block_4464_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt ((x2 + x0) + (UInt256.ofNat 32)) x3)) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4464) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4484) (l2tol2_block_4464_fallthrough_stack (x0 := x0) (x2 := x2) (x3 := x3) (R := R)) mem aw rdata σ (k + 16) (C + ((52))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 32) (by native_decide) (by evm_ov)
  have r3 := r2.dup4 (by native_decide) (by evm_ov)
  have r4 := r3.add (by native_decide) (by evm_ov)
  have r5 := r4.swap2 (by native_decide) (by evm_ov)
  have r6 := r5.pop (by native_decide) (by evm_ov)
  have r7 := r6.dup4 (by native_decide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 32) (by native_decide) (by evm_ov)
  have r9 := r8.dup3 (by native_decide) (by evm_ov)
  have r10 := r9.dup6 (by native_decide) (by evm_ov)
  have r11 := r10.add (by native_decide) (by evm_ov)
  have r12 := r11.add (by native_decide) (by evm_ov)
  have r13 := r12.gt (by native_decide) (by evm_ov)
  have r14 := r13.iszero (by native_decide) (by evm_ov)
  have r15 := r14.push2 (UInt256.ofNat 4487) (by native_decide) (by evm_ov)
  have r16 := r15.jumpiNT (by native_decide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 4484)) r16 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_4464_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt ((x2 + x0) + (UInt256.ofNat 32)) x3)) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4464) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4484) (l2tol2_block_4464_fallthrough_stack (x0 := x0) (x2 := x2) (x3 := x3) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_4464_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 4484. -/
theorem l2tol2_block_4484 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4484) R mem aw rdata σ k C)
    : RDrev ExpiryEvm.l2tol2Runtime g s0 := by
  let r0 := h
  have r1 := r0.push0 (by native_decide) (by evm_ov)
  have r2 := r1.dup1 (by native_decide) (by evm_ov)
  exact RD.genRev r2 (by native_decide) (by evm_ov)

/-- Final stack for bytecode block summary `l2tol2_block_4487`. -/
def l2tol2_block_4487_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 4487. -/
theorem l2tol2_block_4487 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x4 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4487) (x0 :: x1 :: x2 :: x3 :: x4 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 x4 (l2tol2_block_4487_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 7) (C + ((22))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.swap3 (by native_decide) (by evm_ov)
  have r3 := r2.pop (by native_decide) (by evm_ov)
  have r4 := r3.swap3 (by native_decide) (by evm_ov)
  have r5 := r4.swap1 (by native_decide) (by evm_ov)
  have r6 := r5.pop (by native_decide) (by evm_ov)
  have r7 := r6.jump (by native_decide) hvalid (by evm_ov)
  exact RD.normalizeCounters r7 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_4487_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x4 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4487) (x0 :: x1 :: x2 :: x3 :: x4 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 x4 (l2tol2_block_4487_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_4487 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_4494_taken`. -/
def l2tol2_block_4494_taken_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((⟨0⟩ : UInt256) :: (⟨0⟩ : UInt256) :: (⟨0⟩ : UInt256) :: (⟨0⟩ : UInt256) :: (⟨0⟩ : UInt256) :: (⟨0⟩ : UInt256) :: (⟨0⟩ : UInt256) :: (⟨0⟩ : UInt256) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 4494. -/
theorem l2tol2_block_4494_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 224))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 4517) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4494) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4517) (l2tol2_block_4494_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 17) (C + ((52))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.push0 (by native_decide) (by evm_ov)
  have r3 := r2.dup1 (by native_decide) (by evm_ov)
  have r4 := r3.push0 (by native_decide) (by evm_ov)
  have r5 := r4.dup1 (by native_decide) (by evm_ov)
  have r6 := r5.push0 (by native_decide) (by evm_ov)
  have r7 := r6.dup1 (by native_decide) (by evm_ov)
  have r8 := r7.push0 (by native_decide) (by evm_ov)
  have r9 := r8.dup1 (by native_decide) (by evm_ov)
  have r10 := r9.push1 (UInt256.ofNat 224) (by native_decide) (by evm_ov)
  have r11 := r10.dup10 (by native_decide) (by evm_ov)
  have r12 := r11.dup12 (by native_decide) (by evm_ov)
  have r13 := r12.sub (by native_decide) (by evm_ov)
  have r14 := r13.slt (by native_decide) (by evm_ov)
  have r15 := r14.iszero (by native_decide) (by evm_ov)
  have r16 := r15.push2 (UInt256.ofNat 4517) (by native_decide) (by evm_ov)
  have r17 := r16.jumpiT (by native_decide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 4517)) r17 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_4494_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 224))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 4517) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4494) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4517) (l2tol2_block_4494_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_4494_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_4494_fallthrough`. -/
def l2tol2_block_4494_fallthrough_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((⟨0⟩ : UInt256) :: (⟨0⟩ : UInt256) :: (⟨0⟩ : UInt256) :: (⟨0⟩ : UInt256) :: (⟨0⟩ : UInt256) :: (⟨0⟩ : UInt256) :: (⟨0⟩ : UInt256) :: (⟨0⟩ : UInt256) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 4494. -/
theorem l2tol2_block_4494_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 224))) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4494) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4514) (l2tol2_block_4494_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 17) (C + ((52))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.push0 (by native_decide) (by evm_ov)
  have r3 := r2.dup1 (by native_decide) (by evm_ov)
  have r4 := r3.push0 (by native_decide) (by evm_ov)
  have r5 := r4.dup1 (by native_decide) (by evm_ov)
  have r6 := r5.push0 (by native_decide) (by evm_ov)
  have r7 := r6.dup1 (by native_decide) (by evm_ov)
  have r8 := r7.push0 (by native_decide) (by evm_ov)
  have r9 := r8.dup1 (by native_decide) (by evm_ov)
  have r10 := r9.push1 (UInt256.ofNat 224) (by native_decide) (by evm_ov)
  have r11 := r10.dup10 (by native_decide) (by evm_ov)
  have r12 := r11.dup12 (by native_decide) (by evm_ov)
  have r13 := r12.sub (by native_decide) (by evm_ov)
  have r14 := r13.slt (by native_decide) (by evm_ov)
  have r15 := r14.iszero (by native_decide) (by evm_ov)
  have r16 := r15.push2 (UInt256.ofNat 4517) (by native_decide) (by evm_ov)
  have r17 := r16.jumpiNT (by native_decide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 4514)) r17 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_4494_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 224))) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4494) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4514) (l2tol2_block_4494_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_4494_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 4514. -/
theorem l2tol2_block_4514 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4514) R mem aw rdata σ k C)
    : RDrev ExpiryEvm.l2tol2Runtime g s0 := by
  let r0 := h
  have r1 := r0.push0 (by native_decide) (by evm_ov)
  have r2 := r1.dup1 (by native_decide) (by evm_ov)
  exact RD.genRev r2 (by native_decide) (by evm_ov)

/-- Final stack for bytecode block summary `l2tol2_block_4517`. -/
def l2tol2_block_4517_stack {ee : ExecutionEnv} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {x6 : UInt256} {x7 : UInt256} {x8 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes x8.toNat 32)) :: (UInt256.ofNat 4528) :: (uInt256OfByteArray (ee.calldata.readBytes x8.toNat 32)) :: x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R)

/-- Automatically generated RD summary for bytecode block at pc 4517. -/
theorem l2tol2_block_4517 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 4389) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4517) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4389) (l2tol2_block_4517_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (x7 := x7) (x8 := x8) (R := R)) mem aw rdata σ (k + 7) (C + ((24))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.dup9 (by native_decide) (by evm_ov)
  have r3 := r2.calldataload (by native_decide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 4528) (by native_decide) (by evm_ov)
  have r5 := r4.dup2 (by native_decide) (by evm_ov)
  have r6 := r5.push2 (UInt256.ofNat 4389) (by native_decide) (by evm_ov)
  have r7 := r6.jump (by native_decide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 4389)) r7 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_4517_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 4389) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4517) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4389) (l2tol2_block_4517_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (x7 := x7) (x8 := x8) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_4517 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_4528`. -/
def l2tol2_block_4528_stack {ee : ExecutionEnv} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {x9 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes (x9 + (UInt256.ofNat 96)).toNat 32)) :: (UInt256.ofNat 4558) :: (uInt256OfByteArray (ee.calldata.readBytes (x9 + (UInt256.ofNat 96)).toNat 32)) :: x1 :: x2 :: x3 :: x4 :: x5 :: (uInt256OfByteArray (ee.calldata.readBytes (x9 + (UInt256.ofNat 64)).toNat 32)) :: (uInt256OfByteArray (ee.calldata.readBytes (x9 + (UInt256.ofNat 32)).toNat 32)) :: x0 :: x9 :: R)

/-- Automatically generated RD summary for bytecode block at pc 4528. -/
theorem l2tol2_block_4528 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 4389) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4528) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4389) (l2tol2_block_4528_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x9 := x9) (R := R)) mem aw rdata σ (k + 23) (C + ((69))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.swap8 (by native_decide) (by evm_ov)
  have r3 := r2.pop (by native_decide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 32) (by native_decide) (by evm_ov)
  have r5 := r4.dup10 (by native_decide) (by evm_ov)
  have r6 := r5.add (by native_decide) (by evm_ov)
  have r7 := r6.calldataload (by native_decide) (by evm_ov)
  have r8 := r7.swap7 (by native_decide) (by evm_ov)
  have r9 := r8.pop (by native_decide) (by evm_ov)
  have r10 := r9.push1 (UInt256.ofNat 64) (by native_decide) (by evm_ov)
  have r11 := r10.dup10 (by native_decide) (by evm_ov)
  have r12 := r11.add (by native_decide) (by evm_ov)
  have r13 := r12.calldataload (by native_decide) (by evm_ov)
  have r14 := r13.swap6 (by native_decide) (by evm_ov)
  have r15 := r14.pop (by native_decide) (by evm_ov)
  have r16 := r15.push1 (UInt256.ofNat 96) (by native_decide) (by evm_ov)
  have r17 := r16.dup10 (by native_decide) (by evm_ov)
  have r18 := r17.add (by native_decide) (by evm_ov)
  have r19 := r18.calldataload (by native_decide) (by evm_ov)
  have r20 := r19.push2 (UInt256.ofNat 4558) (by native_decide) (by evm_ov)
  have r21 := r20.dup2 (by native_decide) (by evm_ov)
  have r22 := r21.push2 (UInt256.ofNat 4389) (by native_decide) (by evm_ov)
  have r23 := r22.jump (by native_decide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 4389)) r23 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_4528_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 4389) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4528) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4389) (l2tol2_block_4528_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x9 := x9) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_4528 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_4558`. -/
def l2tol2_block_4558_stack {ee : ExecutionEnv} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x6 : UInt256} {x7 : UInt256} {x8 : UInt256} {x9 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes (x9 + (UInt256.ofNat 128)).toNat 32)) :: (UInt256.ofNat 4574) :: (uInt256OfByteArray (ee.calldata.readBytes (x9 + (UInt256.ofNat 128)).toNat 32)) :: x1 :: x2 :: x3 :: x4 :: x0 :: x6 :: x7 :: x8 :: x9 :: R)

/-- Automatically generated RD summary for bytecode block at pc 4558. -/
theorem l2tol2_block_4558 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 4389) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4558) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4389) (l2tol2_block_4558_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x6 := x6) (x7 := x7) (x8 := x8) (x9 := x9) (R := R)) mem aw rdata σ (k + 11) (C + ((35))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.swap5 (by native_decide) (by evm_ov)
  have r3 := r2.pop (by native_decide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 128) (by native_decide) (by evm_ov)
  have r5 := r4.dup10 (by native_decide) (by evm_ov)
  have r6 := r5.add (by native_decide) (by evm_ov)
  have r7 := r6.calldataload (by native_decide) (by evm_ov)
  have r8 := r7.push2 (UInt256.ofNat 4574) (by native_decide) (by evm_ov)
  have r9 := r8.dup2 (by native_decide) (by evm_ov)
  have r10 := r9.push2 (UInt256.ofNat 4389) (by native_decide) (by evm_ov)
  have r11 := r10.jump (by native_decide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 4389)) r11 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_4558_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 4389) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4558) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4389) (l2tol2_block_4558_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x6 := x6) (x7 := x7) (x8 := x8) (x9 := x9) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_4558 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_4574_taken`. -/
def l2tol2_block_4574_taken_stack {ee : ExecutionEnv} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x5 : UInt256} {x6 : UInt256} {x7 : UInt256} {x8 : UInt256} {x9 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes (x9 + (UInt256.ofNat 160)).toNat 32)) :: x1 :: x2 :: x3 :: x0 :: x5 :: x6 :: x7 :: x8 :: x9 :: R)

/-- Automatically generated RD summary for bytecode block at pc 4574. -/
theorem l2tol2_block_4574_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes (x9 + (UInt256.ofNat 160)).toNat 32)) (UInt256.ofNat 18446744073709551615))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 4601) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4574) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4601) (l2tol2_block_4574_taken_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x5 := x5) (x6 := x6) (x7 := x7) (x8 := x8) (x9 := x9) (R := R)) mem aw rdata σ (k + 13) (C + ((43))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.swap4 (by native_decide) (by evm_ov)
  have r3 := r2.pop (by native_decide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 160) (by native_decide) (by evm_ov)
  have r5 := r4.dup10 (by native_decide) (by evm_ov)
  have r6 := r5.add (by native_decide) (by evm_ov)
  have r7 := r6.calldataload (by native_decide) (by evm_ov)
  have r8 := r7.pushConst (UInt256.ofNat 18446744073709551615) (width := 8) (op := .PUSH8) (by decide) (by native_decide) (by evm_ov)
  have r9 := r8.dup2 (by native_decide) (by evm_ov)
  have r10 := r9.gt (by native_decide) (by evm_ov)
  have r11 := r10.iszero (by native_decide) (by evm_ov)
  have r12 := r11.push2 (UInt256.ofNat 4601) (by native_decide) (by evm_ov)
  have r13 := r12.jumpiT (by native_decide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 4601)) r13 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_4574_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes (x9 + (UInt256.ofNat 160)).toNat 32)) (UInt256.ofNat 18446744073709551615))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 4601) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4574) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4601) (l2tol2_block_4574_taken_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x5 := x5) (x6 := x6) (x7 := x7) (x8 := x8) (x9 := x9) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_4574_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_4574_fallthrough`. -/
def l2tol2_block_4574_fallthrough_stack {ee : ExecutionEnv} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x5 : UInt256} {x6 : UInt256} {x7 : UInt256} {x8 : UInt256} {x9 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes (x9 + (UInt256.ofNat 160)).toNat 32)) :: x1 :: x2 :: x3 :: x0 :: x5 :: x6 :: x7 :: x8 :: x9 :: R)

/-- Automatically generated RD summary for bytecode block at pc 4574. -/
theorem l2tol2_block_4574_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes (x9 + (UInt256.ofNat 160)).toNat 32)) (UInt256.ofNat 18446744073709551615))) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4574) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4598) (l2tol2_block_4574_fallthrough_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x5 := x5) (x6 := x6) (x7 := x7) (x8 := x8) (x9 := x9) (R := R)) mem aw rdata σ (k + 13) (C + ((43))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.swap4 (by native_decide) (by evm_ov)
  have r3 := r2.pop (by native_decide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 160) (by native_decide) (by evm_ov)
  have r5 := r4.dup10 (by native_decide) (by evm_ov)
  have r6 := r5.add (by native_decide) (by evm_ov)
  have r7 := r6.calldataload (by native_decide) (by evm_ov)
  have r8 := r7.pushConst (UInt256.ofNat 18446744073709551615) (width := 8) (op := .PUSH8) (by decide) (by native_decide) (by evm_ov)
  have r9 := r8.dup2 (by native_decide) (by evm_ov)
  have r10 := r9.gt (by native_decide) (by evm_ov)
  have r11 := r10.iszero (by native_decide) (by evm_ov)
  have r12 := r11.push2 (UInt256.ofNat 4601) (by native_decide) (by evm_ov)
  have r13 := r12.jumpiNT (by native_decide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 4598)) r13 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_4574_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes (x9 + (UInt256.ofNat 160)).toNat 32)) (UInt256.ofNat 18446744073709551615))) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4574) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4598) (l2tol2_block_4574_fallthrough_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x5 := x5) (x6 := x6) (x7 := x7) (x8 := x8) (x9 := x9) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_4574_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 4598. -/
theorem l2tol2_block_4598 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4598) R mem aw rdata σ k C)
    : RDrev ExpiryEvm.l2tol2Runtime g s0 := by
  let r0 := h
  have r1 := r0.push0 (by native_decide) (by evm_ov)
  have r2 := r1.dup1 (by native_decide) (by evm_ov)
  exact RD.genRev r2 (by native_decide) (by evm_ov)

end l2tol2Blocks
