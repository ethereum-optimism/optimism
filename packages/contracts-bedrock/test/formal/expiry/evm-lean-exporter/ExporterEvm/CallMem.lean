import ExporterEvm.TraceStatic

/-!
# Memory lemmas for `L2CrossDomainMessenger.sendMessage(...)` (pc 439 → 693)

All writes are relative to the free-memory pointer `fp` of a memory `B` with `FmpMem B fp`:
memory is `ofL (pre B' fp ++ Q)` with `Q` a concrete-shaped list. The bytes of `B` at or above
`fp` (at most 32 of them, `FmpMem.hi`) are overwritten before they are read (`head3`).
-/

namespace ExporterEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach exporterBlocks Mem

theorem write_rel_pre (w : UInt256) (B Q : List UInt8) (fp c : ℕ) :
    (UInt256.toByteArray w).write 0 (ofL (pre B fp ++ Q)) (fp + c) 32 =
      ofL (pre B fp ++ (Q.take c ++ List.replicate (c - Q.length) 0 ++ wb w ++ Q.drop (c + 32))) :=
  write_rel w _ Q _ c (by rw [length_pre])

theorem write_rel_pre0 (w : UInt256) (B Q : List UInt8) (fp : ℕ) :
    (UInt256.toByteArray w).write 0 (ofL (pre B fp ++ Q)) fp 32 =
      ofL (pre B fp ++ (wb w ++ Q.drop 32)) := by
  simpa using write_rel_pre w B Q fp 0

theorem drop_pre_append (B Q : List UInt8) (fp c : ℕ) : (pre B fp ++ Q).drop (fp + c) = Q.drop c := by
  rw [List.drop_append, List.drop_eq_nil_of_le (by rw [length_pre]; omega), List.nil_append, length_pre,
    Nat.add_sub_cancel_left]

theorem memLoad_rel_pre (B Q : List UInt8) (fp c : ℕ) (hc : c + 32 ≤ Q.length) (hb : fp + c < 2 ^ 200) :
    memLoad (UInt256.ofNat (fp + c)) (ofL (pre B fp ++ Q)) =
      UInt256.ofNat (fromBytesBigEndian ((Q.drop c).take 32)) := by
  rw [memLoad_ofL _ _ (by rw [toNat_small hb]; simp; omega), toNat_small hb, drop_pre_append]

theorem read_rel_pre (B Q : List UInt8) (fp c n : ℕ) (hn : 0 < n) (hn64 : n < 2 ^ 64)
    (hc : c + n ≤ Q.length) :
    (ofL (pre B fp ++ Q)).readWithPadding (fp + c) n = ofL ((Q.drop c).take n) :=
  read_rel _ Q _ c n (by rw [length_pre]) hn hn64 hc

theorem setFmp_pre (B Q : List UInt8) (fp fp' : ℕ) (h96 : 96 ≤ B.length) (hfp : 96 ≤ fp) :
    setFmp (pre B fp ++ Q) fp' = pre (setFmp B fp') fp ++ Q := by
  have htf : (B.take fp).length = min fp B.length := List.length_take
  have hA : (pre B fp ++ Q).take 64 = B.take 64 := by
    unfold pre
    rw [List.append_assoc, List.take_append_of_le_length (by omega), List.take_take,
      Nat.min_eq_left (by omega)]
  have hD : (pre B fp ++ Q).drop 96 = (B.drop 96).take (fp - 96) ++ (List.replicate (fp - B.length) 0 ++ Q) := by
    unfold pre
    rw [List.append_assoc, List.drop_append_of_le_length (by omega), List.drop_take]
  have hT : (setFmp B fp').take fp = B.take 64 ++ (wb (UInt256.ofNat fp') ++ (B.drop 96).take (fp - 96)) := by
    unfold setFmp
    have h64 : (B.take 64).length = 64 := by simp; omega
    rw [List.append_assoc, List.take_append, List.take_take, Nat.min_eq_right (by omega), h64,
      List.take_append, List.take_of_length_le (l := wb (UInt256.ofNat fp')) (by simp; omega), length_wb,
      show fp - 64 - 32 = fp - 96 by omega]
  have hL := setFmp_length B fp' h96
  unfold setFmp at hL ⊢
  rw [hA, hD]
  unfold pre
  rw [hL]
  have hT' := hT
  unfold setFmp at hT'
  rw [hT']
  simp only [List.append_assoc]

theorem write_fmp_pre (B Q : List UInt8) (fp fp' : ℕ) (h96 : 96 ≤ B.length) (hfp : 96 ≤ fp) :
    (UInt256.toByteArray (UInt256.ofNat fp')).write 0 (ofL (pre B fp ++ Q)) (UInt256.ofNat 64).toNat 32 =
      ofL (pre (setFmp B fp') fp ++ Q) := by
  rw [write_fmp _ _ (by simp; omega), setFmp_pre B Q fp fp' h96 hfp]

theorem memLoad_setFmp_pre (B Q : List UInt8) (fp fp' : ℕ) (h96 : 96 ≤ B.length) (hfp : 96 ≤ fp) :
    memLoad (UInt256.ofNat 64) (ofL (pre (setFmp B fp') fp ++ Q)) = UInt256.ofNat fp' := by
  have hl : 96 ≤ (pre B fp ++ Q).length := by simp; omega
  rw [← setFmp_pre B Q fp fp' h96 hfp]
  have hl2 := setFmp_length _ fp' hl
  rw [memLoad_ofL _ _ (by rw [toNat_small (n := 64) (by norm_num), hl2]; omega),
    toNat_small (n := 64) (by norm_num), setFmp_fmp _ _ hl, ofNat_fromBE_wb]

/-- The first three stores of `abi.encodeCall(relayUndeliveredMessage, (H, t))` at `fp`: `H` at
    `fp + 36`, `t` at `fp + 68`, then the length word `v` at `fp`. Whatever `B` held at
    `fp .. fp + 32` (`FmpMem.hi`) is overwritten; bytes `fp + 32 .. fp + 36` are zero. -/
theorem head3 (B : List UInt8) (fp : ℕ) (hB : FmpMem B fp) (H t v : UInt256) :
    (UInt256.toByteArray v).write 0 ((UInt256.toByteArray t).write 0
      ((UInt256.toByteArray H).write 0 (ofL B) (fp + 36) 32) (fp + 68) 32) fp 32 =
    ofL (pre B fp ++ (wb v ++ (List.replicate 4 0 ++ (wb H ++ wb t)))) := by
  have h1 := hB.hi
  rw [write_wb, List.take_of_length_le (by omega), List.drop_eq_nil_of_le (by omega), List.append_nil]
  have hl1 : (B ++ List.replicate (fp + 36 - B.length) (0 : UInt8) ++ wb H).length = fp + 68 := by simp; omega
  rw [write_wb, List.take_of_length_le (by omega), List.drop_eq_nil_of_le (by omega), List.append_nil,
    hl1, Nat.sub_self, List.replicate_zero, List.append_nil, write_wb]
  have hl2 : (B ++ List.replicate (fp + 36 - B.length) (0 : UInt8)).length = fp + 36 := by simp; omega
  rw [show fp - (B ++ List.replicate (fp + 36 - B.length) (0 : UInt8) ++ wb H ++ wb t).length = 0 by simp; omega,
    List.replicate_zero, List.append_nil]
  have hp : (B ++ List.replicate (fp + 36 - B.length) (0 : UInt8) ++ wb H ++ wb t).take fp = pre B fp := by
    rw [List.append_assoc, List.append_assoc, ← List.append_assoc B, List.take_append_of_le_length (by omega),
      List.take_append, List.take_replicate]
    unfold pre
    congr 2; omega
  have hd : (B ++ List.replicate (fp + 36 - B.length) (0 : UInt8) ++ wb H ++ wb t).drop (fp + 32) =
      List.replicate 4 0 ++ (wb H ++ wb t) := by
    rw [List.append_assoc, List.append_assoc, ← List.append_assoc B, List.drop_append_of_le_length (by omega),
      List.drop_append, List.drop_eq_nil_of_le (by omega), List.nil_append, List.drop_replicate]
    congr 2; omega
  rw [hp, hd]
  simp only [List.append_assoc]

theorem memLoad_W2 (B : List UInt8) (fp : ℕ) (hB : FmpMem B fp) (H t : UInt256) :
    memLoad (UInt256.ofNat 64) ((UInt256.toByteArray t).write 0
      ((UInt256.toByteArray H).write 0 (ofL B) (fp + 36) 32) (fp + 68) 32) = UInt256.ofNat fp := by
  have h1 := hB.hi
  have h2 := hB.len96
  rw [write_wb, List.take_of_length_le (by omega), List.drop_eq_nil_of_le (by omega), List.append_nil,
    write_wb, List.take_of_length_le (by simp; omega), List.drop_eq_nil_of_le (by simp; omega), List.append_nil,
    memLoad_ofL _ _ (by rw [toNat_small (n := 64) (by norm_num)]; simp; omega),
    toNat_small (n := 64) (by norm_num)]
  simp only [List.append_assoc]
  rw [List.drop_append_of_le_length (by omega), List.take_append_of_le_length (by simp; omega), hB.fmp,
    ofNat_fromBE_wb]

theorem write_fmp_pre' (B Q : List UInt8) (fp fp' : ℕ) (h96 : 96 ≤ B.length) (hfp : 96 ≤ fp) :
    (UInt256.toByteArray (UInt256.ofNat fp')).write 0 (ofL (pre B fp ++ Q)) 64 32 =
      ofL (pre (setFmp B fp') fp ++ Q) :=
  write_fmp_pre B Q fp fp' h96 hfp

theorem setFmp_len96 (B : List UInt8) (fp' : ℕ) (h96 : 96 ≤ B.length) : 96 ≤ (setFmp B fp').length := by
  rw [setFmp_length _ _ h96]; exact h96

theorem memLoad_rel_pre0 (B Q : List UInt8) (fp : ℕ) (hc : 32 ≤ Q.length) (hb : fp < 2 ^ 200) :
    memLoad (UInt256.ofNat fp) (ofL (pre B fp ++ Q)) = UInt256.ofNat (fromBytesBigEndian (Q.take 32)) := by
  simpa using memLoad_rel_pre B Q fp 0 (by omega) (by omega)

end ExporterEvm
