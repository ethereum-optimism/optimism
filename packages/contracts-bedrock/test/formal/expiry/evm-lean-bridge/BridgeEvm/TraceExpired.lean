import BridgeEvm.TraceHash
import Ethereum.Theory.StaticStorage

namespace BridgeEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach ethbridgeBlocks Mem

/-- Memory at pc 1897 (after the `expiredMessages` call and its decoding): `hashMem` with the
    call input at `0x284` overwritten by the first 32 returned bytes `R`, and the free-memory
    pointer advanced to `fp = 0x284 + roundUp32(returndatasize)`. -/
abbrev expMem (I : ExecutionEnv) (H : UInt256) (R : List UInt8) (fp : ℕ) : List UInt8 :=
  List.replicate 64 0 ++ (wb (UInt256.ofNat fp) ++ (List.replicate 32 0 ++ (wb (UInt256.ofNat 100) ++ ((wb relaySelWord).take 4 ++ (wb (argFrom I) ++ (wb (argTo I) ++ (wb (argAmount I) ++ (wb (UInt256.ofNat 352) ++ (wb (argDest I) ++ (wb chainIdWord ++ (wb (argNonce I) ++ (wb (selfWord I) ++ (wb (selfWord I) ++ (wb (UInt256.ofNat 192) ++ (wb (UInt256.ofNat 100) ++ ((wb relaySelWord).take 4 ++ (wb (argFrom I) ++ (wb (argTo I) ++ (wb (argAmount I) ++ ((wb (UInt256.ofNat 0)).take 28 ++ (R ++ ((wb H).drop 28))))))))))))))))))))))

/-- The stack at pc 1897. -/
abbrev bodyStack (I : ExecutionEnv) (H : UInt256) : List UInt256 :=
  [H, argAmount I, argTo I, argFrom I, argNonce I, argDest I, UInt256.ofNat 127, refundSelector]

set_option maxHeartbeats 4000000 in
/-- Trace segment 3 (pc 1699 → 1897): the `STATICCALL` `expiredMessages(H)` to
    0x4200…0023 and the decoding of its result. Either the run reverts, or the call (with exactly
    the calldata `expiredCalldata H`) succeeded, returned at least 32 bytes whose first word is
    `1`, and execution continues at pc 1897. The reverting cases are: call depth 1024, the call
    failed, fewer than 32 bytes returned, a non-boolean first word, or the first word `0`
    (`SuperchainETHBridge_MessageNotExpired`). -/
theorem seg_expired {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256}
    {aw : UInt256} {k C : ℕ} {H : UInt256}
    (h : RD ethbridgeRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 1699)
        (H :: hashStackTail I) (ofL (hashMem I)) aw ByteArray.empty σ k C) :
    RDrev ethbridgeRuntime g (initState σ σ₀ g A I) ∨
    ∃ σ₁ o, StaticCall σ₀ I l2l2 (expiredCalldata H) σ σ₁ true o ∧ 32 ≤ o.size ∧
      o.size < 2 ^ 138 ∧ returnWord o = UInt256.ofNat 1 ∧
      accountStorageStateEq σ σ₁ ∧ accountCodeStateEq σ σ₁ ∧
      ∃ aw' k' C', RD ethbridgeRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 1897)
        (bodyStack I H) (ofL (expMem I H (o.data.toList.take 32) (644 + (o.size + 31) / 32 * 32)))
        aw' o σ₁ k' C' := by
  obtain ⟨aw1, k1, C1, r1⟩ := ethbridge_block_1699_packed (by simp) h
  simp only [ethbridge_block_1699_stack, ethbridge_block_1699_memory] at r1
  msimp at r1
  have hdec : decode ethbridgeRuntime (UInt256.ofNat 1790) = some (.STATICCALL, .none) := by
    evm_kdecide
  by_cases hd : I.depth.val < 1024
  · obtain ⟨σ', z, o, A_in, callGas, k3, C3, ⟨g'', A', hΘ⟩, r3, hosize⟩ :=
      RD.solcStaticcall r1 hdec hd (by simp)
    msimp at hΘ
    rw [← expiredCalldata_eq] at hΘ
    have hcall : StaticCall σ₀ I l2l2 (expiredCalldata H) σ σ' z o :=
      ⟨A_in, callGas, g'', A', hΘ⟩
    have hst : accountStorageStateEq σ σ' := Theta_static_accountStorageStateEq hΘ.symm
    have hcd : accountCodeStateEq σ σ' := Theta_static_accountCodeStateEq hΘ.symm
    cases z with
    | false =>
      simp only [Bool.false_eq_true, if_false] at r3
      have r4 := ethbridge_block_1791_fallthrough (by simp) (by decide) r3
      exact Or.inl (ethbridge_block_1798 (by simp [ethbridge_block_1791_fallthrough_stack]) r4)
    | true =>
      simp only [if_true] at r3
      have hob : o.size < 2 ^ 138 := Theta_returnData_size_lt_2pow138_of_eq _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ hΘ
        (by rw [expiredCalldata_eq]; simp [Ethereum.EVM.maxReturnDataSizeByGas, Ethereum.EVM.maxReturnDataWordsByGas])
      clear hΘ
      have r4 := ethbridge_block_1791_taken (by simp) (by decide) (by kjump_dest) r3
      simp only [ethbridge_block_1791_taken_stack] at r4
      by_cases hlen : 32 ≤ o.size
      swap
      · obtain ⟨_, _, _, r5⟩ := ethbridge_block_1807_packed (by simp) (by kjump_dest) r4
        simp only [ethbridge_block_1807_stack] at r5
        have r6 := ethbridge_block_2908_fallthrough (by simp)
          (by rw [Words.sub_add_self]; exact Words.retlen_bad _ (by omega)) r5
        exact Or.inl (ethbridge_block_2922 (by simp [ethbridge_block_2908_fallthrough_stack]) r6)
      rw [Words.min32_toNat _ hlen (by omega), write_bytes32 _ _ _ hlen] at r4
      have hRl := length_take32 o hlen
      generalize hR : o.data.toList.take 32 = R at r4 hRl
      msimp [hRl] at r4
      obtain ⟨aw5, k5, C5, r5⟩ := ethbridge_block_1807_packed (by simp) (by kjump_dest) r4
      simp only [ethbridge_block_1807_stack, ethbridge_block_1807_memory] at r5
      rw [Words.round32 _ (by omega)] at r5
      have hsz : Bounded 31 o.size ∧ Bounded o.size (2 ^ 138) := ⟨hlen, hob⟩
      clear hlen hob
      msimp [hRl] at r5
      have hlen : 32 ≤ o.size := hsz.1
      have hob : o.size < 2 ^ 138 := hsz.2
      have r6 := ethbridge_block_2908_taken (by simp)
        (by rw [show UInt256.ofNat (644 + o.size) = UInt256.ofNat 644 + UInt256.ofNat o.size from
              (ofNat_add_ofNat _ _).symm, Words.sub_add_self]
            exact Words.retlen_ok _ hlen (by omega)) (by kjump_dest) r5
      simp only [ethbridge_block_2908_taken_stack] at r6
      clear hlen hob
      by_cases hb : UInt256.eq (UInt256.ofNat (fromBytesBigEndian R))
          (UInt256.isZero (UInt256.isZero (UInt256.ofNat (fromBytesBigEndian R)))) = UInt256.ofNat 0
      · have r7 := ethbridge_block_2926_fallthrough (by simp) (by msimpg [hRl]; exact hb) r6
        exact Or.inl (ethbridge_block_2938 (by simp [ethbridge_block_2926_fallthrough_stack]) r7)
      obtain ⟨_, _, _, r7⟩ := ethbridge_block_2926_taken_packed (by simp) (by msimpg [hRl]; exact hb) (by kjump_dest) r6
      simp only [ethbridge_block_2926_taken_stack] at r7
      msimp [hRl] at r7
      have r8 := ethbridge_block_2617 (by simp) (by kjump_dest) r7
      simp only [ethbridge_block_2617_stack] at r8
      by_cases hz : UInt256.ofNat (fromBytesBigEndian R) = UInt256.ofNat 0
      · have r9 := ethbridge_block_1843_fallthrough (by simp) hz r8
        exact Or.inl (ethbridge_block_1848 (by simp [ethbridge_block_1843_fallthrough_stack]) r9)
      have r9 := ethbridge_block_1843_taken (by simp) hz (by kjump_dest) r8
      simp only [ethbridge_block_1843_taken_stack] at r9
      have hw : returnWord o = UInt256.ofNat 1 := by
        unfold returnWord; rw [hR]; exact Words.bool_true hb hz
      right
      subst hR
      exact ⟨σ', o, hcall, hsz.1, hsz.2, hw, hst, hcd, _, _, _, r9⟩
  · have hd' : I.depth = 1024 := by
      apply Fin.ext; have := I.depth.isLt; simp only [Fin.val_ofNat] at *; omega
    obtain ⟨k3, C3, r3⟩ := RD.solcStaticcallDepthLimit r1 hdec hd' (by simp)
    have r4 := ethbridge_block_1791_fallthrough (by simp) (by decide) r3
    exact Or.inl (ethbridge_block_1798 (by simp [ethbridge_block_1791_fallthrough_stack]) r4)

end BridgeEvm
