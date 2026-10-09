import L1cdmEvm.Spec
import L1cdmEvm.Words
import Reasoning.EVMWord
import Reasoning.WordArithmetic

/-!
# Memory facts for solc's view-call pattern (solc 0.8.15)

For `x.f(args)` with one static return value solc emits, with `p = mload(0x40)`:
`mstore(p, selector)`, `mstore(p+4, arg)` (if any), `staticcall(gas, x, p, 4 + 32·#args, p, 32)`,
then on success `mstore(0x40, p + roundup32(returndatasize))` and decodes `mload(p)` after checking
`returndatasize ≥ 32`. These lemmas give the call input and the decoded word, for a symbolic
free memory pointer `p` (it depends on earlier calls' return data sizes).
-/

namespace L1cdmEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach L1cdmEvm.SymMem

theorem u_toNat_add (a b : UInt256) (h : a.toNat + b.toNat < 2 ^ 256) : (a + b).toNat = a.toNat + b.toNat := by
  rw [Words.toNat_add, Nat.mod_eq_of_lt h]

theorem u_toNat_ofNat {n : ℕ} (h : n < 2 ^ 256) : (UInt256.ofNat n).toNat = n :=
  Words.toNat_ofNat (by rw [Words.size_eq]; exact h)

theorem u_ext {a b : UInt256} (h : a.toNat = b.toNat) : a = b := Words.ext_iff.mpr h

theorem u_sub_add_self (a b : UInt256) : UInt256.sub (a + b) a = b := by
  cases a with
  | mk av =>
    cases b with
    | mk bv =>
      show UInt256.mk ((av + bv) - av) = UInt256.mk bv
      rw [add_sub_cancel_left]

theorem u_sub_self (a : UInt256) : UInt256.sub a a = ⟨0⟩ := by
  cases a with
  | mk v => show UInt256.mk (v - v) = UInt256.mk 0; rw [sub_self]

theorem u_add_comm (a b : UInt256) : a + b = b + a := by
  apply u_ext; rw [Words.toNat_add, Words.toNat_add, Nat.add_comm]

theorem u_sub_add_comm (a b : UInt256) : UInt256.sub (b + a) a = b := by
  rw [u_add_comm b a]; exact u_sub_add_self a b

/-- solc's `and(add(n, 31), not(31))` for `n < 2^32`. -/
theorem roundup_toNat (n : ℕ) (hn : n < 2 ^ 32) :
    (UInt256.land (UInt256.ofNat n + UInt256.ofNat 31) (UInt256.lnot (UInt256.ofNat 31))).toNat =
      32 * ((n + 31) / 32) := by
  rw [uland_toNat, show UInt256.lnot (UInt256.ofNat 31) = UInt256.lnot ⟨31⟩ from rfl, lnot31_toNat,
    u_toNat_add _ _ (by rw [u_toNat_ofNat (by omega), u_toNat_ofNat (by omega)]; omega),
    u_toNat_ofNat (by omega), u_toNat_ofNat (by omega), nat_land_mask _ (by omega)]

theorem sel4_eq_Mem (env : List UInt256) (s : ℕ) : Mem env (csel s) = sel4 s := rfl

/-- **Call input, selector only.** -/
theorem callmem_sel {m : ByteArray} {Bw : UInt256} (hF : Fmp m Bw) (hB : 96 ≤ Bw.toNat)
    (W : UInt256) (N s : ℕ) (hW : W = UInt256.ofNat N)
    (hk : sreadU (swriteU (csyms N) [] 0) 0 4 = csel s) (off : ℕ) (hoff : off = Bw.toNat) :
    Fmp ((UInt256.toByteArray W).write 0 m off 32) Bw ∧
      ((UInt256.toByteArray W).write 0 m off 32).readWithPadding Bw.toNat 4 = sel4 s := by
  subst hoff
  refine ⟨fmp_keep hF _ 0 _ 32 (by norm_num) (by rw [toByteArray_size]) (Or.inr hB), ?_⟩
  have hF1 := frame_mstore_const (frame_nil ([] : List UInt256) m Bw.toNat) N W hW 0
  rw [Nat.add_zero] at hF1
  have := frame_read hF1 0 4 (by norm_num) (by rw [hk]; rfl)
  simp only [Nat.add_zero] at this
  rw [this, hk]; rfl

/-- **Call input, selector and one word argument.** -/
theorem callmem_sel_arg {m : ByteArray} {Bw : UInt256} (hF : Fmp m Bw) (hB : 96 ≤ Bw.toNat)
    (hBb : Bw.toNat < 2 ^ 64)
    (W : UInt256) (N s : ℕ) (hW : W = UInt256.ofNat N) (X : UInt256)
    (hk : sreadU (swriteU (vsyms 0) (swriteU (csyms N) [] 0) 4) 0 36 = csel s ++ vsyms 0)
    (off : ℕ) (hoff : off = Bw.toNat) (off4 : ℕ) (hoff4 : off4 = Bw.toNat + 4) :
    Fmp ((UInt256.toByteArray X).write 0 ((UInt256.toByteArray W).write 0 m off 32) off4 32) Bw ∧
      ((UInt256.toByteArray X).write 0 ((UInt256.toByteArray W).write 0 m off 32) off4 32).readWithPadding
        Bw.toNat 36 = sel4 s ++ w32 X := by
  subst hoff hoff4
  refine ⟨fmp_keep (fmp_keep hF _ 0 _ 32 (by norm_num) (by rw [toByteArray_size]) (Or.inr hB))
    _ 0 _ 32 (by norm_num) (by rw [toByteArray_size]) (Or.inr (by omega)), ?_⟩
  have hF1 := frame_mstore_const (frame_nil ([X] : List UInt256) m Bw.toNat) N W hW 0
  rw [Nat.add_zero] at hF1
  have hF2 := frame_mstore_var hF1 0 X rfl 4
  have := frame_read hF2 0 36 (by norm_num) (by rw [hk]; rfl)
  simp only [Nat.add_zero] at this
  rw [this, hk, Mem_append, Mem_vsyms]; rfl

/-- The output copy of a call keeps the free memory pointer (whatever the output size). -/
theorem fmp_keep_out {m : ByteArray} {Bw : UInt256} (hF : Fmp m Bw) (hB : 96 ≤ Bw.toNat)
    (o : ByteArray) (ho : o.size < 2 ^ 256) :
    Fmp (o.write 0 m Bw.toNat (min (UInt256.ofNat 32) (UInt256.ofNat o.size)).toNat) Bw := by
  rw [show UInt256.ofNat 32 = (⟨32⟩ : UInt256) from rfl, callOutputLen32 (by rw [Words.size_eq]; exact ho)]
  by_cases h0 : min 32 o.size = 0
  · rw [h0, byteArray_write_len_zero]; exact hF
  · exact fmp_keep hF _ 0 _ _ (by omega) (by omega) (Or.inr hB)

/-- **After a successful view call** returning at least 32 bytes whose first word is `w`:
    the output is copied to `p = Bw`, the free memory pointer becomes `p + roundup(rds)`, and
    `mload(p)` is `w`. -/
theorem aftercall {m : ByteArray} {Bw : UInt256} (hF : Fmp m Bw) (hB : 96 ≤ Bw.toNat)
    (hBb : Bw.toNat < 2 ^ 64) (o : ByteArray) (w : UInt256) (h32 : 32 ≤ o.size)
    (hosz : o.size < 2 ^ 32) (hout : o.data.toList.take 32 = (w32 w).data.toList) :
    let m2 := o.write 0 m Bw.toNat (min (UInt256.ofNat 32) (UInt256.ofNat o.size)).toNat
    let nb := Bw + UInt256.land (UInt256.ofNat o.size + UInt256.ofNat 31) (UInt256.lnot (UInt256.ofNat 31))
    let m3 := (UInt256.toByteArray nb).write 0 m2 (UInt256.ofNat 64).toNat 32
    Fmp m2 Bw ∧ Fmp m3 nb ∧ memLoad Bw m3 = w ∧ nb.toNat = Bw.toNat + 32 * ((o.size + 31) / 32) := by
  intro m2 nb m3
  have hmin : (min (UInt256.ofNat 32) (UInt256.ofNat o.size)).toNat = 32 := by
    rw [show UInt256.ofNat 32 = (⟨32⟩ : UInt256) from rfl,
      callOutputLen32 (by rw [Words.size_eq]; omega)]
    omega
  have hm2 : m2 = o.write 0 m (Bw.toNat + 0) 32 := by simp only [m2, hmin, Nat.add_zero]
  have hF2 : Fmp m2 Bw := by
    rw [hm2]; exact fmp_keep hF _ 0 _ 32 (by norm_num) (by omega) (Or.inr (by omega))
  have hO2 := frame_output (frame_nil [w] m Bw.toNat) o 0 hout h32 0
  rw [← hm2, show swriteU (vsyms 0) [] 0 = vsyms 0 by decide +kernel] at hO2
  have hO3 : Frame [w] m3 Bw.toNat (vsyms 0) :=
    frame_write_disjoint _ 0 _ 32 hO2 (by norm_num) (by rw [toByteArray_size])
      (Or.inl (by rw [show (UInt256.ofNat 64).toNat = 64 from rfl]; omega))
  refine ⟨hF2, fmp_store _ _ _ rfl, ?_, ?_⟩
  · rw [frame_memLoad hO3 Bw 0 (by omega) (by decide +kernel),
      show sreadU (vsyms 0) 0 32 = vsyms 0 by decide +kernel, wordOf_vsyms]; rfl
  · have hr := roundup_toNat o.size hosz
    rw [u_toNat_add _ _ (by rw [hr]; omega), hr]

end L1cdmEvm

namespace L1cdmEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach L1cdmEvm.SymMem

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

theorem ofUInt256_land_mask (x : UInt256) :
    AccountAddress.ofUInt256 (UInt256.land x addrMask) = AccountAddress.ofUInt256 x := by
  unfold AccountAddress.ofUInt256
  congr 1
  show (UInt256.land x addrMask).toNat % _ = x.toNat % _
  rw [uland_toNat, show addrMask.toNat = 2 ^ 160 - 1 from rfl, Nat.and_two_pow_sub_one_eq_mod]
  simp [AccountAddress.size]

theorem ofUInt256_mask_land (x : UInt256) :
    AccountAddress.ofUInt256 (UInt256.land addrMask x) = AccountAddress.ofUInt256 x := by
  rw [show UInt256.land addrMask x = UInt256.land x addrMask by
    apply u_ext; rw [uland_toNat, uland_toNat, Nat.land_comm]]
  exact ofUInt256_land_mask x

theorem addr_source_word (I : ExecutionEnv) :
    AccountAddress.ofUInt256 (UInt256.land (UInt256.ofNat I.source.val) addrMask) = I.source := by
  rw [ofUInt256_land_mask, AccountAddress.ofUInt256_ofNat]

/-- `iszero(iszero(w))`. -/
theorem isZero_isZero (w : UInt256) :
    UInt256.isZero (UInt256.isZero w) = if w = ⟨0⟩ then ⟨0⟩ else UInt256.ofNat 1 := by
  by_cases h : w = ⟨0⟩
  · subst h; rfl
  · rw [if_neg h]
    have : UInt256.isZero w = UInt256.ofNat 0 := Words.isZero_eq0.mpr h
    rw [this]; rfl

/-- The bool decoder's check `w == iszero(iszero(w))` together with `w ≠ 0` gives `w = 1`. -/
theorem bool_true {w : UInt256} (hb : UInt256.eq w (UInt256.isZero (UInt256.isZero w)) ≠ UInt256.ofNat 0)
    (hne : w ≠ UInt256.ofNat 0) : w = UInt256.ofNat 1 := by
  have := Words.eq_ne0.mp hb
  rw [isZero_isZero, if_neg (show ¬ w = ⟨0⟩ from hne)] at this
  exact this

/-- `slt(n, 32)` for a small `n`: the return-data length check. -/
theorem slt_small (n : ℕ) (hn : n < 2 ^ 32) :
    UInt256.isZero (UInt256.slt (UInt256.ofNat n) (UInt256.ofNat 32)) ≠ UInt256.ofNat 0 ↔ 32 ≤ n := by
  rw [Words.isZero_ne0]
  unfold UInt256.slt UInt256.fromBool UInt256.sltBool
  rw [u_toNat_ofNat (by omega), show (UInt256.ofNat 32).toNat = 32 from rfl, if_neg (by omega),
    if_neg (by norm_num)]
  constructor
  · intro h
    by_contra hc
    have : (UInt256.ofNat n < UInt256.ofNat 32) := by
      show (UInt256.ofNat n).val < (UInt256.ofNat 32).val
      rw [Fin.lt_def]; show (UInt256.ofNat n).toNat < (UInt256.ofNat 32).toNat
      rw [u_toNat_ofNat (by omega)]; show n < 32; omega
    simp [this] at h
    exact absurd h (by decide)
  · intro h
    have : ¬ (UInt256.ofNat n < UInt256.ofNat 32) := by
      show ¬ (UInt256.ofNat n).val < (UInt256.ofNat 32).val
      rw [Fin.lt_def]; show ¬ (UInt256.ofNat n).toNat < (UInt256.ofNat 32).toNat
      rw [u_toNat_ofNat (by omega)]; show ¬ n < 32; omega
    simp [this]; rfl

end L1cdmEvm

namespace L1cdmEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach L1cdmEvm.SymMem

theorem selc_toNat (s : ℕ) (hs : s < 2 ^ 32) : (UInt256.ofNat (s * 2 ^ 224)).toNat = 2 ^ 224 * s := by
  rw [u_toNat_ofNat (by omega)]; ring

theorem mask224_toNat (w : UInt256) :
    (UInt256.land (UInt256.ofNat (2 ^ 224 - 1)) w).toNat = w.toNat % 2 ^ 224 := by
  rw [uland_toNat, u_toNat_ofNat (by norm_num), Nat.land_comm, Nat.and_two_pow_sub_one_eq_mod]

theorem lor_toNat (a b : UInt256) : (UInt256.lor a b).toNat = Nat.lor a.toNat b.toNat % UInt256.size :=
  rfl

theorem or_disjoint (s x : ℕ) (hs : s < 2 ^ 32) :
    (2 ^ 224 * s ||| x % 2 ^ 224) % UInt256.size = 2 ^ 224 * s + x % 2 ^ 224 := by
  rw [← Nat.two_pow_add_eq_or_of_lt (Nat.mod_lt _ (by norm_num)), Words.size_eq]
  apply Nat.mod_eq_of_lt
  have := Nat.mod_lt x (show 2 ^ 224 > 0 by norm_num)
  omega

theorem selmask_toNat (s : ℕ) (hs : s < 2 ^ 32) (w : UInt256) :
    (UInt256.lor (UInt256.ofNat (s * 2 ^ 224)) (UInt256.land (UInt256.ofNat (2 ^ 224 - 1)) w)).toNat =
      2 ^ 224 * s + w.toNat % 2 ^ 224 := by
  rw [lor_toNat, selc_toNat s hs, mask224_toNat]
  exact or_disjoint s _ hs

/-- Bytes of solc's `or(shl(224, selector), and(mload(p), 2^224-1))`. -/
theorem getB_selmask (s : ℕ) (hs : s < 2 ^ 32) (w : UInt256) (j : ℕ) (hj : j < 32) :
    getB (UInt256.toByteArray
      (UInt256.lor (UInt256.ofNat (s * 2 ^ 224)) (UInt256.land (UInt256.ofNat (2 ^ 224 - 1)) w))) j =
      if j < 4 then UInt8.ofNat (s / 2 ^ (24 - 8 * j) % 256) else byteBE w.toNat j := by
  rw [getB_toByteArray _ _ hj, selmask_toNat s hs]
  unfold byteBE
  generalize w.toNat = x
  by_cases h4 : j < 4
  · rw [if_pos h4]; congr 1; interval_cases j <;> omega
  · rw [if_neg h4]; congr 1; interval_cases j <;> omega

theorem csel_getD (s j : ℕ) (hs : s < 2 ^ 32) (hj : j < 4) :
    (csel s).getD j .u = .c (s / 2 ^ (24 - 8 * j) % 256) := by
  interval_cases j <;> simp [csel] <;> omega

/-- `MSTORE` of `or(shl(224, s), and(mload(a), 2^224-1))`: the selector over the loaded bytes. -/
theorem frame_mstore_selmask {env : List UInt256} {m : ByteArray} {B : ℕ} {D : List Sym}
    (hF : Frame env m B D) (s : ℕ) (hs : s < 2 ^ 32) (A : UInt256) (oa : ℕ) (ha : A.toNat = B + oa)
    (o : ℕ) :
    Frame env ((UInt256.toByteArray (UInt256.lor (UInt256.ofNat (s * 2 ^ 224))
        (UInt256.land (UInt256.ofNat (2 ^ 224 - 1)) (memLoad A m)))).write 0 m (B + o) 32) B
      (swriteU (csel s ++ sreadU D (oa + 4) 28) D o) := by
  apply frame_write _ _ 0 o 32 hF (by norm_num) (by rw [toByteArray_size]) (by simp [csel, sreadU])
  intro j hj
  have hj32 : j < 32 := by
    by_contra hc; apply hj
    rw [List.getD_eq_default _ _ (by simp [csel, sreadU]; omega)]
  rw [Nat.zero_add, getB_selmask s hs _ j hj32]
  by_cases h4 : j < 4
  · rw [if_pos h4, List.getD_append _ _ _ _ (by simp [csel]; omega), csel_getD s j hs h4]
    rfl
  · rw [if_neg h4]
    rw [List.getD_append_right _ _ _ _ (by simp [csel]; omega)] at hj ⊢
    simp only [show (csel s).length = 4 from rfl] at hj ⊢
    rw [sreadU_getD _ _ _ _ (by omega)] at hj ⊢
    rw [← getB_toByteArray _ _ hj32, getB_toByteArray_memLoad _ _ _ hj32, ha]
    have := hF _ hj
    rw [show B + oa + j = B + (oa + 4 + (j - 4)) by omega]
    exact this

end L1cdmEvm

namespace L1cdmEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach L1cdmEvm.SymMem

theorem ofNat_toNat_mod (n : ℕ) : (UInt256.ofNat n).toNat = n % 2 ^ 256 := by
  rw [show n % 2 ^ 256 = n % UInt256.size from rfl]
  unfold UInt256.ofNat UInt256.toNat
  simp [Id.run]

/-- Normalize a word equation between sums of a base word and literals (no wrap-around). -/
macro "wnorm" : tactic =>
  `(tactic| (apply u_ext; simp only [Words.toNat_add, ofNat_toNat_mod]; omega))

/-- Normalize a `toNat` of a sum of a base word and literals. -/
macro "tnorm" : tactic =>
  `(tactic| (simp only [Words.toNat_add, ofNat_toNat_mod]; omega))

theorem frame_memLoad_c {env : List UInt256} {m : ByteArray} {B : ℕ} {D : List Sym}
    (hF : Frame env m B D) (A : UInt256) (o : ℕ) (ha : A.toNat = B + o) (n : ℕ)
    (hk : sreadU D o 32 = csyms n) : memLoad A m = UInt256.ofNat n := by
  rw [frame_memLoad hF A o ha (by rw [hk]; simp [noU, csyms]), hk, wordOf_csyms]

end L1cdmEvm

namespace L1cdmEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach L1cdmEvm.SymMem

theorem frame_read_eq {env : List UInt256} {m : ByteArray} {B : ℕ} {D : List Sym}
    (hF : Frame env m B D) (o n : ℕ) (hn : n < 2 ^ 64) (S : List Sym) (hk : sreadU D o n = S)
    (hS : noU S = true) : m.readWithPadding (B + o) n = Mem env S := by
  rw [frame_read hF o n hn (by rw [hk]; exact hS), hk]

end L1cdmEvm

namespace L1cdmEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach L1cdmEvm.SymMem

/-- `CALLDATALOAD` from a calldata that is a `Mem`, evaluated to a constant. -/
theorem cdload_eq {env : List UInt256} {I : ExecutionEnv} {S : List Sym} (h : I.calldata = Mem env S)
    (o : ℕ) (ho : o < 2 ^ 64) (n : ℕ) (hk : sread S o 32 = csyms n) :
    uInt256OfByteArray (I.calldata.readBytes o 32) = UInt256.ofNat n := by
  rw [h, calldataload_Mem _ _ _ ho, hk, wordOf_csyms]

end L1cdmEvm

namespace L1cdmEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach L1cdmEvm.SymMem

/-! Variants with the offset as it appears in a summary (`off`, with `off = B + o`). -/

theorem fr_const {env : List UInt256} {m : ByteArray} {B : ℕ} {D : List Sym}
    (hF : Frame env m B D) (n : ℕ) (w : UInt256) (hw : w = UInt256.ofNat n) (off o : ℕ)
    (hoff : off = B + o) :
    Frame env ((UInt256.toByteArray w).write 0 m off 32) B (swriteU (csyms n) D o) := by
  subst hoff; exact frame_mstore_const hF n w hw o

theorem fr_var {env : List UInt256} {m : ByteArray} {B : ℕ} {D : List Sym}
    (hF : Frame env m B D) (k : ℕ) (w : UInt256) (hw : env.getD k ⟨0⟩ = w) (off o : ℕ)
    (hoff : off = B + o) :
    Frame env ((UInt256.toByteArray w).write 0 m off 32) B (swriteU (vsyms k) D o) := by
  subst hoff; exact frame_mstore_var hF k w hw o

theorem fr_load {env : List UInt256} {m : ByteArray} {B : ℕ} {D : List Sym}
    (hF : Frame env m B D) (a : UInt256) (oa : ℕ) (ha : a.toNat = B + oa) (off o : ℕ)
    (hoff : off = B + o) :
    Frame env ((UInt256.toByteArray (memLoad a m)).write 0 m off 32) B
      (swriteU (sreadU D oa 32) D o) := by
  subst hoff; exact frame_mstore_load hF a oa ha o

theorem fr_copy {env : List UInt256} {m : ByteArray} {B : ℕ} {D : List Sym}
    (hF : Frame env m B D) (CD : List Sym) (sa len : ℕ) (hlen : 0 < len)
    (hle : sa + len ≤ CD.length) (off o : ℕ) (hoff : off = B + o) :
    Frame env ((Mem env CD).write sa m off len) B (swriteU ((CD.drop sa).take len) D o) := by
  subst hoff; exact frame_copy_Mem hF CD sa o len hlen hle

theorem fr_selmask {env : List UInt256} {m : ByteArray} {B : ℕ} {D : List Sym}
    (hF : Frame env m B D) (s : ℕ) (hs : s < 2 ^ 32) (A : UInt256) (oa : ℕ) (ha : A.toNat = B + oa)
    (off o : ℕ) (hoff : off = B + o) :
    Frame env ((UInt256.toByteArray (UInt256.lor (UInt256.ofNat (s * 2 ^ 224))
        (UInt256.land (UInt256.ofNat (2 ^ 224 - 1)) (memLoad A m)))).write 0 m off 32) B
      (swriteU (csel s ++ sreadU D (oa + 4) 28) D o) := by
  subst hoff; exact frame_mstore_selmask hF s hs A oa ha o

end L1cdmEvm

namespace L1cdmEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach

/-- Replace the top stack word by an equal one (used to evaluate closed arithmetic). -/
theorem RD_head_eq {code : ByteArray} {ee : ExecutionEnv} {g : Sat256} {s0 : State} {pc : UInt256}
    {a b : UInt256} {R : List UInt256} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray}
    {σ : AccountMap} {k C : ℕ} (hab : a = b) (h : RD code ee g s0 pc (a :: R) mem aw rdata σ k C) :
    RD code ee g s0 pc (b :: R) mem aw rdata σ k C := hab ▸ h

end L1cdmEvm


namespace L1cdmEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach L1cdmEvm.SymMem

/-- The word a decoder reads from the first 32 bytes of return data. -/
def firstWord (o : ByteArray) : UInt256 := wordOf ⟨(o.data.toList.take 32).toArray⟩

theorem firstWord_spec (o : ByteArray) :
    32 ≤ o.size → o.data.toList.take 32 = (w32 (firstWord o)).data.toList := by
  intro h
  unfold firstWord wordOf w32
  rw [toByteArray_ofNat_fromByteArrayBigEndian_of_size]
  show (List.take 32 o.data.toList).toArray.size = 32
  have h' : 32 ≤ o.data.size := h
  simp only [List.size_toArray, List.length_take, Array.length_toList]
  omega

theorem w32_inj {a b : UInt256} (h : (w32 a).data.toList = (w32 b).data.toList) : a = b := by
  have ha := fromByteArrayBigEndian_toByteArray a
  have hb := fromByteArrayBigEndian_toByteArray b
  have : w32 a = w32 b := ba_ext h
  apply u_ext
  rw [← ha, ← hb]; unfold w32 at this; rw [this]

theorem bound_of_returns {σ σ₀ : AccountMap} {I : ExecutionEnv} {T : AccountAddress} {cd : ByteArray}
    {w : UInt256} (h : ReturnsWord σ σ₀ I T cd w) : CallBound σ σ₀ I T cd :=
  fun σc σ' o hs hc hcall => (h σc σ' true o hs hc hcall rfl).1

theorem returned_eq {σ σ₀ : AccountMap} {I : ExecutionEnv} {T : AccountAddress} {cd : ByteArray}
    {w w' : UInt256} (hr : CallReturned σ σ₀ I T cd w') (h : ReturnsWord σ σ₀ I T cd w) : w' = w := by
  obtain ⟨σc, σ', o, hs, hc, hcall, h32, hfirst⟩ := hr
  have := (h σc σ' true o hs hc hcall rfl).2 h32
  rw [hfirst] at this
  exact w32_inj this

end L1cdmEvm
