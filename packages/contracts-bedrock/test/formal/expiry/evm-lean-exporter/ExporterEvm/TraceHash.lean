import ExporterEvm.HashMem

/-!
# Trace segment 2 (pc 170 → 240): the message hash

`bytes memory m = _message` (pc 170), `abi.encode(block.chainid, _source, _nonce, _sender,
_target, m)` (pc 808, 1368, 1132, the copy loop with ⌈len/32⌉ iterations, its tail, 1443) and
`KECCAK256` (pc 837). No branch can revert here: the word left on the stack is exactly
`exportHash I`. Afterwards only the free-memory pointer `endPtr I` matters, so the memory is
abstracted as any list `B` with `FmpMem B (endPtr I)`.
-/

namespace ExporterEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach exporterBlocks Mem

/-- The free-memory pointer after the hash: `p1 + 256 + roundUp32(len)`. -/
def endPtr (I : ExecutionEnv) : ℕ := p1 I + 256 + roundUp32 (msgLen I)

/-- Memory `B` whose free-memory-pointer word (`0x40`) holds `fp`, with `96 ≤ |B| ≤ fp + 32`
    (nothing at or above `fp + 32` was written). -/
structure FmpMem (B : List UInt8) (fp : ℕ) : Prop where
  fmp : (B.drop 64).take 32 = wb (UInt256.ofNat fp)
  len96 : 96 ≤ B.length
  hi : B.length ≤ fp + 32
  ge : 96 ≤ fp
  bound : fp < 2 ^ 150

/-- The stack at pc 240 below the hash. -/
abbrev hashTail (I : ExecutionEnv) : List UInt256 :=
  [UInt256.ofNat 0, argMinGas I, argMsgLen I, UInt256.ofNat (msgPos I + 32), argTarget I, argSender I,
    argNonce I, argSource I, argSrcMessenger I, UInt256.ofNat 78, exportSelector]

/-- The six head words and the length word of the encoding (`p1 .. p1 + 256`). -/
def headQ (I : ExecutionEnv) : List UInt8 :=
  List.replicate 32 0 ++ (wb (UInt256.ofNat Ethereum.chainId) ++ (wb (argSource I) ++ (wb (argNonce I) ++
    (wb (UInt256.land (argSender I) addrMask) ++ (wb (UInt256.land (argTarget I) addrMask) ++
    wb (UInt256.ofNat 192)))))) ++ wb (argMsgLen I)

theorem headQ_eq (I : ExecutionEnv) :
    List.replicate 32 0 ++ (wb (UInt256.ofNat Ethereum.chainId) ++ (wb (argSource I) ++ (wb (argNonce I) ++
      (wb (UInt256.land (argSender I) addrMask) ++ (wb (UInt256.land (argTarget I) addrMask) ++
      wb (UInt256.ofNat 192)))))) ++ wb (argMsgLen I) = headQ I := rfl

set_option maxHeartbeats 4000000 in
theorem seg_hash {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256}
    {aw : UInt256} {k C : ℕ} (hargs : ArgsOk I) (hcds : I.calldata.size < 2 ^ 63)
    (h : RD exporterRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 170)
        (entryStack I) (ofL entryMem) aw ByteArray.empty σ k C) :
    ∃ B aw' k' C', FmpMem B (endPtr I) ∧
      RD exporterRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 240)
        (exportHash I :: hashTail I) (ofL B) aw' ByteArray.empty σ k' C' := by
  have hA := memA_length I hargs
  have hL := msgLen_lt I hargs hcds
  have hr1 := roundUp32_ge (msgLen I)
  have hr2 := roundUp32_lt (msgLen I)
  have hre := roundUp32_eq (msgLen I)
  have hMl := msgData_length I hargs
  have hp1 : p1 I = 160 + roundUp32 (msgLen I) := rfl
  obtain ⟨_, _, _, r1⟩ := exporter_block_170_packed (by simp) (by kjump_dest) h
  simp only [exporter_block_170_stack] at r1
  rw [mem170_eq I hargs hcds] at r1
  have hml0 : memLoad (UInt256.ofNat 64) (ofL entryMem) = UInt256.ofNat 128 := by msimpg
  rw [hml0] at r1
  have r2 := exporter_block_808 (by simp) (by kjump_dest) r1
  simp only [exporter_block_808_stack, memLoad64, ← hA] at r2
  obtain ⟨_, _, _, r3⟩ := exporter_block_1368_packed (by simp) (by kjump_dest) r2
  simp only [exporter_block_1368_stack] at r3
  rw [mem1368_eq (memA I) _ (by omega) (by omega)] at r3
  obtain ⟨_, _, _, r4⟩ := exporter_block_1132_packed (by simp) r3
  simp only [exporter_block_1132_stack] at r4
  rw [mem1132_eq I hargs hcds _ (by simp), memLoad128] at r4
  have hpos : UInt256.ofNat 32 + UInt256.ofNat (memA I).length + UInt256.ofNat 192 =
      UInt256.ofNat ((memA I).length + 224) := by
    rw [ofNat_add_ofNat, ofNat_add_ofNat, show 32 + (memA I).length + 192 = (memA I).length + 224 by omega]
  rw [hpos, headQ_eq] at r4
  -- the copy loop
  have hQl : (headQ I).length = 256 := by simp [headQ]
  have hsrc := loop_src I hargs (headQ I)
  have hMpl : (msgData I ++ padZ I).length = 32 * ((msgLen I + 31) / 32) := by
    rw [List.length_append, hMl]; unfold padZ; rw [List.length_replicate]; omega
  obtain ⟨_, _, _, r5⟩ := copy_loop (ee := I) (Pre := memA I ++ headQ I) (Mp := msgData I ++ padZ I)
    (v := 128) (pos := (memA I).length + 224) (L := msgLen I) (n := (msgLen I + 31) / 32)
    (Lw := argMsgLen I) rfl (by omega) (by omega) hsrc (by rw [List.length_append, hQl]; omega)
    (by rw [List.length_append, hQl]) (by omega) ((msgLen I + 31) / 32) 0
    (by omega) _ _ _ (by simpa only [Nat.mul_zero, List.take_zero, List.append_nil] using r4) (by simp)
  obtain ⟨_, _, _, r6⟩ := copy_tail (L := msgLen I) (n := (msgLen I + 31) / 32) rfl (by omega) (by omega)
    hMpl (by rw [List.length_append, hQl]) (by omega) (by simp) (by kjump_dest) r5
  have r7 := exporter_block_1443 (by simp) (by kjump_dest) r6
  simp only [exporter_block_1443_stack, List.append_assoc,
    show (memA I).length + 224 + 32 + 32 * ((msgLen I + 31) / 32) = p1 I + 256 + roundUp32 (msgLen I) by omega] at r7
  have hT := padTail_length I hargs
  obtain ⟨_, _, _, r8⟩ := exporter_block_837_packed (by simp) (by kjump_dest) r7
  simp only [exporter_block_837_stack, exporter_block_837_memory, memLoad64] at r8
  rw [mem837_eq I hargs hcds (headQ I) _ hQl hT.2, keccak837 I hargs hcds (headQ I) _ hQl hT.1,
    padTail_take I hargs] at r8
  have hH : UInt256.ofNat (fromByteArrayBigEndian (KEC (ofL ((headQ I).drop 32 ++ (msgData I ++ padZ I))))) =
      exportHash I := by
    unfold exportHash
    rw [exportPreimage_eq I hargs, show chainIdWord = UInt256.ofNat Ethereum.chainId from rfl]
    unfold headQ addrMask
    rw [Words.land_mask_of_lt hargs.senderClean, Words.land_mask_of_lt hargs.targetClean]
    simp only [List.append_assoc]
    rw [List.drop_left' (by simp)]
    simp only [List.append_assoc]
  rw [hH] at r8
  have hR := memRest_length I hargs
  refine ⟨_, _, _, _, ⟨?_, ?_, ?_, ?_, ?_⟩, r8⟩
  · rw [List.drop_left' (by simp), List.take_left' (by simp)]; rfl
  · simp only [List.length_append, List.length_replicate, length_wb, List.length_drop, hR, hQl]
    unfold p1 at hR ⊢; omega
  · simp only [List.length_append, List.length_replicate, length_wb, List.length_drop, hR, hQl]
    unfold endPtr; omega
  · unfold endPtr; omega
  · unfold endPtr; omega

end ExporterEvm
