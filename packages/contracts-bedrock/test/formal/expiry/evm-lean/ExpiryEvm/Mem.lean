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
