import Reasoning.Reach
import L1cdmEvm.Bytecode

open Solm ABI Ethereum Ethereum.EVM
open Reasoning.Theory Reasoning.Reach

namespace l1cdmBlocks

/-- Final stack for bytecode block summary `l1cdm_block_10566`. -/
def l1cdm_block_10566_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: R)

/-- Final memory for bytecode block summary `l1cdm_block_10566`. -/
def l1cdm_block_10566_memory {mem : ByteArray} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} : ByteArray :=
  ((UInt256.land x3 (UInt256.ofNat 4294967295)).toByteArray.write 0 (x4.toByteArray.write 0 mem (x2 + (UInt256.ofNat 64)).toNat 32) (x2 + (UInt256.ofNat 96)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 10566. -/
theorem l1cdm_block_10566 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 11 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x8 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10566) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 x8 (l1cdm_block_10566_stack (x0 := x0) (R := R)) (l1cdm_block_10566_memory (mem := mem) (x2 := x2) (x3 := x3) (x4 := x4)) (M (M aw (x2 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) (x2 + (UInt256.ofNat 96)) (⟨32⟩ : UInt256)) rdata σ (k + 24) (C + ((68) + (memExpansionCost aw (x2 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (x2 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) (x2 + (UInt256.ofNat 96)) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pop (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup4 (by evm_kdecide) (by evm_ov)
  have r5 := r4.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup4 (by evm_kdecide) (by evm_ov)
  have r7 := r6.add (by evm_kdecide) (by evm_ov)
  have r8 := RD.genMstore r7 (by evm_kdecide) (by evm_ov)
  have r9 := r8.push4 (UInt256.ofNat 4294967295) (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup4 (by evm_kdecide) (by evm_ov)
  have r11 := r10.and (by evm_kdecide) (by evm_ov)
  have r12 := r11.push1 (UInt256.ofNat 96) (by evm_kdecide) (by evm_ov)
  have r13 := r12.dup4 (by evm_kdecide) (by evm_ov)
  have r14 := r13.add (by evm_kdecide) (by evm_ov)
  have r15 := RD.genMstore r14 (by evm_kdecide) (by evm_ov)
  have r16 := r15.swap7 (by evm_kdecide) (by evm_ov)
  have r17 := r16.swap6 (by evm_kdecide) (by evm_ov)
  have r18 := r17.pop (by evm_kdecide) (by evm_ov)
  have r19 := r18.pop (by evm_kdecide) (by evm_ov)
  have r20 := r19.pop (by evm_kdecide) (by evm_ov)
  have r21 := r20.pop (by evm_kdecide) (by evm_ov)
  have r22 := r21.pop (by evm_kdecide) (by evm_ov)
  have r23 := r22.pop (by evm_kdecide) (by evm_ov)
  have r24 := r23.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r24 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10566_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 11 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x8 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10566) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 x8 (l1cdm_block_10566_stack (x0 := x0) (R := R)) (l1cdm_block_10566_memory (mem := mem) (x2 := x2) (x3 := x3) (x4 := x4)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10566 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 10596. -/
theorem l1cdm_block_10596 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10596) R mem aw rdata σ k C)
    : RDrev L1cdmEvm.l1cdmRuntime g s0 := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pushConst (UInt256.ofNat 35408467139433450592217433187231851964531694900788300625387963629091585785856) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r4 := RD.genMstore r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.push1 (UInt256.ofNat 17) (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r7 := RD.genMstore r6 (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 36) (by evm_kdecide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  exact RD.genRev r9 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `l1cdm_block_10643_taken`. -/
def l1cdm_block_10643_taken_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 10643. -/
theorem l1cdm_block_10643_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.land (UInt256.isZero (UInt256.isZero x0)) (UInt256.gt x1 (UInt256.div (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639935) x0)))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10699) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10643) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10699) (l1cdm_block_10643_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 14) (C + ((49))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.pushConst (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639935) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r5 := r4.div (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup4 (by evm_kdecide) (by evm_ov)
  have r7 := r6.gt (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup3 (by evm_kdecide) (by evm_ov)
  have r9 := r8.iszero (by evm_kdecide) (by evm_ov)
  have r10 := r9.iszero (by evm_kdecide) (by evm_ov)
  have r11 := r10.and (by evm_kdecide) (by evm_ov)
  have r12 := r11.iszero (by evm_kdecide) (by evm_ov)
  have r13 := r12.push2 (UInt256.ofNat 10699) (by evm_kdecide) (by evm_ov)
  have r14 := r13.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10699)) r14 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10643_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.land (UInt256.isZero (UInt256.isZero x0)) (UInt256.gt x1 (UInt256.div (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639935) x0)))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10699) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10643) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10699) (l1cdm_block_10643_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10643_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10643_fallthrough`. -/
def l1cdm_block_10643_fallthrough_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 10643. -/
theorem l1cdm_block_10643_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.land (UInt256.isZero (UInt256.isZero x0)) (UInt256.gt x1 (UInt256.div (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639935) x0)))) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10643) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10692) (l1cdm_block_10643_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 14) (C + ((49))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.pushConst (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639935) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r5 := r4.div (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup4 (by evm_kdecide) (by evm_ov)
  have r7 := r6.gt (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup3 (by evm_kdecide) (by evm_ov)
  have r9 := r8.iszero (by evm_kdecide) (by evm_ov)
  have r10 := r9.iszero (by evm_kdecide) (by evm_ov)
  have r11 := r10.and (by evm_kdecide) (by evm_ov)
  have r12 := r11.iszero (by evm_kdecide) (by evm_ov)
  have r13 := r12.push2 (UInt256.ofNat 10699) (by evm_kdecide) (by evm_ov)
  have r14 := r13.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10692)) r14 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10643_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.land (UInt256.isZero (UInt256.isZero x0)) (UInt256.gt x1 (UInt256.div (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639935) x0)))) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10643) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10692) (l1cdm_block_10643_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10643_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10692`. -/
def l1cdm_block_10692_stack {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 10699) :: R)

/-- Automatically generated RD summary for bytecode block at pc 10692. -/
theorem l1cdm_block_10692 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10596) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10692) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10596) (l1cdm_block_10692_stack (R := R)) mem aw rdata σ (k + 3) (C + ((14))) := by
  let r0 := h
  have r1 := r0.push2 (UInt256.ofNat 10699) (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 10596) (by evm_kdecide) (by evm_ov)
  have r3 := r2.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10596)) r3 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10692_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10596) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10692) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10596) (l1cdm_block_10692_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10692 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10699`. -/
def l1cdm_block_10699_stack {x1 : UInt256} {x2 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.mul x1 x2) :: R)

/-- Automatically generated RD summary for bytecode block at pc 10699. -/
theorem l1cdm_block_10699 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x3 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10699) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 x3 (l1cdm_block_10699_stack (x1 := x1) (x2 := x2) (R := R)) mem aw rdata σ (k + 5) (C + ((19))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := r2.mul (by evm_kdecide) (by evm_ov)
  have r4 := r3.swap1 (by evm_kdecide) (by evm_ov)
  have r5 := r4.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r5 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10699_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x3 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10699) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 x3 (l1cdm_block_10699_stack (x1 := x1) (x2 := x2) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10699 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10704_taken`. -/
def l1cdm_block_10704_taken_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.land x1 (UInt256.ofNat 18446744073709551615)) :: (UInt256.land x0 (UInt256.ofNat 18446744073709551615)) :: (UInt256.ofNat 18446744073709551615) :: (UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 10704. -/
theorem l1cdm_block_10704_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.land (UInt256.isZero (UInt256.isZero (UInt256.land x0 (UInt256.ofNat 18446744073709551615)))) (UInt256.gt (UInt256.land x1 (UInt256.ofNat 18446744073709551615)) (UInt256.div (UInt256.ofNat 18446744073709551615) (UInt256.land x0 (UInt256.ofNat 18446744073709551615)))))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10743) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10704) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10743) (l1cdm_block_10704_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 21) (C + ((70))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.pushConst (UInt256.ofNat 18446744073709551615) (width := 8) (op := .PUSH8) (by decide) (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup1 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup4 (by evm_kdecide) (by evm_ov)
  have r6 := r5.and (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup2 (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup6 (by evm_kdecide) (by evm_ov)
  have r9 := r8.and (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup2 (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup4 (by evm_kdecide) (by evm_ov)
  have r12 := r11.div (by evm_kdecide) (by evm_ov)
  have r13 := r12.dup2 (by evm_kdecide) (by evm_ov)
  have r14 := r13.gt (by evm_kdecide) (by evm_ov)
  have r15 := r14.dup3 (by evm_kdecide) (by evm_ov)
  have r16 := r15.iszero (by evm_kdecide) (by evm_ov)
  have r17 := r16.iszero (by evm_kdecide) (by evm_ov)
  have r18 := r17.and (by evm_kdecide) (by evm_ov)
  have r19 := r18.iszero (by evm_kdecide) (by evm_ov)
  have r20 := r19.push2 (UInt256.ofNat 10743) (by evm_kdecide) (by evm_ov)
  have r21 := r20.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10743)) r21 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10704_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.land (UInt256.isZero (UInt256.isZero (UInt256.land x0 (UInt256.ofNat 18446744073709551615)))) (UInt256.gt (UInt256.land x1 (UInt256.ofNat 18446744073709551615)) (UInt256.div (UInt256.ofNat 18446744073709551615) (UInt256.land x0 (UInt256.ofNat 18446744073709551615)))))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10743) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10704) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10743) (l1cdm_block_10704_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10704_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10704_fallthrough`. -/
def l1cdm_block_10704_fallthrough_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.land x1 (UInt256.ofNat 18446744073709551615)) :: (UInt256.land x0 (UInt256.ofNat 18446744073709551615)) :: (UInt256.ofNat 18446744073709551615) :: (UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 10704. -/
theorem l1cdm_block_10704_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.land (UInt256.isZero (UInt256.isZero (UInt256.land x0 (UInt256.ofNat 18446744073709551615)))) (UInt256.gt (UInt256.land x1 (UInt256.ofNat 18446744073709551615)) (UInt256.div (UInt256.ofNat 18446744073709551615) (UInt256.land x0 (UInt256.ofNat 18446744073709551615)))))) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10704) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10736) (l1cdm_block_10704_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 21) (C + ((70))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.pushConst (UInt256.ofNat 18446744073709551615) (width := 8) (op := .PUSH8) (by decide) (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup1 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup4 (by evm_kdecide) (by evm_ov)
  have r6 := r5.and (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup2 (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup6 (by evm_kdecide) (by evm_ov)
  have r9 := r8.and (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup2 (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup4 (by evm_kdecide) (by evm_ov)
  have r12 := r11.div (by evm_kdecide) (by evm_ov)
  have r13 := r12.dup2 (by evm_kdecide) (by evm_ov)
  have r14 := r13.gt (by evm_kdecide) (by evm_ov)
  have r15 := r14.dup3 (by evm_kdecide) (by evm_ov)
  have r16 := r15.iszero (by evm_kdecide) (by evm_ov)
  have r17 := r16.iszero (by evm_kdecide) (by evm_ov)
  have r18 := r17.and (by evm_kdecide) (by evm_ov)
  have r19 := r18.iszero (by evm_kdecide) (by evm_ov)
  have r20 := r19.push2 (UInt256.ofNat 10743) (by evm_kdecide) (by evm_ov)
  have r21 := r20.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10736)) r21 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10704_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.land (UInt256.isZero (UInt256.isZero (UInt256.land x0 (UInt256.ofNat 18446744073709551615)))) (UInt256.gt (UInt256.land x1 (UInt256.ofNat 18446744073709551615)) (UInt256.div (UInt256.ofNat 18446744073709551615) (UInt256.land x0 (UInt256.ofNat 18446744073709551615)))))) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10704) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10736) (l1cdm_block_10704_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10704_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10736`. -/
def l1cdm_block_10736_stack {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 10743) :: R)

/-- Automatically generated RD summary for bytecode block at pc 10736. -/
theorem l1cdm_block_10736 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10596) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10736) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10596) (l1cdm_block_10736_stack (R := R)) mem aw rdata σ (k + 3) (C + ((14))) := by
  let r0 := h
  have r1 := r0.push2 (UInt256.ofNat 10743) (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 10596) (by evm_kdecide) (by evm_ov)
  have r3 := r2.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10596)) r3 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10736_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10596) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10736) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10596) (l1cdm_block_10736_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10736 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10743`. -/
def l1cdm_block_10743_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.mul x0 x1) :: R)

/-- Automatically generated RD summary for bytecode block at pc 10743. -/
theorem l1cdm_block_10743 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x6 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10743) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 x6 (l1cdm_block_10743_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 9) (C + ((28))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.mul (by evm_kdecide) (by evm_ov)
  have r3 := r2.swap5 (by evm_kdecide) (by evm_ov)
  have r4 := r3.swap4 (by evm_kdecide) (by evm_ov)
  have r5 := r4.pop (by evm_kdecide) (by evm_ov)
  have r6 := r5.pop (by evm_kdecide) (by evm_ov)
  have r7 := r6.pop (by evm_kdecide) (by evm_ov)
  have r8 := r7.pop (by evm_kdecide) (by evm_ov)
  have r9 := r8.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r9 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10743_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x6 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10743) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 x6 (l1cdm_block_10743_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10743 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10752_taken`. -/
def l1cdm_block_10752_taken_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.land x1 (UInt256.ofNat 18446744073709551615)) :: (UInt256.ofNat 18446744073709551615) :: (UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 10752. -/
theorem l1cdm_block_10752_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : (UInt256.land x1 (UInt256.ofNat 18446744073709551615)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10818) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10752) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10818) (l1cdm_block_10752_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 9) (C + ((32))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.pushConst (UInt256.ofNat 18446744073709551615) (width := 8) (op := .PUSH8) (by decide) (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup1 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup5 (by evm_kdecide) (by evm_ov)
  have r6 := r5.and (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup1 (by evm_kdecide) (by evm_ov)
  have r8 := r7.push2 (UInt256.ofNat 10818) (by evm_kdecide) (by evm_ov)
  have r9 := r8.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10818)) r9 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10752_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : (UInt256.land x1 (UInt256.ofNat 18446744073709551615)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10818) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10752) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10818) (l1cdm_block_10752_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10752_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10752_fallthrough`. -/
def l1cdm_block_10752_fallthrough_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.land x1 (UInt256.ofNat 18446744073709551615)) :: (UInt256.ofNat 18446744073709551615) :: (UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 10752. -/
theorem l1cdm_block_10752_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : (UInt256.land x1 (UInt256.ofNat 18446744073709551615)) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10752) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10772) (l1cdm_block_10752_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 9) (C + ((32))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.pushConst (UInt256.ofNat 18446744073709551615) (width := 8) (op := .PUSH8) (by decide) (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup1 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup5 (by evm_kdecide) (by evm_ov)
  have r6 := r5.and (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup1 (by evm_kdecide) (by evm_ov)
  have r8 := r7.push2 (UInt256.ofNat 10818) (by evm_kdecide) (by evm_ov)
  have r9 := r8.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10772)) r9 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10752_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : (UInt256.land x1 (UInt256.ofNat 18446744073709551615)) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10752) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10772) (l1cdm_block_10752_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10752_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 10772. -/
theorem l1cdm_block_10772 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10772) R mem aw rdata σ k C)
    : RDrev L1cdmEvm.l1cdmRuntime g s0 := by
  let r0 := h
  have r1 := r0.pushConst (UInt256.ofNat 35408467139433450592217433187231851964531694900788300625387963629091585785856) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := RD.genMstore r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 18) (by evm_kdecide) (by evm_ov)
  have r5 := r4.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r6 := RD.genMstore r5 (by evm_kdecide) (by evm_ov)
  have r7 := r6.push1 (UInt256.ofNat 36) (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  exact RD.genRev r8 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `l1cdm_block_10818`. -/
def l1cdm_block_10818_stack {x0 : UInt256} {x1 : UInt256} {x3 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.div (UInt256.land x3 x1) x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 10818. -/
theorem l1cdm_block_10818 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x5 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10818) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 x5 (l1cdm_block_10818_stack (x0 := x0) (x1 := x1) (x3 := x3) (R := R)) mem aw rdata σ (k + 12) (C + ((39))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap3 (by evm_kdecide) (by evm_ov)
  have r3 := r2.and (by evm_kdecide) (by evm_ov)
  have r4 := r3.swap2 (by evm_kdecide) (by evm_ov)
  have r5 := r4.swap1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.swap2 (by evm_kdecide) (by evm_ov)
  have r7 := r6.div (by evm_kdecide) (by evm_ov)
  have r8 := r7.swap3 (by evm_kdecide) (by evm_ov)
  have r9 := r8.swap2 (by evm_kdecide) (by evm_ov)
  have r10 := r9.pop (by evm_kdecide) (by evm_ov)
  have r11 := r10.pop (by evm_kdecide) (by evm_ov)
  have r12 := r11.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r12 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10818_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x5 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10818) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 x5 (l1cdm_block_10818_stack (x0 := x0) (x1 := x1) (x3 := x3) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10818 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10830_taken`. -/
def l1cdm_block_10830_taken_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.land x1 (UInt256.ofNat 18446744073709551615)) :: (UInt256.land x0 (UInt256.ofNat 18446744073709551615)) :: (UInt256.ofNat 18446744073709551615) :: (UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 10830. -/
theorem l1cdm_block_10830_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (UInt256.land x0 (UInt256.ofNat 18446744073709551615)) (UInt256.sub (UInt256.ofNat 18446744073709551615) (UInt256.land x1 (UInt256.ofNat 18446744073709551615))))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10865) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10830) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10865) (l1cdm_block_10830_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 17) (C + ((56))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.pushConst (UInt256.ofNat 18446744073709551615) (width := 8) (op := .PUSH8) (by decide) (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup1 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup4 (by evm_kdecide) (by evm_ov)
  have r6 := r5.and (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup2 (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup6 (by evm_kdecide) (by evm_ov)
  have r9 := r8.and (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup1 (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup4 (by evm_kdecide) (by evm_ov)
  have r12 := r11.sub (by evm_kdecide) (by evm_ov)
  have r13 := r12.dup3 (by evm_kdecide) (by evm_ov)
  have r14 := r13.gt (by evm_kdecide) (by evm_ov)
  have r15 := r14.iszero (by evm_kdecide) (by evm_ov)
  have r16 := r15.push2 (UInt256.ofNat 10865) (by evm_kdecide) (by evm_ov)
  have r17 := r16.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10865)) r17 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10830_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (UInt256.land x0 (UInt256.ofNat 18446744073709551615)) (UInt256.sub (UInt256.ofNat 18446744073709551615) (UInt256.land x1 (UInt256.ofNat 18446744073709551615))))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10865) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10830) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10865) (l1cdm_block_10830_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10830_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10830_fallthrough`. -/
def l1cdm_block_10830_fallthrough_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.land x1 (UInt256.ofNat 18446744073709551615)) :: (UInt256.land x0 (UInt256.ofNat 18446744073709551615)) :: (UInt256.ofNat 18446744073709551615) :: (UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 10830. -/
theorem l1cdm_block_10830_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (UInt256.land x0 (UInt256.ofNat 18446744073709551615)) (UInt256.sub (UInt256.ofNat 18446744073709551615) (UInt256.land x1 (UInt256.ofNat 18446744073709551615))))) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10830) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10858) (l1cdm_block_10830_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 17) (C + ((56))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.pushConst (UInt256.ofNat 18446744073709551615) (width := 8) (op := .PUSH8) (by decide) (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup1 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup4 (by evm_kdecide) (by evm_ov)
  have r6 := r5.and (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup2 (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup6 (by evm_kdecide) (by evm_ov)
  have r9 := r8.and (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup1 (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup4 (by evm_kdecide) (by evm_ov)
  have r12 := r11.sub (by evm_kdecide) (by evm_ov)
  have r13 := r12.dup3 (by evm_kdecide) (by evm_ov)
  have r14 := r13.gt (by evm_kdecide) (by evm_ov)
  have r15 := r14.iszero (by evm_kdecide) (by evm_ov)
  have r16 := r15.push2 (UInt256.ofNat 10865) (by evm_kdecide) (by evm_ov)
  have r17 := r16.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10858)) r17 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10830_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (UInt256.land x0 (UInt256.ofNat 18446744073709551615)) (UInt256.sub (UInt256.ofNat 18446744073709551615) (UInt256.land x1 (UInt256.ofNat 18446744073709551615))))) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10830) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10858) (l1cdm_block_10830_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10830_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10858`. -/
def l1cdm_block_10858_stack {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 10865) :: R)

/-- Automatically generated RD summary for bytecode block at pc 10858. -/
theorem l1cdm_block_10858 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10596) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10858) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10596) (l1cdm_block_10858_stack (R := R)) mem aw rdata σ (k + 3) (C + ((14))) := by
  let r0 := h
  have r1 := r0.push2 (UInt256.ofNat 10865) (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 10596) (by evm_kdecide) (by evm_ov)
  have r3 := r2.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10596)) r3 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10858_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10596) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10858) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10596) (l1cdm_block_10858_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10858 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10865`. -/
def l1cdm_block_10865_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((x0 + x1) :: R)

/-- Automatically generated RD summary for bytecode block at pc 10865. -/
theorem l1cdm_block_10865 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x6 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10865) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 x6 (l1cdm_block_10865_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 9) (C + ((26))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.add (by evm_kdecide) (by evm_ov)
  have r3 := r2.swap5 (by evm_kdecide) (by evm_ov)
  have r4 := r3.swap4 (by evm_kdecide) (by evm_ov)
  have r5 := r4.pop (by evm_kdecide) (by evm_ov)
  have r6 := r5.pop (by evm_kdecide) (by evm_ov)
  have r7 := r6.pop (by evm_kdecide) (by evm_ov)
  have r8 := r7.pop (by evm_kdecide) (by evm_ov)
  have r9 := r8.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r9 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10865_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x6 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10865) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 x6 (l1cdm_block_10865_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10865 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10874_taken`. -/
def l1cdm_block_10874_taken_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 10874. -/
theorem l1cdm_block_10874_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt x0 (UInt256.lnot x1))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10893) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10874) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10893) (l1cdm_block_10874_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 9) (C + ((32))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup3 (by evm_kdecide) (by evm_ov)
  have r4 := r3.not (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup3 (by evm_kdecide) (by evm_ov)
  have r6 := r5.gt (by evm_kdecide) (by evm_ov)
  have r7 := r6.iszero (by evm_kdecide) (by evm_ov)
  have r8 := r7.push2 (UInt256.ofNat 10893) (by evm_kdecide) (by evm_ov)
  have r9 := r8.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10893)) r9 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10874_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt x0 (UInt256.lnot x1))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10893) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10874) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10893) (l1cdm_block_10874_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10874_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10874_fallthrough`. -/
def l1cdm_block_10874_fallthrough_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 10874. -/
theorem l1cdm_block_10874_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt x0 (UInt256.lnot x1))) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10874) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10886) (l1cdm_block_10874_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 9) (C + ((32))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup3 (by evm_kdecide) (by evm_ov)
  have r4 := r3.not (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup3 (by evm_kdecide) (by evm_ov)
  have r6 := r5.gt (by evm_kdecide) (by evm_ov)
  have r7 := r6.iszero (by evm_kdecide) (by evm_ov)
  have r8 := r7.push2 (UInt256.ofNat 10893) (by evm_kdecide) (by evm_ov)
  have r9 := r8.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10886)) r9 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10874_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt x0 (UInt256.lnot x1))) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10874) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10886) (l1cdm_block_10874_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10874_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

end l1cdmBlocks
