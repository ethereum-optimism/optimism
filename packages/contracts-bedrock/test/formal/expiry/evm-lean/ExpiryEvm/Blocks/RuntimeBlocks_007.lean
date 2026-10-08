import Reasoning.Reach
import ExpiryEvm.Bytecode

open Solm ABI Ethereum Ethereum.EVM
open Reasoning.Theory Reasoning.Reach

namespace l2tol2Blocks

/-- Automatically generated RD summary for bytecode block at pc 1167. -/
theorem l2tol2_block_1167_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : ((UInt256.land x3 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)) + (UInt256.ofNat 115792089237316195423570985008311114462395611257041176543522917291908084531165)) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1167) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1229) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ (k + 8) (C + ((29))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pushConst (UInt256.ofNat 115792089237316195423570985008311114462395611257041176543522917291908084531165) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r3 := r2.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup6 (by evm_kdecide) (by evm_ov)
  have r5 := r4.and (by evm_kdecide) (by evm_ov)
  have r6 := r5.add (by evm_kdecide) (by evm_ov)
  have r7 := r6.push2 (UInt256.ofNat 1278) (by evm_kdecide) (by evm_ov)
  have r8 := r7.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1229)) r8 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1167_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : ((UInt256.land x3 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)) + (UInt256.ofNat 115792089237316195423570985008311114462395611257041176543522917291908084531165)) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1167) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1229) (x0 :: x1 :: x2 :: x3 :: R) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_1167_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1229. -/
theorem l2tol2_block_1229 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1229) R mem aw rdata σ k C)
    : RDrev ExpiryEvm.l2tol2Runtime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r2 := RD.genMload r1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pushConst (UInt256.ofNat 36033334646344721272737662468121120244904123346647643139099730212490460528640) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup2 (by evm_kdecide) (by evm_ov)
  have r5 := RD.genMstore r4 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r7 := r6.add (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r9 := RD.genMload r8 (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup1 (by evm_kdecide) (by evm_ov)
  have r11 := r10.swap2 (by evm_kdecide) (by evm_ov)
  have r12 := r11.sub (by evm_kdecide) (by evm_ov)
  have r13 := r12.swap1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r13 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `l2tol2_block_1278`. -/
def l2tol2_block_1278_stack {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {R : List UInt256} : List UInt256 :=
  (x3 :: (UInt256.ofNat 1287) :: x0 :: x1 :: x2 :: x3 :: R)

/-- Automatically generated RD summary for bytecode block at pc 1278. -/
theorem l2tol2_block_1278 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 3490) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1278) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 3490) (l2tol2_block_1278_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (R := R)) mem aw rdata σ (k + 5) (C + ((18))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push2 (UInt256.ofNat 1287) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup5 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 3490) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 3490)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1278_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 3490) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1278) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 3490) (l2tol2_block_1278_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_1278 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_1287_taken`. -/
def l2tol2_block_1287_taken_stack {R : List UInt256} : List UInt256 :=
  R

/-- Automatically generated RD summary for bytecode block at pc 1287. -/
theorem l2tol2_block_1287_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (UInt256.isZero x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 1342) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1287) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1342) (l2tol2_block_1287_taken_stack (R := R)) mem aw rdata σ (k + 4) (C + ((17))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.iszero (by evm_kdecide) (by evm_ov)
  have r3 := r2.push2 (UInt256.ofNat 1342) (by evm_kdecide) (by evm_ov)
  have r4 := r3.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1342)) r4 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1287_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (UInt256.isZero x0) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 1342) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1287) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1342) (l2tol2_block_1287_taken_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_1287_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_1287_fallthrough`. -/
def l2tol2_block_1287_fallthrough_stack {R : List UInt256} : List UInt256 :=
  R

/-- Automatically generated RD summary for bytecode block at pc 1287. -/
theorem l2tol2_block_1287_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (UInt256.isZero x0) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1287) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1293) (l2tol2_block_1287_fallthrough_stack (R := R)) mem aw rdata σ (k + 4) (C + ((17))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.iszero (by evm_kdecide) (by evm_ov)
  have r3 := r2.push2 (UInt256.ofNat 1342) (by evm_kdecide) (by evm_ov)
  have r4 := r3.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1293)) r4 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1287_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (hcond : (UInt256.isZero x0) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1287) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1293) (l2tol2_block_1287_fallthrough_stack (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_1287_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1293. -/
theorem l2tol2_block_1293 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1293) R mem aw rdata σ k C)
    : RDrev ExpiryEvm.l2tol2Runtime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r2 := RD.genMload r1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pushConst (UInt256.ofNat 70304313862865592181084378138880879792529513439229040269615496009570261663744) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup2 (by evm_kdecide) (by evm_ov)
  have r5 := RD.genMstore r4 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r7 := r6.add (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r9 := RD.genMload r8 (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup1 (by evm_kdecide) (by evm_ov)
  have r11 := r10.swap2 (by evm_kdecide) (by evm_ov)
  have r12 := r11.sub (by evm_kdecide) (by evm_ov)
  have r13 := r12.swap1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r13 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `l2tol2_block_1342`. -/
def l2tol2_block_1342_stack {ee : ExecutionEnv} {σ : AccountMap} {R : List UInt256} : List UInt256 :=
  ((UInt256.land (UInt256.ofNat 1766847064778384329583297500742918515827483896875618958121606201292619775) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 1) (⟨0⟩ : UInt256)))) :: (⟨0⟩ : UInt256) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1342. -/
theorem l2tol2_block_1342 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 1384) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1342) R mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1384) (l2tol2_block_1342_stack (ee := ee) (σ := σ) (R := R)) mem aw rdata σ k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push0 (by evm_kdecide) (by evm_ov)
  have r3 := r2.push2 (UInt256.ofNat 1384) (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 1) (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r5⟩ := RD.sload r4 (by evm_kdecide) (by evm_ov)
  have r6 := r5.pushConst (UInt256.ofNat 1766847064778384329583297500742918515827483896875618958121606201292619775) (width := 30) (op := .PUSH30) (by decide) (by evm_kdecide) (by evm_ov)
  have r7 := r6.and (by evm_kdecide) (by evm_ov)
  have r8 := r7.swap1 (by evm_kdecide) (by evm_ov)
  have r9 := r8.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1384)) r9 (by evm_kdecide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1342_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 1384) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1342) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1384) (l2tol2_block_1342_stack (ee := ee) (σ := σ) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k0, C0, h0⟩ := l2tol2_block_1342 hstack hvalid h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_1384`. -/
def l2tol2_block_1384_stack {ee : ExecutionEnv} {mem : ByteArray} {x0 : UInt256} {x2 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {x6 : UInt256} {R : List UInt256} : List UInt256 :=
  ((memLoad (UInt256.ofNat 64) mem) :: x5 :: (UInt256.ofNat ee.source.val) :: x0 :: (UInt256.ofNat Ethereum.chainId) :: x6 :: (UInt256.ofNat 1453) :: x0 :: x2 :: x3 :: x4 :: x5 :: x6 :: R)

/-- Final memory for bytecode block summary `l2tol2_block_1384`. -/
def l2tol2_block_1384_memory {ee : ExecutionEnv} {mem : ByteArray} {x3 : UInt256} {x4 : UInt256} : ByteArray :=
  ((⟨0⟩ : UInt256).toByteArray.write 0 (ee.calldata.write x4.toNat (x3.toByteArray.write 0 (((memLoad (UInt256.ofNat 64) mem) + ((UInt256.ofNat 32) + (UInt256.mul (UInt256.div ((UInt256.ofNat 31) + x3) (UInt256.ofNat 32)) (UInt256.ofNat 32)))).toByteArray.write 0 mem (UInt256.ofNat 64).toNat 32) (memLoad (UInt256.ofNat 64) mem).toNat 32) ((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)).toNat x3.toNat) (((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) + x3).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1384. -/
theorem l2tol2_block_1384 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 : UInt256} {R : List UInt256}
    (hstack : R.length + 22 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 3594) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1384) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 3594) (l2tol2_block_1384_stack (ee := ee) (mem := mem) (x0 := x0) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (R := R)) (l2tol2_block_1384_memory (ee := ee) (mem := mem) (x3 := x3) (x4 := x4)) (M (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) x3) (((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) + x3) (⟨32⟩ : UInt256)) rdata σ (k + 59) (C + ((173) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) x3) + (3 + 3 * ((x3.toNat + 31) / 32)) + (memExpansionCost (M (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) ((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) x3) (((UInt256.ofNat 32) + (memLoad (UInt256.ofNat 64) mem)) + x3) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pop (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 1453) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup7 (by evm_kdecide) (by evm_ov)
  have r6 := r5.chainid (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup4 (by evm_kdecide) (by evm_ov)
  have r8 := r7.caller (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup10 (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup10 (by evm_kdecide) (by evm_ov)
  have r11 := r10.dup10 (by evm_kdecide) (by evm_ov)
  have r12 := r11.dup1 (by evm_kdecide) (by evm_ov)
  have r13 := r12.dup1 (by evm_kdecide) (by evm_ov)
  have r14 := r13.push1 (UInt256.ofNat 31) (by evm_kdecide) (by evm_ov)
  have r15 := r14.add (by evm_kdecide) (by evm_ov)
  have r16 := r15.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r17 := r16.dup1 (by evm_kdecide) (by evm_ov)
  have r18 := r17.swap2 (by evm_kdecide) (by evm_ov)
  have r19 := r18.div (by evm_kdecide) (by evm_ov)
  have r20 := r19.mul (by evm_kdecide) (by evm_ov)
  have r21 := r20.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r22 := r21.add (by evm_kdecide) (by evm_ov)
  have r23 := r22.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r24 := RD.genMload r23 (by evm_kdecide) (by evm_ov)
  have r25 := r24.swap1 (by evm_kdecide) (by evm_ov)
  have r26 := r25.dup2 (by evm_kdecide) (by evm_ov)
  have r27 := r26.add (by evm_kdecide) (by evm_ov)
  have r28 := r27.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r29 := RD.genMstore r28 (by evm_kdecide) (by evm_ov)
  have r30 := r29.dup1 (by evm_kdecide) (by evm_ov)
  have r31 := r30.swap4 (by evm_kdecide) (by evm_ov)
  have r32 := r31.swap3 (by evm_kdecide) (by evm_ov)
  have r33 := r32.swap2 (by evm_kdecide) (by evm_ov)
  have r34 := r33.swap1 (by evm_kdecide) (by evm_ov)
  have r35 := r34.dup2 (by evm_kdecide) (by evm_ov)
  have r36 := r35.dup2 (by evm_kdecide) (by evm_ov)
  have r37 := RD.genMstore r36 (by evm_kdecide) (by evm_ov)
  have r38 := r37.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r39 := r38.add (by evm_kdecide) (by evm_ov)
  have r40 := r39.dup4 (by evm_kdecide) (by evm_ov)
  have r41 := r40.dup4 (by evm_kdecide) (by evm_ov)
  have r42 := r41.dup1 (by evm_kdecide) (by evm_ov)
  have r43 := r42.dup3 (by evm_kdecide) (by evm_ov)
  have r44 := r43.dup5 (by evm_kdecide) (by evm_ov)
  have r45 := RD.genCalldatacopy r44 (by evm_kdecide) (by evm_ov)
  have r46 := r45.push0 (by evm_kdecide) (by evm_ov)
  have r47 := r46.swap3 (by evm_kdecide) (by evm_ov)
  have r48 := r47.add (by evm_kdecide) (by evm_ov)
  have r49 := r48.swap2 (by evm_kdecide) (by evm_ov)
  have r50 := r49.swap1 (by evm_kdecide) (by evm_ov)
  have r51 := r50.swap2 (by evm_kdecide) (by evm_ov)
  have r52 := RD.genMstore r51 (by evm_kdecide) (by evm_ov)
  have r53 := r52.pop (by evm_kdecide) (by evm_ov)
  have r54 := r53.push2 (UInt256.ofNat 3594) (by evm_kdecide) (by evm_ov)
  have r55 := r54.swap3 (by evm_kdecide) (by evm_ov)
  have r56 := r55.pop (by evm_kdecide) (by evm_ov)
  have r57 := r56.pop (by evm_kdecide) (by evm_ov)
  have r58 := r57.pop (by evm_kdecide) (by evm_ov)
  have r59 := r58.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 3594)) r59 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1384_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 : UInt256} {R : List UInt256}
    (hstack : R.length + 22 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 3594) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1384) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 3594) (l2tol2_block_1384_stack (ee := ee) (mem := mem) (x0 := x0) (x2 := x2) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (R := R)) (l2tol2_block_1384_memory (ee := ee) (mem := mem) (x3 := x3) (x4 := x4)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_1384 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_1453`. -/
def l2tol2_block_1453_stack {ee : ExecutionEnv} {mem : ByteArray} {σ : AccountMap} {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.land ((sstoreAccountMap ee.codeOwner (sstoreAccountMap ee.codeOwner σ (keccakWord (⟨0⟩ : UInt256) (UInt256.ofNat 64) ((UInt256.ofNat 2).toByteArray.write 0 (x1.toByteArray.write 0 mem (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32)) x0) (keccakWord (⟨0⟩ : UInt256) (UInt256.ofNat 64) ((UInt256.ofNat 3).toByteArray.write 0 (x0.toByteArray.write 0 ((UInt256.ofNat 2).toByteArray.write 0 (x1.toByteArray.write 0 mem (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32) (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32)) (UInt256.ofNat ee.header.timestamp)).get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 1) (⟨0⟩ : UInt256))) (UInt256.ofNat 1766847064778384329583297500742918515827483896875618958121606201292619775)) :: (UInt256.ofNat 1537) :: (⟨0⟩ : UInt256) :: (UInt256.ofNat 1) :: (UInt256.land ((sstoreAccountMap ee.codeOwner (sstoreAccountMap ee.codeOwner σ (keccakWord (⟨0⟩ : UInt256) (UInt256.ofNat 64) ((UInt256.ofNat 2).toByteArray.write 0 (x1.toByteArray.write 0 mem (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32)) x0) (keccakWord (⟨0⟩ : UInt256) (UInt256.ofNat 64) ((UInt256.ofNat 3).toByteArray.write 0 (x0.toByteArray.write 0 ((UInt256.ofNat 2).toByteArray.write 0 (x1.toByteArray.write 0 mem (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32) (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32)) (UInt256.ofNat ee.header.timestamp)).get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD (UInt256.ofNat 1) (⟨0⟩ : UInt256))) (UInt256.ofNat 1766847064778384329583297500742918515827483896875618958121606201292619775)) :: x1 :: x0 :: R)

/-- Final memory for bytecode block summary `l2tol2_block_1453`. -/
def l2tol2_block_1453_memory {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} : ByteArray :=
  ((UInt256.ofNat 3).toByteArray.write 0 (x0.toByteArray.write 0 ((UInt256.ofNat 2).toByteArray.write 0 (x1.toByteArray.write 0 mem (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32) (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1453. -/
theorem l2tol2_block_1453 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hperm : ee.perm = true)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 4411) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1453) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4411) (l2tol2_block_1453_stack (ee := ee) (mem := mem) (σ := σ) (x0 := x0) (x1 := x1) (R := R)) (l2tol2_block_1453_memory (mem := mem) (x0 := x0) (x1 := x1)) (M (M (M (M (M (M aw (⟨0⟩ : UInt256) (⟨32⟩ : UInt256)) (UInt256.ofNat 32) (⟨32⟩ : UInt256)) (⟨0⟩ : UInt256) (UInt256.ofNat 64)) (⟨0⟩ : UInt256) (⟨32⟩ : UInt256)) (UInt256.ofNat 32) (⟨32⟩ : UInt256)) (⟨0⟩ : UInt256) (UInt256.ofNat 64)) rdata (sstoreAccountMap ee.codeOwner (sstoreAccountMap ee.codeOwner σ (keccakWord (⟨0⟩ : UInt256) (UInt256.ofNat 64) ((UInt256.ofNat 2).toByteArray.write 0 (x1.toByteArray.write 0 mem (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32)) x0) (keccakWord (⟨0⟩ : UInt256) (UInt256.ofNat 64) ((UInt256.ofNat 3).toByteArray.write 0 (x0.toByteArray.write 0 ((UInt256.ofNat 2).toByteArray.write 0 (x1.toByteArray.write 0 mem (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32) (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32)) (UInt256.ofNat ee.header.timestamp)) k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push0 (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup3 (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup2 (by evm_kdecide) (by evm_ov)
  have r5 := RD.genMstore r4 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 2) (by evm_kdecide) (by evm_ov)
  have r7 := r6.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r8 := r7.swap1 (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup2 (by evm_kdecide) (by evm_ov)
  have r10 := RD.genMstore r9 (by evm_kdecide) (by evm_ov)
  have r11 := r10.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r12 := r11.dup1 (by evm_kdecide) (by evm_ov)
  have r13 := r12.dup4 (by evm_kdecide) (by evm_ov)
  have r14 := RD.genKeccak256 r13 (by evm_kdecide) (by evm_ov)
  have r15 := r14.dup5 (by evm_kdecide) (by evm_ov)
  have r16 := r15.swap1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r17⟩ := RD.sstore r16 hperm (by evm_kdecide) (by evm_ov)
  have r18 := r17.dup4 (by evm_kdecide) (by evm_ov)
  have r19 := r18.dup4 (by evm_kdecide) (by evm_ov)
  have r20 := RD.genMstore r19 (by evm_kdecide) (by evm_ov)
  have r21 := r20.push1 (UInt256.ofNat 3) (by evm_kdecide) (by evm_ov)
  have r22 := r21.swap1 (by evm_kdecide) (by evm_ov)
  have r23 := r22.swap2 (by evm_kdecide) (by evm_ov)
  have r24 := RD.genMstore r23 (by evm_kdecide) (by evm_ov)
  have r25 := r24.dup2 (by evm_kdecide) (by evm_ov)
  have r26 := RD.genKeccak256 r25 (by evm_kdecide) (by evm_ov)
  have r27 := r26.timestamp (by evm_kdecide) (by evm_ov)
  have r28 := r27.swap1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r29⟩ := RD.sstore r28 hperm (by evm_kdecide) (by evm_ov)
  have r30 := r29.push1 (UInt256.ofNat 1) (by evm_kdecide) (by evm_ov)
  have r31 := r30.dup1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r32⟩ := RD.sload r31 (by evm_kdecide) (by evm_ov)
  have r33 := r32.swap3 (by evm_kdecide) (by evm_ov)
  have r34 := r33.swap5 (by evm_kdecide) (by evm_ov)
  have r35 := r34.pop (by evm_kdecide) (by evm_ov)
  have r36 := r35.pushConst (UInt256.ofNat 1766847064778384329583297500742918515827483896875618958121606201292619775) (width := 30) (op := .PUSH30) (by decide) (by evm_kdecide) (by evm_ov)
  have r37 := r36.swap1 (by evm_kdecide) (by evm_ov)
  have r38 := r37.swap3 (by evm_kdecide) (by evm_ov)
  have r39 := r38.and (by evm_kdecide) (by evm_ov)
  have r40 := r39.swap2 (by evm_kdecide) (by evm_ov)
  have r41 := r40.swap1 (by evm_kdecide) (by evm_ov)
  have r42 := r41.push2 (UInt256.ofNat 1537) (by evm_kdecide) (by evm_ov)
  have r43 := r42.dup4 (by evm_kdecide) (by evm_ov)
  have r44 := r43.push2 (UInt256.ofNat 4411) (by evm_kdecide) (by evm_ov)
  have r45 := r44.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 4411)) r45 (by evm_kdecide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1453_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 8 ≤ 1024)
    (hperm : ee.perm = true)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 4411) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1453) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4411) (l2tol2_block_1453_stack (ee := ee) (mem := mem) (σ := σ) (x0 := x0) (x1 := x1) (R := R)) (l2tol2_block_1453_memory (mem := mem) (x0 := x0) (x1 := x1)) aw' rdata (sstoreAccountMap ee.codeOwner (sstoreAccountMap ee.codeOwner σ (keccakWord (⟨0⟩ : UInt256) (UInt256.ofNat 64) ((UInt256.ofNat 2).toByteArray.write 0 (x1.toByteArray.write 0 mem (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32)) x0) (keccakWord (⟨0⟩ : UInt256) (UInt256.ofNat 64) ((UInt256.ofNat 3).toByteArray.write 0 (x0.toByteArray.write 0 ((UInt256.ofNat 2).toByteArray.write 0 (x1.toByteArray.write 0 mem (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32) (⟨0⟩ : UInt256).toNat 32) (UInt256.ofNat 32).toNat 32)) (UInt256.ofNat ee.header.timestamp)) k' C' := by
  obtain ⟨k0, C0, h0⟩ := l2tol2_block_1453 hstack hperm hvalid h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_1537`. -/
def l2tol2_block_1537_stack {ee : ExecutionEnv} {mem : ByteArray} {x4 : UInt256} {x5 : UInt256} {x6 : UInt256} {x7 : UInt256} {x8 : UInt256} {x9 : UInt256} {R : List UInt256} : List UInt256 :=
  ((memLoad (UInt256.ofNat 64) mem) :: x6 :: x7 :: (UInt256.ofNat ee.source.val) :: (UInt256.ofNat 1696) :: (UInt256.ofNat 25393192778880726393412780032357160991586898263777241801633272823902746309408) :: x9 :: (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) x8) :: x4 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R)

/-- Automatically generated RD summary for bytecode block at pc 1537. -/
theorem l2tol2_block_1537 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 16 ≤ 1024)
    (hperm : ee.perm = true)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 4471) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1537) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : ∃ (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4471) (l2tol2_block_1537_stack (ee := ee) (mem := mem) (x4 := x4) (x5 := x5) (x6 := x6) (x7 := x7) (x8 := x8) (x9 := x9) (R := R)) mem (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata (sstoreAccountMap ee.codeOwner σ x2 (UInt256.lor (UInt256.mul (UInt256.land (UInt256.ofNat 1766847064778384329583297500742918515827483896875618958121606201292619775) x0) (UInt256.exp (UInt256.ofNat 256) x1)) (UInt256.land (UInt256.lnot (UInt256.mul (UInt256.ofNat 1766847064778384329583297500742918515827483896875618958121606201292619775) (UInt256.exp (UInt256.ofNat 256) x1))) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD x2 (⟨0⟩ : UInt256)))))) k' C' := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap2 (by evm_kdecide) (by evm_ov)
  have r3 := r2.swap1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 256) (by evm_kdecide) (by evm_ov)
  have r5 := r4.exp (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup2 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r7⟩ := RD.sload r6 (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup2 (by evm_kdecide) (by evm_ov)
  have r9 := r8.pushConst (UInt256.ofNat 1766847064778384329583297500742918515827483896875618958121606201292619775) (width := 30) (op := .PUSH30) (by decide) (by evm_kdecide) (by evm_ov)
  have r10 := r9.mul (by evm_kdecide) (by evm_ov)
  have r11 := r10.not (by evm_kdecide) (by evm_ov)
  have r12 := r11.and (by evm_kdecide) (by evm_ov)
  have r13 := r12.swap1 (by evm_kdecide) (by evm_ov)
  have r14 := r13.dup4 (by evm_kdecide) (by evm_ov)
  have r15 := r14.pushConst (UInt256.ofNat 1766847064778384329583297500742918515827483896875618958121606201292619775) (width := 30) (op := .PUSH30) (by decide) (by evm_kdecide) (by evm_ov)
  have r16 := r15.and (by evm_kdecide) (by evm_ov)
  have r17 := r16.mul (by evm_kdecide) (by evm_ov)
  have r18 := r17.or (by evm_kdecide) (by evm_ov)
  have r19 := r18.swap1 (by evm_kdecide) (by evm_ov)
  obtain ⟨_, _, r20⟩ := RD.sstore r19 hperm (by evm_kdecide) (by evm_ov)
  have r21 := r20.pop (by evm_kdecide) (by evm_ov)
  have r22 := r21.pop (by evm_kdecide) (by evm_ov)
  have r23 := r22.dup1 (by evm_kdecide) (by evm_ov)
  have r24 := r23.dup6 (by evm_kdecide) (by evm_ov)
  have r25 := r24.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r26 := r25.and (by evm_kdecide) (by evm_ov)
  have r27 := r26.dup8 (by evm_kdecide) (by evm_ov)
  have r28 := r27.pushConst (UInt256.ofNat 25393192778880726393412780032357160991586898263777241801633272823902746309408) (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
  have r29 := r28.caller (by evm_kdecide) (by evm_ov)
  have r30 := r29.dup9 (by evm_kdecide) (by evm_ov)
  have r31 := r30.dup9 (by evm_kdecide) (by evm_ov)
  have r32 := r31.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r33 := RD.genMload r32 (by evm_kdecide) (by evm_ov)
  have r34 := r33.push2 (UInt256.ofNat 1696) (by evm_kdecide) (by evm_ov)
  have r35 := r34.swap4 (by evm_kdecide) (by evm_ov)
  have r36 := r35.swap3 (by evm_kdecide) (by evm_ov)
  have r37 := r36.swap2 (by evm_kdecide) (by evm_ov)
  have r38 := r37.swap1 (by evm_kdecide) (by evm_ov)
  have r39 := r38.push2 (UInt256.ofNat 4471) (by evm_kdecide) (by evm_ov)
  have r40 := r39.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 4471)) r40 (by evm_kdecide)
  exact ⟨_, _, rFinal⟩

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1537_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 : UInt256} {R : List UInt256}
    (hstack : R.length + 16 ≤ 1024)
    (hperm : ee.perm = true)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 4471) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1537) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4471) (l2tol2_block_1537_stack (ee := ee) (mem := mem) (x4 := x4) (x5 := x5) (x6 := x6) (x7 := x7) (x8 := x8) (x9 := x9) (R := R)) mem aw' rdata (sstoreAccountMap ee.codeOwner σ x2 (UInt256.lor (UInt256.mul (UInt256.land (UInt256.ofNat 1766847064778384329583297500742918515827483896875618958121606201292619775) x0) (UInt256.exp (UInt256.ofNat 256) x1)) (UInt256.land (UInt256.lnot (UInt256.mul (UInt256.ofNat 1766847064778384329583297500742918515827483896875618958121606201292619775) (UInt256.exp (UInt256.ofNat 256) x1))) (σ.get? ee.codeOwner |>.option (⟨0⟩ : UInt256) (fun ac => ac.storage.getD x2 (⟨0⟩ : UInt256)))))) k' C' := by
  obtain ⟨k0, C0, h0⟩ := l2tol2_block_1537 hstack hperm hvalid h
  obtain ⟨k', C', h'⟩ := RD.pack h0
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_1696`. -/
def l2tol2_block_1696_stack {x6 : UInt256} {R : List UInt256} : List UInt256 :=
  (x6 :: R)

/-- Automatically generated RD summary for bytecode block at pc 1696. -/
theorem l2tol2_block_1696 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 x10 x11 : UInt256} {R : List UInt256}
    (hstack : R.length + 14 ≤ 1024)
    (hperm : ee.perm = true)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x11 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1696) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: x10 :: x11 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 x11 (l2tol2_block_1696_stack (x6 := x6) (R := R)) mem (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem))) rdata σ (k + 16) (C + ((43) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem))) + (375 + 8 * (UInt256.sub x0 (memLoad (UInt256.ofNat 64) mem)).toNat + 4 * 375))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r3 := RD.genMload r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup1 (by evm_kdecide) (by evm_ov)
  have r5 := r4.swap2 (by evm_kdecide) (by evm_ov)
  have r6 := r5.sub (by evm_kdecide) (by evm_ov)
  have r7 := r6.swap1 (by evm_kdecide) (by evm_ov)
  have r8 := RD.genLog4 r7 (by evm_kdecide) hperm (by evm_ov)
  have r9 := r8.pop (by evm_kdecide) (by evm_ov)
  have r10 := r9.swap5 (by evm_kdecide) (by evm_ov)
  have r11 := r10.swap4 (by evm_kdecide) (by evm_ov)
  have r12 := r11.pop (by evm_kdecide) (by evm_ov)
  have r13 := r12.pop (by evm_kdecide) (by evm_ov)
  have r14 := r13.pop (by evm_kdecide) (by evm_ov)
  have r15 := r14.pop (by evm_kdecide) (by evm_ov)
  have r16 := r15.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r16 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1696_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 x9 x10 x11 : UInt256} {R : List UInt256}
    (hstack : R.length + 14 ≤ 1024)
    (hperm : ee.perm = true)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains x11 = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1696) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: x9 :: x10 :: x11 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 x11 (l2tol2_block_1696_stack (x6 := x6) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_1696 hstack hperm hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_1713_taken`. -/
def l2tol2_block_1713_taken_stack {ee : ExecutionEnv} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero (UInt256.eq (UInt256.ofNat 376793390874373408599387495934666716005045108743) (UInt256.ofNat ee.source.val))) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1713. -/
theorem l2tol2_block_1713_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.eq (UInt256.ofNat 376793390874373408599387495934666716005045108743) (UInt256.ofNat ee.source.val))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 2048) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1713) R mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2048) (l2tol2_block_1713_taken_stack (ee := ee) (R := R)) mem aw rdata σ (k + 8) (C + ((28))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.caller (by evm_kdecide) (by evm_ov)
  have r3 := r2.push20 (UInt256.ofNat 376793390874373408599387495934666716005045108743) (by evm_kdecide) (by evm_ov)
  have r4 := r3.eq (by evm_kdecide) (by evm_ov)
  have r5 := r4.iszero (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup1 (by evm_kdecide) (by evm_ov)
  have r7 := r6.push2 (UInt256.ofNat 2048) (by evm_kdecide) (by evm_ov)
  have r8 := r7.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2048)) r8 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1713_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.eq (UInt256.ofNat 376793390874373408599387495934666716005045108743) (UInt256.ofNat ee.source.val))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 2048) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1713) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 2048) (l2tol2_block_1713_taken_stack (ee := ee) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_1713_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_1713_fallthrough`. -/
def l2tol2_block_1713_fallthrough_stack {ee : ExecutionEnv} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero (UInt256.eq (UInt256.ofNat 376793390874373408599387495934666716005045108743) (UInt256.ofNat ee.source.val))) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1713. -/
theorem l2tol2_block_1713_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.eq (UInt256.ofNat 376793390874373408599387495934666716005045108743) (UInt256.ofNat ee.source.val))) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1713) R mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1743) (l2tol2_block_1713_fallthrough_stack (ee := ee) (R := R)) mem aw rdata σ (k + 8) (C + ((28))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.caller (by evm_kdecide) (by evm_ov)
  have r3 := r2.push20 (UInt256.ofNat 376793390874373408599387495934666716005045108743) (by evm_kdecide) (by evm_ov)
  have r4 := r3.eq (by evm_kdecide) (by evm_ov)
  have r5 := r4.iszero (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup1 (by evm_kdecide) (by evm_ov)
  have r7 := r6.push2 (UInt256.ofNat 2048) (by evm_kdecide) (by evm_ov)
  have r8 := r7.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1743)) r8 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1713_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.eq (UInt256.ofNat 376793390874373408599387495934666716005045108743) (UInt256.ofNat ee.source.val))) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1713) R mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1743) (l2tol2_block_1713_fallthrough_stack (ee := ee) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_1713_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_1743`. -/
def l2tol2_block_1743_stack {g : Sat256} {mem : ByteArray} {aw : UInt256} {C : ℕ} {R : List UInt256} : List UInt256 :=
  (((g.subNat (C + ((71) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256))) + 2)).toUInt256) :: (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (UInt256.ofNat 376793390874373408599387495934666716005045108743)) :: (memLoad (UInt256.ofNat 64) ((UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 3679477120)) (UInt256.ofNat 224)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)) :: (UInt256.sub ((UInt256.ofNat 4) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (UInt256.ofNat 64) ((UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 3679477120)) (UInt256.ofNat 224)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32))) :: (memLoad (UInt256.ofNat 64) ((UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 3679477120)) (UInt256.ofNat 224)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)) :: (UInt256.ofNat 32) :: ((UInt256.ofNat 4) + (memLoad (UInt256.ofNat 64) mem)) :: (UInt256.ofNat 3679477120) :: (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (UInt256.ofNat 376793390874373408599387495934666716005045108743)) :: R)

/-- Final memory for bytecode block summary `l2tol2_block_1743`. -/
def l2tol2_block_1743_memory {mem : ByteArray} : ByteArray :=
  ((UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 3679477120)) (UInt256.ofNat 224)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1743. -/
theorem l2tol2_block_1743 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1743) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1821) (l2tol2_block_1743_stack (g := g) (mem := mem) (aw := aw) (C := C) (R := R)) (l2tol2_block_1743_memory (mem := mem)) (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 25) (C + ((73) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.pop (by evm_kdecide) (by evm_ov)
  have r2 := r1.push20 (UInt256.ofNat 376793390874373408599387495934666716005045108743) (by evm_kdecide) (by evm_ov)
  have r3 := r2.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r4 := r3.and (by evm_kdecide) (by evm_ov)
  have r5 := r4.push4 (UInt256.ofNat 3679477120) (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r7 := RD.genMload r6 (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup2 (by evm_kdecide) (by evm_ov)
  have r9 := r8.push4 (UInt256.ofNat 4294967295) (by evm_kdecide) (by evm_ov)
  have r10 := r9.and (by evm_kdecide) (by evm_ov)
  have r11 := r10.push1 (UInt256.ofNat 224) (by evm_kdecide) (by evm_ov)
  have r12 := r11.shl (by evm_kdecide) (by evm_ov)
  have r13 := r12.dup2 (by evm_kdecide) (by evm_ov)
  have r14 := RD.genMstore r13 (by evm_kdecide) (by evm_ov)
  have r15 := r14.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r16 := r15.add (by evm_kdecide) (by evm_ov)
  have r17 := r16.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r18 := r17.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r19 := RD.genMload r18 (by evm_kdecide) (by evm_ov)
  have r20 := r19.dup1 (by evm_kdecide) (by evm_ov)
  have r21 := r20.dup4 (by evm_kdecide) (by evm_ov)
  have r22 := r21.sub (by evm_kdecide) (by evm_ov)
  have r23 := r22.dup2 (by evm_kdecide) (by evm_ov)
  have r24 := r23.dup7 (by evm_kdecide) (by evm_ov)
  have r25 := RD.genGas (RD.normalizeCounters (k' := k + 24) (C' := C + ((71) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) r24 (by omega) (by omega)) (by evm_kdecide) (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1821)) r25 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1743_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1743) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1821) (l2tol2_block_1743_stack (g := g) (mem := mem) (aw := aw) (C := C) (R := R)) (l2tol2_block_1743_memory (mem := mem)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_1743 hstack h)
  exact ⟨_, k', C', h'⟩

/- Unsupported instruction boundary at pc 1821: staticcall (0xfa). No RD transition is asserted. Summaries resume at pc 1822 from a fresh symbolic RD state. -/

/-- Final stack for bytecode block summary `l2tol2_block_1822_taken`. -/
def l2tol2_block_1822_taken_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1822. -/
theorem l2tol2_block_1822_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 1836) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1822) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1836) (l2tol2_block_1822_taken_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.iszero (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.iszero (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 1836) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1836)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1822_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 1836) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1822) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1836) (l2tol2_block_1822_taken_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_1822_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_1822_fallthrough`. -/
def l2tol2_block_1822_fallthrough_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1822. -/
theorem l2tol2_block_1822_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1822) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1829) (l2tol2_block_1822_fallthrough_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.iszero (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.iszero (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 1836) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1829)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1822_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) = (UInt256.ofNat 0))
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1822) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1829) (l2tol2_block_1822_fallthrough_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_1822_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 1829. -/
theorem l2tol2_block_1829 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1829) R mem aw rdata σ k C)
    : RDrev ExpiryEvm.l2tol2Runtime g s0 := by
  let r0 := h
  have r1 := r0.returndatasize (by evm_kdecide) (by evm_ov)
  have r2 := r1.push0 (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := RD.genReturndatacopy r3 (by evm_kdecide) (by
    have hz : UInt256.ofNat 0 = (⟨0⟩ : UInt256) := by decide
    simpa only [hz] using returnDataCopyFullGuard rdata) (by evm_ov)
  have r5 := r4.returndatasize (by evm_kdecide) (by evm_ov)
  have r6 := r5.push0 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r6 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `l2tol2_block_1836`. -/
def l2tol2_block_1836_stack {mem : ByteArray} {rdata : ByteArray} {R : List UInt256} : List UInt256 :=
  ((memLoad (UInt256.ofNat 64) mem) :: ((memLoad (UInt256.ofNat 64) mem) + (UInt256.ofNat rdata.size)) :: (UInt256.ofNat 1872) :: R)

/-- Final memory for bytecode block summary `l2tol2_block_1836`. -/
def l2tol2_block_1836_memory {mem : ByteArray} {rdata : ByteArray} : ByteArray :=
  (((memLoad (UInt256.ofNat 64) mem) + (UInt256.land ((UInt256.ofNat rdata.size) + (UInt256.ofNat 31)) (UInt256.lnot (UInt256.ofNat 31)))).toByteArray.write 0 mem (UInt256.ofNat 64).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1836. -/
theorem l2tol2_block_1836 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 4576) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1836) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4576) (l2tol2_block_1836_stack (mem := mem) (rdata := rdata) (R := R)) (l2tol2_block_1836_memory (mem := mem) (rdata := rdata)) (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 28) (C + ((81) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
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
  have r24 := r23.push2 (UInt256.ofNat 1872) (by evm_kdecide) (by evm_ov)
  have r25 := r24.swap2 (by evm_kdecide) (by evm_ov)
  have r26 := r25.swap1 (by evm_kdecide) (by evm_ov)
  have r27 := r26.push2 (UInt256.ofNat 4576) (by evm_kdecide) (by evm_ov)
  have r28 := r27.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 4576)) r28 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1836_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 5 ≤ 1024)
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 4576) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1836) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 4576) (l2tol2_block_1836_stack (mem := mem) (rdata := rdata) (R := R)) (l2tol2_block_1836_memory (mem := mem) (rdata := rdata)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_1836 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `l2tol2_block_1872`. -/
def l2tol2_block_1872_stack {g : Sat256} {mem : ByteArray} {aw : UInt256} {C : ℕ} {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  (((g.subNat (C + ((76) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256))) + 2)).toUInt256) :: (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (UInt256.ofNat 376793390874373408599387495934666716005045108743)) :: (memLoad (UInt256.ofNat 64) ((UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 1848208965)) (UInt256.ofNat 224)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)) :: (UInt256.sub ((UInt256.ofNat 4) + (memLoad (UInt256.ofNat 64) mem)) (memLoad (UInt256.ofNat 64) ((UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 1848208965)) (UInt256.ofNat 224)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32))) :: (memLoad (UInt256.ofNat 64) ((UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 1848208965)) (UInt256.ofNat 224)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)) :: (UInt256.ofNat 32) :: ((UInt256.ofNat 4) + (memLoad (UInt256.ofNat 64) mem)) :: (UInt256.ofNat 1848208965) :: (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (UInt256.ofNat 376793390874373408599387495934666716005045108743)) :: (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) x0) :: R)

/-- Final memory for bytecode block summary `l2tol2_block_1872`. -/
def l2tol2_block_1872_memory {mem : ByteArray} : ByteArray :=
  ((UInt256.shiftLeft (UInt256.land (UInt256.ofNat 4294967295) (UInt256.ofNat 1848208965)) (UInt256.ofNat 224)).toByteArray.write 0 mem (memLoad (UInt256.ofNat 64) mem).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 1872. -/
theorem l2tol2_block_1872 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1872) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1972) (l2tol2_block_1872_stack (g := g) (mem := mem) (aw := aw) (C := C) (x0 := x0) (R := R)) (l2tol2_block_1872_memory (mem := mem)) (M (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)) rdata σ (k + 27) (C + ((78) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r3 := r2.and (by evm_kdecide) (by evm_ov)
  have r4 := r3.push20 (UInt256.ofNat 376793390874373408599387495934666716005045108743) (by evm_kdecide) (by evm_ov)
  have r5 := r4.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r6 := r5.and (by evm_kdecide) (by evm_ov)
  have r7 := r6.push4 (UInt256.ofNat 1848208965) (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r9 := RD.genMload r8 (by evm_kdecide) (by evm_ov)
  have r10 := r9.dup2 (by evm_kdecide) (by evm_ov)
  have r11 := r10.push4 (UInt256.ofNat 4294967295) (by evm_kdecide) (by evm_ov)
  have r12 := r11.and (by evm_kdecide) (by evm_ov)
  have r13 := r12.push1 (UInt256.ofNat 224) (by evm_kdecide) (by evm_ov)
  have r14 := r13.shl (by evm_kdecide) (by evm_ov)
  have r15 := r14.dup2 (by evm_kdecide) (by evm_ov)
  have r16 := RD.genMstore r15 (by evm_kdecide) (by evm_ov)
  have r17 := r16.push1 (UInt256.ofNat 4) (by evm_kdecide) (by evm_ov)
  have r18 := r17.add (by evm_kdecide) (by evm_ov)
  have r19 := r18.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r20 := r19.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r21 := RD.genMload r20 (by evm_kdecide) (by evm_ov)
  have r22 := r21.dup1 (by evm_kdecide) (by evm_ov)
  have r23 := r22.dup4 (by evm_kdecide) (by evm_ov)
  have r24 := r23.sub (by evm_kdecide) (by evm_ov)
  have r25 := r24.dup2 (by evm_kdecide) (by evm_ov)
  have r26 := r25.dup7 (by evm_kdecide) (by evm_ov)
  have r27 := RD.genGas (RD.normalizeCounters (k' := k + 26) (C' := C + ((76) + (memExpansionCost aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) + (memExpansionCost (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw (UInt256.ofNat 64) (⟨32⟩ : UInt256)) (memLoad (UInt256.ofNat 64) mem) (⟨32⟩ : UInt256)) (UInt256.ofNat 64) (⟨32⟩ : UInt256)))) r26 (by omega) (by omega)) (by evm_kdecide) (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1972)) r27 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1872_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1872) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1972) (l2tol2_block_1872_stack (g := g) (mem := mem) (aw := aw) (C := C) (x0 := x0) (R := R)) (l2tol2_block_1872_memory (mem := mem)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_1872 hstack h)
  exact ⟨_, k', C', h'⟩

/- Unsupported instruction boundary at pc 1972: staticcall (0xfa). No RD transition is asserted. Summaries resume at pc 1973 from a fresh symbolic RD state. -/

/-- Final stack for bytecode block summary `l2tol2_block_1973_taken`. -/
def l2tol2_block_1973_taken_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.isZero x0) :: R)

/-- Automatically generated RD summary for bytecode block at pc 1973. -/
theorem l2tol2_block_1973_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 1987) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1973) (x0 :: R) mem aw rdata σ k C)
    : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1987) (l2tol2_block_1973_taken_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 5) (C + ((22))) := by
  let r0 := h
  have r1 := r0.iszero (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  have r3 := r2.iszero (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 1987) (by evm_kdecide) (by evm_ov)
  have r5 := r4.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 1987)) r5 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem l2tol2_block_1973_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 : UInt256} {R : List UInt256}
    (hstack : R.length + 3 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.isZero x0)) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J ExpiryEvm.l2tol2Runtime 0).contains (UInt256.ofNat 1987) = true)
    (h : RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1973) (x0 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD ExpiryEvm.l2tol2Runtime ee g s0 (UInt256.ofNat 1987) (l2tol2_block_1973_taken_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (l2tol2_block_1973_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

end l2tol2Blocks
