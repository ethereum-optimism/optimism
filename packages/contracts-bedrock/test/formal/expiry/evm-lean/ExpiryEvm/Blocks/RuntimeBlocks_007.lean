import Reasoning.Reach
import ExpiryEvm.Bytecode

open Solm ABI Ethereum Ethereum.EVM
open Reasoning.Theory Reasoning.Reach

namespace l2tol2Blocks

/-- Final stack for bytecode block summary `l2tol2_block_1232`. -/
def l2tol2_block_1232_stack {R : List UInt256} : List UInt256 :=
  R

/-- Automatically generated RD summary for bytecode block at pc 1232. -/
theorem l2tol2_block_1232 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 1 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x0 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1232) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 x0 (l2tol2_block_1232_stack (R := R)) mem aw rdata σ (k + 1) (C + ((8))) := by
  let r0 := h
  have r1 := r0.jump (by native_decide) hvalid (by evm_ov)
  exact RD.normalizeCounters r1 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1232_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 1 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x0 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1232) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 x0 (l2tol2_block_1232_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_1232 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_1233_taken`. -/
def l2tol2_block_1233_taken_stack {mem : ByteArray} {σ : AccountMap} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero (extCodeSizeWord σ x2)) :: x2 :: (⟨0⟩ : UInt256) :: (memLoad (UInt256.ofNat 64) mem) :: (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem)) :: (memLoad (UInt256.ofNat 64) mem) :: (⟨0⟩ : UInt256) :: x0 :: x1 :: x2 :: R)

/-- Automatically generated RD summary for bytecode block at pc 1233. -/
theorem l2tol2_block_1233_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero (extCodeSizeWord σ x2))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 1256) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1233) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1256) (l2tol2_block_1233_taken_stack (mem := mem) (σ := σ) (x0 := x0) (x1 := x1) (x2 := x2) (R := R)) mem (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.push0 (by native_decide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 64) (by native_decide) (by evm_ov)
  have r4 := RD.genMload r3 (by native_decide) (by evm_ov)
  have r5 := r4.dup1 (by native_decide) (by evm_ov)
  have r6 := r5.dup4 (by native_decide) (by evm_ov)
  have r7 := r6.sub (by native_decide) (by evm_ov)
  have r8 := r7.dup2 (by native_decide) (by evm_ov)
  have r9 := r8.push0 (by native_decide) (by evm_ov)
  have r10 := r9.dup8 (by native_decide) (by evm_ov)
  have r11 := r10.dup1 (by native_decide) (by evm_ov)
  obtain ⟨_, _, r12⟩ := r11.extcodesize (by native_decide) (by evm_ov)
  have r13 := r12.iszero (by native_decide) (by evm_ov)
  have r14 := r13.dup1 (by native_decide) (by evm_ov)
  have r15 := r14.iszero (by native_decide) (by evm_ov)
  have r16 := r15.push2 (UInt256.ofNat 1256) (by native_decide) (by evm_ov)
  have r17 := r16.jumpiT (by native_decide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1256)) r17 (by native_decide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1233_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero (extCodeSizeWord σ x2))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 1256) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1233) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1256) (l2tol2_block_1233_taken_stack (mem := mem) (σ := σ) (x0 := x0) (x1 := x1) (x2 := x2) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k0, C0, h0⟩ := l2tol2_block_1233_taken hstack hcond hvalid h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_1233_fallthrough`. -/
def l2tol2_block_1233_fallthrough_stack {mem : ByteArray} {σ : AccountMap} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero (extCodeSizeWord σ x2)) :: x2 :: (⟨0⟩ : UInt256) :: (memLoad (UInt256.ofNat 64) mem) :: (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem)) :: (memLoad (UInt256.ofNat 64) mem) :: (⟨0⟩ : UInt256) :: x0 :: x1 :: x2 :: R)

/-- Automatically generated RD summary for bytecode block at pc 1233. -/
theorem l2tol2_block_1233_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero (extCodeSizeWord σ x2))) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1233) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1253) (l2tol2_block_1233_fallthrough_stack (mem := mem) (σ := σ) (x0 := x0) (x1 := x1) (x2 := x2) (R := R)) mem (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.push0 (by native_decide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 64) (by native_decide) (by evm_ov)
  have r4 := RD.genMload r3 (by native_decide) (by evm_ov)
  have r5 := r4.dup1 (by native_decide) (by evm_ov)
  have r6 := r5.dup4 (by native_decide) (by evm_ov)
  have r7 := r6.sub (by native_decide) (by evm_ov)
  have r8 := r7.dup2 (by native_decide) (by evm_ov)
  have r9 := r8.push0 (by native_decide) (by evm_ov)
  have r10 := r9.dup8 (by native_decide) (by evm_ov)
  have r11 := r10.dup1 (by native_decide) (by evm_ov)
  obtain ⟨_, _, r12⟩ := r11.extcodesize (by native_decide) (by evm_ov)
  have r13 := r12.iszero (by native_decide) (by evm_ov)
  have r14 := r13.dup1 (by native_decide) (by evm_ov)
  have r15 := r14.iszero (by native_decide) (by evm_ov)
  have r16 := r15.push2 (UInt256.ofNat 1256) (by native_decide) (by evm_ov)
  have r17 := r16.jumpiNT (by native_decide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1253)) r17 (by native_decide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1233_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 12 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero (extCodeSizeWord σ x2))) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1233) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1253) (l2tol2_block_1233_fallthrough_stack (mem := mem) (σ := σ) (x0 := x0) (x1 := x1) (x2 := x2) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k0, C0, h0⟩ := l2tol2_block_1233_fallthrough hstack hcond h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1253. -/
theorem l2tol2_block_1253 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1253) R mem aw rdata σ k C)
    : RDrev ExpiryEvm.l2tol2Runtime g s0 := by
  let r0 := h
  have r1 := r0.push0 (by native_decide) (by evm_ov)
  have r2 := r1.dup1 (by native_decide) (by evm_ov)
  exact RD.genRev r2 (by native_decide) (by evm_ov)

/-- Final stack for bytecode block summary `l2tol2_block_1256`. -/
def l2tol2_block_1256_stack {g : Sat256} {C : ℕ} {R : List UInt256} : List UInt256 :=
  (((g.subNat (C + ((3)) + 2)).toUInt256) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1256. -/
theorem l2tol2_block_1256 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 1 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1256) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1259) (l2tol2_block_1256_stack (g := g) (C := C) (R := R)) mem aw rdata σ (k + 3) (C + ((5))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.pop (by native_decide) (by evm_ov)
  have r3 := RD.genGas (RD.normalizeCounters (k' := k + 2) (C' := C + ((3))) r2 (by omega) (by omega)) (by native_decide) (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1259)) r3 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1256_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 1 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1256) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1259) (l2tol2_block_1256_stack (g := g) (C := C) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_1256 hstack h)
  exact ⟨_, k', C', h'⟩

/- Unsupported instruction boundary at pc 1259: call (0xf1). No RD transition is asserted. Summaries resume at pc 1260 from a fresh symbolic RD state. -/

/-- Final stack for bytecode block summary `l2tol2_block_1260_taken`. -/
def l2tol2_block_1260_taken_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1260. -/
theorem l2tol2_block_1260_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 1274) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1260) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1274) (l2tol2_block_1260_taken_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.iszero (by native_decide) (by evm_ov)
  have r2 := r1.dup1 (by native_decide) (by evm_ov)
  have r3 := r2.iszero (by native_decide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 1274) (by native_decide) (by evm_ov)
  have r5 := r4.jumpiT (by native_decide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1274)) r5 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1260_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 1274) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1260) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1274) (l2tol2_block_1260_taken_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_1260_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_1260_fallthrough`. -/
def l2tol2_block_1260_fallthrough_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1260. -/
theorem l2tol2_block_1260_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1260) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1267) (l2tol2_block_1260_fallthrough_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.iszero (by native_decide) (by evm_ov)
  have r2 := r1.dup1 (by native_decide) (by evm_ov)
  have r3 := r2.iszero (by native_decide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 1274) (by native_decide) (by evm_ov)
  have r5 := r4.jumpiNT (by native_decide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1267)) r5 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1260_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1260) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1267) (l2tol2_block_1260_fallthrough_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_1260_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1267. -/
theorem l2tol2_block_1267 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1267) R mem aw rdata σ k C)
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

/-- Final stack for bytecode block summary `l2tol2_block_1274`. -/
def l2tol2_block_1274_stack {x4 : UInt256} {R : List UInt256} : List UInt256 :=
  (x4 :: R)

/-- Automatically generated RD summary for bytecode block at pc 1274. -/
theorem l2tol2_block_1274 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 x10 x11 x12 x13 : UInt256} {R : List UInt256}
    (hstack : R.length + 14 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x13 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1274) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: x10 :: x11 :: x12 :: x13 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 x13 (l2tol2_block_1274_stack (x4 := x4) (R := R)) mem aw rdata σ (k + 16) (C + ((39))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.pop (by native_decide) (by evm_ov)
  have r3 := r2.pop (by native_decide) (by evm_ov)
  have r4 := r3.pop (by native_decide) (by evm_ov)
  have r5 := r4.pop (by native_decide) (by evm_ov)
  have r6 := r5.swap9 (by native_decide) (by evm_ov)
  have r7 := r6.swap8 (by native_decide) (by evm_ov)
  have r8 := r7.pop (by native_decide) (by evm_ov)
  have r9 := r8.pop (by native_decide) (by evm_ov)
  have r10 := r9.pop (by native_decide) (by evm_ov)
  have r11 := r10.pop (by native_decide) (by evm_ov)
  have r12 := r11.pop (by native_decide) (by evm_ov)
  have r13 := r12.pop (by native_decide) (by evm_ov)
  have r14 := r13.pop (by native_decide) (by evm_ov)
  have r15 := r14.pop (by native_decide) (by evm_ov)
  have r16 := r15.jump (by native_decide) hvalid (by evm_ov)
  exact RD.normalizeCounters r16 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1274_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 x10 x11 x12 x13 : UInt256} {R : List UInt256}
    (hstack : R.length + 14 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x13 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1274) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: x10 :: x11 :: x12 :: x13 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 x13 (l2tol2_block_1274_stack (x4 := x4) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_1274 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_1290_taken`. -/
def l2tol2_block_1290_taken_stack {R : List UInt256} : List UInt256 :=
  ((⟨0⟩ : UInt256) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1290. -/
theorem l2tol2_block_1290_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.tstorage.getD (UInt256.ofNat 109101767571825637468208191185038832016786111485836296546894784897825145752586) (⟨0⟩ : UInt256))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 1379) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1290) R mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1379) (l2tol2_block_1290_taken_stack (R := R)) mem aw rdata σ (k + 6) (C + ((19) + (Ctload))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.push0 (by native_decide) (by evm_ov)
  have r3 := r2.pushConst (UInt256.ofNat 109101767571825637468208191185038832016786111485836296546894784897825145752586) (width := 32) (op := .PUSH32) (by decide) (by native_decide) (by evm_ov)
  have r4 := r3.tload (by native_decide) (by evm_ov)
  have r5 := r4.push2 (UInt256.ofNat 1379) (by native_decide) (by evm_ov)
  have r6 := r5.jumpiT (by native_decide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1379)) r6 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1290_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.tstorage.getD (UInt256.ofNat 109101767571825637468208191185038832016786111485836296546894784897825145752586) (⟨0⟩ : UInt256))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 1379) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1290) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1379) (l2tol2_block_1290_taken_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_1290_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_1290_fallthrough`. -/
def l2tol2_block_1290_fallthrough_stack {R : List UInt256} : List UInt256 :=
  ((⟨0⟩ : UInt256) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1290. -/
theorem l2tol2_block_1290_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.tstorage.getD (UInt256.ofNat 109101767571825637468208191185038832016786111485836296546894784897825145752586) (⟨0⟩ : UInt256))) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1290) R mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1330) (l2tol2_block_1290_fallthrough_stack (R := R)) mem aw rdata σ (k + 6) (C + ((19) + (Ctload))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.push0 (by native_decide) (by evm_ov)
  have r3 := r2.pushConst (UInt256.ofNat 109101767571825637468208191185038832016786111485836296546894784897825145752586) (width := 32) (op := .PUSH32) (by decide) (by native_decide) (by evm_ov)
  have r4 := r3.tload (by native_decide) (by evm_ov)
  have r5 := r4.push2 (UInt256.ofNat 1379) (by native_decide) (by evm_ov)
  have r6 := r5.jumpiNT (by native_decide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1330)) r6 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1290_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.tstorage.getD (UInt256.ofNat 109101767571825637468208191185038832016786111485836296546894784897825145752586) (⟨0⟩ : UInt256))) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1290) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1330) (l2tol2_block_1290_fallthrough_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_1290_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1330. -/
theorem l2tol2_block_1330 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1330) R mem aw rdata σ k C)
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

/-- Final stack for bytecode block summary `l2tol2_block_1379`. -/
def l2tol2_block_1379_stack {ee : ExecutionEnv} {σ : AccountMap} {R : List UInt256} : List UInt256 :=
  ((σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.tstorage.getD (UInt256.ofNat 51164317248826882881343213066748959193860054907587800719701918628625062166247) (⟨0⟩ : UInt256))) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1379. -/
theorem l2tol2_block_1379 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x1 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1379) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 x1 (l2tol2_block_1379_stack (ee := ee) (σ := σ) (R := R)) mem aw rdata σ (k + 6) (C + ((17) + (Ctload))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.pop (by native_decide) (by evm_ov)
  have r3 := r2.pushConst (UInt256.ofNat 51164317248826882881343213066748959193860054907587800719701918628625062166247) (width := 32) (op := .PUSH32) (by decide) (by native_decide) (by evm_ov)
  have r4 := r3.tload (by native_decide) (by evm_ov)
  have r5 := r4.swap1 (by native_decide) (by evm_ov)
  have r6 := r5.jump (by native_decide) hvalid (by evm_ov)
  exact RD.normalizeCounters r6 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1379_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x1 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1379) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 x1 (l2tol2_block_1379_stack (ee := ee) (σ := σ) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_1379 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_1417_taken`. -/
def l2tol2_block_1417_taken_stack {R : List UInt256} : List UInt256 :=
  ((⟨0⟩ : UInt256) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1417. -/
theorem l2tol2_block_1417_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.tstorage.getD (UInt256.ofNat 109101767571825637468208191185038832016786111485836296546894784897825145752586) (⟨0⟩ : UInt256))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 1506) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1417) R mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1506) (l2tol2_block_1417_taken_stack (R := R)) mem aw rdata σ (k + 6) (C + ((19) + (Ctload))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.push0 (by native_decide) (by evm_ov)
  have r3 := r2.pushConst (UInt256.ofNat 109101767571825637468208191185038832016786111485836296546894784897825145752586) (width := 32) (op := .PUSH32) (by decide) (by native_decide) (by evm_ov)
  have r4 := r3.tload (by native_decide) (by evm_ov)
  have r5 := r4.push2 (UInt256.ofNat 1506) (by native_decide) (by evm_ov)
  have r6 := r5.jumpiT (by native_decide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1506)) r6 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1417_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.tstorage.getD (UInt256.ofNat 109101767571825637468208191185038832016786111485836296546894784897825145752586) (⟨0⟩ : UInt256))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 1506) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1417) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1506) (l2tol2_block_1417_taken_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_1417_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_1417_fallthrough`. -/
def l2tol2_block_1417_fallthrough_stack {R : List UInt256} : List UInt256 :=
  ((⟨0⟩ : UInt256) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1417. -/
theorem l2tol2_block_1417_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.tstorage.getD (UInt256.ofNat 109101767571825637468208191185038832016786111485836296546894784897825145752586) (⟨0⟩ : UInt256))) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1417) R mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1457) (l2tol2_block_1417_fallthrough_stack (R := R)) mem aw rdata σ (k + 6) (C + ((19) + (Ctload))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.push0 (by native_decide) (by evm_ov)
  have r3 := r2.pushConst (UInt256.ofNat 109101767571825637468208191185038832016786111485836296546894784897825145752586) (width := 32) (op := .PUSH32) (by decide) (by native_decide) (by evm_ov)
  have r4 := r3.tload (by native_decide) (by evm_ov)
  have r5 := r4.push2 (UInt256.ofNat 1506) (by native_decide) (by evm_ov)
  have r6 := r5.jumpiNT (by native_decide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1457)) r6 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1417_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.tstorage.getD (UInt256.ofNat 109101767571825637468208191185038832016786111485836296546894784897825145752586) (⟨0⟩ : UInt256))) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1417) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1457) (l2tol2_block_1417_fallthrough_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_1417_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1457. -/
theorem l2tol2_block_1457 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1457) R mem aw rdata σ k C)
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

/-- Final stack for bytecode block summary `l2tol2_block_1506`. -/
def l2tol2_block_1506_stack {ee : ExecutionEnv} {σ : AccountMap} {R : List UInt256} : List UInt256 :=
  ((σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.tstorage.getD (UInt256.ofNat 83317915124952138165281597070141736405785001779305661129532428155124339880947) (⟨0⟩ : UInt256))) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1506. -/
theorem l2tol2_block_1506 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x1 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1506) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 x1 (l2tol2_block_1506_stack (ee := ee) (σ := σ) (R := R)) mem aw rdata σ (k + 6) (C + ((17) + (Ctload))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.pop (by native_decide) (by evm_ov)
  have r3 := r2.pushConst (UInt256.ofNat 83317915124952138165281597070141736405785001779305661129532428155124339880947) (width := 32) (op := .PUSH32) (by decide) (by native_decide) (by evm_ov)
  have r4 := r3.tload (by native_decide) (by evm_ov)
  have r5 := r4.swap1 (by native_decide) (by evm_ov)
  have r6 := r5.jump (by native_decide) hvalid (by evm_ov)
  exact RD.normalizeCounters r6 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1506_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x1 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1506) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 x1 (l2tol2_block_1506_stack (ee := ee) (σ := σ) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_1506 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_1544_taken`. -/
def l2tol2_block_1544_taken_stack {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {R : List UInt256} : List UInt256 :=
  ((⟨0⟩ : UInt256) :: x0 :: x1 :: x2 :: x3 :: R)

/-- Automatically generated RD summary for bytecode block at pc 1544. -/
theorem l2tol2_block_1544_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : (UInt256.sub x3 (UInt256.ofNat Ethereum.chainId)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 1602) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1544) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1602) (l2tol2_block_1544_taken_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (R := R)) mem aw rdata σ (k + 7) (C + ((24))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.push0 (by native_decide) (by evm_ov)
  have r3 := r2.chainid (by native_decide) (by evm_ov)
  have r4 := r3.dup6 (by native_decide) (by evm_ov)
  have r5 := r4.sub (by native_decide) (by evm_ov)
  have r6 := r5.push2 (UInt256.ofNat 1602) (by native_decide) (by evm_ov)
  have r7 := r6.jumpiT (by native_decide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1602)) r7 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1544_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : (UInt256.sub x3 (UInt256.ofNat Ethereum.chainId)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 1602) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1544) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1602) (l2tol2_block_1544_taken_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_1544_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_1544_fallthrough`. -/
def l2tol2_block_1544_fallthrough_stack {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {R : List UInt256} : List UInt256 :=
  ((⟨0⟩ : UInt256) :: x0 :: x1 :: x2 :: x3 :: R)

/-- Automatically generated RD summary for bytecode block at pc 1544. -/
theorem l2tol2_block_1544_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : (UInt256.sub x3 (UInt256.ofNat Ethereum.chainId)) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1544) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1553) (l2tol2_block_1544_fallthrough_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (R := R)) mem aw rdata σ (k + 7) (C + ((24))) := by
  let r0 := h
  have r1 := r0.jumpdest (by native_decide) (by evm_ov)
  have r2 := r1.push0 (by native_decide) (by evm_ov)
  have r3 := r2.chainid (by native_decide) (by evm_ov)
  have r4 := r3.dup6 (by native_decide) (by evm_ov)
  have r5 := r4.sub (by native_decide) (by evm_ov)
  have r6 := r5.push2 (UInt256.ofNat 1602) (by native_decide) (by evm_ov)
  have r7 := r6.jumpiNT (by native_decide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1553)) r7 (by native_decide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1544_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : (UInt256.sub x3 (UInt256.ofNat Ethereum.chainId)) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1544) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1553) (l2tol2_block_1544_fallthrough_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_1544_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1553. -/
theorem l2tol2_block_1553 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1553) R mem aw rdata σ k C)
    : RDrev ExpiryEvm.l2tol2Runtime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 64) (by native_decide) (by evm_ov)
  have r2 := RD.genMload r1 (by native_decide) (by evm_ov)
  have r3 := r2.pushConst (UInt256.ofNat 64612999214288384198821183682440915115422375510515237771150080384299387846656) (width := 32) (op := .PUSH32) (by decide) (by native_decide) (by evm_ov)
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

end l2tol2Blocks
