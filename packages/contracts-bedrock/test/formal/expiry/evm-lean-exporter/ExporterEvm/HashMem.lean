import ExporterEvm.Loop

/-!
# Memory lemmas for the message hash (pc 170 → 240)

The body first copies `_message` from calldata into a `bytes memory` at `0x80` (pc 170), then
`Hashing.hashL2toL2CrossDomainMessage` ABI-encodes the six arguments at the new free-memory
pointer `p1 = 0xa0 + roundUp32(len)` (pc 1368, 1132, the copy loop) and hashes the encoding
(pc 837). The memory up to `p1` is `memA I`; the encoding follows it.
-/

namespace ExporterEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach exporterBlocks Mem

/-- `_message` as a list of bytes (the calldata bytes after the length word). -/
def msgData (I : ExecutionEnv) : List UInt8 := (I.calldata.data.toList.drop (msgPos I + 32)).take (msgLen I)

/-- The ABI zero padding of `_message`. -/
def padZ (I : ExecutionEnv) : List UInt8 := List.replicate (roundUp32 (msgLen I) - msgLen I) 0

/-- The free-memory pointer after `bytes memory m = _message`: `0x80 + 32 + roundUp32(len)`. -/
def p1 (I : ExecutionEnv) : ℕ := 160 + roundUp32 (msgLen I)

/-- Memory below `p1`: scratch, free-memory pointer `p1`, zero slot, then `_message` as a
    `bytes memory` at `0x80` (length word, bytes, padding). -/
def memA (I : ExecutionEnv) : List UInt8 :=
  List.replicate 64 0 ++ (wb (UInt256.ofNat (p1 I)) ++ (List.replicate 32 0 ++
    (wb (argMsgLen I) ++ (msgData I ++ padZ I))))

theorem msgLen_lt (I : ExecutionEnv) (hargs : ArgsOk I) (hcds : I.calldata.size < 2 ^ 63) :
    msgLen I < 2 ^ 63 := by
  have := hargs.bytesInside; unfold msgPos at this; omega

theorem roundUp32_ge (n : ℕ) : n ≤ roundUp32 n := by
  unfold roundUp32; have := Nat.div_add_mod (n + 31) 32; have := Nat.mod_lt (n + 31) (show 32 > 0 by norm_num)
  omega

theorem roundUp32_lt (n : ℕ) : roundUp32 n < n + 32 := by
  unfold roundUp32; have := Nat.div_mul_le_self (n + 31) 32; omega

theorem roundUp32_eq (n : ℕ) : roundUp32 n = 32 * ((n + 31) / 32) := by
  unfold roundUp32; ring

theorem msgData_length (I : ExecutionEnv) (hargs : ArgsOk I) : (msgData I).length = msgLen I := by
  unfold msgData
  have h := hargs.bytesInside
  simp only [List.length_take, List.length_drop, Array.length_toList]
  have : I.calldata.data.size = I.calldata.size := rfl
  omega

theorem memA_length (I : ExecutionEnv) (hargs : ArgsOk I) : (memA I).length = p1 I := by
  unfold memA p1 padZ
  simp only [List.length_append, List.length_replicate, length_wb, msgData_length I hargs]
  have := roundUp32_ge (msgLen I)
  omega

theorem argMsgLen_eq (I : ExecutionEnv) : argMsgLen I = UInt256.ofNat (msgLen I) :=
  Words.eq_ofNat_toNat _

/-- `CALLDATACOPY` to the end of the list memory. -/
theorem cd_write_end (cd : ByteArray) (P : List UInt8) (sa n : ℕ) (h : sa + n ≤ cd.size) :
    cd.write sa (ofL P) P.length n = ofL (P ++ (cd.data.toList.drop sa).take n) := by
  by_cases hn : n = 0
  · subst hn; rw [write_len0]; simp
  · rw [show ofL P = ofL (P ++ []) by simp, write_rel_src cd P [] sa P.length 0 n hn h (by simp)]
    simp

theorem toNat_div (a b : UInt256) : (UInt256.div a b).toNat = a.toNat / b.toNat := by
  rfl

theorem toNat_mul (a b : UInt256) : (UInt256.mul a b).toNat = (a.toNat * b.toNat) % 2 ^ 256 := by
  show (a.val * b.val).val = _
  rw [Fin.val_mul]; rfl

/-- solc's `(len + 31) / 32 * 32` (allocation size of the `bytes` payload). -/
theorem divmul_round (L : ℕ) (hL : L < 2 ^ 100) :
    UInt256.mul (UInt256.div (UInt256.ofNat 31 + UInt256.ofNat L) (UInt256.ofNat 32)) (UInt256.ofNat 32) =
      UInt256.ofNat (roundUp32 L) := by
  have h2 := roundUp32_lt L
  rw [ofNat_add_ofNat, Words.ext_iff, toNat_mul, toNat_div, toNat_small (by omega), toNat_small (by norm_num),
    toNat_small (by omega)]
  unfold roundUp32
  rw [Nat.mod_eq_of_lt (by have := Nat.div_mul_le_self (31 + L) 32; omega)]
  congr 2; omega

/-- Memory at pc 808, after `bytes memory m = _message` (pc 170). -/
theorem mem170_eq (I : ExecutionEnv) (hargs : ArgsOk I) (hcds : I.calldata.size < 2 ^ 63) :
    exporter_block_170_memory (ee := I) (mem := ofL (List.replicate 64 0 ++ wb (UInt256.ofNat 128)))
      (x1 := argMsgLen I) (x2 := UInt256.ofNat (msgPos I + 32)) =
    ofL (memA I ++ List.replicate (32 - (roundUp32 (msgLen I) - msgLen I)) 0) := by
  have hL := msgLen_lt I hargs hcds
  have hr1 := roundUp32_ge (msgLen I)
  have hr2 := roundUp32_lt (msgLen I)
  have hMl := msgData_length I hargs
  have hml : memLoad (UInt256.ofNat 64) (ofL (List.replicate 64 0 ++ wb (UInt256.ofNat 128))) =
      UInt256.ofNat 128 := by msimpg
  have hdm : UInt256.ofNat 128 + (UInt256.ofNat 32 + UInt256.mul (UInt256.div
      (UInt256.ofNat 31 + argMsgLen I) (UInt256.ofNat 32)) (UInt256.ofNat 32)) = UInt256.ofNat (p1 I) := by
    rw [argMsgLen_eq, divmul_round _ (by omega), ofNat_add_ofNat, ofNat_add_ofNat]
    unfold p1; rw [show 128 + (32 + roundUp32 (msgLen I)) = 160 + roundUp32 (msgLen I) by omega]
  unfold exporter_block_170_memory
  rw [hml, hdm]
  have e1 : (UInt256.ofNat (p1 I)).toByteArray.write 0 (ofL (List.replicate 64 0 ++ wb (UInt256.ofNat 128)))
      (UInt256.ofNat 64).toNat 32 = ofL (List.replicate 64 0 ++ wb (UInt256.ofNat (p1 I))) := by msimpg
  rw [e1]
  have e2 : (argMsgLen I).toByteArray.write 0 (ofL (List.replicate 64 0 ++ wb (UInt256.ofNat (p1 I))))
      (UInt256.ofNat 128).toNat 32 = ofL (List.replicate 64 0 ++ (wb (UInt256.ofNat (p1 I)) ++
        (List.replicate 32 0 ++ wb (argMsgLen I)))) := by msimpg
  rw [e2]
  have hW : (List.replicate 64 0 ++ (wb (UInt256.ofNat (p1 I)) ++ (List.replicate 32 0 ++
      wb (argMsgLen I)))).length = 160 := by simp
  have e3 : I.calldata.write (UInt256.ofNat (msgPos I + 32)).toNat (ofL (List.replicate 64 0 ++
      (wb (UInt256.ofNat (p1 I)) ++ (List.replicate 32 0 ++ wb (argMsgLen I)))))
      (UInt256.ofNat 32 + UInt256.ofNat 128).toNat (argMsgLen I).toNat =
      ofL ((List.replicate 64 0 ++ (wb (UInt256.ofNat (p1 I)) ++ (List.replicate 32 0 ++
        wb (argMsgLen I)))) ++ msgData I) := by
    rw [toNat_small (by have := hargs.offsetOk; unfold msgPos; omega),
      show (UInt256.ofNat 32 + UInt256.ofNat 128).toNat = (List.replicate 64 0 ++ (wb (UInt256.ofNat (p1 I)) ++
        (List.replicate 32 0 ++ wb (argMsgLen I)))).length by rw [hW]; rfl,
      cd_write_end _ _ _ _ (by have := hargs.bytesInside; exact this)]
    rfl
  rw [e3]
  have hW3 : ((List.replicate 64 0 ++ (wb (UInt256.ofNat (p1 I)) ++ (List.replicate 32 0 ++
      wb (argMsgLen I)))) ++ msgData I).length = 160 + msgLen I := by
    rw [List.length_append, hW, hMl]
  rw [write_wb, show ((UInt256.ofNat 32 + UInt256.ofNat 128) + argMsgLen I).toNat = 160 + msgLen I by
      rw [argMsgLen_eq, ofNat_add_ofNat, ofNat_add_ofNat, toNat_small (by omega)],
    List.take_of_length_le (by rw [hW3]), List.drop_eq_nil_of_le (by rw [hW3]; omega),
    hW3, Nat.sub_self, List.replicate_zero, List.append_nil, List.append_nil, wb_zero]
  unfold memA padZ
  apply congrArg ofL
  simp only [List.append_assoc, List.append_cancel_left_eq]
  rw [← List.replicate_add, Nat.add_sub_cancel' (by omega)]

theorem rep_take_fill (z c : ℕ) (hz : z ≤ c) :
    (List.replicate z (0 : UInt8)).take c ++ List.replicate (c - (List.replicate z (0 : UInt8)).length) 0 =
      List.replicate c 0 := by
  rw [List.take_of_length_le (by simp; omega), List.length_replicate, ← List.replicate_add,
    Nat.add_sub_cancel' hz]

/-- The six head words of `abi.encode(chainid, source, nonce, sender, target, …)` written at
    `p1 + 32 …` (pc 1368), over the zero word solc wrote after `_message` (whose last `z` bytes
    stick out past `p1`). -/
theorem mem1368_eq (A : List UInt8) (z : ℕ) (hz : z ≤ 32) (hA : A.length < 2 ^ 100) (a b c d e : UInt256) :
    exporter_block_1368_memory (mem := ofL (A ++ List.replicate z 0))
      (x0 := UInt256.ofNat 32 + UInt256.ofNat A.length) (x2 := e) (x3 := d) (x4 := c) (x5 := b) (x6 := a) =
    ofL (A ++ (List.replicate 32 0 ++ (wb a ++ (wb b ++ (wb c ++ (wb (UInt256.land d addrMask) ++
      (wb (UInt256.land e addrMask) ++ wb (UInt256.ofNat 192)))))))) := by
  unfold exporter_block_1368_memory
  simp only [ofNat_add_ofNat]
  simp (disch := omega) only [toNat_small]
  have wr0 : ∀ (w : UInt256) (Q : List UInt8), (UInt256.toByteArray w).write 0 (ofL (A ++ Q)) (32 + A.length) 32 =
      ofL (A ++ (Q.take 32 ++ List.replicate (32 - Q.length) 0 ++ wb w ++ Q.drop (32 + 32))) :=
    fun w Q => write_rel w A Q _ 32 (by omega)
  have wr : ∀ (w : UInt256) (Q : List UInt8) (c : ℕ), (UInt256.toByteArray w).write 0 (ofL (A ++ Q))
      (32 + A.length + c) 32 =
      ofL (A ++ (Q.take (32 + c) ++ List.replicate (32 + c - Q.length) 0 ++ wb w ++ Q.drop (32 + c + 32))) :=
    fun w Q c => write_rel w A Q _ (32 + c) (by omega)
  rw [wr0, rep_take_fill z 32 hz, List.drop_eq_nil_of_le (by simp; omega), List.append_nil]
  simp only [wr]
  unfold addrMask
  mnorm

/-- The rest of `memA` after the free-memory-pointer word. -/
def memRest (I : ExecutionEnv) : List UInt8 :=
  List.replicate 32 0 ++ (wb (argMsgLen I) ++ (msgData I ++ padZ I))

theorem memA_eq (I : ExecutionEnv) : memA I = List.replicate 64 0 ++ (wb (UInt256.ofNat (p1 I)) ++ memRest I) := rfl

theorem memRest_length (I : ExecutionEnv) (hargs : ArgsOk I) : (memRest I).length = p1 I - 96 := by
  have := memA_length I hargs
  rw [memA_eq] at this
  simp only [List.length_append, List.length_replicate, length_wb] at this
  omega

theorem memLoad128 (I : ExecutionEnv) (X : List UInt8) :
    memLoad (UInt256.ofNat 128) (ofL (memA I ++ X)) = argMsgLen I := by
  unfold memA
  rw [memLoad_ofL _ _ (by simp only [toNat_ofNat_mod, Nat.reduceMod, List.length_append,
    List.length_replicate, length_wb]; omega)]
  msimpg

theorem memLoad64 (I : ExecutionEnv) (X : List UInt8) :
    memLoad (UInt256.ofNat 64) (ofL (memA I ++ X)) = UInt256.ofNat (p1 I) := by
  unfold memA
  rw [memLoad_ofL _ _ (by simp only [toNat_ofNat_mod, Nat.reduceMod, List.length_append,
    List.length_replicate, length_wb]; omega)]
  msimpg

/-- pc 1132: the length word of `_message` stored after the head (at `p1 + 224`). -/
theorem mem1132_eq (I : ExecutionEnv) (hargs : ArgsOk I) (hcds : I.calldata.size < 2 ^ 63)
    (Q : List UInt8) (hQ : Q.length = 224) :
    exporter_block_1132_memory (mem := ofL (memA I ++ Q)) (x0 := UInt256.ofNat 128)
      (x1 := UInt256.ofNat 32 + UInt256.ofNat (memA I).length + UInt256.ofNat 192) =
    ofL (memA I ++ (Q ++ wb (argMsgLen I))) := by
  have hA := memA_length I hargs
  have hL := msgLen_lt I hargs hcds
  have hr := roundUp32_lt (msgLen I)
  unfold exporter_block_1132_memory
  rw [memLoad128, ofNat_add_ofNat, ofNat_add_ofNat, toNat_small (by unfold p1 at hA; omega),
    show 32 + (memA I).length + 192 = (memA I).length + 224 by omega,
    write_rel _ _ _ _ 224 rfl, List.take_of_length_le (by omega), List.drop_eq_nil_of_le (by omega),
    hQ, Nat.sub_self, List.replicate_zero, List.append_nil, List.append_nil]

/-- The bytes the copy loop reads: `_message` and its zero padding. -/
theorem loop_src (I : ExecutionEnv) (hargs : ArgsOk I) (X : List UInt8) :
    ((memA I ++ X).drop (128 + 32)).take (32 * ((msgLen I + 31) / 32)) = msgData I ++ padZ I := by
  have hMl := msgData_length I hargs
  have h1 := roundUp32_ge (msgLen I)
  rw [← roundUp32_eq]
  unfold memA
  simp only [List.append_assoc]
  rw [show 128 + 32 = 64 + (32 + (32 + 32)) by rfl, ← List.drop_drop, List.drop_left' (by simp),
    ← List.drop_drop, List.drop_left' (by simp), ← List.drop_drop, List.drop_left' (by simp),
    List.drop_left' (by simp), ← List.append_assoc,
    List.take_left' (by unfold padZ; simp only [List.length_append, List.length_replicate, hMl]; omega)]

/-- After the loop's tail, the first `roundUp32(len)` bytes of the payload area are `_message`
    and its padding. -/
theorem padTail_take (I : ExecutionEnv) (hargs : ArgsOk I) :
    (padTail (msgData I ++ padZ I) (msgLen I) ((msgLen I + 31) / 32)).take (roundUp32 (msgLen I)) =
      msgData I ++ padZ I := by
  have hMl := msgData_length I hargs
  have h1 := roundUp32_ge (msgLen I)
  have h2 := roundUp32_lt (msgLen I)
  unfold padTail
  rw [← roundUp32_eq]
  split
  · rw [List.take_left' hMl, List.take_append, List.take_of_length_le (by omega), hMl,
      List.take_replicate, Nat.min_def, if_pos (by omega)]
    rfl
  · rw [List.take_of_length_le (by unfold padZ; simp [hMl]; omega)]

theorem padTail_length (I : ExecutionEnv) (hargs : ArgsOk I) :
    roundUp32 (msgLen I) ≤ (padTail (msgData I ++ padZ I) (msgLen I) ((msgLen I + 31) / 32)).length ∧
    (padTail (msgData I ++ padZ I) (msgLen I) ((msgLen I + 31) / 32)).length ≤ roundUp32 (msgLen I) + 32 := by
  have hMl := msgData_length I hargs
  have h1 := roundUp32_ge (msgLen I)
  have h2 := roundUp32_lt (msgLen I)
  unfold padTail
  rw [← roundUp32_eq]
  split
  · simp only [List.length_append, List.length_take, List.length_replicate, hMl, padZ]; omega
  · simp only [List.length_append, List.length_replicate, hMl, padZ]; omega

/-- The statement's preimage as a list. -/
theorem exportPreimage_eq (I : ExecutionEnv) (hargs : ArgsOk I) :
    exportPreimage I = ofL (wb chainIdWord ++ (wb (argSource I) ++ (wb (argNonce I) ++ (wb (argSender I) ++
      (wb (argTarget I) ++ (wb (UInt256.ofNat 192) ++ (wb (argMsgLen I) ++ (msgData I ++ padZ I)))))))) := by
  unfold exportPreimage argMessage padZ
  rw [eq_ofL I.calldata, extract_ofL, zeroes_ofL]
  simp only [toByteArray_eq_ofL, append_ofL, List.append_assoc]
  unfold msgData
  rw [List.drop_take, Nat.add_sub_cancel_left]

/-- pc 837 (end of `hashL2toL2CrossDomainMessage`): `mstore(p1, end - p1 - 32)` (the encoding's
    length word) and `mstore(0x40, end)`. -/
theorem mem837_eq (I : ExecutionEnv) (hargs : ArgsOk I) (hcds : I.calldata.size < 2 ^ 63)
    (Q T : List UInt8) (hQ : Q.length = 256) (hTl : T.length ≤ roundUp32 (msgLen I) + 32) :
    (UInt256.ofNat (p1 I + 256 + roundUp32 (msgLen I))).toByteArray.write 0
      ((UInt256.sub (UInt256.sub (UInt256.ofNat (p1 I + 256 + roundUp32 (msgLen I))) (UInt256.ofNat (p1 I)))
        (UInt256.ofNat 32)).toByteArray.write 0 (ofL (memA I ++ (Q ++ T))) (UInt256.ofNat (p1 I)).toNat 32)
      (UInt256.ofNat 64).toNat 32 =
    ofL (List.replicate 64 0 ++ (wb (UInt256.ofNat (p1 I + 256 + roundUp32 (msgLen I))) ++
      (memRest I ++ (wb (UInt256.ofNat (224 + roundUp32 (msgLen I))) ++ (Q.drop 32 ++ T))))) := by
  have hA := memA_length I hargs
  have hR := memRest_length I hargs
  have hL := msgLen_lt I hargs hcds
  have hr := roundUp32_lt (msgLen I)
  have hp1 : p1 I < 2 ^ 70 := by unfold p1; omega
  rw [ofNat_sub_ofNat _ _ (by omega) (by omega), ofNat_sub_ofNat _ _ (by omega) (by omega),
    show p1 I + 256 + roundUp32 (msgLen I) - p1 I - 32 = 224 + roundUp32 (msgLen I) by omega,
    toNat_small (by omega), show p1 I = (memA I).length + 0 by omega, write_rel _ _ _ _ 0 rfl]
  rw [show (memA I).length + 0 = p1 I by omega]
  simp only [List.take_zero, Nat.zero_sub, List.replicate_zero, List.nil_append, Nat.zero_add]
  rw [List.drop_append_of_le_length (by omega), memA_eq, write_wb]
  have hX : ∀ X : List UInt8, (List.replicate 64 (0 : UInt8) ++ (wb (UInt256.ofNat (p1 I)) ++ memRest I) ++ X).take 64 =
      List.replicate 64 0 := fun X => by
    rw [List.append_assoc, List.take_left' (by simp)]
  have hY : ∀ X : List UInt8, (List.replicate 64 (0 : UInt8) ++ (wb (UInt256.ofNat (p1 I)) ++ memRest I) ++ X).drop (64 + 32) =
      memRest I ++ X := fun X => by
    rw [List.append_assoc, ← List.drop_drop, List.drop_left' (by simp), List.append_assoc,
      List.drop_left' (by simp)]
  rw [show (UInt256.ofNat 64).toNat = 64 by rfl, hX, hY]
  simp only [List.length_append, List.length_replicate, length_wb]
  rw [show 64 - (64 + (32 + (memRest I).length) + (32 + ((Q.drop 32).length + T.length))) = 0 by omega,
    List.replicate_zero, List.append_nil]
  simp only [List.append_assoc]

/-- The word `KECCAK256` leaves on the stack at pc 837: the hash of the encoding's 224-byte head
    (`Q` minus its first word) and the first `roundUp32(len)` payload bytes. -/
theorem keccak837 (I : ExecutionEnv) (hargs : ArgsOk I) (hcds : I.calldata.size < 2 ^ 63)
    (Q T : List UInt8) (hQ : Q.length = 256) (hT1 : roundUp32 (msgLen I) ≤ T.length) :
    keccakWord (UInt256.ofNat 32 + UInt256.ofNat (p1 I))
      (memLoad (UInt256.ofNat (p1 I)) (ofL (List.replicate 64 0 ++ (wb (UInt256.ofNat (p1 I + 256 + roundUp32 (msgLen I))) ++
        (memRest I ++ (wb (UInt256.ofNat (224 + roundUp32 (msgLen I))) ++ (Q.drop 32 ++ T)))))))
      (ofL (List.replicate 64 0 ++ (wb (UInt256.ofNat (p1 I + 256 + roundUp32 (msgLen I))) ++
        (memRest I ++ (wb (UInt256.ofNat (224 + roundUp32 (msgLen I))) ++ (Q.drop 32 ++ T)))))) =
    UInt256.ofNat (fromByteArrayBigEndian (KEC (ofL (Q.drop 32 ++ T.take (roundUp32 (msgLen I)))))) := by
  have hR := memRest_length I hargs
  have hL := msgLen_lt I hargs hcds
  have hr := roundUp32_lt (msgLen I)
  have hp1 : p1 I < 2 ^ 70 := by unfold p1; omega
  have hp96 : 96 ≤ p1 I := by unfold p1; omega
  have hP : (List.replicate 64 (0 : UInt8) ++ (wb (UInt256.ofNat (p1 I + 256 + roundUp32 (msgLen I))) ++
      memRest I)).length = p1 I := by simp [hR]; omega
  have e : List.replicate 64 (0 : UInt8) ++ (wb (UInt256.ofNat (p1 I + 256 + roundUp32 (msgLen I))) ++
        (memRest I ++ (wb (UInt256.ofNat (224 + roundUp32 (msgLen I))) ++ (Q.drop 32 ++ T)))) =
      (List.replicate 64 0 ++ (wb (UInt256.ofNat (p1 I + 256 + roundUp32 (msgLen I))) ++ memRest I)) ++
        (wb (UInt256.ofNat (224 + roundUp32 (msgLen I))) ++ (Q.drop 32 ++ T)) := by
    simp only [List.append_assoc]
  rw [e]
  generalize hPdef : (List.replicate 64 (0 : UInt8) ++ (wb (UInt256.ofNat (p1 I + 256 + roundUp32 (msgLen I))) ++
      memRest I)) = P at hP
  have hQd : (Q.drop 32).length = 224 := by simp [hQ]
  rw [memLoad_ofL _ _ (by rw [toNat_small (by omega)]; simp [hP, hQd]; try omega), toNat_small (by omega),
    List.drop_left' hP, List.take_left' (by simp), ofNat_fromBE_wb, ofNat_add_ofNat,
    keccakWord_ofL _ _ _ (by rw [toNat_small (by omega)]; omega) (by rw [toNat_small (by omega)]; omega)
      (by rw [toNat_small (by omega), toNat_small (by omega)]; simp [hP, hQd]; try omega),
    toNat_small (by omega), toNat_small (by omega),
    show 32 + p1 I = P.length + 32 by omega, ← List.drop_drop, List.drop_left' rfl,
    List.drop_left' (by simp), List.take_append, List.take_of_length_le (by omega), hQd,
    show 224 + roundUp32 (msgLen I) - 224 = roundUp32 (msgLen I) by omega]

end ExporterEvm
