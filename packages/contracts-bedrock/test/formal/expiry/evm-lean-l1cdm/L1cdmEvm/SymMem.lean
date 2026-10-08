import Reasoning.Reach
import Reasoning.Solc

/-!
# Byte-level symbolic memory by reflection

The paths proved here write 32-byte words at offsets that are not multiples of 32 (ABI encodings
after a 4-byte selector, `bytes` payloads copied by solc's `copy_memory_to_memory` loop), so the
word-aligned memory view of `../evm-lean` is not enough. Instead every memory (and the inner
call's calldata) is the *denotation* of a list of byte descriptors:

* `Sym.c b` — the concrete byte `b`;
* `Sym.v k i` — byte `i` (big-endian, `0` = most significant) of the `k`-th word of an
  environment `env : List UInt256` that holds every symbolic word of the proof (calldata
  arguments, values returned by calls, addresses, …).

`Mem env l` is that byte array. The EVM's memory operations (`MSTORE` = `ByteArray.write`, `MLOAD`
= `memLoad`, call inputs = `readWithPadding`, `CALLDATACOPY`/`CALLDATALOAD` on a `Mem` calldata)
commute with the denotation (`write_Mem`, `memLoad_Mem`, `readWithPadding_Mem`, …), turning
into the list functions `swrite`/`sread` on descriptors. Descriptor lists are closed terms, so any
fact about them (e.g. "the call input is exactly this envelope") is decided by `decide +kernel`
(kernel evaluation; no `evm_kdecide`, no extra axioms).
-/

namespace L1cdmEvm.SymMem

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach

/-! ## Bytes of a word -/

/-- Big-endian byte `i` of a natural number seen as a 32-byte word. -/
def byteBE (n i : ℕ) : UInt8 := UInt8.ofNat (n / 256 ^ (31 - i) % 256)

theorem toBytes'_getD (n j : ℕ) :
    (toBytes' n).getD j 0 = UInt8.ofNat (n / 256 ^ j % 256) := by
  induction j generalizing n with
  | zero =>
    cases n with
    | zero => simp [toBytes']
    | succ n =>
      simp only [toBytes', List.getD_cons_zero, pow_zero, Nat.div_one]
      apply UInt8.toNat_inj.mp
      simp
      rfl
  | succ j ih =>
    cases n with
    | zero =>
      simp [toBytes']
    | succ n =>
      simp only [toBytes', List.getD_cons_succ]
      rw [ih]
      congr 2
      rw [Nat.div_div_eq_div_mul, pow_succ, Nat.mul_comm]

theorem toBytes'_length_le (w : UInt256) : (toBytes' w.toNat).length ≤ 32 :=
  toBytes'_UInt256_le w.val.isLt

/-- The bytes of `UInt256.toByteArray w`. -/
theorem toByteArray_toList (w : UInt256) :
    (UInt256.toByteArray w).data.toList = (List.range 32).map (byteBE w.toNat) := by
  have hL := toBytes'_length_le w
  set L := (toBytes' w.toNat).length with hLdef
  have hdata : (UInt256.toByteArray w).data.toList =
      List.replicate (32 - L) 0 ++ (toBytes' w.toNat).reverse := by
    unfold UInt256.toByteArray BE
    simp [ByteArray.data_append, Array.toList_append, byteArray_zeroes_toList, toBytesBigEndian,
      hLdef]
  rw [hdata]
  have hlen : (List.replicate (32 - L) (0 : UInt8) ++ (toBytes' w.toNat).reverse).length = 32 := by
    simp; omega
  apply List.ext_getElem
  · rw [hlen]; simp
  · intro i h1 h2
    rw [hlen] at h1
    simp only [List.getElem_map, List.getElem_range]
    unfold byteBE
    rw [← toBytes'_getD]
    by_cases hi : i < 32 - L
    · rw [List.getElem_append_left (by simp; omega)]
      simp only [List.getElem_replicate]
      rw [List.getD_eq_default _ _ (by omega)]
    · rw [List.getElem_append_right (by simp; omega)]
      simp only [List.length_replicate, List.getElem_reverse]
      rw [List.getD_eq_getElem _ _ (by omega)]
      congr 1
      omega

/-! ## Descriptors and their denotation -/

inductive Sym where
  | c (b : ℕ)
  | v (k i : ℕ)
  /-- An unknown byte (stale memory contents); never part of a fact that is used. -/
  | u
  deriving DecidableEq, Repr

def Sym.den (env : List UInt256) : Sym → UInt8
  | .c b => UInt8.ofNat b
  | .v k i => byteBE (env.getD k ⟨0⟩).toNat i
  | .u => 0

def toBA (l : List UInt8) : ByteArray := ⟨l.toArray⟩

/-- The byte array denoted by a descriptor list. -/
def Mem (env : List UInt256) (l : List Sym) : ByteArray := toBA (l.map (Sym.den env))

/-- The 32 descriptors of environment word `k`. -/
def vsyms (k : ℕ) : List Sym := (List.range 32).map (Sym.v k)

/-- The 32 descriptors of the constant word `n` (taken mod 2^256). -/
def csyms (n : ℕ) : List Sym := (List.range 32).map (fun i => Sym.c (n % 2 ^ 256 / 256 ^ (31 - i) % 256))

def zeros (n : ℕ) : List Sym := List.replicate n (.c 0)

/-- `MSTORE`-like write of `src` at offset `o` (zero-extending `dst` up to `o` if needed). -/
def swrite (src dst : List Sym) (o : ℕ) : List Sym :=
  dst.take o ++ zeros (o - dst.length) ++ src ++ dst.drop (o + src.length)

/-- Read `n` bytes at `o`, zero-padded past the end. -/
def sread (d : List Sym) (o n : ℕ) : List Sym :=
  (d.drop o).take n ++ zeros (n - ((d.drop o).take n).length)

theorem Mem_toList (env : List UInt256) (l : List Sym) :
    (Mem env l).data.toList = l.map (Sym.den env) := by
  simp [Mem, toBA]

theorem Mem_size (env : List UInt256) (l : List Sym) : (Mem env l).size = l.length := by
  simp [Mem, toBA, ByteArray.size]

theorem ba_ext {a b : ByteArray} (h : a.data.toList = b.data.toList) : a = b := by
  cases a; cases b; simp at h; rw [h]

theorem toByteArray_vsyms (env : List UInt256) (k : ℕ) :
    UInt256.toByteArray (env.getD k ⟨0⟩) = Mem env (vsyms k) := by
  apply ba_ext
  rw [toByteArray_toList, Mem_toList, vsyms, List.map_map]
  rfl

theorem toByteArray_csyms (env : List UInt256) (n : ℕ) :
    UInt256.toByteArray (UInt256.ofNat n) = Mem env (csyms n) := by
  apply ba_ext
  rw [toByteArray_toList, Mem_toList, csyms, List.map_map]
  apply List.map_congr_left
  intro i _
  simp only [Function.comp, Sym.den, byteBE]
  congr 2

theorem map_zeros (env : List UInt256) (n : ℕ) :
    (zeros n).map (Sym.den env) = List.replicate n 0 := by
  simp [zeros, Sym.den]

/-! ## Memory operations commute with the denotation -/

theorem toList_write (src dest : ByteArray) (sa da len : ℕ) (hlen : 0 < len)
    (hsrc : sa + len ≤ src.size) :
    (src.write sa dest da len).data.toList =
      dest.data.toList.take da ++ List.replicate (da - dest.size) 0 ++
        (src.data.toList.drop sa).take len ++ dest.data.toList.drop (da + len) := by
  have hs : sa + len ≤ src.data.size := hsrc
  unfold ByteArray.write
  rw [if_neg (by omega : ¬ len = 0), if_neg (show ¬ sa ≥ src.size by omega)]
  have e1 : min len (src.data.size - sa) = len := by omega
  have e2 : min dest.data.size (da + len) - (da + len) = 0 := by omega
  simp only [ByteArray.copySlice, ByteArray.data_append, ByteArray.zeroes, Array.toList_append,
    Array.toList_extract, ByteArray.size, e1, e2, Array.replicate_zero, Array.append_empty,
    Nat.add_zero, Array.size_append, Array.size_replicate, List.extract_eq_take_drop,
    Array.toList_replicate]
  generalize hD : dest.data.toList = D
  have hDl : dest.data.size = D.length := by rw [← hD]; simp
  rw [hDl]
  simp only [List.drop_zero, Nat.sub_zero, show sa + len - sa = len by omega]
  rw [List.take_append, List.take_replicate]
  congr 2
  · rw [Nat.min_self]
  · rcases Nat.lt_or_ge D.length da with h | h
    · rw [List.drop_append, List.drop_eq_nil_of_le (by omega)]
      simp only [List.nil_append, List.drop_replicate, List.take_replicate]
      rw [show min (D.length + (da - D.length) - (da + len)) (da - D.length - (da + len - D.length))
        = 0 by omega]
      rfl
    · rw [show da - D.length = 0 by omega]
      simp only [List.replicate_zero, List.append_nil, Nat.add_zero]
      exact List.take_of_length_le (by simp)

theorem sread_length (d : List Sym) (o n : ℕ) : (sread d o n).length = n := by
  simp [sread, zeros]

theorem map_sread (env : List UInt256) (d : List Sym) (o n : ℕ) :
    (sread d o n).map (Sym.den env) =
      ((d.map (Sym.den env)).drop o).take n ++
        List.replicate (n - (((d.map (Sym.den env)).drop o).take n).length) 0 := by
  simp [sread, map_zeros, List.map_take, List.map_drop]

theorem map_swrite (env : List UInt256) (src d : List Sym) (o : ℕ) :
    (swrite src d o).map (Sym.den env) =
      (d.map (Sym.den env)).take o ++ List.replicate (o - d.length) 0 ++ src.map (Sym.den env) ++
        (d.map (Sym.den env)).drop (o + src.length) := by
  simp [swrite, map_zeros, List.map_take, List.map_drop]

/-- **Write.** Writing the first `len` bytes of any byte array whose first `len` bytes are the
    denotation of `S` (an `MSTORE` of a word, or a call's output copied to memory) into a `Mem`. -/
theorem write_Mem (env : List UInt256) (src : ByteArray) (S D : List Sym) (sa o len : ℕ)
    (hlen : 0 < len) (hle : sa + len ≤ src.size)
    (hS : (src.data.toList.drop sa).take len = S.map (Sym.den env)) :
    src.write sa (Mem env D) o len = Mem env (swrite S D o) := by
  have hSl : S.length = len := by
    have := congrArg List.length hS
    simp only [List.length_take, List.length_drop, List.length_map, Array.length_toList] at this
    have hle' : sa + len ≤ src.data.size := hle
    omega
  apply ba_ext
  rw [toList_write _ _ _ _ _ hlen hle, Mem_toList, Mem_toList, map_swrite, hS, Mem_size, hSl]

theorem write_Mem_Mem (env : List UInt256) (S D : List Sym) (o : ℕ) (hlen : 0 < S.length) :
    (Mem env S).write 0 (Mem env D) o S.length = Mem env (swrite S D o) :=
  write_Mem env _ S D 0 o _ hlen (by rw [Mem_size]; omega)
    (by rw [Mem_toList, List.drop_zero, List.take_of_length_le (by simp)])

/-- **Copy from a `Mem` source** (`CALLDATACOPY` from a `Mem` calldata). -/
theorem copy_Mem (env : List UInt256) (S D : List Sym) (sa o len : ℕ) (hlen : 0 < len)
    (hle : sa + len ≤ S.length) :
    (Mem env S).write sa (Mem env D) o len = Mem env (swrite ((S.drop sa).take len) D o) :=
  write_Mem env _ _ D sa o len hlen (by rw [Mem_size]; exact hle)
    (by rw [Mem_toList, List.map_take, List.map_drop])

/-- **Read with padding** (call inputs, `KECCAK256` inputs, log data). -/
theorem readWithPadding_Mem (env : List UInt256) (D : List Sym) (o n : ℕ) (hn : n < 2 ^ 64) :
    (Mem env D).readWithPadding o n = Mem env (sread D o n) := by
  apply ba_ext
  rw [Mem_toList, map_sread]
  unfold ByteArray.readWithPadding ByteArray.readWithoutPadding
  rw [if_neg (by omega)]
  simp only [Mem_size]
  by_cases ho : o ≥ D.length
  · rw [if_pos ho]
    have : (List.map (Sym.den env) D).drop o = [] := List.drop_eq_nil_of_le (by simp; omega)
    rw [this]
    simp [ByteArray.zeroes, ByteArray.size]
  · rw [if_neg ho]
    simp only [ByteArray.data_append, Array.toList_append, ByteArray.zeroes, Array.toList_replicate,
      ByteArray.extract, ByteArray.copySlice, ByteArray.empty, ByteArray.size, Array.toList_extract,
      List.extract_eq_take_drop, Mem_toList]
    simp [Mem, toBA, List.take_drop]
    have h2 : min (o + min n D.length) D.length = min (o + n) D.length := by omega
    rw [h2]
    congr 2
    rcases Nat.le_total n D.length with h | h
    · rw [Nat.min_eq_left h]
    · rw [Nat.min_eq_right h, List.take_of_length_le (by simp),
        List.take_of_length_le (l := List.map _ D) (by simp; omega)]

/-! ## Words read from memory or calldata -/

theorem ofNat_toNat' (a : UInt256) : UInt256.ofNat a.toNat = a := by
  cases a with
  | mk v =>
    unfold UInt256.ofNat UInt256.toNat
    simp [Id.run]

/-- The word a 32-byte read decodes to (`MLOAD`, `CALLDATALOAD`). -/
def wordOf (b : ByteArray) : UInt256 := UInt256.ofNat (fromByteArrayBigEndian b)

theorem wordOf_zeros (env : List UInt256) (n : ℕ) : wordOf (Mem env (zeros n)) = ⟨0⟩ := by
  unfold wordOf fromByteArrayBigEndian fromBytesBigEndian
  rw [byteArray_toList_eq, Mem_toList, map_zeros]
  simp [List.reverse_replicate, fromBytes'_replicate_zero]
  rfl

/-- **`MLOAD`.** -/
theorem memLoad_Mem (env : List UInt256) (D : List Sym) (a : UInt256) :
    memLoad a (Mem env D) = wordOf (Mem env (sread D a.toNat 32)) := by
  unfold memLoad
  rw [readWithPadding_Mem _ _ _ _ (by norm_num)]
  split
  · rename_i h
    rw [Mem_size] at h
    have : sread D a.toNat 32 = zeros 32 := by
      simp [sread, List.drop_eq_nil_of_le h, zeros]
    rw [this, wordOf_zeros]
  · rfl

theorem wordOf_vsyms (env : List UInt256) (k : ℕ) :
    wordOf (Mem env (vsyms k)) = env.getD k ⟨0⟩ := by
  rw [← toByteArray_vsyms, wordOf, fromByteArrayBigEndian_toByteArray]
  exact ofNat_toNat' _

theorem wordOf_csyms (env : List UInt256) (n : ℕ) :
    wordOf (Mem env (csyms n)) = UInt256.ofNat n := by
  rw [← toByteArray_csyms, wordOf, fromByteArrayBigEndian_toByteArray, ofNat_toNat']

theorem toByteArray_wordOf (env : List UInt256) (S : List Sym) (h : S.length = 32) :
    UInt256.toByteArray (wordOf (Mem env S)) = Mem env S :=
  toByteArray_ofNat_fromByteArrayBigEndian_of_size (by rw [Mem_size, h])

/-- **`CALLDATALOAD`** from a `Mem` calldata. -/
theorem calldataload_Mem (env : List UInt256) (D : List Sym) (o : ℕ) (ho : o < 2 ^ 64) :
    uInt256OfByteArray ((Mem env D).readBytes o 32) = wordOf (Mem env (sread D o 32)) := by
  rw [uInt256OfByteArray_eq]
  congr 2
  apply ba_ext
  rw [Mem_toList, map_sread]
  unfold ByteArray.readBytes
  have ho' : o < 18446744073709551616 := ho
  simp only [show (decide (o < 2 ^ 64) && decide (32 < 2 ^ 64)) = true by simp [ho'], if_true]
  simp only [ByteArray.copySlice, ByteArray.data_append, Array.toList_append,
    Array.toList_extract, List.extract_eq_take_drop, ByteArray.zeroes, Array.toList_replicate]
  have he : ByteArray.empty.data = #[] := rfl
  simp only [he, Array.toList_empty, List.drop_nil, List.take_nil, List.nil_append, List.append_nil,
    Mem_toList, show o + 32 - o = 32 by omega]
  congr 2
  show 32 - (#[] ++ (Mem env D).data.extract o (o + 32) ++ #[].extract _ _).size = _
  simp [Mem_toList, Mem, toBA]

/-! ## Frames: partial knowledge of a memory at a (possibly symbolic) base offset

`Frame env m B D`: for every `i < D.length` with `D[i] ≠ u`, byte `B + i` of `m` (zero past the
end, as the EVM reads it) is the denotation of `D[i]`. `B` may be symbolic (solc's free memory
pointer after calls depends on the callees' return data sizes); the descriptors are relative. -/

def getB (m : ByteArray) (i : ℕ) : UInt8 := m.data.toList.getD i 0

def Frame (env : List UInt256) (m : ByteArray) (B : ℕ) (D : List Sym) : Prop :=
  ∀ i, D.getD i .u ≠ .u → getB m (B + i) = Sym.den env (D.getD i .u)

/-- `MSTORE`-like write into a descriptor list; a gap is filled with unknown bytes. -/
def swriteU (src dst : List Sym) (o : ℕ) : List Sym :=
  dst.take o ++ List.replicate (o - dst.length) .u ++ src ++ dst.drop (o + src.length)

/-- Read `n` descriptors at `o`; past the end they are unknown. -/
def sreadU (d : List Sym) (o n : ℕ) : List Sym :=
  (d.drop o).take n ++ List.replicate (n - ((d.drop o).take n).length) .u

def noU (l : List Sym) : Bool := l.all (· != .u)

theorem frame_nil (env : List UInt256) (m : ByteArray) (B : ℕ) : Frame env m B [] := by
  intro i h; simp at h

theorem list_getD_write {α : Type} (D S : List α) (z : α) (sa da len i : ℕ)
    (hs : sa + len ≤ S.length) :
    (D.take da ++ List.replicate (da - D.length) z ++ (S.drop sa).take len ++ D.drop (da + len)).getD i z =
      if da ≤ i ∧ i < da + len then S.getD (sa + (i - da)) z else D.getD i z := by
  simp only [List.getD_eq_getElem?_getD, List.getElem?_append, List.length_append, List.length_take,
    List.length_replicate, List.length_drop, List.getElem?_take, List.getElem?_drop,
    List.getElem?_replicate]
  have e1 : min da D.length + (da - D.length) = da := by omega
  have e2 : min len (S.length - sa) = len := by omega
  simp only [e1, e2]
  split_ifs <;> first | omega | rfl | (congr 2; omega) | (rw [List.getElem?_eq_none (by omega)]; rfl)

theorem size_eq_length (b : ByteArray) : b.size = b.data.toList.length := by
  rw [Array.length_toList]; rfl

theorem getB_write (src dest : ByteArray) (sa da len i : ℕ) (hlen : 0 < len)
    (hsrc : sa + len ≤ src.size) :
    getB (src.write sa dest da len) i =
      if da ≤ i ∧ i < da + len then getB src (sa + (i - da)) else getB dest i := by
  unfold getB
  rw [toList_write _ _ _ _ _ hlen hsrc, size_eq_length dest]
  exact list_getD_write _ _ _ _ _ _ _ (by rw [← size_eq_length]; exact hsrc)

theorem swriteU_getD (S D : List Sym) (o i : ℕ) :
    (swriteU S D o).getD i .u = if o ≤ i ∧ i < o + S.length then S.getD (i - o) .u else D.getD i .u := by
  have := list_getD_write D S Sym.u 0 o S.length i (by omega)
  simp only [List.drop_zero, List.take_length, Nat.zero_add] at this
  exact this

theorem sreadU_getD (D : List Sym) (o n j : ℕ) (hj : j < n) :
    (sreadU D o n).getD j .u = D.getD (o + j) .u := by
  unfold sreadU
  simp only [List.getD_eq_getElem?_getD, List.getElem?_append, List.length_take, List.length_drop,
    List.getElem?_take, List.getElem?_drop, List.getElem?_replicate]
  split_ifs <;> first | omega | rfl | (rw [List.getElem?_eq_none (by omega)]; rfl)

/-- Any write: the bytes `[o, o + len)` of the frame become `S` (bytes `sa…` of the source). -/
theorem frame_write {env : List UInt256} {m : ByteArray} {B : ℕ} {D : List Sym}
    (src : ByteArray) (S : List Sym) (sa o len : ℕ)
    (hF : Frame env m B D) (hlen : 0 < len) (hsrc : sa + len ≤ src.size) (hSl : S.length = len)
    (hS : ∀ j, S.getD j .u ≠ .u → getB src (sa + j) = Sym.den env (S.getD j .u)) :
    Frame env (src.write sa m (B + o) len) B (swriteU S D o) := by
  intro i hu
  rw [swriteU_getD] at hu ⊢
  rw [getB_write _ _ _ _ _ _ hlen hsrc]
  by_cases hi : o ≤ i ∧ i < o + S.length
  · rw [if_pos hi] at hu ⊢
    rw [if_pos (by omega), show sa + (B + i - (B + o)) = sa + (i - o) by omega]
    exact hS _ hu
  · rw [if_neg hi] at hu ⊢
    rw [if_neg (by omega)]
    exact hF _ hu

theorem getD_ne_u_lt {D : List Sym} {i : ℕ} (h : D.getD i .u ≠ .u) : i < D.length := by
  by_contra hc
  apply h
  rw [List.getD_eq_default _ _ (by omega)]

/-- A write outside the frame's range leaves it intact. -/
theorem frame_write_disjoint {env : List UInt256} {m : ByteArray} {B : ℕ} {D : List Sym}
    (src : ByteArray) (sa da len : ℕ) (hF : Frame env m B D) (hlen : 0 < len)
    (hsrc : sa + len ≤ src.size) (hdis : da + len ≤ B ∨ B + D.length ≤ da) :
    Frame env (src.write sa m da len) B D := by
  intro i hu
  have := getD_ne_u_lt hu
  rw [getB_write _ _ _ _ _ _ hlen hsrc, if_neg (by omega)]
  exact hF _ hu

theorem frame_mono {env : List UInt256} {m : ByteArray} {B : ℕ} {D D' : List Sym}
    (hF : Frame env m B D) (h : ∀ i, D'.getD i .u ≠ .u → D'.getD i .u = D.getD i .u) :
    Frame env m B D' := by
  intro i hu
  rw [h i hu]
  exact hF i (by rw [← h i hu]; exact hu)

/-! ### Reads -/

theorem readWithPadding_toList (m : ByteArray) (a n : ℕ) (hn : n < 2 ^ 64) :
    (m.readWithPadding a n).data.toList = (List.range n).map (fun j => getB m (a + j)) := by
  unfold ByteArray.readWithPadding ByteArray.readWithoutPadding
  rw [if_neg (by omega)]
  rw [size_eq_length]
  generalize hL : m.data.toList = L
  have hget : ∀ i, getB m i = L.getD i 0 := by intro i; unfold getB; rw [hL]
  simp only [hget]
  split
  · rename_i h
    simp only [ByteArray.data_append, Array.toList_append, ByteArray.zeroes, Array.toList_replicate]
    have he : ByteArray.empty.data.toList = [] := rfl
    rw [he, List.nil_append]
    apply List.ext_getElem
    · simp [ByteArray.size]
    · intro j h1 h2
      simp only [List.getElem_replicate, List.getElem_map, List.getElem_range]
      rw [List.getD_eq_default _ _ (by omega)]
  · rename_i h
    simp only [ByteArray.data_append, Array.toList_append, ByteArray.zeroes, Array.toList_replicate,
      ByteArray.extract, ByteArray.copySlice, Array.toList_extract, List.extract_eq_take_drop]
    have he : ByteArray.empty.data.toList = [] := rfl
    have he2 : ByteArray.empty.data.size = 0 := rfl
    simp only [he, he2, List.drop_nil, List.take_nil, List.nil_append, List.append_nil]
    simp only [ByteArray.size, Array.size_append, Array.size_extract, he2, hL]
    have hs : m.data.size = L.length := by rw [← hL]; simp
    rw [hs]
    apply List.ext_getElem
    · simp; omega
    · intro j h1 h2
      simp only [List.getElem_map, List.getElem_range]
      rw [List.getElem_append]
      split
      · simp only [List.getElem_take, List.getElem_drop]
        rw [List.getD_eq_getElem]
      · simp only [List.getElem_replicate]
        rw [List.getD_eq_default]
        simp at *; omega

theorem wordOf_of_zero (b : ByteArray) (n : ℕ) (h : b.data.toList = List.replicate n 0) :
    wordOf b = ⟨0⟩ := by
  unfold wordOf fromByteArrayBigEndian fromBytesBigEndian
  rw [byteArray_toList_eq, h]
  simp [List.reverse_replicate, fromBytes'_replicate_zero]
  rfl

/-- `MLOAD` is the decode of a padded 32-byte read (also past the end of memory). -/
theorem memLoad_eq (a : UInt256) (m : ByteArray) :
    memLoad a m = wordOf (m.readWithPadding a.toNat 32) := by
  unfold memLoad
  split
  · rename_i h
    symm
    apply wordOf_of_zero _ 32
    rw [readWithPadding_toList _ _ _ (by norm_num)]
    apply List.ext_getElem
    · simp
    · intro j h1 h2
      simp only [List.getElem_map, List.getElem_range, List.getElem_replicate, getB]
      rw [List.getD_eq_default]
      rw [size_eq_length] at h; omega
  · rfl

theorem frame_read {env : List UInt256} {m : ByteArray} {B : ℕ} {D : List Sym}
    (hF : Frame env m B D) (o n : ℕ) (hn : n < 2 ^ 64) (hk : noU (sreadU D o n) = true) :
    m.readWithPadding (B + o) n = Mem env (sreadU D o n) := by
  apply ba_ext
  rw [readWithPadding_toList _ _ _ hn, Mem_toList]
  have hlen : (sreadU D o n).length = n := by simp [sreadU]
  apply List.ext_getElem
  · simp [hlen]
  · intro j h1 h2
    simp only [List.getElem_map, List.getElem_range]
    have hj : j < n := by simpa using h1
    have hne : (sreadU D o n).getD j .u ≠ .u := by
      unfold noU at hk
      rw [List.all_eq_true] at hk
      have := hk ((sreadU D o n)[j]'(by omega)) (List.getElem_mem _)
      rw [List.getD_eq_getElem _ _ (by omega)]
      simpa using this
    rw [← List.getD_eq_getElem _ (Sym.u) (by omega)]
    rw [sreadU_getD _ _ _ _ hj] at hne ⊢
    rw [Nat.add_assoc]
    exact hF _ hne

theorem frame_memLoad {env : List UInt256} {m : ByteArray} {B : ℕ} {D : List Sym}
    (hF : Frame env m B D) (a : UInt256) (o : ℕ) (ha : a.toNat = B + o)
    (hk : noU (sreadU D o 32) = true) :
    memLoad a m = wordOf (Mem env (sreadU D o 32)) := by
  rw [memLoad_eq, ha, frame_read hF o 32 (by norm_num) hk]

/-! ### Bytes of the sources of writes -/

theorem getB_toByteArray (w : UInt256) (j : ℕ) (hj : j < 32) :
    getB (UInt256.toByteArray w) j = byteBE w.toNat j := by
  unfold getB
  rw [toByteArray_toList, List.getD_eq_getElem _ _ (by simpa using hj)]
  simp

theorem vsyms_getD (k j : ℕ) : (vsyms k).getD j .u = if j < 32 then .v k j else .u := by
  unfold vsyms
  split
  · rw [List.getD_eq_getElem _ _ (by simpa)]; simp
  · rw [List.getD_eq_default _ _ (by simpa using (by omega : 32 ≤ j))]

theorem csyms_getD (n j : ℕ) :
    (csyms n).getD j .u = if j < 32 then .c (n % 2 ^ 256 / 256 ^ (31 - j) % 256) else .u := by
  unfold csyms
  split
  · rw [List.getD_eq_getElem _ _ (by simpa)]; simp
  · rw [List.getD_eq_default _ _ (by simpa using (by omega : 32 ≤ j))]

theorem vsyms_length (k : ℕ) : (vsyms k).length = 32 := by simp [vsyms]
theorem csyms_length (n : ℕ) : (csyms n).length = 32 := by simp [csyms]

/-- `MSTORE` of an environment word. -/
theorem frame_mstore_var {env : List UInt256} {m : ByteArray} {B : ℕ} {D : List Sym}
    (hF : Frame env m B D) (k : ℕ) (w : UInt256) (hw : env.getD k ⟨0⟩ = w) (o : ℕ) :
    Frame env ((UInt256.toByteArray w).write 0 m (B + o) 32) B (swriteU (vsyms k) D o) := by
  apply frame_write _ _ 0 o 32 hF (by norm_num) (by rw [toByteArray_size]) (vsyms_length k)
  intro j hj
  rw [vsyms_getD] at hj ⊢
  split at hj
  · rename_i h
    rw [if_pos h, Nat.zero_add, getB_toByteArray _ _ h, ← hw]; rfl
  · exact absurd rfl hj

/-- `MSTORE` of a constant word. -/
theorem frame_mstore_const {env : List UInt256} {m : ByteArray} {B : ℕ} {D : List Sym}
    (hF : Frame env m B D) (n : ℕ) (w : UInt256) (hw : w = UInt256.ofNat n) (o : ℕ) :
    Frame env ((UInt256.toByteArray w).write 0 m (B + o) 32) B (swriteU (csyms n) D o) := by
  apply frame_write _ _ 0 o 32 hF (by norm_num) (by rw [toByteArray_size]) (csyms_length n)
  intro j hj
  rw [csyms_getD] at hj ⊢
  split at hj
  · rename_i h
    rw [if_pos h, Nat.zero_add, getB_toByteArray _ _ h, hw]
    simp only [Sym.den, byteBE]
    congr 2
  · exact absurd rfl hj

/-- The bytes of a word loaded from memory are the memory's bytes. -/
theorem getB_toByteArray_memLoad (a : UInt256) (m : ByteArray) (j : ℕ) (hj : j < 32) :
    getB (UInt256.toByteArray (memLoad a m)) j = getB m (a.toNat + j) := by
  rw [memLoad_eq, wordOf, toByteArray_ofNat_fromByteArrayBigEndian_of_size]
  · unfold getB
    rw [readWithPadding_toList _ _ _ (by norm_num), List.getD_eq_getElem _ _ (by simpa using hj)]
    simp [getB]
  · rw [size_eq_length, readWithPadding_toList _ _ _ (by norm_num)]; simp

/-- `MSTORE` of a word just loaded from the same frame (solc's copy loops). -/
theorem frame_mstore_load {env : List UInt256} {m : ByteArray} {B : ℕ} {D : List Sym}
    (hF : Frame env m B D) (a : UInt256) (oa : ℕ) (ha : a.toNat = B + oa) (o : ℕ) :
    Frame env ((UInt256.toByteArray (memLoad a m)).write 0 m (B + o) 32) B
      (swriteU (sreadU D oa 32) D o) := by
  apply frame_write _ _ 0 o 32 hF (by norm_num) (by rw [toByteArray_size]) (by simp [sreadU])
  intro j hj
  have hj32 : j < 32 := by
    by_contra hc; apply hj
    rw [List.getD_eq_default _ _ (by simp [sreadU]; omega)]
  rw [sreadU_getD _ _ _ _ hj32] at hj ⊢
  rw [Nat.zero_add, getB_toByteArray_memLoad _ _ _ hj32, ha, Nat.add_assoc]
  exact hF _ hj

/-- A call's output copied to memory: its first 32 bytes are the ABI word `w = env[k]`. -/
theorem frame_output {env : List UInt256} {m : ByteArray} {B : ℕ} {D : List Sym}
    (hF : Frame env m B D) (out : ByteArray) (k : ℕ)
    (hout : (out.data.toList).take 32 = (UInt256.toByteArray (env.getD k ⟨0⟩)).data.toList)
    (h32 : 32 ≤ out.size) (o : ℕ) :
    Frame env (out.write 0 m (B + o) 32) B (swriteU (vsyms k) D o) := by
  apply frame_write _ _ 0 o 32 hF (by norm_num) (by omega) (vsyms_length k)
  intro j hj
  rw [vsyms_getD] at hj ⊢
  split at hj
  · rename_i h
    rw [if_pos h, Nat.zero_add]
    have e1 : getB out j = getB (UInt256.toByteArray (env.getD k ⟨0⟩)) j := by
      unfold getB
      rw [List.getD_eq_getElem?_getD, List.getD_eq_getElem?_getD, ← hout]
      simp [List.getElem?_take, h]
    rw [e1, getB_toByteArray _ _ h]; rfl
  · exact absurd rfl hj

/-- `CALLDATACOPY` from a calldata that is a `Mem`. -/
theorem frame_copy_Mem {env : List UInt256} {m : ByteArray} {B : ℕ} {D : List Sym}
    (hF : Frame env m B D) (CD : List Sym) (sa o len : ℕ) (hlen : 0 < len)
    (hle : sa + len ≤ CD.length) :
    Frame env ((Mem env CD).write sa m (B + o) len) B (swriteU ((CD.drop sa).take len) D o) := by
  apply frame_write _ _ sa o len hF hlen (by rw [Mem_size]; exact hle) (by simp; omega)
  intro j hj
  have hjl : j < len := by
    have := getD_ne_u_lt hj; simp at this; omega
  rw [List.getD_eq_getElem _ _ (by simp; omega)] at hj ⊢
  unfold getB
  rw [Mem_toList, List.getD_eq_getElem _ _ (by simp; omega)]
  simp

/-! ### Assembling byte strings -/

theorem Mem_append (env : List UInt256) (a b : List Sym) :
    Mem env (a ++ b) = Mem env a ++ Mem env b := by
  apply ba_ext
  simp [Mem_toList, ByteArray.data_append]

theorem Mem_csyms (env : List UInt256) (n : ℕ) : Mem env (csyms n) = UInt256.toByteArray (UInt256.ofNat n) :=
  (toByteArray_csyms env n).symm

theorem Mem_vsyms (env : List UInt256) (k : ℕ) : Mem env (vsyms k) = UInt256.toByteArray (env.getD k ⟨0⟩) :=
  (toByteArray_vsyms env k).symm

/-- The 4 descriptors of a selector. -/
def csel (s : ℕ) : List Sym := [.c (s / 2^24), .c (s / 2^16 % 256), .c (s / 2^8 % 256), .c (s % 256)]

theorem Mem_zeros (env : List UInt256) (n : ℕ) : Mem env (zeros n) = ByteArray.zeroes n := by
  apply ba_ext
  rw [Mem_toList, map_zeros]
  simp [ByteArray.zeroes]

/-! ### The free memory pointer (`mload(0x40)`) as a one-word frame at 64 -/

/-- `memLoad 64 m = w`, as a frame (so that writes elsewhere preserve it). -/
def Fmp (m : ByteArray) (w : UInt256) : Prop := Frame [w] m 64 (vsyms 0)

theorem fmp_load {m : ByteArray} {w : UInt256} (h : Fmp m w) (a : UInt256) (ha : a.toNat = 64) :
    memLoad a m = w := by
  rw [frame_memLoad h a 0 ha (by decide +kernel)]
  rw [show sreadU (vsyms 0) 0 32 = vsyms 0 by decide +kernel, wordOf_vsyms]
  rfl

theorem fmp_store (m : ByteArray) (w : UInt256) (off : ℕ) (hoff : off = 64) :
    Fmp ((UInt256.toByteArray w).write 0 m off 32) w := by
  have := frame_mstore_var (frame_nil [w] m 64) 0 w rfl 0
  rw [show swriteU (vsyms 0) [] 0 = vsyms 0 by decide +kernel] at this
  subst hoff; exact this

theorem fmp_keep {m : ByteArray} {w : UInt256} (h : Fmp m w) (src : ByteArray) (sa da len : ℕ)
    (hlen : 0 < len) (hsrc : sa + len ≤ src.size) (hdis : da + len ≤ 64 ∨ 96 ≤ da) :
    Fmp (src.write sa m da len) w :=
  frame_write_disjoint src sa da len h hlen hsrc (by rw [vsyms_length]; omega)

end L1cdmEvm.SymMem
