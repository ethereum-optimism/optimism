import Reasoning.Reach
import L1cdmEvm.Bytecode

open Solm ABI Ethereum Ethereum.EVM
open Reasoning.Theory Reasoning.Reach

namespace l1cdmBlocks

/-- Final stack for bytecode block summary `l1cdm_block_9860_fallthrough`. -/
def l1cdm_block_9860_fallthrough_stack {ee : ExecutionEnv} {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes x1.toNat 32)) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 9860. -/
theorem l1cdm_block_9860_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes x1.toNat 32)) x0)) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9860) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9871) (l1cdm_block_9860_fallthrough_stack (ee := ee) (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 9) (C + ((32))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup2 (by evm_kdecide) (by evm_ov)
  have r3 := r2.calldataload (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup2 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup2 (by evm_kdecide) (by evm_ov)
  have r6 := r5.gt (by evm_kdecide) (by evm_ov)
  have r7 := r6.iszero (by evm_kdecide) (by evm_ov)
  have r8 := r7.push2 (UInt256.ofNat 9878) (by evm_kdecide) (by evm_ov)
  have r9 := r8.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 9871)) r9 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_9860_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes x1.toNat 32)) x0)) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9860) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9871) (l1cdm_block_9860_fallthrough_stack (ee := ee) (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_9860_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_9871`. -/
def l1cdm_block_9871_stack {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 9878) :: R)

/-- Automatically generated RD summary for bytecode block at pc 9871. -/
theorem l1cdm_block_9871 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9750) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9871) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9750) (l1cdm_block_9871_stack (R := R)) mem aw rdata σ (k + 3) (C + ((14))) := by
  let r0 := h
  have r1 := r0.push2 (UInt256.ofNat 9878) (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 9750) (by evm_kdecide) (by evm_ov)
  have r3 := r2.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 9750)) r3 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_9871_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9750) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9871) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9750) (l1cdm_block_9871_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_9871 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_9878_taken`. -/
def l1cdm_block_9878_taken_stack {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((memLoad (UInt256.ofNat 64) mem) :: ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 9878. -/
theorem l1cdm_block_9878_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.lor (UInt256.lt ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) (memLoad (UInt256.ofNat 64) mem)) (UInt256.gt ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) x1))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9948) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9878) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9948) (l1cdm_block_9878_taken_stack (mem := mem) (x0 := x0) (x1 := x1) (R := R)) mem (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 26) (C + ((83) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r3 := RD.genMload r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 31) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup3 (by evm_kdecide) (by evm_ov)
  have r6 := r5.add (by evm_kdecide) (by evm_ov)
  have r7 := r6.pushConst (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r8 := r7.swap1 (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup2 (by evm_kdecide) (by evm_ov)
  have r10 := r9.and (by evm_kdecide) (by evm_ov)
  have r11 := r10.push1 (UInt256.ofNat 63) (by evm_kdecide) (by evm_ov)
  have r12 := r11.add (by evm_kdecide) (by evm_ov)
  have r13 := r12.and (by evm_kdecide) (by evm_ov)
  have r14 := r13.dup2 (by evm_kdecide) (by evm_ov)
  have r15 := r14.add (by evm_kdecide) (by evm_ov)
  have r16 := r15.swap1 (by evm_kdecide) (by evm_ov)
  have r17 := r16.dup4 (by evm_kdecide) (by evm_ov)
  have r18 := r17.dup3 (by evm_kdecide) (by evm_ov)
  have r19 := r18.gt (by evm_kdecide) (by evm_ov)
  have r20 := r19.dup2 (by evm_kdecide) (by evm_ov)
  have r21 := r20.dup4 (by evm_kdecide) (by evm_ov)
  have r22 := r21.lt (by evm_kdecide) (by evm_ov)
  have r23 := r22.or (by evm_kdecide) (by evm_ov)
  have r24 := r23.iszero (by evm_kdecide) (by evm_ov)
  have r25 := r24.push2 (UInt256.ofNat 9948) (by evm_kdecide) (by evm_ov)
  have r26 := r25.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 9948)) r26 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_9878_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.lor (UInt256.lt ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) (memLoad (UInt256.ofNat 64) mem)) (UInt256.gt ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) x1))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9948) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9878) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9948) (l1cdm_block_9878_taken_stack (mem := mem) (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_9878_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_9878_fallthrough`. -/
def l1cdm_block_9878_fallthrough_stack {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((memLoad (UInt256.ofNat 64) mem) :: ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 9878. -/
theorem l1cdm_block_9878_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.lor (UInt256.lt ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) (memLoad (UInt256.ofNat 64) mem)) (UInt256.gt ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) x1))) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9878) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9941) (l1cdm_block_9878_fallthrough_stack (mem := mem) (x0 := x0) (x1 := x1) (R := R)) mem (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 26) (C + ((83) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r3 := RD.genMload r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 31) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup3 (by evm_kdecide) (by evm_ov)
  have r6 := r5.add (by evm_kdecide) (by evm_ov)
  have r7 := r6.pushConst (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r8 := r7.swap1 (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup2 (by evm_kdecide) (by evm_ov)
  have r10 := r9.and (by evm_kdecide) (by evm_ov)
  have r11 := r10.push1 (UInt256.ofNat 63) (by evm_kdecide) (by evm_ov)
  have r12 := r11.add (by evm_kdecide) (by evm_ov)
  have r13 := r12.and (by evm_kdecide) (by evm_ov)
  have r14 := r13.dup2 (by evm_kdecide) (by evm_ov)
  have r15 := r14.add (by evm_kdecide) (by evm_ov)
  have r16 := r15.swap1 (by evm_kdecide) (by evm_ov)
  have r17 := r16.dup4 (by evm_kdecide) (by evm_ov)
  have r18 := r17.dup3 (by evm_kdecide) (by evm_ov)
  have r19 := r18.gt (by evm_kdecide) (by evm_ov)
  have r20 := r19.dup2 (by evm_kdecide) (by evm_ov)
  have r21 := r20.dup4 (by evm_kdecide) (by evm_ov)
  have r22 := r21.lt (by evm_kdecide) (by evm_ov)
  have r23 := r22.or (by evm_kdecide) (by evm_ov)
  have r24 := r23.iszero (by evm_kdecide) (by evm_ov)
  have r25 := r24.push2 (UInt256.ofNat 9948) (by evm_kdecide) (by evm_ov)
  have r26 := r25.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 9941)) r26 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_9878_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.lor (UInt256.lt ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) (memLoad (UInt256.ofNat 64) mem)) (UInt256.gt ((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat 63) + (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904) (x0 + (UInt256.ofNat 31)))) (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904))) x1))) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9878) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9941) (l1cdm_block_9878_fallthrough_stack (mem := mem) (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_9878_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_9941`. -/
def l1cdm_block_9941_stack {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 9948) :: R)

/-- Automatically generated RD summary for bytecode block at pc 9941. -/
theorem l1cdm_block_9941 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9750) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9941) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9750) (l1cdm_block_9941_stack (R := R)) mem aw rdata σ (k + 3) (C + ((14))) := by
  let r0 := h
  have r1 := r0.push2 (UInt256.ofNat 9948) (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 9750) (by evm_kdecide) (by evm_ov)
  have r3 := r2.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 9750)) r3 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_9941_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9750) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9941) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9750) (l1cdm_block_9941_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_9941 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final memory for bytecode block summary `l1cdm_block_9948_taken`. -/
def l1cdm_block_9948_taken_memory {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} : ByteArray :=
  (x2.toByteArray.write 0 (x1.toByteArray.write 0 mem (UInt256.ofNat 64).toNat 32) x0.toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 9948. -/
theorem l1cdm_block_9948_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt ((x4 + x2) + (UInt256.ofNat 32)) x8)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9973) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9948) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9973) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) (l1cdm_block_9948_taken_memory (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2)) (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) x0 (⟨32⟩ : UInt256)) rdata σ (k + 17) (C + ((56) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) x0 (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup2 (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r4 := RD.genMstore r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup3 (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup2 (by evm_kdecide) (by evm_ov)
  have r7 := RD.genMstore r6 (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup9 (by evm_kdecide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup5 (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup8 (by evm_kdecide) (by evm_ov)
  have r12 := r11.add (by evm_kdecide) (by evm_ov)
  have r13 := r12.add (by evm_kdecide) (by evm_ov)
  have r14 := r13.gt (by evm_kdecide) (by evm_ov)
  have r15 := r14.iszero (by evm_kdecide) (by evm_ov)
  have r16 := r15.push2 (UInt256.ofNat 9973) (by evm_kdecide) (by evm_ov)
  have r17 := r16.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 9973)) r17 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_9948_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt ((x4 + x2) + (UInt256.ofNat 32)) x8)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9973) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9948) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9973) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) (l1cdm_block_9948_taken_memory (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_9948_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final memory for bytecode block summary `l1cdm_block_9948_fallthrough`. -/
def l1cdm_block_9948_fallthrough_memory {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} : ByteArray :=
  (x2.toByteArray.write 0 (x1.toByteArray.write 0 mem (UInt256.ofNat 64).toNat 32) x0.toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 9948. -/
theorem l1cdm_block_9948_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt ((x4 + x2) + (UInt256.ofNat 32)) x8)) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9948) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9969) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) (l1cdm_block_9948_fallthrough_memory (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2)) (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) x0 (⟨32⟩ : UInt256)) rdata σ (k + 17) (C + ((56) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) x0 (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup2 (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r4 := RD.genMstore r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup3 (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup2 (by evm_kdecide) (by evm_ov)
  have r7 := RD.genMstore r6 (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup9 (by evm_kdecide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup5 (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup8 (by evm_kdecide) (by evm_ov)
  have r12 := r11.add (by evm_kdecide) (by evm_ov)
  have r13 := r12.add (by evm_kdecide) (by evm_ov)
  have r14 := r13.gt (by evm_kdecide) (by evm_ov)
  have r15 := r14.iszero (by evm_kdecide) (by evm_ov)
  have r16 := r15.push2 (UInt256.ofNat 9973) (by evm_kdecide) (by evm_ov)
  have r17 := r16.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 9969)) r17 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_9948_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt ((x4 + x2) + (UInt256.ofNat 32)) x8)) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9948) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9969) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) (l1cdm_block_9948_fallthrough_memory (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_9948_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 9969. -/
theorem l1cdm_block_9969 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9969) R mem aw rdata σ k C)
    : RDrev L1cdmEvm.l1cdmRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `l1cdm_block_9973`. -/
def l1cdm_block_9973_stack {x0 : UInt256} {x5 : UInt256} {x7 : UInt256} {R : List UInt256} : List UInt256 :=
  ((x7 + (UInt256.ofNat 32)) :: (UInt256.ofNat 10012) :: x5 :: x0 :: x7 :: R)

/-- Final memory for bytecode block summary `l1cdm_block_9973`. -/
def l1cdm_block_9973_memory {ee : ExecutionEnv} {mem : ByteArray} {x0 : UInt256} {x2 : UInt256} {x4 : UInt256} : ByteArray :=
  ((UInt256.ofNat 0).toByteArray.write 0 (ee.calldata.write (x4 + (UInt256.ofNat 32)).toNat mem (x0 + (UInt256.ofNat 32)).toNat x2.toNat) ((x0 + x2) + (UInt256.ofNat 32)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 9973. -/
theorem l1cdm_block_9973 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9414) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9973) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9414) (l1cdm_block_9973_stack (x0 := x0) (x5 := x5) (x7 := x7) (R := R)) (l1cdm_block_9973_memory (ee := ee) (mem := mem) (x0 := x0) (x2 := x2) (x4 := x4)) (M (M aw (x0 + (UInt256.ofNat 32)) x2) ((x0 + x2) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) rdata σ (k + 30) (C + ((84) + (memExpansionCost aw (x0 + (UInt256.ofNat 32)) x2) + (3 + 3 * ((x2.toNat + 31) / 32)) + (memExpansionCost (M aw (x0 + (UInt256.ofNat 32)) x2) ((x0 + x2) + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup3 (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup7 (by evm_kdecide) (by evm_ov)
  have r5 := r4.add (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup4 (by evm_kdecide) (by evm_ov)
  have r8 := r7.add (by evm_kdecide) (by evm_ov)
  have r9 := RD.genCalldatacopy r8 (by evm_kdecide) (by evm_ov)
  have r10 := r9.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r11 := r10.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r12 := r11.dup5 (by evm_kdecide) (by evm_ov)
  have r13 := r12.dup4 (by evm_kdecide) (by evm_ov)
  have r14 := r13.add (by evm_kdecide) (by evm_ov)
  have r15 := r14.add (by evm_kdecide) (by evm_ov)
  have r16 := RD.genMstore r15 (by evm_kdecide) (by evm_ov)
  have r17 := r16.dup1 (by evm_kdecide) (by evm_ov)
  have r18 := r17.swap7 (by evm_kdecide) (by evm_ov)
  have r19 := r18.pop (by evm_kdecide) (by evm_ov)
  have r20 := r19.pop (by evm_kdecide) (by evm_ov)
  have r21 := r20.pop (by evm_kdecide) (by evm_ov)
  have r22 := r21.pop (by evm_kdecide) (by evm_ov)
  have r23 := r22.pop (by evm_kdecide) (by evm_ov)
  have r24 := r23.pop (by evm_kdecide) (by evm_ov)
  have r25 := r24.push2 (UInt256.ofNat 10012) (by evm_kdecide) (by evm_ov)
  have r26 := r25.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r27 := r26.dup5 (by evm_kdecide) (by evm_ov)
  have r28 := r27.add (by evm_kdecide) (by evm_ov)
  have r29 := r28.push2 (UInt256.ofNat 9414) (by evm_kdecide) (by evm_ov)
  have r30 := r29.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 9414)) r30 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_9973_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9414) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9973) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9414) (l1cdm_block_9973_stack (x0 := x0) (x5 := x5) (x7 := x7) (R := R)) (l1cdm_block_9973_memory (ee := ee) (mem := mem) (x0 := x0) (x2 := x2) (x4 := x4)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_9973 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10012`. -/
def l1cdm_block_10012_stack {x0 : UInt256} {x2 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: x2 :: R)

/-- Automatically generated RD summary for bytecode block at pc 10012. -/
theorem l1cdm_block_10012 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x5 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10012) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 x5 (l1cdm_block_10012_stack (x0 := x0) (x2 := x2) (R := R)) mem aw rdata σ (k + 9) (C + ((27))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pop (by evm_kdecide) (by evm_ov)
  have r4 := r3.swap3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.pop (by evm_kdecide) (by evm_ov)
  have r6 := r5.swap3 (by evm_kdecide) (by evm_ov)
  have r7 := r6.swap1 (by evm_kdecide) (by evm_ov)
  have r8 := r7.pop (by evm_kdecide) (by evm_ov)
  have r9 := r8.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r9 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10012_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x5 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10012) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 x5 (l1cdm_block_10012_stack (x0 := x0) (x2 := x2) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10012 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10021_taken`. -/
def l1cdm_block_10021_taken_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 10021. -/
theorem l1cdm_block_10021_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 192))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10048) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10021) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10048) (l1cdm_block_10021_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 16) (C + ((53))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup1 (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 192) (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup9 (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup11 (by evm_kdecide) (by evm_ov)
  have r12 := r11.sub (by evm_kdecide) (by evm_ov)
  have r13 := r12.slt (by evm_kdecide) (by evm_ov)
  have r14 := r13.iszero (by evm_kdecide) (by evm_ov)
  have r15 := r14.push2 (UInt256.ofNat 10048) (by evm_kdecide) (by evm_ov)
  have r16 := r15.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10048)) r16 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10021_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 192))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10048) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10021) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10048) (l1cdm_block_10021_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10021_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10021_fallthrough`. -/
def l1cdm_block_10021_fallthrough_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 10021. -/
theorem l1cdm_block_10021_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 192))) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10021) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10044) (l1cdm_block_10021_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 16) (C + ((53))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup1 (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 192) (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup9 (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup11 (by evm_kdecide) (by evm_ov)
  have r12 := r11.sub (by evm_kdecide) (by evm_ov)
  have r13 := r12.slt (by evm_kdecide) (by evm_ov)
  have r14 := r13.iszero (by evm_kdecide) (by evm_ov)
  have r15 := r14.push2 (UInt256.ofNat 10048) (by evm_kdecide) (by evm_ov)
  have r16 := r15.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10044)) r16 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10021_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 192))) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10021) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10044) (l1cdm_block_10021_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10021_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 10044. -/
theorem l1cdm_block_10044 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10044) R mem aw rdata σ k C)
    : RDrev L1cdmEvm.l1cdmRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `l1cdm_block_10048`. -/
def l1cdm_block_10048_stack {ee : ExecutionEnv} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {x7 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes (x7 + (UInt256.ofNat 32)).toNat 32)) :: (UInt256.ofNat 10066) :: (uInt256OfByteArray (ee.calldata.readBytes (x7 + (UInt256.ofNat 32)).toNat 32)) :: x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: (uInt256OfByteArray (ee.calldata.readBytes x7.toNat 32)) :: x7 :: R)

/-- Automatically generated RD summary for bytecode block at pc 10048. -/
theorem l1cdm_block_10048 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9304) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10048) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9304) (l1cdm_block_10048_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x7 := x7) (R := R)) mem aw rdata σ (k + 13) (C + ((41))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup8 (by evm_kdecide) (by evm_ov)
  have r3 := r2.calldataload (by evm_kdecide) (by evm_ov)
  have r4 := r3.swap7 (by evm_kdecide) (by evm_ov)
  have r5 := r4.pop (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup9 (by evm_kdecide) (by evm_ov)
  have r8 := r7.add (by evm_kdecide) (by evm_ov)
  have r9 := r8.calldataload (by evm_kdecide) (by evm_ov)
  have r10 := r9.push2 (UInt256.ofNat 10066) (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup2 (by evm_kdecide) (by evm_ov)
  have r12 := r11.push2 (UInt256.ofNat 9304) (by evm_kdecide) (by evm_ov)
  have r13 := r12.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 9304)) r13 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10048_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9304) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10048) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9304) (l1cdm_block_10048_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x7 := x7) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10048 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10066`. -/
def l1cdm_block_10066_stack {ee : ExecutionEnv} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {x7 : UInt256} {x8 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes (x8 + (UInt256.ofNat 64)).toNat 32)) :: (UInt256.ofNat 10082) :: (uInt256OfByteArray (ee.calldata.readBytes (x8 + (UInt256.ofNat 64)).toNat 32)) :: x1 :: x2 :: x3 :: x4 :: x5 :: x0 :: x7 :: x8 :: R)

/-- Automatically generated RD summary for bytecode block at pc 10066. -/
theorem l1cdm_block_10066 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9304) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10066) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9304) (l1cdm_block_10066_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x7 := x7) (x8 := x8) (R := R)) mem aw rdata σ (k + 11) (C + ((35))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap6 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pop (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup9 (by evm_kdecide) (by evm_ov)
  have r6 := r5.add (by evm_kdecide) (by evm_ov)
  have r7 := r6.calldataload (by evm_kdecide) (by evm_ov)
  have r8 := r7.push2 (UInt256.ofNat 10082) (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup2 (by evm_kdecide) (by evm_ov)
  have r10 := r9.push2 (UInt256.ofNat 9304) (by evm_kdecide) (by evm_ov)
  have r11 := r10.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 9304)) r11 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10066_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9304) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10066) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9304) (l1cdm_block_10066_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x7 := x7) (x8 := x8) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10066 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10082_taken`. -/
def l1cdm_block_10082_taken_stack {ee : ExecutionEnv} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x6 : UInt256} {x7 : UInt256} {x8 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes (x8 + (UInt256.ofNat 160)).toNat 32)) :: x1 :: x2 :: (uInt256OfByteArray (ee.calldata.readBytes (x8 + (UInt256.ofNat 128)).toNat 32)) :: (uInt256OfByteArray (ee.calldata.readBytes (x8 + (UInt256.ofNat 96)).toNat 32)) :: x0 :: x6 :: x7 :: x8 :: R)

/-- Automatically generated RD summary for bytecode block at pc 10082. -/
theorem l1cdm_block_10082_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 11 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes (x8 + (UInt256.ofNat 160)).toNat 32)) (UInt256.ofNat 18446744073709551615))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10124) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10082) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10124) (l1cdm_block_10082_taken_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x6 := x6) (x7 := x7) (x8 := x8) (R := R)) mem aw rdata σ (k + 25) (C + ((77))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap5 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pop (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 96) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup9 (by evm_kdecide) (by evm_ov)
  have r6 := r5.add (by evm_kdecide) (by evm_ov)
  have r7 := r6.calldataload (by evm_kdecide) (by evm_ov)
  have r8 := r7.swap4 (by evm_kdecide) (by evm_ov)
  have r9 := r8.pop (by evm_kdecide) (by evm_ov)
  have r10 := r9.push1 (UInt256.ofNat 128) (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup9 (by evm_kdecide) (by evm_ov)
  have r12 := r11.add (by evm_kdecide) (by evm_ov)
  have r13 := r12.calldataload (by evm_kdecide) (by evm_ov)
  have r14 := r13.swap3 (by evm_kdecide) (by evm_ov)
  have r15 := r14.pop (by evm_kdecide) (by evm_ov)
  have r16 := r15.push1 (UInt256.ofNat 160) (by evm_kdecide) (by evm_ov)
  have r17 := r16.dup9 (by evm_kdecide) (by evm_ov)
  have r18 := r17.add (by evm_kdecide) (by evm_ov)
  have r19 := r18.calldataload (by evm_kdecide) (by evm_ov)
  have r20 := r19.pushConst (UInt256.ofNat 18446744073709551615) (width := 8) (op := .PUSH8) (by decide) (by evm_kdecide) (by evm_ov)
  have r21 := r20.dup2 (by evm_kdecide) (by evm_ov)
  have r22 := r21.gt (by evm_kdecide) (by evm_ov)
  have r23 := r22.iszero (by evm_kdecide) (by evm_ov)
  have r24 := r23.push2 (UInt256.ofNat 10124) (by evm_kdecide) (by evm_ov)
  have r25 := r24.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10124)) r25 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10082_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 11 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes (x8 + (UInt256.ofNat 160)).toNat 32)) (UInt256.ofNat 18446744073709551615))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10124) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10082) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10124) (l1cdm_block_10082_taken_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x6 := x6) (x7 := x7) (x8 := x8) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10082_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10082_fallthrough`. -/
def l1cdm_block_10082_fallthrough_stack {ee : ExecutionEnv} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x6 : UInt256} {x7 : UInt256} {x8 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes (x8 + (UInt256.ofNat 160)).toNat 32)) :: x1 :: x2 :: (uInt256OfByteArray (ee.calldata.readBytes (x8 + (UInt256.ofNat 128)).toNat 32)) :: (uInt256OfByteArray (ee.calldata.readBytes (x8 + (UInt256.ofNat 96)).toNat 32)) :: x0 :: x6 :: x7 :: x8 :: R)

/-- Automatically generated RD summary for bytecode block at pc 10082. -/
theorem l1cdm_block_10082_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 11 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes (x8 + (UInt256.ofNat 160)).toNat 32)) (UInt256.ofNat 18446744073709551615))) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10082) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10120) (l1cdm_block_10082_fallthrough_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x6 := x6) (x7 := x7) (x8 := x8) (R := R)) mem aw rdata σ (k + 25) (C + ((77))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap5 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pop (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 96) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup9 (by evm_kdecide) (by evm_ov)
  have r6 := r5.add (by evm_kdecide) (by evm_ov)
  have r7 := r6.calldataload (by evm_kdecide) (by evm_ov)
  have r8 := r7.swap4 (by evm_kdecide) (by evm_ov)
  have r9 := r8.pop (by evm_kdecide) (by evm_ov)
  have r10 := r9.push1 (UInt256.ofNat 128) (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup9 (by evm_kdecide) (by evm_ov)
  have r12 := r11.add (by evm_kdecide) (by evm_ov)
  have r13 := r12.calldataload (by evm_kdecide) (by evm_ov)
  have r14 := r13.swap3 (by evm_kdecide) (by evm_ov)
  have r15 := r14.pop (by evm_kdecide) (by evm_ov)
  have r16 := r15.push1 (UInt256.ofNat 160) (by evm_kdecide) (by evm_ov)
  have r17 := r16.dup9 (by evm_kdecide) (by evm_ov)
  have r18 := r17.add (by evm_kdecide) (by evm_ov)
  have r19 := r18.calldataload (by evm_kdecide) (by evm_ov)
  have r20 := r19.pushConst (UInt256.ofNat 18446744073709551615) (width := 8) (op := .PUSH8) (by decide) (by evm_kdecide) (by evm_ov)
  have r21 := r20.dup2 (by evm_kdecide) (by evm_ov)
  have r22 := r21.gt (by evm_kdecide) (by evm_ov)
  have r23 := r22.iszero (by evm_kdecide) (by evm_ov)
  have r24 := r23.push2 (UInt256.ofNat 10124) (by evm_kdecide) (by evm_ov)
  have r25 := r24.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10120)) r25 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10082_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 11 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.gt (uInt256OfByteArray (ee.calldata.readBytes (x8 + (UInt256.ofNat 160)).toNat 32)) (UInt256.ofNat 18446744073709551615))) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10082) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10120) (l1cdm_block_10082_fallthrough_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x6 := x6) (x7 := x7) (x8 := x8) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10082_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 10120. -/
theorem l1cdm_block_10120 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10120) R mem aw rdata σ k C)
    : RDrev L1cdmEvm.l1cdmRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `l1cdm_block_10124`. -/
def l1cdm_block_10124_stack {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {x6 : UInt256} {x7 : UInt256} {x8 : UInt256} {x9 : UInt256} {R : List UInt256} : List UInt256 :=
  ((x8 + x0) :: x9 :: (UInt256.ofNat 10136) :: x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R)

/-- Automatically generated RD summary for bytecode block at pc 10124. -/
theorem l1cdm_block_10124 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 14 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9341) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10124) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9341) (l1cdm_block_10124_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (x7 := x7) (x8 := x8) (x9 := x9) (R := R)) mem aw rdata σ (k + 8) (C + ((27))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 10136) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup11 (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup12 (by evm_kdecide) (by evm_ov)
  have r6 := r5.add (by evm_kdecide) (by evm_ov)
  have r7 := r6.push2 (UInt256.ofNat 9341) (by evm_kdecide) (by evm_ov)
  have r8 := r7.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 9341)) r8 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10124_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 14 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9341) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10124) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9341) (l1cdm_block_10124_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (x7 := x7) (x8 := x8) (x9 := x9) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10124 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10136`. -/
def l1cdm_block_10136_stack {x0 : UInt256} {x1 : UInt256} {x5 : UInt256} {x6 : UInt256} {x7 : UInt256} {x8 : UInt256} {x9 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: x1 :: x5 :: x6 :: x7 :: x8 :: x9 :: R)

/-- Automatically generated RD summary for bytecode block at pc 10136. -/
theorem l1cdm_block_10136 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 x10 x11 x12 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x12 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10136) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: x10 :: x11 :: x12 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 x12 (l1cdm_block_10136_stack (x0 := x0) (x1 := x1) (x5 := x5) (x6 := x6) (x7 := x7) (x8 := x8) (x9 := x9) (R := R)) mem aw rdata σ (k + 19) (C + ((55))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap9 (by evm_kdecide) (by evm_ov)
  have r3 := r2.swap12 (by evm_kdecide) (by evm_ov)
  have r4 := r3.swap8 (by evm_kdecide) (by evm_ov)
  have r5 := r4.swap11 (by evm_kdecide) (by evm_ov)
  have r6 := r5.pop (by evm_kdecide) (by evm_ov)
  have r7 := r6.swap6 (by evm_kdecide) (by evm_ov)
  have r8 := r7.swap9 (by evm_kdecide) (by evm_ov)
  have r9 := r8.pop (by evm_kdecide) (by evm_ov)
  have r10 := r9.swap4 (by evm_kdecide) (by evm_ov)
  have r11 := r10.swap7 (by evm_kdecide) (by evm_ov)
  have r12 := r11.swap3 (by evm_kdecide) (by evm_ov)
  have r13 := r12.swap6 (by evm_kdecide) (by evm_ov)
  have r14 := r13.swap3 (by evm_kdecide) (by evm_ov)
  have r15 := r14.swap4 (by evm_kdecide) (by evm_ov)
  have r16 := r15.pop (by evm_kdecide) (by evm_ov)
  have r17 := r16.pop (by evm_kdecide) (by evm_ov)
  have r18 := r17.pop (by evm_kdecide) (by evm_ov)
  have r19 := r18.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r19 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10136_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 x10 x11 x12 : UInt256} {R : List UInt256}
    (hstack : R.length + 13 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x12 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10136) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: x10 :: x11 :: x12 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 x12 (l1cdm_block_10136_stack (x0 := x0) (x1 := x1) (x5 := x5) (x6 := x6) (x7 := x7) (x8 := x8) (x9 := x9) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10136 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

end l1cdmBlocks
