import Reasoning.Reach
import L1cdmEvm.Bytecode

open Solm ABI Ethereum Ethereum.EVM
open Reasoning.Theory Reasoning.Reach

namespace l1cdmBlocks

/-- Final stack for bytecode block summary `l1cdm_block_10886`. -/
def l1cdm_block_10886_stack {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 10893) :: R)

/-- Automatically generated RD summary for bytecode block at pc 10886. -/
theorem l1cdm_block_10886 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10596) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10886) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10596) (l1cdm_block_10886_stack (R := R)) mem aw rdata σ (k + 3) (C + ((14))) := by
  let r0 := h
  have r1 := r0.push2 (UInt256.ofNat 10893) (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 10596) (by evm_kdecide) (by evm_ov)
  have r3 := r2.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10596)) r3 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10886_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10596) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10886) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10596) (l1cdm_block_10886_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10886 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10893`. -/
def l1cdm_block_10893_stack {x1 : UInt256} {x2 : UInt256} {R : List UInt256} : List UInt256 :=
  ((x1 + x2) :: R)

/-- Automatically generated RD summary for bytecode block at pc 10893. -/
theorem l1cdm_block_10893 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x3 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10893) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 x3 (l1cdm_block_10893_stack (x1 := x1) (x2 := x2) (R := R)) mem aw rdata σ (k + 5) (C + ((17))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := r2.add (by evm_kdecide) (by evm_ov)
  have r4 := r3.swap1 (by evm_kdecide) (by evm_ov)
  have r5 := r4.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r5 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10893_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x3 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10893) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 x3 (l1cdm_block_10893_stack (x1 := x1) (x2 := x2) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10893 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 10898. -/
theorem l1cdm_block_10898 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10898) R mem aw rdata σ k C)
    : RDrev L1cdmEvm.l1cdmRuntime g s0 := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pushConst (UInt256.ofNat 35408467139433450592217433187231851964531694900788300625387963629091585785856) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r4 := RD.genMstore r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.push1 (UInt256.ofNat 1) (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r7 := RD.genMstore r6 (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 36) (by evm_kdecide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  exact RD.genRev r9 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `l1cdm_block_10945_taken`. -/
def l1cdm_block_10945_taken_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 10945. -/
theorem l1cdm_block_10945_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.lt x0 x1)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10963) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10945) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10963) (l1cdm_block_10945_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 8) (C + ((29))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup3 (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.lt (by evm_kdecide) (by evm_ov)
  have r6 := r5.iszero (by evm_kdecide) (by evm_ov)
  have r7 := r6.push2 (UInt256.ofNat 10963) (by evm_kdecide) (by evm_ov)
  have r8 := r7.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10963)) r8 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10945_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.lt x0 x1)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10963) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10945) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10963) (l1cdm_block_10945_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10945_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10945_fallthrough`. -/
def l1cdm_block_10945_fallthrough_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 10945. -/
theorem l1cdm_block_10945_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.lt x0 x1)) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10945) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10956) (l1cdm_block_10945_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 8) (C + ((29))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup3 (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.lt (by evm_kdecide) (by evm_ov)
  have r6 := r5.iszero (by evm_kdecide) (by evm_ov)
  have r7 := r6.push2 (UInt256.ofNat 10963) (by evm_kdecide) (by evm_ov)
  have r8 := r7.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10956)) r8 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10945_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.lt x0 x1)) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10945) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10956) (l1cdm_block_10945_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10945_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10956`. -/
def l1cdm_block_10956_stack {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 10963) :: R)

/-- Automatically generated RD summary for bytecode block at pc 10956. -/
theorem l1cdm_block_10956 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10596) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10956) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10596) (l1cdm_block_10956_stack (R := R)) mem aw rdata σ (k + 3) (C + ((14))) := by
  let r0 := h
  have r1 := r0.push2 (UInt256.ofNat 10963) (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 10596) (by evm_kdecide) (by evm_ov)
  have r3 := r2.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10596)) r3 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10956_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10596) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10956) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10596) (l1cdm_block_10956_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10956 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10963`. -/
def l1cdm_block_10963_stack {x1 : UInt256} {x2 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.sub x1 x2) :: R)

/-- Automatically generated RD summary for bytecode block at pc 10963. -/
theorem l1cdm_block_10963 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x3 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10963) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 x3 (l1cdm_block_10963_stack (x1 := x1) (x2 := x2) (R := R)) mem aw rdata σ (k + 5) (C + ((17))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := r2.sub (by evm_kdecide) (by evm_ov)
  have r4 := r3.swap1 (by evm_kdecide) (by evm_ov)
  have r5 := r4.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r5 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10963_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x3 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10963) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 x3 (l1cdm_block_10963_stack (x1 := x1) (x2 := x2) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10963 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_10968`. -/
def l1cdm_block_10968_stack {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {R : List UInt256} : List UInt256 :=
  (x1 :: (x0 + (UInt256.ofNat 160)) :: (UInt256.ofNat 11045) :: (UInt256.ofNat 0) :: x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R)

/-- Final memory for bytecode block summary `l1cdm_block_10968`. -/
def l1cdm_block_10968_memory {mem : ByteArray} {x0 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} : ByteArray :=
  ((UInt256.ofNat 160).toByteArray.write 0 ((UInt256.isZero (UInt256.isZero x2)).toByteArray.write 0 ((UInt256.land x3 (UInt256.ofNat 18446744073709551615)).toByteArray.write 0 (x4.toByteArray.write 0 ((UInt256.land x5 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 mem x0.toNat 32) (x0 + (UInt256.ofNat 32)).toNat 32) (x0 + (UInt256.ofNat 64)).toNat 32) (x0 + (UInt256.ofNat 96)).toNat 32) (x0 + (UInt256.ofNat 128)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 10968. -/
theorem l1cdm_block_10968 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 11 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9662) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10968) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9662) (l1cdm_block_10968_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (R := R)) (l1cdm_block_10968_memory (mem := mem) (x0 := x0) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5)) (M (M (M (M (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 96)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 128)) (⟨32⟩ : UInt256)) rdata σ (k + 38) (C + ((117) + (memExpansionCost aw x0 (⟨32⟩ : UInt256)) + (memExpansionCost (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 96)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 96)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 128)) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup7 (by evm_kdecide) (by evm_ov)
  have r4 := r3.and (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup2 (by evm_kdecide) (by evm_ov)
  have r6 := RD.genMstore r5 (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup5 (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup3 (by evm_kdecide) (by evm_ov)
  have r10 := r9.add (by evm_kdecide) (by evm_ov)
  have r11 := RD.genMstore r10 (by evm_kdecide) (by evm_ov)
  have r12 := r11.pushConst (UInt256.ofNat 18446744073709551615) (width := 8) (op := .PUSH8) (by decide) (by evm_kdecide) (by evm_ov)
  have r13 := r12.dup5 (by evm_kdecide) (by evm_ov)
  have r14 := r13.and (by evm_kdecide) (by evm_ov)
  have r15 := r14.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r16 := r15.dup3 (by evm_kdecide) (by evm_ov)
  have r17 := r16.add (by evm_kdecide) (by evm_ov)
  have r18 := RD.genMstore r17 (by evm_kdecide) (by evm_ov)
  have r19 := r18.dup3 (by evm_kdecide) (by evm_ov)
  have r20 := r19.iszero (by evm_kdecide) (by evm_ov)
  have r21 := r20.iszero (by evm_kdecide) (by evm_ov)
  have r22 := r21.push1 (UInt256.ofNat 96) (by evm_kdecide) (by evm_ov)
  have r23 := r22.dup3 (by evm_kdecide) (by evm_ov)
  have r24 := r23.add (by evm_kdecide) (by evm_ov)
  have r25 := RD.genMstore r24 (by evm_kdecide) (by evm_ov)
  have r26 := r25.push1 (UInt256.ofNat 160) (by evm_kdecide) (by evm_ov)
  have r27 := r26.push1 (UInt256.ofNat 128) (by evm_kdecide) (by evm_ov)
  have r28 := r27.dup3 (by evm_kdecide) (by evm_ov)
  have r29 := r28.add (by evm_kdecide) (by evm_ov)
  have r30 := RD.genMstore r29 (by evm_kdecide) (by evm_ov)
  have r31 := r30.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r32 := r31.push2 (UInt256.ofNat 11045) (by evm_kdecide) (by evm_ov)
  have r33 := r32.push1 (UInt256.ofNat 160) (by evm_kdecide) (by evm_ov)
  have r34 := r33.dup4 (by evm_kdecide) (by evm_ov)
  have r35 := r34.add (by evm_kdecide) (by evm_ov)
  have r36 := r35.dup5 (by evm_kdecide) (by evm_ov)
  have r37 := r36.push2 (UInt256.ofNat 9662) (by evm_kdecide) (by evm_ov)
  have r38 := r37.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 9662)) r38 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_10968_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 11 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9662) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10968) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9662) (l1cdm_block_10968_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (R := R)) (l1cdm_block_10968_memory (mem := mem) (x0 := x0) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_10968 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_11045`. -/
def l1cdm_block_11045_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: R)

/-- Automatically generated RD summary for bytecode block at pc 11045. -/
theorem l1cdm_block_11045 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x8 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 11045) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 x8 (l1cdm_block_11045_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 11) (C + ((29))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap8 (by evm_kdecide) (by evm_ov)
  have r3 := r2.swap7 (by evm_kdecide) (by evm_ov)
  have r4 := r3.pop (by evm_kdecide) (by evm_ov)
  have r5 := r4.pop (by evm_kdecide) (by evm_ov)
  have r6 := r5.pop (by evm_kdecide) (by evm_ov)
  have r7 := r6.pop (by evm_kdecide) (by evm_ov)
  have r8 := r7.pop (by evm_kdecide) (by evm_ov)
  have r9 := r8.pop (by evm_kdecide) (by evm_ov)
  have r10 := r9.pop (by evm_kdecide) (by evm_ov)
  have r11 := r10.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r11 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_11045_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x8 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 11045) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 x8 (l1cdm_block_11045_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_11045 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_11056`. -/
def l1cdm_block_11056_stack {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {R : List UInt256} : List UInt256 :=
  (x2 :: (x0 + (UInt256.ofNat 128)) :: (UInt256.ofNat 11113) :: (UInt256.ofNat 0) :: x0 :: x1 :: x2 :: x3 :: x4 :: R)

/-- Final memory for bytecode block summary `l1cdm_block_11056`. -/
def l1cdm_block_11056_memory {mem : ByteArray} {x0 : UInt256} {x3 : UInt256} {x4 : UInt256} : ByteArray :=
  ((UInt256.ofNat 128).toByteArray.write 0 ((UInt256.land x3 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.land x4 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 mem x0.toNat 32) (x0 + (UInt256.ofNat 32)).toNat 32) (x0 + (UInt256.ofNat 64)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 11056. -/
theorem l1cdm_block_11056 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9662) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 11056) (x0 :: x1 :: x2 :: x3 :: x4 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9662) (l1cdm_block_11056_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (R := R)) (l1cdm_block_11056_memory (mem := mem) (x0 := x0) (x3 := x3) (x4 := x4)) (M (M (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) rdata σ (k + 28) (C + ((86) + (memExpansionCost aw x0 (⟨32⟩ : UInt256)) + (memExpansionCost (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup1 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup8 (by evm_kdecide) (by evm_ov)
  have r6 := r5.and (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup4 (by evm_kdecide) (by evm_ov)
  have r8 := RD.genMstore r7 (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup1 (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup7 (by evm_kdecide) (by evm_ov)
  have r11 := r10.and (by evm_kdecide) (by evm_ov)
  have r12 := r11.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r13 := r12.dup5 (by evm_kdecide) (by evm_ov)
  have r14 := r13.add (by evm_kdecide) (by evm_ov)
  have r15 := RD.genMstore r14 (by evm_kdecide) (by evm_ov)
  have r16 := r15.pop (by evm_kdecide) (by evm_ov)
  have r17 := r16.push1 (UInt256.ofNat 128) (by evm_kdecide) (by evm_ov)
  have r18 := r17.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r19 := r18.dup4 (by evm_kdecide) (by evm_ov)
  have r20 := r19.add (by evm_kdecide) (by evm_ov)
  have r21 := RD.genMstore r20 (by evm_kdecide) (by evm_ov)
  have r22 := r21.push2 (UInt256.ofNat 11113) (by evm_kdecide) (by evm_ov)
  have r23 := r22.push1 (UInt256.ofNat 128) (by evm_kdecide) (by evm_ov)
  have r24 := r23.dup4 (by evm_kdecide) (by evm_ov)
  have r25 := r24.add (by evm_kdecide) (by evm_ov)
  have r26 := r25.dup6 (by evm_kdecide) (by evm_ov)
  have r27 := r26.push2 (UInt256.ofNat 9662) (by evm_kdecide) (by evm_ov)
  have r28 := r27.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 9662)) r28 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_11056_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9662) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 11056) (x0 :: x1 :: x2 :: x3 :: x4 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9662) (l1cdm_block_11056_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (R := R)) (l1cdm_block_11056_memory (mem := mem) (x0 := x0) (x3 := x3) (x4 := x4)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_11056 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_11113`. -/
def l1cdm_block_11113_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: R)

/-- Final memory for bytecode block summary `l1cdm_block_11113`. -/
def l1cdm_block_11113_memory {mem : ByteArray} {x2 : UInt256} {x3 : UInt256} : ByteArray :=
  (x3.toByteArray.write 0 mem (x2 + (UInt256.ofNat 96)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 11113. -/
theorem l1cdm_block_11113 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x7 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 11113) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 x7 (l1cdm_block_11113_stack (x0 := x0) (R := R)) (l1cdm_block_11113_memory (mem := mem) (x2 := x2) (x3 := x3)) (M aw (x2 + (UInt256.ofNat 96)) (⟨32⟩ : UInt256)) rdata σ (k + 16) (C + ((45) + (memExpansionCost aw (x2 + (UInt256.ofNat 96)) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pop (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.push1 (UInt256.ofNat 96) (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup4 (by evm_kdecide) (by evm_ov)
  have r7 := r6.add (by evm_kdecide) (by evm_ov)
  have r8 := RD.genMstore r7 (by evm_kdecide) (by evm_ov)
  have r9 := r8.swap6 (by evm_kdecide) (by evm_ov)
  have r10 := r9.swap5 (by evm_kdecide) (by evm_ov)
  have r11 := r10.pop (by evm_kdecide) (by evm_ov)
  have r12 := r11.pop (by evm_kdecide) (by evm_ov)
  have r13 := r12.pop (by evm_kdecide) (by evm_ov)
  have r14 := r13.pop (by evm_kdecide) (by evm_ov)
  have r15 := r14.pop (by evm_kdecide) (by evm_ov)
  have r16 := r15.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r16 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_11113_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x7 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 11113) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 x7 (l1cdm_block_11113_stack (x0 := x0) (R := R)) (l1cdm_block_11113_memory (mem := mem) (x2 := x2) (x3 := x3)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_11113 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_11130`. -/
def l1cdm_block_11130_stack {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {x6 : UInt256} {R : List UInt256} : List UInt256 :=
  (x1 :: (x0 + (UInt256.ofNat 192)) :: (UInt256.ofNat 11205) :: (UInt256.ofNat 0) :: x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: R)

/-- Final memory for bytecode block summary `l1cdm_block_11130`. -/
def l1cdm_block_11130_memory {mem : ByteArray} {x0 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {x6 : UInt256} : ByteArray :=
  ((UInt256.ofNat 192).toByteArray.write 0 (x2.toByteArray.write 0 (x3.toByteArray.write 0 ((UInt256.land x4 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 ((UInt256.land x5 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 (x6.toByteArray.write 0 mem x0.toNat 32) (x0 + (UInt256.ofNat 32)).toNat 32) (x0 + (UInt256.ofNat 64)).toNat 32) (x0 + (UInt256.ofNat 96)).toNat 32) (x0 + (UInt256.ofNat 128)).toNat 32) (x0 + (UInt256.ofNat 160)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 11130. -/
theorem l1cdm_block_11130 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9662) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 11130) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9662) (l1cdm_block_11130_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (R := R)) (l1cdm_block_11130_memory (mem := mem) (x0 := x0) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6)) (M (M (M (M (M (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 96)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 128)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 160)) (⟨32⟩ : UInt256)) rdata σ (k + 43) (C + ((131) + (memExpansionCost aw x0 (⟨32⟩ : UInt256)) + (memExpansionCost (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 96)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 96)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 128)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M (M (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 96)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 128)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 160)) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup7 (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup2 (by evm_kdecide) (by evm_ov)
  have r4 := RD.genMstore r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r6 := r5.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup1 (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup9 (by evm_kdecide) (by evm_ov)
  have r9 := r8.and (by evm_kdecide) (by evm_ov)
  have r10 := r9.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup5 (by evm_kdecide) (by evm_ov)
  have r12 := r11.add (by evm_kdecide) (by evm_ov)
  have r13 := RD.genMstore r12 (by evm_kdecide) (by evm_ov)
  have r14 := r13.dup1 (by evm_kdecide) (by evm_ov)
  have r15 := r14.dup8 (by evm_kdecide) (by evm_ov)
  have r16 := r15.and (by evm_kdecide) (by evm_ov)
  have r17 := r16.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r18 := r17.dup5 (by evm_kdecide) (by evm_ov)
  have r19 := r18.add (by evm_kdecide) (by evm_ov)
  have r20 := RD.genMstore r19 (by evm_kdecide) (by evm_ov)
  have r21 := r20.pop (by evm_kdecide) (by evm_ov)
  have r22 := r21.dup5 (by evm_kdecide) (by evm_ov)
  have r23 := r22.push1 (UInt256.ofNat 96) (by evm_kdecide) (by evm_ov)
  have r24 := r23.dup4 (by evm_kdecide) (by evm_ov)
  have r25 := r24.add (by evm_kdecide) (by evm_ov)
  have r26 := RD.genMstore r25 (by evm_kdecide) (by evm_ov)
  have r27 := r26.dup4 (by evm_kdecide) (by evm_ov)
  have r28 := r27.push1 (UInt256.ofNat 128) (by evm_kdecide) (by evm_ov)
  have r29 := r28.dup4 (by evm_kdecide) (by evm_ov)
  have r30 := r29.add (by evm_kdecide) (by evm_ov)
  have r31 := RD.genMstore r30 (by evm_kdecide) (by evm_ov)
  have r32 := r31.push1 (UInt256.ofNat 192) (by evm_kdecide) (by evm_ov)
  have r33 := r32.push1 (UInt256.ofNat 160) (by evm_kdecide) (by evm_ov)
  have r34 := r33.dup4 (by evm_kdecide) (by evm_ov)
  have r35 := r34.add (by evm_kdecide) (by evm_ov)
  have r36 := RD.genMstore r35 (by evm_kdecide) (by evm_ov)
  have r37 := r36.push2 (UInt256.ofNat 11205) (by evm_kdecide) (by evm_ov)
  have r38 := r37.push1 (UInt256.ofNat 192) (by evm_kdecide) (by evm_ov)
  have r39 := r38.dup4 (by evm_kdecide) (by evm_ov)
  have r40 := r39.add (by evm_kdecide) (by evm_ov)
  have r41 := r40.dup5 (by evm_kdecide) (by evm_ov)
  have r42 := r41.push2 (UInt256.ofNat 9662) (by evm_kdecide) (by evm_ov)
  have r43 := r42.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 9662)) r43 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_11130_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9662) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 11130) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9662) (l1cdm_block_11130_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (R := R)) (l1cdm_block_11130_memory (mem := mem) (x0 := x0) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_11130 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_11205`. -/
def l1cdm_block_11205_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: R)

/-- Automatically generated RD summary for bytecode block at pc 11205. -/
theorem l1cdm_block_11205 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x9 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 11205) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 x9 (l1cdm_block_11205_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 12) (C + ((31))) := by
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
theorem l1cdm_block_11205_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x9 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 11205) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 x9 (l1cdm_block_11205_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_11205 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 11217. -/
theorem l1cdm_block_11217 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 11217) R mem aw rdata σ k C)
    : RDinvalid L1cdmEvm.l1cdmRuntime g s0 := by
  let r0 := h
  exact RD.invalid r0 (by evm_kdecide)

end l1cdmBlocks
