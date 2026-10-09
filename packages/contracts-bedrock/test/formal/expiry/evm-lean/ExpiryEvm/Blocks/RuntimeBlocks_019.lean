import Reasoning.Reach
import ExpiryEvm.Bytecode

open Solm ABI Ethereum Ethereum.EVM
open Reasoning.Theory Reasoning.Reach

namespace l2tol2Blocks

/-- Final stack for bytecode block summary `l2tol2_block_5876`. -/
def l2tol2_block_5876_stack {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 5883) :: R)

/-- Automatically generated RD summary for bytecode block at pc 5876. -/
theorem l2tol2_block_5876 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 5738) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5876) R mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5738) (l2tol2_block_5876_stack (R := R)) mem aw rdata σ (k + 3) (C + ((14))) := by
  let r0 := h
  have r1 := r0.push2 (UInt256.ofNat 5883) (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 5738) (by evm_kdecide) (by evm_ov)
  have r3 := r2.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 5738)) r3 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_5876_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 5738) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5876) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5738) (l2tol2_block_5876_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_5876 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_5883`. -/
def l2tol2_block_5883_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  (((UInt256.ofNat 1) + x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 5883. -/
theorem l2tol2_block_5883 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x4 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5883) (x0 :: x1 :: x2 :: x3 :: x4 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 x4 (l2tol2_block_5883_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 9) (C + ((27))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 1) (by evm_kdecide) (by evm_ov)
  have r3 := r2.add (by evm_kdecide) (by evm_ov)
  have r4 := r3.swap4 (by evm_kdecide) (by evm_ov)
  have r5 := r4.swap3 (by evm_kdecide) (by evm_ov)
  have r6 := r5.pop (by evm_kdecide) (by evm_ov)
  have r7 := r6.pop (by evm_kdecide) (by evm_ov)
  have r8 := r7.pop (by evm_kdecide) (by evm_ov)
  have r9 := r8.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r9 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_5883_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x4 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5883) (x0 :: x1 :: x2 :: x3 :: x4 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 x4 (l2tol2_block_5883_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_5883 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_5893`. -/
def l2tol2_block_5893_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((((UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x1 + (UInt256.ofNat 31))) + x0) + (UInt256.ofNat 96)) :: R)

/-- Final memory for bytecode block summary `l2tol2_block_5893`. -/
def l2tol2_block_5893_memory {ee : ExecutionEnv} {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} : ByteArray :=
  ((⟨0⟩ : UInt256).toByteArray.write 0 (ee.calldata.write x2.toNat (x1.toByteArray.write 0 ((UInt256.ofNat 64).toByteArray.write 0 ((UInt256.land x3 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 mem x0.toNat 32) (x0 + (UInt256.ofNat 32)).toNat 32) (x0 + (UInt256.ofNat 64)).toNat 32) (x0 + (UInt256.ofNat 96)).toNat x1.toNat) ((UInt256.ofNat 96) + (x1 + x0)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 5893. -/
theorem l2tol2_block_5893 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x4 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5893) (x0 :: x1 :: x2 :: x3 :: x4 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 x4 (l2tol2_block_5893_stack (x0 := x0) (x1 := x1) (R := R)) (l2tol2_block_5893_memory (ee := ee) (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3)) (M (M (M (M (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 96)) x1) ((UInt256.ofNat 96) + (x1 + x0)) (⟨32⟩ : UInt256)) rdata σ (k + 47) (C + ((138) + (memExpansionCost aw x0 (⟨32⟩ : UInt256)) + (memExpansionCost (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 96)) x1) + (3 + 3 * ((x1.toNat + 31) / 32)) + (memExpansionCost (M (M (M (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 96)) x1) ((UInt256.ofNat 96) + (x1 + x0)) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup5 (by evm_kdecide) (by evm_ov)
  have r4 := r3.and (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup2 (by evm_kdecide) (by evm_ov)
  have r6 := RD.genMstore r5 (by evm_kdecide) (by evm_ov)
  have r7 := r6.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup3 (by evm_kdecide) (by evm_ov)
  have r10 := r9.add (by evm_kdecide) (by evm_ov)
  have r11 := RD.genMstore r10 (by evm_kdecide) (by evm_ov)
  have r12 := r11.dup2 (by evm_kdecide) (by evm_ov)
  have r13 := r12.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r14 := r13.dup3 (by evm_kdecide) (by evm_ov)
  have r15 := r14.add (by evm_kdecide) (by evm_ov)
  have r16 := RD.genMstore r15 (by evm_kdecide) (by evm_ov)
  have r17 := r16.dup2 (by evm_kdecide) (by evm_ov)
  have r18 := r17.dup4 (by evm_kdecide) (by evm_ov)
  have r19 := r18.push1 (UInt256.ofNat 96) (by evm_kdecide) (by evm_ov)
  have r20 := r19.dup4 (by evm_kdecide) (by evm_ov)
  have r21 := r20.add (by evm_kdecide) (by evm_ov)
  have r22 := RD.genCalldatacopy r21 (by evm_kdecide) (by evm_ov)
  have r23 := r22.push0 (by evm_kdecide) (by evm_ov)
  have r24 := r23.dup2 (by evm_kdecide) (by evm_ov)
  have r25 := r24.dup4 (by evm_kdecide) (by evm_ov)
  have r26 := r25.add (by evm_kdecide) (by evm_ov)
  have r27 := r26.push1 (UInt256.ofNat 96) (by evm_kdecide) (by evm_ov)
  have r28 := r27.swap1 (by evm_kdecide) (by evm_ov)
  have r29 := r28.dup2 (by evm_kdecide) (by evm_ov)
  have r30 := r29.add (by evm_kdecide) (by evm_ov)
  have r31 := r30.swap2 (by evm_kdecide) (by evm_ov)
  have r32 := r31.swap1 (by evm_kdecide) (by evm_ov)
  have r33 := r32.swap2 (by evm_kdecide) (by evm_ov)
  have r34 := RD.genMstore r33 (by evm_kdecide) (by evm_ov)
  have r35 := r34.push1 (UInt256.ofNat 31) (by evm_kdecide) (by evm_ov)
  have r36 := r35.swap1 (by evm_kdecide) (by evm_ov)
  have r37 := r36.swap3 (by evm_kdecide) (by evm_ov)
  have r38 := r37.add (by evm_kdecide) (by evm_ov)
  have r39 := r38.pushConst (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r40 := r39.and (by evm_kdecide) (by evm_ov)
  have r41 := r40.add (by evm_kdecide) (by evm_ov)
  have r42 := r41.add (by evm_kdecide) (by evm_ov)
  have r43 := r42.swap3 (by evm_kdecide) (by evm_ov)
  have r44 := r43.swap2 (by evm_kdecide) (by evm_ov)
  have r45 := r44.pop (by evm_kdecide) (by evm_ov)
  have r46 := r45.pop (by evm_kdecide) (by evm_ov)
  have r47 := r46.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r47 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_5893_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x4 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5893) (x0 :: x1 :: x2 :: x3 :: x4 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 x4 (l2tol2_block_5893_stack (x0 := x0) (x1 := x1) (R := R)) (l2tol2_block_5893_memory (ee := ee) (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_5893 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_5998_taken`. -/
def l2tol2_block_5998_taken_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((x1 + x0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 5998. -/
theorem l2tol2_block_5998_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt x0 (x1 + x0))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 4829) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5998) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4829) (l2tol2_block_5998_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 10) (C + ((35))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup3 (by evm_kdecide) (by evm_ov)
  have r4 := r3.add (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup3 (by evm_kdecide) (by evm_ov)
  have r7 := r6.gt (by evm_kdecide) (by evm_ov)
  have r8 := r7.iszero (by evm_kdecide) (by evm_ov)
  have r9 := r8.push2 (UInt256.ofNat 4829) (by evm_kdecide) (by evm_ov)
  have r10 := r9.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 4829)) r10 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_5998_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt x0 (x1 + x0))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 4829) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5998) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4829) (l2tol2_block_5998_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_5998_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_5998_fallthrough`. -/
def l2tol2_block_5998_fallthrough_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((x1 + x0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 5998. -/
theorem l2tol2_block_5998_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt x0 (x1 + x0))) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5998) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6010) (l2tol2_block_5998_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 10) (C + ((35))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup3 (by evm_kdecide) (by evm_ov)
  have r4 := r3.add (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup3 (by evm_kdecide) (by evm_ov)
  have r7 := r6.gt (by evm_kdecide) (by evm_ov)
  have r8 := r7.iszero (by evm_kdecide) (by evm_ov)
  have r9 := r8.push2 (UInt256.ofNat 4829) (by evm_kdecide) (by evm_ov)
  have r10 := r9.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 6010)) r10 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_5998_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt x0 (x1 + x0))) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5998) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6010) (l2tol2_block_5998_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_5998_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_6010`. -/
def l2tol2_block_6010_stack {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 4829) :: R)

/-- Automatically generated RD summary for bytecode block at pc 6010. -/
theorem l2tol2_block_6010 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 5738) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6010) R mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5738) (l2tol2_block_6010_stack (R := R)) mem aw rdata σ (k + 3) (C + ((14))) := by
  let r0 := h
  have r1 := r0.push2 (UInt256.ofNat 4829) (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 5738) (by evm_kdecide) (by evm_ov)
  have r3 := r2.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 5738)) r3 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_6010_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 5738) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6010) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5738) (l2tol2_block_6010_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_6010 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_6017_taken`. -/
def l2tol2_block_6017_taken_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((⟨0⟩ : UInt256) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 6017. -/
theorem l2tol2_block_6017_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 32))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 6033) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6017) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6033) (l2tol2_block_6017_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 10) (C + ((34))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push0 (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup5 (by evm_kdecide) (by evm_ov)
  have r6 := r5.sub (by evm_kdecide) (by evm_ov)
  have r7 := r6.slt (by evm_kdecide) (by evm_ov)
  have r8 := r7.iszero (by evm_kdecide) (by evm_ov)
  have r9 := r8.push2 (UInt256.ofNat 6033) (by evm_kdecide) (by evm_ov)
  have r10 := r9.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 6033)) r10 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_6017_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 32))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 6033) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6017) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6033) (l2tol2_block_6017_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_6017_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_6017_fallthrough`. -/
def l2tol2_block_6017_fallthrough_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((⟨0⟩ : UInt256) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 6017. -/
theorem l2tol2_block_6017_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 32))) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6017) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6030) (l2tol2_block_6017_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 10) (C + ((34))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push0 (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup5 (by evm_kdecide) (by evm_ov)
  have r6 := r5.sub (by evm_kdecide) (by evm_ov)
  have r7 := r6.slt (by evm_kdecide) (by evm_ov)
  have r8 := r7.iszero (by evm_kdecide) (by evm_ov)
  have r9 := r8.push2 (UInt256.ofNat 6033) (by evm_kdecide) (by evm_ov)
  have r10 := r9.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 6030)) r10 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_6017_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 32))) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6017) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6030) (l2tol2_block_6017_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_6017_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 6030. -/
theorem l2tol2_block_6030 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6030) R mem aw rdata σ k C)
    : RDrev ExpiryEvm.l2tol2Runtime g s0 := by
  let r0 := h
  have r1 := r0.push0 (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `l2tol2_block_6033`. -/
def l2tol2_block_6033_stack {ee : ExecutionEnv} {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes x1.toNat 32)) :: (UInt256.ofNat 5397) :: (uInt256OfByteArray (ee.calldata.readBytes x1.toNat 32)) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 6033. -/
theorem l2tol2_block_6033 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 5427) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6033) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5427) (l2tol2_block_6033_stack (ee := ee) (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 7) (C + ((24))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup2 (by evm_kdecide) (by evm_ov)
  have r3 := r2.calldataload (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 5397) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup2 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push2 (UInt256.ofNat 5427) (by evm_kdecide) (by evm_ov)
  have r7 := r6.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 5427)) r7 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_6033_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 5427) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6033) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5427) (l2tol2_block_6033_stack (ee := ee) (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_6033 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_6044`. -/
def l2tol2_block_6044_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((x1 + x0) :: R)

/-- Final memory for bytecode block summary `l2tol2_block_6044`. -/
def l2tol2_block_6044_memory {ee : ExecutionEnv} {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} : ByteArray :=
  ((⟨0⟩ : UInt256).toByteArray.write 0 (ee.calldata.write x2.toNat mem x0.toNat x1.toNat) (x1 + x0).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 6044. -/
theorem l2tol2_block_6044 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x3 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6044) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 x3 (l2tol2_block_6044_stack (x0 := x0) (x1 := x1) (R := R)) (l2tol2_block_6044_memory (ee := ee) (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2)) (M (M aw x0 x1) (x1 + x0) (⟨32⟩ : UInt256)) rdata σ (k + 15) (C + ((43) + (memExpansionCost aw x0 x1) + (3 + 3 * ((x1.toNat + 31) / 32)) + (memExpansionCost (M aw x0 x1) (x1 + x0) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup2 (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup4 (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup3 (by evm_kdecide) (by evm_ov)
  have r5 := RD.genCalldatacopy r4 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push0 (by evm_kdecide) (by evm_ov)
  have r7 := r6.swap2 (by evm_kdecide) (by evm_ov)
  have r8 := r7.add (by evm_kdecide) (by evm_ov)
  have r9 := r8.swap1 (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup2 (by evm_kdecide) (by evm_ov)
  have r11 := RD.genMstore r10 (by evm_kdecide) (by evm_ov)
  have r12 := r11.swap2 (by evm_kdecide) (by evm_ov)
  have r13 := r12.swap1 (by evm_kdecide) (by evm_ov)
  have r14 := r13.pop (by evm_kdecide) (by evm_ov)
  have r15 := r14.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r15 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_6044_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x3 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6044) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 x3 (l2tol2_block_6044_stack (x0 := x0) (x1 := x1) (R := R)) (l2tol2_block_6044_memory (ee := ee) (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_6044 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_6059`. -/
def l2tol2_block_6059_stack {ee : ExecutionEnv} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes x2.toNat 32)) :: (UInt256.ofNat 6074) :: (uInt256OfByteArray (ee.calldata.readBytes x2.toNat 32)) :: (x0 + (UInt256.ofNat 192)) :: x0 :: x1 :: x2 :: R)

/-- Automatically generated RD summary for bytecode block at pc 6059. -/
theorem l2tol2_block_6059 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 5427) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6059) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5427) (l2tol2_block_6059_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (R := R)) mem aw rdata σ (k + 10) (C + ((33))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 192) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.add (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup4 (by evm_kdecide) (by evm_ov)
  have r6 := r5.calldataload (by evm_kdecide) (by evm_ov)
  have r7 := r6.push2 (UInt256.ofNat 6074) (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup2 (by evm_kdecide) (by evm_ov)
  have r9 := r8.push2 (UInt256.ofNat 5427) (by evm_kdecide) (by evm_ov)
  have r10 := r9.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 5427)) r10 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_6059_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 5427) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6059) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5427) (l2tol2_block_6059_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_6059 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_6074`. -/
def l2tol2_block_6074_stack {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  (x1 :: R)

/-- Final memory for bytecode block summary `l2tol2_block_6074`. -/
def l2tol2_block_6074_memory {ee : ExecutionEnv} {mem : ByteArray} {x0 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} : ByteArray :=
  (x3.toByteArray.write 0 ((uInt256OfByteArray (ee.calldata.readBytes ((UInt256.ofNat 128) + x4).toNat 32)).toByteArray.write 0 ((uInt256OfByteArray (ee.calldata.readBytes (x4 + (UInt256.ofNat 96)).toNat 32)).toByteArray.write 0 ((uInt256OfByteArray (ee.calldata.readBytes (x4 + (UInt256.ofNat 64)).toNat 32)).toByteArray.write 0 ((uInt256OfByteArray (ee.calldata.readBytes ((UInt256.ofNat 32) + x4).toNat 32)).toByteArray.write 0 ((UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) x0).toByteArray.write 0 mem x2.toNat 32) (x2 + (UInt256.ofNat 32)).toNat 32) (x2 + (UInt256.ofNat 64)).toNat 32) (x2 + (UInt256.ofNat 96)).toNat 32) (x2 + (UInt256.ofNat 128)).toNat 32) ((UInt256.ofNat 160) + x2).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 6074. -/
theorem l2tol2_block_6074 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x5 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6074) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 x5 (l2tol2_block_6074_stack (x1 := x1) (R := R)) (l2tol2_block_6074_memory (ee := ee) (mem := mem) (x0 := x0) (x2 := x2) (x3 := x3) (x4 := x4)) (M (M (M (M (M (M aw x2 (⟨32⟩ : UInt256)) (x2 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x2 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) (x2 + (UInt256.ofNat 96)) (⟨32⟩ : UInt256)) (x2 + (UInt256.ofNat 128)) (⟨32⟩ : UInt256)) ((UInt256.ofNat 160) + x2) (⟨32⟩ : UInt256)) rdata σ (k + 49) (C + ((150) + (memExpansionCost aw x2 (⟨32⟩ : UInt256)) + (memExpansionCost (M aw x2 (⟨32⟩ : UInt256)) (x2 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw x2 (⟨32⟩ : UInt256)) (x2 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x2 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M aw x2 (⟨32⟩ : UInt256)) (x2 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x2 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) (x2 + (UInt256.ofNat 96)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M aw x2 (⟨32⟩ : UInt256)) (x2 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x2 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) (x2 + (UInt256.ofNat 96)) (⟨32⟩ : UInt256)) (x2 + (UInt256.ofNat 128)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M (M aw x2 (⟨32⟩ : UInt256)) (x2 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x2 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) (x2 + (UInt256.ofNat 96)) (⟨32⟩ : UInt256)) (x2 + (UInt256.ofNat 128)) (⟨32⟩ : UInt256)) ((UInt256.ofNat 160) + x2) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r3 := r2.and (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup3 (by evm_kdecide) (by evm_ov)
  have r5 := RD.genMstore r4 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup5 (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup2 (by evm_kdecide) (by evm_ov)
  have r9 := r8.add (by evm_kdecide) (by evm_ov)
  have r10 := r9.calldataload (by evm_kdecide) (by evm_ov)
  have r11 := r10.swap1 (by evm_kdecide) (by evm_ov)
  have r12 := r11.dup4 (by evm_kdecide) (by evm_ov)
  have r13 := r12.add (by evm_kdecide) (by evm_ov)
  have r14 := RD.genMstore r13 (by evm_kdecide) (by evm_ov)
  have r15 := r14.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r16 := r15.dup1 (by evm_kdecide) (by evm_ov)
  have r17 := r16.dup6 (by evm_kdecide) (by evm_ov)
  have r18 := r17.add (by evm_kdecide) (by evm_ov)
  have r19 := r18.calldataload (by evm_kdecide) (by evm_ov)
  have r20 := r19.swap1 (by evm_kdecide) (by evm_ov)
  have r21 := r20.dup4 (by evm_kdecide) (by evm_ov)
  have r22 := r21.add (by evm_kdecide) (by evm_ov)
  have r23 := RD.genMstore r22 (by evm_kdecide) (by evm_ov)
  have r24 := r23.push1 (UInt256.ofNat 96) (by evm_kdecide) (by evm_ov)
  have r25 := r24.dup1 (by evm_kdecide) (by evm_ov)
  have r26 := r25.dup6 (by evm_kdecide) (by evm_ov)
  have r27 := r26.add (by evm_kdecide) (by evm_ov)
  have r28 := r27.calldataload (by evm_kdecide) (by evm_ov)
  have r29 := r28.swap1 (by evm_kdecide) (by evm_ov)
  have r30 := r29.dup4 (by evm_kdecide) (by evm_ov)
  have r31 := r30.add (by evm_kdecide) (by evm_ov)
  have r32 := RD.genMstore r31 (by evm_kdecide) (by evm_ov)
  have r33 := r32.push1 (UInt256.ofNat 128) (by evm_kdecide) (by evm_ov)
  have r34 := r33.swap4 (by evm_kdecide) (by evm_ov)
  have r35 := r34.dup5 (by evm_kdecide) (by evm_ov)
  have r36 := r35.add (by evm_kdecide) (by evm_ov)
  have r37 := r36.calldataload (by evm_kdecide) (by evm_ov)
  have r38 := r37.swap4 (by evm_kdecide) (by evm_ov)
  have r39 := r38.dup3 (by evm_kdecide) (by evm_ov)
  have r40 := r39.add (by evm_kdecide) (by evm_ov)
  have r41 := r40.swap4 (by evm_kdecide) (by evm_ov)
  have r42 := r41.swap1 (by evm_kdecide) (by evm_ov)
  have r43 := r42.swap4 (by evm_kdecide) (by evm_ov)
  have r44 := RD.genMstore r43 (by evm_kdecide) (by evm_ov)
  have r45 := r44.push1 (UInt256.ofNat 160) (by evm_kdecide) (by evm_ov)
  have r46 := r45.add (by evm_kdecide) (by evm_ov)
  have r47 := RD.genMstore r46 (by evm_kdecide) (by evm_ov)
  have r48 := r47.swap1 (by evm_kdecide) (by evm_ov)
  have r49 := r48.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r49 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_6074_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x5 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6074) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 x5 (l2tol2_block_6074_stack (x1 := x1) (R := R)) (l2tol2_block_6074_memory (ee := ee) (mem := mem) (x0 := x0) (x2 := x2) (x3 := x3) (x4 := x4)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_6074 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_6148`. -/
def l2tol2_block_6148_stack {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((x0 + (memLoad x1 mem)) :: R)

/-- Final memory for bytecode block summary `l2tol2_block_6148`. -/
def l2tol2_block_6148_memory {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} : ByteArray :=
  ((⟨0⟩ : UInt256).toByteArray.write 0 (mem.write (x1 + (UInt256.ofNat 32)).toNat mem x0.toNat (memLoad x1 mem).toNat) (x0 + (memLoad x1 mem)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 6148. -/
theorem l2tol2_block_6148 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x2 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6148) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 x2 (l2tol2_block_6148_stack (mem := mem) (x0 := x0) (x1 := x1) (R := R)) (l2tol2_block_6148_memory (mem := mem) (x0 := x0) (x1 := x1)) (M (Mmcopy (M aw x1 (⟨32⟩ : UInt256)) x0 (x1 + (UInt256.ofNat 32)) (memLoad x1 mem)) (x0 + (memLoad x1 mem)) (⟨32⟩ : UInt256)) rdata σ (k + 21) (C + ((59) + (memExpansionCost aw x1 (⟨32⟩ : UInt256)) + (mcopyExpansionCost (M aw x1 (⟨32⟩ : UInt256)) x0 (x1 + (UInt256.ofNat 32)) (memLoad x1 mem)) + (3 + 3 * (((memLoad x1 mem).toNat + 31) / 32)) + (memExpansionCost (Mmcopy (M aw x1 (⟨32⟩ : UInt256)) x0 (x1 + (UInt256.ofNat 32)) (memLoad x1 mem)) (x0 + (memLoad x1 mem)) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push0 (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup3 (by evm_kdecide) (by evm_ov)
  have r4 := RD.genMload r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup6 (by evm_kdecide) (by evm_ov)
  have r8 := r7.add (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup5 (by evm_kdecide) (by evm_ov)
  have r10 := RD.genMcopy r9 (by evm_kdecide) (by evm_ov)
  have r11 := r10.push0 (by evm_kdecide) (by evm_ov)
  have r12 := r11.swap3 (by evm_kdecide) (by evm_ov)
  have r13 := r12.add (by evm_kdecide) (by evm_ov)
  have r14 := r13.swap2 (by evm_kdecide) (by evm_ov)
  have r15 := r14.dup3 (by evm_kdecide) (by evm_ov)
  have r16 := RD.genMstore r15 (by evm_kdecide) (by evm_ov)
  have r17 := r16.pop (by evm_kdecide) (by evm_ov)
  have r18 := r17.swap2 (by evm_kdecide) (by evm_ov)
  have r19 := r18.swap1 (by evm_kdecide) (by evm_ov)
  have r20 := r19.pop (by evm_kdecide) (by evm_ov)
  have r21 := r20.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r21 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_6148_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x2 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6148) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 x2 (l2tol2_block_6148_stack (mem := mem) (x0 := x0) (x1 := x1) (R := R)) (l2tol2_block_6148_memory (mem := mem) (x0 := x0) (x1 := x1)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_6148 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_6170`. -/
def l2tol2_block_6170_stack {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {x6 : UInt256} {R : List UInt256} : List UInt256 :=
  (x1 :: (x0 + (UInt256.ofNat 192)) :: (UInt256.ofNat 6244) :: (⟨0⟩ : UInt256) :: x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: R)

/-- Final memory for bytecode block summary `l2tol2_block_6170`. -/
def l2tol2_block_6170_memory {mem : ByteArray} {x0 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {x6 : UInt256} : ByteArray :=
  ((UInt256.ofNat 192).toByteArray.write 0 ((UInt256.land x2 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.land x3 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 (x4.toByteArray.write 0 (x5.toByteArray.write 0 (x6.toByteArray.write 0 mem x0.toNat 32) (x0 + (UInt256.ofNat 32)).toNat 32) (x0 + (UInt256.ofNat 64)).toNat 32) (x0 + (UInt256.ofNat 96)).toNat 32) (x0 + (UInt256.ofNat 128)).toNat 32) (x0 + (UInt256.ofNat 160)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 6170. -/
theorem l2tol2_block_6170 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 5303) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6170) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5303) (l2tol2_block_6170_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (R := R)) (l2tol2_block_6170_memory (mem := mem) (x0 := x0) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6)) (M (M (M (M (M (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 96)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 128)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 160)) (⟨32⟩ : UInt256)) rdata σ (k + 43) (C + ((130) + (memExpansionCost aw x0 (⟨32⟩ : UInt256)) + (memExpansionCost (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 96)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 96)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 128)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 96)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 128)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 160)) (⟨32⟩ : UInt256)))) := by
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
  have r15 := r14.push0 (by evm_kdecide) (by evm_ov)
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
  have r37 := r36.push2 (UInt256.ofNat 6244) (by evm_kdecide) (by evm_ov)
  have r38 := r37.push1 (UInt256.ofNat 192) (by evm_kdecide) (by evm_ov)
  have r39 := r38.dup4 (by evm_kdecide) (by evm_ov)
  have r40 := r39.add (by evm_kdecide) (by evm_ov)
  have r41 := r40.dup5 (by evm_kdecide) (by evm_ov)
  have r42 := r41.push2 (UInt256.ofNat 5303) (by evm_kdecide) (by evm_ov)
  have r43 := r42.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 5303)) r43 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_6170_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 5303) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6170) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 5303) (l2tol2_block_6170_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (R := R)) (l2tol2_block_6170_memory (mem := mem) (x0 := x0) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_6170 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_6244`. -/
def l2tol2_block_6244_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: R)

/-- Automatically generated RD summary for bytecode block at pc 6244. -/
theorem l2tol2_block_6244 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x9 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6244) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 x9 (l2tol2_block_6244_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 12) (C + ((31))) := by
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
theorem l2tol2_block_6244_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x9 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6244) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 x9 (l2tol2_block_6244_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_6244 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_6256_taken`. -/
def l2tol2_block_6256_taken_stack {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {R : List UInt256} : List UInt256 :=
  ((⟨0⟩ : UInt256) :: (⟨0⟩ : UInt256) :: x0 :: x1 :: x2 :: x3 :: R)

/-- Automatically generated RD summary for bytecode block at pc 6256. -/
theorem l2tol2_block_6256_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt x2 x3)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 6270) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6256) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6270) (l2tol2_block_6256_taken_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (R := R)) mem aw rdata σ (k + 9) (C + ((31))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push0 (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup6 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup6 (by evm_kdecide) (by evm_ov)
  have r6 := r5.gt (by evm_kdecide) (by evm_ov)
  have r7 := r6.iszero (by evm_kdecide) (by evm_ov)
  have r8 := r7.push2 (UInt256.ofNat 6270) (by evm_kdecide) (by evm_ov)
  have r9 := r8.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 6270)) r9 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_6256_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt x2 x3)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 6270) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6256) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6270) (l2tol2_block_6256_taken_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_6256_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_6256_fallthrough`. -/
def l2tol2_block_6256_fallthrough_stack {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {R : List UInt256} : List UInt256 :=
  ((⟨0⟩ : UInt256) :: (⟨0⟩ : UInt256) :: x0 :: x1 :: x2 :: x3 :: R)

/-- Automatically generated RD summary for bytecode block at pc 6256. -/
theorem l2tol2_block_6256_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt x2 x3)) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6256) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6267) (l2tol2_block_6256_fallthrough_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (R := R)) mem aw rdata σ (k + 9) (C + ((31))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push0 (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup6 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup6 (by evm_kdecide) (by evm_ov)
  have r6 := r5.gt (by evm_kdecide) (by evm_ov)
  have r7 := r6.iszero (by evm_kdecide) (by evm_ov)
  have r8 := r7.push2 (UInt256.ofNat 6270) (by evm_kdecide) (by evm_ov)
  have r9 := r8.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 6267)) r9 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_6256_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt x2 x3)) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6256) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6267) (l2tol2_block_6256_fallthrough_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_6256_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 6267. -/
theorem l2tol2_block_6267 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6267) R mem aw rdata σ k C)
    : RDrev ExpiryEvm.l2tol2Runtime g s0 := by
  let r0 := h
  have r1 := r0.push0 (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Automatically generated RD summary for bytecode block at pc 6270. -/
theorem l2tol2_block_6270_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt x5 x3)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 6282) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6270) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6282) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ (k + 7) (C + ((26))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup4 (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup7 (by evm_kdecide) (by evm_ov)
  have r4 := r3.gt (by evm_kdecide) (by evm_ov)
  have r5 := r4.iszero (by evm_kdecide) (by evm_ov)
  have r6 := r5.push2 (UInt256.ofNat 6282) (by evm_kdecide) (by evm_ov)
  have r7 := r6.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 6282)) r7 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_6270_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt x5 x3)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 6282) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6270) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 6282) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_6270_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

end l2tol2Blocks
