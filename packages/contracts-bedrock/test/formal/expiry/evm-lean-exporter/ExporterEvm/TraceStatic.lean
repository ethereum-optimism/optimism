import ExporterEvm.TraceHash
import Ethereum.Theory.StaticStorage

/-!
# Trace segment 3 (pc 240 → 439): `successfulMessages(H)` and the relayed check

`STATICCALL` to the L2ToL2CrossDomainMessenger (0x4200…0023) with calldata exactly
`successfulMessages(H)` (pc 331), the return-data decoding (`abi_decode_bool`, pc 1265/1283) and
`if (relayed) revert UndeliveredMessageExporter_MessageRelayed()` (pc 384/390). The revert
outputs are recorded: empty (depth limit, short or non-boolean return data), the failed call's
own return data (bubbled), or exactly the 4-byte custom error.
-/

namespace ExporterEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach exporterBlocks Mem

/-! ## Memory above a free-memory pointer -/

theorem memLoad_fmp {B : List UInt8} {fp : ℕ} (hB : FmpMem B fp) :
    memLoad (UInt256.ofNat 64) (ofL B) = UInt256.ofNat fp := by
  rw [memLoad_ofL _ _ (by rw [toNat_small (by norm_num)]; exact hB.len96),
    toNat_small (by norm_num), hB.fmp, ofNat_fromBE_wb]

theorem memLoad_fmp_pre {B Q : List UInt8} {fp : ℕ} (hB : FmpMem B fp) :
    memLoad (UInt256.ofNat 64) (ofL (pre B fp ++ Q)) = UInt256.ofNat fp := by
  rw [memLoad_pre _ _ _ _ (by rw [toNat_small (by norm_num)]; exact hB.ge)
    (by rw [toNat_small (by norm_num)]; exact hB.len96), toNat_small (by norm_num), hB.fmp,
    ofNat_fromBE_wb]

/-- `mstore(0x40, fp')`: the new memory has its free-memory-pointer word replaced. -/
def setFmp (L : List UInt8) (fp' : ℕ) : List UInt8 := L.take 64 ++ wb (UInt256.ofNat fp') ++ L.drop 96

theorem write_fmp (L : List UInt8) (fp' : ℕ) (hL : 96 ≤ L.length) :
    (UInt256.toByteArray (UInt256.ofNat fp')).write 0 (ofL L) (UInt256.ofNat 64).toNat 32 =
      ofL (setFmp L fp') := by
  rw [write_wb, toNat_small (by norm_num), show 64 - L.length = 0 by omega, List.replicate_zero,
    List.append_nil]
  rfl

theorem setFmp_length (L : List UInt8) (fp' : ℕ) (hL : 96 ≤ L.length) :
    (setFmp L fp').length = L.length := by
  unfold setFmp; simp; omega

theorem setFmp_fmp (L : List UInt8) (fp' : ℕ) (hL : 96 ≤ L.length) :
    ((setFmp L fp').drop 64).take 32 = wb (UInt256.ofNat fp') := by
  unfold setFmp
  rw [List.append_assoc, List.drop_left' (by stk_ov), List.take_left' (by simp)]

theorem fmpMem_setFmp (L : List UInt8) (fp' : ℕ) (hL : 96 ≤ L.length) (hfp : 96 ≤ fp')
    (hhi : L.length ≤ fp' + 32) (hb : fp' < 2 ^ 150) : FmpMem (setFmp L fp') fp' :=
  ⟨setFmp_fmp L fp' hL, by rw [setFmp_length L fp' hL]; exact hL, by rw [setFmp_length L fp' hL]; exact hhi,
    hfp, hb⟩

/-- Bytes `[fp, fp + 32)` of `setFmp` are those of the old list (`fp ≥ 96`). -/
theorem setFmp_drop (L : List UInt8) (fp' a : ℕ) (hL : 96 ≤ L.length) (ha : 96 ≤ a) :
    (setFmp L fp').drop a = L.drop a := by
  unfold setFmp
  have h64 : (L.take 64).length = 64 := by simp; omega
  rw [List.append_assoc, List.drop_append, List.drop_eq_nil_of_le (by omega), List.nil_append,
    List.drop_append, List.drop_eq_nil_of_le (by stk_ov), List.nil_append, List.drop_drop, h64,
    length_wb, show 96 + (a - 64 - 32) = a by omega]

/-! ## Calldata and revert outputs -/

abbrev succSelW : UInt256 :=
  UInt256.ofNat 80373334883193173493124541549841241460183627345106376917665715749844309508096

theorem successfulCalldata_eq (H : UInt256) :
    successfulCalldata H = ofL ((wb succSelW).take 4 ++ wb H) := by
  unfold successfulCalldata
  rw [show successfulSelector = ofL ((wb succSelW).take 4) by decide +kernel, toByteArray_eq_ofL, append_ofL]

/-- An empty byte array. -/
theorem size_zero_empty (b : ByteArray) (h : b.size = 0) : b = ByteArray.empty := by
  apply ByteArray.ext
  exact Array.eq_empty_of_size_eq_zero h

/-- `RETURNDATACOPY(0, 0, n); REVERT(0, n)` outputs `bubble rd`. -/
theorem bubble_read (rd mem : ByteArray) :
    (rd.write 0 mem 0 rd.size).readWithPadding 0 rd.size = bubble rd := by
  unfold bubble
  by_cases h0 : rd.size = 0
  · rw [h0, readWithPadding_zero, if_pos (by norm_num), size_zero_empty rd h0]
  by_cases h64 : rd.size < 2 ^ 64
  · rw [if_pos h64, eq_ofL mem, eq_ofL rd, write_ofL _ _ _ _ (by simpa using h0) (by simp)]
    simp only [ofL_size, List.take_zero, Nat.zero_sub, List.replicate_zero, List.nil_append,
      List.take_length, Nat.zero_add] at h0 h64 ⊢
    have hl : rd.data.toList.length = rd.size := Array.length_toList
    rw [read_ofL _ _ _ (by rw [hl]; omega) (by rw [hl]; exact h64) (by simp), List.drop_zero,
      List.take_left' rfl]
  · rw [if_neg h64]
    unfold ByteArray.readWithPadding
    rw [if_pos (by omega)]
    rfl

/-- solc's revert-data bubbling (`RETURNDATASIZE PUSH1 0 DUP1 RETURNDATACOPY RETURNDATASIZE
    PUSH1 0 REVERT`) reverts with `bubble rdata`. -/
theorem RD.bubble {ee : ExecutionEnv} {g : Sat256} {s0 : State}
    {pc : UInt256} {stk : List UInt256} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray}
    {acc : AccountMap} {k C : ℕ}
    (h : RD exporterRuntime ee g s0 pc stk mem aw rdata acc k C)
    (hsz : rdata.size < 2 ^ 256)
    (h0 : decode exporterRuntime pc = some (.RETURNDATASIZE, .none))
    (h1 : decode exporterRuntime (pc + ⟨1⟩) = some (.Push .PUSH1, some (UInt256.ofNat 0, 1)))
    (h2 : decode exporterRuntime (pc + ⟨1⟩ + UInt256.ofNat 2) = some (.DUP1, .none))
    (h3 : decode exporterRuntime (pc + ⟨1⟩ + UInt256.ofNat 2 + ⟨1⟩) = some (.RETURNDATACOPY, .none))
    (h4 : decode exporterRuntime (pc + ⟨1⟩ + UInt256.ofNat 2 + ⟨1⟩ + ⟨1⟩) = some (.RETURNDATASIZE, .none))
    (h5 : decode exporterRuntime (pc + ⟨1⟩ + UInt256.ofNat 2 + ⟨1⟩ + ⟨1⟩ + ⟨1⟩) =
      some (.Push .PUSH1, some (UInt256.ofNat 0, 1)))
    (h6 : decode exporterRuntime (pc + ⟨1⟩ + UInt256.ofNat 2 + ⟨1⟩ + ⟨1⟩ + ⟨1⟩ + UInt256.ofNat 2) =
      some (.REVERT, .none))
    (hov : stk.length + 3 ≤ 1024) :
    RDrevP exporterRuntime g s0 (fun o => o = bubble rdata) := by
  have hs : (UInt256.ofNat rdata.size).toNat = rdata.size := Words.toNat_ofNat_lt hsz
  have r1 := h.returndatasize h0 (by omega)
  have r2 := r1.push1 (UInt256.ofNat 0) h1 (by stk_ov)
  have r3 := r2.dup1 h2 (by stk_ov)
  have r4 := RD.genReturndatacopy r3 h3 (by rw [hs, toNat_small (by norm_num)]; omega) (by omega)
  have r5 := r4.returndatasize h4 (by omega)
  have r6 := r5.push1 (UInt256.ofNat 0) h5 (by stk_ov)
  refine (RD.revP r6 h6 (by omega)).mono ?_
  intro o ho
  rw [ho, hs, show (UInt256.ofNat 0).toNat = 0 from rfl, bubble_read]

/-- `UndeliveredMessageExporter_MessageRelayed()` as solc's `PUSH32` word. -/
abbrev relayedErrW : UInt256 :=
  UInt256.ofNat 92618038157931011697939822198512234313800822569940725936850136633179623129088

theorem relayed_out (B : List UInt8) (fp : ℕ) (hB : FmpMem B fp) :
    memLoad (UInt256.ofNat 64) ((UInt256.toByteArray relayedErrW).write 0 (ofL B) fp 32) = UInt256.ofNat fp ∧
    ((UInt256.toByteArray relayedErrW).write 0 (ofL B) fp 32).readWithPadding fp 4 = messageRelayedError := by
  have h1 := hB.ge
  have h2 := hB.len96
  rw [write_wb]
  have hlen : (B.take fp ++ List.replicate (fp - B.length) (0 : UInt8)).length = fp := by simp; omega
  have hpre : ((B.take fp ++ List.replicate (fp - B.length) (0 : UInt8)).drop 64).take 32 = (B.drop 64).take 32 := by
    rw [List.drop_append_of_le_length (by stk_ov), List.take_append_of_le_length (by stk_ov),
      List.drop_take, List.take_take]
    congr 1; omega
  constructor
  · rw [memLoad_ofL _ _ (by rw [toNat_small (n := 64) (by norm_num)]; simp; omega),
      toNat_small (n := 64) (by norm_num), List.append_assoc, List.append_assoc, ← List.append_assoc (B.take fp),
      List.drop_append_of_le_length (by rw [hlen]; omega), List.take_append_of_le_length (by stk_ov),
      hpre, hB.fmp, ofNat_fromBE_wb]
  · rw [read_ofL _ _ _ (by norm_num) (by norm_num) (by stk_ov), List.append_assoc, List.append_assoc,
      ← List.append_assoc (B.take fp), List.drop_left' hlen, List.take_append_of_le_length (by simp)]
    decide +kernel

/-- pc 390: `revert UndeliveredMessageExporter_MessageRelayed()` outputs exactly its 4-byte
    selector. -/
theorem rev_relayed {ee : ExecutionEnv} {g : Sat256} {s0 : State}
    {R : List UInt256} {B : List UInt8} {fp : ℕ} {aw : UInt256} {rdata : ByteArray}
    {acc : AccountMap} {k C : ℕ} (hB : FmpMem B fp) (hR : R.length ≤ 1000)
    (h : RD exporterRuntime ee g s0 (UInt256.ofNat 390) R (ofL B) aw rdata acc k C) :
    RDrevP exporterRuntime g s0 (fun o => o = messageRelayedError) := by
  have r1 := kevm_run h with [push1 (UInt256.ofNat 64)]
  have r2 := RD.genMload r1 (by evm_kdecide) (by stk_ov)
  rw [memLoad_fmp hB] at r2
  have r3 := r2.pushConst relayedErrW (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide)
    (by stk_ov)
  have r4 := kevm_run r3 with [dup2]
  have r5 := RD.genMstore r4 (by evm_kdecide) (by stk_ov)
  have r6 := kevm_run r5 with [push1 (UInt256.ofNat 4), add, push1 (UInt256.ofNat 64)]
  have r7 := RD.genMload r6 (by evm_kdecide) (by stk_ov)
  have r8 := kevm_run r7 with [dup1, swap2, sub, swap1]
  refine (RD.revP r8 (by evm_kdecide) (by stk_ov)).mono ?_
  intro o ho
  have hfp : (UInt256.ofNat fp).toNat = fp := toNat_small (by have := hB.bound; omega)
  rw [hfp, (relayed_out B fp hB).1, Words.sub_add_self', hfp, toNat_small (n := 4) (by norm_num)] at ho
  rw [ho, (relayed_out B fp hB).2]

/-- The revert outputs this segment can produce. -/
def StaticRevert (σ σ₀ : AccountMap) (I : ExecutionEnv) (H : UInt256) (o : ByteArray) : Prop :=
  o = ByteArray.empty ∨
  (∃ σ₁ rd, StaticCall σ₀ I l2l2 (successfulCalldata H) σ σ₁ false rd ∧ o = bubble rd) ∨
  (o = messageRelayedError ∧ ∃ σ₁ oS, StaticCall σ₀ I l2l2 (successfulCalldata H) σ σ₁ true oS ∧
    32 ≤ oS.size ∧ returnWord oS = UInt256.ofNat 1)

theorem bubble_empty : bubble ByteArray.empty = ByteArray.empty := by
  unfold bubble; rw [if_pos (by decide)]

set_option maxHeartbeats 4000000 in
/-- **Trace segment 3** (pc 240 → 439). -/
theorem seg_static {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256}
    {aw : UInt256} {k C : ℕ} {H : UInt256} {R : List UInt256} {B : List UInt8} {fp : ℕ}
    (hB : FmpMem B fp) (hfp : fp < 2 ^ 139) (hR : R.length ≤ 900)
    (h : RD exporterRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 240)
        (H :: UInt256.ofNat 0 :: R) (ofL B) aw ByteArray.empty σ k C) :
    RDrevP exporterRuntime g (initState σ σ₀ g A I) (StaticRevert σ σ₀ I H) ∨
    ∃ σ₁ oS, StaticCall σ₀ I l2l2 (successfulCalldata H) σ σ₁ true oS ∧ 32 ≤ oS.size ∧
      oS.size < 2 ^ 138 ∧ returnWord oS = UInt256.ofNat 0 ∧
      accountStorageStateEq σ σ₁ ∧ accountCodeStateEq σ σ₁ ∧
      ∃ B2 fp2 aw' k' C', FmpMem B2 fp2 ∧ fp2 < 2 ^ 140 ∧
        RD exporterRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 439) (H :: R) (ofL B2) aw' oS σ₁ k' C' := by
  have hfpN : (UInt256.ofNat fp).toNat = fp := toNat_small (by omega)
  obtain ⟨_, _, _, r1⟩ := exporter_block_240_packed (by stk_ov) h
  simp only [exporter_block_240_stack, exporter_block_240_memory, memLoad_fmp hB] at r1
  have hm : H.toByteArray.write 0 (succSelW.toByteArray.write 0 (ofL B) (UInt256.ofNat fp).toNat 32)
      (UInt256.ofNat fp + UInt256.ofNat 4).toNat 32 = ofL (pre B fp ++ ((wb succSelW).take 4 ++ wb H)) := by
    rw [hfpN, write_first _ _ _ hB.hi, ofNat_add_ofNat, toNat_small (by omega),
      write_rel _ _ _ _ 4 (by simp)]
    congr 2
    simp
  rw [hm, memLoad_fmp_pre hB, Words.sub_add_self'] at r1
  have hdec : decode exporterRuntime (UInt256.ofNat 331) = some (.STATICCALL, .none) := by evm_kdecide
  by_cases hd : I.depth.val < 1024
  · obtain ⟨σ', z, o, A_in, callGas, k3, C3, ⟨g'', A', hΘ⟩, r3, hosize⟩ :=
      RD.solcStaticcall r1 hdec hd (by stk_ov)
    rw [hfpN, toNat_small (n := 36) (by norm_num), read_rel _ _ _ 0 36 (by simp) (by decide) (by decide) (by simp),
      List.drop_zero, List.take_of_length_le (by simp), ← successfulCalldata_eq] at hΘ
    have hcall : StaticCall σ₀ I l2l2 (successfulCalldata H) σ σ' z o := ⟨A_in, callGas, g'', A', hΘ⟩
    have hst : accountStorageStateEq σ σ' := Theta_static_accountStorageStateEq hΘ.symm
    have hcd : accountCodeStateEq σ σ' := Theta_static_accountCodeStateEq hΘ.symm
    have hos : o.size < 2 ^ 256 := by have := hosize; rw [Words.size_eq] at this; exact this
    cases z with
    | false =>
      simp only [Bool.false_eq_true, if_false] at r3
      have r4 := exporter_block_332_fallthrough (by stk_ov) (by decide) r3
      simp only [exporter_block_332_fallthrough_stack] at r4
      left
      refine (RD.bubble r4 hos (by evm_kdecide) (by evm_kdecide) (by evm_kdecide) (by evm_kdecide)
        (by evm_kdecide) (by evm_kdecide) (by evm_kdecide) (by stk_ov)).mono ?_
      intro o' ho'
      exact Or.inr (Or.inl ⟨σ', o, hcall, ho'⟩)
    | true =>
      simp only [if_true] at r3
      have hob : o.size < 2 ^ 138 := Theta_returnData_size_lt_2pow138_of_eq _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ hΘ
        (by rw [successfulCalldata_eq]; simp [Ethereum.EVM.maxReturnDataSizeByGas, Ethereum.EVM.maxReturnDataWordsByGas])
      clear hΘ
      have r4 := exporter_block_332_taken (by stk_ov) (by decide) (by kjump_dest) r3
      simp only [exporter_block_332_taken_stack] at r4
      by_cases hlen : 32 ≤ o.size
      swap
      · obtain ⟨_, _, _, r5⟩ := exporter_block_348_packed (by stk_ov) (by kjump_dest) r4
        simp only [exporter_block_348_stack] at r5
        have r6 := exporter_block_1265_fallthrough (by stk_ov)
          (by rw [Words.sub_add_self]; exact Words.retlen_bad _ (by omega)) r5
        simp only [exporter_block_1265_fallthrough_stack] at r6
        left
        exact (RD.rev00 r6 (by evm_kdecide) (by evm_kdecide) (by evm_kdecide) (by stk_ov)).mono
          (fun _ ho => Or.inl ho)
      have hpl : (pre B fp).length = fp := length_pre B fp
      rw [Words.min32_toNat _ hlen (by omega), hfpN, write_bytes32 _ _ _ hlen,
        List.take_left' hpl, show fp - (pre B fp ++ ((wb succSelW).take 4 ++ wb H)).length = 0 by simp,
        List.replicate_zero, List.append_nil, List.drop_append, List.drop_eq_nil_of_le (by omega), List.nil_append,
        hpl, show fp + 32 - fp = 32 by omega, List.drop_append, List.drop_eq_nil_of_le (by simp), List.nil_append,
        show 32 - ((wb succSelW).take 4).length = 28 by simp, List.append_assoc] at r4
      have hRl := length_take32 o hlen
      generalize hRw : o.data.toList.take 32 = Rw at r4 hRl
      obtain ⟨_, _, _, r5⟩ := exporter_block_348_packed (by stk_ov) (by kjump_dest) r4
      simp only [exporter_block_348_stack, exporter_block_348_memory, memLoad_fmp_pre hB] at r5
      have hL96 : 96 ≤ (pre B fp ++ (Rw ++ (wb H).drop 28)).length := by
        simp [hRl]; have := hB.ge; omega
      rw [Words.alloc_ret fp o.size (by omega) (by omega), write_fmp _ _ hL96] at r5
      have r6 := exporter_block_1265_taken (by stk_ov)
        (by rw [Words.sub_add_self]; exact Words.retlen_ok _ hlen (by omega)) (by kjump_dest) r5
      simp only [exporter_block_1265_taken_stack] at r6
      set fp2 := fp + (o.size + 31) / 32 * 32 with hfp2
      have hfp2a : fp + 32 ≤ fp2 := by
        have : 1 ≤ (o.size + 31) / 32 := (Nat.le_div_iff_mul_le (by norm_num)).mpr (by omega)
        omega
      have hfp2b : fp2 < 2 ^ 140 := by
        have := Nat.div_mul_le_self (o.size + 31) 32
        omega
      have hlenL : (pre B fp ++ (Rw ++ (wb H).drop 28)).length = fp + 36 := by simp [hRl]
      have hmw : memLoad (UInt256.ofNat fp) (ofL (setFmp (pre B fp ++ (Rw ++ (wb H).drop 28)) fp2)) =
          UInt256.ofNat (fromBytesBigEndian Rw) := by
        rw [memLoad_ofL _ _ (by rw [hfpN, setFmp_length _ _ hL96, hlenL]; omega), hfpN,
          setFmp_drop _ _ _ hL96 hB.ge, List.drop_left' hpl, List.take_left' hRl]
      by_cases hb : UInt256.eq (UInt256.ofNat (fromBytesBigEndian Rw))
          (UInt256.isZero (UInt256.isZero (UInt256.ofNat (fromBytesBigEndian Rw)))) = UInt256.ofNat 0
      · have r7 := exporter_block_1283_fallthrough (by stk_ov) (by rw [hmw]; exact hb) r6
        simp only [exporter_block_1283_fallthrough_stack] at r7
        left
        exact (RD.rev00 r7 (by evm_kdecide) (by evm_kdecide) (by evm_kdecide) (by stk_ov)).mono
          (fun _ ho => Or.inl ho)
      have r7 := exporter_block_1283_taken (by stk_ov) (by rw [hmw]; exact hb) (by kjump_dest) r6
      simp only [exporter_block_1283_taken_stack, hmw] at r7
      have r8 := exporter_block_1258 (by stk_ov) (by kjump_dest) r7
      simp only [exporter_block_1258_stack] at r8
      have hB2 : FmpMem (setFmp (pre B fp ++ (Rw ++ (wb H).drop 28)) fp2) fp2 :=
        fmpMem_setFmp _ _ hL96 (by have := hB.ge; omega) (by rw [hlenL]; omega) (by omega)
      have hw : returnWord o = UInt256.ofNat (fromBytesBigEndian Rw) := by unfold returnWord; rw [hRw]
      by_cases hz : UInt256.ofNat (fromBytesBigEndian Rw) = UInt256.ofNat 0
      · have r9 := exporter_block_384_taken (by stk_ov) (by rw [hz]; decide) (by kjump_dest) r8
        simp only [exporter_block_384_taken_stack] at r9
        right
        exact ⟨σ', o, hcall, hlen, hob, by rw [hw, hz], hst, hcd, _, fp2, _, _, _, hB2, hfp2b, r9⟩
      · have r9 := exporter_block_384_fallthrough (by stk_ov) (by rw [Words.isZero_eq0]; exact hz) r8
        simp only [exporter_block_384_fallthrough_stack] at r9
        left
        refine (rev_relayed hB2 (by stk_ov) r9).mono ?_
        intro o' ho'
        exact Or.inr (Or.inr ⟨ho', σ', o, hcall, hlen, by rw [hw]; exact Words.bool_true hb hz⟩)
  · have hd' : I.depth = 1024 := by
      apply Fin.ext; have := I.depth.isLt; simp only [Fin.val_ofNat] at *; omega
    obtain ⟨k3, C3, r3⟩ := RD.solcStaticcallDepthLimit r1 hdec hd' (by stk_ov)
    have r4 := exporter_block_332_fallthrough (by stk_ov) (by decide) r3
    simp only [exporter_block_332_fallthrough_stack] at r4
    left
    refine (RD.bubble r4 (by simp) (by evm_kdecide) (by evm_kdecide) (by evm_kdecide) (by evm_kdecide)
      (by evm_kdecide) (by evm_kdecide) (by evm_kdecide) (by stk_ov)).mono ?_
    intro o' ho'
    rw [bubble_empty] at ho'
    exact Or.inl ho'

end ExporterEvm
