import Reasoning.Reach
import Reasoning.Solc
import ExporterEvm.Words

/-! # Memory as a list of bytes

Copied from `../evm-lean-bridge/BridgeEvm/Mem.lean` (namespace renamed, otherwise unchanged); the
layout examples in the comments are the bridge's. `refundETH` builds its message hash over a dynamic `bytes` value whose words sit at offsets
`≡ 4 (mod 32)`, so the word-aligned view of `../evm-lean/ExpiryEvm/Mem.lean` does not apply. Here
memory is kept as `ofL L` for a list `L : List UInt8` made of 32-byte word encodings `wb w`,
zero runs `List.replicate n 0`, and slices of those. Every `MSTORE`, `MLOAD`, `KECCAK256` input and
call input in the trace reduces to `List.take`/`List.drop` on such lists (`write_wb`,
`memLoad_ofL`, `keccakWord_ofL`, `read_ofL`), which `simp` normalises when offsets are numerals.
All lemmas here are proved from EVMLean's definitions (`ByteArray.write`, `readWithPadding`) via
EquiVM's `Reasoning.Theory` memory lemmas; nothing is assumed. -/

namespace ExporterEvm.Mem

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach

/-- A byte array given by its list of bytes. -/
def ofL (L : List UInt8) : ByteArray := ⟨L.toArray⟩

/-- The 32-byte big-endian encoding of a word, as a list (`UInt256.toByteArray`). -/
def wb (w : UInt256) : List UInt8 := (UInt256.toByteArray w).data.toList

@[simp] theorem ofL_size (L : List UInt8) : (ofL L).size = L.length := by
  simp [ofL, ByteArray.size]

@[simp] theorem ofL_data (L : List UInt8) : (ofL L).data = L.toArray := rfl

theorem eq_ofL (b : ByteArray) : b = ofL b.data.toList := by
  cases b; simp [ofL]

theorem ofL_inj {A B : List UInt8} : ofL A = ofL B ↔ A = B := by
  constructor
  · intro h
    have := congrArg (fun b : ByteArray => b.data.toList) h
    simpa [ofL] using this
  · intro h; rw [h]

@[simp] theorem length_wb (w : UInt256) : (wb w).length = 32 := by
  unfold wb
  rw [Array.length_toList]
  exact toByteArray_size w

theorem toByteArray_eq_ofL (w : UInt256) : UInt256.toByteArray w = ofL (wb w) := eq_ofL _

theorem fromBE_wb (w : UInt256) : fromBytesBigEndian (wb w) = w.toNat := by
  have h := fromBytesBE_word w
  rw [word_toBytesBE_eq_toByteArray_toList, byteArray_toList_eq] at h
  exact h

theorem ofNat_toNat (a : UInt256) : UInt256.ofNat a.toNat = a := by
  cases a with
  | mk v =>
    unfold UInt256.ofNat UInt256.toNat
    simp [Id.run]

@[simp] theorem ofNat_fromBE_wb (w : UInt256) : UInt256.ofNat (fromBytesBigEndian (wb w)) = w := by
  rw [fromBE_wb, ofNat_toNat]

@[simp] theorem fromByteArrayBigEndian_ofL (L : List UInt8) :
    fromByteArrayBigEndian (ofL L) = fromBytesBigEndian L := by
  unfold fromByteArrayBigEndian
  rw [byteArray_toList_eq]; rfl

theorem wb_ofNat_fromBE (l : List UInt8) (h : l.length = 32) :
    wb (UInt256.ofNat (fromBytesBigEndian l)) = l := by
  have hs : (ofL l).size = 32 := by simpa using h
  have := toByteArray_ofNat_fromByteArrayBigEndian_of_size hs
  rw [fromByteArrayBigEndian_ofL] at this
  unfold wb; rw [this]; simp [ofL]

theorem extract_ofL (L : List UInt8) (a b : ℕ) :
    (ofL L).extract a b = ofL ((L.take b).drop a) := by
  apply ByteArray.ext
  simp only [ByteArray.data_extract, ofL_data]
  apply Array.toList_inj.mp
  simp only [Array.toList_extract, List.extract, ofL, List.toList_toArray]
  rw [List.drop_take]

theorem append_ofL (A B : List UInt8) : ofL A ++ ofL B = ofL (A ++ B) := by
  apply ByteArray.ext
  simp [ofL, ByteArray.data_append]

theorem zeroes_ofL (n : ℕ) : ByteArray.zeroes n = ofL (List.replicate n 0) := by
  apply ByteArray.ext
  simp [ofL, ByteArray.zeroes]

theorem empty_ofL : ByteArray.empty = ofL [] := rfl

/-- EquiVM's `write_from_gap_eq` without its (unused) `USize` bound on the gap. -/
theorem write_from_gap_eq_nb (src base : ByteArray) (srcAddr destAddr len : Nat)
    (hlen : len ≠ 0) (hsrc : srcAddr + len ≤ src.size) (hge : base.size ≤ destAddr) :
    src.write srcAddr base destAddr len =
      base ++ ByteArray.zeroes (destAddr - base.size) ++
        src.extract srcAddr (srcAddr + len) := by
  apply ByteArray.ext
  unfold ByteArray.write
  rw [if_neg hlen, if_neg (show ¬ srcAddr ≥ src.size from by omega)]
  have hcopy : min len (src.size - srcAddr) = len := by omega
  have htail : min base.size (destAddr + len) - (destAddr + len) = 0 := by omega
  simp only [hcopy, htail, ByteArray.data_copySlice, ByteArray.data_append,
    ByteArray.data_extract]
  have hpz : (ByteArray.zeroes (destAddr - base.size)).data.size =
      destAddr - base.size := by
    rw [show (ByteArray.zeroes (destAddr - base.size)).data.size =
          (ByteArray.zeroes (destAddr - base.size)).size from rfl,
      ByteArray_zeroes_size]
  have hDsz :
      (base.data ++
        (ByteArray.zeroes (destAddr - base.size)).data).size =
        destAddr := by
    rw [Array.size_append, hpz, show base.data.size = base.size from rfl]
    omega
  rw [show (ByteArray.zeroes 0).data = (#[] : Array UInt8) from by
    rw [zeroes_zero (n := 0) (by rfl)]
    rfl]
  simp only [Array.append_empty, Nat.add_zero]
  rw [show min len (src.data.size - srcAddr) = len by
    have : src.data.size = src.size := rfl
    omega]
  rw [Array.extract_eq_self_of_le (by rw [hDsz])]
  rw [show (base.data ++
        (ByteArray.zeroes (destAddr - base.size)).data).extract
          (destAddr + len) = (#[] : Array UInt8) from by
    apply Array.extract_eq_empty_of_le
    rw [hDsz]
    omega]
  simp [Array.append_assoc]

/-- **`write` of `n` bytes of a list at `off`**, in all three cases (inside, overlapping the end,
    past the end with a zero gap). -/
theorem write_ofL (src L : List UInt8) (off n : ℕ) (hn0 : n ≠ 0) (hn : n ≤ src.length) :
    (ofL src).write 0 (ofL L) off n =
      ofL (L.take off ++ List.replicate (off - L.length) 0 ++ src.take n ++ L.drop (off + n)) := by
  have hsrc : n ≤ (ofL src).size := by simpa using hn
  by_cases hin : off + n ≤ L.length
  · rw [write_eq_gen _ _ _ _ hn0 hsrc (by simpa using hin)]
    simp only [extract_ofL, ofL_size, append_ofL]
    have h1 : off - L.length = 0 := by omega
    simp only [h1, List.replicate_zero, List.append_nil, List.take_length, List.drop_zero]
  · by_cases hle : off ≤ L.length
    · rw [write_eq_gen_extend _ _ _ _ hn0 hsrc (by simpa using hle)
        (by rw [ofL_size]; omega)]
      simp only [extract_ofL, append_ofL]
      have h1 : off - L.length = 0 := by omega
      have h2 : L.drop (off + n) = [] := List.drop_eq_nil_of_le (by omega)
      simp only [h1, h2, List.replicate_zero, List.append_nil, List.drop_zero]
    · rw [write_from_gap_eq_nb _ _ _ _ _ hn0 (by simpa using hn) (by rw [ofL_size]; omega)]
      simp only [extract_ofL, append_ofL, zeroes_ofL, ofL_size, Nat.zero_add, List.drop_zero]
      have h2 : L.drop (off + n) = [] := List.drop_eq_nil_of_le (by omega)
      have h3 : L.take off = L := List.take_of_length_le (by omega)
      simp only [h2, h3, List.append_nil]

/-- Copying the first 32 bytes of a call's return data (`o.size ≥ 32`). -/
theorem write_bytes32 (o : ByteArray) (L : List UInt8) (off : ℕ) (h : 32 ≤ o.size) :
    o.write 0 (ofL L) off 32 =
      ofL (L.take off ++ List.replicate (off - L.length) 0 ++ o.data.toList.take 32 ++
        L.drop (off + 32)) := by
  conv_lhs => rw [eq_ofL o]
  exact write_ofL _ _ _ _ (by decide) (by simpa using h)

theorem length_take32 (o : ByteArray) (h : 32 ≤ o.size) : (o.data.toList.take 32).length = 32 := by
  simp; exact h

/-- `MSTORE` of a word. -/
theorem write_wb (w : UInt256) (L : List UInt8) (off : ℕ) :
    (UInt256.toByteArray w).write 0 (ofL L) off 32 =
      ofL (L.take off ++ List.replicate (off - L.length) 0 ++ wb w ++ L.drop (off + 32)) := by
  rw [toByteArray_eq_ofL, write_ofL _ _ _ _ (by decide) (by simp),
    List.take_of_length_le (l := wb w) (by simp)]

/-- `write_wb` restricted to small (numeral) offsets; this is the version `msimp` uses, so writes
    at symbolic offsets above the free-memory pointer are left for `write_first`/`write_rel`. -/
theorem write_wb_small (w : UInt256) (L : List UInt8) (off : ℕ) (_h : off < 100000) :
    (UInt256.toByteArray w).write 0 (ofL L) off 32 =
      ofL (L.take off ++ List.replicate (off - L.length) 0 ++ wb w ++ L.drop (off + 32)) :=
  write_wb w L off

/-- An in-bounds read. -/
theorem read_ofL (L : List UInt8) (a n : ℕ) (hn : 0 < n) (hn64 : n < 2 ^ 64)
    (h : a + n ≤ L.length) :
    (ofL L).readWithPadding a n = ofL ((L.drop a).take n) := by
  rw [readWithPadding_eq_extract' _ _ _ hn hn64 (by simpa using h), extract_ofL]
  congr 1
  rw [List.drop_take]
  congr 1; omega

/-- `MLOAD` in bounds. -/
theorem memLoad_ofL (a : UInt256) (L : List UInt8) (h : a.toNat + 32 ≤ L.length) :
    memLoad a (ofL L) = UInt256.ofNat (fromBytesBigEndian ((L.drop a.toNat).take 32)) := by
  unfold memLoad
  rw [if_neg (by simp; omega), read_ofL _ _ _ (by decide) (by decide) h, fromByteArrayBigEndian_ofL]

/-- `KECCAK256` over an in-bounds window. -/
theorem keccakWord_ofL (a b : UInt256) (L : List UInt8) (hb : 0 < b.toNat)
    (hb64 : b.toNat < 2 ^ 64) (h : a.toNat + b.toNat ≤ L.length) :
    keccakWord a b (ofL L) =
      UInt256.ofNat (fromByteArrayBigEndian (KEC (ofL ((L.drop a.toNat).take b.toNat)))) := by
  unfold keccakWord
  rw [read_ofL _ _ _ hb hb64 h]

theorem take_drop_cons (n : ℕ) (l r : List UInt8) : l.take n ++ (l.drop n ++ r) = l ++ r := by
  rw [← List.append_assoc, List.take_append_drop]

theorem take_wb_of_le (w : UInt256) (n : ℕ) (h : 32 ≤ n) : (wb w).take n = wb w :=
  List.take_of_length_le (by simp; exact h)

theorem drop_wb_of_le (w : UInt256) (n : ℕ) (h : 32 ≤ n) : (wb w).drop n = [] :=
  List.drop_eq_nil_of_le (by simp; exact h)

theorem fromBE_append (a b : List UInt8) :
    fromBytesBigEndian (a ++ b) = fromBytesBigEndian a * 2 ^ (8 * b.length) + fromBytesBigEndian b := by
  unfold fromBytesBigEndian Function.comp
  rw [List.reverse_append, fromBytes'_append, List.length_reverse]; ring

theorem fromBE_lt (l : List UInt8) : fromBytesBigEndian l < 2 ^ (8 * l.length) := by
  unfold fromBytesBigEndian Function.comp
  have := fromBytes'_le (bs := l.reverse); rwa [List.length_reverse] at this

/-- solc's selector merge `or(and(mload(p), 2^224-1), sel << 224)` writes the selector bytes over
    the first 4 bytes of a word and keeps the other 28. -/
theorem wb_merge (sel l : List UInt8) (hsel : sel.length = 4) (hl : l.length = 32) :
    wb (UInt256.lor (UInt256.ofNat (fromBytesBigEndian sel * 2 ^ 224))
        (UInt256.land (UInt256.ofNat (2 ^ 224 - 1)) (UInt256.ofNat (fromBytesBigEndian l)))) =
      sel ++ l.drop 4 := by
  have hd : (l.drop 4).length = 28 := by simp [hl]
  have hs := fromBE_lt sel
  have hv := fromBE_lt l
  have hdl := fromBE_lt (l.drop 4)
  rw [hsel] at hs; rw [hl] at hv; rw [hd] at hdl
  have hsplit : fromBytesBigEndian l =
      fromBytesBigEndian (l.take 4) * 2 ^ 224 + fromBytesBigEndian (l.drop 4) := by
    conv_lhs => rw [← List.take_append_drop 4 l]
    rw [fromBE_append, hd]
  have hmod : fromBytesBigEndian l % 2 ^ 224 = fromBytesBigEndian (l.drop 4) := by
    rw [hsplit, Nat.mul_comm, Nat.mul_add_mod, Nat.mod_eq_of_lt (by norm_num at hdl ⊢; omega)]
  have hS : fromBytesBigEndian sel * 2 ^ 224 < 2 ^ 256 := by
    have : (2:ℕ) ^ 256 = 2 ^ 32 * 2 ^ 224 := by norm_num
    rw [this]; exact Nat.mul_lt_mul_of_pos_right (by norm_num at hs ⊢; omega) (by positivity)
  have hX : UInt256.lor (UInt256.ofNat (fromBytesBigEndian sel * 2 ^ 224))
        (UInt256.land (UInt256.ofNat (2 ^ 224 - 1)) (UInt256.ofNat (fromBytesBigEndian l))) =
      UInt256.ofNat (fromBytesBigEndian (sel ++ l.drop 4)) := by
    have hsz : UInt256.size = 2 ^ 256 := Words.size_eq
    apply (Words.ext_iff).mpr
    rw [Words.lor_toNat, Words.land_toNat, fromBE_append, hd,
      Words.toNat_ofNat (by rw [hsz]; exact hS),
      Words.toNat_ofNat (by rw [hsz]; exact lt_of_le_of_lt (Nat.sub_le _ _) (by norm_num)),
      Words.toNat_ofNat (by rw [hsz]; exact hv),
      Nat.land_comm, Nat.and_two_pow_sub_one_eq_mod, hmod, Nat.mul_comm,
      ← Nat.two_pow_add_eq_or_of_lt (by norm_num at hdl ⊢; omega)]
    rw [Words.toNat_ofNat (by rw [hsz]; norm_num at hdl hS ⊢; omega)]
  rw [hX, wb_ofNat_fromBE _ (by simp [hsel, hd])]

/-- The selector merge with the selector word `S` (low 224 bits zero) and solc's literal mask
    `2^224 - 1`: the merged word's bytes are `S`'s first 4 bytes followed by bytes 4..32 of the
    loaded word. -/
theorem wb_merge' (S : UInt256) (l : List UInt8) (hS : S.toNat % 2 ^ 224 = 0)
    (hl : l.length = 32) :
    wb (UInt256.lor S
        (UInt256.land (UInt256.ofNat 26959946667150639794667015087019630673637144422540572481103610249215)
          (UInt256.ofNat (fromBytesBigEndian l)))) = (wb S).take 4 ++ l.drop 4 := by
  have hsel : ((wb S).take 4).length = 4 := by simp
  have hd : ((wb S).drop 4).length = 28 := by simp
  have hsplit : S.toNat = fromBytesBigEndian ((wb S).take 4) * 2 ^ 224 :=  by
    have h1 : fromBytesBigEndian (wb S) =
        fromBytesBigEndian ((wb S).take 4) * 2 ^ 224 + fromBytesBigEndian ((wb S).drop 4) := by
      conv_lhs => rw [← List.take_append_drop 4 (wb S)]
      rw [fromBE_append, hd]
    have hlt := fromBE_lt ((wb S).drop 4)
    rw [hd] at hlt
    rw [fromBE_wb] at h1
    have h2 : fromBytesBigEndian ((wb S).drop 4) = 0 := by
      have := congrArg (· % 2 ^ 224) h1
      simp only at this
      rw [hS, Nat.mul_comm, Nat.mul_add_mod, Nat.mod_eq_of_lt (by norm_num at hlt ⊢; omega)] at this
      exact this.symm
    rw [h1, h2, Nat.add_zero]
  have hS' : S = UInt256.ofNat (fromBytesBigEndian ((wb S).take 4) * 2 ^ 224) := by
    rw [← hsplit, ofNat_toNat]
  have hM : (26959946667150639794667015087019630673637144422540572481103610249215 : ℕ) =
      2 ^ 224 - 1 := by norm_num
  rw [hM]
  conv_lhs => rw [hS']
  exact wb_merge _ _ hsel hl

/-! ## Memory above a symbolic free-memory pointer

After the `expiredMessages` call the free-memory pointer is `fp = 0x284 + roundUp32(returndatasize)`,
symbolic. Memory is then kept as `ofL (pre L fp ++ Q)`: the old contents `L` cut or zero-padded to
exactly `fp` bytes, followed by a list `Q` written relative to `fp`. -/

/-- `L` cut or zero-padded to length `fp`. -/
def pre (L : List UInt8) (fp : ℕ) : List UInt8 := L.take fp ++ List.replicate (fp - L.length) 0

@[simp] theorem length_pre (L : List UInt8) (fp : ℕ) : (pre L fp).length = fp := by
  unfold pre; simp; omega

/-- The first write at `fp` (where `fp + 32 ≥ L.length`). -/
theorem write_first (w : UInt256) (L : List UInt8) (fp : ℕ) (h : L.length ≤ fp + 32) :
    (UInt256.toByteArray w).write 0 (ofL L) fp 32 = ofL (pre L fp ++ wb w) := by
  rw [write_wb, List.drop_eq_nil_of_le (by omega), List.append_nil]; rfl

/-- A word write at `fp + c`. -/
theorem write_rel (w : UInt256) (P Q : List UInt8) (off c : ℕ) (hoff : off = P.length + c) :
    (UInt256.toByteArray w).write 0 (ofL (P ++ Q)) off 32 =
      ofL (P ++ (Q.take c ++ List.replicate (c - Q.length) 0 ++ wb w ++ Q.drop (c + 32))) := by
  subst hoff
  rw [write_wb]
  congr 1
  have h1 : (P ++ Q).take (P.length + c) = P ++ Q.take c := by
    rw [List.take_append, List.take_of_length_le (by omega), Nat.add_sub_cancel_left]
  have h2 : P.length + c - (P ++ Q).length = c - Q.length := by simp; omega
  have h3 : (P ++ Q).drop (P.length + c + 32) = Q.drop (c + 32) := by
    rw [List.drop_append, List.drop_eq_nil_of_le (by omega), List.nil_append]; congr 1; omega
  rw [h1, h2, h3]; simp only [List.append_assoc]

/-- An in-bounds read at `fp + c`. -/
theorem read_rel (P Q : List UInt8) (off c n : ℕ) (hoff : off = P.length + c) (hn : 0 < n)
    (hn64 : n < 2 ^ 64) (h : c + n ≤ Q.length) :
    (ofL (P ++ Q)).readWithPadding off n = ofL ((Q.drop c).take n) := by
  subst hoff
  rw [read_ofL _ _ _ hn hn64 (by simp; omega)]
  congr 1
  rw [List.drop_append, List.drop_eq_nil_of_le (by omega), List.nil_append]
  congr 2; omega

/-- `MLOAD` below `fp` reads the old contents. -/
theorem memLoad_pre (a : UInt256) (L Q : List UInt8) (fp : ℕ) (h1 : a.toNat + 32 ≤ fp)
    (h2 : a.toNat + 32 ≤ L.length) :
    memLoad a (ofL (pre L fp ++ Q)) = UInt256.ofNat (fromBytesBigEndian ((L.drop a.toNat).take 32)) := by
  rw [memLoad_ofL _ _ (by simp; omega)]
  congr 2
  have hp : (pre L fp).length = fp := length_pre L fp
  rw [List.drop_append_of_le_length (by omega), List.take_append_of_le_length (by simp; omega)]
  unfold pre
  rw [List.drop_append_of_le_length (by simp; omega), List.take_append_of_le_length (by simp; omega),
    List.drop_take, List.take_take]
  congr 1; omega

/-- A write from source offset `sa` is a write of the extracted window from offset 0. -/
theorem write_from_extract (src m : ByteArray) (sa d n : ℕ) (hn : n ≠ 0) (h : sa + n ≤ src.size) :
    src.write sa m d n = (src.extract sa (sa + n)).write 0 m d n := by
  have hes : (src.extract sa (sa + n)).size = n := by simp; omega
  apply ByteArray.ext
  unfold ByteArray.write
  rw [if_neg hn, if_neg hn, if_neg (by omega), if_neg (by omega)]
  simp only [ByteArray.data_copySlice, ByteArray.data_append, ByteArray.data_extract, hes]
  have hsz : src.data.size = src.size := rfl
  have e1 : min n (src.size - sa) = n := by omega
  have e2 : min n (n - 0) = n := by omega
  rw [e1, e2]
  congr 1
  · congr 1
    rw [Array.extract_append, Array.extract_append]
    simp only [Array.size_extract, hsz]
    rw [Array.extract_extract]
    congr 1
    · congr 1; omega
    · congr 1 <;> simp <;> omega
  · simp only [Array.size_append, Array.size_extract, hsz]
    congr 1; omega

/-- A write from a byte array at source offset `sa` at `fp + c`. -/
theorem write_rel_src (src : ByteArray) (P Q : List UInt8) (sa off c n : ℕ) (hn : n ≠ 0)
    (h : sa + n ≤ src.size) (hoff : off = P.length + c) :
    src.write sa (ofL (P ++ Q)) off n =
      ofL (P ++ (Q.take c ++ List.replicate (c - Q.length) 0 ++ (src.data.toList.drop sa).take n ++
        Q.drop (c + n))) := by
  subst hoff
  rw [write_from_extract _ _ _ _ _ hn h, eq_ofL (src.extract sa (sa + n)),
    write_ofL _ _ _ _ hn (by simp; omega)]
  congr 1
  have h1 : (P ++ Q).take (P.length + c) = P ++ Q.take c := by
    rw [List.take_append, List.take_of_length_le (by omega), Nat.add_sub_cancel_left]
  have h2 : P.length + c - (P ++ Q).length = c - Q.length := by simp; omega
  have h3 : (P ++ Q).drop (P.length + c + n) = Q.drop (c + n) := by
    rw [List.drop_append, List.drop_eq_nil_of_le (by omega), List.nil_append]; congr 1; omega
  have h4 : (src.extract sa (sa + n)).data.toList.take n = (src.data.toList.drop sa).take n := by
    simp only [ByteArray.data_extract, Array.toList_extract, List.extract, List.drop_take,
      List.take_take]
    congr 1; omega
  rw [h1, h2, h3, h4]; simp only [List.append_assoc]

/-- A zero-length write is the identity. -/
theorem write_len0 (src m : ByteArray) (sa da : ℕ) : src.write sa m da 0 = m := by
  unfold ByteArray.write; simp

/-! ## Literal word arithmetic -/

theorem ofNat_add_ofNat (a b : ℕ) : UInt256.ofNat a + UInt256.ofNat b = UInt256.ofNat (a + b) := by
  show UInt256.add _ _ = _
  unfold UInt256.add UInt256.ofNat
  simp only [Id.run]
  congr 1
  apply Fin.ext
  simp [Fin.val_add, Fin.val_ofNat, Nat.add_mod]

theorem toNat_ofNat_mod (n : ℕ) :
    (UInt256.ofNat n).toNat =
      n % 115792089237316195423570985008687907853269984665640564039457584007913129639936 := by
  unfold UInt256.ofNat UInt256.toNat
  simp [Id.run, Fin.val_ofNat, UInt256.size]

theorem ofNat_sub_ofNat (a b : ℕ) (h : b ≤ a)
    (ha : a < 115792089237316195423570985008687907853269984665640564039457584007913129639936) :
    UInt256.sub (UInt256.ofNat a) (UInt256.ofNat b) = UInt256.ofNat (a - b) := by
  unfold UInt256.sub UInt256.ofNat
  simp only [Id.run]
  congr 1
  apply Fin.ext
  have hs : UInt256.size = 115792089237316195423570985008687907853269984665640564039457584007913129639936 := rfl
  simp only [Fin.sub_def, Fin.val_ofNat, hs]
  rw [Nat.mod_eq_of_lt ha, Nat.mod_eq_of_lt (show b < _ by omega)]
  rw [Nat.mod_eq_of_lt (show a - b < _ by omega)]
  omega

/-- solc's rounding mask `not(31)` applied to a literal-sized value (`n < 2^256`). -/
theorem land_not31_ofNat (n : ℕ)
    (hn : n < 115792089237316195423570985008687907853269984665640564039457584007913129639936) :
    UInt256.land (UInt256.ofNat 115792089237316195423570985008687907853269984665640564039457584007913129639904)
      (UInt256.ofNat n) = UInt256.ofNat (n / 32 * 32) := by
  apply Words.ext_iff.mpr
  rw [show (115792089237316195423570985008687907853269984665640564039457584007913129639904 : ℕ) =
      2 ^ 256 - 2 ^ 5 by norm_num, u256_land_high_mask_toNat _ 5 (by norm_num),
    toNat_ofNat_mod, toNat_ofNat_mod, Nat.mod_eq_of_lt hn,
    Nat.mod_eq_of_lt (lt_of_le_of_lt (Nat.div_mul_le_self n 32) hn)]
  norm_num

theorem fin_lit (n : ℕ) : (⟨(OfNat.ofNat n : Fin UInt256.size)⟩ : UInt256) = UInt256.ofNat n := rfl

/-- Side-condition discharger for the memory rewrites: compute list lengths and literal
    `toNat`s, then decide the remaining numeral comparison. -/
macro "mdisch" : tactic => `(tactic| (first
  | (simp only [List.length_append, List.length_replicate,
      length_wb, List.length_take, List.length_drop, List.length_cons, List.length_nil,
      toNat_ofNat_mod, ofNat_add_ofNat, Nat.reduceMod, Nat.reduceAdd, Nat.reduceSub, Nat.reduceMul,
      Nat.reducePow, Nat.min_def, Nat.reduceLeDiff, Nat.reduceLTLE, ↓reduceIte]; done)
  | (simp only [List.length_append, List.length_replicate,
      length_wb, List.length_take, List.length_drop, List.length_cons, List.length_nil,
      toNat_ofNat_mod, ofNat_add_ofNat, Nat.reduceMod, Nat.reduceAdd, Nat.reduceSub, Nat.reduceMul,
      Nat.reducePow, Nat.min_def, Nat.reduceLeDiff, Nat.reduceLTLE, ↓reduceIte]; omega)
  | omega))

/-- Normalise list-memory terms with numeral offsets: split `take`/`drop` over `++`, compute
    lengths, reduce literal word arithmetic and `toNat`. Never unfolds `List.replicate`. -/
macro "mnorm" : tactic => `(tactic| simp (disch := mdisch) only [List.take_append,
  List.drop_append, List.length_append, List.length_replicate, length_wb,
  List.take_replicate, List.drop_replicate, List.take_nil, List.drop_nil, List.take_zero,
  List.drop_zero, List.append_nil, List.nil_append, List.append_assoc, List.replicate_zero, Nat.zero_sub,
  toNat_ofNat_mod, ofNat_add_ofNat, Nat.reduceMod, Nat.reduceAdd, Nat.reduceSub, Nat.reduceMul,
  Nat.reducePow, Nat.reduceLTLE, Nat.min_def, Nat.reduceLeDiff, ↓reduceIte, List.length_take,
  List.length_drop, take_drop_cons, List.take_of_length_le, List.drop_eq_nil_of_le, List.length_cons,
  List.length_nil, ofL_inj])

/-- `mnorm` plus the memory-operation rewrites (`MSTORE`, `MLOAD`, `KECCAK256` on list memory).
    `msimp [h₁, …] at h` adds extra rewrite facts (e.g. the length of an abstracted list). -/
syntax "msimp" (" [" Lean.Parser.Tactic.simpLemma,* "]")? " at " ident : tactic
macro_rules
  | `(tactic| msimp at $h:ident) => `(tactic| simp (disch := mdisch) only [List.take_append,
  List.drop_append, List.length_append, List.length_replicate, length_wb,
  List.take_replicate, List.drop_replicate, List.take_nil, List.drop_nil, List.take_zero,
  List.drop_zero, List.append_nil, List.nil_append, List.append_assoc, List.replicate_zero, Nat.zero_sub,
  toNat_ofNat_mod, ofNat_add_ofNat, Nat.reduceMod, Nat.reduceAdd, Nat.reduceSub, Nat.reduceMul,
  Nat.reducePow, Nat.reduceLTLE, Nat.min_def, Nat.reduceLeDiff, ↓reduceIte, List.length_take,
  List.length_drop, take_drop_cons, List.take_of_length_le, List.drop_eq_nil_of_le, List.length_cons,
  List.length_nil, write_wb_small, memLoad_ofL, keccakWord_ofL, ofNat_fromBE_wb, ofNat_sub_ofNat,
  wb_ofNat_fromBE, land_not31_ofNat, Nat.reduceDiv, read_ofL] at $h:ident)
  | `(tactic| msimp [$xs,*] at $h:ident) =>
    `(tactic| simp (disch := mdisch) only [List.take_append,
  List.drop_append, List.length_append, List.length_replicate, length_wb,
  List.take_replicate, List.drop_replicate, List.take_nil, List.drop_nil, List.take_zero,
  List.drop_zero, List.append_nil, List.nil_append, List.append_assoc, List.replicate_zero, Nat.zero_sub,
  toNat_ofNat_mod, ofNat_add_ofNat, Nat.reduceMod, Nat.reduceAdd, Nat.reduceSub, Nat.reduceMul,
  Nat.reducePow, Nat.reduceLTLE, Nat.min_def, Nat.reduceLeDiff, ↓reduceIte, List.length_take,
  List.length_drop, take_drop_cons, List.take_of_length_le, List.drop_eq_nil_of_le, List.length_cons,
  List.length_nil, write_wb_small, memLoad_ofL, keccakWord_ofL, ofNat_fromBE_wb, ofNat_sub_ofNat,
  wb_ofNat_fromBE, land_not31_ofNat, Nat.reduceDiv, read_ofL, $xs,*] at $h:ident)

/-- `msimp` on the goal. -/
syntax "msimpg" (" [" Lean.Parser.Tactic.simpLemma,* "]")? : tactic
macro_rules
  | `(tactic| msimpg) => `(tactic| simp (disch := mdisch) only [List.take_append,
  List.drop_append, List.length_append, List.length_replicate, length_wb,
  List.take_replicate, List.drop_replicate, List.take_nil, List.drop_nil, List.take_zero,
  List.drop_zero, List.append_nil, List.nil_append, List.append_assoc, List.replicate_zero, Nat.zero_sub,
  toNat_ofNat_mod, ofNat_add_ofNat, Nat.reduceMod, Nat.reduceAdd, Nat.reduceSub, Nat.reduceMul,
  Nat.reducePow, Nat.reduceLTLE, Nat.min_def, Nat.reduceLeDiff, ↓reduceIte, List.length_take,
  List.length_drop, take_drop_cons, List.take_of_length_le, List.drop_eq_nil_of_le, List.length_cons,
  List.length_nil, write_wb_small, memLoad_ofL, keccakWord_ofL, ofNat_fromBE_wb, ofNat_sub_ofNat,
  wb_ofNat_fromBE, land_not31_ofNat, Nat.reduceDiv, read_ofL])
  | `(tactic| msimpg [$xs,*]) => `(tactic| simp (disch := mdisch) only [List.take_append,
  List.drop_append, List.length_append, List.length_replicate, length_wb,
  List.take_replicate, List.drop_replicate, List.take_nil, List.drop_nil, List.take_zero,
  List.drop_zero, List.append_nil, List.nil_append, List.append_assoc, List.replicate_zero, Nat.zero_sub,
  toNat_ofNat_mod, ofNat_add_ofNat, Nat.reduceMod, Nat.reduceAdd, Nat.reduceSub, Nat.reduceMul,
  Nat.reducePow, Nat.reduceLTLE, Nat.min_def, Nat.reduceLeDiff, ↓reduceIte, List.length_take,
  List.length_drop, take_drop_cons, List.take_of_length_le, List.drop_eq_nil_of_le, List.length_cons,
  List.length_nil, write_wb_small, memLoad_ofL, keccakWord_ofL, ofNat_fromBE_wb, ofNat_sub_ofNat,
  wb_ofNat_fromBE, land_not31_ofNat, Nat.reduceDiv, read_ofL, $xs,*])

/-- A bound kept opaque so that `omega` (inside `mdisch`) does not grind on it. -/
def Bounded (n b : ℕ) : Prop := n < b

end ExporterEvm.Mem
