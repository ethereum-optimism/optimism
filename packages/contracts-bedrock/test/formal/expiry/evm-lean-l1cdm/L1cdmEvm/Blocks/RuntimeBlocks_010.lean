import Reasoning.Reach
import L1cdmEvm.Bytecode

open Solm ABI Ethereum Ethereum.EVM
open Reasoning.Theory Reasoning.Reach

namespace l1cdmBlocks

/-- Final stack for bytecode block summary `l1cdm_block_1253`. -/
def l1cdm_block_1253_stack {ee : ExecutionEnv} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 4) :: (UInt256.ofNat ee.calldata.size) :: (UInt256.ofNat 1268) :: (UInt256.ofNat 1093) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1253. -/
theorem l1cdm_block_1253 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9788) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1253) (x0 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9788) (l1cdm_block_1253_stack (ee := ee) (R := R)) mem aw rdata σ (k + 8) (C + ((25))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := r2.push2 (UInt256.ofNat 1093) (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 1268) (by evm_kdecide) (by evm_ov)
  have r5 := r4.calldatasize (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r7 := r6.push2 (UInt256.ofNat 9788) (by evm_kdecide) (by evm_ov)
  have r8 := r7.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 9788)) r8 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1253_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9788) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1253) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9788) (l1cdm_block_1253_stack (ee := ee) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_1253 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_1268`. -/
def l1cdm_block_1268_stack {ee : ExecutionEnv} {mem : ByteArray} {σ : AccountMap} {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.land (UInt256.ofNat 255) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) (x0.toByteArray.write 0 ((UInt256.ofNat 206).toByteArray.write 0 mem (UInt256.ofNat 32).toNat 32) (UInt256.ofNat 0).toNat 32)) (⟨0⟩ : UInt256)))) :: x1 :: R)

/-- Final memory for bytecode block summary `l1cdm_block_1268`. -/
def l1cdm_block_1268_memory {mem : ByteArray} {x0 : UInt256} : ByteArray :=
  (x0.toByteArray.write 0 ((UInt256.ofNat 206).toByteArray.write 0 mem (UInt256.ofNat 32).toNat 32) (UInt256.ofNat 0).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1268. -/
theorem l1cdm_block_1268 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x1 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1268) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 x1 (l1cdm_block_1268_stack (ee := ee) (mem := mem) (σ := σ) (x0 := x0) (x1 := x1) (R := R)) (l1cdm_block_1268_memory (mem := mem) (x0 := x0)) (M (M (M aw (UInt256.ofNat 32) (⟨32⟩ : UInt256)) (UInt256.ofNat 0) (⟨32⟩ : UInt256)) (UInt256.ofNat 0) (UInt256.ofNat 64)) rdata σ k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 206) (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r4 := RD.genMstore r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r6 := r5.swap1 (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup2 (by evm_kdecide) (by evm_ov)
  have r8 := RD.genMstore r7 (by evm_kdecide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r10 := r9.swap1 (by evm_kdecide) (by evm_ov)
  have r11 := RD.genKeccak256 r10 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r12⟩ := RD.sload r11 (by evm_kdecide) (by evm_ov)
  have r13 := r12.push1 (UInt256.ofNat 255) (by evm_kdecide) (by evm_ov)
  have r14 := r13.and (by evm_kdecide) (by evm_ov)
  have r15 := r14.dup2 (by evm_kdecide) (by evm_ov)
  have r16 := r15.jump (by evm_kdecide) hvalid (by evm_ov)
  exact ⟨_, _, r16⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1268_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x1 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1268) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 x1 (l1cdm_block_1268_stack (ee := ee) (mem := mem) (σ := σ) (x0 := x0) (x1 := x1) (R := R)) (l1cdm_block_1268_memory (mem := mem) (x0 := x0)) aw' rdata σ k' C' := by
  obtain ⟨k0, C0, h0⟩ := l1cdm_block_1268 hstack hvalid h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_1289_taken`. -/
def l1cdm_block_1289_taken_stack {ee : ExecutionEnv} {R : List UInt256} : List UInt256 :=
  (ee.weiValue :: R)

/-- Automatically generated RD summary for bytecode block at pc 1289. -/
theorem l1cdm_block_1289_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 1301) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1289) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1301) (l1cdm_block_1289_taken_stack (ee := ee) (R := R)) mem aw rdata σ (k + 6) (C + ((22))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.callvalue (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.iszero (by evm_kdecide) (by evm_ov)
  have r5 := r4.push2 (UInt256.ofNat 1301) (by evm_kdecide) (by evm_ov)
  have r6 := r5.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1301)) r6 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1289_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 1301) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1289) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1301) (l1cdm_block_1289_taken_stack (ee := ee) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_1289_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_1289_fallthrough`. -/
def l1cdm_block_1289_fallthrough_stack {ee : ExecutionEnv} {R : List UInt256} : List UInt256 :=
  (ee.weiValue :: R)

/-- Automatically generated RD summary for bytecode block at pc 1289. -/
theorem l1cdm_block_1289_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1289) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1297) (l1cdm_block_1289_fallthrough_stack (ee := ee) (R := R)) mem aw rdata σ (k + 6) (C + ((22))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.callvalue (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.iszero (by evm_kdecide) (by evm_ov)
  have r5 := r4.push2 (UInt256.ofNat 1301) (by evm_kdecide) (by evm_ov)
  have r6 := r5.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1297)) r6 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1289_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1289) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1297) (l1cdm_block_1289_fallthrough_stack (ee := ee) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_1289_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1297. -/
theorem l1cdm_block_1297 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1297) R mem aw rdata σ k C)
    : RDrev L1cdmEvm.l1cdmRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `l1cdm_block_1301`. -/
def l1cdm_block_1301_stack {ee : ExecutionEnv} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 4) :: (UInt256.ofNat ee.calldata.size) :: (UInt256.ofNat 1316) :: (UInt256.ofNat 1093) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1301. -/
theorem l1cdm_block_1301 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9788) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1301) (x0 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9788) (l1cdm_block_1301_stack (ee := ee) (R := R)) mem aw rdata σ (k + 8) (C + ((25))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := r2.push2 (UInt256.ofNat 1093) (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 1316) (by evm_kdecide) (by evm_ov)
  have r5 := r4.calldatasize (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r7 := r6.push2 (UInt256.ofNat 9788) (by evm_kdecide) (by evm_ov)
  have r8 := r7.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 9788)) r8 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1301_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9788) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1301) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9788) (l1cdm_block_1301_stack (ee := ee) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_1301 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_1316`. -/
def l1cdm_block_1316_stack {ee : ExecutionEnv} {mem : ByteArray} {σ : AccountMap} {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.land (UInt256.ofNat 255) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (keccakWord (UInt256.ofNat 0) (UInt256.ofNat 64) (x0.toByteArray.write 0 ((UInt256.ofNat 203).toByteArray.write 0 mem (UInt256.ofNat 32).toNat 32) (UInt256.ofNat 0).toNat 32)) (⟨0⟩ : UInt256)))) :: x1 :: R)

/-- Final memory for bytecode block summary `l1cdm_block_1316`. -/
def l1cdm_block_1316_memory {mem : ByteArray} {x0 : UInt256} : ByteArray :=
  (x0.toByteArray.write 0 ((UInt256.ofNat 203).toByteArray.write 0 mem (UInt256.ofNat 32).toNat 32) (UInt256.ofNat 0).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1316. -/
theorem l1cdm_block_1316 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x1 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1316) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 x1 (l1cdm_block_1316_stack (ee := ee) (mem := mem) (σ := σ) (x0 := x0) (x1 := x1) (R := R)) (l1cdm_block_1316_memory (mem := mem) (x0 := x0)) (M (M (M aw (UInt256.ofNat 32) (⟨32⟩ : UInt256)) (UInt256.ofNat 0) (⟨32⟩ : UInt256)) (UInt256.ofNat 0) (UInt256.ofNat 64)) rdata σ k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 203) (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r4 := RD.genMstore r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r6 := r5.swap1 (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup2 (by evm_kdecide) (by evm_ov)
  have r8 := RD.genMstore r7 (by evm_kdecide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r10 := r9.swap1 (by evm_kdecide) (by evm_ov)
  have r11 := RD.genKeccak256 r10 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r12⟩ := RD.sload r11 (by evm_kdecide) (by evm_ov)
  have r13 := r12.push1 (UInt256.ofNat 255) (by evm_kdecide) (by evm_ov)
  have r14 := r13.and (by evm_kdecide) (by evm_ov)
  have r15 := r14.dup2 (by evm_kdecide) (by evm_ov)
  have r16 := r15.jump (by evm_kdecide) hvalid (by evm_ov)
  exact ⟨_, _, r16⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1316_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains x1 = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1316) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 x1 (l1cdm_block_1316_stack (ee := ee) (mem := mem) (σ := σ) (x0 := x0) (x1 := x1) (R := R)) (l1cdm_block_1316_memory (mem := mem) (x0 := x0)) aw' rdata σ k' C' := by
  obtain ⟨k0, C0, h0⟩ := l1cdm_block_1316 hstack hvalid h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_1337_taken`. -/
def l1cdm_block_1337_taken_stack {ee : ExecutionEnv} {R : List UInt256} : List UInt256 :=
  (ee.weiValue :: R)

/-- Automatically generated RD summary for bytecode block at pc 1337. -/
theorem l1cdm_block_1337_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 1349) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1337) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1349) (l1cdm_block_1337_taken_stack (ee := ee) (R := R)) mem aw rdata σ (k + 6) (C + ((22))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.callvalue (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.iszero (by evm_kdecide) (by evm_ov)
  have r5 := r4.push2 (UInt256.ofNat 1349) (by evm_kdecide) (by evm_ov)
  have r6 := r5.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1349)) r6 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1337_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 1349) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1337) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1349) (l1cdm_block_1337_taken_stack (ee := ee) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_1337_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_1337_fallthrough`. -/
def l1cdm_block_1337_fallthrough_stack {ee : ExecutionEnv} {R : List UInt256} : List UInt256 :=
  (ee.weiValue :: R)

/-- Automatically generated RD summary for bytecode block at pc 1337. -/
theorem l1cdm_block_1337_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1337) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1345) (l1cdm_block_1337_fallthrough_stack (ee := ee) (R := R)) mem aw rdata σ (k + 6) (C + ((22))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.callvalue (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.iszero (by evm_kdecide) (by evm_ov)
  have r5 := r4.push2 (UInt256.ofNat 1349) (by evm_kdecide) (by evm_ov)
  have r6 := r5.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1345)) r6 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1337_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1337) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1345) (l1cdm_block_1337_fallthrough_stack (ee := ee) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_1337_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1345. -/
theorem l1cdm_block_1345 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1345) R mem aw rdata σ k C)
    : RDrev L1cdmEvm.l1cdmRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `l1cdm_block_1349`. -/
def l1cdm_block_1349_stack {ee : ExecutionEnv} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 4) :: (UInt256.ofNat ee.calldata.size) :: (UInt256.ofNat 1364) :: (UInt256.ofNat 498) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1349. -/
theorem l1cdm_block_1349 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9860) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1349) (x0 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9860) (l1cdm_block_1349_stack (ee := ee) (R := R)) mem aw rdata σ (k + 8) (C + ((25))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := r2.push2 (UInt256.ofNat 498) (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 1364) (by evm_kdecide) (by evm_ov)
  have r5 := r4.calldatasize (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r7 := r6.push2 (UInt256.ofNat 9860) (by evm_kdecide) (by evm_ov)
  have r8 := r7.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 9860)) r8 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1349_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 9860) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1349) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 9860) (l1cdm_block_1349_stack (ee := ee) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_1349 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1364. -/
theorem l1cdm_block_1364 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 1 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 5384) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1364) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 5384) R mem aw rdata σ (k + 3) (C + ((12))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 5384) (by evm_kdecide) (by evm_ov)
  have r3 := r2.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 5384)) r3 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1364_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 1 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 5384) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1364) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 5384) R mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_1364 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_1369`. -/
def l1cdm_block_1369_stack {ee : ExecutionEnv} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 4) :: (UInt256.ofNat ee.calldata.size) :: (UInt256.ofNat 1383) :: (UInt256.ofNat 766) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1369. -/
theorem l1cdm_block_1369 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10084) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1369) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10084) (l1cdm_block_1369_stack (ee := ee) (R := R)) mem aw rdata σ (k + 7) (C + ((23))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 766) (by evm_kdecide) (by evm_ov)
  have r3 := r2.push2 (UInt256.ofNat 1383) (by evm_kdecide) (by evm_ov)
  have r4 := r3.calldatasize (by evm_kdecide) (by evm_ov)
  have r5 := r4.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r6 := r5.push2 (UInt256.ofNat 10084) (by evm_kdecide) (by evm_ov)
  have r7 := r6.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 10084)) r7 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1369_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 10084) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1369) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 10084) (l1cdm_block_1369_stack (ee := ee) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_1369 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1383. -/
theorem l1cdm_block_1383 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 1 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 5577) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1383) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 5577) R mem aw rdata σ (k + 3) (C + ((12))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 5577) (by evm_kdecide) (by evm_ov)
  have r3 := r2.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 5577)) r3 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1383_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 1 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 5577) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1383) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 5577) R mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_1383 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_1388_taken`. -/
def l1cdm_block_1388_taken_stack {ee : ExecutionEnv} {R : List UInt256} : List UInt256 :=
  (ee.weiValue :: R)

/-- Automatically generated RD summary for bytecode block at pc 1388. -/
theorem l1cdm_block_1388_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 1400) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1388) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1400) (l1cdm_block_1388_taken_stack (ee := ee) (R := R)) mem aw rdata σ (k + 6) (C + ((22))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.callvalue (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.iszero (by evm_kdecide) (by evm_ov)
  have r5 := r4.push2 (UInt256.ofNat 1400) (by evm_kdecide) (by evm_ov)
  have r6 := r5.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1400)) r6 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1388_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 1400) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1388) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1400) (l1cdm_block_1388_taken_stack (ee := ee) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_1388_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_1388_fallthrough`. -/
def l1cdm_block_1388_fallthrough_stack {ee : ExecutionEnv} {R : List UInt256} : List UInt256 :=
  (ee.weiValue :: R)

/-- Automatically generated RD summary for bytecode block at pc 1388. -/
theorem l1cdm_block_1388_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1388) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1396) (l1cdm_block_1388_fallthrough_stack (ee := ee) (R := R)) mem aw rdata σ (k + 6) (C + ((22))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.callvalue (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.iszero (by evm_kdecide) (by evm_ov)
  have r5 := r4.push2 (UInt256.ofNat 1400) (by evm_kdecide) (by evm_ov)
  have r6 := r5.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1396)) r6 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1388_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1388) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1396) (l1cdm_block_1388_fallthrough_stack (ee := ee) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_1388_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1396. -/
theorem l1cdm_block_1396 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1396) R mem aw rdata σ k C)
    : RDrev L1cdmEvm.l1cdmRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `l1cdm_block_1400`. -/
def l1cdm_block_1400_stack {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 588) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1400. -/
theorem l1cdm_block_1400 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 7902) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1400) (x0 :: R) mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7902) (l1cdm_block_1400_stack (R := R)) mem aw rdata σ (k + 5) (C + ((17))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := r2.push2 (UInt256.ofNat 588) (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 7902) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 7902)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1400_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 7902) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1400) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 7902) (l1cdm_block_1400_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_1400 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_1409_taken`. -/
def l1cdm_block_1409_taken_stack {ee : ExecutionEnv} {R : List UInt256} : List UInt256 :=
  (ee.weiValue :: R)

/-- Automatically generated RD summary for bytecode block at pc 1409. -/
theorem l1cdm_block_1409_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 1421) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1409) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1421) (l1cdm_block_1409_taken_stack (ee := ee) (R := R)) mem aw rdata σ (k + 6) (C + ((22))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.callvalue (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.iszero (by evm_kdecide) (by evm_ov)
  have r5 := r4.push2 (UInt256.ofNat 1421) (by evm_kdecide) (by evm_ov)
  have r6 := r5.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1421)) r6 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1409_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J L1cdmEvm.l1cdmRuntime 0).contains (UInt256.ofNat 1421) = true)
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1409) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1421) (l1cdm_block_1409_taken_stack (ee := ee) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_1409_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l1cdm_block_1409_fallthrough`. -/
def l1cdm_block_1409_fallthrough_stack {ee : ExecutionEnv} {R : List UInt256} : List UInt256 :=
  (ee.weiValue :: R)

/-- Automatically generated RD summary for bytecode block at pc 1409. -/
theorem l1cdm_block_1409_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1409) R mem aw rdata σ k C)
    : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1417) (l1cdm_block_1409_fallthrough_stack (ee := ee) (R := R)) mem aw rdata σ (k + 6) (C + ((22))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.callvalue (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.iszero (by evm_kdecide) (by evm_ov)
  have r5 := r4.push2 (UInt256.ofNat 1421) (by evm_kdecide) (by evm_ov)
  have r6 := r5.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1417)) r6 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l1cdm_block_1409_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero ee.weiValue) = (UInt256.ofNat 0))
    (h : RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1409) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD L1cdmEvm.l1cdmRuntime ee g s0 (UInt256.ofNat 1417) (l1cdm_block_1409_fallthrough_stack (ee := ee) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l1cdm_block_1409_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

end l1cdmBlocks
