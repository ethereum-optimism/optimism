import Reasoning.Reach
import L1cdmEvm.Bytecode

open Solm ABI Ethereum Ethereum.EVM
open Reasoning.Theory Reasoning.Reach

namespace l1cdmBlocks

/-- Automatically generated RD summary for bytecode block at pc 1417. -/
theorem l1cdm_block_1417 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1417) R mem aw rdata σ k C)
    : RDrev L1cdmEvm.l1cdmRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `l1cdm_block_1421`. -/
def l1cdm_block_1421_stack {ee : ExecutionEnv} {σ : AccountMap} {R : List UInt256} : List UInt256 :=
  ((UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 207) (⟨0⟩ : UInt256)))) :: (UInt256.ofNat 588) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1421. -/
theorem l1cdm_block_1421 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 588) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1421) (x0 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 588) (l1cdm_block_1421_stack (ee := ee) (σ := σ) (R := R)) mem aw rdata σ k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 207) (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r4⟩ := RD.sload r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.push2 (UInt256.ofNat 588) (by evm_kdecide) (by evm_ov)
  have r6 := r5.swap1 (by evm_kdecide) (by evm_ov)
  have r7 := r6.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r8 := r7.and (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup2 (by evm_kdecide) (by evm_ov)
  have r10 := r9.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 588)) r10 (by evm_kdecide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1421_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 588) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1421) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 588) (l1cdm_block_1421_stack (ee := ee) (σ := σ) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k0, C0, h0⟩ := l1cdm_block_1421 hstack hvalid h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_1454_taken`. -/
def l1cdm_block_1454_taken_stack {ee : ExecutionEnv} {R : List UInt256} : List UInt256 :=
  (ee.weiValue :: R)

/-- Automatically generated RD summary for bytecode block at pc 1454. -/
theorem l1cdm_block_1454_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 1466) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1454) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1466) (l1cdm_block_1454_taken_stack (ee := ee) (R := R)) mem aw rdata σ (k + 6) (C + ((22))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.callvalue (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.iszero (by evm_kdecide) (by evm_ov)
  have r5 := r4.push2 (UInt256.ofNat 1466) (by evm_kdecide) (by evm_ov)
  have r6 := r5.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1466)) r6 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1454_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 1466) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1454) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1466) (l1cdm_block_1454_taken_stack (ee := ee) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_1454_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_1454_fallthrough`. -/
def l1cdm_block_1454_fallthrough_stack {ee : ExecutionEnv} {R : List UInt256} : List UInt256 :=
  (ee.weiValue :: R)

/-- Automatically generated RD summary for bytecode block at pc 1454. -/
theorem l1cdm_block_1454_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1454) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1462) (l1cdm_block_1454_fallthrough_stack (ee := ee) (R := R)) mem aw rdata σ (k + 6) (C + ((22))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.callvalue (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.iszero (by evm_kdecide) (by evm_ov)
  have r5 := r4.push2 (UInt256.ofNat 1466) (by evm_kdecide) (by evm_ov)
  have r6 := r5.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1462)) r6 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1454_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1454) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1462) (l1cdm_block_1454_fallthrough_stack (ee := ee) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_1454_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1462. -/
theorem l1cdm_block_1462 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1462) R mem aw rdata σ k C)
    : RDrev L1cdmEvm.l1cdmRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `l1cdm_block_1466`. -/
def l1cdm_block_1466_stack {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 260) :: (UInt256.ofNat 498) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1466. -/
theorem l1cdm_block_1466 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 498) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1466) (x0 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 498) (l1cdm_block_1466_stack (R := R)) mem aw rdata σ (k + 6) (C + ((20))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := r2.push2 (UInt256.ofNat 498) (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 260) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup2 (by evm_kdecide) (by evm_ov)
  have r6 := r5.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 498)) r6 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1466_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 498) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1466) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 498) (l1cdm_block_1466_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_1466 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_1476_taken`. -/
def l1cdm_block_1476_taken_stack {ee : ExecutionEnv} {R : List UInt256} : List UInt256 :=
  (ee.weiValue :: R)

/-- Automatically generated RD summary for bytecode block at pc 1476. -/
theorem l1cdm_block_1476_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 1488) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1476) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1488) (l1cdm_block_1476_taken_stack (ee := ee) (R := R)) mem aw rdata σ (k + 6) (C + ((22))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.callvalue (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.iszero (by evm_kdecide) (by evm_ov)
  have r5 := r4.push2 (UInt256.ofNat 1488) (by evm_kdecide) (by evm_ov)
  have r6 := r5.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1488)) r6 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1476_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 1488) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1476) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1488) (l1cdm_block_1476_taken_stack (ee := ee) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_1476_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_1476_fallthrough`. -/
def l1cdm_block_1476_fallthrough_stack {ee : ExecutionEnv} {R : List UInt256} : List UInt256 :=
  (ee.weiValue :: R)

/-- Automatically generated RD summary for bytecode block at pc 1476. -/
theorem l1cdm_block_1476_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1476) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1484) (l1cdm_block_1476_fallthrough_stack (ee := ee) (R := R)) mem aw rdata σ (k + 6) (C + ((22))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.callvalue (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.iszero (by evm_kdecide) (by evm_ov)
  have r5 := r4.push2 (UInt256.ofNat 1488) (by evm_kdecide) (by evm_ov)
  have r6 := r5.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1484)) r6 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1476_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1476) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1484) (l1cdm_block_1476_fallthrough_stack (ee := ee) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_1476_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1484. -/
theorem l1cdm_block_1484 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1484) R mem aw rdata σ k C)
    : RDrev L1cdmEvm.l1cdmRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `l1cdm_block_1488`. -/
def l1cdm_block_1488_stack {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 40) :: (UInt256.ofNat 498) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1488. -/
theorem l1cdm_block_1488 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 498) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1488) (x0 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 498) (l1cdm_block_1488_stack (R := R)) mem aw rdata σ (k + 6) (C + ((20))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := r2.push2 (UInt256.ofNat 498) (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 40) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup2 (by evm_kdecide) (by evm_ov)
  have r6 := r5.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 498)) r6 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1488_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 498) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1488) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 498) (l1cdm_block_1488_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_1488 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_1497_taken`. -/
def l1cdm_block_1497_taken_stack {ee : ExecutionEnv} {R : List UInt256} : List UInt256 :=
  (ee.weiValue :: R)

/-- Automatically generated RD summary for bytecode block at pc 1497. -/
theorem l1cdm_block_1497_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 1509) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1497) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1509) (l1cdm_block_1497_taken_stack (ee := ee) (R := R)) mem aw rdata σ (k + 6) (C + ((22))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.callvalue (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.iszero (by evm_kdecide) (by evm_ov)
  have r5 := r4.push2 (UInt256.ofNat 1509) (by evm_kdecide) (by evm_ov)
  have r6 := r5.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1509)) r6 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1497_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 1509) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1497) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1509) (l1cdm_block_1497_taken_stack (ee := ee) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_1497_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_1497_fallthrough`. -/
def l1cdm_block_1497_fallthrough_stack {ee : ExecutionEnv} {R : List UInt256} : List UInt256 :=
  (ee.weiValue :: R)

/-- Automatically generated RD summary for bytecode block at pc 1497. -/
theorem l1cdm_block_1497_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1497) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1505) (l1cdm_block_1497_fallthrough_stack (ee := ee) (R := R)) mem aw rdata σ (k + 6) (C + ((22))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.callvalue (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.iszero (by evm_kdecide) (by evm_ov)
  have r5 := r4.push2 (UInt256.ofNat 1509) (by evm_kdecide) (by evm_ov)
  have r6 := r5.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1505)) r6 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1497_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1497) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1505) (l1cdm_block_1497_fallthrough_stack (ee := ee) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_1497_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1505. -/
theorem l1cdm_block_1505 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1505) R mem aw rdata σ k C)
    : RDrev L1cdmEvm.l1cdmRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `l1cdm_block_1509`. -/
def l1cdm_block_1509_stack {ee : ExecutionEnv} {σ : AccountMap} {R : List UInt256} : List UInt256 :=
  ((UInt256.lor (UInt256.ofNat 1766847064778384329583297500742918515827483896875618958121606201292619776) (UInt256.land (UInt256.ofNat 1766847064778384329583297500742918515827483896875618958121606201292619775) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 205) (⟨0⟩ : UInt256))))) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1509. -/
theorem l1cdm_block_1509 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 1584) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1509) (x0 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1584) (l1cdm_block_1509_stack (ee := ee) (σ := σ) (R := R)) mem aw rdata σ k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := r2.push2 (UInt256.ofNat 1584) (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 205) (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r5⟩ := RD.sload r4 (by evm_kdecide) (by evm_ov)
  have r6 := r5.pushConst (UInt256.ofNat 1766847064778384329583297500742918515827483896875618958121606201292619775) (width := 30) (op := .PUSH30) (by decide) (by evm_kdecide) (by evm_ov)
  have r7 := r6.and (by evm_kdecide) (by evm_ov)
  have r8 := r7.pushConst (UInt256.ofNat 1766847064778384329583297500742918515827483896875618958121606201292619776) (width := 31) (op := .PUSH31) (by decide) (by evm_kdecide) (by evm_ov)
  have r9 := r8.or (by evm_kdecide) (by evm_ov)
  have r10 := r9.swap1 (by evm_kdecide) (by evm_ov)
  have r11 := r10.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1584)) r11 (by evm_kdecide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1509_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 1584) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1509) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1584) (l1cdm_block_1509_stack (ee := ee) (σ := σ) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k0, C0, h0⟩ := l1cdm_block_1509 hstack hvalid h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_1584`. -/
def l1cdm_block_1584_stack {mem : ByteArray} {R : List UInt256} : List UInt256 :=
  (((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) :: R)

/-- Final memory for bytecode block summary `l1cdm_block_1584`. -/
def l1cdm_block_1584_memory {mem : ByteArray} {x0 : UInt256} : ByteArray :=
  (x0.toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1584. -/
theorem l1cdm_block_1584 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 519) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1584) (x0 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 519) (l1cdm_block_1584_stack (mem := mem) (R := R)) (l1cdm_block_1584_memory (mem := mem) (x0 := x0)) (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) rdata σ (k + 10) (C + ((33) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r3 := RD.genMload r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.swap1 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup2 (by evm_kdecide) (by evm_ov)
  have r6 := RD.genMstore r5 (by evm_kdecide) (by evm_ov)
  have r7 := r6.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r8 := r7.add (by evm_kdecide) (by evm_ov)
  have r9 := r8.push2 (UInt256.ofNat 519) (by evm_kdecide) (by evm_ov)
  have r10 := r9.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 519)) r10 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1584_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 519) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1584) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 519) (l1cdm_block_1584_stack (mem := mem) (R := R)) (l1cdm_block_1584_memory (mem := mem) (x0 := x0)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_1584 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_1598`. -/
def l1cdm_block_1598_stack {ee : ExecutionEnv} {mem : ByteArray} {σ : AccountMap} {R : List UInt256} {gasWord0 : UInt256} : List UInt256 :=
  (gasWord0 :: (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 254) (⟨0⟩ : UInt256)))) :: (memLoad (UInt256.ofNat 64) ((UInt256.ofNat 24382563337235625073851903103206598865187692869022114960706191320626625511424).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)) :: ((UInt256.sub (memLoad (UInt256.ofNat 64) mem) (memLoad (UInt256.ofNat 64) ((UInt256.ofNat 24382563337235625073851903103206598865187692869022114960706191320626625511424).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32))) + (UInt256.ofNat 4)) :: (memLoad (UInt256.ofNat 64) ((UInt256.ofNat 24382563337235625073851903103206598865187692869022114960706191320626625511424).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)) :: (UInt256.ofNat 32) :: ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat 4)) :: (UInt256.ofNat 904399539) :: (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 254) (⟨0⟩ : UInt256)))) :: (UInt256.ofNat 0) :: R)

/-- Final memory for bytecode block summary `l1cdm_block_1598`. -/
def l1cdm_block_1598_memory {mem : ByteArray} : ByteArray :=
  ((UInt256.ofNat 24382563337235625073851903103206598865187692869022114960706191320626625511424).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1598. -/
theorem l1cdm_block_1598 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1598) R mem aw rdata σ k C)
    : ∃ (gasWord0 : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1693) (l1cdm_block_1598_stack (ee := ee) (mem := mem) (σ := σ) (R := R) (gasWord0 := gasWord0)) (l1cdm_block_1598_memory (mem := mem)) (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 254) (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r3⟩ := RD.sload r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := RD.genMload r5 (by evm_kdecide) (by evm_ov)
  have r7 := r6.pushConst (UInt256.ofNat 24382563337235625073851903103206598865187692869022114960706191320626625511424) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup2 (by evm_kdecide) (by evm_ov)
  have r9 := RD.genMstore r8 (by evm_kdecide) (by evm_ov)
  have r10 := r9.swap1 (by evm_kdecide) (by evm_ov)
  have r11 := RD.genMload r10 (by evm_kdecide) (by evm_ov)
  have r12 := r11.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r13 := r12.swap3 (by evm_kdecide) (by evm_ov)
  have r14 := r13.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r15 := r14.and (by evm_kdecide) (by evm_ov)
  have r16 := r15.swap2 (by evm_kdecide) (by evm_ov)
  have r17 := r16.push4 (UInt256.ofNat 904399539) (by evm_kdecide) (by evm_ov)
  have r18 := r17.swap2 (by evm_kdecide) (by evm_ov)
  have r19 := r18.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r20 := r19.dup1 (by evm_kdecide) (by evm_ov)
  have r21 := r20.dup4 (by evm_kdecide) (by evm_ov)
  have r22 := r21.add (by evm_kdecide) (by evm_ov)
  have r23 := r22.swap3 (by evm_kdecide) (by evm_ov)
  have r24 := r23.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r25 := r24.swap3 (by evm_kdecide) (by evm_ov)
  have r26 := r25.swap2 (by evm_kdecide) (by evm_ov)
  have r27 := r26.swap1 (by evm_kdecide) (by evm_ov)
  have r28 := r27.dup3 (by evm_kdecide) (by evm_ov)
  have r29 := r28.swap1 (by evm_kdecide) (by evm_ov)
  have r30 := r29.sub (by evm_kdecide) (by evm_ov)
  have r31 := r30.add (by evm_kdecide) (by evm_ov)
  have r32 := r31.dup2 (by evm_kdecide) (by evm_ov)
  have r33 := r32.dup7 (by evm_kdecide) (by evm_ov)
  have r34 := RD.genGas r33 (by evm_kdecide) (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1693)) r34 (by evm_kdecide)
  exact ⟨_, _, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1598_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1598) R mem aw rdata σ k C)
    : ∃ (gasWord0 : UInt256) (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1693) (l1cdm_block_1598_stack (ee := ee) (mem := mem) (σ := σ) (R := R) (gasWord0 := gasWord0)) (l1cdm_block_1598_memory (mem := mem)) aw' rdata σ k' C' := by
  obtain ⟨gasWord0, k0, C0, h0⟩ := l1cdm_block_1598 hstack h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨gasWord0, _, k', C', h'⟩

/- Unsupported instruction boundary at pc 1693: staticcall (0xfa). No RD transition is asserted. Summaries resume at pc 1694 from a fresh symbolic RD state. -/

/-- Final stack for bytecode block summary `l1cdm_block_1694_taken`. -/
def l1cdm_block_1694_taken_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1694. -/
theorem l1cdm_block_1694_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 1710) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1694) (x0 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1710) (l1cdm_block_1694_taken_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.iszero (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.iszero (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 1710) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1710)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1694_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 1710) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1694) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1710) (l1cdm_block_1694_taken_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_1694_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_1694_fallthrough`. -/
def l1cdm_block_1694_fallthrough_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1694. -/
theorem l1cdm_block_1694_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1694) (x0 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1701) (l1cdm_block_1694_fallthrough_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.iszero (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.iszero (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 1710) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1701)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1694_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1694) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1701) (l1cdm_block_1694_fallthrough_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_1694_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1701. -/
theorem l1cdm_block_1701 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1701) R mem aw rdata σ k C)
    : RDrev L1cdmEvm.l1cdmRuntime g s0 := by
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

/-- Final stack for bytecode block summary `l1cdm_block_1710`. -/
def l1cdm_block_1710_stack {mem : ByteArray} {rdata : ByteArray} {R : List UInt256} : List UInt256 :=
  ((memLoad (UInt256.ofNat 64) mem) :: ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat rdata.size)) :: (UInt256.ofNat 1746) :: R)

/-- Final memory for bytecode block summary `l1cdm_block_1710`. -/
def l1cdm_block_1710_memory {mem : ByteArray} {rdata : ByteArray} : ByteArray :=
  (((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat rdata.size) + (UInt256.ofNat 31)) (UInt256.lnot (UInt256.ofNat 31)))).toByteArray.write 0 mem (UInt256.ofNat 64).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1710. -/
theorem l1cdm_block_1710 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10218) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1710) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10218) (l1cdm_block_1710_stack (mem := mem) (rdata := rdata) (R := R)) (l1cdm_block_1710_memory (mem := mem) (rdata := rdata)) (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 28) (C + ((81) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
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
  have r24 := r23.push2 (UInt256.ofNat 1746) (by evm_kdecide) (by evm_ov)
  have r25 := r24.swap2 (by evm_kdecide) (by evm_ov)
  have r26 := r25.swap1 (by evm_kdecide) (by evm_ov)
  have r27 := r26.push2 (UInt256.ofNat 10218) (by evm_kdecide) (by evm_ov)
  have r28 := r27.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10218)) r28 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1710_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10218) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1710) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10218) (l1cdm_block_1710_stack (mem := mem) (rdata := rdata) (R := R)) (l1cdm_block_1710_memory (mem := mem) (rdata := rdata)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_1710 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

end l1cdmBlocks
