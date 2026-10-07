import Reasoning.Reach
import ExpiryEvm.Bytecode

open Solm ABI Ethereum Ethereum.EVM
open Reasoning.Theory Reasoning.Reach

namespace l2tol2Blocks

/-- Automatically generated RD summary for bytecode block at pc 5741. -/
theorem l2tol2_block_5741 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5741) R mem aw rdata σ k C)
    : RDrev ExpiryEvm.l2tol2Runtime g s0 := by
  let r0 := h
  have r1 := r0.push0 (by native_decide) (by evm_ov)
  have r2 := r1.dup1 (by native_decide) (by evm_ov)
  exact RD.genRev r2 (by native_decide) (by evm_ov)

/-- Final stack for bytecode block summary `l2tol2_block_5744_taken`. -/
def l2tol2_block_5744_taken_stack {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: (x4 + x1) :: x2 :: x3 :: x4 :: x5 :: R)

/-- Automatically generated RD summary for bytecode block at pc 5744. -/
theorem l2tol2_block_5744_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hcond : (UInt256.slt ((x4 + x1) + (UInt256.ofNat 31)) x5) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 5763) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5744) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5763) (l2tol2_block_5744_taken_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (R := R)) mem aw rdata σ (k + 13) (C + ((43))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.dup2 (by native_decide) (by evm_ov)
  have r3 := r2.dup6 (by native_decide) (by evm_ov)
  have r4 := r3.add (by native_decide) (by evm_ov)
  have r5 := r4.swap2 (by native_decide) (by evm_ov)
  have r6 := r5.pop (by native_decide) (by evm_ov)
  have r7 := r6.dup6 (by native_decide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 31) (by native_decide) (by evm_ov)
  have r9 := r8.dup4 (by native_decide) (by evm_ov)
  have r10 := r9.add (by native_decide) (by evm_ov)
  have r11 := r10.slt (by native_decide) (by evm_ov)
  have r12 := r11.push2 (UInt256.ofNat 5763) (by native_decide) (by evm_ov)
  have r13 := r12.jumpiT (by native_decide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 5763)) r13 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_5744_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hcond : (UInt256.slt ((x4 + x1) + (UInt256.ofNat 31)) x5) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 5763) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5744) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5763) (l2tol2_block_5744_taken_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_5744_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_5744_fallthrough`. -/
def l2tol2_block_5744_fallthrough_stack {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: (x4 + x1) :: x2 :: x3 :: x4 :: x5 :: R)

/-- Automatically generated RD summary for bytecode block at pc 5744. -/
theorem l2tol2_block_5744_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hcond : (UInt256.slt ((x4 + x1) + (UInt256.ofNat 31)) x5) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5744) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5760) (l2tol2_block_5744_fallthrough_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (R := R)) mem aw rdata σ (k + 13) (C + ((43))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.dup2 (by native_decide) (by evm_ov)
  have r3 := r2.dup6 (by native_decide) (by evm_ov)
  have r4 := r3.add (by native_decide) (by evm_ov)
  have r5 := r4.swap2 (by native_decide) (by evm_ov)
  have r6 := r5.pop (by native_decide) (by evm_ov)
  have r7 := r6.dup6 (by native_decide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 31) (by native_decide) (by evm_ov)
  have r9 := r8.dup4 (by native_decide) (by evm_ov)
  have r10 := r9.add (by native_decide) (by evm_ov)
  have r11 := r10.slt (by native_decide) (by evm_ov)
  have r12 := r11.push2 (UInt256.ofNat 5763) (by native_decide) (by evm_ov)
  have r13 := r12.jumpiNT (by native_decide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 5760)) r13 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_5744_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hcond : (UInt256.slt ((x4 + x1) + (UInt256.ofNat 31)) x5) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5744) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5760) (l2tol2_block_5744_fallthrough_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_5744_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 5760. -/
theorem l2tol2_block_5760 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5760) R mem aw rdata σ k C)
    : RDrev ExpiryEvm.l2tol2Runtime g s0 := by
  let r0 := h
  have r1 := r0.push0 (by native_decide) (by evm_ov)
  have r2 := r1.dup1 (by native_decide) (by evm_ov)
  exact RD.genRev r2 (by native_decide) (by evm_ov)

/-- Final stack for bytecode block summary `l2tol2_block_5763_taken`. -/
def l2tol2_block_5763_taken_stack {ee : ExecutionEnv} {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes x1.toNat 32)) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 5763. -/
theorem l2tol2_block_5763_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes x1.toNat 32)) x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 5781) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5763) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5781) (l2tol2_block_5763_taken_stack (ee := ee) (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 9) (C + ((32))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.dup2 (by native_decide) (by evm_ov)
  have r3 := r2.calldataload (by native_decide) (by evm_ov)
  have r4 := r3.dup2 (by native_decide) (by evm_ov)
  have r5 := r4.dup2 (by native_decide) (by evm_ov)
  have r6 := r5.gt (by native_decide) (by evm_ov)
  have r7 := r6.iszero (by native_decide) (by evm_ov)
  have r8 := r7.push2 (UInt256.ofNat 5781) (by native_decide) (by evm_ov)
  have r9 := r8.jumpiT (by native_decide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 5781)) r9 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_5763_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes x1.toNat 32)) x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 5781) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5763) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5781) (l2tol2_block_5763_taken_stack (ee := ee) (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_5763_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_5763_fallthrough`. -/
def l2tol2_block_5763_fallthrough_stack {ee : ExecutionEnv} {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes x1.toNat 32)) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 5763. -/
theorem l2tol2_block_5763_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes x1.toNat 32)) x0)) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5763) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5774) (l2tol2_block_5763_fallthrough_stack (ee := ee) (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 9) (C + ((32))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.dup2 (by native_decide) (by evm_ov)
  have r3 := r2.calldataload (by native_decide) (by evm_ov)
  have r4 := r3.dup2 (by native_decide) (by evm_ov)
  have r5 := r4.dup2 (by native_decide) (by evm_ov)
  have r6 := r5.gt (by native_decide) (by evm_ov)
  have r7 := r6.iszero (by native_decide) (by evm_ov)
  have r8 := r7.push2 (UInt256.ofNat 5781) (by native_decide) (by evm_ov)
  have r9 := r8.jumpiNT (by native_decide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 5774)) r9 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_5763_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes x1.toNat 32)) x0)) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5763) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5774) (l2tol2_block_5763_fallthrough_stack (ee := ee) (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_5763_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_5774`. -/
def l2tol2_block_5774_stack {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 5781) :: R)

/-- Automatically generated RD summary for bytecode block at pc 5774. -/
theorem l2tol2_block_5774 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 5643) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5774) R mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5643) (l2tol2_block_5774_stack (R := R)) mem aw rdata σ (k + 3) (C + ((14))) := by
  let r0 := h
  have r1 := r0.push2 (UInt256.ofNat 5781) (by native_decide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 5643) (by native_decide) (by evm_ov)
  have r3 := r2.jump (by native_decide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 5643)) r3 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_5774_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 5643) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5774) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5643) (l2tol2_block_5774_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_5774 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_5781_taken`. -/
def l2tol2_block_5781_taken_stack {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((memLoad (UInt256.ofNat 64) mem) :: ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 5781. -/
theorem l2tol2_block_5781_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.lor (UInt256.lt ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) (memLoad (UInt256.ofNat 64) mem)) (UInt256.gt ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) x1))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 5851) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5781) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5851) (l2tol2_block_5781_taken_stack (mem := mem) (x0 := x0) (x1 := x1) (R := R)) mem (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 26) (C + ((83) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 64) (by native_decide) (by evm_ov)
  have r3 := RD.genMload r2 (by native_decide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 31) (by native_decide) (by evm_ov)
  have r5 := r4.dup3 (by native_decide) (by evm_ov)
  have r6 := r5.add (by native_decide) (by evm_ov)
  have r7 := r6.pushConst (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (width := 32) (op := .PUSH32) (by decide) (by native_decide) (by evm_ov)
  have r8 := r7.swap1 (by native_decide) (by evm_ov)
  have r9 := r8.dup2 (by native_decide) (by evm_ov)
  have r10 := r9.and (by native_decide) (by evm_ov)
  have r11 := r10.push1 (UInt256.ofNat 63) (by native_decide) (by evm_ov)
  have r12 := r11.add (by native_decide) (by evm_ov)
  have r13 := r12.and (by native_decide) (by evm_ov)
  have r14 := r13.dup2 (by native_decide) (by evm_ov)
  have r15 := r14.add (by native_decide) (by evm_ov)
  have r16 := r15.swap1 (by native_decide) (by evm_ov)
  have r17 := r16.dup4 (by native_decide) (by evm_ov)
  have r18 := r17.dup3 (by native_decide) (by evm_ov)
  have r19 := r18.gt (by native_decide) (by evm_ov)
  have r20 := r19.dup2 (by native_decide) (by evm_ov)
  have r21 := r20.dup4 (by native_decide) (by evm_ov)
  have r22 := r21.lt (by native_decide) (by evm_ov)
  have r23 := r22.or (by native_decide) (by evm_ov)
  have r24 := r23.iszero (by native_decide) (by evm_ov)
  have r25 := r24.push2 (UInt256.ofNat 5851) (by native_decide) (by evm_ov)
  have r26 := r25.jumpiT (by native_decide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 5851)) r26 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_5781_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.lor (UInt256.lt ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) (memLoad (UInt256.ofNat 64) mem)) (UInt256.gt ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) x1))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 5851) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5781) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5851) (l2tol2_block_5781_taken_stack (mem := mem) (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_5781_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_5781_fallthrough`. -/
def l2tol2_block_5781_fallthrough_stack {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((memLoad (UInt256.ofNat 64) mem) :: ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 5781. -/
theorem l2tol2_block_5781_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.lor (UInt256.lt ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) (memLoad (UInt256.ofNat 64) mem)) (UInt256.gt ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) x1))) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5781) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5844) (l2tol2_block_5781_fallthrough_stack (mem := mem) (x0 := x0) (x1 := x1) (R := R)) mem (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 26) (C + ((83) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 64) (by native_decide) (by evm_ov)
  have r3 := RD.genMload r2 (by native_decide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 31) (by native_decide) (by evm_ov)
  have r5 := r4.dup3 (by native_decide) (by evm_ov)
  have r6 := r5.add (by native_decide) (by evm_ov)
  have r7 := r6.pushConst (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (width := 32) (op := .PUSH32) (by decide) (by native_decide) (by evm_ov)
  have r8 := r7.swap1 (by native_decide) (by evm_ov)
  have r9 := r8.dup2 (by native_decide) (by evm_ov)
  have r10 := r9.and (by native_decide) (by evm_ov)
  have r11 := r10.push1 (UInt256.ofNat 63) (by native_decide) (by evm_ov)
  have r12 := r11.add (by native_decide) (by evm_ov)
  have r13 := r12.and (by native_decide) (by evm_ov)
  have r14 := r13.dup2 (by native_decide) (by evm_ov)
  have r15 := r14.add (by native_decide) (by evm_ov)
  have r16 := r15.swap1 (by native_decide) (by evm_ov)
  have r17 := r16.dup4 (by native_decide) (by evm_ov)
  have r18 := r17.dup3 (by native_decide) (by evm_ov)
  have r19 := r18.gt (by native_decide) (by evm_ov)
  have r20 := r19.dup2 (by native_decide) (by evm_ov)
  have r21 := r20.dup4 (by native_decide) (by evm_ov)
  have r22 := r21.lt (by native_decide) (by evm_ov)
  have r23 := r22.or (by native_decide) (by evm_ov)
  have r24 := r23.iszero (by native_decide) (by evm_ov)
  have r25 := r24.push2 (UInt256.ofNat 5851) (by native_decide) (by evm_ov)
  have r26 := r25.jumpiNT (by native_decide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 5844)) r26 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_5781_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.lor (UInt256.lt ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) (memLoad (UInt256.ofNat 64) mem)) (UInt256.gt ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) x1))) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5781) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5844) (l2tol2_block_5781_fallthrough_stack (mem := mem) (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_5781_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_5844`. -/
def l2tol2_block_5844_stack {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 5851) :: R)

/-- Automatically generated RD summary for bytecode block at pc 5844. -/
theorem l2tol2_block_5844 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 5643) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5844) R mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5643) (l2tol2_block_5844_stack (R := R)) mem aw rdata σ (k + 3) (C + ((14))) := by
  let r0 := h
  have r1 := r0.push2 (UInt256.ofNat 5851) (by native_decide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 5643) (by native_decide) (by evm_ov)
  have r3 := r2.jump (by native_decide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 5643)) r3 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_5844_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 5643) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5844) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5643) (l2tol2_block_5844_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_5844 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final memory for bytecode block summary `l2tol2_block_5851_taken`. -/
def l2tol2_block_5851_taken_memory {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} : ByteArray :=
  (x2.toByteArray.write 0 (x1.toByteArray.write 0 mem (UInt256.ofNat 64).toNat 32) x0.toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 5851. -/
theorem l2tol2_block_5851_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt ((x4 + x2) + (UInt256.ofNat 32)) x8)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 5875) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5851) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5875) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) (l2tol2_block_5851_taken_memory (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2)) (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) x0 (⟨32⟩ : UInt256)) rdata σ (k + 17) (C + ((56) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) x0 (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.dup2 (by native_decide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 64) (by native_decide) (by evm_ov)
  have r4 := RD.genMstore r3 (by native_decide) (by evm_ov)
  have r5 := r4.dup3 (by native_decide) (by evm_ov)
  have r6 := r5.dup2 (by native_decide) (by evm_ov)
  have r7 := RD.genMstore r6 (by native_decide) (by evm_ov)
  have r8 := r7.dup9 (by native_decide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 32) (by native_decide) (by evm_ov)
  have r10 := r9.dup5 (by native_decide) (by evm_ov)
  have r11 := r10.dup8 (by native_decide) (by evm_ov)
  have r12 := r11.add (by native_decide) (by evm_ov)
  have r13 := r12.add (by native_decide) (by evm_ov)
  have r14 := r13.gt (by native_decide) (by evm_ov)
  have r15 := r14.iszero (by native_decide) (by evm_ov)
  have r16 := r15.push2 (UInt256.ofNat 5875) (by native_decide) (by evm_ov)
  have r17 := r16.jumpiT (by native_decide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 5875)) r17 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_5851_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt ((x4 + x2) + (UInt256.ofNat 32)) x8)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 5875) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5851) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5875) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) (l2tol2_block_5851_taken_memory (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_5851_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final memory for bytecode block summary `l2tol2_block_5851_fallthrough`. -/
def l2tol2_block_5851_fallthrough_memory {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} : ByteArray :=
  (x2.toByteArray.write 0 (x1.toByteArray.write 0 mem (UInt256.ofNat 64).toNat 32) x0.toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 5851. -/
theorem l2tol2_block_5851_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt ((x4 + x2) + (UInt256.ofNat 32)) x8)) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5851) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5872) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) (l2tol2_block_5851_fallthrough_memory (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2)) (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) x0 (⟨32⟩ : UInt256)) rdata σ (k + 17) (C + ((56) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) x0 (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.dup2 (by native_decide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 64) (by native_decide) (by evm_ov)
  have r4 := RD.genMstore r3 (by native_decide) (by evm_ov)
  have r5 := r4.dup3 (by native_decide) (by evm_ov)
  have r6 := r5.dup2 (by native_decide) (by evm_ov)
  have r7 := RD.genMstore r6 (by native_decide) (by evm_ov)
  have r8 := r7.dup9 (by native_decide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 32) (by native_decide) (by evm_ov)
  have r10 := r9.dup5 (by native_decide) (by evm_ov)
  have r11 := r10.dup8 (by native_decide) (by evm_ov)
  have r12 := r11.add (by native_decide) (by evm_ov)
  have r13 := r12.add (by native_decide) (by evm_ov)
  have r14 := r13.gt (by native_decide) (by evm_ov)
  have r15 := r14.iszero (by native_decide) (by evm_ov)
  have r16 := r15.push2 (UInt256.ofNat 5875) (by native_decide) (by evm_ov)
  have r17 := r16.jumpiNT (by native_decide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 5872)) r17 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_5851_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt ((x4 + x2) + (UInt256.ofNat 32)) x8)) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5851) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5872) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) (l2tol2_block_5851_fallthrough_memory (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_5851_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 5872. -/
theorem l2tol2_block_5872 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5872) R mem aw rdata σ k C)
    : RDrev ExpiryEvm.l2tol2Runtime g s0 := by
  let r0 := h
  have r1 := r0.push0 (by native_decide) (by evm_ov)
  have r2 := r1.dup1 (by native_decide) (by evm_ov)
  exact RD.genRev r2 (by native_decide) (by evm_ov)

/-- Final stack for bytecode block summary `l2tol2_block_5875`. -/
def l2tol2_block_5875_stack {x0 : UInt256} {x6 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: x6 :: R)

/-- Final memory for bytecode block summary `l2tol2_block_5875`. -/
def l2tol2_block_5875_memory {ee : ExecutionEnv} {mem : ByteArray} {x0 : UInt256} {x2 : UInt256} {x4 : UInt256} : ByteArray :=
  ((⟨0⟩ : UInt256).toByteArray.write 0 (ee.calldata.write (x4 + (UInt256.ofNat 32)).toNat mem (x0 + (UInt256.ofNat 32)).toNat x2.toNat) ((x0 + x2) + (UInt256.ofNat 32)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 5875. -/
theorem l2tol2_block_5875 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 14 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x9 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5875) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 x9 (l2tol2_block_5875_stack (x0 := x0) (x6 := x6) (R := R)) (l2tol2_block_5875_memory (ee := ee) (mem := mem) (x0 := x0) (x2 := x2) (x4 := x4)) (M (M aw (x0 + (UInt256.ofNat 32)) x2) ((x0 + x2) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) rdata σ (k + 30) (C + ((81) + (memExpansionCost aw (x0 + (UInt256.ofNat 32)) x2) + (3 + 3 * ((x2.toNat + 31) / 32)) + (memExpansionCost (M aw (x0 + (UInt256.ofNat 32)) x2) ((x0 + x2) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.dup3 (by native_decide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 32) (by native_decide) (by evm_ov)
  have r4 := r3.dup7 (by native_decide) (by evm_ov)
  have r5 := r4.add (by native_decide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 32) (by native_decide) (by evm_ov)
  have r7 := r6.dup4 (by native_decide) (by evm_ov)
  have r8 := r7.add (by native_decide) (by evm_ov)
  have r9 := RD.genCalldatacopy r8 (by native_decide) (by evm_ov)
  have r10 := r9.push0 (by native_decide) (by evm_ov)
  have r11 := r10.push1 (UInt256.ofNat 32) (by native_decide) (by evm_ov)
  have r12 := r11.dup5 (by native_decide) (by evm_ov)
  have r13 := r12.dup4 (by native_decide) (by evm_ov)
  have r14 := r13.add (by native_decide) (by evm_ov)
  have r15 := r14.add (by native_decide) (by evm_ov)
  have r16 := RD.genMstore r15 (by native_decide) (by evm_ov)
  have r17 := r16.dup1 (by native_decide) (by evm_ov)
  have r18 := r17.swap6 (by native_decide) (by evm_ov)
  have r19 := r18.pop (by native_decide) (by evm_ov)
  have r20 := r19.pop (by native_decide) (by evm_ov)
  have r21 := r20.pop (by native_decide) (by evm_ov)
  have r22 := r21.pop (by native_decide) (by evm_ov)
  have r23 := r22.pop (by native_decide) (by evm_ov)
  have r24 := r23.pop (by native_decide) (by evm_ov)
  have r25 := r24.swap3 (by native_decide) (by evm_ov)
  have r26 := r25.pop (by native_decide) (by evm_ov)
  have r27 := r26.swap3 (by native_decide) (by evm_ov)
  have r28 := r27.swap1 (by native_decide) (by evm_ov)
  have r29 := r28.pop (by native_decide) (by evm_ov)
  have r30 := r29.jump (by native_decide) hvalid (by evm_ov)
  exact RD.normalizeCounters r30 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_5875_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 14 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x9 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5875) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 x9 (l2tol2_block_5875_stack (x0 := x0) (x6 := x6) (R := R)) (l2tol2_block_5875_memory (ee := ee) (mem := mem) (x0 := x0) (x2 := x2) (x4 := x4)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_5875 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 5908. -/
theorem l2tol2_block_5908 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5908) R mem aw rdata σ k C)
    : RDinvalid ExpiryEvm.l2tol2Runtime g s0 := by
  let r0 := h
  exact RD.invalid r0 (by native_decide)

end l2tol2Blocks
