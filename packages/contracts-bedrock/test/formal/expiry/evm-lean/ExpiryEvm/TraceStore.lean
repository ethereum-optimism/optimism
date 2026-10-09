import ExpiryEvm.KernelRun
import ExpiryEvm.TraceCall2

/-! # Trace segment 4: `expiredMessages` early return, `sentMessageTimestamps` read, expiry check,
`expiredMessages` write, event, `STOP` (pc 2739 → end) -/

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

/-- The storage conditions of a first expiry on the pre-state `σ` (`Spec.FreshConds`). -/
abbrev StoreConds (σ : AccountMap) (I : ExecutionEnv) : Prop := FreshConds σ I

/-- The account map after the successful `SSTORE`. -/
def finalMap (σ₂ : AccountMap) (I : ExecutionEnv) : AccountMap :=
  sstoreAccountMap I.codeOwner σ₂ (expiredSlot (argHash I))
    (setTrueWord (storageWord σ₂ I.codeOwner (expiredSlot (argHash I))))

theorem noOverflow_iff' (s p : UInt256) :
    UInt256.isZero (UInt256.gt s (p + s)) ≠ UInt256.ofNat 0 ↔ s.toNat + p.toNat < 2 ^ 256 := by
  have h := Words.noOverflow_iff s p.toNat (by have := Words.toNat_lt p; rwa [Words.size_eq] at this)
  rwa [ofNat_toNat] at h

theorem expired_iff' (t s p : UInt256) (h : s.toNat + p.toNat < 2 ^ 256) :
    UInt256.gt t (p + s) ≠ UInt256.ofNat 0 ↔ s.toNat + p.toNat < t.toNat := by
  have h' := Words.expired_iff t s p.toNat (by have := Words.toNat_lt p; rwa [Words.size_eq] at this) h
  rwa [ofNat_toNat] at h'

theorem alreadyExpired_iff (σ : AccountMap) (I : ExecutionEnv) :
    UInt256.land (UInt256.ofNat 255) (expiredWord σ I) ≠ ⟨0⟩ ↔ AlreadyExpired σ I := by
  unfold AlreadyExpired; rw [u256_land_comm]

theorem seg_store {σ σ₂ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256}
    {aw : UInt256} {k C : ℕ} {rest : List UInt256} {o₂ : ByteArray}
    (hst2 : accountStorageStateEq σ σ₂)
    (h : RD l2tol2Runtime I g (initState σ σ₀ g A I) (UInt256.ofNat 2739)
        [argTime I, argHash I, UInt256.ofNat 634, expireSelector]
        (wordsMem (⟨0⟩ :: ⟨0⟩ :: rest)) aw o₂ σ₂ k C) :
    (RDrev l2tol2Runtime g (initState σ σ₀ g A I) ∧ ¬ AlreadyExpired σ I ∧ ¬ StoreConds σ I) ∨
    (AlreadyExpired σ I ∧ RDret l2tol2Runtime g (initState σ σ₀ g A I) σ₂ ByteArray.empty) ∨
    (¬ AlreadyExpired σ I ∧ StoreConds σ I ∧ I.perm = false ∧
      RDstatic l2tol2Runtime g (initState σ σ₀ g A I)) ∨
    (¬ AlreadyExpired σ I ∧ StoreConds σ I ∧ I.perm = true ∧
      RDret l2tol2Runtime g (initState σ σ₀ g A I) (finalMap σ₂ I) ByteArray.empty) := by
  set H := argHash I with hH
  set t := argTime I with ht
  have hM1 : (UInt256.toByteArray H).write 0 (wordsMem (⟨0⟩ :: ⟨0⟩ :: rest))
      (⟨0⟩ : UInt256).toNat 32 = wordsMem (H :: ⟨0⟩ :: rest) :=
    wordsMem_write (⟨0⟩ :: ⟨0⟩ :: rest) 0 (by simp) H _ rfl
  have hM4 : (UInt256.toByteArray (UInt256.ofNat 4)).write 0 (wordsMem (H :: ⟨0⟩ :: rest))
      (UInt256.ofNat 32).toNat 32 = wordsMem (H :: UInt256.ofNat 4 :: rest) :=
    wordsMem_write (H :: ⟨0⟩ :: rest) 1 (by simp) (UInt256.ofNat 4) _ rfl
  have hKE : keccakWord ⟨0⟩ (UInt256.ofNat 64) (wordsMem (H :: UInt256.ofNat 4 :: rest)) =
      expiredSlot H := keccakWord_wordsMem _ _ _
  have hE : storageWord σ₂ I.codeOwner (expiredSlot H) = expiredWord σ I :=
    storageWord_eq_of_storageEq hst2 _ _
  by_cases hae : AlreadyExpired σ I
  · -- `expiredMessages[H]` already true: early return, no storage write, no event
    right; left
    refine ⟨hae, ?_⟩
    obtain ⟨k0, C0, r0⟩ := l2tol2_block_2739_fallthrough (by simp)
      (by
        rw [hM1, hM4, hKE, optWord_eq, hE]
        exact Words.isZero_eq0.mpr ((alreadyExpired_iff σ I).mpr hae)) h
    have r1 := l2tol2_block_2762 (by simp) (by kjump_dest) r0
    exact l2tol2_block_634 (by simp [l2tol2_block_2762_stack]) r1
  obtain ⟨k0, C0, r0⟩ := l2tol2_block_2739_taken (by simp)
    (by
      rw [hM1, hM4, hKE, optWord_eq, hE]
      refine Words.isZero_ne0.mpr ?_
      by_contra hne
      exact hae ((alreadyExpired_iff σ I).mp hne)) (by kjump_dest) h
  simp only [l2tol2_block_2739_taken_memory, hM1, hM4] at r0
  have hM1' : (UInt256.toByteArray H).write 0 (wordsMem (H :: UInt256.ofNat 4 :: rest))
      (⟨0⟩ : UInt256).toNat 32 = wordsMem (H :: UInt256.ofNat 4 :: rest) :=
    wordsMem_write (H :: UInt256.ofNat 4 :: rest) 0 (by simp) H _ rfl
  have hM2 : (UInt256.toByteArray (UInt256.ofNat 3)).write 0
      (wordsMem (H :: UInt256.ofNat 4 :: rest))
      (UInt256.ofNat 32).toNat 32 = wordsMem (H :: UInt256.ofNat 3 :: rest) :=
    wordsMem_write (H :: UInt256.ofNat 4 :: rest) 1 (by simp) (UInt256.ofNat 3) _ rfl
  have hK : keccakWord ⟨0⟩ (UInt256.ofNat 64) (wordsMem (H :: UInt256.ofNat 3 :: rest)) =
      sentAtSlot H := keccakWord_wordsMem _ _ _
  have hS : storageWord σ₂ I.codeOwner (sentAtSlot H) = sentAt σ I :=
    storageWord_eq_of_storageEq hst2 _ _
  by_cases hs0 : sentAt σ I = ⟨0⟩
  · -- InvalidMessage
    left
    refine ⟨?_, hae, fun hc => hc.1 hs0⟩
    obtain ⟨k1, C1, r1⟩ := l2tol2_block_2765_fallthrough (by simp)
      (by rw [hM1', hM2, hK, optWord_eq, hS, hs0]; rfl) r0
    exact l2tol2_block_2788 (by simp [l2tol2_block_2765_fallthrough_stack]) r1
  obtain ⟨k1, C1, r1⟩ := l2tol2_block_2765_taken (by simp)
    (by rw [hM1', hM2, hK, optWord_eq, hS]; exact (Words.sub_zero_ne0 _).mpr hs0) (by kjump_dest) r0
  simp only [l2tol2_block_2765_taken_stack, l2tol2_block_2765_taken_memory, hM1', hM2, hK,
    optWord_eq, hS] at r1
  obtain ⟨k2, C2, r2⟩ := l2tol2_block_2837 (by simp) (by kjump_dest) r1
  have hP : storageWord σ₂ I.codeOwner (UInt256.ofNat 5) = periodWord σ I :=
    storageWord_eq_of_storageEq hst2 _ _
  simp only [l2tol2_block_2837_stack, optWord_eq, hP] at r2
  by_cases hov : (sentAt σ I).toNat + (periodWord σ I).toNat < 2 ^ 256
  swap
  · -- checked-add overflow: Panic(0x11)
    left
    refine ⟨?_, hae, fun hc => hov hc.2.1⟩
    have r3 := l2tol2_block_5998_fallthrough (by simp)
      (by
        by_contra hne
        exact hov ((noOverflow_iff' _ (periodWord σ I)).mp hne)) r2
    simp only [l2tol2_block_5998_fallthrough_stack] at r3
    have r4 := l2tol2_block_6010 (by simp) (by kjump_dest) r3
    exact l2tol2_block_5738 (by simp [l2tol2_block_6010_stack]) r4
  have r3 := l2tol2_block_5998_taken (by simp)
    ((noOverflow_iff' _ (periodWord σ I)).mpr hov) (by kjump_dest) r2
  simp only [l2tol2_block_5998_taken_stack] at r3
  have r4 := l2tol2_block_4829 (by simp) (by kjump_dest) r3
  simp only [l2tol2_block_4829_stack] at r4
  by_cases hexp : (sentAt σ I).toNat + (periodWord σ I).toNat < t.toNat
  swap
  · -- MessageNotExpired
    left
    refine ⟨?_, hae, fun hc => hexp hc.2.2⟩
    have r5 := l2tol2_block_2850_fallthrough (by simp)
      (by
        by_contra hne
        exact hexp ((expired_iff' _ _ (periodWord σ I) hov).mp hne)) r4
    exact l2tol2_block_2857 (by simp [l2tol2_block_2850_fallthrough_stack]) r5
  have hc : StoreConds σ I := ⟨hs0, hov, hexp⟩
  have r5 := l2tol2_block_2850_taken (by simp)
    ((expired_iff' _ _ (periodWord σ I) hov).mpr hexp) (by kjump_dest) r4
  simp only [l2tol2_block_2850_taken_stack] at r5
  cases hp : I.perm with
  | false =>
    right; right; left
    refine ⟨hae, hc, rfl, ?_⟩
    have q := kevm_run r5 with [jumpdest, push0, dup4, dup2]
    have q5 := RD.genMstore q (by evm_kdecide) (by evm_ov)
    have q6 := kevm_run q5 with [push1 (UInt256.ofNat 4), push1 (UInt256.ofNat 32)]
    have q8 := RD.genMstore q6 (by evm_kdecide) (by evm_ov)
    have q11 := kevm_run q8 with [push1 (UInt256.ofNat 64), swap1, dup2, swap1]
    have q13 := RD.genKeccak256 q11 (by evm_kdecide) (by evm_ov)
    have q14 := kevm_run q13 with [dup1]
    obtain ⟨_, _, q15⟩ := RD.sload q14 (by evm_kdecide) (by evm_ov)
    have q16 := q15.pushConst (UInt256.ofNat
      115792089237316195423570985008687907853269984665640564039457584007913129639680)
      (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide) (by evm_ov)
    have q20 := kevm_run q16 with [and, push1 (UInt256.ofNat 1), or, swap1]
    exact RD.sstoreStatic q20 hp (by evm_kdecide) (by evm_ov)
  | true =>
    right; right; right
    refine ⟨hae, hc, rfl, ?_⟩
    obtain ⟨k6, C6, r6⟩ := l2tol2_block_2906 (by simp) hp (by kjump_dest) r5
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
    simp only [l2tol2_block_2906_stack, l2tol2_block_2906_memory, hN1, hN2, hK2,
      optWord_eq] at r6
    have r7 := l2tol2_block_3012 (by simp) hp (by kjump_dest) r6
    simp only [l2tol2_block_3012_stack] at r7
    have r8 := l2tol2_block_634 (by simp) r7
    exact r8

end ExpiryEvm
