import ExpiryEvm.TraceCall2

/-! # Trace segment 4: `sentMessageTimestamps` read, expiry check, `expiredMessages` write,
event, `STOP` (pc 2103 → end) -/

namespace ExpiryEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach l2tol2Blocks Mem

theorem optWord_eq (σ : AccountMap) (a : AccountAddress) (s : UInt256) :
    (σ.get? a |>.option ⟨0⟩ (fun ac => ac.storage.getD s ⟨0⟩)) = storageWord σ a s := by
  unfold storageWord
  cases h : σ.get? a with
  | none =>
    have this : σ[a]? = none := by rw [← Std.ExtTreeMap.get?_eq_getElem?]; exact h
    have hd : σ.getD a default = default := by
      rw [Std.ExtTreeMap.getD_eq_getD_getElem?, this]; rfl
    rw [hd]
    simp only [Option.option]
    show (⟨0⟩ : UInt256) = (∅ : Storage).getD s ⟨0⟩
    simp
  | some ac =>
    have this : σ[a]? = some ac := by rw [← Std.ExtTreeMap.get?_eq_getElem?]; exact h
    have hd : σ.getD a default = ac := by
      rw [Std.ExtTreeMap.getD_eq_getD_getElem?, this]; rfl
    rw [hd]
    rfl

theorem storageWord_eq_of_storageEq {σ σ' : AccountMap} (h : accountStorageStateEq σ σ')
    (a : AccountAddress) (s : UInt256) : storageWord σ' a s = storageWord σ a s := by
  unfold storageWord; rw [(h a).1]

/-- The storage conditions of `expireMessage` on the pre-state `σ`. -/
def StoreConds (σ : AccountMap) (I : ExecutionEnv) : Prop :=
  sentAt σ I ≠ ⟨0⟩ ∧ (sentAt σ I).toNat + P_contract < 2 ^ 256 ∧
    (sentAt σ I).toNat + P_contract < (argTime I).toNat

/-- The account map after the successful `SSTORE`. -/
def finalMap (σ₂ : AccountMap) (I : ExecutionEnv) : AccountMap :=
  sstoreAccountMap I.codeOwner σ₂ (expiredSlot (argHash I))
    (setTrueWord (storageWord σ₂ I.codeOwner (expiredSlot (argHash I))))

theorem seg_store {σ σ₂ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256}
    {aw : UInt256} {k C : ℕ} {rest : List UInt256} {o₂ : ByteArray}
    (hst2 : accountStorageStateEq σ σ₂)
    (h : RD l2tol2Runtime I g (initState σ σ₀ g A I) (UInt256.ofNat 2103)
        [argTime I, argHash I, UInt256.ofNat 567, expireSelector]
        (wordsMem (⟨0⟩ :: ⟨0⟩ :: rest)) aw o₂ σ₂ k C) :
    (RDrev l2tol2Runtime g (initState σ σ₀ g A I) ∧ ¬ StoreConds σ I) ∨
    (StoreConds σ I ∧ I.perm = false ∧ RDstatic l2tol2Runtime g (initState σ σ₀ g A I)) ∨
    (StoreConds σ I ∧ I.perm = true ∧
      RDret l2tol2Runtime g (initState σ σ₀ g A I) (finalMap σ₂ I) ByteArray.empty) := by
  set H := argHash I with hH
  set t := argTime I with ht
  have hM1 : (UInt256.toByteArray H).write 0 (wordsMem (⟨0⟩ :: ⟨0⟩ :: rest))
      (⟨0⟩ : UInt256).toNat 32 = wordsMem (H :: ⟨0⟩ :: rest) :=
    wordsMem_write (⟨0⟩ :: ⟨0⟩ :: rest) 0 (by simp) H _ rfl
  have hM2 : (UInt256.toByteArray (UInt256.ofNat 3)).write 0 (wordsMem (H :: ⟨0⟩ :: rest))
      (UInt256.ofNat 32).toNat 32 = wordsMem (H :: UInt256.ofNat 3 :: rest) :=
    wordsMem_write (H :: ⟨0⟩ :: rest) 1 (by simp) (UInt256.ofNat 3) _ rfl
  have hK : keccakWord ⟨0⟩ (UInt256.ofNat 64) (wordsMem (H :: UInt256.ofNat 3 :: rest)) =
      sentAtSlot H := keccakWord_wordsMem _ _ _
  have hS : storageWord σ₂ I.codeOwner (sentAtSlot H) = sentAt σ I :=
    storageWord_eq_of_storageEq hst2 _ _
  by_cases hs0 : sentAt σ I = ⟨0⟩
  · -- InvalidMessage
    left
    refine ⟨?_, fun hc => hc.1 hs0⟩
    obtain ⟨k1, C1, r1⟩ := l2tol2_block_2103_fallthrough (by simp)
      (by rw [hM1, hM2, hK, optWord_eq, hS, hs0]; rfl) h
    exact l2tol2_block_2126 (by simp [l2tol2_block_2103_fallthrough_stack]) r1
  obtain ⟨k1, C1, r1⟩ := l2tol2_block_2103_taken (by simp)
    (by rw [hM1, hM2, hK, optWord_eq, hS]; exact (Words.sub_zero_ne0 _).mpr hs0) (by jump_dest) h
  simp only [l2tol2_block_2103_taken_stack, l2tol2_block_2103_taken_memory, hM1, hM2, hK,
    optWord_eq, hS] at r1
  have r2 := l2tol2_block_2175 (by simp) (by jump_dest) r1
  simp only [l2tol2_block_2175_stack] at r2
  by_cases hov : (sentAt σ I).toNat + P_contract < 2 ^ 256
  swap
  · -- checked-add overflow: Panic(0x11)
    left
    refine ⟨?_, fun hc => hov hc.2.1⟩
    have r3 := l2tol2_block_4603_fallthrough (by simp)
      (by
        by_contra hne
        exact hov ((Words.noOverflow_iff _ P_contract (by decide)).mp hne)) r2
    simp only [l2tol2_block_4603_fallthrough_stack] at r3
    have r4 := l2tol2_block_4615 (by simp) (by jump_dest) r3
    exact l2tol2_block_4366 (by simp [l2tol2_block_4615_stack]) r4
  have r3 := l2tol2_block_4603_taken (by simp)
    ((Words.noOverflow_iff _ P_contract (by decide)).mpr hov) (by jump_dest) r2
  simp only [l2tol2_block_4603_taken_stack] at r3
  have r4 := l2tol2_block_3588 (by simp) (by jump_dest) r3
  simp only [l2tol2_block_3588_stack] at r4
  by_cases hexp : (sentAt σ I).toNat + P_contract < t.toNat
  swap
  · -- MessageNotExpired
    left
    refine ⟨?_, fun hc => hexp hc.2.2⟩
    have r5 := l2tol2_block_2188_fallthrough (by simp)
      (by
        by_contra hne
        exact hexp ((Words.expired_iff _ _ P_contract (by decide) hov).mp hne)) r4
    exact l2tol2_block_2195 (by simp [l2tol2_block_2188_fallthrough_stack]) r5
  have hc : StoreConds σ I := ⟨hs0, hov, hexp⟩
  have r5 := l2tol2_block_2188_taken (by simp)
    ((Words.expired_iff _ _ P_contract (by decide) hov).mpr hexp) (by jump_dest) r4
  simp only [l2tol2_block_2188_taken_stack] at r5
  cases hp : I.perm with
  | false =>
    right; left
    refine ⟨hc, rfl, ?_⟩
    have q := evm_run r5 with [jumpdest, push0, dup4, dup2]
    have q5 := RD.genMstore q (by native_decide) (by evm_ov)
    have q6 := evm_run q5 with [push1 (UInt256.ofNat 4), push1 (UInt256.ofNat 32)]
    have q8 := RD.genMstore q6 (by native_decide) (by evm_ov)
    have q11 := evm_run q8 with [push1 (UInt256.ofNat 64), swap1, dup2, swap1]
    have q13 := RD.genKeccak256 q11 (by native_decide) (by evm_ov)
    have q14 := evm_run q13 with [dup1]
    obtain ⟨_, _, q15⟩ := RD.sload q14 (by native_decide) (by evm_ov)
    have q16 := q15.pushConst (UInt256.ofNat
      115792089237316195423570985008687907853269984665640564039457584007913129639680)
      (width := 32) (op := .PUSH32) (by decide) (by native_decide) (by evm_ov)
    have q20 := evm_run q16 with [and, push1 (UInt256.ofNat 1), or, swap1]
    exact RD.sstoreStatic q20 hp (by native_decide) (by evm_ov)
  | true =>
    right; right
    refine ⟨hc, rfl, ?_⟩
    obtain ⟨k6, C6, r6⟩ := l2tol2_block_2244 (by simp) hp (by jump_dest) r5
    have hN1 : (UInt256.toByteArray H).write 0 (wordsMem (H :: UInt256.ofNat 3 :: rest))
        (⟨0⟩ : UInt256).toNat 32 = wordsMem (H :: UInt256.ofNat 3 :: rest) :=
      wordsMem_write (H :: UInt256.ofNat 3 :: rest) 0 (by simp) H _ rfl
    have hN2 : (UInt256.toByteArray (UInt256.ofNat 4)).write 0
        (wordsMem (H :: UInt256.ofNat 3 :: rest))
        (UInt256.ofNat 32).toNat 32 = wordsMem (H :: UInt256.ofNat 4 :: rest) :=
      wordsMem_write (H :: UInt256.ofNat 3 :: rest) 1 (by simp) (UInt256.ofNat 4) _ rfl
    have hK2 : keccakWord ⟨0⟩ (UInt256.ofNat 64)
        (wordsMem (H :: UInt256.ofNat 4 :: rest)) = expiredSlot H :=
      keccakWord_wordsMem _ _ _
    simp only [l2tol2_block_2244_stack, l2tol2_block_2244_memory, hN1, hN2, hK2,
      optWord_eq] at r6
    have r7 := l2tol2_block_2350 (by simp) hp (by jump_dest) r6
    simp only [l2tol2_block_2350_stack] at r7
    have r8 := l2tol2_block_567 (by simp) r7
    exact r8

end ExpiryEvm
