import ExporterEvm.Words

/-! # More word facts for the exporter's branch conditions (ABI decoder, loop guards) -/

namespace ExporterEvm.Words

open Ethereum

theorem lt_iff {a b : UInt256} : a < b ↔ a.toNat < b.toNat := by
  show a.val < b.val ↔ _
  rw [Fin.lt_def]; rfl

theorem lt_ne0 {a b : UInt256} : UInt256.lt a b ≠ UInt256.ofNat 0 ↔ a.toNat < b.toNat := by
  rw [Ne, lt_eq0]; omega

theorem isZero_lt_ne0 {a b : UInt256} :
    UInt256.isZero (UInt256.lt a b) ≠ UInt256.ofNat 0 ↔ b.toNat ≤ a.toNat := by
  rw [isZero_ne0, show (⟨0⟩ : UInt256) = UInt256.ofNat 0 from rfl, lt_eq0]

theorem isZero_lt_eq0 {a b : UInt256} :
    UInt256.isZero (UInt256.lt a b) = UInt256.ofNat 0 ↔ a.toNat < b.toNat := by
  rw [isZero_eq0, show (⟨0⟩ : UInt256) = UInt256.ofNat 0 from rfl, Ne, lt_eq0]; omega

theorem isZero_gt_ne0 {a b : UInt256} :
    UInt256.isZero (UInt256.gt a b) ≠ UInt256.ofNat 0 ↔ a.toNat ≤ b.toNat := by
  rw [isZero_ne0, show (⟨0⟩ : UInt256) = UInt256.ofNat 0 from rfl, gt_eq0]

theorem isZero_gt_eq0 {a b : UInt256} :
    UInt256.isZero (UInt256.gt a b) = UInt256.ofNat 0 ↔ b.toNat < a.toNat := by
  rw [isZero_eq0, show (⟨0⟩ : UInt256) = UInt256.ofNat 0 from rfl, Ne, gt_eq0]; omega

/-- `SLT` on two words below `2^255` is `LT`. -/
theorem sltBool_small {a b : UInt256} (ha : a.toNat < 2 ^ 255) (hb : b.toNat < 2 ^ 255) :
    UInt256.sltBool a b = decide (a.toNat < b.toNat) := by
  unfold UInt256.sltBool
  rw [if_neg (by omega), if_neg (by omega)]
  simp only [decide_eq_decide]
  exact lt_iff

theorem slt_small_ne0 {a b : UInt256} (ha : a.toNat < 2 ^ 255) (hb : b.toNat < 2 ^ 255) :
    UInt256.slt a b ≠ UInt256.ofNat 0 ↔ a.toNat < b.toNat := by
  unfold UInt256.slt UInt256.fromBool
  rw [sltBool_small ha hb, toU_ne0]; simp

theorem isZero_slt_small_ne0 {a b : UInt256} (ha : a.toNat < 2 ^ 255) (hb : b.toNat < 2 ^ 255) :
    UInt256.isZero (UInt256.slt a b) ≠ UInt256.ofNat 0 ↔ b.toNat ≤ a.toNat := by
  rw [isZero_ne0]
  unfold UInt256.slt UInt256.fromBool
  rw [sltBool_small ha hb]
  constructor
  · intro h
    by_contra hc
    rw [decide_eq_true (by omega)] at h
    exact absurd h (by decide)
  · intro h
    rw [decide_eq_false (by omega)]; rfl

theorem toNat_add_of_lt {a b : UInt256} (h : a.toNat + b.toNat < 2 ^ 256) :
    (a + b).toNat = a.toNat + b.toNat := by
  rw [toNat_add, Nat.mod_eq_of_lt h]

theorem toNat_ofNat_lt {n : ℕ} (h : n < 2 ^ 256) : (UInt256.ofNat n).toNat = n :=
  toNat_ofNat (by rw [size_eq]; exact h)

theorem eq_ofNat_toNat (a : UInt256) : a = UInt256.ofNat a.toNat := by
  rw [ext_iff, toNat_ofNat_lt (by have := toNat_lt a; rw [size_eq] at this; exact this)]

/-- solc's `uint32` validator `eq(x, and(x, 0xffffffff))`. -/
theorem uint32_clean_iff (x : UInt256) :
    UInt256.eq x (UInt256.land x (UInt256.ofNat 4294967295)) ≠ UInt256.ofNat 0 ↔
      x.toNat < 2 ^ 32 := by
  rw [eq_ne0]
  have hm : (UInt256.land x (UInt256.ofNat 4294967295)).toNat = x.toNat % 2 ^ 32 := by
    rw [land_toNat, toNat_ofNat (by decide), show (4294967295 : ℕ) = 2 ^ 32 - 1 by norm_num,
      Nat.and_two_pow_sub_one_eq_mod]
  constructor
  · intro h
    have := congrArg UInt256.toNat h
    rw [hm] at this
    rw [this]; exact Nat.mod_lt _ (by norm_num)
  · intro h
    rw [ext_iff, hm, Nat.mod_eq_of_lt h]

end ExporterEvm.Words
