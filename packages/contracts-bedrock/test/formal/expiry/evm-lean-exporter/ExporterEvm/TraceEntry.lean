import ExporterEvm.Terminal
import ExporterEvm.Words2
import ExporterEvm.AllBlocks

/-!
# Trace segment 1 (pc 0 → 170): dispatcher, non-payable check, ABI decoder

Either the run reverts with empty output (some ABI condition fails), or `ArgsOk` holds and the
run reaches the body of `exportUndeliveredMessage` (pc 170) with the decoded arguments on the
stack and only the free-memory pointer `0x80` in memory.
-/

namespace ExporterEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach exporterBlocks Mem

theorem mem0 : exporter_block_0_taken_memory (mem := ByteArray.empty) =
    ofL (List.replicate 64 0 ++ wb (UInt256.ofNat 128)) := by
  unfold exporter_block_0_taken_memory
  rw [empty_ofL, write_wb]
  mnorm

/-- The stack at pc 170: the decoded arguments (`_minGasLimit`, `_message.length`, the calldata
    position of `_message`'s bytes, `_target`, `_sender`, `_nonce`, `_source`,
    `_sourceMessenger`), the return pc `0x4e` and the selector. -/
abbrev entryStack (I : ExecutionEnv) : List UInt256 :=
  [argMinGas I, argMsgLen I, UInt256.ofNat (msgPos I + 32), argTarget I, argSender I, argNonce I,
    argSource I, argSrcMessenger I, UInt256.ofNat 78, exportSelector]

/-- Memory at pc 170: the free-memory pointer `0x80` at `0x40`. -/
abbrev entryMem : List UInt8 := List.replicate 64 0 ++ wb (UInt256.ofNat 128)

theorem argWord_eq (I : ExecutionEnv) (n : ℕ) :
    uInt256OfByteArray (I.calldata.readBytes n 32) = argWord I n := rfl

theorem toNat_lit' (n : ℕ) (h : n < 2 ^ 256) : (UInt256.ofNat n).toNat = n := Words.toNat_ofNat_lt h

set_option maxHeartbeats 4000000 in
theorem seg_entry {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256}
    (hcode : I.code = exporterRuntime) (hsel : selectorWord I = exportSelector)
    (hcds : I.calldata.size < 2 ^ 63) :
    RDrevP exporterRuntime g (initState σ σ₀ g A I) (fun o => o = ByteArray.empty) ∨
    (ArgsOk I ∧ ∃ aw k C, RD exporterRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 170)
        (entryStack I) (ofL entryMem) aw ByteArray.empty σ k C) := by
  have r0 := RD.initState (σ := σ) (σ₀ := σ₀) (A := A) (g := g) hcode
  have hsel' : UInt256.shiftRight (uInt256OfByteArray (I.calldata.readBytes (UInt256.ofNat 0).toNat 32))
      (UInt256.ofNat 224) = UInt256.ofNat 409891624 := hsel
  have hsz : (UInt256.ofNat I.calldata.size).toNat = I.calldata.size := toNat_lit' _ (by omega)
  -- callvalue
  by_cases hv : UInt256.isZero I.weiValue = UInt256.ofNat 0
  · left
    exact RD.rev00 (exporter_block_0_fallthrough (R := []) (by simp) hv r0) (by evm_kdecide) (by evm_kdecide)
      (by evm_kdecide) (by simp [exporter_block_0_fallthrough_stack])
  have r1 := exporter_block_0_taken (R := []) (by simp) hv (by kjump_dest) r0
  rw [mem0] at r1
  simp only [exporter_block_0_taken_stack] at r1
  -- calldatasize ≥ 4
  by_cases hlt : UInt256.lt (UInt256.ofNat I.calldata.size) (UInt256.ofNat 4) = UInt256.ofNat 0
  swap
  · left
    have r2 := exporter_block_16_taken (by simp) (by exact hlt) (by kjump_dest) r1
    simp only [exporter_block_16_taken_stack] at r2
    have r3 := kevm_run r2 with [jumpdest]
    exact RD.rev00 r3 (by evm_kdecide) (by evm_kdecide) (by evm_kdecide) (by simp)
  have h4 : 4 ≤ I.calldata.size := by
    have := Words.lt_eq0.mp hlt; rw [hsz] at this; simpa using this
  have r2 := exporter_block_16_fallthrough (by simp) (by exact hlt) r1
  simp only [exporter_block_16_fallthrough_stack] at r2
  -- selector
  have r3 := exporter_block_26_taken (by simp) (by rw [hsel']; decide) (by kjump_dest) r2
  simp only [exporter_block_26_taken_stack] at r3
  rw [hsel'] at r3
  have r4 := exporter_block_59 (by simp) (by kjump_dest) r3
  simp only [exporter_block_59_stack] at r4
  -- static head length
  have hsub : (UInt256.sub (UInt256.ofNat I.calldata.size) (UInt256.ofNat 4)).toNat =
      I.calldata.size - 4 := by
    rw [Words.toNat_sub_of_le (by rw [hsz]; simpa using h4), hsz]; rfl
  by_cases hhd : UInt256.isZero (UInt256.slt (UInt256.sub (UInt256.ofNat I.calldata.size)
      (UInt256.ofNat 4)) (UInt256.ofNat 224)) ≠ UInt256.ofNat 0
  swap
  · left
    have r5 := exporter_block_932_fallthrough (by simp) (by exact not_not.mp hhd) r4
    exact RD.rev00 r5 (by evm_kdecide) (by evm_kdecide) (by evm_kdecide) (by simp [exporter_block_932_fallthrough_stack])
  have h228 : 228 ≤ I.calldata.size := by
    have := (Words.isZero_slt_small_ne0 (by rw [hsub]; omega) (by decide)).mp hhd
    rw [hsub, toNat_lit' 224 (by norm_num)] at this; omega
  have r5 := exporter_block_932_taken (by simp) (by exact hhd) (by kjump_dest) r4
  simp only [exporter_block_932_taken_stack] at r5
  have r6 := exporter_block_960 (by simp) (by kjump_dest) r5
  simp only [exporter_block_960_stack] at r6
  -- _sourceMessenger
  by_cases ha0 : UInt256.eq (argWord I 4) (UInt256.land (argWord I 4) (UInt256.ofNat 1461501637330902918203684832716283019655932542975)) = UInt256.ofNat 0
  · left
    have r7 := exporter_block_871_fallthrough (by simp) (by simp only [toNat_ofNat_mod, Nat.reduceMod, argWord_eq]; exact ha0) r6
    exact RD.rev00 r7 (by evm_kdecide) (by evm_kdecide) (by evm_kdecide) (by simp [exporter_block_871_fallthrough_stack])
  have r7 := exporter_block_871_taken (by simp) (by simp only [toNat_ofNat_mod, Nat.reduceMod, argWord_eq]; exact ha0) (by kjump_dest) r6
  simp only [exporter_block_871_taken_stack, toNat_ofNat_mod, Nat.reduceMod, argWord_eq] at r7
  have r8 := exporter_block_907 (by simp) (by kjump_dest) r7
  simp only [exporter_block_907_stack] at r8
  have r9 := exporter_block_969 (by simp) (by kjump_dest) r8
  simp only [exporter_block_969_stack, ofNat_add_ofNat, toNat_ofNat_mod, Nat.reduceAdd,
    Nat.reduceMod, argWord_eq] at r9
  -- _sender
  by_cases ha3 : UInt256.eq (argWord I 100) (UInt256.land (argWord I 100) (UInt256.ofNat 1461501637330902918203684832716283019655932542975)) = UInt256.ofNat 0
  · left
    have r10 := exporter_block_871_fallthrough (by simp) (by simp only [toNat_ofNat_mod, Nat.reduceMod, argWord_eq]; exact ha3) r9
    exact RD.rev00 r10 (by evm_kdecide) (by evm_kdecide) (by evm_kdecide) (by simp [exporter_block_871_fallthrough_stack])
  have r10 := exporter_block_871_taken (by simp) (by simp only [toNat_ofNat_mod, Nat.reduceMod, argWord_eq]; exact ha3) (by kjump_dest) r9
  simp only [exporter_block_871_taken_stack, toNat_ofNat_mod, Nat.reduceMod, argWord_eq] at r10
  have r11 := exporter_block_907 (by simp) (by kjump_dest) r10
  simp only [exporter_block_907_stack] at r11
  have r12 := exporter_block_997 (by simp) (by kjump_dest) r11
  simp only [exporter_block_997_stack, ofNat_add_ofNat, toNat_ofNat_mod, Nat.reduceAdd,
    Nat.reduceMod, argWord_eq] at r12
  -- _target
  by_cases ha4 : UInt256.eq (argWord I 132) (UInt256.land (argWord I 132) (UInt256.ofNat 1461501637330902918203684832716283019655932542975)) = UInt256.ofNat 0
  · left
    have r13 := exporter_block_871_fallthrough (by simp) (by simp only [toNat_ofNat_mod, Nat.reduceMod, argWord_eq]; exact ha4) r12
    exact RD.rev00 r13 (by evm_kdecide) (by evm_kdecide) (by evm_kdecide) (by simp [exporter_block_871_fallthrough_stack])
  have r13 := exporter_block_871_taken (by simp) (by simp only [toNat_ofNat_mod, Nat.reduceMod, argWord_eq]; exact ha4) (by kjump_dest) r12
  simp only [exporter_block_871_taken_stack, toNat_ofNat_mod, Nat.reduceMod, argWord_eq] at r13
  have r14 := exporter_block_907 (by simp) (by kjump_dest) r13
  simp only [exporter_block_907_stack] at r14
  -- `bytes` offset ≤ 2^64 - 1
  by_cases hoff : UInt256.isZero (UInt256.gt (argWord I 164) (UInt256.ofNat 18446744073709551615))
      = UInt256.ofNat 0
  · left
    have r15 := exporter_block_1011_fallthrough (by simp) (by
      simp only [ofNat_add_ofNat, toNat_ofNat_mod, Nat.reduceAdd, Nat.reduceMod, argWord_eq]; exact hoff) r14
    exact RD.rev00 r15 (by evm_kdecide) (by evm_kdecide) (by evm_kdecide) (by simp [exporter_block_1011_fallthrough_stack])
  have hoffN : (argMsgOffset I).toNat ≤ 2 ^ 64 - 1 := by
    have := Words.isZero_gt_eq0.not.mp hoff
    rw [toNat_lit' _ (by norm_num)] at this; unfold argMsgOffset; norm_num; omega
  have r15 := exporter_block_1011_taken (by simp) (by
      simp only [ofNat_add_ofNat, toNat_ofNat_mod, Nat.reduceAdd, Nat.reduceMod, argWord_eq]; exact hoff)
    (by kjump_dest) r14
  simp only [exporter_block_1011_taken_stack, ofNat_add_ofNat, toNat_ofNat_mod, Nat.reduceAdd,
    Nat.reduceMod, argWord_eq] at r15
  have hp : UInt256.ofNat 4 + argWord I 164 = UInt256.ofNat (msgPos I) := by
    rw [Words.ext_iff, Words.toNat_add_of_lt (by rw [toNat_lit' 4 (by norm_num)]; unfold argMsgOffset at hoffN; omega),
      toNat_lit' 4 (by norm_num), toNat_lit' _ (by unfold msgPos; omega)]
    rfl
  have hpN : (UInt256.ofNat (msgPos I)).toNat = msgPos I := toNat_lit' _ (by unfold msgPos; omega)
  -- length word inside the calldata
  by_cases hin : UInt256.slt (UInt256.ofNat 4 + argWord I 164 + UInt256.ofNat 31)
      (UInt256.ofNat I.calldata.size) = UInt256.ofNat 0
  · left
    have r16 := exporter_block_1040_fallthrough (by simp) (by exact hin) r15
    exact RD.rev00 r16 (by evm_kdecide) (by evm_kdecide) (by evm_kdecide) (by simp [exporter_block_1040_fallthrough_stack])
  have hinN : msgPos I + 31 < I.calldata.size := by
    rw [hp, ofNat_add_ofNat] at hin
    have := (Words.slt_small_ne0 (by rw [toNat_lit' _ (by unfold msgPos; omega)]; unfold msgPos; omega)
      (by rw [hsz]; omega)).mp hin
    rw [toNat_lit' _ (by unfold msgPos; omega), hsz] at this; exact this
  have r16 := exporter_block_1040_taken (by simp) (by exact hin) (by kjump_dest) r15
  simp only [exporter_block_1040_taken_stack, hp] at r16
  -- length ≤ 2^64 - 1
  by_cases hlen : UInt256.isZero (UInt256.gt (argMsgLen I) (UInt256.ofNat 18446744073709551615))
      = UInt256.ofNat 0
  · left
    have r17 := exporter_block_1060_fallthrough (by simp) (by rw [hpN, argWord_eq]; exact hlen) r16
    exact RD.rev00 r17 (by evm_kdecide) (by evm_kdecide) (by evm_kdecide) (by simp [exporter_block_1060_fallthrough_stack])
  have hlenN : msgLen I ≤ 2 ^ 64 - 1 := by
    have := Words.isZero_gt_eq0.not.mp hlen
    rw [toNat_lit' _ (by norm_num)] at this; unfold msgLen; norm_num; omega
  have r17 := exporter_block_1060_taken (by simp) (by rw [hpN, argWord_eq]; exact hlen) (by kjump_dest) r16
  simp only [exporter_block_1060_taken_stack, hpN, argWord_eq] at r17
  have hpl : UInt256.ofNat (msgPos I) + argMsgLen I + UInt256.ofNat 32 =
      UInt256.ofNat (msgPos I + 32 + msgLen I) := by
    rw [Words.ext_iff, Words.toNat_add_of_lt, Words.toNat_add_of_lt, hpN, toNat_lit' 32 (by norm_num),
      toNat_lit' _ (by unfold msgPos msgLen at *; omega)]
    · unfold msgLen; omega
    · rw [hpN]; unfold msgLen at hlenN; unfold msgPos at *; omega
    · rw [Words.toNat_add_of_lt, hpN, toNat_lit' 32 (by norm_num)]
      · unfold msgLen at hlenN; unfold msgPos at *; omega
      · rw [hpN]; unfold msgLen at hlenN; unfold msgPos at *; omega
  -- bytes inside the calldata
  by_cases hbi : UInt256.isZero (UInt256.gt (UInt256.ofNat (msgPos I) + argMsgLen I + UInt256.ofNat 32)
      (UInt256.ofNat I.calldata.size)) = UInt256.ofNat 0
  · left
    have r18 := exporter_block_1075_fallthrough (by simp) (by exact hbi) r17
    exact RD.rev00 r18 (by evm_kdecide) (by evm_kdecide) (by evm_kdecide) (by simp)
  have hbiN : msgPos I + 32 + msgLen I ≤ I.calldata.size := by
    have := Words.isZero_gt_ne0.mp hbi
    rw [hpl, toNat_lit' _ (by unfold msgPos msgLen at *; omega), hsz] at this; exact this
  have r18 := exporter_block_1075_taken (by simp) (by exact hbi) (by kjump_dest) r17
  have r19 := exporter_block_1093 (by simp) (by kjump_dest) r18
  simp only [exporter_block_1093_stack, ofNat_add_ofNat, Nat.reduceAdd] at r19
  -- _minGasLimit
  by_cases hmg : UInt256.eq (argWord I 196) (UInt256.land (argWord I 196) (UInt256.ofNat 4294967295))
      = UInt256.ofNat 0
  · left
    have r20 := exporter_block_912_fallthrough (by simp) (by
      simp only [toNat_ofNat_mod, Nat.reduceMod, argWord_eq]; exact hmg) r19
    exact RD.rev00 r20 (by evm_kdecide) (by evm_kdecide) (by evm_kdecide) (by simp [exporter_block_912_fallthrough_stack])
  have r20 := exporter_block_912_taken (by simp) (by
      simp only [toNat_ofNat_mod, Nat.reduceMod, argWord_eq]; exact hmg) (by kjump_dest) r19
  simp only [exporter_block_912_taken_stack, toNat_ofNat_mod, Nat.reduceMod, argWord_eq] at r20
  have r21 := exporter_block_907 (by simp) (by kjump_dest) r20
  simp only [exporter_block_907_stack] at r21
  have r22 := exporter_block_1117 (by simp) (by kjump_dest) r21
  simp only [exporter_block_1117_stack] at r22
  have r23 := exporter_block_73 (by simp) (by kjump_dest) r22
  right
  exact ⟨⟨Words.isZero_ne0.mp hv, h228, (Words.addr_clean_iff _).mp ha0, (Words.addr_clean_iff _).mp ha3,
    (Words.addr_clean_iff _).mp ha4, hoffN, hinN, hlenN, hbiN, (Words.uint32_clean_iff _).mp hmg⟩,
    _, _, _, r23⟩

end ExporterEvm
