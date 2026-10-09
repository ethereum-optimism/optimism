import Reasoning.Reach
import L1cdmEvm.Bytecode

open Solm ABI Ethereum Ethereum.EVM
open Reasoning.Theory Reasoning.Reach

namespace l1cdmBlocks

/-- Final stack for bytecode block summary `l1cdm_block_6974_taken`. -/
def l1cdm_block_6974_taken_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 6974. -/
theorem l1cdm_block_6974_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 7012) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 6974) (x0 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7012) (l1cdm_block_6974_taken_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((20))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.iszero (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 7012) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 7012)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_6974_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 7012) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 6974) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7012) (l1cdm_block_6974_taken_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_6974_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_6974_fallthrough`. -/
def l1cdm_block_6974_fallthrough_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 6974. -/
theorem l1cdm_block_6974_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero x0) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 6974) (x0 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 6981) (l1cdm_block_6974_fallthrough_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((20))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.iszero (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 7012) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 6981)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_6974_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero x0) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 6974) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 6981) (l1cdm_block_6974_fallthrough_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_6974_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_6981`. -/
def l1cdm_block_6981_stack {ee : ExecutionEnv} {σ : AccountMap} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero (UInt256.eq (UInt256.ofNat 57005) (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 204) (⟨0⟩ : UInt256)))))) :: R)

/-- Automatically generated RD summary for bytecode block at pc 6981. -/
theorem l1cdm_block_6981 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 6981) (x0 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7012) (l1cdm_block_6981_stack (ee := ee) (σ := σ) (R := R)) mem aw rdata σ k' C' := by
  let r0 := h
  have r1 := r0.pop (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 204) (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r3⟩ := RD.sload r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r5 := r4.and (by evm_kdecide) (by evm_ov)
  have r6 := r5.push2 (UInt256.ofNat 57005) (by evm_kdecide) (by evm_ov)
  have r7 := r6.eq (by evm_kdecide) (by evm_ov)
  have r8 := r7.iszero (by evm_kdecide) (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 7012)) r8 (by evm_kdecide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_6981_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 6981) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7012) (l1cdm_block_6981_stack (ee := ee) (σ := σ) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k0, C0, h0⟩ := l1cdm_block_6981 hstack h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_7012_taken`. -/
def l1cdm_block_7012_taken_stack {R : List UInt256} : List UInt256 :=
  R

/-- Automatically generated RD summary for bytecode block at pc 7012. -/
theorem l1cdm_block_7012_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (UInt256.isZero x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 7293) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7012) (x0 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7293) (l1cdm_block_7012_taken_stack (R := R)) mem aw rdata σ (k + 4) (C + ((17))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.iszero (by evm_kdecide) (by evm_ov)
  have r3 := r2.push2 (UInt256.ofNat 7293) (by evm_kdecide) (by evm_ov)
  have r4 := r3.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 7293)) r4 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_7012_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (UInt256.isZero x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 7293) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7012) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7293) (l1cdm_block_7012_taken_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_7012_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_7012_fallthrough`. -/
def l1cdm_block_7012_fallthrough_stack {R : List UInt256} : List UInt256 :=
  R

/-- Automatically generated RD summary for bytecode block at pc 7012. -/
theorem l1cdm_block_7012_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (UInt256.isZero x0) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7012) (x0 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7018) (l1cdm_block_7012_fallthrough_stack (R := R)) mem aw rdata σ (k + 4) (C + ((17))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.iszero (by evm_kdecide) (by evm_ov)
  have r3 := r2.push2 (UInt256.ofNat 7293) (by evm_kdecide) (by evm_ov)
  have r4 := r3.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 7018)) r4 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_7012_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (UInt256.isZero x0) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7012) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7018) (l1cdm_block_7012_fallthrough_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_7012_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_7018`. -/
def l1cdm_block_7018_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639935) :: x0 :: R)

/-- Final memory for bytecode block summary `l1cdm_block_7018`. -/
def l1cdm_block_7018_memory {mem : ByteArray} {x0 : UInt256} : ByteArray :=
  ((UInt256.ofNat 206).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 7018. -/
theorem l1cdm_block_7018 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hperm : ee.perm = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7018) (x0 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7145) (l1cdm_block_7018_stack (x0 := x0) (R := R)) (l1cdm_block_7018_memory (mem := mem) (x0 := x0)) (M (M (M (M (M aw (UInt256.ofNat 0) (⟨32⟩ : UInt256)) (UInt256.ofNat 32) (⟨32⟩ : UInt256)) (UInt256.ofNat 0) (UInt256.ofNat 64)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat 206).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (UInt256.ofNat 0)) rdata (sstoreAccountMap ee.codeOwner σ (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 206).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (UInt256.lor (UInt256.ofNat 1) (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639680) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 206).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256)))))) k' C' := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup2 (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup2 (by evm_kdecide) (by evm_ov)
  have r4 := RD.genMstore r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.push1 (UInt256.ofNat 206) (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r7 := RD.genMstore r6 (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup1 (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup3 (by evm_kdecide) (by evm_ov)
  have r11 := RD.genKeccak256 r10 (by evm_kdecide) (by evm_ov)
  have r12 := r11.dup1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r13⟩ := RD.sload r12 (by evm_kdecide) (by evm_ov)
  have r14 := r13.pushConst (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639680) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r15 := r14.and (by evm_kdecide) (by evm_ov)
  have r16 := r15.push1 (UInt256.ofNat 1) (by evm_kdecide) (by evm_ov)
  have r17 := r16.or (by evm_kdecide) (by evm_ov)
  have r18 := r17.swap1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r19⟩ := RD.sstore r18 hperm (by evm_kdecide) (by evm_ov)
  have r20 := RD.genMload r19 (by evm_kdecide) (by evm_ov)
  have r21 := r20.dup3 (by evm_kdecide) (by evm_ov)
  have r22 := r21.swap2 (by evm_kdecide) (by evm_ov)
  have r23 := r22.pushConst (UInt256.ofNat 69572917962625114042862648717008669217229595705592602298547595161646054841231) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r24 := r23.swap2 (by evm_kdecide) (by evm_ov)
  have r25 := RD.genLog2 r24 (by evm_kdecide) hperm (by evm_ov)
  have r26 := r25.pushConst (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639935) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 7145)) r26 (by evm_kdecide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_7018_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hperm : ee.perm = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7018) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7145) (l1cdm_block_7018_stack (x0 := x0) (R := R)) (l1cdm_block_7018_memory (mem := mem) (x0 := x0)) aw' rdata (sstoreAccountMap ee.codeOwner σ (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 206).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (UInt256.lor (UInt256.ofNat 1) (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639680) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 206).toByteArray.write 0 (x0.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256)))))) k' C' := by
  obtain ⟨k0, C0, h0⟩ := l1cdm_block_7018 hstack hperm h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/- Unsupported instruction boundary at pc 7145: unsupported_32 (0x32). No RD transition is asserted. Summaries resume at pc 7146 from a fresh symbolic RD state. -/

/-- Final stack for bytecode block summary `l1cdm_block_7146_taken`. -/
def l1cdm_block_7146_taken_stack {R : List UInt256} : List UInt256 :=
  R

/-- Automatically generated RD summary for bytecode block at pc 7146. -/
theorem l1cdm_block_7146_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (x0 + x1) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 7286) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7146) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7286) (l1cdm_block_7146_taken_stack (R := R)) mem aw rdata σ (k + 3) (C + ((16))) := by
  let r0 := h
  have r1 := r0.add (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 7286) (by evm_kdecide) (by evm_ov)
  have r3 := r2.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 7286)) r3 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_7146_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (x0 + x1) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 7286) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7146) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7286) (l1cdm_block_7146_taken_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_7146_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_7146_fallthrough`. -/
def l1cdm_block_7146_fallthrough_stack {R : List UInt256} : List UInt256 :=
  R

/-- Automatically generated RD summary for bytecode block at pc 7146. -/
theorem l1cdm_block_7146_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (x0 + x1) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7146) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7151) (l1cdm_block_7146_fallthrough_stack (R := R)) mem aw rdata σ (k + 3) (C + ((16))) := by
  let r0 := h
  have r1 := r0.add (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 7286) (by evm_kdecide) (by evm_ov)
  have r3 := r2.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 7151)) r3 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_7146_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (x0 + x1) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7146) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7151) (l1cdm_block_7146_fallthrough_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_7146_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_7151`. -/
def l1cdm_block_7151_stack {mem : ByteArray} {R : List UInt256} : List UInt256 :=
  (((UInt256.ofNat 132) + (memLoad (UInt256.ofNat 64) mem)) :: R)

/-- Final memory for bytecode block summary `l1cdm_block_7151`. -/
def l1cdm_block_7151_memory {mem : ByteArray} : ByteArray :=
  ((UInt256.ofNat 51742864308511392307766902138383214409594566377941719010920889968520796831744).toByteArray.write 0 ((UInt256.ofNat 30507150626841010485563266520892382849468982703453806342338901273566197870368).toByteArray.write 0 ((UInt256.ofNat 45).toByteArray.write 0 ((UInt256.ofNat 32).toByteArray.write 0 ((UInt256.ofNat 3963877391197344453575983046348115674221700746820753546331534351508065746944).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)).toNat 32) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 7151. -/
theorem l1cdm_block_7151 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 4647) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7151) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4647) (l1cdm_block_7151_stack (mem := mem) (R := R)) (l1cdm_block_7151_memory (mem := mem)) (M (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)) (⟨32⟩ : UInt256)) rdata σ (k + 29) (C + ((92) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 36)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 68)) (⟨32⟩ : UInt256)) ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 100)) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r2 := RD.genMload r1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pushConst (UInt256.ofNat 3963877391197344453575983046348115674221700746820753546331534351508065746944) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup2 (by evm_kdecide) (by evm_ov)
  have r5 := RD.genMstore r4 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r7 := r6.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup3 (by evm_kdecide) (by evm_ov)
  have r9 := r8.add (by evm_kdecide) (by evm_ov)
  have r10 := RD.genMstore r9 (by evm_kdecide) (by evm_ov)
  have r11 := r10.push1 (UInt256.ofNat 45) (by evm_kdecide) (by evm_ov)
  have r12 := r11.push1 (UInt256.ofNat 36) (by evm_kdecide) (by evm_ov)
  have r13 := r12.dup3 (by evm_kdecide) (by evm_ov)
  have r14 := r13.add (by evm_kdecide) (by evm_ov)
  have r15 := RD.genMstore r14 (by evm_kdecide) (by evm_ov)
  have r16 := r15.pushConst (UInt256.ofNat 30507150626841010485563266520892382849468982703453806342338901273566197870368) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r17 := r16.push1 (UInt256.ofNat 68) (by evm_kdecide) (by evm_ov)
  have r18 := r17.dup3 (by evm_kdecide) (by evm_ov)
  have r19 := r18.add (by evm_kdecide) (by evm_ov)
  have r20 := RD.genMstore r19 (by evm_kdecide) (by evm_ov)
  have r21 := r20.pushConst (UInt256.ofNat 51742864308511392307766902138383214409594566377941719010920889968520796831744) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r22 := r21.push1 (UInt256.ofNat 100) (by evm_kdecide) (by evm_ov)
  have r23 := r22.dup3 (by evm_kdecide) (by evm_ov)
  have r24 := r23.add (by evm_kdecide) (by evm_ov)
  have r25 := RD.genMstore r24 (by evm_kdecide) (by evm_ov)
  have r26 := r25.push1 (UInt256.ofNat 132) (by evm_kdecide) (by evm_ov)
  have r27 := r26.add (by evm_kdecide) (by evm_ov)
  have r28 := r27.push2 (UInt256.ofNat 4647) (by evm_kdecide) (by evm_ov)
  have r29 := r28.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 4647)) r29 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_7151_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 4647) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7151) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 4647) (l1cdm_block_7151_stack (mem := mem) (R := R)) (l1cdm_block_7151_memory (mem := mem)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_7151 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_7286`. -/
def l1cdm_block_7286_stack {R : List UInt256} : List UInt256 :=
  R

/-- Automatically generated RD summary for bytecode block at pc 7286. -/
theorem l1cdm_block_7286 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 7893) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7286) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7893) (l1cdm_block_7286_stack (R := R)) mem aw rdata σ (k + 5) (C + ((16))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := r2.pop (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 7893) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 7893)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_7286_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 7893) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7286) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7893) (l1cdm_block_7286_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_7286 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_7293`. -/
def l1cdm_block_7293_stack {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {x6 : UInt256} {x7 : UInt256} {R : List UInt256} {gasWord0 : UInt256} : List UInt256 :=
  (gasWord0 :: (UInt256.ofNat 40000) :: (UInt256.ofNat 7377) :: x6 :: (UInt256.ofNat 7438) :: (UInt256.ofNat 0) :: x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: R)

/-- Automatically generated RD summary for bytecode block at pc 7293. -/
theorem l1cdm_block_7293 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 : UInt256} {R : List UInt256}
    (hstack : R.length + 15 ≤ 1024)
    (hperm : ee.perm = true)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10945) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7293) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: R) mem aw rdata σ k C)
    : ∃ (gasWord0 : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10945) (l1cdm_block_7293_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (x7 := x7) (R := R) (gasWord0 := gasWord0)) mem aw rdata (sstoreAccountMap ee.codeOwner σ (UInt256.ofNat 204) (UInt256.lor (UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)) (UInt256.land (UInt256.ofNat 115792089237316195423570985007226406215939081747436879206741300988257197096960) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 204) (⟨0⟩ : UInt256)))))) k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 204) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r4⟩ := RD.sload r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.pushConst (UInt256.ofNat 115792089237316195423570985007226406215939081747436879206741300988257197096960) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r6 := r5.and (by evm_kdecide) (by evm_ov)
  have r7 := r6.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup11 (by evm_kdecide) (by evm_ov)
  have r9 := r8.and (by evm_kdecide) (by evm_ov)
  have r10 := r9.or (by evm_kdecide) (by evm_ov)
  have r11 := r10.swap1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r12⟩ := RD.sstore r11 hperm (by evm_kdecide) (by evm_ov)
  have r13 := r12.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r14 := r13.push2 (UInt256.ofNat 7438) (by evm_kdecide) (by evm_ov)
  have r15 := r14.dup9 (by evm_kdecide) (by evm_ov)
  have r16 := r15.push2 (UInt256.ofNat 40000) (by evm_kdecide) (by evm_ov)
  have r17 := RD.genGas r16 (by evm_kdecide) (by evm_ov)
  have r18 := r17.push2 (UInt256.ofNat 7377) (by evm_kdecide) (by evm_ov)
  have r19 := r18.swap2 (by evm_kdecide) (by evm_ov)
  have r20 := r19.swap1 (by evm_kdecide) (by evm_ov)
  have r21 := r20.push2 (UInt256.ofNat 10945) (by evm_kdecide) (by evm_ov)
  have r22 := r21.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10945)) r22 (by evm_kdecide)
  exact ⟨_, _, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_7293_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 : UInt256} {R : List UInt256}
    (hstack : R.length + 15 ≤ 1024)
    (hperm : ee.perm = true)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10945) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7293) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: R) mem aw rdata σ k C)
    : ∃ (gasWord0 : UInt256) (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10945) (l1cdm_block_7293_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (x7 := x7) (R := R) (gasWord0 := gasWord0)) mem aw' rdata (sstoreAccountMap ee.codeOwner σ (UInt256.ofNat 204) (UInt256.lor (UInt256.land x7 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)) (UInt256.land (UInt256.ofNat 115792089237316195423570985007226406215939081747436879206741300988257197096960) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 204) (⟨0⟩ : UInt256)))))) k' C' := by
  obtain ⟨gasWord0, k0, C0, h0⟩ := l1cdm_block_7293 hstack hperm hvalid h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨gasWord0, _, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_7377`. -/
def l1cdm_block_7377_stack {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {x6 : UInt256} {x7 : UInt256} {x8 : UInt256} {x9 : UInt256} {R : List UInt256} : List UInt256 :=
  ((memLoad (UInt256.ofNat 64) mem) :: x9 :: x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R)

/-- Final memory for bytecode block summary `l1cdm_block_7377`. -/
def l1cdm_block_7377_memory {ee : ExecutionEnv} {mem : ByteArray} {x6 : UInt256} {x7 : UInt256} : ByteArray :=
  ((UInt256.ofNat 0).toByteArray.write 0 (ee.calldata.write x7.toNat (x6.toByteArray.write 0 (((memLoad (UInt256.ofNat 64) mem) + ((UInt256.ofNat 32) + (UInt256.mul (UInt256.div ((UInt256.ofNat 31) + x6) (UInt256.ofNat 32)) (UInt256.ofNat 32)))).toByteArray.write 0 mem (UInt256.ofNat 64).toNat 32) (memLoad (UInt256.ofNat 64) mem).toNat 32) ((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)).toNat x6.toNat) (((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) + x6).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 7377. -/
theorem l1cdm_block_7377 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 21 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 8997) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7377) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 8997) (l1cdm_block_7377_stack (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (x7 := x7) (x8 := x8) (x9 := x9) (R := R)) (l1cdm_block_7377_memory (ee := ee) (mem := mem) (x6 := x6) (x7 := x7)) (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) x6) (((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) + x6) (⟨32⟩ : UInt256)) rdata σ (k + 52) (C + ((156) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) x6) + (3 + 3 * ((x6.toNat + 31) / 32)) + (memExpansionCost (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) x6) (((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) + x6) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup10 (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup9 (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup9 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup1 (by evm_kdecide) (by evm_ov)
  have r7 := r6.push1 (UInt256.ofNat 31) (by evm_kdecide) (by evm_ov)
  have r8 := r7.add (by evm_kdecide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup1 (by evm_kdecide) (by evm_ov)
  have r11 := r10.swap2 (by evm_kdecide) (by evm_ov)
  have r12 := r11.div (by evm_kdecide) (by evm_ov)
  have r13 := r12.mul (by evm_kdecide) (by evm_ov)
  have r14 := r13.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r15 := r14.add (by evm_kdecide) (by evm_ov)
  have r16 := r15.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r17 := RD.genMload r16 (by evm_kdecide) (by evm_ov)
  have r18 := r17.swap1 (by evm_kdecide) (by evm_ov)
  have r19 := r18.dup2 (by evm_kdecide) (by evm_ov)
  have r20 := r19.add (by evm_kdecide) (by evm_ov)
  have r21 := r20.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r22 := RD.genMstore r21 (by evm_kdecide) (by evm_ov)
  have r23 := r22.dup1 (by evm_kdecide) (by evm_ov)
  have r24 := r23.swap4 (by evm_kdecide) (by evm_ov)
  have r25 := r24.swap3 (by evm_kdecide) (by evm_ov)
  have r26 := r25.swap2 (by evm_kdecide) (by evm_ov)
  have r27 := r26.swap1 (by evm_kdecide) (by evm_ov)
  have r28 := r27.dup2 (by evm_kdecide) (by evm_ov)
  have r29 := r28.dup2 (by evm_kdecide) (by evm_ov)
  have r30 := RD.genMstore r29 (by evm_kdecide) (by evm_ov)
  have r31 := r30.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r32 := r31.add (by evm_kdecide) (by evm_ov)
  have r33 := r32.dup4 (by evm_kdecide) (by evm_ov)
  have r34 := r33.dup4 (by evm_kdecide) (by evm_ov)
  have r35 := r34.dup1 (by evm_kdecide) (by evm_ov)
  have r36 := r35.dup3 (by evm_kdecide) (by evm_ov)
  have r37 := r36.dup5 (by evm_kdecide) (by evm_ov)
  have r38 := RD.genCalldatacopy r37 (by evm_kdecide) (by evm_ov)
  have r39 := r38.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r40 := r39.swap3 (by evm_kdecide) (by evm_ov)
  have r41 := r40.add (by evm_kdecide) (by evm_ov)
  have r42 := r41.swap2 (by evm_kdecide) (by evm_ov)
  have r43 := r42.swap1 (by evm_kdecide) (by evm_ov)
  have r44 := r43.swap2 (by evm_kdecide) (by evm_ov)
  have r45 := RD.genMstore r44 (by evm_kdecide) (by evm_ov)
  have r46 := r45.pop (by evm_kdecide) (by evm_ov)
  have r47 := r46.push2 (UInt256.ofNat 8997) (by evm_kdecide) (by evm_ov)
  have r48 := r47.swap3 (by evm_kdecide) (by evm_ov)
  have r49 := r48.pop (by evm_kdecide) (by evm_ov)
  have r50 := r49.pop (by evm_kdecide) (by evm_ov)
  have r51 := r50.pop (by evm_kdecide) (by evm_ov)
  have r52 := r51.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 8997)) r52 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_7377_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 21 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 8997) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7377) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 8997) (l1cdm_block_7377_stack (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (x7 := x7) (x8 := x8) (x9 := x9) (R := R)) (l1cdm_block_7377_memory (ee := ee) (mem := mem) (x6 := x6) (x7 := x7)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_7377 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_7438_taken`. -/
def l1cdm_block_7438_taken_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: R)

/-- Automatically generated RD summary for bytecode block at pc 7438. -/
theorem l1cdm_block_7438_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hperm : ee.perm = true)
    (hcond : (UInt256.isZero x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 7620) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7438) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7620) (l1cdm_block_7438_taken_stack (x0 := x0) (R := R)) mem aw rdata (sstoreAccountMap ee.codeOwner σ (UInt256.ofNat 204) (UInt256.lor (UInt256.ofNat 57005) (UInt256.land (UInt256.ofNat 115792089237316195423570985007226406215939081747436879206741300988257197096960) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 204) (⟨0⟩ : UInt256)))))) k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 204) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r4⟩ := RD.sload r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.pushConst (UInt256.ofNat 115792089237316195423570985007226406215939081747436879206741300988257197096960) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r6 := r5.and (by evm_kdecide) (by evm_ov)
  have r7 := r6.push2 (UInt256.ofNat 57005) (by evm_kdecide) (by evm_ov)
  have r8 := r7.or (by evm_kdecide) (by evm_ov)
  have r9 := r8.swap1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r10⟩ := RD.sstore r9 hperm (by evm_kdecide) (by evm_ov)
  have r11 := r10.swap1 (by evm_kdecide) (by evm_ov)
  have r12 := r11.pop (by evm_kdecide) (by evm_ov)
  have r13 := r12.dup1 (by evm_kdecide) (by evm_ov)
  have r14 := r13.iszero (by evm_kdecide) (by evm_ov)
  have r15 := r14.push2 (UInt256.ofNat 7620) (by evm_kdecide) (by evm_ov)
  have r16 := r15.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 7620)) r16 (by evm_kdecide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_7438_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hperm : ee.perm = true)
    (hcond : (UInt256.isZero x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 7620) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7438) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7620) (l1cdm_block_7438_taken_stack (x0 := x0) (R := R)) mem aw' rdata (sstoreAccountMap ee.codeOwner σ (UInt256.ofNat 204) (UInt256.lor (UInt256.ofNat 57005) (UInt256.land (UInt256.ofNat 115792089237316195423570985007226406215939081747436879206741300988257197096960) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 204) (⟨0⟩ : UInt256)))))) k' C' := by
  obtain ⟨k0, C0, h0⟩ := l1cdm_block_7438_taken hstack hperm hcond hvalid h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_7438_fallthrough`. -/
def l1cdm_block_7438_fallthrough_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: R)

/-- Automatically generated RD summary for bytecode block at pc 7438. -/
theorem l1cdm_block_7438_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hperm : ee.perm = true)
    (hcond : (UInt256.isZero x0) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7438) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7491) (l1cdm_block_7438_fallthrough_stack (x0 := x0) (R := R)) mem aw rdata (sstoreAccountMap ee.codeOwner σ (UInt256.ofNat 204) (UInt256.lor (UInt256.ofNat 57005) (UInt256.land (UInt256.ofNat 115792089237316195423570985007226406215939081747436879206741300988257197096960) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 204) (⟨0⟩ : UInt256)))))) k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 204) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r4⟩ := RD.sload r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.pushConst (UInt256.ofNat 115792089237316195423570985007226406215939081747436879206741300988257197096960) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r6 := r5.and (by evm_kdecide) (by evm_ov)
  have r7 := r6.push2 (UInt256.ofNat 57005) (by evm_kdecide) (by evm_ov)
  have r8 := r7.or (by evm_kdecide) (by evm_ov)
  have r9 := r8.swap1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r10⟩ := RD.sstore r9 hperm (by evm_kdecide) (by evm_ov)
  have r11 := r10.swap1 (by evm_kdecide) (by evm_ov)
  have r12 := r11.pop (by evm_kdecide) (by evm_ov)
  have r13 := r12.dup1 (by evm_kdecide) (by evm_ov)
  have r14 := r13.iszero (by evm_kdecide) (by evm_ov)
  have r15 := r14.push2 (UInt256.ofNat 7620) (by evm_kdecide) (by evm_ov)
  have r16 := r15.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 7491)) r16 (by evm_kdecide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_7438_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hperm : ee.perm = true)
    (hcond : (UInt256.isZero x0) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7438) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7491) (l1cdm_block_7438_fallthrough_stack (x0 := x0) (R := R)) mem aw' rdata (sstoreAccountMap ee.codeOwner σ (UInt256.ofNat 204) (UInt256.lor (UInt256.ofNat 57005) (UInt256.land (UInt256.ofNat 115792089237316195423570985007226406215939081747436879206741300988257197096960) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 204) (⟨0⟩ : UInt256)))))) k' C' := by
  obtain ⟨k0, C0, h0⟩ := l1cdm_block_7438_fallthrough hstack hperm hcond h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Final memory for bytecode block summary `l1cdm_block_7491_taken`. -/
def l1cdm_block_7491_taken_memory {mem : ByteArray} {x1 : UInt256} : ByteArray :=
  ((UInt256.ofNat 203).toByteArray.write 0 (x1.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 7491. -/
theorem l1cdm_block_7491_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.land (UInt256.ofNat 255) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 203).toByteArray.write 0 (x1.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256))))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 7521) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7491) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7521) (x0 :: x1 :: R) (l1cdm_block_7491_taken_memory (mem := mem) (x1 := x1)) (M (M (M aw (UInt256.ofNat 0) (⟨32⟩ : UInt256)) (UInt256.ofNat 32) (⟨32⟩ : UInt256)) (UInt256.ofNat 0) (UInt256.ofNat 64)) rdata σ k' C' := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup3 (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup2 (by evm_kdecide) (by evm_ov)
  have r4 := RD.genMstore r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.push1 (UInt256.ofNat 203) (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r7 := RD.genMstore r6 (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r9 := r8.swap1 (by evm_kdecide) (by evm_ov)
  have r10 := RD.genKeccak256 r9 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r11⟩ := RD.sload r10 (by evm_kdecide) (by evm_ov)
  have r12 := r11.push1 (UInt256.ofNat 255) (by evm_kdecide) (by evm_ov)
  have r13 := r12.and (by evm_kdecide) (by evm_ov)
  have r14 := r13.iszero (by evm_kdecide) (by evm_ov)
  have r15 := r14.push2 (UInt256.ofNat 7521) (by evm_kdecide) (by evm_ov)
  have r16 := r15.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 7521)) r16 (by evm_kdecide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_7491_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.land (UInt256.ofNat 255) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 203).toByteArray.write 0 (x1.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256))))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 7521) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7491) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7521) (x0 :: x1 :: R) (l1cdm_block_7491_taken_memory (mem := mem) (x1 := x1)) aw' rdata σ k' C' := by
  obtain ⟨k0, C0, h0⟩ := l1cdm_block_7491_taken hstack hcond hvalid h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Final memory for bytecode block summary `l1cdm_block_7491_fallthrough`. -/
def l1cdm_block_7491_fallthrough_memory {mem : ByteArray} {x1 : UInt256} : ByteArray :=
  ((UInt256.ofNat 203).toByteArray.write 0 (x1.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 7491. -/
theorem l1cdm_block_7491_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.land (UInt256.ofNat 255) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 203).toByteArray.write 0 (x1.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256))))) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7491) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7514) (x0 :: x1 :: R) (l1cdm_block_7491_fallthrough_memory (mem := mem) (x1 := x1)) (M (M (M aw (UInt256.ofNat 0) (⟨32⟩ : UInt256)) (UInt256.ofNat 32) (⟨32⟩ : UInt256)) (UInt256.ofNat 0) (UInt256.ofNat 64)) rdata σ k' C' := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup3 (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup2 (by evm_kdecide) (by evm_ov)
  have r4 := RD.genMstore r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.push1 (UInt256.ofNat 203) (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r7 := RD.genMstore r6 (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r9 := r8.swap1 (by evm_kdecide) (by evm_ov)
  have r10 := RD.genKeccak256 r9 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r11⟩ := RD.sload r10 (by evm_kdecide) (by evm_ov)
  have r12 := r11.push1 (UInt256.ofNat 255) (by evm_kdecide) (by evm_ov)
  have r13 := r12.and (by evm_kdecide) (by evm_ov)
  have r14 := r13.iszero (by evm_kdecide) (by evm_ov)
  have r15 := r14.push2 (UInt256.ofNat 7521) (by evm_kdecide) (by evm_ov)
  have r16 := r15.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 7514)) r16 (by evm_kdecide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_7491_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.land (UInt256.ofNat 255) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 203).toByteArray.write 0 (x1.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256))))) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7491) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7514) (x0 :: x1 :: R) (l1cdm_block_7491_fallthrough_memory (mem := mem) (x1 := x1)) aw' rdata σ k' C' := by
  obtain ⟨k0, C0, h0⟩ := l1cdm_block_7491_fallthrough hstack hcond h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_7514`. -/
def l1cdm_block_7514_stack {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 7521) :: R)

/-- Automatically generated RD summary for bytecode block at pc 7514. -/
theorem l1cdm_block_7514 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10898) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7514) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10898) (l1cdm_block_7514_stack (R := R)) mem aw rdata σ (k + 3) (C + ((14))) := by
  let r0 := h
  have r1 := r0.push2 (UInt256.ofNat 7521) (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 10898) (by evm_kdecide) (by evm_ov)
  have r3 := r2.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10898)) r3 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_7514_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10898) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7514) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10898) (l1cdm_block_7514_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_7514 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final memory for bytecode block summary `l1cdm_block_7521`. -/
def l1cdm_block_7521_memory {mem : ByteArray} {x1 : UInt256} : ByteArray :=
  ((UInt256.ofNat 203).toByteArray.write 0 (x1.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 7521. -/
theorem l1cdm_block_7521 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hperm : ee.perm = true)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 7889) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7521) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7889) (x0 :: x1 :: R) (l1cdm_block_7521_memory (mem := mem) (x1 := x1)) (M (M (M (M (M aw (UInt256.ofNat 0) (⟨32⟩ : UInt256)) (UInt256.ofNat 32) (⟨32⟩ : UInt256)) (UInt256.ofNat 0) (UInt256.ofNat 64)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat 203).toByteArray.write 0 (x1.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (UInt256.ofNat 0)) rdata (sstoreAccountMap ee.codeOwner σ (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 203).toByteArray.write 0 (x1.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (UInt256.lor (UInt256.ofNat 1) (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639680) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 203).toByteArray.write 0 (x1.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256)))))) k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup3 (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup2 (by evm_kdecide) (by evm_ov)
  have r5 := RD.genMstore r4 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 203) (by evm_kdecide) (by evm_ov)
  have r7 := r6.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r8 := RD.genMstore r7 (by evm_kdecide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup1 (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup3 (by evm_kdecide) (by evm_ov)
  have r12 := RD.genKeccak256 r11 (by evm_kdecide) (by evm_ov)
  have r13 := r12.dup1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r14⟩ := RD.sload r13 (by evm_kdecide) (by evm_ov)
  have r15 := r14.pushConst (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639680) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r16 := r15.and (by evm_kdecide) (by evm_ov)
  have r17 := r16.push1 (UInt256.ofNat 1) (by evm_kdecide) (by evm_ov)
  have r18 := r17.or (by evm_kdecide) (by evm_ov)
  have r19 := r18.swap1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r20⟩ := RD.sstore r19 hperm (by evm_kdecide) (by evm_ov)
  have r21 := RD.genMload r20 (by evm_kdecide) (by evm_ov)
  have r22 := r21.dup4 (by evm_kdecide) (by evm_ov)
  have r23 := r22.swap2 (by evm_kdecide) (by evm_ov)
  have r24 := r23.pushConst (UInt256.ofNat 31778285560320820616475340538131748700417015051794998020303206270876069532476) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r25 := r24.swap2 (by evm_kdecide) (by evm_ov)
  have r26 := RD.genLog2 r25 (by evm_kdecide) hperm (by evm_ov)
  have r27 := r26.push2 (UInt256.ofNat 7889) (by evm_kdecide) (by evm_ov)
  have r28 := r27.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 7889)) r28 (by evm_kdecide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_7521_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hperm : ee.perm = true)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 7889) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7521) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7889) (x0 :: x1 :: R) (l1cdm_block_7521_memory (mem := mem) (x1 := x1)) aw' rdata (sstoreAccountMap ee.codeOwner σ (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 203).toByteArray.write 0 (x1.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (UInt256.lor (UInt256.ofNat 1) (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639680) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 203).toByteArray.write 0 (x1.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256)))))) k' C' := by
  obtain ⟨k0, C0, h0⟩ := l1cdm_block_7521 hstack hperm hvalid h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_7620`. -/
def l1cdm_block_7620_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639935) :: x0 :: x1 :: R)

/-- Final memory for bytecode block summary `l1cdm_block_7620`. -/
def l1cdm_block_7620_memory {mem : ByteArray} {x1 : UInt256} : ByteArray :=
  ((UInt256.ofNat 206).toByteArray.write 0 (x1.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 7620. -/
theorem l1cdm_block_7620 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hperm : ee.perm = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7620) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7748) (l1cdm_block_7620_stack (x0 := x0) (x1 := x1) (R := R)) (l1cdm_block_7620_memory (mem := mem) (x1 := x1)) (M (M (M (M (M aw (UInt256.ofNat 0) (⟨32⟩ : UInt256)) (UInt256.ofNat 32) (⟨32⟩ : UInt256)) (UInt256.ofNat 0) (UInt256.ofNat 64)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat 206).toByteArray.write 0 (x1.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (UInt256.ofNat 0)) rdata (sstoreAccountMap ee.codeOwner σ (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 206).toByteArray.write 0 (x1.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (UInt256.lor (UInt256.ofNat 1) (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639680) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 206).toByteArray.write 0 (x1.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256)))))) k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup3 (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup2 (by evm_kdecide) (by evm_ov)
  have r5 := RD.genMstore r4 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 206) (by evm_kdecide) (by evm_ov)
  have r7 := r6.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r8 := RD.genMstore r7 (by evm_kdecide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup1 (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup3 (by evm_kdecide) (by evm_ov)
  have r12 := RD.genKeccak256 r11 (by evm_kdecide) (by evm_ov)
  have r13 := r12.dup1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r14⟩ := RD.sload r13 (by evm_kdecide) (by evm_ov)
  have r15 := r14.pushConst (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639680) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r16 := r15.and (by evm_kdecide) (by evm_ov)
  have r17 := r16.push1 (UInt256.ofNat 1) (by evm_kdecide) (by evm_ov)
  have r18 := r17.or (by evm_kdecide) (by evm_ov)
  have r19 := r18.swap1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r20⟩ := RD.sstore r19 hperm (by evm_kdecide) (by evm_ov)
  have r21 := RD.genMload r20 (by evm_kdecide) (by evm_ov)
  have r22 := r21.dup4 (by evm_kdecide) (by evm_ov)
  have r23 := r22.swap2 (by evm_kdecide) (by evm_ov)
  have r24 := r23.pushConst (UInt256.ofNat 69572917962625114042862648717008669217229595705592602298547595161646054841231) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r25 := r24.swap2 (by evm_kdecide) (by evm_ov)
  have r26 := RD.genLog2 r25 (by evm_kdecide) hperm (by evm_ov)
  have r27 := r26.pushConst (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639935) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 7748)) r27 (by evm_kdecide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_7620_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hperm : ee.perm = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7620) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7748) (l1cdm_block_7620_stack (x0 := x0) (x1 := x1) (R := R)) (l1cdm_block_7620_memory (mem := mem) (x1 := x1)) aw' rdata (sstoreAccountMap ee.codeOwner σ (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 206).toByteArray.write 0 (x1.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (UInt256.lor (UInt256.ofNat 1) (UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639680) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) ((UInt256.ofNat 206).toByteArray.write 0 (x1.toByteArray.write 0 mem (UInt256.ofNat 0).toNat 32) (UInt256.ofNat 32).toNat 32)) (⟨0⟩ : UInt256)))))) k' C' := by
  obtain ⟨k0, C0, h0⟩ := l1cdm_block_7620 hstack hperm h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/- Unsupported instruction boundary at pc 7748: unsupported_32 (0x32). No RD transition is asserted. Summaries resume at pc 7749 from a fresh symbolic RD state. -/

/-- Final stack for bytecode block summary `l1cdm_block_7749_taken`. -/
def l1cdm_block_7749_taken_stack {R : List UInt256} : List UInt256 :=
  R

/-- Automatically generated RD summary for bytecode block at pc 7749. -/
theorem l1cdm_block_7749_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (x0 + x1) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 7889) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7749) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7889) (l1cdm_block_7749_taken_stack (R := R)) mem aw rdata σ (k + 3) (C + ((16))) := by
  let r0 := h
  have r1 := r0.add (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 7889) (by evm_kdecide) (by evm_ov)
  have r3 := r2.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 7889)) r3 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_7749_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (x0 + x1) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 7889) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7749) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7889) (l1cdm_block_7749_taken_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_7749_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

end l1cdmBlocks
