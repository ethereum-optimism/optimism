import Reasoning.Reach
import BridgeEvm.Bytecode

open Solm ABI Ethereum Ethereum.EVM
open Reasoning.Theory Reasoning.Reach

namespace ethbridgeBlocks

/-- Final stack for bytecode block summary `ethbridge_block_2668_taken`. -/
def ethbridge_block_2668_taken_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 2668. -/
theorem ethbridge_block_2668_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 160))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2692) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2668) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2692) (ethbridge_block_2668_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 14) (C + ((47))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r7 := r6.push1 (UInt256.ofNat 160) (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup7 (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup9 (by evm_kdecide) (by evm_ov)
  have r10 := r9.sub (by evm_kdecide) (by evm_ov)
  have r11 := r10.slt (by evm_kdecide) (by evm_ov)
  have r12 := r11.iszero (by evm_kdecide) (by evm_ov)
  have r13 := r12.push2 (UInt256.ofNat 2692) (by evm_kdecide) (by evm_ov)
  have r14 := r13.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2692)) r14 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2668_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 160))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2692) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2668) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2692) (ethbridge_block_2668_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2668_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_2668_fallthrough`. -/
def ethbridge_block_2668_fallthrough_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: (UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 2668. -/
theorem ethbridge_block_2668_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 160))) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2668) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2688) (ethbridge_block_2668_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 14) (C + ((47))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r7 := r6.push1 (UInt256.ofNat 160) (by evm_kdecide) (by evm_ov)
  have r8 := r7.dup7 (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup9 (by evm_kdecide) (by evm_ov)
  have r10 := r9.sub (by evm_kdecide) (by evm_ov)
  have r11 := r10.slt (by evm_kdecide) (by evm_ov)
  have r12 := r11.iszero (by evm_kdecide) (by evm_ov)
  have r13 := r12.push2 (UInt256.ofNat 2692) (by evm_kdecide) (by evm_ov)
  have r14 := r13.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2688)) r14 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2668_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 160))) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2668) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2688) (ethbridge_block_2668_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2668_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 2688. -/
theorem ethbridge_block_2688 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2688) R mem aw rdata σ k C)
    : RDrev BridgeEvm.ethbridgeRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `ethbridge_block_2692`. -/
def ethbridge_block_2692_stack {ee : ExecutionEnv} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x5 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes (x5 + (UInt256.ofNat 64)).toNat 32)) :: (UInt256.ofNat 2717) :: (uInt256OfByteArray (ee.calldata.readBytes (x5 + (UInt256.ofNat 64)).toNat 32)) :: x0 :: x1 :: x2 :: (uInt256OfByteArray (ee.calldata.readBytes (x5 + (UInt256.ofNat 32)).toNat 32)) :: (uInt256OfByteArray (ee.calldata.readBytes x5.toNat 32)) :: x5 :: R)

/-- Automatically generated RD summary for bytecode block at pc 2692. -/
theorem ethbridge_block_2692 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2389) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2692) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2389) (ethbridge_block_2692_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x5 := x5) (R := R)) mem aw rdata σ (k + 19) (C + ((58))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup6 (by evm_kdecide) (by evm_ov)
  have r3 := r2.calldataload (by evm_kdecide) (by evm_ov)
  have r4 := r3.swap5 (by evm_kdecide) (by evm_ov)
  have r5 := r4.pop (by evm_kdecide) (by evm_ov)
  have r6 := r5.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r7 := r6.dup7 (by evm_kdecide) (by evm_ov)
  have r8 := r7.add (by evm_kdecide) (by evm_ov)
  have r9 := r8.calldataload (by evm_kdecide) (by evm_ov)
  have r10 := r9.swap4 (by evm_kdecide) (by evm_ov)
  have r11 := r10.pop (by evm_kdecide) (by evm_ov)
  have r12 := r11.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r13 := r12.dup7 (by evm_kdecide) (by evm_ov)
  have r14 := r13.add (by evm_kdecide) (by evm_ov)
  have r15 := r14.calldataload (by evm_kdecide) (by evm_ov)
  have r16 := r15.push2 (UInt256.ofNat 2717) (by evm_kdecide) (by evm_ov)
  have r17 := r16.dup2 (by evm_kdecide) (by evm_ov)
  have r18 := r17.push2 (UInt256.ofNat 2389) (by evm_kdecide) (by evm_ov)
  have r19 := r18.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2389)) r19 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2692_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2389) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2692) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2389) (ethbridge_block_2692_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x5 := x5) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2692 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_2717`. -/
def ethbridge_block_2717_stack {ee : ExecutionEnv} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x4 : UInt256} {x5 : UInt256} {x6 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes (x6 + (UInt256.ofNat 96)).toNat 32)) :: (UInt256.ofNat 2733) :: (uInt256OfByteArray (ee.calldata.readBytes (x6 + (UInt256.ofNat 96)).toNat 32)) :: x1 :: x2 :: x0 :: x4 :: x5 :: x6 :: R)

/-- Automatically generated RD summary for bytecode block at pc 2717. -/
theorem ethbridge_block_2717 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2389) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2717) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2389) (ethbridge_block_2717_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x4 := x4) (x5 := x5) (x6 := x6) (R := R)) mem aw rdata σ (k + 11) (C + ((35))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap3 (by evm_kdecide) (by evm_ov)
  have r3 := r2.pop (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 96) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup7 (by evm_kdecide) (by evm_ov)
  have r6 := r5.add (by evm_kdecide) (by evm_ov)
  have r7 := r6.calldataload (by evm_kdecide) (by evm_ov)
  have r8 := r7.push2 (UInt256.ofNat 2733) (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup2 (by evm_kdecide) (by evm_ov)
  have r10 := r9.push2 (UInt256.ofNat 2389) (by evm_kdecide) (by evm_ov)
  have r11 := r10.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2389)) r11 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2717_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 : UInt256} {R : List UInt256}
    (hstack : R.length + 10 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2389) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2717) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2389) (ethbridge_block_2717_stack (ee := ee) (x0 := x0) (x1 := x1) (x2 := x2) (x4 := x4) (x5 := x5) (x6 := x6) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2717 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_2733`. -/
def ethbridge_block_2733_stack {ee : ExecutionEnv} {x0 : UInt256} {x3 : UInt256} {x4 : UInt256} {x5 : UInt256} {x6 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes ((UInt256.ofNat 128) + x6).toNat 32)) :: x0 :: x3 :: x4 :: x5 :: R)

/-- Automatically generated RD summary for bytecode block at pc 2733. -/
theorem ethbridge_block_2733 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains x8 = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2733) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 x8 (ethbridge_block_2733_stack (ee := ee) (x0 := x0) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (R := R)) mem aw rdata σ (k + 16) (C + ((48))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap5 (by evm_kdecide) (by evm_ov)
  have r3 := r2.swap8 (by evm_kdecide) (by evm_ov)
  have r4 := r3.swap4 (by evm_kdecide) (by evm_ov)
  have r5 := r4.swap7 (by evm_kdecide) (by evm_ov)
  have r6 := r5.pop (by evm_kdecide) (by evm_ov)
  have r7 := r6.swap2 (by evm_kdecide) (by evm_ov)
  have r8 := r7.swap5 (by evm_kdecide) (by evm_ov)
  have r9 := r8.push1 (UInt256.ofNat 128) (by evm_kdecide) (by evm_ov)
  have r10 := r9.add (by evm_kdecide) (by evm_ov)
  have r11 := r10.calldataload (by evm_kdecide) (by evm_ov)
  have r12 := r11.swap3 (by evm_kdecide) (by evm_ov)
  have r13 := r12.swap2 (by evm_kdecide) (by evm_ov)
  have r14 := r13.pop (by evm_kdecide) (by evm_ov)
  have r15 := r14.pop (by evm_kdecide) (by evm_ov)
  have r16 := r15.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r16 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2733_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 x7 x8 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains x8 = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2733) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: x7 :: x8 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 x8 (ethbridge_block_2733_stack (ee := ee) (x0 := x0) (x3 := x3) (x4 := x4) (x5 := x5) (x6 := x6) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2733 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_2750_taken`. -/
def ethbridge_block_2750_taken_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 2750. -/
theorem ethbridge_block_2750_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 32))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2768) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2750) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2768) (ethbridge_block_2750_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 10) (C + ((35))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup5 (by evm_kdecide) (by evm_ov)
  have r6 := r5.sub (by evm_kdecide) (by evm_ov)
  have r7 := r6.slt (by evm_kdecide) (by evm_ov)
  have r8 := r7.iszero (by evm_kdecide) (by evm_ov)
  have r9 := r8.push2 (UInt256.ofNat 2768) (by evm_kdecide) (by evm_ov)
  have r10 := r9.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2768)) r10 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2750_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 32))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2768) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2750) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2768) (ethbridge_block_2750_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2750_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_2750_fallthrough`. -/
def ethbridge_block_2750_fallthrough_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 2750. -/
theorem ethbridge_block_2750_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 32))) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2750) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2764) (ethbridge_block_2750_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 10) (C + ((35))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup5 (by evm_kdecide) (by evm_ov)
  have r6 := r5.sub (by evm_kdecide) (by evm_ov)
  have r7 := r6.slt (by evm_kdecide) (by evm_ov)
  have r8 := r7.iszero (by evm_kdecide) (by evm_ov)
  have r9 := r8.push2 (UInt256.ofNat 2768) (by evm_kdecide) (by evm_ov)
  have r10 := r9.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2764)) r10 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2750_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 32))) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2750) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2764) (ethbridge_block_2750_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2750_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 2764. -/
theorem ethbridge_block_2764 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2764) R mem aw rdata σ k C)
    : RDrev BridgeEvm.ethbridgeRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `ethbridge_block_2768`. -/
def ethbridge_block_2768_stack {ee : ExecutionEnv} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((uInt256OfByteArray (ee.calldata.readBytes x1.toNat 32)) :: R)

/-- Automatically generated RD summary for bytecode block at pc 2768. -/
theorem ethbridge_block_2768 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains x3 = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2768) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 x3 (ethbridge_block_2768_stack (ee := ee) (x1 := x1) (R := R)) mem aw rdata σ (k + 7) (C + ((22))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.pop (by evm_kdecide) (by evm_ov)
  have r3 := r2.calldataload (by evm_kdecide) (by evm_ov)
  have r4 := r3.swap2 (by evm_kdecide) (by evm_ov)
  have r5 := r4.swap1 (by evm_kdecide) (by evm_ov)
  have r6 := r5.pop (by evm_kdecide) (by evm_ov)
  have r7 := r6.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r7 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2768_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 4 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains x3 = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2768) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 x3 (ethbridge_block_2768_stack (ee := ee) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2768 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_2775_taken`. -/
def ethbridge_block_2775_taken_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 0) :: (UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 2775. -/
theorem ethbridge_block_2775_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 64))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2794) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2775) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2794) (ethbridge_block_2775_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 11) (C + ((38))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup4 (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup6 (by evm_kdecide) (by evm_ov)
  have r7 := r6.sub (by evm_kdecide) (by evm_ov)
  have r8 := r7.slt (by evm_kdecide) (by evm_ov)
  have r9 := r8.iszero (by evm_kdecide) (by evm_ov)
  have r10 := r9.push2 (UInt256.ofNat 2794) (by evm_kdecide) (by evm_ov)
  have r11 := r10.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2794)) r11 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2775_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 64))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2794) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2775) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2794) (ethbridge_block_2775_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2775_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_2775_fallthrough`. -/
def ethbridge_block_2775_fallthrough_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 0) :: (UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 2775. -/
theorem ethbridge_block_2775_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 64))) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2775) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2790) (ethbridge_block_2775_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 11) (C + ((38))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup1 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup4 (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup6 (by evm_kdecide) (by evm_ov)
  have r7 := r6.sub (by evm_kdecide) (by evm_ov)
  have r8 := r7.slt (by evm_kdecide) (by evm_ov)
  have r9 := r8.iszero (by evm_kdecide) (by evm_ov)
  have r10 := r9.push2 (UInt256.ofNat 2794) (by evm_kdecide) (by evm_ov)
  have r11 := r10.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2790)) r11 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2775_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 64))) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2775) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2790) (ethbridge_block_2775_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2775_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 2790. -/
theorem ethbridge_block_2790 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2790) R mem aw rdata σ k C)
    : RDrev BridgeEvm.ethbridgeRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

/-- Final stack for bytecode block summary `ethbridge_block_2794`. -/
def ethbridge_block_2794_stack {mem : ByteArray} {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {R : List UInt256} : List UInt256 :=
  ((memLoad x2 mem) :: (UInt256.ofNat 2805) :: (memLoad x2 mem) :: x0 :: x1 :: x2 :: R)

/-- Automatically generated RD summary for bytecode block at pc 2794. -/
theorem ethbridge_block_2794 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2389) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2794) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2389) (ethbridge_block_2794_stack (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2) (R := R)) mem (M aw x2 (⟨32⟩ : UInt256)) rdata σ (k + 7) (C + ((24) + (memExpansionCost aw x2 (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup3 (by evm_kdecide) (by evm_ov)
  have r3 := RD.genMload r2 (by evm_kdecide) (by evm_ov)
  have r4 := r3.push2 (UInt256.ofNat 2805) (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup2 (by evm_kdecide) (by evm_ov)
  have r6 := r5.push2 (UInt256.ofNat 2389) (by evm_kdecide) (by evm_ov)
  have r7 := r6.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2389)) r7 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2794_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2389) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2794) (x0 :: x1 :: x2 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2389) (ethbridge_block_2794_stack (mem := mem) (x0 := x0) (x1 := x1) (x2 := x2) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2794 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_2805`. -/
def ethbridge_block_2805_stack {mem : ByteArray} {x0 : UInt256} {x3 : UInt256} {R : List UInt256} : List UInt256 :=
  ((memLoad ((UInt256.ofNat 32) + x3) mem) :: x0 :: R)

/-- Automatically generated RD summary for bytecode block at pc 2805. -/
theorem ethbridge_block_2805 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains x5 = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2805) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 x5 (ethbridge_block_2805_stack (mem := mem) (x0 := x0) (x3 := x3) (R := R)) mem (M aw ((UInt256.ofNat 32) + x3) (⟨32⟩ : UInt256)) rdata σ (k + 15) (C + ((45) + (memExpansionCost aw ((UInt256.ofNat 32) + x3) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r3 := r2.swap4 (by evm_kdecide) (by evm_ov)
  have r4 := r3.swap1 (by evm_kdecide) (by evm_ov)
  have r5 := r4.swap4 (by evm_kdecide) (by evm_ov)
  have r6 := r5.add (by evm_kdecide) (by evm_ov)
  have r7 := RD.genMload r6 (by evm_kdecide) (by evm_ov)
  have r8 := r7.swap3 (by evm_kdecide) (by evm_ov)
  have r9 := r8.swap5 (by evm_kdecide) (by evm_ov)
  have r10 := r9.swap3 (by evm_kdecide) (by evm_ov)
  have r11 := r10.swap4 (by evm_kdecide) (by evm_ov)
  have r12 := r11.pop (by evm_kdecide) (by evm_ov)
  have r13 := r12.pop (by evm_kdecide) (by evm_ov)
  have r14 := r13.pop (by evm_kdecide) (by evm_ov)
  have r15 := r14.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r15 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2805_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains x5 = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2805) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 x5 (ethbridge_block_2805_stack (mem := mem) (x0 := x0) (x3 := x3) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2805 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_2821`. -/
def ethbridge_block_2821_stack {x0 : UInt256} {x1 : UInt256} {x2 : UInt256} {x3 : UInt256} {R : List UInt256} : List UInt256 :=
  (x1 :: (x0 + (UInt256.ofNat 96)) :: (UInt256.ofNat 2874) :: (UInt256.ofNat 0) :: x0 :: x1 :: x2 :: x3 :: R)

/-- Final memory for bytecode block summary `ethbridge_block_2821`. -/
def ethbridge_block_2821_memory {mem : ByteArray} {x0 : UInt256} {x2 : UInt256} {x3 : UInt256} : ByteArray :=
  ((UInt256.ofNat 96).toByteArray.write 0 ((UInt256.land x2 (UInt256.ofNat 1461501637330902918203684832716283019655932542975)).toByteArray.write 0 (x3.toByteArray.write 0 mem x0.toNat 32) (x0 + (UInt256.ofNat 32)).toNat 32) (x0 + (UInt256.ofNat 64)).toNat 32)

/-- Automatically generated RD summary for bytecode block at pc 2821. -/
theorem ethbridge_block_2821 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2491) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2821) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2491) (ethbridge_block_2821_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (R := R)) (ethbridge_block_2821_memory (mem := mem) (x0 := x0) (x2 := x2) (x3 := x3)) (M (M (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)) rdata σ (k + 24) (C + ((75) + (memExpansionCost aw x0 (⟨32⟩ : UInt256)) + (memExpansionCost (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) + (memExpansionCost (M (M aw x0 (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 32)) (⟨32⟩ : UInt256)) (x0 + (UInt256.ofNat 64)) (⟨32⟩ : UInt256)))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup4 (by evm_kdecide) (by evm_ov)
  have r3 := r2.dup2 (by evm_kdecide) (by evm_ov)
  have r4 := RD.genMstore r3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.push20 (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (by evm_kdecide) (by evm_ov)
  have r6 := r5.dup4 (by evm_kdecide) (by evm_ov)
  have r7 := r6.and (by evm_kdecide) (by evm_ov)
  have r8 := r7.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r9 := r8.dup3 (by evm_kdecide) (by evm_ov)
  have r10 := r9.add (by evm_kdecide) (by evm_ov)
  have r11 := RD.genMstore r10 (by evm_kdecide) (by evm_ov)
  have r12 := r11.push1 (UInt256.ofNat 96) (by evm_kdecide) (by evm_ov)
  have r13 := r12.push1 (UInt256.ofNat 64) (by evm_kdecide) (by evm_ov)
  have r14 := r13.dup3 (by evm_kdecide) (by evm_ov)
  have r15 := r14.add (by evm_kdecide) (by evm_ov)
  have r16 := RD.genMstore r15 (by evm_kdecide) (by evm_ov)
  have r17 := r16.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r18 := r17.push2 (UInt256.ofNat 2874) (by evm_kdecide) (by evm_ov)
  have r19 := r18.push1 (UInt256.ofNat 96) (by evm_kdecide) (by evm_ov)
  have r20 := r19.dup4 (by evm_kdecide) (by evm_ov)
  have r21 := r20.add (by evm_kdecide) (by evm_ov)
  have r22 := r21.dup5 (by evm_kdecide) (by evm_ov)
  have r23 := r22.push2 (UInt256.ofNat 2491) (by evm_kdecide) (by evm_ov)
  have r24 := r23.jump (by evm_kdecide) hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2491)) r24 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2821_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 : UInt256} {R : List UInt256}
    (hstack : R.length + 9 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2491) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2821) (x0 :: x1 :: x2 :: x3 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2491) (ethbridge_block_2821_stack (x0 := x0) (x1 := x1) (x2 := x2) (x3 := x3) (R := R)) (ethbridge_block_2821_memory (mem := mem) (x0 := x0) (x2 := x2) (x3 := x3)) aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2821 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_2874`. -/
def ethbridge_block_2874_stack {x0 : UInt256} {R : List UInt256} : List UInt256 :=
  (x0 :: R)

/-- Automatically generated RD summary for bytecode block at pc 2874. -/
theorem ethbridge_block_2874 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains x6 = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2874) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 x6 (ethbridge_block_2874_stack (x0 := x0) (R := R)) mem aw rdata σ (k + 9) (C + ((25))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.swap6 (by evm_kdecide) (by evm_ov)
  have r3 := r2.swap5 (by evm_kdecide) (by evm_ov)
  have r4 := r3.pop (by evm_kdecide) (by evm_ov)
  have r5 := r4.pop (by evm_kdecide) (by evm_ov)
  have r6 := r5.pop (by evm_kdecide) (by evm_ov)
  have r7 := r6.pop (by evm_kdecide) (by evm_ov)
  have r8 := r7.pop (by evm_kdecide) (by evm_ov)
  have r9 := r8.jump (by evm_kdecide) hvalid (by evm_ov)
  exact RD.normalizeCounters r9 (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2874_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 x2 x3 x4 x5 x6 : UInt256} {R : List UInt256}
    (hstack : R.length + 7 ≤ 1024)
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains x6 = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2874) (x0 :: x1 :: x2 :: x3 :: x4 :: x5 :: x6 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 x6 (ethbridge_block_2874_stack (x0 := x0) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2874 hstack hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_2883_taken`. -/
def ethbridge_block_2883_taken_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 2883. -/
theorem ethbridge_block_2883_taken {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 32))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2901) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2883) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2901) (ethbridge_block_2883_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 10) (C + ((35))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup5 (by evm_kdecide) (by evm_ov)
  have r6 := r5.sub (by evm_kdecide) (by evm_ov)
  have r7 := r6.slt (by evm_kdecide) (by evm_ov)
  have r8 := r7.iszero (by evm_kdecide) (by evm_ov)
  have r9 := r8.push2 (UInt256.ofNat 2901) (by evm_kdecide) (by evm_ov)
  have r10 := r9.jumpiT (by evm_kdecide) hcond hvalid (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2901)) r10 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2883_taken_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 32))) ≠ (UInt256.ofNat 0))
    (hvalid : (D_J BridgeEvm.ethbridgeRuntime 0).contains (UInt256.ofNat 2901) = true)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2883) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2901) (ethbridge_block_2883_taken_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2883_taken hstack hcond hvalid h)
  exact ⟨_, k', C', h'⟩

/-- Final stack for bytecode block summary `ethbridge_block_2883_fallthrough`. -/
def ethbridge_block_2883_fallthrough_stack {x0 : UInt256} {x1 : UInt256} {R : List UInt256} : List UInt256 :=
  ((UInt256.ofNat 0) :: x0 :: x1 :: R)

/-- Automatically generated RD summary for bytecode block at pc 2883. -/
theorem ethbridge_block_2883_fallthrough {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 32))) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2883) (x0 :: x1 :: R) mem aw rdata σ k C)
    : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2897) (ethbridge_block_2883_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw rdata σ (k + 10) (C + ((35))) := by
  let r0 := h
  have r1 := r0.jumpdest (by evm_kdecide) (by evm_ov)
  have r2 := r1.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r3 := r2.push1 (UInt256.ofNat 32) (by evm_kdecide) (by evm_ov)
  have r4 := r3.dup3 (by evm_kdecide) (by evm_ov)
  have r5 := r4.dup5 (by evm_kdecide) (by evm_ov)
  have r6 := r5.sub (by evm_kdecide) (by evm_ov)
  have r7 := r6.slt (by evm_kdecide) (by evm_ov)
  have r8 := r7.iszero (by evm_kdecide) (by evm_ov)
  have r9 := r8.push2 (UInt256.ofNat 2901) (by evm_kdecide) (by evm_ov)
  have r10 := r9.jumpiNT (by evm_kdecide) hcond (by evm_ov)
  have rFinal := RD.normalizePC (pc' := (UInt256.ofNat 2897)) r10 (by evm_kdecide)
  exact RD.normalizeCounters rFinal (by omega) (by omega)

/-- Packed RD summary for bytecode block with abstract final counters and active words. -/
theorem ethbridge_block_2883_fallthrough_packed {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {x0 x1 : UInt256} {R : List UInt256}
    (hstack : R.length + 6 ≤ 1024)
    (hcond : (UInt256.isZero (UInt256.slt (UInt256.sub x1 x0) (UInt256.ofNat 32))) = (UInt256.ofNat 0))
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2883) (x0 :: x1 :: R) mem aw rdata σ k C)
    : ∃ (aw' : UInt256) (k' C' : ℕ), RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2897) (ethbridge_block_2883_fallthrough_stack (x0 := x0) (x1 := x1) (R := R)) mem aw' rdata σ k' C' := by
  obtain ⟨k', C', h'⟩ := RD.pack (ethbridge_block_2883_fallthrough hstack hcond h)
  exact ⟨_, k', C', h'⟩

/-- Automatically generated RD summary for bytecode block at pc 2897. -/
theorem ethbridge_block_2897 {ee : ExecutionEnv} {g : Sat256} {s0 : State} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ} {R : List UInt256}
    (hstack : R.length + 2 ≤ 1024)
    (h : RD BridgeEvm.ethbridgeRuntime ee g s0 (UInt256.ofNat 2897) R mem aw rdata σ k C)
    : RDrev BridgeEvm.ethbridgeRuntime g s0 := by
  let r0 := h
  have r1 := r0.push1 (UInt256.ofNat 0) (by evm_kdecide) (by evm_ov)
  have r2 := r1.dup1 (by evm_kdecide) (by evm_ov)
  exact RD.genRev r2 (by evm_kdecide) (by evm_ov)

end ethbridgeBlocks
