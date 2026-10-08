import L1cdmEvm.OuterEntry
import L1cdmEvm.CallMem

/-! # Shared decode tails of the view calls (`abi_decode_address`/`abi_decode_bool` from memory) -/

namespace L1cdmEvm

set_option maxRecDepth 100000

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach l1cdmBlocks L1cdmEvm.SymMem

theorem clean_of_eq_land {w : UInt256}
    (h : UInt256.eq w (UInt256.land w (UInt256.ofNat 1461501637330902918203684832716283019655932542975))
      ≠ UInt256.ofNat 0) : CleanAddr w := by
  have := Words.eq_ne0.mp h
  have h2 := congrArg UInt256.toNat this
  rw [uland_toNat, show (UInt256.ofNat 1461501637330902918203684832716283019655932542975).toNat
    = 2 ^ 160 - 1 from rfl, Nat.and_two_pow_sub_one_eq_mod] at h2
  unfold CleanAddr
  rw [h2]; exact Nat.mod_lt _ (by norm_num)

/-- `abi_decode_address` from memory `[P, P + rds)` (pc 10155), then return to `ret`. -/
theorem tail_addr {I : ExecutionEnv} {g : Sat256} {s0 : State} {m : ByteArray} {aw : UInt256}
    {o : ByteArray} {σ : AccountMap} {k C : ℕ} {P w ret : UInt256} {R : List UInt256}
    (hR : R.length ≤ 1000) (hret : (D_J l1cdmRuntime 0).contains ret = true)
    (hosz : o.size < 2 ^ 32) (hload : 32 ≤ o.size → memLoad P m = w)
    (h : RD l1cdmRuntime I g s0 (UInt256.ofNat 10155) (P :: (P + UInt256.ofNat o.size) :: ret :: R)
      m aw o σ k C) :
    RDrev l1cdmRuntime g s0 ∨
    (32 ≤ o.size ∧ CleanAddr w ∧ ∃ aw' k' C', RD l1cdmRuntime I g s0 ret (w :: R) m aw' o σ k' C') := by
  by_cases h32 : 32 ≤ o.size
  swap
  · left
    have r1 := l1cdm_block_10155_fallthrough (by simp; omega)
      (by
        rw [u_sub_add_self]
        by_contra hc
        exact h32 ((slt_small _ (by omega)).mp hc)) h
    exact l1cdm_block_10169 (by simp [l1cdm_block_10155_fallthrough_stack]; omega) r1
  have r1 := l1cdm_block_10155_taken (by simp; omega)
    (by rw [u_sub_add_self]; exact (slt_small _ (by omega)).mpr h32) (by kjump_dest) h
  simp only [l1cdm_block_10155_taken_stack] at r1
  have r2 := l1cdm_block_10173 (by simp; omega) (by kjump_dest) r1
  simp only [l1cdm_block_10173_stack, hload h32] at r2
  by_cases hc : UInt256.eq w (UInt256.land w
      (UInt256.ofNat 1461501637330902918203684832716283019655932542975)) = UInt256.ofNat 0
  · left
    exact l1cdm_block_9334 (by simp; omega) (l1cdm_block_9304_fallthrough (by simp; omega) hc r2)
  have r3 := l1cdm_block_9304_taken (by simp; omega) hc (by kjump_dest) r2
  have r4 := l1cdm_block_9338 (by simp; omega) (by kjump_dest) r3
  simp only [l1cdm_block_9338_stack] at r4
  have r5 := l1cdm_block_8541 (by simp; omega) hret r4
  simp only [l1cdm_block_8541_stack] at r5
  exact Or.inr ⟨h32, clean_of_eq_land hc, _, _, _, r5⟩

/-- `abi_decode_bool` from memory `[P, P + rds)` (pc 10184), then return to `ret`. -/
theorem tail_bool {I : ExecutionEnv} {g : Sat256} {s0 : State} {m : ByteArray} {aw : UInt256}
    {o : ByteArray} {σ : AccountMap} {k C : ℕ} {P w ret : UInt256} {R : List UInt256}
    (hR : R.length ≤ 1000) (hret : (D_J l1cdmRuntime 0).contains ret = true)
    (hosz : o.size < 2 ^ 32) (hload : 32 ≤ o.size → memLoad P m = w)
    (h : RD l1cdmRuntime I g s0 (UInt256.ofNat 10184) (P :: (P + UInt256.ofNat o.size) :: ret :: R)
      m aw o σ k C) :
    RDrev l1cdmRuntime g s0 ∨
    (32 ≤ o.size ∧ UInt256.eq w (UInt256.isZero (UInt256.isZero w)) ≠ UInt256.ofNat 0 ∧
      ∃ aw' k' C', RD l1cdmRuntime I g s0 ret (w :: R) m aw' o σ k' C') := by
  by_cases h32 : 32 ≤ o.size
  swap
  · left
    have r6 := l1cdm_block_10184_fallthrough (by simp; omega)
      (by
        rw [u_sub_add_self]
        by_contra hc
        exact h32 ((slt_small _ (by omega)).mp hc)) h
    exact l1cdm_block_10198 (by simp [l1cdm_block_10184_fallthrough_stack]; omega) r6
  have r6 := l1cdm_block_10184_taken (by simp; omega)
    (by rw [u_sub_add_self]; exact (slt_small _ (by omega)).mpr h32) (by kjump_dest) h
  simp only [l1cdm_block_10184_taken_stack] at r6
  by_cases hbool : UInt256.eq w (UInt256.isZero (UInt256.isZero w)) = UInt256.ofNat 0
  · left
    have r7 := l1cdm_block_10202_fallthrough (by simp; omega) (by rw [hload h32]; exact hbool) r6
    exact l1cdm_block_10214 (by simp [l1cdm_block_10202_fallthrough_stack]; omega) r7
  have r7 := l1cdm_block_10202_taken (by simp; omega) (by rw [hload h32]; exact hbool) (by kjump_dest) r6
  simp only [l1cdm_block_10202_taken_stack, hload h32] at r7
  have r8 := l1cdm_block_8541 (by simp; omega) hret r7
  simp only [l1cdm_block_8541_stack] at r8
  exact Or.inr ⟨h32, hbool, _, _, _, r8⟩

end L1cdmEvm

namespace L1cdmEvm

set_option maxRecDepth 100000

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach l1cdmBlocks L1cdmEvm.SymMem

/-- pc 2855 with a nonzero flag (one of the three checks failed): revert. -/
theorem rev_2850 {I : ExecutionEnv} {g : Sat256} {s0 : State} {m : ByteArray} {aw : UInt256}
    {o : ByteArray} {σ : AccountMap} {k C : ℕ} {c : UInt256} {R : List UInt256}
    (hR : R.length ≤ 1000) (hc : c ≠ UInt256.ofNat 0)
    (h : RD l1cdmRuntime I g s0 (UInt256.ofNat 2855) (c :: R) m aw o σ k C) :
    RDrev l1cdmRuntime g s0 :=
  l1cdm_block_2861 (by simp [l1cdm_block_2855_fallthrough_stack]; omega)
    (l1cdm_block_2855_fallthrough (by simp; omega) (Words.isZero_eq0.mpr hc) h)

/-- The `||` chain of the three checks: a true disjunct at pc 2670 leads to the revert. -/
theorem rev_2665 {I : ExecutionEnv} {g : Sat256} {s0 : State} {m : ByteArray} {aw : UInt256}
    {o : ByteArray} {σ : AccountMap} {k C : ℕ} {c : UInt256} {R : List UInt256}
    (hR : R.length ≤ 1000) (hc : c ≠ UInt256.ofNat 0)
    (h : RD l1cdmRuntime I g s0 (UInt256.ofNat 2670) (c :: R) m aw o σ k C) :
    RDrev l1cdmRuntime g s0 :=
  rev_2850 hR hc (l1cdm_block_2670_taken (by simp; omega) hc (by kjump_dest) h)

theorem land_mask_clean {w : UInt256} (h : CleanAddr w) :
    UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975) w = w := by
  apply u_ext
  rw [uland_toNat, show (UInt256.ofNat 1461501637330902918203684832716283019655932542975).toNat
    = 2 ^ 160 - 1 from rfl, Nat.land_comm, Nat.and_two_pow_sub_one_eq_mod, Nat.mod_eq_of_lt h]

theorem source_clean (I : ExecutionEnv) : CleanAddr (UInt256.ofNat I.source.val) := by
  unfold CleanAddr
  rw [u_toNat_ofNat (by have := I.source.isLt; unfold AccountAddress.size at this; omega)]
  exact I.source.isLt

end L1cdmEvm

namespace L1cdmEvm

open Reasoning.Theory in
theorem land_clean_mask {w : Ethereum.UInt256} (h : CleanAddr w) :
    Ethereum.UInt256.land w (Ethereum.UInt256.ofNat 1461501637330902918203684832716283019655932542975) = w := by
  apply u_ext
  rw [uland_toNat, show (Ethereum.UInt256.ofNat 1461501637330902918203684832716283019655932542975).toNat
    = 2 ^ 160 - 1 from rfl, Nat.and_two_pow_sub_one_eq_mod, Nat.mod_eq_of_lt h]

end L1cdmEvm

namespace L1cdmEvm

open Reasoning.Theory in
theorem clean_land_mask (x : Ethereum.UInt256) :
    CleanAddr (Ethereum.UInt256.land (Ethereum.UInt256.ofNat 1461501637330902918203684832716283019655932542975) x) := by
  unfold CleanAddr
  rw [uland_toNat, show (Ethereum.UInt256.ofNat 1461501637330902918203684832716283019655932542975).toNat
    = 2 ^ 160 - 1 from rfl]
  exact lt_of_le_of_lt Nat.and_le_left (by norm_num)

end L1cdmEvm
