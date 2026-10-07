import Reasoning.Reach
import Reasoning.Solc

/-! # Word-aligned memory

All memory the `expireMessage` path touches is a sequence of 32-byte words at word-aligned
offsets. `wordsMem ws` is that memory; writes, reads and keccak inputs at word offsets reduce to
list operations. -/

namespace ExpiryEvm.Mem

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach

def wordsMem : List UInt256 → ByteArray
  | [] => ByteArray.empty
  | w :: ws => UInt256.toByteArray w ++ wordsMem ws

theorem wordsMem_size (ws : List UInt256) : (wordsMem ws).size = 32 * ws.length := by
  induction ws with
  | nil => rfl
  | cons w ws ih =>
    simp only [wordsMem, ByteArray.size_append, toByteArray_size, ih, List.length_cons]; ring

theorem extract_append_right_off (A B : ByteArray) (x y : ℕ) :
    (A ++ B).extract (A.size + x) (A.size + y) = B.extract x y := by
  apply ByteArray.ext
  simp only [ByteArray.data_extract, ByteArray.data_append]
  rw [Array.extract_append]
  simp

theorem extract_same (A : ByteArray) (x : ℕ) : A.extract x x = ByteArray.empty := by
  apply ByteArray.ext
  simp

theorem wordsMem_extract (ws : List UInt256) (i j : ℕ) (hij : i ≤ j) (hj : j ≤ ws.length) :
    (wordsMem ws).extract (32 * i) (32 * j) = wordsMem ((ws.take j).drop i) := by
  induction ws generalizing i j with
  | nil =>
    simp at hj; subst hj; have : i = 0 := by omega
    subst this; simp [wordsMem]
  | cons w ws ih =>
    rcases j with _ | j
    · have : i = 0 := by omega
      subst this; simp [wordsMem]
    · rcases i with _ | i
      · simp only [wordsMem, List.take_succ_cons, List.drop_zero]
        have hsz : (UInt256.toByteArray w).size = 32 := toByteArray_size w
        rw [show 32 * 0 = 0 by rfl, extract_append_span _ _ _ _ (by omega) (by omega), hsz,
          toByteArray_extract_all, show 32 * (j + 1) - 32 = 32 * j by omega]
        have := ih 0 j (by omega) (by simp at hj; omega)
        simp only [Nat.mul_zero, List.drop_zero] at this
        rw [this]
      · simp only [wordsMem, List.take_succ_cons, List.drop_succ_cons]
        have hsz : (UInt256.toByteArray w).size = 32 := toByteArray_size w
        have h1 : 32 * (i + 1) = (UInt256.toByteArray w).size + 32 * i := by rw [hsz]; ring
        have h2 : 32 * (j + 1) = (UInt256.toByteArray w).size + 32 * j := by rw [hsz]; ring
        rw [h1, h2, extract_append_right_off]
        exact ih i j (by omega) (by simp at hj; omega)

theorem wordsMem_extract_all (ws : List UInt256) :
    (wordsMem ws).extract 0 (wordsMem ws).size = wordsMem ws := byteArray_extract_self _

theorem wordsMem_append (a b : List UInt256) : wordsMem (a ++ b) = wordsMem a ++ wordsMem b := by
  induction a with
  | nil => simp [wordsMem]
  | cons w a ih => simp only [List.cons_append, wordsMem, ih, ByteArray.append_assoc]

/-- Writing a word at an existing word slot `i`. -/
theorem wordsMem_write (ws : List UInt256) (i : ℕ) (h : i < ws.length) (w : UInt256) (off : ℕ)
    (hoff : off = 32 * i) :
    (UInt256.toByteArray w).write 0 (wordsMem ws) off 32 = wordsMem (ws.set i w) := by
  subst hoff
  rw [write32_eq _ _ _ (by rw [toByteArray_size]) (by rw [wordsMem_size]; omega),
    toByteArray_extract_all, wordsMem_size]
  have := wordsMem_extract ws 0 i (by omega) (by omega)
  simp only [Nat.mul_zero] at this
  rw [show 32 * i + 32 = 32 * (i + 1) by ring, this,
    wordsMem_extract ws (i + 1) ws.length (by omega) le_rfl]
  rw [List.set_eq_take_append_cons_drop, if_pos h]
  simp only [List.drop_zero, List.take_length, wordsMem_append, wordsMem, ByteArray.append_assoc]

/-- Writing a word right at the end. -/
theorem wordsMem_write_end (ws : List UInt256) (w : UInt256) (off : ℕ) (hoff : off = 32 * ws.length) :
    (UInt256.toByteArray w).write 0 (wordsMem ws) off 32 = wordsMem (ws ++ [w]) := by
  subst hoff
  rw [toByteArray_write_eq _ _ _ (le_of_eq (wordsMem_size ws))
    (by rw [wordsMem_size, Nat.sub_self]; exact USize.size_pos),
    wordsMem_size, Nat.sub_self, zeroes_zero rfl, wordsMem_append]
  simp [wordsMem]

theorem wordsMem_read32 (ws : List UInt256) (i : ℕ) (h : i < ws.length) (off : ℕ)
    (hoff : off = 32 * i) :
    (wordsMem ws).readWithPadding off 32 = UInt256.toByteArray ws[i] := by
  subst hoff
  rw [readWithPadding_eq_extract _ _ (by rw [wordsMem_size]; omega),
    show 32 * i + 32 = 32 * (i + 1) by ring, wordsMem_extract ws i (i + 1) (by omega) (by omega)]
  rw [List.take_add_one, List.drop_append_of_le_length (by simp; omega)]
  simp [List.getElem?_eq_getElem h, wordsMem]

end ExpiryEvm.Mem

namespace ExpiryEvm.Mem

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach

theorem ofNat_toNat (a : UInt256) : UInt256.ofNat a.toNat = a := by
  cases a with
  | mk v =>
    unfold UInt256.ofNat UInt256.toNat
    simp [Id.run]

theorem wordsMem_read4 (ws : List UInt256) (i : ℕ) (h : i < ws.length) (off : ℕ)
    (hoff : off = 32 * i) :
    (wordsMem ws).readWithPadding off 4 = (UInt256.toByteArray ws[i]).extract 0 4 := by
  subst hoff
  rw [readWithPadding_eq_extract' _ _ _ (by norm_num) (by norm_num) (by rw [wordsMem_size]; omega)]
  have h32 := wordsMem_read32 ws i h (32 * i) rfl
  rw [readWithPadding_eq_extract _ _ (by rw [wordsMem_size]; omega)] at h32
  rw [← h32]
  apply ByteArray.ext
  simp only [ByteArray.data_extract, Array.extract_extract]
  congr 1 <;> omega

theorem wordsMem_read64 (a b : UInt256) (rest : List UInt256) :
    (wordsMem (a :: b :: rest)).readWithPadding 0 64 =
      UInt256.toByteArray a ++ UInt256.toByteArray b := by
  rw [readWithPadding_eq_extract' _ _ _ (by norm_num) (by norm_num)
    (by rw [wordsMem_size]; simp; omega)]
  have := wordsMem_extract (a :: b :: rest) 0 2 (by omega) (by simp)
  simp only [Nat.mul_zero, List.take_succ_cons, List.take_zero, List.drop_zero] at this
  rw [show (0 + 64 : ℕ) = 32 * 2 by rfl, this]
  simp [wordsMem]

/-- `MLOAD` at a word slot. -/
theorem memLoad_wordsMem (ws : List UInt256) (i : ℕ) (h : i < ws.length) (a : UInt256)
    (ha : a.toNat = 32 * i) :
    memLoad a (wordsMem ws) = ws[i] := by
  unfold memLoad
  rw [if_neg (by rw [ha, wordsMem_size]; omega), ha, wordsMem_read32 ws i h _ rfl,
    fromByteArrayBigEndian_toByteArray]
  exact ofNat_toNat _

theorem loadedWord_wordsMem (ws : List UInt256) (i : ℕ) (h : i < ws.length) (a : UInt256)
    (ha : a.toNat = 32 * i) :
    loadedWord (wordsMem ws) a = ws[i] := by
  unfold loadedWord
  rw [if_neg (by rw [ha, wordsMem_size]; omega), ha, wordsMem_read32 ws i h _ rfl,
    fromByteArrayBigEndian_toByteArray]
  exact ofNat_toNat _

theorem keccakWord_wordsMem (a b : UInt256) (rest : List UInt256) :
    keccakWord ⟨0⟩ (UInt256.ofNat 64) (wordsMem (a :: b :: rest)) = solcMappingSlot b a := by
  unfold keccakWord solcMappingSlot
  rw [show (⟨0⟩ : UInt256).toNat = 0 from rfl, show (UInt256.ofNat 64).toNat = 64 from rfl,
    wordsMem_read64, mappingSlot_single]

theorem zeroes32_eq : ByteArray.zeroes 32 = UInt256.toByteArray ⟨0⟩ := by
  decide +kernel

/-- Writing a word one word past the end (the gap is zero-filled). -/
theorem wordsMem_write_gap1 (ws : List UInt256) (w : UInt256) (off : ℕ)
    (hoff : off = 32 * (ws.length + 1)) :
    (UInt256.toByteArray w).write 0 (wordsMem ws) off 32 = wordsMem (ws ++ [⟨0⟩, w]) := by
  subst hoff
  rw [toByteArray_write_eq _ _ _ (by rw [wordsMem_size]; omega)
    (by rw [wordsMem_size, show 32 * (ws.length + 1) - 32 * ws.length = 32 by omega]
        exact lt_usize _ (by norm_num)),
    wordsMem_size, show 32 * (ws.length + 1) - 32 * ws.length = 32 by omega, zeroes32_eq,
    wordsMem_append]
  simp [wordsMem, ByteArray.append_assoc]

theorem solcFreePtrMem_eq_wordsMem : solcFreePtrMem = wordsMem [⟨0⟩, ⟨0⟩, ⟨128⟩] := by
  decide +kernel

end ExpiryEvm.Mem

namespace ExpiryEvm.Mem

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach

/-- `toByteArray_write_eq` without its (unused) `USize` bound on the gap. -/
theorem toByteArray_write_eq' (v : UInt256) (mem : ByteArray) (off : ℕ)
    (hoff : mem.size ≤ off) :
    (UInt256.toByteArray v).write 0 mem off 32
      = mem ++ ByteArray.zeroes (off - mem.size) ++ UInt256.toByteArray v := by
  have hsz : (UInt256.toByteArray v).data.size = 32 := UInt256.toByteArrayWithSizeProof v |>.2
  have hpz : (ByteArray.zeroes (off - mem.size)).data.size = off - mem.size := by
    rw [show (ByteArray.zeroes (off - mem.size)).data.size
          = (ByteArray.zeroes (off - mem.size)).size from rfl,
        ByteArray_zeroes_size]
  apply ByteArray.ext
  unfold ByteArray.write
  rw [if_neg (by decide : ¬ ((32:ℕ) = 0)),
      if_neg (show ¬ (0 ≥ (UInt256.toByteArray v).size) from by
                rw [show (UInt256.toByteArray v).size = 32 from hsz]; omega)]
  simp only [ByteArray.data_copySlice, ByteArray.data_append]
  have hv : v.toByteArray.size = 32 := hsz
  have hDsz : (mem.data ++ (ByteArray.zeroes (off - mem.size)).data).size = off := by
    rw [Array.size_append, hpz]; show mem.size + (off - mem.size) = off; omega
  rw [hv, show (min 32 (32 - 0) : ℕ) = 32 from rfl,
      show min mem.size (off + 32) - (off + 32) = 0 from by omega,
      show (ByteArray.zeroes 0).data = (#[] : Array UInt8) from by
        rw [zeroes_zero (n := 0) (by rfl)]; rfl]
  rw [Array.append_empty]
  rw [Array.extract_eq_self_of_le (by rw [hDsz]),
      Array.extract_eq_self_of_le (show v.toByteArray.data.size ≤ 0 + (32 + 0) from by rw [hsz]),
      Array.extract_eq_empty_of_le (by rw [hDsz]; omega),
      Array.append_empty]

theorem zeroes_add (a b : ℕ) : ByteArray.zeroes (a + b) = ByteArray.zeroes a ++ ByteArray.zeroes b := by
  apply ByteArray.ext
  simp only [ByteArray.zeroes, ByteArray.data_append]
  first
  | exact (Array.replicate_append_replicate).symm
  | exact (Array.append_replicate_replicate).symm
  | simp [Array.replicate_append_replicate]

theorem zeroes_words (k : ℕ) : ByteArray.zeroes (32 * k) = wordsMem (List.replicate k ⟨0⟩) := by
  induction k with
  | zero => apply ByteArray.ext; simp [ByteArray.zeroes, wordsMem]
  | succ k ih =>
    rw [show 32 * (k + 1) = 32 + 32 * k by ring, zeroes_add, ih, zeroes32_eq]
    simp [List.replicate_succ, wordsMem]

/-- Writing a word `k` words past the end (the gap is zero-filled). -/
theorem wordsMem_write_gap (ws : List UInt256) (w : UInt256) (k off : ℕ)
    (hoff : off = 32 * (ws.length + k)) :
    (UInt256.toByteArray w).write 0 (wordsMem ws) off 32 =
      wordsMem (ws ++ List.replicate k ⟨0⟩ ++ [w]) := by
  subst hoff
  rw [toByteArray_write_eq' _ _ _ (by rw [wordsMem_size]; omega), wordsMem_size,
    show 32 * (ws.length + k) - 32 * ws.length = 32 * k by omega, zeroes_words,
    wordsMem_append, wordsMem_append]
  simp [wordsMem]

/-- A 32-byte copy of a byte array whose first word is `w` writes `w`. -/
theorem write_prefix32 (o base : ByteArray) (w : UInt256) (off : ℕ)
    (hsz : 32 ≤ o.size) (he : o.extract 0 32 = UInt256.toByteArray w) (hoff : off ≤ base.size) :
    o.write 0 base off 32 = (UInt256.toByteArray w).write 0 base off 32 := by
  rw [write32_eq _ _ _ hsz hoff, write32_eq _ _ _ (by rw [toByteArray_size]) hoff, he,
    toByteArray_extract_all]

end ExpiryEvm.Mem

namespace ExpiryEvm.Mem

open Ethereum

/-- The memory layout after the second call: scratch (2 words), free pointer `p`, a zero word,
    the first call's decoded word `w1` at 0x80, `k` zero words, and the second call's word `w2`
    at `p_old = 32 * (5 + k)`. -/
def memList (p w1 : UInt256) (k : ℕ) (w2 : UInt256) : List UInt256 :=
  [⟨0⟩, ⟨0⟩, p, ⟨0⟩, w1] ++ List.replicate k ⟨0⟩ ++ [w2]

theorem memList_length (p w1 : UInt256) (k : ℕ) (w2 : UInt256) :
    (memList p w1 k w2).length = 6 + k := by
  simp [memList]; omega

theorem memList_get2 (p w1 : UInt256) (k : ℕ) (w2 : UInt256) (h : 2 < (memList p w1 k w2).length) :
    (memList p w1 k w2)[2] = p := by
  simp [memList]

theorem memList_getLast (p w1 : UInt256) (k : ℕ) (w2 : UInt256)
    (h : 5 + k < (memList p w1 k w2).length) : (memList p w1 k w2)[5 + k] = w2 := by
  unfold memList
  rw [List.getElem_append_right (by simp; omega)]
  simp

theorem memList_setLast (p w1 : UInt256) (k : ℕ) (w2 w : UInt256) :
    (memList p w1 k w2).set (5 + k) w = memList p w1 k w := by
  unfold memList
  rw [List.set_append_right _ _ (by simp; omega)]
  simp only [List.length_append, List.length_cons, List.length_nil, List.length_replicate]
  rw [show 5 + k - (0 + 1 + 1 + 1 + 1 + 1 + k) = 0 by omega]
  rfl

theorem memList_set2 (p w1 : UInt256) (k : ℕ) (w2 p' : UInt256) :
    (memList p w1 k w2).set 2 p' = memList p' w1 k w2 := by
  simp [memList]

end ExpiryEvm.Mem
