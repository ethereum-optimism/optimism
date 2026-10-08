import Reasoning.Solc
import Reasoning.EVMWord

/-! # Small 256-bit word facts used to discharge branch conditions

Each EVM comparison (`LT`, `GT`, `EQ`, `ISZERO`, `SLT`) is `Bool.toUInt256` of a decidable
proposition; these lemmas turn the generated summaries' `… = 0` / `… ≠ 0` branch conditions into
statements about `toNat`. -/

namespace ExpiryEvm.Words

open Ethereum

theorem toU_ne0 {b : Bool} : Bool.toUInt256 b ≠ UInt256.ofNat 0 ↔ b = true := by
  cases b <;> decide

theorem toU_eq0 {b : Bool} : Bool.toUInt256 b = UInt256.ofNat 0 ↔ b = false := by
  cases b <;> decide

theorem ext_iff {a b : UInt256} : a = b ↔ a.toNat = b.toNat := by
  constructor
  · intro h; rw [h]
  · intro h; cases a; cases b; simp only [UInt256.toNat] at h; congr; exact Fin.ext h

theorem eq0_iff {a : UInt256} : UInt256.eq0 a = true ↔ a = ⟨0⟩ := by
  cases a with
  | mk v =>
    simp only [UInt256.eq0]
    constructor
    · intro h
      have : (v == (0 : Fin UInt256.size)) = true := h
      simp at this; subst this; rfl
    · intro h; cases h; rfl

theorem isZero_ne0 {a : UInt256} : UInt256.isZero a ≠ UInt256.ofNat 0 ↔ a = ⟨0⟩ := by
  unfold UInt256.isZero UInt256.fromBool; rw [toU_ne0, eq0_iff]

theorem isZero_eq0 {a : UInt256} : UInt256.isZero a = UInt256.ofNat 0 ↔ a ≠ ⟨0⟩ := by
  unfold UInt256.isZero UInt256.fromBool; rw [toU_eq0]
  constructor
  · intro h hc; rw [← eq0_iff] at hc; rw [h] at hc; exact Bool.false_ne_true hc
  · intro h; cases hb : UInt256.eq0 a
    · rfl
    · exact absurd (eq0_iff.mp hb) h

theorem lt_eq0 {a b : UInt256} : UInt256.lt a b = UInt256.ofNat 0 ↔ b.toNat ≤ a.toNat := by
  unfold UInt256.lt UInt256.fromBool; rw [toU_eq0]
  simp only [decide_eq_false_iff_not]
  show ¬ a.val < b.val ↔ b.val.val ≤ a.val.val
  rw [Fin.lt_def]; exact Nat.not_lt

theorem gt_eq0 {a b : UInt256} : UInt256.gt a b = UInt256.ofNat 0 ↔ a.toNat ≤ b.toNat := by
  unfold UInt256.gt UInt256.fromBool; rw [toU_eq0]
  simp only [decide_eq_false_iff_not]
  show ¬ b.val < a.val ↔ a.val.val ≤ b.val.val
  rw [Fin.lt_def]; exact Nat.not_lt

theorem gt_ne0 {a b : UInt256} : UInt256.gt a b ≠ UInt256.ofNat 0 ↔ b.toNat < a.toNat := by
  rw [Ne, gt_eq0]; omega

theorem eq_ne0 {a b : UInt256} : UInt256.eq a b ≠ UInt256.ofNat 0 ↔ a = b := by
  unfold UInt256.eq UInt256.fromBool; rw [toU_ne0]; simp

theorem eq_eq0 {a b : UInt256} : UInt256.eq a b = UInt256.ofNat 0 ↔ a ≠ b := by
  unfold UInt256.eq UInt256.fromBool; rw [toU_eq0]; simp

theorem toNat_ofNat {n : ℕ} (h : n < UInt256.size) : (UInt256.ofNat n).toNat = n :=
  UInt256.toNat_ofNat_of_lt h

theorem toNat_lt (a : UInt256) : a.toNat < UInt256.size := a.val.isLt

theorem size_eq : UInt256.size = 2 ^ 256 := by decide

theorem toNat_add (a b : UInt256) : (a + b).toNat = (a.toNat + b.toNat) % 2 ^ 256 := by
  show (a.val + b.val).val = _
  rw [Fin.val_add]; rfl

theorem toNat_sub_of_le {a b : UInt256} (h : b.toNat ≤ a.toNat) :
    (UInt256.sub a b).toNat = a.toNat - b.toNat := by
  show (a.val - b.val).val = _
  rw [Fin.sub_def]
  simp only [UInt256.toNat] at h ⊢
  have ha := a.val.isLt
  rw [show UInt256.size - b.val.val + a.val.val = UInt256.size + (a.val.val - b.val.val) by omega]
  rw [Nat.add_mod_left, Nat.mod_eq_of_lt (by omega)]

end ExpiryEvm.Words

namespace ExpiryEvm.Words

open Ethereum

theorem sub_zero_ne0 (x : UInt256) : UInt256.sub ⟨0⟩ x ≠ UInt256.ofNat 0 ↔ x ≠ ⟨0⟩ := by
  constructor
  · intro h hx; apply h; rw [hx]; rfl
  · intro hx h
    apply hx
    rw [ext_iff] at h ⊢
    have h' : (UInt256.sub ⟨0⟩ x).toNat = 0 := h
    show x.toNat = 0
    by_contra hne
    have hlt := toNat_lt x
    have : (UInt256.sub ⟨0⟩ x).toNat = UInt256.size - x.toNat := by
      show ((0 : Fin UInt256.size) - x.val).val = _
      rw [Fin.sub_def]
      simp only [Fin.val_zero, Nat.add_zero, UInt256.toNat]
      exact Nat.mod_eq_of_lt (by unfold UInt256.toNat at hne; omega)
    omega

theorem noOverflow_iff (s : UInt256) (n : ℕ) (hn : n < 2 ^ 256) :
    UInt256.isZero (UInt256.gt s (UInt256.ofNat n + s)) ≠ UInt256.ofNat 0 ↔ s.toNat + n < 2 ^ 256 := by
  rw [isZero_ne0]
  have hs := toNat_lt s
  rw [size_eq] at hs
  have hsum : (UInt256.ofNat n + s).toNat = (n + s.toNat) % 2 ^ 256 := by
    rw [toNat_add, toNat_ofNat (by rw [size_eq]; exact hn)]
  constructor
  · intro h
    have h' := gt_eq0.mp h
    rw [hsum] at h'
    by_contra hov
    have : (n + s.toNat) % 2 ^ 256 = n + s.toNat - 2 ^ 256 := by
      rw [Nat.mod_eq_sub_mod (by omega), Nat.mod_eq_of_lt (by omega)]
    omega
  · intro h
    apply gt_eq0.mpr
    rw [hsum, Nat.mod_eq_of_lt (by omega)]
    omega

theorem expired_iff (t s : UInt256) (n : ℕ) (hn : n < 2 ^ 256) (h : s.toNat + n < 2 ^ 256) :
    UInt256.gt t (UInt256.ofNat n + s) ≠ UInt256.ofNat 0 ↔ s.toNat + n < t.toNat := by
  rw [gt_ne0, toNat_add, toNat_ofNat (by rw [size_eq]; exact hn), Nat.mod_eq_of_lt (by omega)]
  omega

end ExpiryEvm.Words

namespace ExpiryEvm.Words

open Ethereum

/-- solc's ABI length check for two static words: `CALLDATASIZE ≥ 4` (dispatcher) and
    `!(CALLDATASIZE - 4 <ₛ 64)`. -/
theorem calldata_ok_iff (n : ℕ) (hn : n < 2 ^ 256) :
    (UInt256.lt (UInt256.ofNat n) (UInt256.ofNat 4) = UInt256.ofNat 0 ∧
      UInt256.isZero (UInt256.slt (UInt256.sub (UInt256.ofNat n) (UInt256.ofNat 4))
        (UInt256.ofNat 64)) ≠ UInt256.ofNat 0) ↔ (68 ≤ n ∧ n < 2 ^ 255 + 4) := by
  have hn' : n < UInt256.size := by rw [size_eq]; exact hn
  rw [lt_eq0, isZero_ne0, toNat_ofNat hn', toNat_ofNat (by decide)]
  constructor
  · rintro ⟨h4, hs⟩
    have hsub : (UInt256.sub (UInt256.ofNat n) (UInt256.ofNat 4)).toNat = n - 4 := by
      rw [toNat_sub_of_le (by rw [toNat_ofNat hn', toNat_ofNat (by decide)]; exact h4),
        toNat_ofNat hn', toNat_ofNat (by decide)]
    have hs' : UInt256.sltBool (UInt256.sub (UInt256.ofNat n) (UInt256.ofNat 4)) (UInt256.ofNat 64)
        = false := by
      unfold UInt256.slt UInt256.fromBool at hs
      cases hb : UInt256.sltBool (UInt256.sub (UInt256.ofNat n) (UInt256.ofNat 4)) (UInt256.ofNat 64)
      · rfl
      · rw [hb] at hs; exact absurd hs (by decide)
    unfold UInt256.sltBool at hs'
    rw [hsub, show (UInt256.ofNat 64).toNat = 64 by decide] at hs'
    by_cases hbig : n - 4 ≥ 2 ^ 255
    · rw [if_pos hbig, if_neg (by norm_num)] at hs'; exact absurd hs' (by decide)
    · rw [if_neg hbig, if_neg (by norm_num)] at hs'
      have : ¬ (UInt256.sub (UInt256.ofNat n) (UInt256.ofNat 4)) < UInt256.ofNat 64 := by
        simpa using hs'
      have h64 : ¬ n - 4 < 64 := by
        intro hc; apply this
        show (UInt256.sub (UInt256.ofNat n) (UInt256.ofNat 4)).val < (UInt256.ofNat 64).val
        rw [Fin.lt_def]
        have := hsub; unfold UInt256.toNat at this; rw [this]
        show n - 4 < (UInt256.ofNat 64).toNat
        rw [show (UInt256.ofNat 64).toNat = 64 by decide]; exact hc
      omega
  · rintro ⟨h68, hbig⟩
    refine ⟨by omega, ?_⟩
    have hsub : (UInt256.sub (UInt256.ofNat n) (UInt256.ofNat 4)).toNat = n - 4 := by
      rw [toNat_sub_of_le (by rw [toNat_ofNat hn', toNat_ofNat (by decide)]; omega),
        toNat_ofNat hn', toNat_ofNat (by decide)]
    unfold UInt256.slt UInt256.fromBool
    have : UInt256.sltBool (UInt256.sub (UInt256.ofNat n) (UInt256.ofNat 4)) (UInt256.ofNat 64)
        = false := by
      unfold UInt256.sltBool
      rw [hsub, show (UInt256.ofNat 64).toNat = 64 by decide, if_neg (by omega),
        if_neg (by norm_num)]
      simp only [decide_eq_false_iff_not]
      show ¬ (UInt256.sub (UInt256.ofNat n) (UInt256.ofNat 4)).val < (UInt256.ofNat 64).val
      rw [Fin.lt_def]
      have := hsub; unfold UInt256.toNat at this; rw [this]
      show ¬ n - 4 < (UInt256.ofNat 64).toNat
      rw [show (UInt256.ofNat 64).toNat = 64 by decide]; omega
    rw [this]; rfl

end ExpiryEvm.Words

namespace ExpiryEvm.Words

open Ethereum

/-- solc's allocation rounding `(n + 31) & ~31`. -/
theorem round_up (n : ℕ) (hn : n < 2 ^ 200) :
    UInt256.land (UInt256.ofNat n + UInt256.ofNat 31) (UInt256.lnot (UInt256.ofNat 31)) =
      UInt256.ofNat (32 * ((n + 31) / 32)) := by
  rw [ext_iff, Reasoning.Theory.uland_toNat, show UInt256.lnot (UInt256.ofNat 31) =
    UInt256.lnot ⟨31⟩ from rfl, Reasoning.Theory.lnot31_toNat, toNat_add,
    toNat_ofNat (by rw [size_eq]; omega), toNat_ofNat (by decide),
    toNat_ofNat (by rw [size_eq]; omega), Nat.mod_eq_of_lt (by omega),
    Reasoning.Theory.nat_land_mask _ (by omega)]

theorem min32_toNat (n : ℕ) (h1 : 32 ≤ n) (h2 : n < 2 ^ 256) :
    (min (UInt256.ofNat 32) (UInt256.ofNat n)).toNat = 32 := by
  have hle : UInt256.ofNat 32 ≤ UInt256.ofNat n := by
    show (UInt256.ofNat 32).val ≤ (UInt256.ofNat n).val
    rw [Fin.le_def]
    show (UInt256.ofNat 32).toNat ≤ (UInt256.ofNat n).toNat
    rw [toNat_ofNat (by decide), toNat_ofNat (by rw [size_eq]; exact h2)]; exact h1
  have hm : min (UInt256.ofNat 32) (UInt256.ofNat n) = UInt256.ofNat 32 := by
    first
    | exact min_eq_left hle
    | (show (if UInt256.ofNat 32 ≤ UInt256.ofNat n then _ else _) = _; rw [if_pos hle])
    | (simp only [Min.min, minOfLe]; rw [if_pos hle])
  show (min (UInt256.ofNat 32) (UInt256.ofNat n)).toNat = 32
  rw [hm]; decide

theorem add_sub_cancel' (a : ℕ) (n : ℕ) (h : a + n < 2 ^ 256) :
    UInt256.sub (UInt256.ofNat a + UInt256.ofNat n) (UInt256.ofNat a) = UInt256.ofNat n := by
  have ha : (UInt256.ofNat a).toNat = a := toNat_ofNat (by rw [size_eq]; omega)
  have hn : (UInt256.ofNat n).toNat = n := toNat_ofNat (by rw [size_eq]; omega)
  have hs : (UInt256.ofNat a + UInt256.ofNat n).toNat = a + n := by
    rw [toNat_add, ha, hn, Nat.mod_eq_of_lt h]
  rw [ext_iff, toNat_sub_of_le (by rw [hs, ha]; omega), hs, ha, hn]; omega

theorem ofNat_add (a b : ℕ) (h : a + b < 2 ^ 256) :
    UInt256.ofNat a + UInt256.ofNat b = UInt256.ofNat (a + b) := by
  rw [ext_iff, toNat_add, toNat_ofNat (by rw [size_eq]; omega), toNat_ofNat (by rw [size_eq]; omega),
    toNat_ofNat (by rw [size_eq]; omega), Nat.mod_eq_of_lt h]

/-- The ABI decoder's return-data length check passes for `n ≥ 32` bytes. -/
theorem retlen_ok (n : ℕ) (h1 : 32 ≤ n) (h2 : n < 2 ^ 255) :
    UInt256.isZero (UInt256.slt (UInt256.ofNat n) (UInt256.ofNat 32)) ≠ UInt256.ofNat 0 := by
  rw [isZero_ne0]
  unfold UInt256.slt UInt256.fromBool
  have : UInt256.sltBool (UInt256.ofNat n) (UInt256.ofNat 32) = false := by
    unfold UInt256.sltBool
    rw [toNat_ofNat (by rw [size_eq]; omega), show (UInt256.ofNat 32).toNat = 32 by decide,
      if_neg (by omega), if_neg (by norm_num)]
    simp only [decide_eq_false_iff_not]
    show ¬ (UInt256.ofNat n).val < (UInt256.ofNat 32).val
    rw [Fin.lt_def]
    show ¬ (UInt256.ofNat n).toNat < (UInt256.ofNat 32).toNat
    rw [toNat_ofNat (by rw [size_eq]; omega), show (UInt256.ofNat 32).toNat = 32 by decide]; omega
  rw [this]; rfl

end ExpiryEvm.Words

namespace ExpiryEvm.Words

open Ethereum

theorem add_sub_cancel_left' (n a : ℕ) (h : n + a < 2 ^ 256) :
    UInt256.sub (UInt256.ofNat n + UInt256.ofNat a) (UInt256.ofNat a) = UInt256.ofNat n := by
  have ha : (UInt256.ofNat a).toNat = a := toNat_ofNat (by rw [size_eq]; omega)
  have hn : (UInt256.ofNat n).toNat = n := toNat_ofNat (by rw [size_eq]; omega)
  have hs : (UInt256.ofNat n + UInt256.ofNat a).toNat = n + a := by
    rw [toNat_add, ha, hn, Nat.mod_eq_of_lt h]
  rw [ext_iff, toNat_sub_of_le (by rw [hs, ha]; omega), hs, ha, hn]; omega

end ExpiryEvm.Words
