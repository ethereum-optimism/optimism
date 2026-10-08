import ExporterEvm.TraceEntry

/-!
# solc's `copy_memory_to_memory` loop with a symbolic trip count

The ABI encoder of `bytes memory` (pc 1132, `abi_encode_t_bytes_memory_ptr`) stores the length,
then copies the payload word by word (pc 1142 loop head, pc 1151 body), then zeroes the word after
the payload if the length is not a multiple of 32 (pc 1170/1179) and returns the rounded end
(pc 1188). For `_message` the trip count `⌈len/32⌉` is symbolic, so the loop is proved by
induction (`copy_loop`), not unrolled. Memory is `ofL (Pre ++ D)`: `Pre` is everything up to and
including the length word at `pos`, `D` the part of the payload copied so far.
-/

namespace ExporterEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach exporterBlocks Mem

theorem toNat_small {n : ℕ} (h : n < 2 ^ 200) : (UInt256.ofNat n).toNat = n :=
  Words.toNat_ofNat_lt (by omega)

/-- The word the loop body loads at iteration `j` is the `j`-th word of the source window. -/
theorem loop_read (Pre X Mp : List UInt8) (v j n : ℕ) (hj : j < n)
    (hsrc : (Pre.drop (v + 32)).take (32 * n) = Mp) (hv : v + 32 + 32 * n ≤ Pre.length) :
    ((Pre ++ X).drop (32 + (v + 32 * j))).take 32 = (Mp.drop (32 * j)).take 32 := by
  have h1 : ((Pre ++ X).drop (32 + (v + 32 * j))).take 32 = (Pre.drop (32 + (v + 32 * j))).take 32 := by
    rw [List.drop_append_of_le_length (by omega)]
    exact List.take_append_of_le_length (by simp only [List.length_drop]; omega)
  rw [h1, ← hsrc, List.drop_take, List.take_take, List.drop_drop]
  rw [show min 32 (32 * n - 32 * j) = 32 by omega, show v + 32 + 32 * j = 32 + (v + 32 * j) by omega]

theorem take_succ_words (Mp : List UInt8) (j : ℕ) :
    Mp.take (32 * j) ++ (Mp.drop (32 * j)).take 32 = Mp.take (32 * (j + 1)) := by
  rw [show 32 * (j + 1) = 32 * j + 32 by ring, List.take_add]

/-- One iteration of the loop body on list memory. -/
theorem body_mem (Pre Mp : List UInt8) (v pos j n : ℕ) (hj : j < n)
    (hsrc : (Pre.drop (v + 32)).take (32 * n) = Mp) (hv : v + 32 + 32 * n ≤ Pre.length)
    (hpre : Pre.length = pos + 32) (hb : pos + 64 + 32 * n < 2 ^ 200) :
    (memLoad (UInt256.ofNat (32 + (v + 32 * j))) (ofL (Pre ++ Mp.take (32 * j)))).toByteArray.write 0
      (ofL (Pre ++ Mp.take (32 * j))) (UInt256.ofNat (32 + (32 * j + pos))).toNat 32 =
    ofL (Pre ++ Mp.take (32 * (j + 1))) := by
  have hMp : Mp.length = 32 * n := by rw [← hsrc]; simp; omega
  have hlen : (Pre ++ Mp.take (32 * j)).length = pos + 32 + 32 * j := by
    simp [hpre, hMp]; omega
  rw [memLoad_ofL _ _ (by rw [toNat_small (by omega), hlen]; omega), toNat_small (by omega),
    loop_read Pre _ Mp v j n hj hsrc hv, write_wb, toNat_small (by omega)]
  have hl : ((Mp.drop (32 * j)).take 32).length = 32 := by simp [hMp]; omega
  rw [wb_ofNat_fromBE _ hl, List.take_of_length_le (l := Pre ++ Mp.take (32 * j)) (by rw [hlen]; omega),
    List.drop_eq_nil_of_le (as := Pre ++ Mp.take (32 * j)) (by rw [hlen]; omega),
    show 32 + (32 * j + pos) - (Pre ++ Mp.take (32 * j)).length = 0 by rw [hlen]; omega,
    List.replicate_zero, List.append_nil, List.append_nil, List.append_assoc, take_succ_words]

/-- **The copy loop, by induction.** From the loop head with `i = 32·j` and the first `32·j`
    payload bytes copied, the loop exits (pc 1170) with `i = 32·n` and the whole window `Mp`
    (`32·n` bytes read from `v + 32`) copied after `Pre`. -/
theorem copy_loop {ee : ExecutionEnv} {g : Sat256} {s0 : State} {rd : ByteArray} {σ : AccountMap}
    {R : List UInt256} {Pre Mp : List UInt8} {v pos L n : ℕ} {Lw : UInt256}
    (hLw : Lw.toNat = L) (hn : L ≤ 32 * n) (hn' : 32 * n < L + 32)
    (hsrc : (Pre.drop (v + 32)).take (32 * n) = Mp) (hv : v + 32 + 32 * n ≤ Pre.length)
    (hpre : Pre.length = pos + 32) (hb : pos + 64 + 32 * n < 2 ^ 200) :
    ∀ m j, j + m = n → ∀ aw k C,
      RD exporterRuntime ee g s0 (UInt256.ofNat 1142)
        (UInt256.ofNat (32 * j) :: Lw :: UInt256.ofNat 0 :: UInt256.ofNat v :: UInt256.ofNat pos :: R)
        (ofL (Pre ++ Mp.take (32 * j))) aw rd σ k C → R.length ≤ 1000 →
      ∃ aw' k' C', RD exporterRuntime ee g s0 (UInt256.ofNat 1170)
        (UInt256.ofNat (32 * n) :: Lw :: UInt256.ofNat 0 :: UInt256.ofNat v :: UInt256.ofNat pos :: R)
        (ofL (Pre ++ Mp)) aw' rd σ k' C' := by
  have hMp : Mp.length = 32 * n := by rw [← hsrc]; simp; omega
  intro m
  induction m with
  | zero =>
    intro j hj aw k C h hR
    simp only [Nat.add_zero] at hj
    subst hj
    rw [List.take_of_length_le (by omega)] at h
    obtain ⟨aw', k', C', h'⟩ := exporter_block_1142_taken_packed (by simp; omega)
      (by rw [Words.isZero_lt_ne0, hLw, toNat_small (by omega)]; omega) (by kjump_dest) h
    exact ⟨aw', k', C', h'⟩
  | succ m ih =>
    intro j hj aw k C h hR
    have hjn : j < n := by omega
    obtain ⟨_, _, _, r1⟩ := exporter_block_1142_fallthrough_packed (by simp; omega)
      (by rw [Words.isZero_lt_eq0, hLw, toNat_small (by omega)]; omega) h
    obtain ⟨aw2, k2, C2, r2⟩ := exporter_block_1151_packed (by simp; omega) (by kjump_dest) r1
    simp only [exporter_block_1151_stack, exporter_block_1151_memory, ofNat_add_ofNat] at r2
    rw [body_mem Pre Mp v pos j n hjn hsrc hv hpre hb, show 32 + 32 * j = 32 * (j + 1) by ring] at r2
    exact ih (j + 1) (by omega) _ _ _ r2 hR

/-- The payload as it stands after the loop's tail: the copied window `Mp`, or, if the length
    `L` is not a multiple of 32, its first `L` bytes followed by the zero word solc writes. -/
def padTail (Mp : List UInt8) (L n : ℕ) : List UInt8 :=
  if L < 32 * n then Mp.take L ++ List.replicate 32 0 else Mp

theorem wb_zero : wb (UInt256.ofNat 0) = List.replicate 32 0 := by decide +kernel

/-- **After the loop** (pc 1170 → return pc `ret`): the zero word for a partial last word, and
    the rounded end `pos + 32 + 32·n` returned on the stack. -/
theorem copy_tail {ee : ExecutionEnv} {g : Sat256} {s0 : State} {rd : ByteArray} {σ : AccountMap}
    {R : List UInt256} {Pre Mp : List UInt8} {v pos L n : ℕ} {Lw ret : UInt256} {aw : UInt256} {k C : ℕ}
    (hLw : Lw.toNat = L) (hn : L ≤ 32 * n) (hn' : 32 * n < L + 32) (hMp : Mp.length = 32 * n)
    (hpre : Pre.length = pos + 32) (hb : pos + 64 + 32 * n < 2 ^ 200) (hR : R.length ≤ 1000)
    (hret : (D_J exporterRuntime 0).contains ret = true)
    (h : RD exporterRuntime ee g s0 (UInt256.ofNat 1170)
        (UInt256.ofNat (32 * n) :: Lw :: UInt256.ofNat 0 :: UInt256.ofNat v :: UInt256.ofNat pos :: ret :: R)
        (ofL (Pre ++ Mp)) aw rd σ k C) :
    ∃ aw' k' C', RD exporterRuntime ee g s0 ret (UInt256.ofNat (pos + 32 + 32 * n) :: R)
        (ofL (Pre ++ padTail Mp L n)) aw' rd σ k' C' := by
  have hLw' : Lw = UInt256.ofNat L := by rw [Words.eq_ofNat_toNat Lw, hLw]
  subst hLw'
  have hL : L < 2 ^ 200 := by omega
  have hround : UInt256.land (UInt256.ofNat
      115792089237316195423570985008687907853269984665640564039457584007913129639904)
      (UInt256.ofNat 31 + UInt256.ofNat L) = UInt256.ofNat (32 * n) := by
    rw [ofNat_add_ofNat, land_not31_ofNat _ (by omega)]
    congr 1; omega
  have hend : UInt256.ofNat 32 + (UInt256.ofNat (32 * n) + UInt256.ofNat pos) =
      UInt256.ofNat (pos + 32 + 32 * n) := by
    rw [ofNat_add_ofNat, ofNat_add_ofNat]; congr 1; omega
  by_cases hp : L < 32 * n
  · obtain ⟨_, _, _, r1⟩ := exporter_block_1170_fallthrough_packed (by simp; omega)
      (by rw [Words.isZero_gt_eq0, toNat_small hL, toNat_small (by omega)] <;> exact hp) h
    obtain ⟨_, _, _, r2⟩ := exporter_block_1179_packed (by simp; omega) r1
    simp only [exporter_block_1179_memory, ofNat_add_ofNat] at r2
    have hlen : (Pre ++ Mp).length = pos + 32 + 32 * n := by simp only [List.length_append, hpre, hMp]; try omega
    have ht : (Pre ++ Mp).take (pos + L + 32) = Pre ++ Mp.take L := by
      rw [List.take_append, List.take_of_length_le (by omega), hpre,
        show pos + L + 32 - (pos + 32) = L by omega]
    rw [write_wb, toNat_small (by omega), wb_zero, ht,
      List.drop_eq_nil_of_le (as := Pre ++ Mp) (by rw [hlen]; omega),
      show pos + L + 32 - (Pre ++ Mp).length = 0 by rw [hlen]; omega,
      List.replicate_zero, List.append_nil, List.append_nil, List.append_assoc] at r2
    obtain ⟨_, _, _, r3⟩ := exporter_block_1188_packed (by simp; omega) hret r2
    simp only [exporter_block_1188_stack, hround, hend] at r3
    unfold padTail; rw [if_pos hp]
    exact ⟨_, _, _, r3⟩
  · obtain ⟨_, _, _, r1⟩ := exporter_block_1170_taken_packed (by simp; omega)
      (by rw [Words.isZero_gt_ne0, toNat_small hL, toNat_small (by omega)] <;> omega) (by kjump_dest) h
    obtain ⟨_, _, _, r3⟩ := exporter_block_1188_packed (by simp; omega) hret r1
    simp only [exporter_block_1188_stack, hround, hend] at r3
    unfold padTail; rw [if_neg hp]
    exact ⟨_, _, _, r3⟩

end ExporterEvm
