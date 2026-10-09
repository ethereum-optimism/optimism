import BridgeEvm.TraceEntry

namespace BridgeEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach ethbridgeBlocks Mem

theorem refundHash_eq (I : ExecutionEnv) :
    refundHash I = UInt256.ofNat (fromByteArrayBigEndian (KEC (ofL (refundPreimageL I)))) := by
  rw [refundHash, refundPreimage_eq]

/-- Memory at pc 1699, after the hash: the `relayETH` call data at `0x80` (length word `100`,
    then 100 bytes), the length word `352` at `0x104`, the preimage at `0x124..0x284` and the
    word written by the copy loop's tail, and the free-memory pointer `0x284`. -/
abbrev hashMem (I : ExecutionEnv) : List UInt8 :=
  List.replicate 64 0 ++ (wb (UInt256.ofNat 644) ++ (List.replicate 32 0 ++ (wb (UInt256.ofNat 100) ++ ((wb relaySelWord).take 4 ++ (wb (argFrom I) ++ (wb (argTo I) ++ (wb (argAmount I) ++ (wb (UInt256.ofNat 352) ++ (wb (argDest I) ++ (wb chainIdWord ++ (wb (argNonce I) ++ (wb (selfWord I) ++ (wb (selfWord I) ++ (wb (UInt256.ofNat 192) ++ (wb (UInt256.ofNat 100) ++ ((wb relaySelWord).take 4 ++ (wb (argFrom I) ++ (wb (argTo I) ++ (wb (argAmount I) ++ (wb (UInt256.ofNat 0)))))))))))))))))))))

/-- The stack at pc 1699 below the hash. -/
abbrev hashStackTail (I : ExecutionEnv) : List UInt256 :=
  [UInt256.ofNat 0, argAmount I, argTo I, argFrom I, argNonce I, argDest I, UInt256.ofNat 127,
    refundSelector]

/-- Trace segment 2 (pc 1539 → 1699): `abi.encodeCall(relayETH, …)`, `abi.encode(…)` with the
    unrolled 4-iteration copy loop, and `KECCAK256`: the word on top of the stack is exactly
    `refundHash I`. No branches. -/
theorem seg_hash {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256}
    {aw : UInt256} {k C : ℕ} (hargs : ArgsOk I)
    (h : RD ethbridgeRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 1539)
        (entryStack I) (ofL entryMem) aw ByteArray.empty σ k C) :
    ∃ aw' k' C', RD ethbridgeRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 1699)
        (refundHash I :: hashStackTail I) (ofL (hashMem I)) aw' ByteArray.empty σ k' C' := by
  have hF := Words.land_mask_of_lt hargs.fromClean
  have hF' := Words.land_mask_of_lt' hargs.fromClean
  have hT := Words.land_mask_of_lt hargs.toClean
  have hT' := Words.land_mask_of_lt' hargs.toClean
  have hS := Words.self_mask I.codeOwner
  obtain ⟨aw1, k1, C1, r1⟩ := ethbridge_block_1539_packed (by simp) h
  simp only [ethbridge_block_1539_stack, ethbridge_block_1539_memory, hF, hF', hT, hT'] at r1
  msimp [wb_merge'] at r1
  have r2 := ethbridge_block_1692 (by simp) (by kjump_dest) r1
  simp only [ethbridge_block_1692_stack] at r2
  obtain ⟨aw3, k3, C3, r3⟩ := ethbridge_block_2314_packed (by simp) (by kjump_dest) r2
  simp only [ethbridge_block_2314_stack] at r3
  msimp at r3
  obtain ⟨aw4, k4, C4, r4⟩ := ethbridge_block_2942_packed (by simp) (by kjump_dest) r3
  simp only [ethbridge_block_2942_stack, ethbridge_block_2942_memory, hS] at r4
  msimp at r4
  obtain ⟨aw5, k5, C5, r5⟩ := ethbridge_block_2491_packed (by simp) r4
  simp only [ethbridge_block_2491_stack, ethbridge_block_2491_memory] at r5
  msimp at r5
  have ra0 := ethbridge_block_2501_fallthrough (by simp) (by decide) r5
  obtain ⟨_, _, _, rb0⟩ := ethbridge_block_2510_packed (by simp) (by kjump_dest) ra0
  simp only [ethbridge_block_2510_stack, ethbridge_block_2510_memory] at rb0
  msimp at rb0
  have ra1 := ethbridge_block_2501_fallthrough (by simp) (by decide) rb0
  obtain ⟨_, _, _, rb1⟩ := ethbridge_block_2510_packed (by simp) (by kjump_dest) ra1
  simp only [ethbridge_block_2510_stack, ethbridge_block_2510_memory] at rb1
  msimp at rb1
  have ra2 := ethbridge_block_2501_fallthrough (by simp) (by decide) rb1
  obtain ⟨_, _, _, rb2⟩ := ethbridge_block_2510_packed (by simp) (by kjump_dest) ra2
  simp only [ethbridge_block_2510_stack, ethbridge_block_2510_memory] at rb2
  msimp at rb2
  have ra3 := ethbridge_block_2501_fallthrough (by simp) (by decide) rb2
  obtain ⟨_, _, _, rb3⟩ := ethbridge_block_2510_packed (by simp) (by kjump_dest) ra3
  simp only [ethbridge_block_2510_stack, ethbridge_block_2510_memory] at rb3
  msimp at rb3
  have r6 := ethbridge_block_2501_taken (by simp) (by decide) (by kjump_dest) rb3
  have r7 := ethbridge_block_2529_fallthrough (by simp) (by decide) r6
  obtain ⟨_, _, _, r8⟩ := ethbridge_block_2538_packed (by simp) r7
  simp only [ethbridge_block_2538_memory] at r8
  msimp at r8
  have r9 := ethbridge_block_2547 (by simp) (by kjump_dest) r8
  simp only [ethbridge_block_2547_stack] at r9
  msimp at r9
  have r10 := ethbridge_block_3017 (by simp) (by kjump_dest) r9
  simp only [ethbridge_block_3017_stack] at r10
  obtain ⟨_, _, _, r11⟩ := ethbridge_block_2343_packed (by simp) (by kjump_dest) r10
  simp only [ethbridge_block_2343_stack, ethbridge_block_2343_memory] at r11
  msimp at r11
  rw [refundHash_eq]
  exact ⟨_, _, _, r11⟩

end BridgeEvm
